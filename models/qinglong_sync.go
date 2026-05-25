package models

import (
	"fmt"
	"log"
	"sync"
	"time"
)

type SyncAction string

const (
	SyncActionCreate    SyncAction = "create"
	SyncActionUpdate    SyncAction = "update"
	SyncActionDelete    SyncAction = "delete"
	SyncActionDisable   SyncAction = "disable"
	SyncActionEnable    SyncAction = "enable"
	SyncActionFullSync  SyncAction = "full_sync"
)

type SyncQueue struct {
	mu        sync.Mutex
	processing bool
	running   bool
	stopChan  chan struct{}
	ticker    *time.Ticker
}

var (
	syncQueue     *SyncQueue
	syncQueueOnce sync.Once
)

const (
	syncInterval       = 10 * time.Second
	syncBatchSize      = 20
	syncMaxRetries     = 3
	fullSyncInterval   = 30 * time.Minute
)

func GetSyncQueue() *SyncQueue {
	syncQueueOnce.Do(func() {
		syncQueue = &SyncQueue{
			stopChan: make(chan struct{}),
		}
	})
	return syncQueue
}

func InitSyncService() {
	sq := GetSyncQueue()
	if sq.running {
		return
	}
	sq.running = true
	sq.ticker = time.NewTicker(syncInterval)

	go sq.processLoop()
	go sq.fullSyncLoop()

	log.Println("[同步服务] 数据库→青龙同步服务已启动")
}

func StopSyncService() {
	sq := GetSyncQueue()
	if !sq.running {
		return
	}
	sq.running = false
	if sq.ticker != nil {
		sq.ticker.Stop()
	}
	close(sq.stopChan)
	log.Println("[同步服务] 同步服务已停止")
}

func (sq *SyncQueue) processLoop() {
	for {
		select {
		case <-sq.stopChan:
			return
		case <-sq.ticker.C:
			sq.processPendingSyncs()
		}
	}
}

func (sq *SyncQueue) fullSyncLoop() {
	ticker := time.NewTicker(fullSyncInterval)
	defer ticker.Stop()

	time.Sleep(60 * time.Second)

	for {
		select {
		case <-sq.stopChan:
			return
		case <-ticker.C:
			log.Println("[同步服务] 开始全量同步检查...")
			sq.performFullSyncCheck()
		}
	}
}

func (sq *SyncQueue) processPendingSyncs() {
	if sq.processing {
		return
	}
	sq.mu.Lock()
	if sq.processing {
		sq.mu.Unlock()
		return
	}
	sq.processing = true
	sq.mu.Unlock()

	defer func() {
		sq.mu.Lock()
		sq.processing = false
		sq.mu.Unlock()
	}()

	projects, err := GetPendingSyncProjects(syncBatchSize)
	if err != nil {
		log.Printf("[同步服务] 获取待同步项目失败: %v", err)
		return
	}

	if len(projects) == 0 {
		return
	}

	log.Printf("[同步服务] 发现 %d 个待同步项目", len(projects))

	for _, project := range projects {
		if err := sq.syncProject(&project); err != nil {
			log.Printf("[同步服务] 同步项目失败 ID=%d Remarks=%s: %v", project.ID, project.Remarks, err)
			MarkProjectSyncError(project.ID, SanitizeError(err).Error())
		}
	}

	// 处理已软删除但未同步到青龙的待删除记录
	var deletedProjects []ActivityProject
	db.Unscoped().Where("sync_status = ? AND deleted_at IS NOT NULL", "pending_delete").
		Limit(syncBatchSize).
		Find(&deletedProjects)
	for _, project := range deletedProjects {
		if err := sq.syncProject(&project); err != nil {
			log.Printf("[同步服务] 同步已软删除项目失败 ID=%d Remarks=%s: %v", project.ID, project.Remarks, err)
			MarkProjectSyncError(project.ID, SanitizeError(err).Error())
		}
	}
}

func (sq *SyncQueue) syncProject(project *ActivityProject) error {
	qlConfig := qlManagerGetConfig(project.QingLongConfigName)
	if qlConfig == nil {
		return fmt.Errorf("青龙配置 %s 不存在", project.QingLongConfigName)
	}

	client := NewQingLongClient(qlConfig)

	switch project.SyncStatus {
	case "pending":
		return sq.handleCreate(project, client)
	case "pending_update":
		return sq.handleUpdate(project, client)
	case "pending_delete":
		return sq.handleDelete(project, client)
	case "pending_disable":
		return sq.handleDisable(project, client)
	case "pending_enable":
		return sq.handleEnable(project, client)
	default:
		return fmt.Errorf("未知的同步状态: %s", project.SyncStatus)
	}
}

func (sq *SyncQueue) handleCreate(project *ActivityProject, client *QingLongClient) error {
	duplicate, err := client.CheckDuplicateRemarks(project.Remarks, project.EnvKey)
	if err != nil {
		return fmt.Errorf("检查重复备注失败: %v", SanitizeError(err))
	}
	if duplicate {
		existingEnvs, err := client.QueryEnvByRemarks(fmt.Sprintf("%d", project.UserNumber), project.EnvKey)
		if err == nil && len(existingEnvs) > 0 {
			for _, env := range existingEnvs {
				if env.Remarks == project.Remarks {
					log.Printf("[同步服务] 青龙中已存在相同备注的变量，跳过创建并标记为已同步 ID=%d QLEnvID=%d", project.ID, env.ID)
					MarkProjectSynced(project.ID, env.ID)
					return nil
				}
			}
		}
	}

	if err := client.SubmitEnv(project.EnvKey, project.EnvValue, project.Remarks); err != nil {
		return fmt.Errorf("提交环境变量失败: %v", SanitizeError(err))
	}

	envs, err := client.QueryEnvByRemarks(fmt.Sprintf("%d", project.UserNumber), project.EnvKey)
	if err == nil {
		for _, env := range envs {
			if env.Remarks == project.Remarks {
				MarkProjectSynced(project.ID, env.ID)
				log.Printf("[同步服务] 创建成功 ID=%d QLEnvID=%d", project.ID, env.ID)
				return nil
			}
		}
	}

	MarkProjectSynced(project.ID, 0)
	log.Printf("[同步服务] 创建成功 ID=%d (未找到对应QL EnvID)", project.ID)
	return nil
}

func (sq *SyncQueue) handleUpdate(project *ActivityProject, client *QingLongClient) error {
	if project.QingLongEnvID > 0 {
		if err := client.UpdateEnv(project.QingLongEnvID, project.EnvKey, project.EnvValue, project.Remarks); err != nil {
			return fmt.Errorf("更新环境变量 %d 失败: %v", project.QingLongEnvID, SanitizeError(err))
		}
		MarkProjectSynced(project.ID, project.QingLongEnvID)
		log.Printf("[同步服务] 更新成功 ID=%d QLEnvID=%d", project.ID, project.QingLongEnvID)
		return nil
	}

	envItem, err := client.FindEnvByRemarks(project.Remarks, project.EnvKey)
	if err != nil {
		return fmt.Errorf("查找环境变量失败: %v", SanitizeError(err))
	}

	if err := client.UpdateEnv(envItem.ID, project.EnvKey, project.EnvValue, project.Remarks); err != nil {
		return fmt.Errorf("更新环境变量 %d 失败: %v", envItem.ID, SanitizeError(err))
	}

	MarkProjectSynced(project.ID, envItem.ID)
	log.Printf("[同步服务] 更新成功 ID=%d QLEnvID=%d", project.ID, envItem.ID)
	return nil
}

func (sq *SyncQueue) handleDelete(project *ActivityProject, client *QingLongClient) error {
	if project.QingLongEnvID > 0 {
		if err := client.DeleteEnv(project.QingLongEnvID); err != nil {
			sanitizedErr := SanitizeError(err)
			if sanitizedErr.Error() == "" {
				return fmt.Errorf("删除环境变量 %d 失败: %v", project.QingLongEnvID, err)
			}
		}
		MarkProjectDeleted(project.ID)
		log.Printf("[同步服务] 删除成功 ID=%d QLEnvID=%d", project.ID, project.QingLongEnvID)
		return nil
	}

	envItem, err := client.FindEnvByRemarks(project.Remarks, project.EnvKey)
	if err != nil {
		log.Printf("[同步服务] 青龙中未找到对应变量，直接标记为已删除 ID=%d", project.ID)
		MarkProjectDeleted(project.ID)
		return nil
	}

	if err := client.DeleteEnv(envItem.ID); err != nil {
		sanitizedErr := SanitizeError(err)
		if sanitizedErr.Error() == "" {
			return fmt.Errorf("删除环境变量 %d 失败: %v", envItem.ID, err)
		}
	}

	MarkProjectDeleted(project.ID)
	log.Printf("[同步服务] 删除成功 ID=%d QLEnvID=%d", project.ID, envItem.ID)
	return nil
}

func (sq *SyncQueue) handleDisable(project *ActivityProject, client *QingLongClient) error {
	if project.QingLongEnvID > 0 {
		if err := client.DisableEnvs([]int{project.QingLongEnvID}); err != nil {
			return fmt.Errorf("禁用环境变量 %d 失败: %v", project.QingLongEnvID, SanitizeError(err))
		}
		MarkProjectSynced(project.ID, project.QingLongEnvID)
		log.Printf("[同步服务] 禁用成功 ID=%d QLEnvID=%d", project.ID, project.QingLongEnvID)
		return nil
	}

	envItem, err := client.FindEnvByRemarks(project.Remarks, project.EnvKey)
	if err != nil {
		return fmt.Errorf("查找环境变量失败: %v", SanitizeError(err))
	}

	if err := client.DisableEnvs([]int{envItem.ID}); err != nil {
		return fmt.Errorf("禁用环境变量 %d 失败: %v", envItem.ID, SanitizeError(err))
	}

	MarkProjectSynced(project.ID, envItem.ID)
	log.Printf("[同步服务] 禁用成功 ID=%d QLEnvID=%d", project.ID, envItem.ID)
	return nil
}

func (sq *SyncQueue) handleEnable(project *ActivityProject, client *QingLongClient) error {
	if project.QingLongEnvID > 0 {
		if err := client.EnableEnv(project.QingLongEnvID); err != nil {
			return fmt.Errorf("启用环境变量 %d 失败: %v", project.QingLongEnvID, SanitizeError(err))
		}
		MarkProjectSynced(project.ID, project.QingLongEnvID)
		log.Printf("[同步服务] 启用成功 ID=%d QLEnvID=%d", project.ID, project.QingLongEnvID)
		return nil
	}

	envItem, err := client.FindEnvByRemarks(project.Remarks, project.EnvKey)
	if err != nil {
		return fmt.Errorf("查找环境变量失败: %v", SanitizeError(err))
	}

	if err := client.EnableEnv(envItem.ID); err != nil {
		return fmt.Errorf("启用环境变量 %d 失败: %v", envItem.ID, SanitizeError(err))
	}

	MarkProjectSynced(project.ID, envItem.ID)
	log.Printf("[同步服务] 启用成功 ID=%d QLEnvID=%d", project.ID, envItem.ID)
	return nil
}

func (sq *SyncQueue) performFullSyncCheck() {
	activityConfigsMu.RLock()
	configs := make([]*ActivityConfig, 0, len(ActivityConfigs))
	for _, cfg := range ActivityConfigs {
		if cfg != nil {
			configs = append(configs, cfg)
		}
	}
	activityConfigsMu.RUnlock()

	for _, cfg := range configs {
		qlConfig := qlManagerGetConfig(cfg.QingLongConfigName)
		if qlConfig == nil {
			continue
		}

		client := NewQingLongClient(qlConfig)

		dbProjects, err := GetActivityProjectsByActivityAndEnvKey(cfg.ID, cfg.EnvKey)
		if err != nil {
			log.Printf("[全量同步] 获取数据库项目失败 活动=%s: %v", cfg.ID, err)
			continue
		}

		qlEnvs, err := client.QueryEnvs(cfg.EnvKey)
		if err != nil {
			log.Printf("[全量同步] 获取青龙环境变量失败 活动=%s: %v", cfg.ID, err)
			continue
		}

		dbRemarksMap := make(map[string]*ActivityProject)
		for i := range dbProjects {
			dbRemarksMap[dbProjects[i].Remarks] = &dbProjects[i]
		}

		qlRemarksMap := make(map[string]QLEnvItem)
		for _, env := range qlEnvs {
			qlRemarksMap[env.Remarks] = env
		}

		// 数据库有但青龙没有 → 需要创建
		for remarks, dbProj := range dbRemarksMap {
			if _, exists := qlRemarksMap[remarks]; !exists {
				log.Printf("[全量同步] 数据库有但青龙缺失 Remarks=%s，触发创建同步", remarks)
				db.Model(&ActivityProject{}).Where("id = ?", dbProj.ID).
					Updates(map[string]interface{}{"sync_status": "pending", "updated_at": time.Now()})
			}
		}

		// 青龙有但数据库没有（可能是旧数据或外部直接创建的）→ 不计入统计
		for remarks, qlEnv := range qlRemarksMap {
			if _, exists := dbRemarksMap[remarks]; !exists {
				log.Printf("[全量同步] 青龙有但数据库缺失 Remarks=%s QLEnvID=%d，跳过", remarks, qlEnv.ID)
			}
		}

		// 两边都有，确保 qinglong_env_id 正确
		for remarks, dbProj := range dbRemarksMap {
			if qlEnv, exists := qlRemarksMap[remarks]; exists {
				if dbProj.QingLongEnvID != qlEnv.ID {
					log.Printf("[全量同步] 修正EnvID ID=%d DB=%d → QL=%d", dbProj.ID, dbProj.QingLongEnvID, qlEnv.ID)
					db.Model(&ActivityProject{}).Where("id = ?", dbProj.ID).
						Updates(map[string]interface{}{
							"qinglong_env_id": qlEnv.ID,
							"updated_at":      time.Now(),
						})
				}
				if dbProj.Status != qlEnv.Status {
					log.Printf("[全量同步] 状态不一致 Remarks=%s DBStatus=%d QLStatus=%d", remarks, dbProj.Status, qlEnv.Status)
				}
			}
		}
	}

	log.Println("[全量同步] 全量同步检查完成")
}

func TriggerSync(projectID int) {
	sq := GetSyncQueue()
	go func() {
		time.Sleep(500 * time.Millisecond)
		var project ActivityProject
		if err := db.Unscoped().Where("id = ?", projectID).First(&project).Error; err != nil {
			log.Printf("[同步触发] 获取项目失败 ID=%d: %v", projectID, err)
			return
		}
		if err := sq.syncProject(&project); err != nil {
			log.Printf("[同步触发] 同步失败 ID=%d: %v", projectID, err)
		}
	}()
}

func qlManagerGetConfig(name string) *QingLongConfig {
	qlManager.mu.RLock()
	defer qlManager.mu.RUnlock()

	if name == "" {
		name = qlManager.Default
	}

	cfg, exists := qlManager.Configs[name]
	if !exists {
		for _, c := range qlManager.Configs {
			return c
		}
		return nil
	}
	return cfg
}

// SanitizeError 已定义在 jltask.go 中，这里仅引用

func MigrateFromQingLongToDB() {
	log.Println("[数据迁移] 开始从青龙迁移数据到数据库...")

	activityConfigsMu.RLock()
	configs := make([]*ActivityConfig, 0, len(ActivityConfigs))
	for _, cfg := range ActivityConfigs {
		if cfg != nil {
			configs = append(configs, cfg)
		}
	}
	activityConfigsMu.RUnlock()

	// 检查是否已有数据
	// 检查表是否存在
	if !db.Migrator().HasTable(&ActivityProject{}) {
		log.Println("[数据迁移] activity_project 表不存在，请先运行程序完成数据库迁移")
		return
	}

	var count int64
	if err := db.Model(&ActivityProject{}).Count(&count).Error; err != nil {
		log.Printf("[数据迁移] 查询数据库失败: %v，请检查 activity_project 表是否存在", err)
		return
	}
	if count > 0 {
		log.Printf("[数据迁移] 数据库已有 %d 条记录，跳过迁移", count)
		return
	}

	totalMigrated := 0
	for _, cfg := range configs {
		qlConfig := qlManagerGetConfig(cfg.QingLongConfigName)
		if qlConfig == nil {
			continue
		}

		client := NewQingLongClient(qlConfig)
		envs, err := client.QueryEnvs(cfg.EnvKey)
		if err != nil {
			log.Printf("[数据迁移] 查询青龙环境变量失败 活动=%s: %v", cfg.Name, err)
			continue
		}

		for _, env := range envs {
			userID := ExtractUserIDFromRemarks(env.Remarks)
			userNumber := 0
			fmt.Sscanf(userID, "%d", &userNumber)

			remarkAlias := GetFirstRemarkParam(env.Remarks)
			expireDate := ""
			if cfg.IsMonthlyDeduct {
				if d, ok := ParseRemarksDate(env.Remarks); ok {
					expireDate = d.Format(DateLayout)
				}
			}

			project := &ActivityProject{
				ActivityID:         cfg.ID,
				ActivityName:       cfg.Name,
				EnvKey:             cfg.EnvKey,
				EnvValue:           env.Value,
				Remarks:            env.Remarks,
				RemarkAlias:        remarkAlias,
				UserNumber:         userNumber,
				QingLongConfigName: cfg.QingLongConfigName,
				QingLongEnvID:      env.ID,
				Status:             env.Status,
				ExpireDate:         expireDate,
				IsMonthlyDeduct:    cfg.IsMonthlyDeduct,
				MonthlyCoin:        cfg.MonthlyCoin,
				SyncStatus:         "synced",
				CreatedAt:          time.Now(),
				UpdatedAt:          time.Now(),
			}
			if !cfg.IsMonthlyDeduct {
				project.NeedCoin = cfg.NeedCoin
			}

			if err := db.Create(project).Error; err != nil {
				log.Printf("[数据迁移] 创建项目失败 Remarks=%s: %v", env.Remarks, err)
				continue
			}
			totalMigrated++
		}
	}

	log.Printf("[数据迁移] 迁移完成，共 %d 条记录", totalMigrated)
}

var _ = log.Printf