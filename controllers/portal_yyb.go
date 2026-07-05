package controllers

import (
	"encoding/json"
	"net/http"

	"github.com/cdle/xdd/models"
	"github.com/cdle/xdd/vweb"
	"github.com/cdle/xdd/yybportal"
)

// PortalYybController 应用宝门户（独立页面 + API）
type PortalYybController struct {
	BaseController
}

func (c *PortalYybController) NextPrepare() {
	c.PortalLogined()
	c.ClientCtx = models.ResolveClientContext(
		c.Ctx.Input.Header("X-Request-Source"),
		c.Ctx.Input.Header("User-Agent"),
		c.Ctx.Input.Header("X-Sign-DeviceID"),
		c.Ctx.Input.Header("X-Client-Platform"),
	)
}

func (c *PortalYybController) jsonOK(data any, msg string) {
	if msg == "" {
		msg = "ok"
	}
	c.Data["json"] = map[string]interface{}{"code": 0, "data": data, "msg": msg}
	c.ServeJSON()
}

func (c *PortalYybController) jsonErr(err error) {
	c.Data["json"] = map[string]interface{}{"code": 1, "msg": err.Error()}
	c.ServeJSON()
}

// Index 应用宝门户页面
func (c *PortalYybController) Index() {
	file, err := vweb.ReadFile("html/portal_yyb.html")
	if err != nil {
		c.Ctx.WriteString("portal yyb page not found")
		return
	}
	c.Ctx.WriteString(string(file))
}

// Status 模块与用户状态
func (c *PortalYybController) Status() {
	data, err := yybportal.PortalStatus(c.PortalUserID)
	if err != nil {
		c.jsonErr(err)
		return
	}
	c.jsonOK(data, "查询成功")
}

// Accounts 账号列表
func (c *PortalYybController) Accounts() {
	data, err := yybportal.PortalListAccounts(c.PortalUserID)
	if err != nil {
		c.jsonErr(err)
		return
	}
	c.jsonOK(data, "查询成功")
}

// CreateQR 创建扫码
func (c *PortalYybController) CreateQR() {
	data, err := yybportal.PortalCreateQR(c.PortalUserID)
	if err != nil {
		c.jsonErr(err)
		return
	}
	c.jsonOK(data, "二维码已生成")
}

// PollQR 轮询扫码
func (c *PortalYybController) PollQR() {
	sessionID := c.Ctx.Input.Param(":id")
	data, err := yybportal.PortalPollQR(c.PortalUserID, sessionID)
	if err != nil {
		c.jsonErr(err)
		return
	}
	c.jsonOK(data, "ok")
}

// ConfirmQR 确认扫码
func (c *PortalYybController) ConfirmQR() {
	sessionID := c.Ctx.Input.Param(":id")
	data, err := yybportal.PortalConfirmQR(c.PortalUserID, sessionID, c.ClientCtx.WithDefault())
	if err != nil {
		c.jsonErr(err)
		return
	}
	c.jsonOK(data, "登录成功")
}

// DeleteAccount 删除账号
func (c *PortalYybController) DeleteAccount() {
	var req struct {
		Ref string `json:"ref"`
	}
	if err := json.Unmarshal(c.Ctx.Input.RequestBody, &req); err != nil || req.Ref == "" {
		c.jsonErr(errEmptyRef)
		return
	}
	if err := yybportal.PortalDeleteAccount(c.PortalUserID, req.Ref); err != nil {
		c.jsonErr(err)
		return
	}
	c.jsonOK(nil, "删除成功")
}

// RefreshAccount 刷新存活
func (c *PortalYybController) RefreshAccount() {
	var req struct {
		Ref string `json:"ref"`
	}
	if err := json.Unmarshal(c.Ctx.Input.RequestBody, &req); err != nil || req.Ref == "" {
		c.jsonErr(errEmptyRef)
		return
	}
	data, err := yybportal.PortalRefreshAccount(c.PortalUserID, req.Ref)
	if err != nil {
		c.jsonErr(err)
		return
	}
	c.jsonOK(data, "刷新完成")
}

// ResyncAccount 同步资料
func (c *PortalYybController) ResyncAccount() {
	var req struct {
		Ref string `json:"ref"`
	}
	if err := json.Unmarshal(c.Ctx.Input.RequestBody, &req); err != nil || req.Ref == "" {
		c.jsonErr(errEmptyRef)
		return
	}
	data, err := yybportal.PortalResyncAccount(c.PortalUserID, req.Ref)
	if err != nil {
		c.jsonErr(err)
		return
	}
	c.jsonOK(data, "同步完成")
}

// ClaimAccount 认领应用宝库中已有账号到当前门户用户
func (c *PortalYybController) ClaimAccount() {
	var req struct {
		Ref string `json:"ref"`
	}
	if err := json.Unmarshal(c.Ctx.Input.RequestBody, &req); err != nil || req.Ref == "" {
		c.jsonErr(errEmptyRef)
		return
	}
	data, err := yybportal.PortalClaimAccount(c.PortalUserID, req.Ref)
	if err != nil {
		c.jsonErr(err)
		return
	}
	c.jsonOK(data, "认领成功")
}

// Avatar 头像
func (c *PortalYybController) Avatar() {
	ref := c.GetString("ref")
	if ref == "" {
		c.Ctx.Output.SetStatus(http.StatusBadRequest)
		return
	}
	if err := yybportal.PortalServeAvatar(c.Ctx.ResponseWriter, c.Ctx.Request, c.PortalUserID, ref); err != nil {
		c.Ctx.Output.SetStatus(http.StatusNotFound)
	}
}

// WxappGetCode 小程序 code
func (c *PortalYybController) WxappGetCode() {
	var req struct {
		Ref   string `json:"ref"`
		AppID string `json:"appId"`
	}
	if err := json.Unmarshal(c.Ctx.Input.RequestBody, &req); err != nil {
		c.jsonErr(err)
		return
	}
	data, err := yybportal.PortalWxappGetCode(c.PortalUserID, req.Ref, req.AppID)
	if err != nil {
		c.jsonErr(err)
		return
	}
	c.jsonOK(data, "ok")
}

// WxappGetPhone 手机号
func (c *PortalYybController) WxappGetPhone() {
	var req struct {
		Ref   string `json:"ref"`
		AppID string `json:"appId"`
	}
	if err := json.Unmarshal(c.Ctx.Input.RequestBody, &req); err != nil {
		c.jsonErr(err)
		return
	}
	data, err := yybportal.PortalWxappGetPhone(c.PortalUserID, req.Ref, req.AppID)
	if err != nil {
		c.jsonErr(err)
		return
	}
	c.jsonOK(data, "ok")
}

// WxappOperate 云函数
func (c *PortalYybController) WxappOperate() {
	var req struct {
		Ref     string         `json:"ref"`
		AppID   string         `json:"appId"`
		Payload map[string]any `json:"payload"`
	}
	if err := json.Unmarshal(c.Ctx.Input.RequestBody, &req); err != nil {
		c.jsonErr(err)
		return
	}
	data, err := yybportal.PortalWxappOperate(c.PortalUserID, req.Ref, req.AppID, req.Payload)
	if err != nil {
		c.jsonErr(err)
		return
	}
	c.jsonOK(data, "ok")
}

var errEmptyRef = &jsonRefError{}

type jsonRefError struct{}

func (e *jsonRefError) Error() string { return "ref 不能为空" }
