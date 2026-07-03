package models

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

type SyncAction string

const (
	SyncActionCreate   SyncAction = "create"
	SyncActionUpdate   SyncAction = "update"
	SyncActionDelete   SyncAction = "delete"
	SyncActionDisable  SyncAction = "disable"
	SyncActionEnable   SyncAction = "enable"
	SyncActionFullSync SyncAction = "full_sync"
)

type SyncQueue struct {
	mu         sync.Mutex
	processing bool
	running    bool
	stopChan   chan struct{}
	ticker     *time.Ticker
}

var (
	syncQueue        *SyncQueue
	syncQueueOnce    sync.Once
	projectSyncLocks sync.Map
)

const (
	syncInterval     = 10 * time.Minute
	syncBatchSize    = 100
	syncMaxRetries   = 3
	fullSyncInterval = 1 * time.Hour
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

	Sync().Infof("[同步服务] 数据库→青龙同步服务已启动")
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
	Sync().Infof("[同步服务] 同步服务已停止")
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
			Sync().Infof("[同步服务] 开始全量同步检查...")
			sq.performFullSyncCheck()
		}
	}
}

func getProjectSyncLock(projectID int) *sync.Mutex {
	lock, _ := projectSyncLocks.LoadOrStore(projectID, &sync.Mutex{})
	return lock.(*sync.Mutex)
}

func (sq *SyncQueue) syncProjectSafe(projectID int) error {
	lock := getProjectSyncLock(projectID)
	lock.Lock()
	defer lock.Unlock()

	var project ActivityProject
	if err := db.Unscoped().Where("id = ?", projectID).First(&project).Error; err != nil {
		return fmt.Errorf("获取项目失败: %v", err)
	}
	return sq.syncProject(&project)
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
		Sync().Infof("[同步服务] 获取待同步项目失败: %v", err)
		return
	}

	if len(projects) == 0 {
		return
	}

	totalPending := CountPendingSyncProjects()
	if totalPending > int64(len(projects)) {
		Sync().Infof("[同步服务] 待同步队列共 %d 条，本批处理 %d 条", totalPending, len(projects))
	} else {
		Sync().Infof("[同步服务] 本批处理 %d 个待同步项目", len(projects))
	}

	for _, project := range projects {
		if err := sq.syncProjectSafe(project.ID); err != nil {
			Sync().Infof("[同步服务] 同步项目失败 ID=%d Remarks=%s: %v", project.ID, project.Remarks, err)
			_ = MarkProjectSyncError(project.ID, err.Error())
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
	case "error":
		return sq.handleSyncErrorRecovery(project, client)
	default:
		return fmt.Errorf("未知的同步状态: %s", project.SyncStatus)
	}
}

func (sq *SyncQueue) handleSyncErrorRecovery(project *ActivityProject, client *QingLongClient) error {
	if project.DeletedAt != nil {
		project.SyncStatus = "pending_delete"
		return sq.handleDelete(project, client)
	}
	if project.Status != 0 {
		project.SyncStatus = "pending_disable"
		return sq.handleDisable(project, client)
	}
	project.SyncStatus = "pending_update"
	return sq.handleUpdate(project, client)
}

func (sq *SyncQueue) applyDBStatusToQL(client *QingLongClient, project *ActivityProject, envID int) error {
	if envID <= 0 {
		return nil
	}
	if project.Status != 0 {
		if err := client.DisableEnvs([]int{envID}); err != nil && !IsQLEnvNotFoundError(err) {
			return err
		}
		return nil
	}
	if err := client.EnableEnv(envID); err != nil && !IsQLEnvNotFoundError(err) {
		return err
	}
	return nil
}

func (sq *SyncQueue) bindExistingEnv(project *ActivityProject, client *QingLongClient, env *QLEnvItem) error {
	if env.Value != project.EnvValue || env.Name != project.EnvKey || env.Remarks != project.Remarks {
		if err := client.UpdateEnvContent(env.ID, project.EnvKey, project.EnvValue, project.Remarks); err != nil {
			return fmt.Errorf("绑定已有环境变量并更新失败: %v", SanitizeError(err))
		}
	}
	if err := sq.applyDBStatusToQL(client, project, env.ID); err != nil {
		return fmt.Errorf("同步青龙状态失败: %v", SanitizeError(err))
	}
	if err := MarkProjectSynced(project.ID, env.ID); err != nil {
		return err
	}
	Sync().Infof("[同步服务] 绑定已有青龙变量 ID=%d QLEnvID=%d", project.ID, env.ID)
	return nil
}

func (sq *SyncQueue) handleCreate(project *ActivityProject, client *QingLongClient) error {
	if env, how, err := client.FindEnvForProject(project); err == nil {
		Sync().Infof("[同步服务] 创建前发现已有青龙变量 ID=%d 匹配方式=%s", env.ID, how)
		return sq.bindExistingEnv(project, client, env)
	}

	if err := client.SubmitEnv(project.EnvKey, project.EnvValue, project.Remarks); err != nil {
		if isQLUniqueConstraintError(err) {
			if env, how, findErr := client.FindEnvForProject(project); findErr == nil {
				Sync().Infof("[同步服务] 重复创建，绑定已有青龙变量 ID=%d 匹配方式=%s", env.ID, how)
				return sq.bindExistingEnv(project, client, env)
			}
			return FormatQLSyncError("提交环境变量", fmt.Errorf("青龙中已有重复记录但未能自动绑定，请在青龙检查备注/CK是否与数据库一致；原始错误: %v", SanitizeError(err)), project)
		}
		return FormatQLSyncError("提交环境变量", err, project)
	}

	env, how, err := client.FindEnvForProject(project)
	if err != nil {
		return FormatQLSyncError("创建后查找环境变量", err, project)
	}
	Sync().Infof("[同步服务] 创建后定位青龙变量 ID=%d 匹配方式=%s", env.ID, how)

	if err := sq.applyDBStatusToQL(client, project, env.ID); err != nil {
		return FormatQLSyncError("创建后同步状态", err, project)
	}
	if err := MarkProjectSynced(project.ID, env.ID); err != nil {
		return err
	}
	Sync().Infof("[同步服务] 创建成功 ID=%d QLEnvID=%d", project.ID, env.ID)
	return nil
}

func (sq *SyncQueue) handleUpdate(project *ActivityProject, client *QingLongClient) error {
	envItem, err := client.ResolveEnvByProject(project)
	if err != nil {
		_ = db.Model(&ActivityProject{}).Where("id = ?", project.ID).Updates(map[string]interface{}{
			"qinglong_env_id":  0,
			"sync_status":      "pending",
			"sync_error":       "",
			"sync_retry_count": 0,
			"updated_at":       time.Now(),
		}).Error
		Sync().Infof("[同步服务] 青龙变量不存在，ID=%d 转为重新创建", project.ID)
		go TriggerSync(project.ID)
		return nil
	}

	if err := client.UpdateEnvContent(envItem.ID, project.EnvKey, project.EnvValue, project.Remarks); err != nil {
		if IsQLEnvNotFoundError(err) {
			_ = db.Model(&ActivityProject{}).Where("id = ?", project.ID).Updates(map[string]interface{}{
				"qinglong_env_id":  0,
				"sync_status":      "pending",
				"sync_error":       "",
				"sync_retry_count": 0,
				"updated_at":       time.Now(),
			}).Error
			Sync().Infof("[同步服务] 青龙变量已失效，ID=%d 转为重新创建", project.ID)
			go TriggerSync(project.ID)
			return nil
		}
		// 过期 EnvID 或备注已变更时，按备注重新定位再更新
		if strings.Contains(err.Error(), "Validation error") || isQLUniqueConstraintError(err) {
			if fresh, _, findErr := client.FindEnvForProject(project); findErr == nil && fresh.ID != envItem.ID {
				Sync().Infof("[同步服务] 更新失败，改按备注绑定 ID=%d → QL=%d", project.ID, fresh.ID)
				if err2 := client.UpdateEnvContent(fresh.ID, project.EnvKey, project.EnvValue, project.Remarks); err2 != nil {
					return FormatQLSyncError(fmt.Sprintf("更新环境变量 %d", fresh.ID), err2, project)
				}
				if err := sq.applyDBStatusToQL(client, project, fresh.ID); err != nil {
					return FormatQLSyncError("更新后同步状态", err, project)
				}
				return MarkProjectSynced(project.ID, fresh.ID)
			}
		}
		return FormatQLSyncError(fmt.Sprintf("更新环境变量 %d", envItem.ID), err, project)
	}

	if err := sq.applyDBStatusToQL(client, project, envItem.ID); err != nil {
		return FormatQLSyncError("更新后同步状态", err, project)
	}
	if err := MarkProjectSynced(project.ID, envItem.ID); err != nil {
		return err
	}
	Sync().Infof("[同步服务] 更新成功 ID=%d QLEnvID=%d", project.ID, envItem.ID)
	return nil
}

func (sq *SyncQueue) handleDelete(project *ActivityProject, client *QingLongClient) error {
	envItem, err := client.ResolveEnvByProject(project)
	if err != nil {
		Sync().Infof("[同步服务] 青龙中未找到对应变量，直接标记为已删除 ID=%d", project.ID)
		return MarkProjectDeleted(project.ID)
	}

	if err := client.DeleteEnv(envItem.ID); err != nil {
		if IsQLEnvNotFoundError(err) {
			return MarkProjectDeleted(project.ID)
		}
		return fmt.Errorf("删除环境变量 %d 失败: %v", envItem.ID, SanitizeError(err))
	}

	if _, verifyErr := client.FindEnvByRemarks(project.Remarks, project.EnvKey); verifyErr == nil {
		return fmt.Errorf("删除环境变量 %d 后验证失败，青龙中仍存在", envItem.ID)
	}

	if err := MarkProjectDeleted(project.ID); err != nil {
		return err
	}
	Sync().Infof("[同步服务] 删除成功 ID=%d QLEnvID=%d", project.ID, envItem.ID)
	return nil
}

func (sq *SyncQueue) handleDisable(project *ActivityProject, client *QingLongClient) error {
	envItem, err := client.ResolveEnvByProject(project)
	if err != nil {
		if IsQLEnvNotFoundError(err) {
			if err := MarkProjectSynced(project.ID, 0); err != nil {
				return err
			}
			Sync().Infof("[同步服务] 青龙中无对应变量，禁用视为完成 ID=%d", project.ID)
			return nil
		}
		return fmt.Errorf("禁用失败，解析青龙变量失败: %v", SanitizeError(err))
	}

	if err := client.DisableEnvs([]int{envItem.ID}); err != nil {
		if IsQLEnvNotFoundError(err) {
			return MarkProjectSynced(project.ID, 0)
		}
		return fmt.Errorf("禁用环境变量 %d 失败: %v", envItem.ID, SanitizeError(err))
	}

	if err := MarkProjectSynced(project.ID, envItem.ID); err != nil {
		return err
	}
	Sync().Infof("[同步服务] 禁用成功 ID=%d QLEnvID=%d", project.ID, envItem.ID)
	return nil
}

func (sq *SyncQueue) handleEnable(project *ActivityProject, client *QingLongClient) error {
	envItem, err := client.ResolveEnvByProject(project)
	if err != nil {
		return fmt.Errorf("启用失败，青龙中未找到变量: %v", SanitizeError(err))
	}

	if envItem.Value != project.EnvValue || envItem.Remarks != project.Remarks {
		if err := client.UpdateEnvContent(envItem.ID, project.EnvKey, project.EnvValue, project.Remarks); err != nil {
			if !IsQLEnvNotFoundError(err) {
				return fmt.Errorf("启用前更新内容失败: %v", SanitizeError(err))
			}
		}
	}

	if err := client.EnableEnv(envItem.ID); err != nil {
		if IsQLEnvNotFoundError(err) {
			return fmt.Errorf("启用失败，青龙变量 %d 不存在", envItem.ID)
		}
		return fmt.Errorf("启用环境变量 %d 失败: %v", envItem.ID, SanitizeError(err))
	}

	if err := MarkProjectSynced(project.ID, envItem.ID); err != nil {
		return err
	}
	Sync().Infof("[同步服务] 启用成功 ID=%d QLEnvID=%d", project.ID, envItem.ID)
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
			Sync().Infof("[全量同步] 获取数据库项目失败 活动=%s: %v", cfg.ID, err)
			continue
		}

		qlEnvs, err := client.QueryEnvs(cfg.EnvKey)
		if err != nil {
			Sync().Infof("[全量同步] 获取青龙环境变量失败 活动=%s: %v", cfg.ID, err)
			continue
		}

		matchedQLEnvID := make(map[int]bool)
		for i := range dbProjects {
			dbProj := &dbProjects[i]
			if dbProj.SyncStatus != "synced" && dbProj.SyncStatus != "error" {
				continue
			}

			env, matchHow, findErr := client.FindEnvForProject(dbProj)
			updates := map[string]interface{}{}

			if findErr != nil {
				Sync().Infof("[全量同步] 数据库有但青龙未匹配 ID=%d Remarks=%s", dbProj.ID, dbProj.Remarks)
				updates["sync_status"] = "pending"
				updates["sync_error"] = ""
				updates["sync_retry_count"] = 0
			} else {
				matchedQLEnvID[env.ID] = true
				if dbProj.QingLongEnvID != env.ID {
					Sync().Infof("[全量同步] 修正EnvID ID=%d DB=%d → QL=%d (%s)", dbProj.ID, dbProj.QingLongEnvID, env.ID, matchHow)
					updates["qinglong_env_id"] = env.ID
				}

				dbEnabled := dbProj.Status == 0
				qlEnabled := env.Status == 1
				if dbEnabled != qlEnabled {
					Sync().Infof("[全量同步] 状态不一致 ID=%d Remarks=%s DB启用=%v QL启用=%v", dbProj.ID, dbProj.Remarks, dbEnabled, qlEnabled)
					if !dbEnabled {
						updates["sync_status"] = "pending_disable"
					} else {
						updates["sync_status"] = "pending_enable"
					}
					updates["sync_error"] = ""
					updates["sync_retry_count"] = 0
				} else if dbProj.EnvValue != env.Value || strings.TrimSpace(dbProj.Remarks) != strings.TrimSpace(env.Remarks) {
					Sync().Infof("[全量同步] 内容/备注不一致 ID=%d，触发更新同步", dbProj.ID)
					updates["sync_status"] = "pending_update"
					updates["sync_error"] = ""
					updates["sync_retry_count"] = 0
				}
			}

			if len(updates) > 0 {
				updates["updated_at"] = time.Now()
				db.Model(&ActivityProject{}).Where("id = ?", dbProj.ID).Updates(updates)
			}
		}

		for _, env := range qlEnvs {
			if env.Name != cfg.EnvKey || matchedQLEnvID[env.ID] {
				continue
			}
			Sync().Infof("[全量同步] 青龙有但数据库未匹配 Remarks=%s QLEnvID=%d，跳过", env.Remarks, env.ID)
		}
	}

	Sync().Infof("[全量同步] 全量同步检查完成")
}

func TriggerSync(projectID int) {
	sq := GetSyncQueue()
	go func() {
		time.Sleep(500 * time.Millisecond)
		if err := sq.syncProjectSafe(projectID); err != nil {
			Sync().Infof("[同步触发] 同步失败 ID=%d: %v", projectID, err)
			_ = MarkProjectSyncError(projectID, err.Error())
		}
	}()
}

// SyncProjectNow 同步执行青龙同步（删除/禁用等需即时生效的场景）
func SyncProjectNow(projectID int) error {
	return GetSyncQueue().syncProjectSafe(projectID)
}

func qlManagerGetConfig(name string) *QingLongConfig {
	qlManager.mu.RLock()
	defer qlManager.mu.RUnlock()

	if name == "" {
		name = qlManager.Default
	}

	cfg, exists := qlManager.Configs[name]
	if !exists {
		return nil
	}
	return cfg
}
