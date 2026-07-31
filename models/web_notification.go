package models

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
)

const (
	NotifyCategoryActivity = "活动相关"
	NotifyCategoryAuth     = "项目授权相关"
	NotifyCategoryFeedback = "建议反馈类"
	NotifyCategoryWx       = "微信协议类"
	NotifyCategoryJd       = "京东类"
	NotifyCategoryOther    = "其他"
	NotifySourceAdmin      = "管理员推送"
	NotifySourceRobot      = "机器人消息"
	NotifySourceJd         = "京东相关"
	NotifySourceAuth       = "项目授权相关"
	NotifySourceFeedback   = "意见反馈"
	NotifySourceWx         = "微信协议"
	NotifySourceYyb        = "应用宝协议"
	NotifyDisplayNormal    = "normal"
	NotifyDisplayPopup     = "popup"

	NotifyTitleWxOffline       = "微信协议掉线提醒"
	NotifyTitleYybOffline      = "应用宝协议掉线提醒"
	OfflineNotifyRetentionDays = 3

	TargetScopeAll   = "all"
	TargetScopeUser  = "user"
	TargetScopeAdmin = "admin"
)

var AllNotifyCategories = []string{
	NotifyCategoryActivity,
	NotifyCategoryAuth,
	NotifyCategoryFeedback,
	NotifyCategoryWx,
	NotifyCategoryJd,
	NotifyCategoryOther,
}

type WebNotification struct {
	ID          int       `gorm:"primaryKey" json:"id"`
	Title       string    `gorm:"size:120;index" json:"title"`
	Content     string    `gorm:"type:text" json:"content"`
	Category    string    `gorm:"size:24;index" json:"category"`
	Source      string    `gorm:"size:32;index" json:"source"`
	Channels    string    `gorm:"size:64" json:"channels"`
	TargetScope string    `gorm:"size:32;index" json:"targetScope"`
	TargetUser  int       `gorm:"index" json:"targetUser"`
	ClickCount  int       `json:"clickCount"`
	ReadCount   int       `json:"readCount"`
	DisplayType string    `gorm:"size:16;index;default:normal" json:"displayType"`
	IsTop       bool      `gorm:"index;default:false" json:"isTop"`
	VisibleToAll bool     `gorm:"index;default:true" json:"visibleToAll"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type WebNotificationRead struct {
	ID             int       `gorm:"primaryKey" json:"id"`
	NotificationID int       `gorm:"uniqueIndex:idx_notify_user;index" json:"notificationId"`
	UserNumber     int       `gorm:"uniqueIndex:idx_notify_user;index" json:"userNumber"`
	ClickCount     int       `json:"clickCount"`
	ReadAt         time.Time `json:"readAt"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

type AdminNotificationItem struct {
	ID          int       `json:"id"`
	Title       string    `json:"title"`
	Content     string    `json:"content"`
	Category    string    `json:"category"`
	Source      string    `json:"source"`
	Channels    string    `json:"channels"`
	TargetScope string    `json:"targetScope"`
	TargetUser  int       `json:"targetUser"`
	ClickCount  int       `json:"clickCount"`
	ReadCount   int       `json:"readCount"`
	DisplayType string    `json:"displayType"`
	IsTop       bool      `json:"isTop"`
	VisibleToAll bool     `json:"visibleToAll"`
	CreatedAt   time.Time `json:"createdAt"`
}

type PortalNotificationItem struct {
	ID          int       `json:"id"`
	Title       string    `json:"title"`
	Content     string    `json:"content,omitempty"`
	Category    string    `json:"category"`
	Source      string    `json:"source"`
	Channels    string    `json:"channels"`
	IsRead      bool      `json:"isRead"`
	ClickCount  int       `json:"clickCount"`
	DisplayType string    `json:"displayType"`
	IsTop       bool      `json:"isTop"`
	CreatedAt   time.Time `json:"createdAt"`
	ReadAt      time.Time `json:"readAt"`
}

type NotifyUnreadStats struct {
	Total      int64            `json:"total"`
	Unread     int64            `json:"unread"`
	BySource   map[string]int64 `json:"bySource"`
	ByCategory map[string]int64 `json:"byCategory"`
}

type NotifyChannels struct {
	Web   bool `json:"web"`
	App   bool `json:"app"`
	Robot bool `json:"robot"`
}

func NormalizeNotifyChannels(channels []string) NotifyChannels {
	result := NotifyChannels{Web: true, App: true, Robot: true}
	if len(channels) == 0 {
		return result
	}
	result = NotifyChannels{}
	for _, ch := range channels {
		switch strings.ToLower(strings.TrimSpace(ch)) {
		case "all", "所有":
			return NotifyChannels{Web: true, App: true, Robot: true}
		case "webapp", "app/网页", "网页/app":
			result.Web = true
			result.App = true
		case "web", "网页":
			result.Web = true
		case "app":
			result.App = true
		case "robot", "bot", "机器人":
			result.Robot = true
		}
	}
	if !result.Web && !result.App && !result.Robot {
		return NotifyChannels{Web: true, App: true, Robot: true}
	}
	return result
}

// DefaultProtocolOfflineNotifyChannels 协议掉线默认定时推送：App + 机器人（不含网页库）
func DefaultProtocolOfflineNotifyChannels() NotifyChannels {
	return NotifyChannels{App: true, Robot: true, Web: false}
}

// ProtocolOfflineNotifyChannels 协议掉线推送渠道（强制关闭网页）
func ProtocolOfflineNotifyChannels(ch NotifyChannels) NotifyChannels {
	return NotifyChannels{App: ch.App, Robot: ch.Robot, Web: false}
}

// ProtocolOfflineNotifyChannelsFromStrings 解析后台勾选（webapp 仅保留 App，不写网页库）
func ProtocolOfflineNotifyChannelsFromStrings(channels []string) NotifyChannels {
	if len(channels) == 0 {
		return DefaultProtocolOfflineNotifyChannels()
	}
	n := NormalizeNotifyChannels(channels)
	out := NotifyChannels{App: n.App, Robot: n.Robot, Web: false}
	if !out.App && !out.Robot {
		return DefaultProtocolOfflineNotifyChannels()
	}
	return out
}

// PushProtocolOfflineNotification 协议掉线：App 极光 + 机器人(QQ/微信私聊)，不写网页通知库
func PushProtocolOfflineNotification(title, content, category, source string, userNumber int, channels NotifyChannels) {
	if userNumber <= 0 {
		return
	}
	ch := ProtocolOfflineNotifyChannels(channels)
	if ch.Robot {
		go PushByQQ(strconv.Itoa(userNumber), content)
	}
	if ch.App {
		_ = PushUserOfflineNotification(title, content, category, source, userNumber, ch)
	}
}

// PushProtocolOfflineToWxid 找不到系统用户时，机器人渠道仅发微信私聊兜底
func PushProtocolOfflineToWxid(wxid, content string, channels NotifyChannels) {
	ch := ProtocolOfflineNotifyChannels(channels)
	if ch.Robot {
		wxid = strings.TrimSpace(wxid)
		if wxid != "" {
			go SendWxMsg(wxid, content)
		}
	}
}

const protocolOfflineNotifyFooter = "📌 本消息仅发送一次，账号恢复后若再次掉线将重新提醒。\n📌 狗东 App 将同步推送，请及时处理。"

// ProtocolOfflineNotifyFooter 协议掉线推送统一页脚
func ProtocolOfflineNotifyFooter() string {
	return protocolOfflineNotifyFooter
}

func channelsToString(ch NotifyChannels) string {
	parts := make([]string, 0, 3)
	if ch.Web {
		parts = append(parts, "web")
	}
	if ch.App {
		parts = append(parts, "app")
	}
	if ch.Robot {
		parts = append(parts, "robot")
	}
	return strings.Join(parts, ",")
}

func normalizeNoticeCategory(category string) string {
	category = strings.TrimSpace(category)
	switch category {
	case NotifyCategoryActivity, NotifyCategoryAuth, NotifyCategoryFeedback, NotifyCategoryWx, NotifyCategoryJd, NotifyCategoryOther:
		return category
	default:
		return NotifyCategoryOther
	}
}

func normalizeNotifyDisplayType(displayType string) string {
	displayType = strings.ToLower(strings.TrimSpace(displayType))
	switch displayType {
	case NotifyDisplayPopup, "弹窗":
		return NotifyDisplayPopup
	default:
		return NotifyDisplayNormal
	}
}

func CreateAdminWebNotification(title, content, category, displayType string, isTop, visibleToAll bool) (*WebNotification, error) {
	title = strings.TrimSpace(title)
	content = strings.TrimSpace(content)
	if title == "" {
		return nil, fmt.Errorf("主题不能为空")
	}
	if content == "" {
		return nil, fmt.Errorf("详细内容不能为空")
	}
	n := &WebNotification{
		Title:        title,
		Content:      content,
		Category:     normalizeNoticeCategory(category),
		Source:       NotifySourceAdmin,
		Channels:     "web,app",
		TargetScope:  "all",
		DisplayType:  normalizeNotifyDisplayType(displayType),
		IsTop:        isTop,
		VisibleToAll: visibleToAll,
	}
	if err := db.Create(n).Error; err != nil {
		return nil, err
	}
	DispatchJPushNotification(n)
	return n, nil
}

// portalNotificationQuery 通知中心仅展示管理员全体公告（持久通知）
func portalNotificationQuery(userNumber int, hasAccess bool) *gorm.DB {
	q := db.Model(&WebNotification{}).Where(
		"target_scope = ? AND source = ?",
		TargetScopeAll, NotifySourceAdmin,
	)
	if !hasAccess {
		q = q.Where("visible_to_all = ?", true)
	}
	return q
}

// CreateSystemWebNotification 系统瞬时提示：仅 App 极光推送，不写库（机器人等渠道由调用方自行处理）
func CreateSystemWebNotification(title, content, category, source string, userNumber int, channels NotifyChannels) error {
	if !channels.App || userNumber <= 0 {
		return nil
	}
	title = strings.TrimSpace(title)
	content = strings.TrimSpace(content)
	if title == "" || content == "" {
		return nil
	}
	if source == "" {
		source = NotifySourceRobot
	}
	DispatchEphemeralPush(userNumber, title, content, normalizeNoticeCategory(category), source, "", NotifyDisplayNormal)
	return nil
}

// CleanupStaleOfflineNotifications 清理历史遗留的系统通知（迁移后不再写入库）
func CleanupStaleOfflineNotifications() int {
	return cleanupLegacySystemNotifications()
}

func cleanupLegacySystemNotifications() int {
	systemSources := []string{
		NotifySourceRobot, NotifySourceWx, NotifySourceYyb,
		NotifySourceAuth, NotifySourceFeedback, NotifySourceJd,
	}
	systemTitles := []string{
		NotifyTitleWxOffline, NotifyTitleYybOffline,
		"授权时间调整通知", "活动计费方式变更通知", "授权过期删除提醒", "反馈处理结果",
	}
	var ids []int
	db.Model(&WebNotification{}).
		Where("source IN ? OR title IN ?", systemSources, systemTitles).
		Where("source != ?", NotifySourceAdmin).
		Pluck("id", &ids)
	if len(ids) == 0 {
		return 0
	}
	if err := deleteNotificationsByIDs(ids); err != nil {
		System().Warnf("清理遗留系统通知失败: %v", err)
		return 0
	}
	return len(ids)
}

func deleteNotificationsByIDs(ids []int) error {
	cleanIDs := make([]int, 0, len(ids))
	seen := map[int]bool{}
	for _, id := range ids {
		if id > 0 && !seen[id] {
			cleanIDs = append(cleanIDs, id)
			seen[id] = true
		}
	}
	if len(cleanIDs) == 0 {
		return nil
	}
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("notification_id IN ?", cleanIDs).Delete(&WebNotificationRead{}).Error; err != nil {
			return err
		}
		return tx.Where("id IN ?", cleanIDs).Delete(&WebNotification{}).Error
	})
}

// PushUserOfflineNotification 协议掉线等系统提醒：仅 App 推送，不写库
func PushUserOfflineNotification(title, content, category, source string, userNumber int, channels NotifyChannels) error {
	if !channels.App || userNumber <= 0 {
		return nil
	}
	return CreateSystemWebNotification(title, content, category, source, userNumber, channels)
}

func CreateAdminOnlyWebNotification(title, content, category, source string, channels NotifyChannels) error {
	if !channels.Web && !channels.App {
		return nil
	}
	title = strings.TrimSpace(title)
	content = strings.TrimSpace(content)
	if title == "" || content == "" {
		return nil
	}
	if source == "" {
		source = NotifySourceAdmin
	}
	return db.Create(&WebNotification{
		Title:       title,
		Content:     content,
		Category:    normalizeNoticeCategory(category),
		Source:      source,
		Channels:    channelsToString(NotifyChannels{Web: channels.Web, App: channels.App}),
		TargetScope: TargetScopeAdmin,
		TargetUser:  0,
	}).Error
}

func GetAdminNotifications(search string, category string, page, limit int) ([]AdminNotificationItem, int64) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 200 {
		limit = 20
	}
	q := db.Model(&WebNotification{})
	search = strings.TrimSpace(search)
	if search != "" {
		like := "%" + search + "%"
		q = q.Where("title LIKE ? OR content LIKE ? OR category LIKE ? OR source LIKE ? OR channels LIKE ? OR display_type LIKE ?", like, like, like, like, like, like)
	}
	category = strings.TrimSpace(category)
	if category != "" {
		q = q.Where("category = ?", category)
	}
	var total int64
	q.Count(&total)
	var list []WebNotification
	q.Order("is_top desc").Order("id desc").Offset((page - 1) * limit).Limit(limit).Find(&list)
	items := make([]AdminNotificationItem, 0, len(list))
	for _, n := range list {
		items = append(items, AdminNotificationItem{
			ID: n.ID, Title: n.Title, Content: n.Content, Category: n.Category, Source: n.Source,
			Channels: n.Channels, TargetScope: n.TargetScope, TargetUser: n.TargetUser,
			ClickCount: n.ClickCount, ReadCount: n.ReadCount, DisplayType: normalizeNotifyDisplayType(n.DisplayType), IsTop: n.IsTop, VisibleToAll: n.VisibleToAll, CreatedAt: n.CreatedAt,
		})
	}
	return items, total
}

func GetPortalNotifications(userNumber int, category string, page, limit int) ([]PortalNotificationItem, int64, int64) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	coin := GetCoin(userNumber)
	hasAccess := CanAccessPortalContent(userNumber, coin)
	q := portalNotificationQuery(userNumber, hasAccess)
	if strings.TrimSpace(category) != "" && category != "全部" {
		q = q.Where("category = ?", category)
	}
	var total int64
	q.Count(&total)
	var list []WebNotification
	q.Order("is_top desc").Order("id desc").Offset((page - 1) * limit).Limit(limit).Find(&list)
	ids := make([]int, 0, len(list))
	for _, n := range list {
		ids = append(ids, n.ID)
	}
	reads := map[int]WebNotificationRead{}
	if len(ids) > 0 {
		var readRows []WebNotificationRead
		db.Where("user_number = ? AND notification_id IN ?", userNumber, ids).Find(&readRows)
		for _, r := range readRows {
			reads[r.NotificationID] = r
		}
	}
	items := make([]PortalNotificationItem, 0, len(list))
	for _, n := range list {
		r, ok := reads[n.ID]
		items = append(items, PortalNotificationItem{
			ID: n.ID, Title: n.Title, Category: n.Category, Source: n.Source, Channels: n.Channels,
			IsRead: ok, ClickCount: r.ClickCount, DisplayType: normalizeNotifyDisplayType(n.DisplayType), IsTop: n.IsTop, CreatedAt: n.CreatedAt, ReadAt: r.ReadAt,
		})
	}
	var unread int64
	portalNotificationQuery(userNumber, hasAccess).
		Where("id NOT IN (?)", db.Model(&WebNotificationRead{}).Select("notification_id").Where("user_number = ?", userNumber)).
		Count(&unread)
	return items, total, unread
}

func GetPortalNotificationCounts(userNumber int) (int64, int64) {
	coin := GetCoin(userNumber)
	hasAccess := CanAccessPortalContent(userNumber, coin)
	base := portalNotificationQuery(userNumber, hasAccess)
	var total int64
	base.Count(&total)
	var unread int64
	portalNotificationQuery(userNumber, hasAccess).
		Where("id NOT IN (?)", db.Model(&WebNotificationRead{}).Select("notification_id").Where("user_number = ?", userNumber)).
		Count(&unread)
	return total, unread
}

func GetPortalNotificationUnreadStats(userNumber int) NotifyUnreadStats {
	total, unread := GetPortalNotificationCounts(userNumber)
	stats := NotifyUnreadStats{
		Total:      total,
		Unread:     unread,
		BySource:   map[string]int64{},
		ByCategory: map[string]int64{},
	}
	var rows []WebNotification
	portalNotificationQuery(userNumber, CanAccessPortalContent(userNumber, GetCoin(userNumber))).
		Select("source, category").
		Where("id NOT IN (?)", db.Model(&WebNotificationRead{}).Select("notification_id").Where("user_number = ?", userNumber)).
		Find(&rows)
	for _, row := range rows {
		stats.BySource[row.Source]++
		stats.ByCategory[row.Category]++
	}
	return stats
}

func DeleteAdminNotification(id int) error {
	if id <= 0 {
		return fmt.Errorf("通知ID无效")
	}
	return DeleteAdminNotifications([]int{id})
}

func DeleteAdminNotifications(ids []int) error {
	cleanIDs := make([]int, 0, len(ids))
	seen := map[int]bool{}
	for _, id := range ids {
		if id > 0 && !seen[id] {
			cleanIDs = append(cleanIDs, id)
			seen[id] = true
		}
	}
	if len(cleanIDs) == 0 {
		return fmt.Errorf("请选择要删除的通知")
	}
	return deleteNotificationsByIDs(cleanIDs)
}

func UpdateAdminNotification(id int, title, content, category, displayType string, isTop, visibleToAll bool) error {
	if id <= 0 {
		return fmt.Errorf("通知ID无效")
	}
	title = strings.TrimSpace(title)
	content = strings.TrimSpace(content)
	if title == "" {
		return fmt.Errorf("主题不能为空")
	}
	if content == "" {
		return fmt.Errorf("详细内容不能为空")
	}
	res := db.Model(&WebNotification{}).Where("id = ?", id).Updates(map[string]interface{}{
		"title":          title,
		"content":        content,
		"category":       normalizeNoticeCategory(category),
		"display_type":   normalizeNotifyDisplayType(displayType),
		"is_top":         isTop,
		"visible_to_all": visibleToAll,
	})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return fmt.Errorf("通知不存在")
	}
	return nil
}

func GetPortalNotificationPreview(userNumber, id int) (*PortalNotificationItem, error) {
	coin := GetCoin(userNumber)
	hasAccess := CanAccessPortalContent(userNumber, coin)
	var n WebNotification
	if err := portalNotificationQuery(userNumber, hasAccess).Where("id = ?", id).First(&n).Error; err != nil {
		return nil, fmt.Errorf("通知不存在或无权限查看")
	}
	return &PortalNotificationItem{ID: n.ID, Title: n.Title, Content: n.Content, Category: n.Category, Source: n.Source, Channels: n.Channels, DisplayType: normalizeNotifyDisplayType(n.DisplayType), IsTop: n.IsTop, CreatedAt: n.CreatedAt}, nil
}

func GetPortalNotificationDetail(userNumber, id int) (*PortalNotificationItem, error) {
	coin := GetCoin(userNumber)
	hasAccess := CanAccessPortalContent(userNumber, coin)
	var n WebNotification
	if err := portalNotificationQuery(userNumber, hasAccess).Where("id = ?", id).First(&n).Error; err != nil {
		return nil, fmt.Errorf("通知不存在或无权限查看")
	}
	n.ClickCount++
	db.Save(&n)
	var r WebNotificationRead
	created := false
	if err := db.Where("notification_id = ? AND user_number = ?", id, userNumber).First(&r).Error; err != nil {
		r = WebNotificationRead{NotificationID: id, UserNumber: userNumber, ReadAt: time.Now(), ClickCount: 1}
		created = true
		db.Create(&r)
	} else {
		r.ClickCount++
		if r.ReadAt.IsZero() {
			r.ReadAt = time.Now()
		}
		db.Save(&r)
	}
	if created {
		n.ReadCount++
		db.Save(&n)
	}
	return &PortalNotificationItem{ID: n.ID, Title: n.Title, Content: n.Content, Category: n.Category, Source: n.Source, Channels: n.Channels, IsRead: true, ClickCount: r.ClickCount, DisplayType: normalizeNotifyDisplayType(n.DisplayType), IsTop: n.IsTop, CreatedAt: n.CreatedAt, ReadAt: r.ReadAt}, nil
}
