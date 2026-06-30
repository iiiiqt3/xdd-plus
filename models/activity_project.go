package models

import (
	"fmt"
	"log"
	"strings"
	"time"

	"gorm.io/gorm"
)

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
	CreatedAt          time.Time  `gorm:"column:created_at"`
	UpdatedAt          time.Time  `gorm:"column:updated_at"`
	DeletedAt          *time.Time `gorm:"column:deleted_at;index"`
	SyncStatus         string     `gorm:"column:sync_status;size:16;default:'pending'"`
	SyncError          string     `gorm:"column:sync_error;type:text"`
	SyncAt             *time.Time `gorm:"column:sync_at"`
}

func (ActivityProject) TableName() string {
	return "activity_project"
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

	paidDays := totalRemaining - grantedRemaining
	if paidDays < 0 {
		paidDays = 0
	}
	return int(paidDays)
}

func CreateActivityProject(project *ActivityProject) error {
	now := time.Now()
	project.CreatedAt = now
	project.UpdatedAt = now
	project.SyncStatus = "pending"

	if project.RemarkAlias == "" {
		project.RemarkAlias = GetFirstRemarkParam(project.Remarks)
	}
	return db.Create(project).Error
}

func UpdateActivityProject(project *ActivityProject) error {
	project.UpdatedAt = time.Now()
	return db.Save(project).Error
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

func CheckDuplicateRemarksDB(remarks, envKey string) (bool, error) {
	var count int64
	err := db.Model(&ActivityProject{}).
		Where("remarks = ? AND env_key = ? AND deleted_at IS NULL", remarks, envKey).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
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
	updated := 0
	now := time.Now()
	for _, cfg := range configs {
		if cfg == nil || cfg.ID == "" || cfg.Name == "" {
			continue
		}
		result := db.Model(&ActivityProject{}).
			Where("activity_id = ? AND activity_name <> ? AND deleted_at IS NULL", cfg.ID, cfg.Name).
			Updates(map[string]interface{}{
				"activity_name": cfg.Name,
				"updated_at":    now,
			})
		if result.Error != nil {
			System().Infof("[活动名称同步] activity_id=%s 失败: %v", cfg.ID, result.Error)
			continue
		}
		updated += int(result.RowsAffected)
	}
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

func HardDeleteActivityProject(id int) error {
	return db.Where("id = ?", id).Delete(&ActivityProject{}).Error
}

func GetPendingSyncProjects(limit int) ([]ActivityProject, error) {
	var projects []ActivityProject
	err := db.Where("sync_status IN ('pending', 'pending_update', 'pending_disable', 'pending_enable', 'pending_delete') AND deleted_at IS NULL").
		Order("updated_at ASC").
		Limit(limit).
		Find(&projects).Error
	return projects, err
}

func GetProjectsBySyncStatus(status string) ([]ActivityProject, error) {
	var projects []ActivityProject
	err := db.Where("sync_status = ?", status).Find(&projects).Error
	return projects, err
}

func MarkProjectSynced(id int, qinglongEnvID int) error {
	now := time.Now()
	return db.Model(&ActivityProject{}).Where("id = ?", id).Updates(map[string]interface{}{
		"sync_status":     "synced",
		"qinglong_env_id": qinglongEnvID,
		"sync_error":      "",
		"sync_at":         now,
		"updated_at":      now,
	}).Error
}

func MarkProjectSyncError(id int, errMsg string) error {
	now := time.Now()
	return db.Model(&ActivityProject{}).Where("id = ?", id).Updates(map[string]interface{}{
		"sync_status": "error",
		"sync_error":  errMsg,
		"sync_at":     now,
		"updated_at":  now,
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