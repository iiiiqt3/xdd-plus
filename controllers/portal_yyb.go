package controllers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/cdle/xdd/models"
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

// Index 重定向到主后台嵌入页
func (c *PortalYybController) Index() {
	c.Redirect("/portal?panel=yyb", 302)
}

// Status 模块与用户状态
func (c *PortalYybController) Status() {
	autoCheck := c.GetString("check") == "1" || strings.EqualFold(c.GetString("check"), "true")
	data, err := yybportal.PortalStatus(c.PortalUserID, autoCheck)
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
	var req struct {
		RegionCode string `json:"regionCode"`
		RegionName string `json:"regionName"`
		PackID     string `json:"packId"`
		UseProxy   bool   `json:"useProxy"`
	}
	_ = json.Unmarshal(c.Ctx.Input.RequestBody, &req)
	opt := models.YybProxyLoginOption{
		Enabled:    req.UseProxy || models.Config.Yyb.Proxy51Enabled,
		PackID:     strings.TrimSpace(req.PackID),
		RegionCode: strings.TrimSpace(req.RegionCode),
		RegionName: strings.TrimSpace(req.RegionName),
	}
	if opt.Enabled && opt.RegionCode == "" {
		c.jsonErr(fmt.Errorf("请选择登录地区（省/市）"))
		return
	}
	data, err := yybportal.PortalCreateQR(c.PortalUserID, opt)
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

// UpdateRemark 更新账号备注
func (c *PortalYybController) UpdateRemark() {
	var req struct {
		Ref    string `json:"ref"`
		Remark string `json:"remark"`
	}
	if err := json.Unmarshal(c.Ctx.Input.RequestBody, &req); err != nil || req.Ref == "" {
		c.jsonErr(errEmptyRef)
		return
	}
	data, err := yybportal.PortalUpdateRemark(c.PortalUserID, req.Ref, req.Remark)
	if err != nil {
		c.jsonErr(err)
		return
	}
	c.jsonOK(data, "备注已保存")
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
