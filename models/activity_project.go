package models

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"
)

var (
	projectSubmitMu       sync.Mutex
	projectSubmitInFlight = make(map[string]time.Time)
)

const projectSubmitLockTTL = 90 * time.Second

type ActivityProject struct {
	ID                 int        `gorm:"primaryKey;autoIncrement"`
	ActivityID         string     `gorm:"column:activity_id;size:64;index"`
	ActivityName       string     `gorm:"column:activity_name;size:128"`
	EnvKey             string     `gorm:"column:env_key;size:128;index"`
	EnvValue           string     `gorm:"column:env_value;type:text"`
	Remarks            string     `gorm:"column:remarks;type:text;index"`
	RemarkAlias        string     `gorm:"column:remark_alias;size:128"`
	UserNumber         int        `gorm:"column:user_number;index"`
	QingLongConfigName string     `gorm:"column:qinglong_config_name;size:64"`
	QingLongEnvID      int        `gorm:"column:qinglong_env_id;default:0"`
	Status             int        `gorm:"column:status;default:0"`
	ExpireDate         string     `gorm:"column:expire_date;size:10"`
	IsMonthlyDeduct    bool       `gorm:"column:is_monthly_deduct;default:false"`
	MonthlyCoin        int        `gorm:"column:monthly_coin;default:0"`
	IsDailyDeduct      bool       `gorm:"column:is_daily_deduct;default:false"`
	DailyCoin          int        `gorm:"column:daily_coin;default:0"`
	MinDays            *int       `gorm:"column:min_days"`
	NeedCoin           int        `gorm:"column:need_coin;default:0"`
	GrantExpireDate    string     `gorm:"column:grant_expire_date;size:10"`
	AdminGrantDays     int        `gorm:"column:admin_grant_days;default:0"` // 后台赠送天数（删号不退积分）
	CreatedAt          time.Time  `gorm:"column:created_at"`
	UpdatedAt          time.Time  `gorm:"column:updated_at"`
	DeletedAt          *time.Time `gorm:"column:deleted_at;index"`
	SyncStatus         string     `gorm:"column:sync_status;size:16;default:'pending'"`
	SyncError          string     `gorm:"column:sync_error;type:text"`
	SyncRetryCount     int        `gorm:"column:sync_retry_count;default:0"`
	SyncAt             *time.Time `gorm:"column:sync_at"`
}

func (ActivityProject) TableName() string {
	return "activity_project"
}

// NormalizeProjectBilling 互斥计费字段：按月则日分为0，按天则月分为0
func NormalizeProjectBilling(project *ActivityProject) {
	if project == nil {
		return
	}
	if project.IsDailyDeduct && !project.IsMonthlyDeduct {
		project.MonthlyCoin = 0
		project.IsMonthlyDeduct = false
		project.NeedCoin = 0
		return
	}
	if project.IsMonthlyDeduct && !project.IsDailyDeduct {
		project.DailyCoin = 0
		project.IsDailyDeduct = false
		project.MinDays = nil
		project.NeedCoin = 0
		return
	}
	if !project.IsMonthlyDeduct && !project.IsDailyDeduct {
		project.MonthlyCoin = 0
		project.DailyCoin = 0
		project.MinDays = nil
	}
}

// ApplyConfigBillingToProject 从活动配置写入项目计费字段（已规范化）
func ApplyConfigBillingToProject(project *ActivityProject, cfg *ActivityConfig) {
	if project == nil || cfg == nil {
		return
	}
	cfg.NormalizeBilling()
	project.IsMonthlyDeduct = cfg.IsMonthlyDeduct
	project.IsDailyDeduct = cfg.IsDailyDeduct
	project.MonthlyCoin = cfg.MonthlyCoin
	project.DailyCoin = cfg.DailyCoin
	project.NeedCoin = 0
	project.MinDays = nil
	if cfg.IsDailyDeduct && cfg.MinDays > 0 {
		minDays := cfg.MinDays
		project.MinDays = &minDays
	}
	if !cfg.IsMonthlyDeduct && !cfg.IsDailyDeduct {
		project.NeedCoin = cfg.NeedCoin
	}
}

// RepairActivityProjectBillingRecords 修正历史数据中互斥计费字段，并回填空 remark_alias
func RepairActivityProjectBillingRecords() {
	if db == nil {
		return
	}
	db.Model(&ActivityProject{}).
		Where("is_monthly_deduct = ? AND is_daily_deduct = ?", true, false).
		Updates(map[string]interface{}{"daily_coin": 0})
	db.Model(&ActivityProject{}).
		Where("is_daily_deduct = ? AND is_monthly_deduct = ?", true, false).
		Updates(map[string]interface{}{"monthly_coin": 0})
	db.Model(&ActivityProject{}).
		Where("is_monthly_deduct = ? AND is_daily_deduct = ?", false, false).
		Updates(map[string]interface{}{"monthly_coin": 0, "daily_coin": 0})

	var emptyAlias []ActivityProject
	if err := db.Where("deleted_at IS NULL AND (remark_alias = '' OR remark_alias IS NULL) AND remarks != ''").
		Find(&emptyAlias).Error; err != nil {
		return
	}
	for i := range emptyAlias {
		alias := GetFirstRemarkParam(emptyAlias[i].Remarks)
		if alias == "" {
			continue
		}
		_ = db.Model(&ActivityProject{}).Where("id = ?", emptyAlias[i].ID).
			Update("remark_alias", alias).Error
	}
}

func CalcPaidRemainingDays(project *ActivityProject) int {
	if project.ExpireDate == "" {
		return 0
	}
	expireDate, err := time.Parse(DateLayout, project.ExpireDate)
	if err != nil {
		return 0
	}
	now := time.Now()
	expireThreshold := time.Date(expireDate.Year(), expireDate.Month(), expireDate.Day(), 0, 0, 0, 0, time.Local).AddDate(0, 0, 1)
	totalRemaining := expireThreshold.Sub(now).Hours() / 24
	if totalRemaining < 0 {
		totalRemaining = 0
	}
	if totalRemaining > 0 {
		totalRemaining = totalRemaining - 1
	}

	grantedRemaining := 0.0
	if project.GrantExpireDate != "" {
		grantDate, err := time.Parse(DateLayout, project.GrantExpireDate)
		if err == nil {
			grantThreshold := time.Date(grantDate.Year(), grantDate.Month(), grantDate.Day(), 0, 0, 0, 0, time.Local).AddDate(0, 0, 1)
			diff := grantThreshold.Sub(now).Hours() / 24
			if diff > 1 {
				grantedRemaining = diff - 1
			}
		}
	}

	adminGrant := project.AdminGrantDays
	if adminGrant < 0 {
		adminGrant = 0
	}
	if adminGrant > int(totalRemaining) {
		adminGrant = int(totalRemaining)
	}
	nonRefundable := grantedRemaining + float64(adminGrant)
	if nonRefundable > totalRemaining {
		nonRefundable = totalRemaining
	}

	paidDays := totalRemaining - nonRefundable
	if paidDays < 0 {
		paidDays = 0
	}
	return int(paidDays)
}

func CreateActivityProject(project *ActivityProject) error {
	NormalizeProjectBilling(project)

	if project.RemarkAlias == "" {
		project.RemarkAlias = GetFirstRemarkParam(project.Remarks)
	}

	// 同用户 + 同活动：备注别名不可重复
	if project.UserNumber > 0 && project.ActivityID != "" && project.RemarkAlias != "" {
		exists, err := HasActiveProjectByUserActivityAlias(project.UserNumber, project.ActivityID, project.RemarkAlias)
		if err != nil {
			return fmt.Errorf("检查重复备注失败：%v", err)
		}
		if exists {
			return fmt.Errorf("该账号备注已存在，请勿重复提交")
		}
	}

	duplicate, err := CheckDuplicateRemarksDB(project.ActivityID, project.Remarks, project.EnvKey)
	if err != nil {
		return fmt.Errorf("检查重复备注失败：%v", err)
	}
	if duplicate {
		return fmt.Errorf("该账号备注已存在，请勿重复提交")
	}

	now := time.Now()
	project.CreatedAt = now
	project.UpdatedAt = now
	project.SyncStatus = "pending"
	project.SyncRetryCount = 0
	project.SyncError = ""

	return db.Create(project).Error
}

// projectSubmitLockKey 上车防重复提交锁键（用户+活动+备注别名）
func projectSubmitLockKey(userNumber int, activityID, remarkAlias string) string {
	return fmt.Sprintf("%d:%s:%s", userNumber, activityID, strings.TrimSpace(remarkAlias))
}

// TryAcquireProjectSubmitLock 获取上车提交锁，防止连点重复扣费
func TryAcquireProjectSubmitLock(userNumber int, activityID, remarkAlias string) (release func(), acquired bool) {
	return tryAcquireProjectLock(projectSubmitLockKey(userNumber, activityID, remarkAlias))
}

// TryAcquireProjectActionLock 获取续费/删除等操作锁，防止重复提交
func TryAcquireProjectActionLock(userNumber int, activityID, action, remarks string) (release func(), acquired bool) {
	key := fmt.Sprintf("%d:%s:%s:%s", userNumber, activityID, action, remarks)
	return tryAcquireProjectLock(key)
}

func tryAcquireProjectLock(key string) (release func(), acquired bool) {
	now := time.Now()

	projectSubmitMu.Lock()
	defer projectSubmitMu.Unlock()

	for k, exp := range projectSubmitInFlight {
		if now.After(exp) {
			delete(projectSubmitInFlight, k)
		}
	}

	if exp, exists := projectSubmitInFlight[key]; exists && now.Before(exp) {
		return nil, false
	}
	projectSubmitInFlight[key] = now.Add(projectSubmitLockTTL)

	return func() {
		projectSubmitMu.Lock()
		delete(projectSubmitInFlight, key)
		projectSubmitMu.Unlock()
	}, true
}

// HasActiveProjectByUserActivityAlias 检查用户在某活动下是否已有相同备注别名
// 兼容历史数据 remark_alias 为空：按 remarks 前缀「别名/」或整段别名匹配
func HasActiveProjectByUserActivityAlias(userNumber int, activityID, remarkAlias string) (bool, error) {
	remarkAlias = strings.TrimSpace(remarkAlias)
	if remarkAlias == "" {
		return false, nil
	}
	var count int64
	err := db.Model(&ActivityProject{}).
		Where("user_number = ? AND activity_id = ? AND deleted_at IS NULL", userNumber, activityID).
		Where("(remark_alias = ? OR ((remark_alias = '' OR remark_alias IS NULL) AND (remarks = ? OR remarks LIKE ?)))",
			remarkAlias, remarkAlias, remarkAlias+"/%").
		Count(&count).Error
	return count > 0, err
}

func UpdateActivityProject(project *ActivityProject) error {
	NormalizeProjectBilling(project)
	project.UpdatedAt = time.Now()
	return db.Save(project).Error
}

// renewProjectSnapshot 续费前快照，青龙同步失败时用于回滚 DB
type renewProjectSnapshot struct {
	Remarks    string
	ExpireDate string
	Status     int
	SyncStatus string
	SyncError  string
	NeedCoin   int
}

func SnapshotRenewProject(project *ActivityProject) renewProjectSnapshot {
	if project == nil {
		return renewProjectSnapshot{}
	}
	return renewProjectSnapshot{
		Remarks:    project.Remarks,
		ExpireDate: project.ExpireDate,
		Status:     project.Status,
		SyncStatus: project.SyncStatus,
		SyncError:  project.SyncError,
		NeedCoin:   project.NeedCoin,
	}
}

// FinishRenewWithQLSync 续费入库后立刻同步青龙；失败则回滚数据库字段
func FinishRenewWithQLSync(project *ActivityProject, snap renewProjectSnapshot) error {
	if project == nil {
		return fmt.Errorf("项目不存在")
	}
	if err := SyncProjectNow(project.ID); err != nil {
		project.Remarks = snap.Remarks
		project.ExpireDate = snap.ExpireDate
		project.Status = snap.Status
		project.SyncStatus = snap.SyncStatus
		project.SyncError = snap.SyncError
		project.NeedCoin = snap.NeedCoin
		if uerr := UpdateActivityProject(project); uerr != nil {
			Sync().Infof("[续费] 青龙同步失败且回滚 DB 失败 ID=%d syncErr=%v rollbackErr=%v", project.ID, err, uerr)
			return fmt.Errorf("青龙同步失败，且数据库回滚也失败，请联系管理员处理（项目ID=%d）", project.ID)
		}
		return FormatQLSyncError("续费后同步青龙", err, project)
	}
	return nil
}

func GetActivityProjectByID(id int) (*ActivityProject, error) {
	var project ActivityProject
	err := db.Where("id = ? AND deleted_at IS NULL", id).First(&project).Error
	if err != nil {
		return nil, err
	}
	return &project, nil
}

func GetActivityProjectByRemarks(activityID, remarks, envKey string) (*ActivityProject, error) {
	var project ActivityProject
	err := db.Where("activity_id = ? AND remarks = ? AND env_key = ? AND deleted_at IS NULL",
		activityID, remarks, envKey).First(&project).Error
	if err != nil {
		return nil, err
	}
	return &project, nil
}

// CheckDuplicateRemarksDB 检查同活动下备注是否重复
// 规则：完整备注相同，或「备注别名 + 用户编号」相同（忽略到期日）
func CheckDuplicateRemarksDB(activityID, remarks, envKey string) (bool, error) {
	remarks = strings.TrimSpace(remarks)
	envKey = strings.TrimSpace(envKey)
	activityID = strings.TrimSpace(activityID)
	if remarks == "" || envKey == "" {
		return false, nil
	}

	q := db.Model(&ActivityProject{}).Where("env_key = ? AND deleted_at IS NULL", envKey)
	if activityID != "" {
		q = q.Where("activity_id = ?", activityID)
	}

	var count int64
	if err := q.Where("remarks = ?", remarks).Count(&count).Error; err != nil {
		return false, err
	}
	if count > 0 {
		return true, nil
	}

	alias := GetFirstRemarkParam(remarks)
	uid := ExtractUserIDFromRemarks(remarks)
	if alias == "" || uid == "" {
		return false, nil
	}

	var candidates []ActivityProject
	cq := db.Where("env_key = ? AND deleted_at IS NULL", envKey)
	if activityID != "" {
		cq = cq.Where("activity_id = ?", activityID)
	}
	// 先按别名收窄，再在应用层精确比对用户号（避免 LIKE %/123/% 误伤 9123）
	if err := cq.Where("remark_alias = ? OR remarks = ? OR remarks LIKE ?",
		alias, alias, alias+"/%").Find(&candidates).Error; err != nil {
		return false, err
	}
	for _, p := range candidates {
		pAlias := strings.TrimSpace(p.RemarkAlias)
		if pAlias == "" {
			pAlias = GetFirstRemarkParam(p.Remarks)
		}
		if pAlias == alias && ExtractUserIDFromRemarks(p.Remarks) == uid {
			return true, nil
		}
	}
	return false, nil
}

func GetActivityProjectsByUser(userNumber int) ([]ActivityProject, error) {
	var projects []ActivityProject
	err := db.Where("user_number = ? AND deleted_at IS NULL", userNumber).
		Order("activity_id ASC, remark_alias ASC").
		Find(&projects).Error
	if err != nil {
		return nil, err
	}
	return projects, nil
}

func GetActivityProjectsByActivity(activityID string) ([]ActivityProject, error) {
	var projects []ActivityProject
	err := db.Where("activity_id = ? AND deleted_at IS NULL", activityID).
		Find(&projects).Error
	if err != nil {
		return nil, err
	}
	return projects, nil
}

func GetActivityProjectsByActivityID(activityID string) ([]ActivityProject, error) {
	return GetActivityProjectsByActivity(activityID)
}

// SyncActivityProjectNames 将已上车记录的活动名称与当前配置对齐（管理员改名后同步）
func SyncActivityProjectNames(configs []*ActivityConfig) int {
	if db == nil || len(configs) == 0 {
		return 0
	}
	ids := make([]string, 0, len(configs))
	var caseSQL strings.Builder
	caseSQL.WriteString("CASE activity_id ")
	args := make([]interface{}, 0, len(configs)*2+1+len(configs))
	for _, cfg := range configs {
		if cfg == nil || cfg.ID == "" || cfg.Name == "" {
			continue
		}
		ids = append(ids, cfg.ID)
		caseSQL.WriteString("WHEN ? THEN ? ")
		args = append(args, cfg.ID, cfg.Name)
	}
	if len(ids) == 0 {
		return 0
	}
	caseSQL.WriteString("ELSE activity_name END")
	inPlaceholders := strings.Repeat("?,", len(ids))
	inPlaceholders = inPlaceholders[:len(inPlaceholders)-1]
	now := time.Now()
	args = append(args, now)
	for _, id := range ids {
		args = append(args, id)
	}
	sql := fmt.Sprintf(
		"UPDATE activity_project SET activity_name = %s, updated_at = ? WHERE activity_id IN (%s) AND deleted_at IS NULL",
		caseSQL.String(), inPlaceholders,
	)
	result := db.Exec(sql, args...)
	if result.Error != nil {
		System().Infof("[活动名称同步] 批量更新失败: %v", result.Error)
		return 0
	}
	updated := int(result.RowsAffected)
	if updated > 0 {
		System().Infof("[活动名称同步] 已更新 %d 条上车记录", updated)
	}
	return updated
}

func GetActivityProjectsByActivityAndEnvKey(activityID, envKey string) ([]ActivityProject, error) {
	var projects []ActivityProject
	err := db.Where("activity_id = ? AND env_key = ? AND deleted_at IS NULL", activityID, envKey).
		Find(&projects).Error
	if err != nil {
		return nil, err
	}
	return projects, nil
}

func GetActivityProjectsByUserAndEnv(userNumber int, activityID, envKey string) ([]ActivityProject, error) {
	var projects []ActivityProject
	err := db.Where("user_number = ? AND activity_id = ? AND env_key = ? AND deleted_at IS NULL",
		userNumber, activityID, envKey).
		Order("id ASC").
		Find(&projects).Error
	if err != nil {
		return nil, err
	}
	return projects, nil
}

func SoftDeleteActivityProject(id int) error {
	now := time.Now()
	return db.Model(&ActivityProject{}).Where("id = ?", id).Updates(map[string]interface{}{
		"deleted_at":  now,
		"sync_status": "pending_delete",
		"updated_at":  now,
	}).Error
}

// RevertSoftDeleteActivityProject 青龙删除失败时回滚软删除
func RevertSoftDeleteActivityProject(id int, syncStatus string) error {
	if syncStatus == "" {
		syncStatus = "synced"
	}
	now := time.Now()
	return db.Unscoped().Model(&ActivityProject{}).Where("id = ?", id).Updates(map[string]interface{}{
		"deleted_at":  nil,
		"sync_status": syncStatus,
		"sync_error":  "",
		"updated_at":  now,
	}).Error
}

// DeleteProjectWithQinglongSync 软删除并同步青龙，成功后才可退积分
func DeleteProjectWithQinglongSync(projectID int) error {
	var project ActivityProject
	if err := db.Where("id = ? AND deleted_at IS NULL", projectID).First(&project).Error; err != nil {
		return fmt.Errorf("未找到对应项目记录")
	}
	prevSyncStatus := project.SyncStatus
	if err := SoftDeleteActivityProject(projectID); err != nil {
		return fmt.Errorf("删除失败：%v", err)
	}
	if err := SyncProjectNow(projectID); err != nil {
		_ = RevertSoftDeleteActivityProject(projectID, prevSyncStatus)
		return fmt.Errorf("青龙删除失败：%v", SanitizeError(err))
	}
	return nil
}

func HardDeleteActivityProject(id int) error {
	return db.Where("id = ?", id).Delete(&ActivityProject{}).Error
}

func GetPendingSyncProjects(limit int) ([]ActivityProject, error) {
	var projects []ActivityProject
	err := db.Where(
		"(sync_status IN ('pending', 'pending_update', 'pending_disable', 'pending_enable') AND deleted_at IS NULL) OR "+
			"(sync_status = 'pending_delete' AND deleted_at IS NOT NULL) OR "+
			"(sync_status = 'error' AND sync_retry_count < ? AND deleted_at IS NULL)",
		syncMaxRetries,
	).
		Order("updated_at ASC").
		Limit(limit).
		Find(&projects).Error
	if err != nil {
		return nil, err
	}

	var deletedProjects []ActivityProject
	if err := db.Unscoped().
		Where("sync_status = ? AND deleted_at IS NOT NULL", "pending_delete").
		Order("updated_at ASC").
		Limit(limit).
		Find(&deletedProjects).Error; err != nil {
		return projects, err
	}

	seen := make(map[int]bool, len(projects))
	for _, p := range projects {
		seen[p.ID] = true
	}
	for _, p := range deletedProjects {
		if !seen[p.ID] {
			projects = append(projects, p)
			seen[p.ID] = true
		}
	}
	if len(projects) > limit {
		projects = projects[:limit]
	}
	return projects, nil
}

// CountPendingSyncProjects 统计当前待同步队列总量（不含本批上限）
func CountPendingSyncProjects() int64 {
	var n int64
	_ = db.Model(&ActivityProject{}).Where(
		"(sync_status IN ('pending', 'pending_update', 'pending_disable', 'pending_enable') AND deleted_at IS NULL) OR "+
			"(sync_status = 'error' AND sync_retry_count < ? AND deleted_at IS NULL)",
		syncMaxRetries,
	).Count(&n).Error
	var delN int64
	_ = db.Unscoped().Model(&ActivityProject{}).
		Where("sync_status = ? AND deleted_at IS NOT NULL", "pending_delete").
		Count(&delN).Error
	return n + delN
}

func GetProjectsBySyncStatus(status string) ([]ActivityProject, error) {
	var projects []ActivityProject
	err := db.Where("sync_status = ?", status).Find(&projects).Error
	return projects, err
}

func MarkProjectSynced(id int, qinglongEnvID int) error {
	now := time.Now()
	return db.Model(&ActivityProject{}).Where("id = ?", id).Updates(map[string]interface{}{
		"sync_status":      "synced",
		"qinglong_env_id":  qinglongEnvID,
		"sync_error":       "",
		"sync_retry_count": 0,
		"sync_at":          now,
		"updated_at":       now,
	}).Error
}

func MarkProjectSyncError(id int, errMsg string) error {
	now := time.Now()
	var project ActivityProject
	if err := db.Unscoped().Where("id = ?", id).First(&project).Error; err != nil {
		return err
	}

	retry := project.SyncRetryCount + 1
	updates := map[string]interface{}{
		"sync_error":       errMsg,
		"sync_retry_count": retry,
		"sync_at":          now,
		"updated_at":       now,
	}
	if retry >= syncMaxRetries {
		updates["sync_status"] = "error"
	}
	return db.Model(&ActivityProject{}).Where("id = ?", id).Updates(updates).Error
}

// RequeueProjectSync 将 error 状态重新加入同步队列
func RequeueProjectSync(id int, syncStatus string) error {
	now := time.Now()
	return db.Model(&ActivityProject{}).Where("id = ?", id).Updates(map[string]interface{}{
		"sync_status":      syncStatus,
		"sync_error":       "",
		"sync_retry_count": 0,
		"updated_at":       now,
	}).Error
}

func MarkProjectDeleted(id int) error {
	return db.Unscoped().Where("id = ?", id).Delete(&ActivityProject{}).Error
}

func CountActivityProjectStats(activityID, envKey string) (total, valid, expiringSoon, expired, disabled int) {
	now := time.Now()

	var projects []ActivityProject
	if activityID != "" {
		db.Where("activity_id = ? AND env_key = ? AND deleted_at IS NULL", activityID, envKey).Find(&projects)
	} else {
		db.Where("env_key = ? AND deleted_at IS NULL", envKey).Find(&projects)
	}

	for _, p := range projects {
		total++
		if p.Status == 0 {
			if (p.IsMonthlyDeduct || p.IsDailyDeduct) && p.ExpireDate != "" {
				expireDate, err := time.Parse(DateLayout, p.ExpireDate)
				if err == nil {
					expireThreshold := time.Date(expireDate.Year(), expireDate.Month(), expireDate.Day(), 0, 0, 0, 0, time.Local).AddDate(0, 0, 1)
					remain := expireThreshold.Sub(now)
					if remain <= 0 {
						expired++
					} else if remain <= 7*24*time.Hour {
						expiringSoon++
						valid++
					} else {
						valid++
					}
				} else {
					valid++
				}
			} else {
				valid++
			}
		} else {
			if (p.IsMonthlyDeduct || p.IsDailyDeduct) && p.ExpireDate != "" {
				expireDate, err := time.Parse(DateLayout, p.ExpireDate)
				if err == nil {
					expireThreshold := time.Date(expireDate.Year(), expireDate.Month(), expireDate.Day(), 0, 0, 0, 0, time.Local).AddDate(0, 0, 1)
					if now.After(expireThreshold) || now.Equal(expireThreshold) {
						expired++
					} else {
						disabled++
					}
				} else {
					disabled++
				}
			} else {
				disabled++
			}
		}
	}
	return
}

func CountActivityProjectByStatus(activityID, envKey string) (total, valid, expired int) {
	now := time.Now()

	var projects []ActivityProject
	if activityID != "" {
		db.Where("activity_id = ? AND env_key = ? AND deleted_at IS NULL", activityID, envKey).Find(&projects)
	} else {
		db.Where("env_key = ? AND deleted_at IS NULL", envKey).Find(&projects)
	}

	for _, p := range projects {
		total++
		if p.Status == 0 {
			if (p.IsMonthlyDeduct || p.IsDailyDeduct) && p.ExpireDate != "" {
				expireDate, err := time.Parse(DateLayout, p.ExpireDate)
				if err == nil {
					expireThreshold := time.Date(expireDate.Year(), expireDate.Month(), expireDate.Day(), 0, 0, 0, 0, time.Local).AddDate(0, 0, 1)
					if now.After(expireThreshold) || now.Equal(expireThreshold) {
						expired++
					} else {
						valid++
					}
				} else {
					valid++
				}
			} else {
				valid++
			}
		} else {
			expired++
		}
	}
	return
}

func GetExpiringProjects(daysThreshold int) ([]ActivityProject, error) {
	var projects []ActivityProject
	err := db.Where("(is_monthly_deduct = ? OR is_daily_deduct = ?) AND status = 0 AND expire_date != '' AND deleted_at IS NULL", true, true).
		Find(&projects).Error
	if err != nil {
		return nil, err
	}

	now := time.Now()
	var result []ActivityProject
	for _, p := range projects {
		expireDate, err := time.Parse(DateLayout, p.ExpireDate)
		if err != nil {
			continue
		}
		expireThreshold := time.Date(expireDate.Year(), expireDate.Month(), expireDate.Day(), 0, 0, 0, 0, time.Local).AddDate(0, 0, 1)
		durationLeft := expireThreshold.Sub(now)
		if durationLeft > 0 && durationLeft <= time.Duration(daysThreshold)*24*time.Hour {
			result = append(result, p)
		}
	}
	return result, nil
}

func GetExpiredProjects() ([]ActivityProject, error) {
	var projects []ActivityProject
	err := db.Where("(is_monthly_deduct = ? OR is_daily_deduct = ?) AND status = 0 AND expire_date != '' AND deleted_at IS NULL", true, true).
		Find(&projects).Error
	if err != nil {
		return nil, err
	}

	now := time.Now()
	var result []ActivityProject
	for _, p := range projects {
		expireDate, err := time.Parse(DateLayout, p.ExpireDate)
		if err != nil {
			continue
		}
		expireThreshold := time.Date(expireDate.Year(), expireDate.Month(), expireDate.Day(), 0, 0, 0, 0, time.Local).AddDate(0, 0, 1)
		if now.After(expireThreshold) || now.Equal(expireThreshold) {
			result = append(result, p)
		}
	}
	return result, nil
}

func GetDisabledExpiredProjects() ([]ActivityProject, error) {
	var projects []ActivityProject
	err := db.Where("(is_monthly_deduct = ? OR is_daily_deduct = ?) AND status != 0 AND expire_date != '' AND deleted_at IS NULL", true, true).
		Find(&projects).Error
	if err != nil {
		return nil, err
	}

	now := time.Now()
	var result []ActivityProject
	for _, p := range projects {
		expireDate, err := time.Parse(DateLayout, p.ExpireDate)
		if err != nil {
			continue
		}
		expireThreshold := time.Date(expireDate.Year(), expireDate.Month(), expireDate.Day(), 0, 0, 0, 0, time.Local).AddDate(0, 0, 1)
		expiredDuration := now.Sub(expireThreshold)
		if expiredDuration >= 0 {
			result = append(result, p)
		}
	}
	return result, nil
}

func GetFirstRemarkParam(remarks string) string {
	if remarks == "" {
		return ""
	}
	parts := strings.Split(remarks, "/")
	if len(parts) == 0 {
		return ""
	}
	return strings.TrimSpace(parts[0])
}

func GetUserIDFromRemarks(remarks string) int {
	parts := strings.Split(remarks, "/")
	if len(parts) >= 2 {
		userID := 0
		fmt.Sscanf(strings.TrimSpace(parts[1]), "%d", &userID)
		return userID
	}
	return 0
}

// GetAllActivityProjects 获取所有活动项目（用于全量同步）
func GetAllActivityProjects() ([]ActivityProject, error) {
	var projects []ActivityProject
	err := db.Where("deleted_at IS NULL").Find(&projects).Error
	return projects, err
}

// GetProjectsNeedingFullSync 获取需要全量同步的项目
func GetProjectsNeedingFullSync(configName string) ([]ActivityProject, error) {
	var projects []ActivityProject
	err := db.Where("qinglong_config_name = ? AND deleted_at IS NULL AND sync_status != 'deleted'",
		configName).Find(&projects).Error
	return projects, err
}

var _ = (*gorm.DB)(nil)