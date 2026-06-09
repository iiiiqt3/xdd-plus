package models

import (
	"fmt"
	"strings"
	"time"
)

type PortalWxDevice struct {
	ID         int       `gorm:"primaryKey"`
	UserNumber int       `gorm:"index"`
	Wxid       string    `gorm:"size:128;uniqueIndex"`
	CreatedAt  time.Time
}

type PortalWxDeviceStatus struct {
	ID          int    `json:"id"`
	Wxid        string `json:"wxid"`
	Nickname    string `json:"nickname"`
	Device      string `json:"device"`
	Status      string `json:"status"`
	Online      bool   `json:"online"`
	LoginTime   string `json:"loginTime"`
	RefreshTime string `json:"refreshTime"`
	IsPrimary   bool   `json:"isPrimary"`
}

const maxWxDevicesPerUser = 5

func GetPortalWxDevices(userNumber int) ([]PortalWxDeviceStatus, error) {
	user, err := getPortalUserByNumber(userNumber)
	if err != nil {
		return nil, err
	}

	// 同时查询新旧两个地址
	raw, _ := getWxUserStatusRaw()

	var rawOld *portalWxUserStatusResponse
	if isNewProtocolEnabled() && getNewWxLoginBaseURL() != getOldWxLoginBaseURL() {
		rawOld, _ = getWxUserStatusRawFromURL(getOldWxLoginBaseURL())
	}

	var rawNew *portalWxUserStatusResponse
	if Config.WxProtocol.NewLoginBaseURL != "" && getWxLoginBaseURL() != getNewWxLoginBaseURL() {
		rawNew, _ = getWxUserStatusRawFromURL(getNewWxLoginBaseURL())
	}

	// 合并三个来源的数据（优先在线）
	mergedRaw := mergePortalWxRaw(raw, rawOld, rawNew)

	var result []PortalWxDeviceStatus

	primaryWxid := strings.TrimSpace(user.Wxid)
	if primaryWxid != "" {
		primaryStatus := buildWxDeviceStatus(0, primaryWxid, true, mergedRaw, user.Nickname)
		result = append(result, primaryStatus)
	}

	var devices []PortalWxDevice
	db.Where("user_number = ?", userNumber).Order("created_at asc").Find(&devices)
	for _, d := range devices {
		if strings.TrimSpace(d.Wxid) == primaryWxid {
			continue
		}
		status := buildWxDeviceStatus(d.ID, d.Wxid, false, mergedRaw, "")
		result = append(result, status)
	}

	return result, nil
}

// mergePortalWxRaw 合并多个来源的设备状态，在线优先
func mergePortalWxRaw(sources ...*portalWxUserStatusResponse) *portalWxUserStatusResponse {
	merged := &portalWxUserStatusResponse{
		Status: true,
		Data:   make(map[string]portalWxDeviceInfo),
	}
	for _, src := range sources {
		if src == nil || src.Data == nil {
			continue
		}
		for wxid, info := range src.Data {
			if existing, ok := merged.Data[wxid]; ok {
				// 如果新来源在线而已有数据离线，用新的
				if info.Survival == 1 && existing.Survival != 1 {
					merged.Data[wxid] = info
				} else if info.Survival == existing.Survival && info.RefreshDate > existing.RefreshDate {
					merged.Data[wxid] = info
				}
			} else {
				merged.Data[wxid] = info
			}
		}
	}
	return merged
}

func AddPortalWxDevice(userNumber int, wxid string) (*PortalWxDeviceStatus, error) {
	wxid = strings.TrimSpace(wxid)
	if wxid == "" {
		return nil, fmt.Errorf("微信ID不能为空")
	}

	user, err := getPortalUserByNumber(userNumber)
	if err != nil {
		return nil, err
	}

	if strings.TrimSpace(user.Wxid) == wxid {
		return nil, fmt.Errorf("该微信ID已是您的主绑定设备，无需重复添加")
	}

	// 同时检查新旧两个地址
	raw, err := getWxUserStatusRaw()
	rawOld, errOld := getWxUserStatusRawFromURL(getOldWxLoginBaseURL())
	if err != nil && errOld != nil {
		return nil, fmt.Errorf("无法连接微信协议服务，请稍后重试")
	}
	mergedRaw := mergePortalWxRaw(raw, rawOld)
	if _, ok := mergedRaw.Data[wxid]; !ok {
		return nil, fmt.Errorf("该微信ID未在微信协议系统中注册，请先扫码登录后再添加")
	}

	var count int64
	db.Model(&PortalWxDevice{}).Where("user_number = ?", userNumber).Count(&count)
	primaryCount := int64(0)
	if strings.TrimSpace(user.Wxid) != "" {
		primaryCount = 1
	}
	if count+primaryCount >= maxWxDevicesPerUser {
		return nil, fmt.Errorf("最多添加 %d 个微信ID（含主绑定）", maxWxDevicesPerUser)
	}

	var existing PortalWxDevice
	if db.Where("wxid = ?", wxid).First(&existing).Error == nil {
		if existing.UserNumber == userNumber {
			return nil, fmt.Errorf("该微信ID已添加")
		}
		return nil, fmt.Errorf("该微信ID已被其他用户添加")
	}

	device := PortalWxDevice{
		UserNumber: userNumber,
		Wxid:       wxid,
	}
	if err := db.Create(&device).Error; err != nil {
		return nil, fmt.Errorf("添加失败：%v", err)
	}

	status := buildWxDeviceStatus(device.ID, wxid, false, raw, "")
	return &status, nil
}

func RemovePortalWxDevice(userNumber int, deviceID int) error {
	result := db.Where("id = ? AND user_number = ?", deviceID, userNumber).Delete(&PortalWxDevice{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("设备不存在或无权操作")
	}
	return nil
}

func buildWxDeviceStatus(id int, wxid string, isPrimary bool, raw *portalWxUserStatusResponse, fallbackNickname string) PortalWxDeviceStatus {
	status := PortalWxDeviceStatus{
		ID:        id,
		Wxid:      wxid,
		IsPrimary: isPrimary,
		Status:    "🔴 离线",
		Device:    "-",
		LoginTime: "-",
	}

	if raw != nil {
		if info, ok := raw.Data[wxid]; ok {
			status.Nickname = info.Nickname
			status.Device = info.Device
			status.Status = wxStatusText(info.Survival)
			status.Online = info.Survival == 1
			status.LoginTime = formatPortalWxUnix(info.LoginDate)
			status.RefreshTime = formatPortalWxUnix(info.RefreshDate)
		}
	}

	if status.Nickname == "" {
		if fallbackNickname != "" {
			status.Nickname = fallbackNickname
		} else {
			status.Nickname = "未知昵称"
		}
	}

	return status
}