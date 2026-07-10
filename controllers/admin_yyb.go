package controllers

import (
	"encoding/json"
	"net/http"

	"github.com/cdle/xdd/yybportal"
)

// AdminYybController 应用宝管理后台（独立页面）
type AdminYybController struct {
	BaseController
}

func (c *AdminYybController) NextPrepare() {
	c.Logined()
}

func (c *AdminYybController) jsonOK(data any, msg string) {
	if msg == "" {
		msg = "ok"
	}
	c.Data["json"] = map[string]interface{}{"code": 0, "data": data, "msg": msg}
	c.ServeJSON()
}

func (c *AdminYybController) jsonErr(err error) {
	c.Data["json"] = map[string]interface{}{"code": 1, "msg": err.Error()}
	c.ServeJSON()
}

// Index 重定向到主后台嵌入页
func (c *AdminYybController) Index() {
	c.Redirect("/admin?page=yyb", 302)
}

// Status 状态
func (c *AdminYybController) Status() {
	data := yybportal.StatusPayload()
	for k, v := range yybportal.AdminStatusExtras() {
		data[k] = v
	}
	data["config"] = yybportal.AdminConfigView()
	c.jsonOK(data, "查询成功")
}

// Config 配置
func (c *AdminYybController) Config() {
	c.jsonOK(yybportal.AdminConfigView(), "查询成功")
}

// Accounts 全站门户绑定
func (c *AdminYybController) Accounts() {
	data, err := yybportal.AdminListAccounts()
	if err != nil {
		c.jsonErr(err)
		return
	}
	c.jsonOK(data, "查询成功")
}

// ProtocolAccounts 协议库全部账号
func (c *AdminYybController) ProtocolAccounts() {
	data, err := yybportal.AdminListProtocolAccounts()
	if err != nil {
		c.jsonErr(err)
		return
	}
	c.jsonOK(data, "查询成功")
}

// Avatar 头像
func (c *AdminYybController) Avatar() {
	ref := c.GetString("ref")
	if ref == "" {
		c.Ctx.Output.SetStatus(http.StatusBadRequest)
		c.Ctx.WriteString("ref required")
		return
	}
	if err := yybportal.AdminServeAvatar(c.Ctx.ResponseWriter, c.Ctx.Request, ref); err != nil {
		c.Ctx.Output.SetStatus(http.StatusNotFound)
		c.Ctx.WriteString(err.Error())
	}
}

// DeleteAccount 删除
func (c *AdminYybController) DeleteAccount() {
	var req struct {
		Ref string `json:"ref"`
	}
	if err := json.Unmarshal(c.Ctx.Input.RequestBody, &req); err != nil || req.Ref == "" {
		c.jsonErr(errEmptyRef)
		return
	}
	if err := yybportal.AdminDeleteAccount(req.Ref); err != nil {
		c.jsonErr(err)
		return
	}
	c.jsonOK(nil, "删除成功")
}

// RefreshAccount 刷新
func (c *AdminYybController) RefreshAccount() {
	var req struct {
		Ref string `json:"ref"`
	}
	if err := json.Unmarshal(c.Ctx.Input.RequestBody, &req); err != nil || req.Ref == "" {
		c.jsonErr(errEmptyRef)
		return
	}
	data, err := yybportal.AdminRefreshAccount(req.Ref)
	if err != nil {
		c.jsonErr(err)
		return
	}
	c.jsonOK(data, "刷新完成")
}

// CheckAllAccounts 一键检测全部门户绑定
func (c *AdminYybController) CheckAllAccounts() {
	data, err := yybportal.AdminCheckAllBindings()
	if err != nil {
		c.jsonErr(err)
		return
	}
	c.jsonOK(data, "检测完成")
}

// WarmupProtocol 管理员进入后台时后台预检协议账号存活（立即返回）
func (c *AdminYybController) WarmupProtocol() {
	c.jsonOK(yybportal.AdminTriggerProtocolWarmup(), "ok")
}

// CheckAllProtocol 同步检测全部协议账号存活
func (c *AdminYybController) CheckAllProtocol() {
	data, err := yybportal.AdminCheckAllProtocolAccounts()
	if err != nil {
		c.jsonErr(err)
		return
	}
	c.jsonOK(data, "检测完成")
}

// ResyncAccount 同步
func (c *AdminYybController) ResyncAccount() {
	var req struct {
		Ref string `json:"ref"`
	}
	if err := json.Unmarshal(c.Ctx.Input.RequestBody, &req); err != nil || req.Ref == "" {
		c.jsonErr(errEmptyRef)
		return
	}
	data, err := yybportal.AdminResyncAccount(req.Ref)
	if err != nil {
		c.jsonErr(err)
		return
	}
	c.jsonOK(data, "同步完成")
}

// CreateQR 测试扫码
func (c *AdminYybController) CreateQR() {
	data, err := yybportal.AdminCreateQR()
	if err != nil {
		c.jsonErr(err)
		return
	}
	c.jsonOK(data, "二维码已生成")
}

// PollQR 轮询
func (c *AdminYybController) PollQR() {
	sessionID := c.Ctx.Input.Param(":id")
	data, err := yybportal.AdminPollQR(sessionID)
	if err != nil {
		c.jsonErr(err)
		return
	}
	c.jsonOK(data, "ok")
}

// ConfirmQR 确认
func (c *AdminYybController) ConfirmQR() {
	sessionID := c.Ctx.Input.Param(":id")
	data, err := yybportal.AdminConfirmQR(sessionID)
	if err != nil {
		c.jsonErr(err)
		return
	}
	c.jsonOK(data, "登录成功")
}

// WxappGetCode 调试
func (c *AdminYybController) WxappGetCode() {
	var req struct {
		Ref   string `json:"ref"`
		AppID string `json:"appId"`
	}
	if err := json.Unmarshal(c.Ctx.Input.RequestBody, &req); err != nil {
		c.jsonErr(err)
		return
	}
	data, err := yybportal.AdminWxappGetCode(req.Ref, req.AppID)
	if err != nil {
		c.jsonErr(err)
		return
	}
	c.jsonOK(data, "ok")
}

// WxappGetPhone 调试
func (c *AdminYybController) WxappGetPhone() {
	var req struct {
		Ref   string `json:"ref"`
		AppID string `json:"appId"`
	}
	if err := json.Unmarshal(c.Ctx.Input.RequestBody, &req); err != nil {
		c.jsonErr(err)
		return
	}
	data, err := yybportal.AdminWxappGetPhone(req.Ref, req.AppID)
	if err != nil {
		c.jsonErr(err)
		return
	}
	c.jsonOK(data, "ok")
}

// WxappOperate 调试
func (c *AdminYybController) WxappOperate() {
	var req struct {
		Ref     string         `json:"ref"`
		AppID   string         `json:"appId"`
		Payload map[string]any `json:"payload"`
	}
	if err := json.Unmarshal(c.Ctx.Input.RequestBody, &req); err != nil {
		c.jsonErr(err)
		return
	}
	data, err := yybportal.AdminWxappOperate(req.Ref, req.AppID, req.Payload)
	if err != nil {
		c.jsonErr(err)
		return
	}
	c.jsonOK(data, "ok")
}

// YybScriptController 青龙/脚本 API（Token 鉴权，与门户会话分离）
type YybScriptController struct {
	BaseController
}

func (c *YybScriptController) checkToken() bool {
	token := c.GetString("api_token")
	if token == "" {
		token = c.Ctx.Input.Header("Authorization")
		if len(token) > 7 && token[:7] == "Bearer " {
			token = token[7:]
		}
	}
	return yybportal.CheckScriptToken(token)
}

func (c *YybScriptController) deny() {
	c.Ctx.Output.SetStatus(http.StatusForbidden)
	c.Data["json"] = map[string]interface{}{"code": 403, "msg": "api_token 无效"}
	c.ServeJSON()
}

func (c *YybScriptController) jsonOK(data any) {
	c.Data["json"] = map[string]interface{}{"code": 0, "data": data}
	c.ServeJSON()
}

func (c *YybScriptController) jsonErr(err error) {
	c.Data["json"] = map[string]interface{}{"code": 1, "msg": err.Error()}
	c.ServeJSON()
}

// Accounts 脚本账号列表
func (c *YybScriptController) Accounts() {
	if !c.checkToken() {
		c.deny()
		return
	}
	data, err := yybportal.ScriptListAccounts()
	if err != nil {
		c.jsonErr(err)
		return
	}
	c.jsonOK(data)
}

// WxappGetCode 脚本 code
func (c *YybScriptController) WxappGetCode() {
	if !c.checkToken() {
		c.deny()
		return
	}
	var req struct {
		Ref   string `json:"ref"`
		AppID string `json:"appId"`
	}
	if err := json.Unmarshal(c.Ctx.Input.RequestBody, &req); err != nil {
		c.jsonErr(err)
		return
	}
	data, err := yybportal.ScriptWxappGetCode(req.Ref, req.AppID)
	if err != nil {
		c.jsonErr(err)
		return
	}
	c.jsonOK(data)
}

// WxappGetPhone 脚本手机号
func (c *YybScriptController) WxappGetPhone() {
	if !c.checkToken() {
		c.deny()
		return
	}
	var req struct {
		Ref   string `json:"ref"`
		AppID string `json:"appId"`
	}
	if err := json.Unmarshal(c.Ctx.Input.RequestBody, &req); err != nil {
		c.jsonErr(err)
		return
	}
	data, err := yybportal.ScriptWxappGetPhone(req.Ref, req.AppID)
	if err != nil {
		c.jsonErr(err)
		return
	}
	c.jsonOK(data)
}

// WxappOperate 脚本云函数
func (c *YybScriptController) WxappOperate() {
	if !c.checkToken() {
		c.deny()
		return
	}
	var req struct {
		Ref     string         `json:"ref"`
		AppID   string         `json:"appId"`
		Payload map[string]any `json:"payload"`
	}
	if err := json.Unmarshal(c.Ctx.Input.RequestBody, &req); err != nil {
		c.jsonErr(err)
		return
	}
	data, err := yybportal.ScriptWxappOperate(req.Ref, req.AppID, req.Payload)
	if err != nil {
		c.jsonErr(err)
		return
	}
	c.jsonOK(data)
}

// RefreshAccount 脚本刷新
func (c *YybScriptController) RefreshAccount() {
	if !c.checkToken() {
		c.deny()
		return
	}
	var req struct {
		Ref string `json:"ref"`
	}
	if err := json.Unmarshal(c.Ctx.Input.RequestBody, &req); err != nil {
		c.jsonErr(err)
		return
	}
	data, err := yybportal.ScriptRefreshAccount(req.Ref)
	if err != nil {
		c.jsonErr(err)
		return
	}
	c.jsonOK(data)
}
