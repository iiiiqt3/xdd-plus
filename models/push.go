package models

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"strconv" // 用于字符串转数字
	"strings" // 用于字符串判断 (修复 undefined: strings)
	"time"

	"github.com/beego/beego/v2/client/httplib"
)

// Push 推送消息到用户，根据用户配置推送到QQ、微信、TG等渠道
// 推送渠道说明：0所有，1QQ，2微信，3TG，12QQ+微信，13QQ+TG，23微信+TG
func (ck JdCookie) Push(msg string) {
	if ck.PtPin != "" {
		go SendQQ(ck.QQ, msg)
		go pushPlus(ck.PushPlus, msg)
		go SendTgMsg(ck.Telegram, msg)
		if ck.WeiXin != "" {
			go SendWxMsg(ck.WeiXin, msg)
		}
	} else {
		value := GetEnv("AdminPush")
		if value == "" {
			value = "0"
		}
		
		if value == "0" || value == "1" || value == "12" || value == "13" {
			go SendQQ(Config.QQID, msg) // QQ通知
		}
		
		if value == "0" || value == "2" || value == "12" || value == "23" {
			WeiXin := getWeiXinId(Config.QQID)
			if WeiXin != "找不到对应的微信ID" {
				go SendWxMsg(WeiXin, msg)  // 微信通知
			}
		}
		
		if value == "0" || value == "3" || value == "13" || value == "23" {
			go SendTgMsg(Config.TelegramUserID, msg) // TG通知
		}
		
		if value == "0" || value == "4" {
			go qywxNotify(&QywxConfig{QywxKey: Config.QywxKey, Content: msg}) // 企业微信通知
		}
	}
}


func PushByQQ(qq string, msg string) {
	if qq == "" {
		return
	}

	// 定义局部结构体匹配 users 表字段
	// 修改点：将 gorm:"column:qq" 改为 gorm:"column:number"
	type UserRecord struct {
		QQ     string `gorm:"column:number"` // 这里映射到数据库的 number 字段
		WxID   string `gorm:"column:wxid"`
	}

	var user UserRecord

	// 查询 users 表
	// 修改点：Where 条件中的字段名也要改为 "number"
	err := db.Table("users").Where("number = ?", qq).First(&user).Error
	if err != nil {
		// 没找到或出错，直接返回
		return
	}

	// --- 类型转换 ---
	// 将字符串类型的 number (原逻辑中的 QQ) 转换为 int
	qqInt, err := strconv.Atoi(user.QQ)
	if err != nil {
		return
	}

	// 推送 QQ (现在传入的是 int 类型)
	go SendQQ(qqInt, msg)

	// 推送微信 (仅当 wxid 存在且非空)
	if user.WxID != "" && user.WxID != "[NULL]" {
		// 简单的格式校验
		if strings.HasPrefix(user.WxID, "wxid_") || len(user.WxID) > 5 {
			go SendWxMsg(user.WxID, msg)
		}
	}
}


func pushPlus(token string, content string) {
	if token == "" {
		return
	}
	data, _ := json.Marshal(struct {
		Token    string `json:"token"`
		Content  string `json:"content"`
		Template string `json:"template"`
	}{
		Token:    token,
		Content:  content,
		Template: "txt",
	})
	req := httplib.Post("http://pushplus.hxtrip.com/send")
	req.Header("Content-Type", "application/json")
	req.Body(data)
	req.Response()
	req = httplib.Post("http://www.pushplus.plus/send")
	req.Header("Content-Type", "application/json")
	req.Body(data)
	req.Response()
}

type QywxConfig struct {
	QywxKey string
	Content string
}

type QywxNotifyMessage struct {
	Msgtype string `json:"msgtype"`
	Text    struct {
		Content string `json:"content"`
	} `json:"text"`
}

func qywxNotify(c *QywxConfig) {
	if c.QywxKey == "" {
		return
	}
	wx := QywxNotifyMessage{
		Msgtype: "text",
	}
	wx.Text.Content = c.Content
	req := httplib.Post("https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=" + c.QywxKey)
	req.Header("Content-Type", "application/json")
	req, _ = req.JSONBody(wx)
	req.SetTimeout(time.Second*2, time.Second*2)
	req.Response()
}

// JpushConfig 极光推送（仅 Android，本期不含 iOS）
type JpushConfig struct {
	Enabled      bool   `yaml:"enabled"`
	AppKey       string `yaml:"app_key"`
	MasterSecret string `yaml:"master_secret"`
	Production   bool   `yaml:"production"`
}

func jpushConfig() JpushConfig {
	return Config.Jpush
}

func JPushEnabled() bool {
	cfg := jpushConfig()
	return cfg.Enabled && strings.TrimSpace(cfg.AppKey) != "" && strings.TrimSpace(cfg.MasterSecret) != ""
}

// PortalJPushAlias 与安卓端一致的别名格式
func PortalJPushAlias(userNumber int) string {
	return fmt.Sprintf("portal_%d", userNumber)
}

func channelsIncludeApp(channels string) bool {
	channels = strings.TrimSpace(channels)
	if channels == "" {
		return false
	}
	for _, part := range strings.Split(channels, ",") {
		switch strings.ToLower(strings.TrimSpace(part)) {
		case "app", "webapp", "所有", "all":
			return true
		}
	}
	return false
}

const jpushAndroidChannelID = "admin_push_channel"

func listUserPushRegistrationIDs(userNumber int) []string {
	if userNumber <= 0 {
		return nil
	}
	var devices []UserPushDevice
	if err := db.Where("user_number = ? AND registration_id != ''", userNumber).Find(&devices).Error; err != nil {
		return nil
	}
	seen := make(map[string]struct{}, len(devices))
	ids := make([]string, 0, len(devices))
	for _, device := range devices {
		regID := strings.TrimSpace(device.RegistrationID)
		if regID == "" {
			continue
		}
		if _, ok := seen[regID]; ok {
			continue
		}
		seen[regID] = struct{}{}
		ids = append(ids, regID)
	}
	return ids
}

func summarizeNotificationBody(content string) string {
	text := strings.TrimSpace(content)
	text = strings.ReplaceAll(text, "\r\n", "\n")
	if len([]rune(text)) > 120 {
		rs := []rune(text)
		return string(rs[:120]) + "..."
	}
	if text == "" {
		return "您有一条新消息"
	}
	return text
}

// DispatchJPushNotification 门户通知写入后触发极光推送（异步）
func DispatchJPushNotification(n *WebNotification) {
	if n == nil || n.ID <= 0 || !JPushEnabled() {
		return
	}
	if !channelsIncludeApp(n.Channels) {
		return
	}
	title := strings.TrimSpace(n.Title)
	if title == "" {
		title = "狗东通知"
	}
	body := summarizeNotificationBody(n.Content)
	extras := map[string]interface{}{
		"notificationId": n.ID,
		"category":       n.Category,
		"source":         n.Source,
		"displayType":    n.DisplayType,
	}
	switch strings.TrimSpace(n.TargetScope) {
	case TargetScopeUser:
		if n.TargetUser <= 0 {
			return
		}
		go func() {
			if err := jpushSendToUser(n.TargetUser, title, body, extras); err != nil {
				Warn("[极光推送] 单用户推送失败 user=%d id=%d: %v", n.TargetUser, n.ID, err)
			}
		}()
	default:
		go func() {
			for _, userNumber := range ListPortalAccessUserNumbers() {
				num := userNumber
				go func() {
					if err := jpushSendToUser(num, title, body, extras); err != nil {
						Warn("[极光推送] 定向推送失败 user=%d id=%d: %v", num, n.ID, err)
					}
				}()
			}
		}()
	}
}

func jpushSendToAlias(aliases []string, title, body string, extras map[string]interface{}) error {
	if len(aliases) == 0 {
		return nil
	}
	audience := map[string]interface{}{"alias": aliases}
	return jpushSend(audience, title, body, extras)
}

func jpushSendToRegistrationIDs(registrationIDs []string, title, body string, extras map[string]interface{}) error {
	if len(registrationIDs) == 0 {
		return nil
	}
	audience := map[string]interface{}{"registration_id": registrationIDs}
	return jpushSend(audience, title, body, extras)
}

// jpushSendToUser 优先 registration_id（登录登记），无记录时回退 alias
func jpushSendToUser(userNumber int, title, body string, extras map[string]interface{}) error {
	regIDs := listUserPushRegistrationIDs(userNumber)
	if len(regIDs) > 0 {
		if err := jpushSendToRegistrationIDs(regIDs, title, body, extras); err == nil {
			return nil
		}
	}
	return jpushSendToAlias([]string{PortalJPushAlias(userNumber)}, title, body, extras)
}

func jpushSend(audience interface{}, title, body string, extras map[string]interface{}) error {
	cfg := jpushConfig()
	payload := map[string]interface{}{
		"platform": "android",
		"audience": audience,
		"notification": map[string]interface{}{
			"android": map[string]interface{}{
				"alert":      body,
				"title":      title,
				"extras":     extras,
				"channel_id": jpushAndroidChannelID,
			},
		},
	}
	if cfg.Production {
		payload["options"] = map[string]interface{}{"apns_production": true}
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	auth := base64.StdEncoding.EncodeToString([]byte(cfg.AppKey + ":" + cfg.MasterSecret))
	req := httplib.Post("https://api.jpush.cn/v3/push").
		Header("Authorization", "Basic "+auth).
		Header("Content-Type", "application/json").
		SetTimeout(15*time.Second, 15*time.Second)
	req.Body(raw)
	resp, err := req.Response()
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("status %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	Info("[极光推送] 已发送 title=%s audience=%v", title, audience)
	return nil
}

// App 瞬时推送深链（与安卓 PushNavigationHelper 一致）
const (
	PushActionOpenWx       = "open_wx"
	PushActionOpenJdYyb    = "open_jd_yyb"
	PushActionOpenProjects = "open_projects"
	PushActionOpenFeedback = "open_feedback"
	PushActionShowBody     = "show_body"
)

func resolveEphemeralPushAction(source, category, title string) string {
	title = strings.TrimSpace(title)
	source = strings.TrimSpace(source)
	switch title {
	case NotifyTitleWxOffline:
		return PushActionOpenWx
	case NotifyTitleYybOffline:
		return PushActionOpenJdYyb
	case "反馈处理结果":
		return PushActionOpenFeedback
	}
	switch source {
	case NotifySourceWx:
		return PushActionOpenWx
	case NotifySourceYyb, NotifySourceJd:
		return PushActionOpenJdYyb
	case NotifySourceFeedback:
		return PushActionOpenFeedback
	case NotifySourceAuth:
		return PushActionOpenProjects
	}
	switch strings.TrimSpace(category) {
	case NotifyCategoryWx:
		return PushActionOpenWx
	case NotifyCategoryJd:
		return PushActionOpenJdYyb
	case NotifyCategoryFeedback:
		return PushActionOpenFeedback
	case NotifyCategoryAuth:
		return PushActionOpenProjects
	}
	return PushActionShowBody
}

func truncateEphemeralFullBody(content string) string {
	content = strings.TrimSpace(content)
	rs := []rune(content)
	if len(rs) <= 500 {
		return content
	}
	return string(rs[:500]) + "..."
}

// DispatchEphemeralPush 系统瞬时推送：不写库，仅极光 App
func DispatchEphemeralPush(userNumber int, title, content, category, source, action, displayType string) {
	if userNumber <= 0 || !JPushEnabled() {
		return
	}
	title = strings.TrimSpace(title)
	content = strings.TrimSpace(content)
	if title == "" || content == "" {
		return
	}
	if strings.TrimSpace(action) == "" {
		action = resolveEphemeralPushAction(source, category, title)
	}
	if strings.TrimSpace(displayType) == "" {
		displayType = NotifyDisplayNormal
	}
	body := summarizeNotificationBody(content)
	extras := map[string]interface{}{
		"notificationId": 0,
		"ephemeral":      true,
		"action":         action,
		"category":       category,
		"source":         source,
		"displayType":    displayType,
		"fullBody":       truncateEphemeralFullBody(content),
	}
	go func() {
		if err := jpushSendToUser(userNumber, title, body, extras); err != nil {
			Warn("[极光推送] 瞬时推送失败 user=%d title=%s: %v", userNumber, title, err)
		}
	}()
}
