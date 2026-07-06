package controllers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/cdle/xdd/models"
	"github.com/cdle/xdd/vweb"
)

// PortalController 门户控制器，处理用户门户相关的所有页面和API请求
type PortalController struct {
	BaseController
}

func (c *PortalController) clientContext() models.ClientContext {
	return models.ResolveClientContext(
		c.Ctx.Input.Header("X-Request-Source"),
		c.Ctx.Input.Header("User-Agent"),
		c.Ctx.Input.Header("X-Sign-DeviceID"),
		c.Ctx.Input.Header("X-Client-Platform"),
	)
}

func (c *PortalController) requestSource() string {
	return c.clientContext().LegacyLabel()
}

// NextPrepare 前置处理，验证门户用户登录状态
func (c *PortalController) NextPrepare() {
	c.PortalLogined()
	c.ClientCtx = c.clientContext()
}

// Index 返回门户首页HTML页面
func (c *PortalController) Index() {
	file, err := vweb.ReadFile("html/portal.html")
	if err != nil {
		c.Ctx.WriteString("portal page not found")
		return
	}
	c.Ctx.WriteString(string(file))
}

// Dashboard 获取门户仪表盘数据，包含用户概览信息
func (c *PortalController) Dashboard() {
	if c.PortalAccount == nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": "登录状态失效，请重新登录"}
		c.ServeJSON()
		return
	}
	data, err := models.GetPortalDashboard(c.PortalAccount.ID)
	if err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": err.Error()}
		c.ServeJSON()
		return
	}
	c.Data["json"] = map[string]interface{}{"code": 0, "data": data}
	c.ServeJSON()
}

// Profile 获取用户个人资料信息
func (c *PortalController) Profile() {
	if c.PortalAccount == nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": "登录状态失效，请重新登录"}
		c.ServeJSON()
		return
	}
	data, err := models.GetPortalProfile(c.PortalAccount.ID)
	if err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": err.Error()}
		c.ServeJSON()
		return
	}
	c.Data["json"] = map[string]interface{}{"code": 0, "data": data}
	c.ServeJSON()
}

// Activities 获取门户活动列表
func (c *PortalController) Activities() {
	c.Data["json"] = map[string]interface{}{"code": 0, "data": models.GetPortalActivities()}
	c.ServeJSON()
}

// Projects 获取用户的项目列表
func (c *PortalController) Projects() {
	projects, err := models.GetPortalProjects(c.PortalUserID)
	if err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": err.Error()}
		c.ServeJSON()
		return
	}
	c.Data["json"] = map[string]interface{}{"code": 0, "data": projects}
	c.ServeJSON()
}

// CreateProject 创建新项目，选择活动并配置参数
func (c *PortalController) CreateProject() {
	var req struct {
		ActivityID string            `json:"activityId"`
		Inputs     map[string]string `json:"inputs"`
		Remarks    string            `json:"remarks"`
		Months     int               `json:"months"`
	}
	if err := json.Unmarshal(c.Ctx.Input.RequestBody, &req); err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": "请求数据格式错误"}
		c.ServeJSON()
		return
	}
	msg, err := models.PortalCreateProject(c.PortalUserID, req.ActivityID, req.Inputs, req.Remarks, req.Months, c.ClientCtx)
	if err != nil {
		c.logPortalWarn("上车失败 activity=%s: %v", req.ActivityID, err)
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": err.Error()}
		c.ServeJSON()
		return
	}
	c.logPortalInfo("上车成功 activity=%s remarks=%s", req.ActivityID, req.Remarks)
	c.Data["json"] = map[string]interface{}{"code": 0, "msg": msg}
	c.ServeJSON()
}

// RenewProject 续费已有项目
func (c *PortalController) RenewProject() {
	var req struct {
		ActivityID string `json:"activityId"`
		Remarks    string `json:"remarks"`
		Months     int    `json:"months"`
	}
	if err := json.Unmarshal(c.Ctx.Input.RequestBody, &req); err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": "请求数据格式错误"}
		c.ServeJSON()
		return
	}
	msg, err := models.PortalRenewProject(c.PortalUserID, req.ActivityID, req.Remarks, req.Months, c.ClientCtx)
	if err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": err.Error()}
		c.ServeJSON()
		return
	}
	c.Data["json"] = map[string]interface{}{"code": 0, "msg": msg}
	c.ServeJSON()
}

// DeleteProject 删除指定项目
func (c *PortalController) DeleteProject() {
	var req struct {
		ActivityID string `json:"activityId"`
		Remarks    string `json:"remarks"`
	}
	if err := json.Unmarshal(c.Ctx.Input.RequestBody, &req); err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": "请求数据格式错误"}
		c.ServeJSON()
		return
	}
	msg, err := models.PortalDeleteProject(c.PortalUserID, req.ActivityID, req.Remarks, c.ClientCtx)
	if err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": err.Error()}
		c.ServeJSON()
		return
	}
	c.Data["json"] = map[string]interface{}{"code": 0, "msg": msg}
	c.ServeJSON()
}

// UpdateProject 更新项目配置，如修改Cookie值
func (c *PortalController) UpdateProject() {
	var req struct {
		ActivityID string `json:"activityId"`
		Remarks    string `json:"remarks"`
		CkValue    string `json:"ckValue"`
	}
	if err := json.Unmarshal(c.Ctx.Input.RequestBody, &req); err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": "请求数据格式错误"}
		c.ServeJSON()
		return
	}
	msg, err := models.PortalUpdateProject(c.PortalUserID, req.ActivityID, req.Remarks, req.CkValue)
	if err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": err.Error()}
		c.ServeJSON()
		return
	}
	c.Data["json"] = map[string]interface{}{"code": 0, "msg": msg}
	c.ServeJSON()
}

// QueryIncome 查询项目收益情况
func (c *PortalController) QueryIncome() {
	var req struct {
		ActivityID string `json:"activityId"`
		Remarks    string `json:"remarks"`
	}
	if err := json.Unmarshal(c.Ctx.Input.RequestBody, &req); err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": "请求数据格式错误"}
		c.ServeJSON()
		return
	}
	data, err := models.PortalQueryProjectIncome(c.PortalUserID, req.ActivityID, req.Remarks)
	if err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": err.Error()}
		c.ServeJSON()
		return
	}
	c.Data["json"] = map[string]interface{}{"code": 0, "data": data, "msg": "查询成功"}
	c.ServeJSON()
}

// RedeemKey 使用兑换码兑换余额
func (c *PortalController) RedeemKey() {
	var req struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(c.Ctx.Input.RequestBody, &req); err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": "请求数据格式错误"}
		c.ServeJSON()
		return
	}
	msg, balance, err := models.PortalRedeemKey(c.PortalUserID, req.Token, c.ClientCtx)
	if err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": msg, "balance": balance}
		c.ServeJSON()
		return
	}
	c.Data["json"] = map[string]interface{}{"code": 0, "msg": msg, "balance": balance}
	c.ServeJSON()
}

// CheckIn 每日签到，获取签到奖励
func (c *PortalController) CheckIn() {
	// 验证签名
	if !verifyRequestSignature(c) {
		return
	}

	msg, err := models.PortalCheckIn(c.PortalUserID, c.ClientCtx)
	if err != nil {
		c.logPortalWarn("签到失败: %v", err)
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": err.Error()}
		c.ServeJSON()
		return
	}
	c.logPortalInfo("签到成功: %s", msg)
	c.Data["json"] = map[string]interface{}{"code": 0, "msg": msg}
	c.ServeJSON()
}

// verifyRequestSignature 验证请求签名
func verifyRequestSignature(c *PortalController) bool {
	headers := map[string]string{
		"X-Sign-Timestamp":  c.Ctx.Input.Header("X-Sign-Timestamp"),
		"X-Sign-Nonce":      c.Ctx.Input.Header("X-Sign-Nonce"),
		"X-Sign-DeviceID":   c.Ctx.Input.Header("X-Sign-DeviceID"),
		"X-Sign-Value":      c.Ctx.Input.Header("X-Sign-Value"),
		"X-Sign-Version":    c.Ctx.Input.Header("X-Sign-Version"),
		"X-App-Version":     c.Ctx.Input.Header("X-App-Version"),
	}

	path := c.Ctx.Input.URL()
	result := models.VerifyRequest(headers, path)

	if !result.Valid {
		c.Data["json"] = map[string]interface{}{
			"code":         4003,
			"msg":          result.Message,
			"need_upgrade": true,
		}
		c.ServeJSON()
		return false
	}
	return true
}

// Pray 祈祷功能，随机获取奖励
func (c *PortalController) Pray() {
	// 验证签名
	if !verifyRequestSignature(c) {
		return
	}

	msg, err := models.PortalPray(c.PortalUserID, c.ClientCtx)
	if err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": err.Error()}
		c.ServeJSON()
		return
	}
	c.Data["json"] = map[string]interface{}{"code": 0, "msg": msg}
	c.ServeJSON()
}

// SignParams 服务端生成签名参数（供网页端使用，避免密钥暴露在前端）
func (c *PortalController) SignParams() {
	path := c.GetString("path")
	if path == "" {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": "缺少path参数"}
		c.ServeJSON()
		return
	}
	params := models.GenerateRequestParams("web-"+c.Ctx.Input.IP(), "2.0.0", path)
	c.Data["json"] = map[string]interface{}{"code": 0, "data": params}
	c.ServeJSON()
}

// WxStatus 查询微信机器人连接状态
func (c *PortalController) WxStatus() {
	data, err := models.GetPortalWxStatus(c.PortalUserID)
	if err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": err.Error()}
		c.ServeJSON()
		return
	}
	c.Data["json"] = map[string]interface{}{"code": 0, "data": data, "msg": "查询成功"}
	c.ServeJSON()
}

// WxScanLogin 微信扫码登录功能
func (c *PortalController) WxScanLogin() {
	data, err := models.PortalWxScanLogin(c.PortalUserID)
	if err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": err.Error()}
		c.ServeJSON()
		return
	}
	c.Data["json"] = map[string]interface{}{"code": 0, "data": data, "msg": data.Message}
	c.ServeJSON()
}

// WxDevices 获取微信设备列表
func (c *PortalController) WxDevices() {
	data, err := models.GetPortalWxDevices(c.PortalUserID)
	if err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": err.Error()}
		c.ServeJSON()
		return
	}
	c.Data["json"] = map[string]interface{}{"code": 0, "data": data, "msg": "查询成功"}
	c.ServeJSON()
}

// WxAddDevice 添加微信设备
func (c *PortalController) WxAddDevice() {
	var req struct {
		Wxid string `json:"wxid"`
	}
	if err := json.Unmarshal(c.Ctx.Input.RequestBody, &req); err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": "请求数据格式错误"}
		c.ServeJSON()
		return
	}
	data, err := models.AddPortalWxDevice(c.PortalUserID, req.Wxid)
	if err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": err.Error()}
		c.ServeJSON()
		return
	}
	c.Data["json"] = map[string]interface{}{"code": 0, "data": data, "msg": "添加成功"}
	c.ServeJSON()
}

// WxRemoveDevice 移除微信设备
func (c *PortalController) WxRemoveDevice() {
	var req struct {
		ID int `json:"id"`
	}
	if err := json.Unmarshal(c.Ctx.Input.RequestBody, &req); err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": "请求数据格式错误"}
		c.ServeJSON()
		return
	}
	if err := models.RemovePortalWxDevice(c.PortalUserID, req.ID); err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": err.Error()}
		c.ServeJSON()
		return
	}
	c.Data["json"] = map[string]interface{}{"code": 0, "msg": "删除成功"}
	c.ServeJSON()
}

// WxRelogin 微信重新登录
func (c *PortalController) WxRelogin() {
	var req struct {
		Wxid string `json:"wxid"`
	}
	json.Unmarshal(c.Ctx.Input.RequestBody, &req)
	var data *models.PortalWxActionResult
	var err error
	if req.Wxid != "" {
		data, err = models.PortalWxRelogin(c.PortalUserID, req.Wxid)
	} else {
		data, err = models.PortalWxRelogin(c.PortalUserID)
	}
	if err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": err.Error()}
		c.ServeJSON()
		return
	}
	c.Data["json"] = map[string]interface{}{"code": 0, "data": data, "msg": data.Message}
	c.ServeJSON()
}

// WxWakeLogin 微信唤醒登录
func (c *PortalController) WxWakeLogin() {
	var req struct {
		Wxid string `json:"wxid"`
	}
	json.Unmarshal(c.Ctx.Input.RequestBody, &req)
	var data *models.PortalWxActionResult
	var err error
	if req.Wxid != "" {
		data, err = models.PortalWxWakeLogin(c.PortalUserID, req.Wxid)
	} else {
		data, err = models.PortalWxWakeLogin(c.PortalUserID)
	}
	if err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": err.Error()}
		c.ServeJSON()
		return
	}
	c.Data["json"] = map[string]interface{}{"code": 0, "data": data, "msg": data.Message}
	c.ServeJSON()
}

// WxLogout 微信退出登录
func (c *PortalController) WxLogout() {
	var req struct {
		Wxid string `json:"wxid"`
	}
	json.Unmarshal(c.Ctx.Input.RequestBody, &req)
	var data *models.PortalWxActionResult
	var err error
	if req.Wxid != "" {
		data, err = models.PortalWxLogout(c.PortalUserID, req.Wxid)
	} else {
		data, err = models.PortalWxLogout(c.PortalUserID)
	}
	if err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": err.Error()}
		c.ServeJSON()
		return
	}
	c.Data["json"] = map[string]interface{}{"code": 0, "data": data, "msg": data.Message}
	c.ServeJSON()
}

// SubmitFeedback 提交用户反馈
func (c *PortalController) SubmitFeedback() {
	var req struct {
		Type    string `json:"type"`
		Title   string `json:"title"`
		Content string `json:"content"`
		Contact string `json:"contact"`
	}
	if err := json.Unmarshal(c.Ctx.Input.RequestBody, &req); err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": "请求数据格式错误"}
		c.ServeJSON()
		return
	}
	title := strings.TrimSpace(req.Title)
	content := strings.TrimSpace(req.Content)
	contact := strings.TrimSpace(req.Contact)
	if title == "" {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": "主题不能为空"}
		c.ServeJSON()
		return
	}
	if content == "" {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": "详细内容不能为空"}
		c.ServeJSON()
		return
	}
	if err := models.CreateAppFeedback(c.PortalUserID, req.Type, title, content, contact, c.ClientCtx); err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": err.Error()}
		c.ServeJSON()
		return
	}
	c.Data["json"] = map[string]interface{}{"code": 0, "msg": "提交成功，管理员会在后台处理"}
	c.ServeJSON()
}

func (c *PortalController) WxDelete() {
	var req struct {
		Wxid string `json:"wxid"`
	}
	json.Unmarshal(c.Ctx.Input.RequestBody, &req)
	var data *models.PortalWxActionResult
	var err error
	if req.Wxid != "" {
		data, err = models.PortalWxDelete(c.PortalUserID, req.Wxid)
	} else {
		data, err = models.PortalWxDelete(c.PortalUserID)
	}
	if err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": err.Error()}
		c.ServeJSON()
		return
	}
	c.Data["json"] = map[string]interface{}{"code": 0, "data": data, "msg": data.Message}
	c.ServeJSON()
}

func (c *PortalController) WxPollLogin() {
	var req struct {
		UUID       string `json:"uuid"`
		DeductCoin bool   `json:"deductCoin"`
	}
	if err := json.Unmarshal(c.Ctx.Input.RequestBody, &req); err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": "请求数据格式错误"}
		c.ServeJSON()
		return
	}
	data, err := models.PortalWxPollLogin(c.PortalUserID, req.UUID, req.DeductCoin, c.ClientCtx)
	if err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": err.Error()}
		c.ServeJSON()
		return
	}
	c.Data["json"] = map[string]interface{}{"code": 0, "data": data, "msg": data.Message}
	c.ServeJSON()
}

func (c *PortalController) Notifications() {
	category := c.GetString("category")
	page := c.GetQueryInt("page")
	limit := c.GetQueryInt("limit")
	includeContent := c.GetString("includeContent") == "1"
	list, total, unread := models.GetPortalNotifications(c.PortalUserID, category, page, limit)
	if includeContent {
		for i := range list {
			if item, err := models.GetPortalNotificationPreview(c.PortalUserID, list[i].ID); err == nil && item != nil {
				list[i].Content = item.Content
			}
		}
	}
	unreadStats := models.GetPortalNotificationUnreadStats(c.PortalUserID)
	c.Data["json"] = map[string]interface{}{
		"code": 0,
		"data": map[string]interface{}{
			"list":        list,
			"total":       total,
			"unread":      unread,
			"unreadStats": unreadStats,
		},
	}
	c.ServeJSON()
}

func (c *PortalController) NotificationDetail() {
	id := c.GetQueryInt("id")
	item, err := models.GetPortalNotificationDetail(c.PortalUserID, id)
	if err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": err.Error()}
		c.ServeJSON()
		return
	}
	c.Data["json"] = map[string]interface{}{"code": 0, "data": item}
	c.ServeJSON()
}

// CoinLogs 获取用户积分变动记录
func (c *PortalController) CoinLogs() {
	logs := models.GetCoinLogs(c.PortalUserID, 50)
	sourceFilter := models.NormalizeSourceFilter(c.GetString("source", ""))

	result := make([]models.CoinLogView, 0, len(logs))
	for _, l := range logs {
		view := models.ToCoinLogView(l)
		if sourceFilter != "" && view.Source != sourceFilter {
			continue
		}
		result = append(result, view)
	}
	c.Data["json"] = map[string]interface{}{"code": 0, "data": result}
	c.ServeJSON()
}

// JdAccounts 获取用户绑定的京东账号列表
func (c *PortalController) JdAccounts() {
	accounts := models.GetPortalJdAccounts(c.PortalUserID)
	c.Data["json"] = map[string]interface{}{"code": 0, "data": accounts}
	c.ServeJSON()
}

// JdQuery 查询京东账号资产
func (c *PortalController) JdQuery() {
	var req struct {
		Index int `json:"index"`
	}
	if err := json.Unmarshal(c.Ctx.Input.RequestBody, &req); err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": "请求数据格式错误"}
		c.ServeJSON()
		return
	}
	result, err := models.PortalJdQuery(c.PortalUserID, req.Index)
	if err != nil {
		c.logPortalWarn("京东查询失败 index=%d: %v", req.Index, err)
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": err.Error()}
		c.ServeJSON()
		return
	}
	c.logPortalInfo("京东查询成功 index=%d", req.Index)
	c.Data["json"] = map[string]interface{}{"code": 0, "data": result}
	c.ServeJSON()
}

// JdSmsSend 发送京东短信验证码
func (c *PortalController) JdSmsSend() {
	var req struct {
		Phone string `json:"phone"`
	}
	if err := json.Unmarshal(c.Ctx.Input.RequestBody, &req); err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": "请求数据格式错误"}
		c.ServeJSON()
		return
	}
	msg, err := models.PortalJdSmsSend(c.PortalUserID, strings.TrimSpace(req.Phone))
	if err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": err.Error()}
		c.ServeJSON()
		return
	}
	c.Data["json"] = map[string]interface{}{"code": 0, "msg": msg}
	c.ServeJSON()
}

// JdSmsVerify 提交京东短信验证码
func (c *PortalController) JdSmsVerify() {
	var req struct {
		Phone  string `json:"phone"`
		Code   string `json:"code"`
		IdCard string `json:"idCard"`
	}
	if err := json.Unmarshal(c.Ctx.Input.RequestBody, &req); err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": "请求数据格式错误"}
		c.ServeJSON()
		return
	}
	result, err := models.PortalJdSmsVerify(c.PortalUserID, strings.TrimSpace(req.Phone), strings.TrimSpace(req.Code), strings.TrimSpace(req.IdCard))
	if err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": err.Error()}
		c.ServeJSON()
		return
	}
	c.Data["json"] = map[string]interface{}{"code": 0, "data": result}
	c.ServeJSON()
}

// JdWxDevices 获取可用于京东登录的微信协议设备
func (c *PortalController) JdWxDevices() {
	devices, err := models.GetPortalJdWxDevices(c.PortalUserID)
	if err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": err.Error()}
		c.ServeJSON()
		return
	}
	c.Data["json"] = map[string]interface{}{"code": 0, "data": devices}
	c.ServeJSON()
}

// JdWxRefresh 通过微信协议刷新京东CK
func (c *PortalController) JdWxRefresh() {
	var req struct {
		Wxid          string `json:"wxid"`
		RiskConfirmed bool   `json:"riskConfirmed"`
	}
	if err := json.Unmarshal(c.Ctx.Input.RequestBody, &req); err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": "请求数据格式错误"}
		c.ServeJSON()
		return
	}
	result, err := models.PortalJdWxRefresh(c.PortalUserID, strings.TrimSpace(req.Wxid), req.RiskConfirmed)
	if err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": err.Error()}
		c.ServeJSON()
		return
	}
	c.Data["json"] = map[string]interface{}{"code": 0, "data": result}
	c.ServeJSON()
}

// JdWxContinueRisk 风控验证完成后继续刷新
func (c *PortalController) JdWxContinueRisk() {
	result, err := models.PortalJdWxContinueAfterRisk(c.PortalUserID)
	if err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": err.Error()}
		c.ServeJSON()
		return
	}
	c.Data["json"] = map[string]interface{}{"code": 0, "data": result}
	c.ServeJSON()
}

// JdTaskExecute 执行京东任务
func (c *PortalController) JdTaskExecute() {
	var req struct {
		TaskId         string `json:"taskId"`
		TaskName       string `json:"taskName"`
		AccountIndexes []int  `json:"accountIndexes"`
	}
	if err := json.Unmarshal(c.Ctx.Input.RequestBody, &req); err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": "请求数据格式错误"}
		c.ServeJSON()
		return
	}

	if req.TaskId == "" || len(req.AccountIndexes) == 0 {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": "参数不完整"}
		c.ServeJSON()
		return
	}

	// 检查是否有同一任务的同一账号正在执行
	conflictTask := models.GetRunningTask(c.PortalUserID, req.TaskId, req.AccountIndexes)
	if conflictTask != "" {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": fmt.Sprintf("该任务的某些账号正在执行中：%s", conflictTask)}
		c.ServeJSON()
		return
	}

	// 生成任务ID
	taskLogId := fmt.Sprintf("%s_%d_%d", req.TaskId, c.PortalUserID, time.Now().UnixNano())
	c.logPortalInfo("启动京东任务 task=%s name=%s accounts=%v source=%s platform=%s", req.TaskId, req.TaskName, req.AccountIndexes, c.ClientCtx.Source, c.ClientCtx.Platform)

	// 创建日志通道并入队
	models.CreateTaskLogChannel(taskLogId)
	if err := models.SubmitPortalJdTask(c.PortalUserID, req.TaskId, req.TaskName, req.AccountIndexes, taskLogId, c.ClientCtx); err != nil {
		models.RemoveTaskLogChannel(taskLogId)
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": err.Error()}
		c.ServeJSON()
		return
	}

	c.Data["json"] = map[string]interface{}{"code": 0, "msg": "任务已启动", "data": map[string]string{"taskId": taskLogId}}
	c.ServeJSON()
}

// JdTaskLogs SSE 实时日志流
func (c *PortalController) JdTaskLogs() {
	taskId := c.GetString("taskId")
	if taskId == "" {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": "缺少taskId参数"}
		c.ServeJSON()
		return
	}

	// 设置 SSE 响应头
	c.Ctx.Output.Header("Content-Type", "text/event-stream")
	c.Ctx.Output.Header("Cache-Control", "no-cache")
	c.Ctx.Output.Header("Connection", "keep-alive")
	c.Ctx.Output.Header("Access-Control-Allow-Origin", "*")
	c.Ctx.Output.Header("X-Accel-Buffering", "no")

	// 获取底层 http.ResponseWriter 和 Flusher
	w := c.Ctx.ResponseWriter.ResponseWriter
	flusher, ok := w.(http.Flusher)
	if !ok {
		c.Ctx.WriteString("event: error\ndata: 服务器不支持SSE\n\n")
		return
	}

	// 获取日志通道
	logChan := models.GetTaskLogChannel(taskId)
	if logChan == nil {
		fmt.Fprintf(w, "event: error\ndata: 任务不存在或已结束\n\n")
		flusher.Flush()
		return
	}

	// 使用超时机制检测客户端断开
	timeout := time.NewTimer(5 * time.Minute)
	defer timeout.Stop()

	// 持续发送日志
	for {
		select {
		case log, ok := <-logChan:
			if !ok {
				// 通道关闭，任务结束
				fmt.Fprintf(w, "event: done\ndata: 任务执行完成\n\n")
				flusher.Flush()
				return
			}
			// 发送日志数据
			fmt.Fprintf(w, "data: %s\n\n", log)
			flusher.Flush()
			// 重置超时
			timeout.Reset(5 * time.Minute)
		case <-timeout.C:
			// 超时，任务可能卡住
			fmt.Fprintf(w, "event: timeout\ndata: 连接超时\n\n")
			flusher.Flush()
			return
		}
	}
}

// JdTaskStop 停止正在执行的任务
func (c *PortalController) JdTaskStop() {
	var req struct {
		TaskId string `json:"taskId"`
	}
	if err := json.Unmarshal(c.Ctx.Input.RequestBody, &req); err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": "请求数据格式错误"}
		c.ServeJSON()
		return
	}

	if req.TaskId == "" {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": "缺少taskId参数"}
		c.ServeJSON()
		return
	}

	models.StopPortalJdTask(req.TaskId)

	c.Data["json"] = map[string]interface{}{"code": 0, "msg": "已发送停止指令"}
	c.ServeJSON()
}

// KuwoGetCredentials 读取用户挂活动提交的酷我账号密码
func (c *PortalController) KuwoGetCredentials() {
	profile, err := models.GetPortalProfile(c.PortalAccount.ID)
	if err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": err.Error()}
		c.ServeJSON()
		return
	}
	accounts, err := models.GetAllKuwoCredentials(profile.User.Number)
	if err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": err.Error()}
		c.ServeJSON()
		return
	}
	if len(accounts) == 0 {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": "未找到酷我音乐活动配置"}
		c.ServeJSON()
		return
	}
	// 默认返回第一个账号（兼容旧逻辑）
	c.Data["json"] = map[string]interface{}{
		"code": 0,
		"data": map[string]interface{}{
			"phone":    accounts[0].Phone,
			"password": accounts[0].Password,
			"accounts": accounts, // 全部账号列表，供前端下拉选择
		},
	}
	c.ServeJSON()
}

// KuwoCheckAuth 检查用户是否有酷我活动授权
func (c *PortalController) KuwoCheckAuth() {
	profile, err := models.GetPortalProfile(c.PortalAccount.ID)
	if err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "authorized": false, "msg": err.Error()}
		c.ServeJSON()
		return
	}
	authorized, msg := models.CheckKuwoAuth(profile.User.Number)
	c.Data["json"] = map[string]interface{}{"code": 0, "authorized": authorized, "msg": msg}
	c.ServeJSON()
}

// KuwoLogin 酷我账号登录，获取loginUid和loginSid
func (c *PortalController) KuwoLogin() {
	var req struct {
		Phone    string `json:"phone"`
		Password string `json:"password"`
	}
	if err := json.Unmarshal(c.Ctx.Input.RequestBody, &req); err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": "请求数据格式错误"}
		c.ServeJSON()
		return
	}
	phone := strings.TrimSpace(req.Phone)
	password := strings.TrimSpace(req.Password)
	if phone == "" || password == "" {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": "手机号和密码不能为空"}
		c.ServeJSON()
		return
	}
	session, err := models.KuwoLogin(phone, password)
	if err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": err.Error()}
		c.ServeJSON()
		return
	}
	c.RecordPortalEvent(models.SourceEventKuwoLogin)
	c.Data["json"] = map[string]interface{}{
		"code": 0,
		"msg":  "登录成功",
		"data": map[string]string{
			"phone":          session.Phone,
			"encryptedPhone": session.EncryptedPhone,
			"loginUid":       session.LoginUid,
			"loginSid":       session.LoginSid,
		},
	}
	c.ServeJSON()
}

// KuwoSendSms 发送酷我提现短信验证码
func (c *PortalController) KuwoSendSms() {
	var req struct {
		Phone    string `json:"phone"`
		Password string `json:"password"`
	}
	if err := json.Unmarshal(c.Ctx.Input.RequestBody, &req); err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": "请求数据格式错误"}
		c.ServeJSON()
		return
	}
	phone := strings.TrimSpace(req.Phone)
	password := strings.TrimSpace(req.Password)
	if phone == "" || password == "" {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": "手机号和密码不能为空"}
		c.ServeJSON()
		return
	}
	session, err := models.KuwoLogin(phone, password)
	if err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": err.Error()}
		c.ServeJSON()
		return
	}
	// 缓存session，到点抢兑时直接用，避免重新登录
	models.KuwoCacheSession(phone, session)
	if err := models.KuwoSendSms(session); err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": err.Error()}
		c.ServeJSON()
		return
	}
	c.RecordPortalEvent(models.SourceEventKuwoSms)
	c.Data["json"] = map[string]interface{}{
		"code": 0,
		"msg":  "验证码已发送",
		"data": map[string]string{
			"phone":          session.Phone,
			"encryptedPhone": session.EncryptedPhone,
			"loginUid":       session.LoginUid,
			"loginSid":       session.LoginSid,
		},
	}
	c.ServeJSON()
}

// KuwoWithdraw 酷我提现（支持并发+重试）
func (c *PortalController) KuwoWithdraw() {
	var req struct {
		Sessions []struct {
			Phone         string `json:"phone"`
			Password      string `json:"password"`
			EncryptedPhone string `json:"encryptedPhone"`
			LoginUID      string `json:"loginUid"`
			LoginSID      string `json:"loginSid"`
		} `json:"sessions"`
		QuotaId    string `json:"quotaId"`
		SmsCode    string `json:"smsCode"`
		RetryCount int    `json:"retryCount"`
		UseProxy   *bool  `json:"useProxy"`
	}
	if err := json.Unmarshal(c.Ctx.Input.RequestBody, &req); err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": "请求数据格式错误"}
		c.ServeJSON()
		return
	}
	if len(req.Sessions) == 0 {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": "没有可提现的账号"}
		c.ServeJSON()
		return
	}
	quotaId := strings.TrimSpace(req.QuotaId)
	smsCode := strings.TrimSpace(req.SmsCode)
	if quotaId == "" {
		quotaId = "30002"
	}
	if smsCode == "" {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": "验证码不能为空"}
		c.ServeJSON()
		return
	}
	useProxy := true
	if req.UseProxy != nil {
		useProxy = *req.UseProxy
	}

	accounts := make([]*models.KuwoAccountInput, 0, len(req.Sessions))
	for _, s := range req.Sessions {
		accounts = append(accounts, &models.KuwoAccountInput{
			Phone:    strings.TrimSpace(s.Phone),
			Password: strings.TrimSpace(s.Password),
		})
	}
	sessions := models.KuwoBuildSessionsFromRequest(accounts)
	if len(sessions) == 0 {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": "登录失败，无法获取有效会话"}
		c.ServeJSON()
		return
	}

	results, proxyHost := models.KuwoManualWithdraw(sessions, quotaId, smsCode, useProxy)
	c.RecordPortalEvent(models.SourceEventKuwoWithdraw)

	type withdrawResult struct {
		Phone   string `json:"phone"`
		Success bool   `json:"success"`
		Message string `json:"message"`
		Error   string `json:"error,omitempty"`
	}
	resultList := make([]withdrawResult, 0, len(results))
	for _, r := range results {
		errMsg := ""
		if r.Error != nil {
			errMsg = r.Error.Error()
		}
		resultList = append(resultList, withdrawResult{
			Phone:   r.Phone,
			Success: r.Success,
			Message: r.Message,
			Error:   errMsg,
		})
	}
	resp := map[string]interface{}{"code": 0, "data": resultList}
	if proxyHost != "" {
		resp["proxyHost"] = proxyHost
	}
	c.Data["json"] = resp
	c.ServeJSON()
}

// KuwoScheduleWithdraw 创建后端定时抢兑任务
func (c *PortalController) KuwoScheduleWithdraw() {
	var req struct {
		Sessions []struct {
			Phone          string `json:"phone"`
			Password       string `json:"password"`
			EncryptedPhone string `json:"encryptedPhone"`
			LoginUID       string `json:"loginUid"`
			LoginSID       string `json:"loginSid"`
		} `json:"sessions"`
		QuotaId    string `json:"quotaId"`
		SmsCode    string `json:"smsCode"`
		TargetHour int    `json:"targetHour"`
		Immediate  bool   `json:"immediate"`
		UseProxy   *bool  `json:"useProxy"`
	}
	if err := json.Unmarshal(c.Ctx.Input.RequestBody, &req); err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": "请求数据格式错误"}
		c.ServeJSON()
		return
	}
	if len(req.Sessions) == 0 {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": "没有可提现的账号"}
		c.ServeJSON()
		return
	}
	quotaId := strings.TrimSpace(req.QuotaId)
	smsCode := strings.TrimSpace(req.SmsCode)
	if quotaId == "" {
		quotaId = "30002"
	}
	if smsCode == "" {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": "验证码不能为空"}
		c.ServeJSON()
		return
	}
	if !req.Immediate {
		if req.TargetHour < 0 || req.TargetHour > 23 {
			c.Data["json"] = map[string]interface{}{"code": 1, "msg": "无效的目标小时"}
			c.ServeJSON()
			return
		}
	}
	useProxy := true
	if req.UseProxy != nil {
		useProxy = *req.UseProxy
	}

	accounts := make([]*models.KuwoAccountInput, 0, len(req.Sessions))
	for _, s := range req.Sessions {
		accounts = append(accounts, &models.KuwoAccountInput{
			Phone:    strings.TrimSpace(s.Phone),
			Password: strings.TrimSpace(s.Password),
		})
	}

	task, reused := models.KuwoScheduleWithdraw(accounts, quotaId, smsCode, req.TargetHour, req.Immediate, useProxy)
	c.RecordPortalEvent(models.SourceEventKuwoSchedule)
	msg := "任务已创建"
	if reused {
		msg = "已有进行中的抢兑任务，已返回现有任务"
	}
	c.Data["json"] = map[string]interface{}{
		"code": 0,
		"msg":  msg,
		"data": map[string]interface{}{
			"taskId":     task.ID,
			"targetHour": task.TargetHour,
			"executeAt":  task.ExecuteAt.Format("2006-01-02 15:04:05"),
			"status":     task.Status,
			"reused":     reused,
			"useProxy":   task.UseProxy,
		},
	}
	c.ServeJSON()
}

// KuwoUpdateSmsCode 倒计时 pending 阶段更新短信验证码
func (c *PortalController) KuwoUpdateSmsCode() {
	var req struct {
		TaskID  string `json:"taskId"`
		SmsCode string `json:"smsCode"`
	}
	if err := json.Unmarshal(c.Ctx.Input.RequestBody, &req); err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": "请求数据格式错误"}
		c.ServeJSON()
		return
	}
	if err := models.KuwoUpdateTaskSmsCode(req.TaskID, req.SmsCode); err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": err.Error()}
		c.ServeJSON()
		return
	}
	c.Data["json"] = map[string]interface{}{"code": 0, "msg": "验证码已更新"}
	c.ServeJSON()
}

// KuwoGetWithdrawStatus 查询定时抢兑任务状态
func (c *PortalController) KuwoGetWithdrawStatus() {
	taskID := c.Ctx.Input.Query("taskId")
	phone := strings.TrimSpace(c.Ctx.Input.Query("phone"))
	if taskID == "" && phone != "" {
		task := models.KuwoGetActiveTaskByPhone(phone)
		if task == nil {
			c.Data["json"] = map[string]interface{}{"code": 0, "data": nil}
			c.ServeJSON()
			return
		}
		c.Data["json"] = map[string]interface{}{
			"code": 0,
			"data": kuwoTaskStatusPayload(task),
		}
		c.ServeJSON()
		return
	}
	if taskID == "" {
		// 返回所有任务
		tasks := models.KuwoListScheduledTasks()
		type taskJSON struct {
			ID          string                      `json:"id"`
			Phone       string                      `json:"phone"`
			QuotaID     string                      `json:"quotaID"`
			TargetHour  int                         `json:"targetHour"`
			ExecuteAt   string                      `json:"executeAt"`
			Status      string                      `json:"status"`
			ResultsJSON []models.WithdrawResultJSON `json:"resultsDetail,omitempty"`
			Logs        []models.KuwoTaskLog        `json:"logs,omitempty"`
		}
		list := make([]taskJSON, 0, len(tasks))
		for _, t := range tasks {
			list = append(list, taskJSON{
				ID:          t.ID,
				Phone:       t.Phone,
				QuotaID:     t.QuotaID,
				TargetHour:  t.TargetHour,
				ExecuteAt:   t.ExecuteAt.Format("15:04"),
				Status:      t.Status,
				ResultsJSON: t.ResultsJSON,
				Logs:        t.LogsSnapshot(),
			})
		}
		c.Data["json"] = map[string]interface{}{"code": 0, "data": list}
		c.ServeJSON()
		return
	}

	task := models.KuwoGetScheduledTask(taskID)
	if task == nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": "任务不存在"}
		c.ServeJSON()
		return
	}
	c.Data["json"] = map[string]interface{}{
		"code": 0,
		"data": kuwoTaskStatusPayload(task),
	}
	c.ServeJSON()
}

func kuwoTaskStatusPayload(task *models.KuwoScheduledTask) map[string]interface{} {
	smsFatal := false
	if len(task.Results) > 0 {
		smsFatal = models.KuwoAllResultsSmsFatalExported(task.Results)
	}
	return map[string]interface{}{
		"id":            task.ID,
		"phone":         task.Phone,
		"quotaID":       task.QuotaID,
		"targetHour":    task.TargetHour,
		"executeAt":     task.ExecuteAt.Format("15:04:05"),
		"status":        task.Status,
		"immediate":     task.Immediate,
		"useProxy":      task.UseProxy,
		"smsEditable":   task.Status == "pending",
		"smsFatal":      smsFatal,
		"results":       task.ResultsJSON,
		"resultsDetail": task.ResultsJSON,
		"logs":          task.LogsSnapshot(),
	}
}

func (c *PortalController) portalCategory() models.Category {
	return c.ClientCtx.LogCategory()
}

func (c *PortalController) logPortalInfo(format string, args ...interface{}) {
	uid := c.PortalUserID
	msg := fmt.Sprintf(format, args...)
	if uid > 0 {
		msg = fmt.Sprintf("[用户%d] %s", uid, msg)
	}
	models.Logf(c.portalCategory(), models.LevelInfo, "%s", msg)
}

func (c *PortalController) logPortalWarn(format string, args ...interface{}) {
	uid := c.PortalUserID
	msg := fmt.Sprintf(format, args...)
	if uid > 0 {
		msg = fmt.Sprintf("[用户%d] %s", uid, msg)
	}
	models.Logf(c.portalCategory(), models.LevelWarn, "%s", msg)
}

func (c *PortalController) logPortalError(format string, args ...interface{}) {
	uid := c.PortalUserID
	msg := fmt.Sprintf(format, args...)
	if uid > 0 {
		msg = fmt.Sprintf("[用户%d] %s", uid, msg)
	}
	models.Logf(c.portalCategory(), models.LevelError, "%s", msg)
}
