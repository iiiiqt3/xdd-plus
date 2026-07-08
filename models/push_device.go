package models

import (
	"strings"
	"time"
)

// UserPushDevice App 极光设备登记（按用户可多设备）
type UserPushDevice struct {
	ID             int       `gorm:"primaryKey" json:"id"`
	UserNumber     int       `gorm:"index;not null" json:"userNumber"`
	Platform       string    `gorm:"size:16;index;not null;default:android" json:"platform"`
	RegistrationID string    `gorm:"size:64;index" json:"registrationId"`
	Alias          string    `gorm:"size:64;index" json:"alias"`
	AppVersion     string    `gorm:"size:32" json:"appVersion"`
	UpdatedAt      time.Time `json:"updatedAt"`
	CreatedAt      time.Time `json:"createdAt"`
}

func (UserPushDevice) TableName() string { return "user_push_device" }

// UpsertUserPushDevice 登录后登记极光 RegistrationID
func UpsertUserPushDevice(userNumber int, registrationID, alias, appVersion string) error {
	if userNumber <= 0 {
		return nil
	}
	registrationID = strings.TrimSpace(registrationID)
	alias = strings.TrimSpace(alias)
	if registrationID == "" && alias == "" {
		return nil
	}
	if alias == "" && userNumber > 0 {
		alias = PortalJPushAlias(userNumber)
	}
	var row UserPushDevice
	q := db.Where("user_number = ?", userNumber)
	if registrationID != "" {
		q = q.Where("registration_id = ?", registrationID)
	} else {
		q = q.Where("alias = ?", alias)
	}
	err := q.First(&row).Error
	now := time.Now()
	if err != nil {
		row = UserPushDevice{
			UserNumber:     userNumber,
			Platform:       ClientPlatformAndroid,
			RegistrationID: registrationID,
			Alias:          alias,
			AppVersion:     appVersion,
			CreatedAt:      now,
			UpdatedAt:      now,
		}
		return db.Create(&row).Error
	}
	updates := map[string]interface{}{
		"alias":       alias,
		"app_version": appVersion,
		"updated_at":  now,
	}
	if registrationID != "" {
		updates["registration_id"] = registrationID
	}
	return db.Model(&row).Updates(updates).Error
}

func DeleteUserPushDevices(userNumber int) error {
	if userNumber <= 0 {
		return nil
	}
	return db.Where("user_number = ?", userNumber).Delete(&UserPushDevice{}).Error
}

func DeleteUserPushDevice(userNumber int, registrationID string) error {
	if userNumber <= 0 {
		return nil
	}
	registrationID = strings.TrimSpace(registrationID)
	if registrationID == "" {
		return DeleteUserPushDevices(userNumber)
	}
	return db.Where("user_number = ? AND registration_id = ?", userNumber, registrationID).Delete(&UserPushDevice{}).Error
}
