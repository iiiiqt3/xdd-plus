package models

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"time"

)

// offlineNotifiedWxIDs 记录已发送过掉线通知的wxid，避免重复通知
// 当设备恢复上线时会清除记录，再次掉线时会重新通知
var offlineNotifiedWxIDs sync.Map

// MarkWxOfflineNotified 标记 wxid 已推送掉线（供应用宝双绑场景交叉去重）
func MarkWxOfflineNotified(wxid string) {
	wxid = strings.TrimSpace(wxid)
	if wxid != "" {
		offlineNotifiedWxIDs.Store(wxid, true)
	}
}

// WxOfflineAlreadyNotified wxid 是否已推送过微信协议掉线
func WxOfflineAlreadyNotified(wxid string) bool {
	wxid = strings.TrimSpace(wxid)
	if wxid == "" {
		return false
	}
	_, loaded := offlineNotifiedWxIDs.Load(wxid)
	return loaded
}

// recentlyLoggedOutWxIDs 记录最近主动登出的 wxid 及登出时间戳
// 用于在 wechat08 API 延迟更新 survival 字段时，强制将设备显示为离线
// 记录会在 60 秒后自动过期
var recentlyLoggedOutWxIDs sync.Map

// ==================== 接口基础配置 ====================

const (
	WxLoginBaseURL = ""
	HTTPTimeout    = 15 * time.Second
)

// markRecentlyLoggedOut 标记 wxid 为最近主动登出
func markRecentlyLoggedOut(wxid string) {
	recentlyLoggedOutWxIDs.Store(wxid, time.Now().Unix())
}

// isRecentlyLoggedOut 检查 wxid 是否在最近主动登出过（60秒内）
func isRecentlyLoggedOut(wxid string) bool {
	val, ok := recentlyLoggedOutWxIDs.Load(wxid)
	if !ok {
		return false
	}
	ts, isInt := val.(int64)
	if !isInt {
		recentlyLoggedOutWxIDs.Delete(wxid)
		return false
	}
	if time.Now().Unix()-ts > 60 {
		recentlyLoggedOutWxIDs.Delete(wxid)
		return false
	}
	return true
}

// clearRecentlyLoggedOut 清除 wxid 的登出标记（设备重新上线时调用）
func clearRecentlyLoggedOut(wxid string) {
	recentlyLoggedOutWxIDs.Delete(wxid)
}

// getWxLoginBaseURL 返回当前活跃的协议地址
func getWxLoginBaseURL() string {
	if WxLoginBaseURL != "" {
		return WxLoginBaseURL
	}
	if Config.WxProtocol.ActiveProtocol == "new" && Config.WxProtocol.NewLoginBaseURL != "" {
		return Config.WxProtocol.NewLoginBaseURL
	}
	return Config.WxProtocol.LoginBaseURL
}

// getNewWxLoginBaseURL 返回新协议地址
func getNewWxLoginBaseURL() string {
	if Config.WxProtocol.NewLoginBaseURL != "" {
		return Config.WxProtocol.NewLoginBaseURL
	}
	return Config.WxProtocol.LoginBaseURL
}

// getOldWxLoginBaseURL 返回旧协议地址
func getOldWxLoginBaseURL() string {
	return Config.WxProtocol.LoginBaseURL
}

// isNewProtocolEnabled 是否启用了新协议地址
func isNewProtocolEnabled() bool {
	return Config.WxProtocol.NewLoginBaseURL != "" && Config.WxProtocol.ActiveProtocol == "new"
}

func getWxDeviceName() string {
	if Config.WxProtocol.DeviceName != "" {
		return Config.WxProtocol.DeviceName
	}
	return "Xiaomi-M2012K11AC"
}

func getWxScanLoginCost() int {
	if Config.WxProtocol.ScanLoginCost > 0 {
		return Config.WxProtocol.ScanLoginCost
	}
	return 2000
}

func getWxScanLoginCoin() int {
	return getWxScanLoginCost()
}

// wxLoginRequest 通用 POST 请求封装
func wxLoginRequest(path string, reqBody interface{}) ([]byte, error) {
	return wxLoginRequestToURL(getWxLoginBaseURL(), path, reqBody)
}

// wxLoginRequestToURL 向指定地址发送 POST 请求
func wxLoginRequestToURL(baseURL, path string, reqBody interface{}) ([]byte, error) {
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("构造请求失败：%s", err.Error())
	}

	url := baseURL + path
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败：%s", err.Error())
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: HTTPTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求接口失败：%s", err.Error())
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败：%s", err.Error())
	}

	Wx().Infof("wxLoginRequest [POST %s] 响应: %s", path, string(body))
	return body, nil
}

// ==================== 接口响应结构体 ====================

// WxLoginCodeResp 获取登录二维码响应
type WxLoginCodeResp struct {
	Status  bool `json:"status"`
	Success bool `json:"success"`
	Data    struct {
		QrBase64 string `json:"qrbase64"`
		Uuid     string `json:"uuid"`
	} `json:"data"`
	Message string `json:"message"`
}

// WxLoginStatusResp 检查扫码状态响应
// 两种返回格式：
// 等待扫码/确认: {"status":true,"code":0/1,"data":{"status":0/1,"nickname":"..."}}
// 登录成功:     {"status":true,"code":2,"user":{"wxid":"...","nickname":"..."}}
type WxLoginStatusResp struct {
	Status bool   `json:"status"`
	Code   int    `json:"code"` // 0=等待扫码 1=已扫码待确认 2=登录成功
	Msg    string `json:"msg"`
	Data   struct {
		Uuid        string `json:"uuid"`
		Status      int    `json:"status"`
		Nickname    string `json:"nickname"`
		Wxid        string `json:"wxid"`
		HeadImg     string `json:"headimgurl"`
		ExpiredTime int    `json:"expiredtime"`
	} `json:"data"`
	User struct {
		Wxid     string `json:"wxid"`
		Nickname string `json:"nickname"`
		Avatar   string `json:"avatar"`
	} `json:"user"`
	Message string `json:"message"`
}

// WxLoginAgainResp 重新登录响应
type WxLoginAgainResp struct {
	Status  bool `json:"status"`
	Success bool `json:"success"`
	Data    struct {
		QrBase64 string `json:"qrbase64"`
		Uuid     string `json:"uuid"`
	} `json:"data"`
	Message string `json:"message"`
}

// WxLogoutResp 登出响应
type WxLogoutResp struct {
	Status  bool   `json:"status"`
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// ==================== 核心功能函数 ====================

// WXID_USER_STATUS 查询所有微信设备状态（管理员专用）
func WXID_USER_STATUS(sender *Sender) {
	sender.Reply("⏳ 正在获取微信设备状态，请稍候...")

	url := getWxLoginBaseURL() + "/api/v1/wx/user/status"
	client := &http.Client{Timeout: HTTPTimeout}
	resp, err := client.Get(url)
	if err != nil {
		sender.Reply("❌ 请求设备列表失败：" + err.Error())
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		sender.Reply("❌ 读取设备列表失败：" + err.Error())
		return
	}

	Wx().Infof("wxLoginGetRequest [GET /api/v1/wx/user/status] 响应: %s", string(body))

	var result struct {
		Status bool `json:"status"`
		Data   map[string]struct {
			Wxid        string `json:"wxid"`
			Avatar      string `json:"avatar"`
			Nickname    string `json:"nickname"`
			Device      string `json:"device"`
			Survival    int    `json:"survival"` // 1=在线 0=掉线
			LoginDate   int64  `json:"loginDate"`
			RefreshDate int64  `json:"refreshDate"`
		} `json:"data"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		sender.Reply("❌ 解析设备列表失败：" + err.Error())
		return
	}

	if !result.Status {
		sender.Reply("❌ 获取设备列表失败：" + result.Message)
		return
	}

	if len(result.Data) == 0 {
		sender.Reply("📭 当前没有微信设备")
		return
	}

	// 构建设备列表消息
	var msg strings.Builder
	onlineCount := 0
	offlineCount := 0
	for _, info := range result.Data {
		if info.Survival == 1 {
			onlineCount++
		} else {
			offlineCount++
		}
	}
	msg.WriteString(fmt.Sprintf("📋 微信设备状态（共 %d 个，在线 %d / 离线 %d）：\n", len(result.Data), onlineCount, offlineCount))

	i := 1
	for wxid, info := range result.Data {
		loginTime := time.Unix(info.LoginDate, 0).Format("01-02 15:04")
		refreshTime := time.Unix(info.RefreshDate, 0).Format("01-02 15:04")

		// 在线状态标识
		survivalTag := "🟢 在线"
		if info.Survival != 1 {
			survivalTag = "🔴 掉线"
		}

		msg.WriteString(fmt.Sprintf("\n%d. %s (%s)\n   🆔 %s\n   📱 %s\n   💡 登录：%s | 刷新：%s\n",
			i, info.Nickname, survivalTag, wxid, info.Device, loginTime, refreshTime))
		i++
	}

	sender.Reply(msg.String())
}

// WXID_MY_STATUS 普通用户查询自己的微信设备在线状态
func WXID_MY_STATUS(sender *Sender) {
	wxid := sender.WxId
	if wxid == "" {
		sender.Reply("❌ 无法获取你的微信号，请确认通过微信端发送此命令")
		return
	}

	sender.Reply("⏳ 正在查询你的设备状态...")

	url := getWxLoginBaseURL() + "/api/v1/wx/user/status"
	client := &http.Client{Timeout: HTTPTimeout}
	resp, err := client.Get(url)
	if err != nil {
		sender.Reply("❌ 请求设备列表失败：" + err.Error())
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		sender.Reply("❌ 读取设备列表失败：" + err.Error())
		return
	}

	var result struct {
		Status bool `json:"status"`
		Data   map[string]struct {
			Wxid        string `json:"wxid"`
			Avatar      string `json:"avatar"`
			Nickname    string `json:"nickname"`
			Device      string `json:"device"`
			Survival    int    `json:"survival"`
			LoginDate   int64  `json:"loginDate"`
			RefreshDate int64  `json:"refreshDate"`
		} `json:"data"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		sender.Reply("❌ 解析设备列表失败：" + err.Error())
		return
	}

	if !result.Status {
		sender.Reply("❌ 获取设备列表失败：" + result.Message)
		return
	}

	info, ok := result.Data[wxid]
	if !ok {
		sender.Reply("📭 你尚未扫码登录，当前无设备信息")
		return
	}

	loginTime := time.Unix(info.LoginDate, 0).Format("2006-01-02 15:04")
	refreshTime := time.Unix(info.RefreshDate, 0).Format("2006-01-02 15:04")

	survivalTag := "🟢 在线"
	tip := ""
	if info.Survival != 1 {
		survivalTag = "🔴 掉线"
		tip = "\n💡 设备已掉线，可发送「微信重新登录」或「微信唤醒登录」重新激活"
	}

	sender.Reply(fmt.Sprintf("📱 你的微信设备状态：\n\n👤 昵称：%s\n🆔 %s\n📊 状态：%s\n📱 设备：%s\n⏰ 登录：%s\n🔄 刷新：%s\n%s",
		info.Nickname, wxid, survivalTag, info.Device, loginTime, refreshTime, tip))
}

// checkWxDeviceOnline 检查指定 wxid 是否在在线设备列表中
// 调用 /api/v1/wx/user/status 接口，返回 data 是 map[wxid]DeviceInfo 格式
func checkWxDeviceOnline(wxid string) (bool, error) {
	// 先检查活跃地址
	online, err := checkWxDeviceOnlineFromURL(getWxLoginBaseURL(), wxid)
	if err == nil && online {
		return true, nil
	}

	// 如果新协议已启用且活跃地址不是旧地址，再检查旧地址
	if isNewProtocolEnabled() && getNewWxLoginBaseURL() != getOldWxLoginBaseURL() {
		oldOnline, oldErr := checkWxDeviceOnlineFromURL(getOldWxLoginBaseURL(), wxid)
		if oldErr == nil && oldOnline {
			return true, nil
		}
	}

	// 如果活跃地址不是新地址且新地址已配置，检查新地址
	if Config.WxProtocol.NewLoginBaseURL != "" && getWxLoginBaseURL() != getNewWxLoginBaseURL() {
		newOnline, newErr := checkWxDeviceOnlineFromURL(getNewWxLoginBaseURL(), wxid)
		if newErr == nil && newOnline {
			return true, nil
		}
	}

	return online, err
}

// findWxDeviceBaseURL 查找 wxid 所在的协议地址
// 优先返回设备在线的地址，其次返回设备存在的地址
// 如果都找不到，返回活跃地址
func findWxDeviceBaseURL(wxid string) string {
	// 先检查活跃地址
	activeURL := getWxLoginBaseURL()
	online, _ := checkWxDeviceOnlineFromURL(activeURL, wxid)
	if online {
		return activeURL
	}

	// 检查旧地址
	if isNewProtocolEnabled() && getNewWxLoginBaseURL() != getOldWxLoginBaseURL() {
		oldOnline, _ := checkWxDeviceOnlineFromURL(getOldWxLoginBaseURL(), wxid)
		if oldOnline {
			return getOldWxLoginBaseURL()
		}
	}

	// 检查新地址
	if Config.WxProtocol.NewLoginBaseURL != "" && getWxLoginBaseURL() != getNewWxLoginBaseURL() {
		newOnline, _ := checkWxDeviceOnlineFromURL(getNewWxLoginBaseURL(), wxid)
		if newOnline {
			return getNewWxLoginBaseURL()
		}
	}

	// 都不在线，检查设备是否存在（不限于 survival=1）
	// 新协议启用时优先新地址，避免新旧都有记录时误路由到旧地址（出码在旧、轮询在新）
	if isNewProtocolEnabled() && Config.WxProtocol.NewLoginBaseURL != "" {
		if exists, _ := checkWxDeviceExistsOnURL(getNewWxLoginBaseURL(), wxid); exists {
			return getNewWxLoginBaseURL()
		}
		if getNewWxLoginBaseURL() != getOldWxLoginBaseURL() {
			if exists, _ := checkWxDeviceExistsOnURL(getOldWxLoginBaseURL(), wxid); exists {
				return getOldWxLoginBaseURL()
			}
		}
	} else {
		if exists, _ := checkWxDeviceExistsOnURL(getOldWxLoginBaseURL(), wxid); exists {
			return getOldWxLoginBaseURL()
		}
		if Config.WxProtocol.NewLoginBaseURL != "" {
			if exists, _ := checkWxDeviceExistsOnURL(getNewWxLoginBaseURL(), wxid); exists {
				return getNewWxLoginBaseURL()
			}
		}
	}

	// 兜底返回活跃地址
	return activeURL
}

// checkWxDeviceOnlineFromURL 从指定地址检查 wxid 是否在线
// 不仅检查设备是否存在，还检查 survival 字段是否为 1（在线）
func checkWxDeviceOnlineFromURL(baseURL, wxid string) (bool, error) {
	url := baseURL + "/api/v1/wx/user/status"
	client := &http.Client{Timeout: HTTPTimeout}
	resp, err := client.Get(url)
	if err != nil {
		return false, fmt.Errorf("请求设备列表失败：%s", err.Error())
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, fmt.Errorf("读取设备列表失败：%s", err.Error())
	}

	var result struct {
		Status bool                   `json:"status"`
		Data   map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return false, fmt.Errorf("解析设备列表失败：%s", err.Error())
	}

	if !result.Status {
		return false, fmt.Errorf("获取设备列表失败")
	}

	info, ok := result.Data[wxid]
	if !ok {
		return false, nil
	}
	// 检查 survival 字段，只有 survival=1 才算在线
	if infoMap, isMap := info.(map[string]interface{}); isMap {
		if survival, has := infoMap["survival"]; has {
			if survivalNum, isNum := survival.(float64); isNum {
				return survivalNum == 1, nil
			}
		}
	}
	// 无法解析 survival 时，保守地认为存在即在线（兼容旧逻辑）
	return true, nil
}

// checkWxDeviceExistsOnURL 检查指定 wxid 是否在指定地址的设备列表中（不要求在线）
func checkWxDeviceExistsOnURL(baseURL, wxid string) (bool, error) {
	url := baseURL + "/api/v1/wx/user/status"
	client := &http.Client{Timeout: HTTPTimeout}
	resp, err := client.Get(url)
	if err != nil {
		return false, fmt.Errorf("请求设备列表失败：%s", err.Error())
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, fmt.Errorf("读取设备列表失败：%s", err.Error())
	}

	var result struct {
		Status bool `json:"status"`
		Data   map[string]struct {
			Wxid     string `json:"wxid"`
			Survival int    `json:"survival"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return false, fmt.Errorf("解析设备列表失败：%s", err.Error())
	}

	if !result.Status {
		return false, fmt.Errorf("获取设备列表失败")
	}

	// 只要 wxid 存在于设备列表中即可（不要求在线）
	_, ok := result.Data[wxid]
	return ok, nil
}

// checkWxDeviceExists 检查 wxid 是否存在于任何协议地址的设备列表中（不要求在线）
func checkWxDeviceExists(wxid string) bool {
	// 先检查活跃地址
	exists, _ := checkWxDeviceExistsOnURL(getWxLoginBaseURL(), wxid)
	if exists {
		return true
	}

	// 检查旧地址
	if isNewProtocolEnabled() && getNewWxLoginBaseURL() != getOldWxLoginBaseURL() {
		exists, _ = checkWxDeviceExistsOnURL(getOldWxLoginBaseURL(), wxid)
		if exists {
			return true
		}
	}

	// 检查新地址
	if Config.WxProtocol.NewLoginBaseURL != "" && getWxLoginBaseURL() != getNewWxLoginBaseURL() {
		exists, _ = checkWxDeviceExistsOnURL(getNewWxLoginBaseURL(), wxid)
		if exists {
			return true
		}
	}

	return false
}

// WxProtocolMigration 微信协议迁移记录
type WxProtocolMigration struct {
	ID         uint      `gorm:"primaryKey"`
	UserNumber int       `gorm:"index"`
	Wxid       string    `gorm:"size:128;index"`
	MigratedAt time.Time
}

// IsWxUserMigrated 检查用户是否已完成协议迁移
func IsWxUserMigrated(userNumber int) bool {
	var count int64
	db.Model(&WxProtocolMigration{}).Where("user_number = ?", userNumber).Count(&count)
	return count > 0
}

// IsWxWxidMigrated 检查 wxid 是否已完成协议迁移
func IsWxWxidMigrated(wxid string) bool {
	var count int64
	db.Model(&WxProtocolMigration{}).Where("wxid = ?", wxid).Count(&count)
	return count > 0
}

// MarkWxMigrated 标记用户已完成协议迁移
func MarkWxMigrated(userNumber int, wxid string) {
	if userNumber == 0 || wxid == "" {
		return
	}
	var existing WxProtocolMigration
	if db.Where("user_number = ? AND wxid = ?", userNumber, wxid).First(&existing).Error != nil {
		db.Create(&WxProtocolMigration{
			UserNumber: userNumber,
			Wxid:       wxid,
			MigratedAt: time.Now(),
		})
		Wx().Infof("用户 %d (wxid: %s) 已标记为协议迁移完成", userNumber, wxid)
	}
}

// GetWxMigrationStats 获取迁移统计数据
func GetWxMigrationStats() (migrated int, notMigrated int) {
	var migratedCount int64
	db.Model(&WxProtocolMigration{}).Distinct("user_number").Count(&migratedCount)

	var totalWithWxid int64
	db.Model(&User{}).Where("wxid != '' AND wxid IS NOT NULL").Count(&totalWithWxid)

	return int(migratedCount), int(totalWithWxid - migratedCount)
}

// WXID_CODE 获取登录二维码 —— 新设备登录
// 流程：先检查积分 >= 2000 → 获取二维码 → 扫码成功后才扣积分
func WXID_CODE(sender *Sender) {
	cost := getWxScanLoginCoin()
	// 管理员也扣积分，统一检查
	coin := GetCoin(sender.UserID)
	if coin < cost {
		sender.Reply(fmt.Sprintf("❌ 积分不足，扫码登录需要 %d 个积分，当前积分 %d\n💡 请私聊机器人转账充值，1元=100积分", cost, coin))
		return
	}

	// 风险确认
	sender.Reply("⚠️ 此功能不会泄露聊天记录，仅仅作为取code用，但是可能会存在封号的概率，请谨慎考虑。\n\n无异议请回复 y  继续流程，回复 q 退出。")

	go func() {
		msgChan := make(chan string)
		ckList[sender.UserID] = msgChan
		defer delete(ckList, sender.UserID)

		select {
		case input := <-msgChan:
			upper := strings.ToUpper(input)
			if upper == "Q" {
				sender.Reply("✅ 已退出扫码登录流程")
				return
			}
			if upper != "Y" {
				sender.Reply("⚠️ 无效输入，已退出扫码登录流程")
				return
			}
		case <-time.After(60 * time.Second):
			sender.Reply("⏰ 确认超时，已退出扫码登录流程")
			return
		}

		sender.Reply("⏳ 正在获取登录二维码，请稍候...")

		reqBody := map[string]interface{}{
			"DeviceID":   "device_" + fmt.Sprintf("%d", time.Now().UnixNano()),
			"DeviceName": getWxDeviceName(),
			"DeviceType": "car",
			"Proxy": map[string]string{
				"ProxyIp":       "",
				"ProxyPassword": "",
				"ProxyUser":     "",
			},
		}

		body, err := wxLoginRequest("/api/v1/wx/login/code", reqBody)
		if err != nil {
			sender.Reply("❌ " + err.Error())
			return
		}

		var result WxLoginCodeResp
		if err := json.Unmarshal(body, &result); err != nil {
			sender.Reply("❌ 解析响应失败：" + err.Error())
			return
		}

		if !result.Status || !result.Success {
			sender.Reply("❌ 获取二维码失败：" + result.Message)
			return
		}

		// 发送二维码图片
		if err := sendBase64Image(sender, result.Data.QrBase64); err != nil {
			Error("发送二维码图片失败: %s", err.Error())
			return
		}

		cost := getWxScanLoginCoin()
		sender.Reply(fmt.Sprintf("✅ 二维码已发送，请使用微信扫码登录。\n💡 扫码后请稍等片刻，系统将自动检测登录状态...\n⚠️ 扫码登录成功后将扣除 %d 积分", cost))

		// 异步轮询扫码状态，成功后扣积分
		go pollLoginStatus(sender, result.Data.Uuid, true)
	}()
}

// WXID_RELOGIN 重新登录 —— 自动读取发指令用户的 wxid
func WXID_RELOGIN(sender *Sender) {
	wxid := sender.WxId
	if wxid == "" {
		sender.Reply("❌ 无法获取你的微信号，请确认通过微信端发送此命令")
		return
	}

	// 检查该用户是否已扫码登录（是否在在线设备中）
	online, _ := checkWxDeviceOnline(wxid)

	// 判断是否为需要迁移的旧用户：旧地址有设备
	isMigration := false
	if isNewProtocolEnabled() && !IsWxWxidMigrated(wxid) {
		oldExists, _ := checkWxDeviceExistsOnURL(getOldWxLoginBaseURL(), wxid)
		if oldExists {
			isMigration = true
			online = true
			sender.Reply("🔄 检测到你使用的是旧协议设备，正在迁移到新协议地址（本次不扣积分）...")
		}
	}

	if !online {
		sender.Reply("❌ 你尚未扫码登录，请先发送「微信扫码登录」完成登录")
		return
	}

	sender.Reply("⏳ 正在为你的账号 [" + wxid + "] 重新获取登录二维码...")

	// 根据是否迁移选择不同的登录方式
	var qrBase64 string
	var uuid string

	if isMigration {
		// 迁移用户：在新地址上走全新扫码登录（不扣积分）
		scanReqBody := map[string]interface{}{
			"DeviceID":   "device_" + fmt.Sprintf("%d", time.Now().UnixNano()),
			"DeviceName": getWxDeviceName(),
			"DeviceType": "car",
			"Proxy": map[string]string{
				"ProxyIp":       "",
				"ProxyPassword": "",
				"ProxyUser":     "",
			},
		}
		body, err := wxLoginRequestToURL(getNewWxLoginBaseURL(), "/api/v1/wx/login/code", scanReqBody)
		if err != nil {
			sender.Reply("❌ 迁移登录失败：" + err.Error())
			return
		}
		var scanResult WxLoginCodeResp
		if err = json.Unmarshal(body, &scanResult); err != nil || !scanResult.Status || !scanResult.Success {
			errMsg := "获取二维码失败"
			if err == nil {
				errMsg = scanResult.Message
			}
			sender.Reply("❌ 迁移登录失败：" + errMsg)
			return
		}
		qrBase64 = scanResult.Data.QrBase64
		uuid = scanResult.Data.Uuid
	} else {
		// 普通用户：走二次登录
		reqBody := map[string]interface{}{
			"wxid": wxid,
			"Proxy": map[string]string{
				"ProxyIp":       "",
				"ProxyPassword": "",
				"ProxyUser":     "",
			},
		}
		baseURL := findWxDeviceBaseURL(wxid)
		body, err := wxLoginRequestToURL(baseURL, "/api/v1/wx/login/again", reqBody)
		if err != nil {
			sender.Reply("❌ " + err.Error())
			return
		}
		var result WxLoginAgainResp
		if err = json.Unmarshal(body, &result); err != nil {
			sender.Reply("❌ 解析响应失败：" + err.Error())
			return
		}
		if !result.Status {
			sender.Reply("❌ 重新登录失败：" + result.Message)
			return
		}
		qrBase64 = result.Data.QrBase64
		uuid = result.Data.Uuid
	}

	// 发送二维码图片
	if err := sendBase64Image(sender, qrBase64); err != nil {
		Error("发送二维码图片失败: %s", err.Error())
		return
	}

	if isMigration {
		sender.Reply("✅ 迁移二维码已发送，请使用微信扫码确认迁移到新协议。\n💡 扫码成功后将自动完成迁移，不扣积分")
	} else {
		sender.Reply("✅ 重新登录二维码已发送，请使用微信扫码。\n💡 扫码后请稍等片刻，系统将自动检测登录状态...")
	}

	// 异步轮询扫码状态（重新登录不扣积分，迁移也不扣积分）
	if uuid != "" {
		go pollLoginStatusWithMigration(sender, uuid, false, isMigration)
	}
}

// WXID_WAKE_LOGIN 唤醒登录 —— 自动读取发指令用户的 wxid
func WXID_WAKE_LOGIN(sender *Sender) {
	wxid := sender.WxId
	if wxid == "" {
		sender.Reply("❌ 无法获取你的微信号，请确认通过微信端发送此命令")
		return
	}

	// 检查该用户是否已扫码登录（是否在在线设备中）
	online, _ := checkWxDeviceOnline(wxid)

	// 判断是否为需要迁移的旧用户：旧地址有设备
	isMigration := false
	if isNewProtocolEnabled() && !IsWxWxidMigrated(wxid) {
		oldExists, _ := checkWxDeviceExistsOnURL(getOldWxLoginBaseURL(), wxid)
		if oldExists {
			isMigration = true
			online = true
			sender.Reply("🔄 检测到你使用的是旧协议设备，正在迁移到新协议地址（本次不扣积分）...")
		}
	}

	if !online {
		sender.Reply("❌ 你尚未扫码登录，请先发送「微信扫码登录」完成登录")
		return
	}

	var qrBase64 string
	var uuid string

	if isMigration {
		// 旧用户迁移：在新地址上走全新扫码登录（不扣积分）
		sender.Reply("⏳ 正在为迁移获取新协议二维码...")
		scanReqBody := map[string]interface{}{
			"DeviceID":   "device_" + fmt.Sprintf("%d", time.Now().UnixNano()),
			"DeviceName": getWxDeviceName(),
			"DeviceType": "car",
			"Proxy": map[string]string{
				"ProxyIp":       "",
				"ProxyPassword": "",
				"ProxyUser":     "",
			},
		}
		body, err := wxLoginRequestToURL(getNewWxLoginBaseURL(), "/api/v1/wx/login/code", scanReqBody)
		if err != nil {
			sender.Reply("❌ 迁移登录失败：" + err.Error())
			return
		}
		var scanResult WxLoginCodeResp
		if err = json.Unmarshal(body, &scanResult); err != nil || !scanResult.Status || !scanResult.Success {
			errMsg := "获取二维码失败"
			if err == nil {
				errMsg = scanResult.Message
			}
			sender.Reply("❌ 迁移登录失败：" + errMsg)
			return
		}
		qrBase64 = scanResult.Data.QrBase64
		uuid = scanResult.Data.Uuid
	} else {
		// 普通用户：走唤醒+二次登录，路由到设备所在地址
		baseURL := findWxDeviceBaseURL(wxid)

		sender.Reply("⏳ 正在唤醒 [" + wxid + "] ...")

		awakeBody := map[string]string{"wxid": wxid}
		body, err := wxLoginRequestToURL(baseURL, "/api/v1/wx/login/awake", awakeBody)
		if err != nil {
			sender.Reply("❌ 唤醒失败：" + err.Error())
			return
		}
		var awakeResult struct {
			Status  bool   `json:"status"`
			Success bool   `json:"success"`
			Message string `json:"message"`
		}
		if err := json.Unmarshal(body, &awakeResult); err != nil {
			sender.Reply("❌ 解析唤醒响应失败：" + err.Error())
			return
		}
		if !awakeResult.Status {
			sender.Reply("❌ 唤醒设备失败：" + awakeResult.Message)
			return
		}

		sender.Reply("✅ [" + wxid + "] 设备已唤醒，正在获取登录二维码...")

		twiceBody := map[string]string{"wxid": wxid}
		body, err = wxLoginRequestToURL(baseURL, "/api/v1/wx/login/twice", twiceBody)
		if err != nil {
			sender.Reply("❌ 获取唤醒登录二维码失败：" + err.Error())
			return
		}
		var twiceResult struct {
			Status  bool   `json:"status"`
			Success bool   `json:"success"`
			Data    struct {
				QrBase64 string `json:"qrbase64"`
				Uuid     string `json:"uuid"`
			} `json:"data"`
			Message string `json:"message"`
		}
		if err := json.Unmarshal(body, &twiceResult); err != nil {
			sender.Reply("❌ 解析唤醒登录响应失败：" + err.Error())
			return
		}
		if !twiceResult.Status {
			sender.Reply("❌ 唤醒登录失败：" + twiceResult.Message)
			return
		}
		qrBase64 = twiceResult.Data.QrBase64
		uuid = twiceResult.Data.Uuid
	}

	if qrBase64 != "" {
		if err := sendBase64Image(sender, qrBase64); err != nil {
			Error("发送二维码图片失败: %s", err.Error())
			return
		}
		if isMigration {
			sender.Reply("✅ [" + wxid + "] 迁移二维码已发送，请扫码确认迁移到新协议。\n💡 扫码成功后将自动完成迁移，不扣积分")
		} else {
			sender.Reply("✅ [" + wxid + "] 唤醒登录二维码已发送，请使用微信扫码。\n💡 扫码后请稍等片刻，系统将自动检测登录状态...")
		}
		if uuid != "" {
			go pollLoginStatusWithMigration(sender, uuid, false, isMigration)
		}
	} else {
		sender.Reply("✅ [" + wxid + "] 唤醒登录请求已发送成功！\n💡 请检查设备端登录状态。")
	}
}

// WXID_LOGOUT 登出账号 —— 自动读取发指令用户的 wxid
func WXID_LOGOUT(sender *Sender) {
	wxid := sender.WxId
	if wxid == "" {
		sender.Reply("❌ 无法获取你的微信号，请确认通过微信端发送此命令")
		return
	}

	// 检查该用户是否已扫码登录（是否在在线设备中）
	online, err := checkWxDeviceOnline(wxid)
	if err != nil {
		sender.Reply("❌ " + err.Error())
		return
	}
	if !online {
		sender.Reply("❌ 你尚未扫码登录，无需登出")
		return
	}

	sender.Reply("⏳ 正在登出 [" + wxid + "] ...")

	reqBody := map[string]string{
		"wxid": wxid,
	}

	body, err := wxLoginRequest("/api/v1/wx/login/logout", reqBody)
	if err != nil {
		sender.Reply("❌ " + err.Error())
		return
	}

	var result WxLogoutResp
	if err := json.Unmarshal(body, &result); err != nil {
		sender.Reply("❌ 解析响应失败：" + err.Error())
		return
	}

	// logout 接口可能不返回 success 字段，只检查 status
	if result.Status {
		sender.Reply("✅ [" + wxid + "] 已成功登出！")
	} else {
		sender.Reply("❌ [" + wxid + "] 登出失败：" + result.Message)
	}
}

// WXID_DELETE 删除用户的微信设备数据（从数据库删除wxid）
// 支持 wx 私聊和群聊发送，需要用户二次确认
func WXID_DELETE(sender *Sender) {
	wxid := sender.WxId
	if wxid == "" {
		wxid = getWeiXinId(sender.UserID)
	}
	wxid = strings.TrimSpace(wxid)
	if wxid == "" || wxid == "找不到对应的微信ID" {
		sender.Reply("❌ 无法获取你的微信ID，请先确认已绑定微信")
		return
	}

	statusRaw, err := getWxUserStatusRaw()
	if err != nil {
		sender.Reply("❌ " + err.Error())
		return
	}

	matchedWxid := ""
	for key, info := range statusRaw.Data {
		if strings.TrimSpace(key) == wxid || strings.TrimSpace(info.Wxid) == wxid {
			matchedWxid = key
			if strings.TrimSpace(info.Wxid) != "" {
				matchedWxid = info.Wxid
			}
			break
		}
	}
	if matchedWxid == "" {
		matchedWxid = wxid
	}

	// 请求用户确认
	sender.Reply(fmt.Sprintf("⚠️ 确认要删除微信设备 [%s] 吗？\n此操作将从数据库中移除你的微信记录，删除后需重新扫码登录。\n\n回复 y 继续删除，回复 q 退出。", matchedWxid))

	go func() {
		msgChan := make(chan string)
		ckList[sender.UserID] = msgChan
		defer delete(ckList, sender.UserID)

		select {
		case input := <-msgChan:
			upper := strings.ToUpper(input)
			if upper == "Q" {
				sender.Reply("✅ 已退出删除操作")
				return
			}
			if upper != "Y" {
				sender.Reply("⚠️ 无效输入，已退出删除操作")
				return
			}
		case <-time.After(60 * time.Second):
			sender.Reply("⏰ 确认超时，已取消删除操作")
			return
		}

		sender.Reply("⏳ 正在删除 [" + matchedWxid + "] 的设备数据...")

		body, err := wxLoginRequest("/api/v1/wx/user/delete", map[string]interface{}{"wxids": []string{matchedWxid}})
		if err != nil {
			sender.Reply("❌ " + err.Error())
			return
		}

		var result WxLogoutResp
		if err := json.Unmarshal(body, &result); err != nil {
			sender.Reply("❌ 解析响应失败：" + err.Error())
			return
		}

		if result.Status {
			var u User
			if db.Where("number = ?", sender.UserID).First(&u).Error == nil {
				db.Model(&u).Update("wxid", "")
			}
			sender.Reply("✅ [" + matchedWxid + "] 的设备数据已成功删除！\n如需重新使用，请发送【微信扫码登录】")
		} else {
			sender.Reply("❌ [" + matchedWxid + "] 删除失败：" + result.Message)
		}
	}()
}

// ==================== 辅助函数 ====================

// sendBase64Image 将 base64 编码的图片发送给用户
// 接口返回的格式为 "data:image/jpg;base64,XXXXX"，需要先去掉前缀再解码
func sendBase64Image(sender *Sender, base64Data string) error {
	// 去掉 "data:image/xxx;base64," 前缀
	if idx := strings.Index(base64Data, ","); idx != -1 {
		base64Data = base64Data[idx+1:]
	}

	base64Data = strings.TrimSpace(base64Data)
	if base64Data == "" {
		sender.Reply("⚠️ 二维码数据为空，请重试")
		return fmt.Errorf("base64 数据为空")
	}

	// 解码 base64 为图片字节
	imgBytes, err := base64.StdEncoding.DecodeString(base64Data)
	if err != nil {
		// 接口返回的 base64 可能没有标准填充，尝试补充
		imgBytes, err = base64.RawStdEncoding.DecodeString(base64Data)
		if err != nil {
			Error("base64解码失败: %s (数据长度: %d)", err.Error(), len(base64Data))
			sender.Reply("⚠️ 二维码图片解码失败，请重试")
			return err
		}
	}

	if len(imgBytes) == 0 {
		sender.Reply("⚠️ 二维码解码后数据为空，请重试")
		return fmt.Errorf("解码后数据为空")
	}

	// 调用 sender.SendImg 发送图片字节
	sender.SendImg(imgBytes)
	return nil
}

func autoBindWxDevice(userNumber int, wxid string) {
	wxid = strings.TrimSpace(wxid)
	if wxid == "" {
		return
	}
	user, err := getPortalUserByNumber(userNumber)
	if err != nil {
		Warn("自动绑定微信设备失败：获取用户信息失败 %v", err)
		return
	}
	if strings.TrimSpace(user.Wxid) == "" {
		if err := db.Model(&user).Update("wxid", wxid).Error; err != nil {
			Warn("自动绑定微信设备失败：设置主wxid失败 %v", err)
		} else {
			Wx().Infof("自动绑定微信设备：已将 %s 设为用户 %d 的主设备", wxid, userNumber)
		}
		return
	}
	if user.Wxid == wxid {
		return
	}
	var existingDevice PortalWxDevice
	if db.Where("user_number = ? AND wxid = ?", userNumber, wxid).First(&existingDevice).Error == nil {
		return
	}
	var deviceCount int64
	db.Model(&PortalWxDevice{}).Where("user_number = ?", userNumber).Count(&deviceCount)
	primaryCount := int64(0)
	if strings.TrimSpace(user.Wxid) != "" {
		primaryCount = 1
	}
	if deviceCount+primaryCount >= maxWxDevicesPerUser {
		Warn("自动绑定微信设备失败：用户 %d 已达到设备上限 %d", userNumber, maxWxDevicesPerUser)
		return
	}
	if err := db.Create(&PortalWxDevice{UserNumber: userNumber, Wxid: wxid}).Error; err != nil {
		Warn("自动绑定微信设备失败：创建记录失败 %v", err)
	} else {
		Wx().Infof("自动绑定微信设备：已将 %s 添加为用户 %d 的监控设备", wxid, userNumber)
	}
}

// pollLoginStatus 异步轮询扫码状态
// deductCoin: 是否在登录成功后扣除积分
func pollLoginStatus(sender *Sender, uuid string, deductCoin bool) {
	const (
		maxRetries = 60              // 最多轮询 60 次
		interval   = 3 * time.Second // 每 3 秒检查一次
		timeout    = 3 * time.Minute // 总超时 3 分钟
	)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	timeoutCh := time.After(timeout)

	for i := 0; i < maxRetries; i++ {
		select {
		case <-ticker.C:
			status, err := checkLoginStatus(uuid)
			if err != nil {
				Warn("检查扫码状态失败: %s", err.Error())
				continue
			}

			switch status.Code {
			case 0:
				// 等待扫码，不做提示（避免刷屏）
				continue
			case 1:
				sender.Reply("📱 已扫描二维码，请在手机上点击「确认登录」...")
			case 2:
				nickname := status.User.Nickname
				wxid := status.User.Wxid
				if nickname == "" {
					nickname = status.Data.Nickname
				}
				if wxid == "" {
					wxid = status.Data.Wxid
				}
				if nickname == "" {
					nickname = "微信用户"
				}

				autoBindWxDevice(sender.UserID, wxid)

				if deductCoin {
					cost := getWxScanLoginCoin()
					RemCoin(sender.UserID, cost)
					RecordCoinForSender(sender, sender.UserID, -cost, "微信登录", "微信扫码登录扣费")
					sender.Reply(fmt.Sprintf("🎉 登录成功！已扣除 %d 积分，剩余积分 %d\n\n👤 昵称：%s\n🆔 微信ID：%s\n✅ 已自动绑定到你的账号，可在APP/网页查看", cost, GetCoin(sender.UserID), nickname, wxid))
				} else {
					sender.Reply(fmt.Sprintf("🎉 登录成功！\n\n👤 昵称：%s\n🆔 微信ID：%s\n✅ 已自动绑定到你的账号", nickname, wxid))
				}
				return
			default:
				// 其他状态（可能是错误状态）
				if status.Msg != "" && status.Code > 2 {
					sender.Reply(fmt.Sprintf("⚠️ 登录异常：%s\n💡 请重新发送「微信扫码登录」获取新二维码", status.Msg))
					return
				}
			}
		case <-timeoutCh:
			sender.Reply("⏰ 扫码超时（3分钟），二维码已失效。\n💡 请重新发送「微信扫码登录」获取新二维码")
			return
		}
	}

	sender.Reply("⏰ 扫码检测超时，二维码已失效。\n💡 请重新发送「微信扫码登录」获取新二维码")
}

// pollLoginStatusWithMigration 支持迁移的轮询扫码状态
// deductCoin: 是否扣除积分, isMigration: 是否为迁移登录
func pollLoginStatusWithMigration(sender *Sender, uuid string, deductCoin bool, isMigration bool) {
	const (
		maxRetries = 60
		interval   = 3 * time.Second
		timeout    = 3 * time.Minute
	)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	timeoutCh := time.After(timeout)

	for i := 0; i < maxRetries; i++ {
		select {
		case <-ticker.C:
			status, err := checkLoginStatus(uuid)
			if err != nil {
				Warn("检查扫码状态失败: %s", err.Error())
				continue
			}

			switch status.Code {
			case 0:
				continue
			case 1:
				sender.Reply("📱 已扫描二维码，请在手机上点击「确认登录」...")
			case 2:
				nickname := status.User.Nickname
				wxid := status.User.Wxid
				if nickname == "" {
					nickname = status.Data.Nickname
				}
				if wxid == "" {
					wxid = status.Data.Wxid
				}
				if nickname == "" {
					nickname = "微信用户"
				}

				autoBindWxDevice(sender.UserID, wxid)

				// 迁移登录成功，标记为已迁移
				if isMigration {
					MarkWxMigrated(sender.UserID, wxid)
				}

				if deductCoin {
					cost := getWxScanLoginCoin()
					RemCoin(sender.UserID, cost)
					RecordCoinForSender(sender, sender.UserID, -cost, "微信登录", "微信扫码登录扣费")
					sender.Reply(fmt.Sprintf("🎉 登录成功！已扣除 %d 积分，剩余积分 %d\n\n👤 昵称：%s\n🆔 微信ID：%s\n✅ 已自动绑定到你的账号，可在APP/网页查看", cost, GetCoin(sender.UserID), nickname, wxid))
				} else if isMigration {
					sender.Reply(fmt.Sprintf("🎉 迁移成功！已切换到新协议地址（未扣积分）\n\n👤 昵称：%s\n🆔 微信ID：%s\n✅ 后续操作将自动使用新协议", nickname, wxid))
				} else {
					sender.Reply(fmt.Sprintf("🎉 登录成功！\n\n👤 昵称：%s\n🆔 微信ID：%s\n✅ 已自动绑定到你的账号", nickname, wxid))
				}
				return
			default:
				if status.Msg != "" && status.Code > 2 {
					sender.Reply(fmt.Sprintf("⚠️ 登录异常：%s\n💡 请重新发送「微信扫码登录」获取新二维码", status.Msg))
					return
				}
			}
		case <-timeoutCh:
			sender.Reply("⏰ 扫码超时（3分钟），二维码已失效。\n💡 请重新发送「微信扫码登录」获取新二维码")
			return
		}
	}

	sender.Reply("⏰ 扫码检测超时，二维码已失效。\n💡 请重新发送「微信扫码登录」获取新二维码")
}

// CheckWxOfflineAndNotify 检查所有微信设备在线状态，给掉线用户推送通知
func CheckWxOfflineAndNotify() {
	if n := CleanupStaleOfflineNotifications(); n > 0 {
		Wx().Infof("已自动清理 %d 条超过 %d 天的微信/应用宝掉线提醒通知", n, OfflineNotifyRetentionDays)
	}
	CheckWxOfflineAndNotifyWithChannels(DefaultProtocolOfflineNotifyChannels(), nil, false)
}

func CheckWxOfflineAndNotifyWithChannels(channels NotifyChannels, wxIDs []string, force bool) {
	channels = ProtocolOfflineNotifyChannels(channels)
	Wx().Infof("开始执行微信掉线检测推送...")

	// 同时查询新旧两个地址，合并结果（在线优先）
	data := fetchWxDevicesFromURL(getWxLoginBaseURL())

	if isNewProtocolEnabled() && getNewWxLoginBaseURL() != getOldWxLoginBaseURL() {
		oldData := fetchWxDevicesFromURL(getOldWxLoginBaseURL())
		data = mergeWxDevices(data, oldData)
	}
	if Config.WxProtocol.NewLoginBaseURL != "" && getWxLoginBaseURL() != getNewWxLoginBaseURL() {
		newData := fetchWxDevicesFromURL(getNewWxLoginBaseURL())
		data = mergeWxDevices(data, newData)
	}

	if len(data) == 0 {
		Wx().Infof("微信掉线检测：当前没有微信设备，跳过推送")
		return
	}

	offlineCount := 0
	notifiedCount := 0

	for wxid, info := range data {
		if len(wxIDs) > 0 {
			found := false
			for _, wid := range wxIDs {
				if wid == wxid {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		if info.Survival == 1 {
			// 设备在线，清除之前的掉线通知记录，下次掉线时可以重新通知
			if _, loaded := offlineNotifiedWxIDs.LoadAndDelete(wxid); loaded {
				Wx().Infof("微信掉线检测：用户 %s (%s) 已恢复上线，清除通知记录", info.Nickname, wxid)
			}
			continue
		}

		if WxOfflineNotifySkippedByDualBind(wxid) {
			Wx().Infof("微信掉线检测：%s (%s) 已双绑应用宝，跳过微信掉线推送", info.Nickname, wxid)
			continue
		}

		offlineCount++

		// 检查是否已经发送过掉线通知，如果已通知则跳过（管理员手动触发时忽略去重）
		if !force {
			if _, loaded := offlineNotifiedWxIDs.LoadOrStore(wxid, true); loaded {
				Wx().Infof("微信掉线检测：用户 %s (%s) 已通知过掉线，跳过", info.Nickname, wxid)
				continue
			}
		} else {
			offlineNotifiedWxIDs.Store(wxid, true)
		}

		Wx().Infof("微信掉线检测：用户 %s (%s) 已掉线，%s推送通知", info.Nickname, wxid, map[bool]string{true: "管理员手动", false: "首次"}[!force])

		loginTime := time.Unix(info.LoginDate, 0).Format("01-02 15:04")
		refreshTime := time.Unix(info.RefreshDate, 0).Format("01-02 15:04")

		notifyMsg := fmt.Sprintf(
			"⚠️ 微信协议已掉线，将影响协议项目获取 CK。\n\n"+
				"📋 设备信息\n"+
				"👤 %s（掉线）\n"+
				"🆔 %s\n"+
				"📱 %s\n"+
				"💡 登录：%s | 刷新：%s\n\n"+
				"💡 请发送【微信唤醒登陆】或【微信重新登陆】重新上线。\n"+
				"%s",
			info.Nickname, wxid, info.Device, loginTime, refreshTime, ProtocolOfflineNotifyFooter(),
		)
		var user User
		if db.Where("wxid = ?", wxid).First(&user).Error == nil {
			PushProtocolOfflineNotification(NotifyTitleWxOffline, notifyMsg, NotifyCategoryWx, NotifySourceWx, user.Number, channels)
		} else {
			PushProtocolOfflineToWxid(wxid, notifyMsg, channels)
		}
		notifiedCount++

		if offlineCount >= 2 {
			delay := 3 + rand.Intn(3)
			time.Sleep(time.Duration(delay) * time.Second)
		}
	}

	Wx().Infof("微信掉线检测推送完成，共 %d 个设备，%d 个掉线，%d 个已通知", len(data), offlineCount, notifiedCount)
}

// checkLoginStatus 检查扫码状态
func checkLoginStatus(uuid string) (*WxLoginStatusResp, error) {
	reqBody := map[string]string{
		"uuid": uuid,
	}

	body, err := wxLoginRequest("/api/v1/wx/login/status", reqBody)
	if err != nil {
		return nil, err
	}

	var result WxLoginStatusResp
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("解析状态响应失败：%s", err.Error())
	}

	return &result, nil
}
