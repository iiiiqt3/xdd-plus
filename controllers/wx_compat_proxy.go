package controllers

import (
	"encoding/json"
	"strings"

	"github.com/cdle/xdd/models"
)

// WxCompatProxyController 青龙脚本兼容网关：/api/v1/wx/* 由 xdd 分流到应用宝或 wechat08
type WxCompatProxyController struct {
	BaseController
}

func (c *WxCompatProxyController) NextPrepare() {}

func (c *WxCompatProxyController) Any() {
	path := c.Ctx.Request.URL.Path
	body := c.Ctx.Input.RequestBody
	rawQuery := c.Ctx.Request.URL.RawQuery
	ref := models.ExtractCompatRef(rawQuery, body)
	route := models.ResolveProtocolRoute(ref)
	action := compatGatewayAction(path)
	method := c.Ctx.Request.Method
	models.Yyb().Infof("[协议网关] 入站 %s %s action=%s ref=%s %s", method, path, action, ref, route.LogSummary())

	if !models.ShouldRouteToYyb(ref) {
		models.Yyb().Infof("[协议网关] 转发 → wechat08 %s %s ref=%s", method, path, ref)
		respBody, status, contentType, err := models.ForwardWxProtoRequest(method, path, rawQuery, body)
		if err != nil {
			models.Yyb().Warnf("[协议网关] wechat08 转发失败 %s %s ref=%s err=%v", method, path, ref, err)
			c.Ctx.Output.SetStatus(502)
			c.Ctx.Output.Body(models.BuildCompatErrorResponse("连接微信协议服务失败"))
			return
		}
		if contentType == "" {
			contentType = "application/json; charset=utf-8"
		}
		c.Ctx.Output.Header("Content-Type", contentType)
		c.Ctx.Output.SetStatus(status)
		c.Ctx.Output.Body(respBody)
		return
	}

	models.Yyb().Infof("[协议网关] 处理 → 应用宝 %s %s action=%s ref=%s", method, path, action, ref)
	resp, status := handleYybCompat(path, rawQuery, body, ref)
	c.Ctx.Output.Header("Content-Type", "application/json; charset=utf-8")
	c.Ctx.Output.SetStatus(status)
	c.Ctx.Output.Body(resp)
}

func compatGatewayAction(path string) string {
	switch {
	case strings.Contains(path, "/get/code") || strings.Contains(path, "GetCode") || strings.Contains(path, "JSLogin"):
		return "getCode"
	case strings.Contains(path, "/call/function") || strings.Contains(path, "CallFunction"):
		return "callFunction"
	case strings.Contains(path, "/operate/wxdata") || strings.Contains(path, "OperateWxData"):
		return "operateWxData"
	case strings.Contains(path, "/user/status") || strings.Contains(path, "UserStatus"):
		return "userStatus"
	default:
		return "other"
	}
}

func handleYybCompat(path, rawQuery string, body []byte, ref string) ([]byte, int) {
	switch {
	case strings.Contains(path, "/get/code") || strings.Contains(path, "GetCode") || strings.Contains(path, "JSLogin"):
		return handleCompatGetCode(body, ref)
	case strings.Contains(path, "/call/function") || strings.Contains(path, "CallFunction"):
		return handleCompatCallFunction(body, ref)
	case strings.Contains(path, "/operate/wxdata") || strings.Contains(path, "OperateWxData"):
		return handleCompatOperate(body, ref)
	case strings.Contains(path, "/user/status") || strings.Contains(path, "UserStatus"):
		return handleCompatStatus()
	default:
		models.Yyb().Infof("[协议网关] 应用宝未实现的路径，回退 wechat08 转发 path=%s ref=%s", path, ref)
		respBody, status, _, err := models.ForwardWxProtoRequest("POST", path, rawQuery, body)
		if err != nil {
			return models.BuildCompatErrorResponse("协议转发失败"), 502
		}
		return respBody, status
	}
}

func handleCompatGetCode(body []byte, ref string) ([]byte, int) {
	appID := extractCompatAppID(body)
	if appID == "" {
		appID = models.WxJdAppID
	}
	code, err := models.ProtocolGetWxAppCode(ref, appID)
	if err != nil {
		return models.BuildCompatErrorResponse(err.Error()), 200
	}
	return models.BuildCompatCodeResponse(code), 200
}

func handleCompatCallFunction(body []byte, ref string) ([]byte, int) {
	appID := extractCompatAppID(body)
	payload := extractCompatPayload(body)
	data, err := models.ProtocolCallWxFunction(ref, appID, payload)
	if err != nil {
		return models.BuildCompatErrorResponse(err.Error()), 200
	}
	b, _ := json.Marshal(data)
	return b, 200
}

func handleCompatOperate(body []byte, ref string) ([]byte, int) {
	appID := extractCompatAppID(body)
	payload := extractCompatPayload(body)
	data, err := models.ProtocolCallWxFunction(ref, appID, payload)
	if err != nil {
		return models.BuildCompatErrorResponse(err.Error()), 200
	}
	b, _ := json.Marshal(data)
	return b, 200
}

func handleCompatStatus() ([]byte, int) {
	respBody, status, _, err := models.ForwardWxProtoRequest("GET", "/api/v1/wx/user/status", "", nil)
	if err != nil {
		return models.BuildCompatErrorResponse("获取状态失败"), 502
	}
	return respBody, status
}

func extractCompatAppID(body []byte) string {
	m := parseCompatJSON(body)
	for _, k := range []string{"appid", "app_id", "Appid", "AppID"} {
		if v, ok := m[k].(string); ok && strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func extractCompatPayload(body []byte) map[string]interface{} {
	m := parseCompatJSON(body)
	if raw, ok := m["data"]; ok {
		switch v := raw.(type) {
		case map[string]interface{}:
			return v
		case string:
			out := map[string]interface{}{}
			_ = jsonUnmarshalCompat([]byte(v), &out)
			return out
		}
	}
	return m
}

func parseCompatJSON(body []byte) map[string]interface{} {
	out := map[string]interface{}{}
	if len(body) == 0 {
		return out
	}
	_ = json.Unmarshal(body, &out)
	return out
}

func jsonUnmarshalCompat(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}
