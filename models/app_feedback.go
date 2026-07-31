package models

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	feedbackUploadMaxImage = 10 * 1024 * 1024
	feedbackUploadMaxVideo = 50 * 1024 * 1024
	feedbackUploadMaxFiles = 6
)

var feedbackImageExt = map[string]bool{
	".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".webp": true, ".bmp": true,
}
var feedbackVideoExt = map[string]bool{
	".mp4": true, ".webm": true, ".mov": true, ".avi": true,
}

type AppFeedback struct {
	ID             int        `gorm:"primaryKey" json:"id"`
	UserID         int        `gorm:"index" json:"userId"`
	Type           string     `gorm:"size:32;index" json:"type"`
	Title          string     `gorm:"size:160;index" json:"title"`
	Content        string     `gorm:"type:text" json:"content"`
	Contact        string     `gorm:"size:120" json:"contact"`
	Attachments    string     `gorm:"type:text" json:"attachments"` // JSON 数组：/uploads/file/...
	Source         string     `gorm:"size:32;index" json:"source"`
	ClientPlatform string     `gorm:"size:16" json:"clientPlatform"`
	Status         string     `gorm:"size:24;index;default:new" json:"status"`
	Reply          string     `gorm:"type:text" json:"reply"`
	RewardCoin     int        `json:"rewardCoin"`
	Handler        string     `gorm:"size:80" json:"handler"`
	ProcessedAt    *time.Time `json:"processedAt"`
	CreatedAt      time.Time  `json:"createdAt"`
	UpdatedAt      time.Time  `json:"updatedAt"`
}

// AppFeedbackView API 输出（含来源展示标签）
type AppFeedbackView struct {
	AppFeedback
	SourceLabel    string   `json:"sourceLabel"`
	SourceTagCls   string   `json:"sourceTagCls"`
	AttachmentList []string `json:"attachmentList"`
}

func ToAppFeedbackView(item AppFeedback) AppFeedbackView {
	ctx := ClientContext{Source: item.Source, Platform: item.ClientPlatform}
	if strings.TrimSpace(item.Source) == "" {
		ctx = ClientContext{Source: ClientSourceApp}
	} else if strings.TrimSpace(item.ClientPlatform) == "" {
		ctx = NormalizeStoredSource(item.Source)
	}
	return AppFeedbackView{
		AppFeedback:    item,
		SourceLabel:    ctx.AdminLabel(),
		SourceTagCls:   ctx.AdminTagClass(),
		AttachmentList: ParseFeedbackAttachments(item.Attachments),
	}
}

func normalizeFeedbackType(value string) string {
	value = strings.TrimSpace(value)
	switch value {
	case "bug反馈", "活动投稿", "建议":
		return value
	case "bug", "BUG":
		return "bug反馈"
	case "post", "投稿":
		return "活动投稿"
	case "suggest", "suggestion":
		return "建议"
	default:
		return "建议"
	}
}

func CreateAppFeedback(userID int, feedbackType string, title string, content string, contact string, attachments []string, ctx ClientContext) error {
	title = strings.TrimSpace(title)
	content = strings.TrimSpace(content)
	contact = strings.TrimSpace(contact)
	ctx = ctx.WithDefault()
	if title == "" {
		title = "未填写主题"
	}
	validAttachments, err := ValidateFeedbackAttachments(userID, attachments)
	if err != nil {
		return err
	}
	item := AppFeedback{
		UserID:         userID,
		Type:           normalizeFeedbackType(feedbackType),
		Title:          title,
		Content:        content,
		Contact:        contact,
		Attachments:    EncodeFeedbackAttachments(validAttachments),
		Source:         ctx.FilterKey(),
		ClientPlatform: ctx.Platform,
		Status:         "new",
	}
	if err := db.Create(&item).Error; err != nil {
		return err
	}
	RecordClientSourceEvent(userID, SourceEventFeedback, ctx)
	return nil
}

func GetAdminAppFeedbacks(search string, page int, limit int, sortField string, sortOrder string) ([]AppFeedbackView, int64) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	query := db.Model(&AppFeedback{})
	search = strings.TrimSpace(search)
	if search != "" {
		like := "%" + search + "%"
		query = query.Where("title LIKE ? OR content LIKE ? OR contact LIKE ? OR source LIKE ? OR status LIKE ? OR type LIKE ? OR handler LIKE ? OR reply LIKE ? OR CAST(user_id AS CHAR) LIKE ?", like, like, like, like, like, like, like, like, like)
	}
	var total int64
	query.Count(&total)

	allowedSorts := map[string]string{
		"id":          "id",
		"userId":      "user_id",
		"type":        "type",
		"title":       "title",
		"status":      "status",
		"rewardCoin":  "reward_coin",
		"handler":     "handler",
		"processedAt": "processed_at",
		"createdAt":   "created_at",
	}
	orderClause := "id desc"
	if col, ok := allowedSorts[sortField]; ok {
		dir := "asc"
		if strings.ToLower(sortOrder) == "desc" {
			dir = "desc"
		}
		orderClause = col + " " + dir + ", id desc"
	}

	var list []AppFeedback
	query.Order(orderClause).Offset((page - 1) * limit).Limit(limit).Find(&list)
	views := make([]AppFeedbackView, 0, len(list))
	for _, item := range list {
		views = append(views, ToAppFeedbackView(item))
	}
	return views, total
}

func GetAdminAppFeedbackDetail(id int) (*AppFeedbackView, error) {
	if id <= 0 {
		return nil, fmt.Errorf("反馈ID不能为空")
	}
	var item AppFeedback
	if err := db.Where("id = ?", id).First(&item).Error; err != nil {
		return nil, err
	}
	view := ToAppFeedbackView(item)
	return &view, nil
}

func UpdateAppFeedbackStatus(id int, status string) error {
	status = strings.TrimSpace(status)
	if status == "" {
		status = "read"
	}
	return db.Model(&AppFeedback{}).Where("id = ?", id).Update("status", status).Error
}

func ProcessAppFeedback(id int, status string, reply string, rewardCoin int, handler string, isFirstReply bool, notifyWebApp bool, notifyBot bool) error {
	if id <= 0 {
		return fmt.Errorf("反馈ID不能为空")
	}
	status = strings.TrimSpace(status)
	if status == "" {
		status = "processed"
	}
	reply = strings.TrimSpace(reply)
	handler = strings.TrimSpace(handler)
	var item AppFeedback
	if err := db.Where("id = ?", id).First(&item).Error; err != nil {
		return err
	}
	now := time.Now()
	if err := db.Model(&AppFeedback{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":       status,
		"reply":        reply,
		"reward_coin":  rewardCoin,
		"handler":      handler,
		"processed_at": &now,
	}).Error; err != nil {
		return err
	}
	if rewardCoin > 0 && item.UserID > 0 {
		AdddCoin(item.UserID, rewardCoin)
		RecordCoinLog(item.UserID, rewardCoin, "反馈奖励", fmt.Sprintf("反馈#%d奖励积分", item.ID), AdminContext())
	}
	if isFirstReply && item.UserID > 0 && (reply != "" || rewardCoin > 0) {
		content := reply
		if content == "" {
			content = "你的反馈已处理"
		}
		if rewardCoin > 0 {
			content += fmt.Sprintf("\n奖励积分：%d", rewardCoin)
		}
		if notifyWebApp {
			_ = CreateSystemWebNotification("反馈处理结果", content, NotifyCategoryFeedback, NotifySourceFeedback, item.UserID, NotifyChannels{Web: true, App: true})
		}
		if notifyBot {
			pushFeedbackResultToBot(item.UserID, content)
		}
	}
	return nil
}

func pushFeedbackResultToBot(userID int, content string) {
	user, err := getPortalUserByNumber(userID)
	if err != nil || user == nil {
		return
	}
	wxid := strings.TrimSpace(user.Wxid)
	qqStr := strings.TrimSpace(user.QQ)
	msg := "【反馈处理结果】\n" + content
	if wxid != "" {
		SendWxMsg(wxid, msg)
	}
	if qqStr != "" {
		qq, err := strconv.Atoi(qqStr)
		if err == nil && qq > 0 {
			SendQQ(qq, msg)
		}
	}
}

func BatchDeleteAppFeedbacks(ids []int) error {
	cleanIDs := make([]int, 0, len(ids))
	seen := map[int]bool{}
	for _, id := range ids {
		if id > 0 && !seen[id] {
			cleanIDs = append(cleanIDs, id)
			seen[id] = true
		}
	}
	if len(cleanIDs) == 0 {
		return fmt.Errorf("请选择要删除的反馈")
	}
	res := db.Where("id IN ?", cleanIDs).Delete(&AppFeedback{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return fmt.Errorf("反馈不存在")
	}
	return nil
}

func BatchReplyAppFeedbacks(ids []int, reply string, handler string) (int, error) {
	cleanIDs := make([]int, 0, len(ids))
	seen := map[int]bool{}
	for _, id := range ids {
		if id > 0 && !seen[id] {
			cleanIDs = append(cleanIDs, id)
			seen[id] = true
		}
	}
	if len(cleanIDs) == 0 {
		return 0, fmt.Errorf("请选择要回复的反馈")
	}
	reply = strings.TrimSpace(reply)
	if reply == "" {
		return 0, fmt.Errorf("回复内容不能为空")
	}
	handler = strings.TrimSpace(handler)
	if handler == "" {
		handler = "大师"
	}
	now := time.Now()
	count := 0
	for _, id := range cleanIDs {
		var item AppFeedback
		if err := db.Where("id = ?", id).First(&item).Error; err != nil {
			continue
		}
		if err := db.Model(&AppFeedback{}).Where("id = ?", id).Updates(map[string]interface{}{
			"status":       "processed",
			"reply":        reply,
			"handler":      handler,
			"processed_at": &now,
		}).Error; err != nil {
			continue
		}
		if item.UserID > 0 && item.Status == "new" {
			_ = CreateSystemWebNotification("反馈处理结果", reply, NotifyCategoryFeedback, NotifySourceFeedback, item.UserID, NotifyChannels{Web: true, App: true})
			pushFeedbackResultToBot(item.UserID, reply)
		}
		count++
	}
	return count, nil
}

func BatchRewardAppFeedbacks(ids []int, rewardCoin int, handler string) (int, error) {
	cleanIDs := make([]int, 0, len(ids))
	seen := map[int]bool{}
	for _, id := range ids {
		if id > 0 && !seen[id] {
			cleanIDs = append(cleanIDs, id)
			seen[id] = true
		}
	}
	if len(cleanIDs) == 0 {
		return 0, fmt.Errorf("请选择要奖励的反馈")
	}
	if rewardCoin <= 0 {
		return 0, fmt.Errorf("奖励积分必须大于0")
	}
	handler = strings.TrimSpace(handler)
	if handler == "" {
		handler = "大师"
	}
	now := time.Now()
	count := 0
	for _, id := range cleanIDs {
		var item AppFeedback
		if err := db.Where("id = ?", id).First(&item).Error; err != nil {
			continue
		}
		if err := db.Model(&AppFeedback{}).Where("id = ?", id).Updates(map[string]interface{}{
			"status":       "processed",
			"reward_coin":  rewardCoin,
			"handler":      handler,
			"processed_at": &now,
		}).Error; err != nil {
			continue
		}
		if item.UserID > 0 && rewardCoin > 0 {
			AdddCoin(item.UserID, rewardCoin)
			RecordCoinLog(item.UserID, rewardCoin, "反馈奖励", fmt.Sprintf("反馈#%d奖励积分", item.ID), AdminContext())
			if item.Status == "new" {
				content := fmt.Sprintf("你的反馈已处理\n奖励积分：%d", rewardCoin)
				_ = CreateSystemWebNotification("反馈处理结果", content, NotifyCategoryFeedback, NotifySourceFeedback, item.UserID, NotifyChannels{Web: true, App: true})
				pushFeedbackResultToBot(item.UserID, content)
			}
		}
		count++
	}
	return count, nil
}

// SaveFeedbackUpload 保存用户投稿附件到 uploads/file/{userNumber}/
func SaveFeedbackUpload(userNumber int, filename string, data []byte) (string, error) {
	if userNumber <= 0 {
		return "", fmt.Errorf("无效用户")
	}
	ext := strings.ToLower(filepath.Ext(strings.TrimSpace(filename)))
	if !feedbackImageExt[ext] && !feedbackVideoExt[ext] {
		return "", fmt.Errorf("仅支持图片(jpg/png/gif/webp)或视频(mp4/webm/mov)")
	}
	maxSize := feedbackUploadMaxImage
	if feedbackVideoExt[ext] {
		maxSize = feedbackUploadMaxVideo
	}
	if len(data) == 0 || len(data) > maxSize {
		if feedbackVideoExt[ext] {
			return "", fmt.Errorf("视频大小不能超过50MB")
		}
		return "", fmt.Errorf("图片大小不能超过10MB")
	}
	dir := filepath.Join(ExecPath, "uploads", "file", fmt.Sprintf("%d", userNumber))
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("创建目录失败")
	}
	saveName := fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)
	savePath := filepath.Join(dir, saveName)
	if err := ioutil.WriteFile(savePath, data, 0644); err != nil {
		return "", fmt.Errorf("保存失败")
	}
	return fmt.Sprintf("/uploads/file/%d/%s", userNumber, saveName), nil
}

func ParseFeedbackAttachments(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "[]" {
		return nil
	}
	var urls []string
	if err := json.Unmarshal([]byte(raw), &urls); err != nil {
		return nil
	}
	out := make([]string, 0, len(urls))
	for _, u := range urls {
		u = strings.TrimSpace(u)
		if u != "" {
			out = append(out, u)
		}
	}
	return out
}

func EncodeFeedbackAttachments(urls []string) string {
	clean := make([]string, 0, len(urls))
	seen := map[string]bool{}
	for _, u := range urls {
		u = strings.TrimSpace(u)
		if u == "" || seen[u] {
			continue
		}
		if !strings.HasPrefix(u, "/uploads/file/") {
			continue
		}
		if _, ok := ResolveUploadAbsPath(u); !ok {
			continue
		}
		seen[u] = true
		clean = append(clean, u)
		if len(clean) >= feedbackUploadMaxFiles {
			break
		}
	}
	if len(clean) == 0 {
		return ""
	}
	b, _ := json.Marshal(clean)
	return string(b)
}

func ValidateFeedbackAttachments(userNumber int, urls []string) ([]string, error) {
	if len(urls) > feedbackUploadMaxFiles {
		return nil, fmt.Errorf("最多上传%d个附件", feedbackUploadMaxFiles)
	}
	out := make([]string, 0, len(urls))
	seen := map[string]bool{}
	prefix := fmt.Sprintf("/uploads/file/%d/", userNumber)
	for _, u := range urls {
		u = strings.TrimSpace(u)
		if u == "" || seen[u] {
			continue
		}
		if !strings.HasPrefix(u, prefix) {
			return nil, fmt.Errorf("附件路径无效")
		}
		abs, ok := ResolveUploadAbsPath(u)
		if !ok {
			return nil, fmt.Errorf("附件不存在或无权使用")
		}
		if st, err := os.Stat(abs); err != nil || st.IsDir() {
			return nil, fmt.Errorf("附件不存在")
		}
		seen[u] = true
		out = append(out, u)
	}
	return out, nil
}
