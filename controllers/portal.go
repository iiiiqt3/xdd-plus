package controllers

import (
	"encoding/json"
	"strings"

	"github.com/cdle/xdd/models"
	"github.com/cdle/xdd/vweb"
)

// PortalController 门户控制器，处理用户门户相关的所有页面和API请求
type PortalController struct {
	BaseController
}

func (c *PortalController) requestSource() string {
	// 优先检查前端显式声明的来源
	if src := c.Ctx.Input.Header("X-Request-Source"); src != "" {
		if src == "web" {
			return "Web端"
		}
		if src == "app" {
			return "App端"
		}
	}
	ua := strings.ToLower(c.Ctx.Input.Header("User-Agent"))
	if strings.Contains(ua, "okhttp") || c.Ctx.Input.Header("X-Sign-DeviceID") != "" {
		return "App端"
	}
	return "Web端"
}

// NextPrepare 前置处理，验证门户用户登录状态
func (c *PortalController) NextPrepare() {
	c.PortalLogined()
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
	msg, err := models.PortalCreateProject(c.PortalUserID, req.ActivityID, req.Inputs, req.Remarks, req.Months, c.requestSource())
	if err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": err.Error()}
		c.ServeJSON()
		return
	}
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
	msg, err := models.PortalRenewProject(c.PortalUserID, req.ActivityID, req.Remarks, req.Months, c.requestSource())
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
	msg, err := models.PortalDeleteProject(c.PortalUserID, req.ActivityID, req.Remarks, c.requestSource())
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
	msg, balance, err := models.PortalRedeemKey(c.PortalUserID, req.Token, c.requestSource())
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

	msg, err := models.PortalCheckIn(c.PortalUserID, c.requestSource())
	if err != nil {
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": err.Error()}
		c.ServeJSON()
		return
	}
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

	msg, err := models.PortalPray(c.PortalUserID, c.requestSource())
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
	if err := models.CreateAppFeedback(c.PortalUserID, req.Type, title, content, contact, "app"); err != nil {
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
	data, err := models.PortalWxPollLogin(c.PortalUserID, req.UUID, req.DeductCoin)
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
	sourceFilter := c.GetString("source", "")

	type coinLogItem struct {
		ID           int    `json:"id"`
		Amount       int    `json:"amount"`
		BalanceAfter int    `json:"balanceAfter"`
		Type         string `json:"type"`
		Detail       string `json:"detail"`
		Source       string `json:"source"`
		CreatedAt    string `json:"createdAt"`
	}

	result := make([]coinLogItem, 0)
	for _, l := range logs {
		src := "后台及其他"
		d := l.Detail
		if len(d) >= 6 && d[:6] == "Web端" {
			src = "Web端"
			d = d[6:]
		} else if len(d) >= 6 && d[:6] == "App端" {
			src = "App端"
			d = d[6:]
		} else if strings.HasPrefix(d, "微信") {
			src = "微信"
		}
		if sourceFilter != "" && src != sourceFilter {
			continue
		}
		result = append(result, coinLogItem{
			ID:           l.ID,
			Amount:       l.Amount,
			BalanceAfter: l.BalanceAfter,
			Type:         l.Type,
			Detail:       d,
			Source:       src,
			CreatedAt:    l.CreatedAt.Format("2006-01-02 15:04"),
		})
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
		c.Data["json"] = map[string]interface{}{"code": 1, "msg": err.Error()}
		c.ServeJSON()
		return
	}
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
