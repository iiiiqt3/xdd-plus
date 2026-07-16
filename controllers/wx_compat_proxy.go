package controllers

import (
	"encoding/json"
	"strings"

	"github.com/cdle/xdd/models"
	"github.com/cdle/xdd/yyb"
	"github.com/cdle/xdd/yybportal"
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
	resp, status := handleYybCompat(path, rawQuery, body, ref, method)
	c.Ctx.Output.Header("Content-Type", "application/json; charset=utf-8")
	c.Ctx.Output.SetStatus(status)
	c.Ctx.Output.Body(resp)
}

func handleYybCompat(path, rawQuery string, body []byte, ref, method string) ([]byte, int) {
	switch compatPathKind(path) {
	case compatKindStatus:
		return handleCompatStatus()
	case compatKindLatestUserKey:
		return handleCompatLatestUserKey(body, ref)
	case compatKindWxOAuth:
		return compatYybUnavailable(path, rawQuery, body, ref, method, "公众号 OAuth，需 official/cgi")
	case compatKindDelete:
		return handleCompatDelete(body, ref)
	case compatKindGetCode:
		return handleCompatGetCode(body, ref)
	case compatKindSessionID:
		return compatYybUnavailable(path, rawQuery, body, ref, method, "runtimeSession，内嵌协议未实现")
	case compatKindGetPhone:
		return handleCompatGetPhone(body, ref)
	case compatKindGetOpenID:
		return handleCompatGetOpenID(body, ref)
	case compatKindGetUserInfo:
		return handleCompatGetUserInfo(path, body, ref)
	case compatKindCallFunction:
		return handleCompatCallFunction(path, body, ref)
	case compatKindOperateWxData:
		return handleCompatOperate(path, body, ref)
	case compatKindRefresh:
		return handleCompatRefresh(body, ref)
	case compatKindTools:
		return compatYybUnavailable(path, rawQuery, body, ref, method, "步数/微信运动，需 tools/*")
	case compatKindOfficial:
		return compatYybUnavailable(path, rawQuery, body, ref, method, "公众号 CGI，需 official/cgi")
	case compatKindTenPay:
		return compatYybUnavailable(path, rawQuery, body, ref, method, "TenPay CGI，内嵌协议未实现")
	case compatKindLoginMisc:
		return compatYybUnavailable(path, rawQuery, body, ref, method, "设备登录类接口，应用宝无对应能力")
	case compatKindWxappMisc:
		return compatYybUnavailable(path, rawQuery, body, ref, method, "小程序记录/头像管理，应用宝无对应能力")
	default:
		return compatYybUnavailable(path, rawQuery, body, ref, method, "")
	}
}

func compatYybUnavailable(path, rawQuery string, body []byte, ref, method, hint string) ([]byte, int) {
	if resp, status, ok := tryWechat08Fallback(path, rawQuery, body, ref, method); ok {
		models.Yyb().Infof("[协议网关] 应用宝不支持，双绑回退 wechat08 path=%s ref=%s", path, ref)
		return resp, status
	}
	return models.BuildCompatUnsupportedResponse(path, hint), 200
}

func tryWechat08Fallback(path, rawQuery string, body []byte, ref, method string) ([]byte, int, bool) {
	route := models.ResolveProtocolRoute(ref)
	if !route.FromBind || strings.TrimSpace(route.WxWxid) == "" {
		return nil, 0, false
	}
	online, _ := models.CheckWxDeviceOnline(route.WxWxid)
	if !online {
		return nil, 0, false
	}
	if method == "" {
		method = "POST"
	}
	respBody, status, _, err := models.ForwardWxProtoRequest(method, path, rawQuery, body)
	if err != nil {
		return nil, 0, false
	}
	return respBody, status, true
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

func handleCompatGetPhone(body []byte, ref string) ([]byte, int) {
	appID := extractCompatAppID(body)
	if appID == "" {
		appID = models.WxJdAppID
	}
	data, err := models.ProtocolGetWxAppPhone(ref, appID)
	if err != nil {
		return models.BuildCompatErrorResponse(err.Error()), 200
	}
	return models.BuildCompatGetAllMobileResponse(data), 200
}

func handleCompatCallFunction(path string, body []byte, ref string) ([]byte, int) {
	appID := extractCompatAppID(body)
	if appID == "" {
		appID = models.WxJdAppID
	}
	root := parseCompatJSON(body)
	inner := extractCompatInnerPayload(root)
	apiName := compatFirstNonEmpty(inner, "api_name")
	if apiName != "" && apiName != "callFunction" {
		data, err := models.ProtocolCallWxFunction(ref, appID, inner)
		if err != nil {
			return models.BuildCompatErrorResponse(err.Error()), 200
		}
		return models.WrapCompatWxappResponse(apiName, data), 200
	}
	payload := buildCloudFunctionOperatePayload(root, inner)
	data, err := models.ProtocolCallWxFunction(ref, appID, payload)
	if err != nil {
		return models.BuildCompatErrorResponse(err.Error()), 200
	}
	return models.WrapCompatWxappResponse("callFunction", data), 200
}

func handleCompatOperate(path string, body []byte, ref string) ([]byte, int) {
	appID := extractCompatAppID(body)
	if appID == "" {
		appID = models.WxJdAppID
	}
	root := parseCompatJSON(body)
	payload := extractCompatOperatePayload(root)
	apiName := models.ExtractCompatAPINameFromPayload(root)
	if apiName == "" {
		apiName = compatFirstNonEmpty(payload, "api_name")
	}
	data, err := models.ProtocolCallWxFunction(ref, appID, payload)
	if err != nil {
		return models.BuildCompatErrorResponse(err.Error()), 200
	}
	return models.WrapCompatWxappResponse(apiName, data), 200
}

func handleCompatLatestUserKey(body []byte, ref string) ([]byte, int) {
	root := parseCompatJSON(body)
	appID := extractCompatAppID(body)
	if appID == "" {
		appID = models.WxJdAppID
	}
	payload := map[string]interface{}{
		"api_name": "webapi_getuserencryptkey",
		"data":     map[string]string{"lang": "zh_CN"},
	}
	data, err := models.ProtocolCallWxFunction(ref, appID, payload)
	if err != nil {
		return models.BuildCompatErrorResponse(err.Error()), 200
	}
	_ = root
	return models.BuildCompatEncryptKeyResponse(data), 200
}

func handleCompatGetUserInfo(path string, body []byte, ref string) ([]byte, int) {
	root := parseCompatJSON(body)
	if _, ok := root["data"]; !ok {
		root["data"] = `{"api_name":"webapi_getuserinfo","data":{"lang":"zh_CN"},"with_credentials":true,"from_component":true}`
	}
	b, _ := json.Marshal(root)
	return handleCompatCallFunction(path, b, ref)
}

func handleCompatGetOpenID(body []byte, ref string) ([]byte, int) {
	if ref == "" {
		ref = models.ExtractCompatRef("", body)
	}
	if ref == "" {
		return models.BuildCompatErrorResponse("缺少 wxid/ref"), 200
	}
	acc, err := yybportal.AccountPublic(ref)
	if err != nil || acc == nil {
		return models.BuildCompatErrorResponse("应用宝账号不存在：" + ref), 200
	}
	nick, avatar := "", ""
	if acc.Nickname != nil {
		nick = *acc.Nickname
	}
	if acc.Avatar != nil {
		avatar = *acc.Avatar
	}
	payload := map[string]interface{}{
		"Code":    0,
		"Success": true,
		"Message": "成功",
		"Data": map[string]interface{}{
			"Openid":     acc.OpenID,
			"openid":     acc.OpenID,
			"NickName":   nick,
			"nickname":   nick,
			"HeadImgUrl": avatar,
			"Sign":       "",
		},
		"Data62": "",
		"Debug":  "",
	}
	b, _ := json.Marshal(payload)
	return b, 200
}

func handleCompatRefresh(body []byte, ref string) ([]byte, int) {
	if ref == "" {
		ref = models.ExtractCompatRef("", body)
	}
	if ref == "" {
		return models.BuildCompatErrorResponse("缺少 wxid/ref"), 200
	}
	data, err := yybportal.ScriptRefreshAccount(ref)
	if err != nil {
		return models.BuildCompatErrorResponse(err.Error()), 200
	}
	return models.BuildCompatSuccessResponse(data), 200
}

func handleCompatDelete(body []byte, ref string) ([]byte, int) {
	if ref == "" {
		ref = models.ExtractCompatRef("", body)
	}
	if ref == "" {
		return models.BuildCompatErrorResponse("缺少 wxid/ref"), 200
	}
	if err := yybportal.ScriptDeleteAccount(ref); err != nil {
		return models.BuildCompatErrorResponse(err.Error()), 200
	}
	return models.BuildCompatSuccessResponse(map[string]interface{}{"ref": ref}), 200
}

func handleCompatStatus() ([]byte, int) {
	merged := map[string]interface{}{}
	if oldBody, status, _, err := models.ForwardWxProtoRequest("GET", "/api/v1/wx/user/status", "", nil); err == nil && status >= 200 && status < 300 {
		var oldResp struct {
			Data map[string]interface{} `json:"data"`
		}
		if json.Unmarshal(oldBody, &oldResp) == nil {
			for k, v := range oldResp.Data {
				merged[k] = v
			}
		}
	}
	accounts, err := yybportal.ScriptListAccounts()
	if err != nil {
		if len(merged) > 0 {
			b, _ := json.Marshal(map[string]interface{}{
				"status": true, "message": "成功(应用宝账号获取失败：" + err.Error() + ")", "data": merged,
			})
			return b, 200
		}
		return models.BuildCompatErrorResponse("获取应用宝账号失败：" + err.Error()), 200
	}
	for _, a := range toAccountPublicSlice(accounts) {
		ref := a.OpenID
		survival := yybStatusSurvival(a.Status)
		merged[ref] = map[string]interface{}{
			"Wxid":         ref,
			"wxid":         ref,
			"NickName":     derefString(a.Nickname),
			"nickname":     derefString(a.Nickname),
			"Device":       "应用宝协议",
			"device":       "应用宝协议",
			"Survival":     survival,
			"survival":     survival,
			"proto_source": "应用宝协议",
		}
	}
	b, _ := json.Marshal(map[string]interface{}{"status": true, "message": "成功", "data": merged})
	return b, 200
}

func toAccountPublicSlice(v any) []yyb.AccountPublic {
	switch rows := v.(type) {
	case []yyb.AccountPublic:
		return rows
	case []interface{}:
		out := make([]yyb.AccountPublic, 0, len(rows))
		for _, item := range rows {
			if m, ok := item.(map[string]interface{}); ok {
				out = append(out, yyb.AccountPublic{OpenID: compatFirstNonEmpty(m, "openid", "OpenID")})
			}
		}
		return out
	default:
		return nil
	}
}

func yybStatusSurvival(status *string) bool {
	if status == nil {
		return false
	}
	st := strings.ToLower(strings.TrimSpace(*status))
	return st == "alive" || st == "online"
}

func derefString(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func buildCloudFunctionOperatePayload(root, inner map[string]interface{}) map[string]interface{} {
	args, _ := inner["data"].(map[string]interface{})
	if args == nil {
		if rawArgs, ok := root["args"]; ok {
			if m, ok := rawArgs.(map[string]interface{}); ok {
				args = m
			}
		}
	}
	if args == nil {
		args = map[string]interface{}{}
	}
	fnName := compatFirstNonEmpty(root, "function_name", "functionName", "name")
	if fnName == "" {
		if dataMap, ok := inner["data"].(map[string]interface{}); ok {
			fnName = compatFirstNonEmpty(dataMap, "name")
		}
	}
	apiName := compatFirstNonEmpty(root, "api_name")
	if apiName == "" {
		apiName = "callFunction"
	}
	return map[string]interface{}{
		"api_name": apiName,
		"data": map[string]interface{}{
			"name": fnName,
			"data": args,
		},
	}
}

func extractCompatOperatePayload(root map[string]interface{}) map[string]interface{} {
	for _, key := range []string{"payload", "Payload", "data", "Data"} {
		if v, ok := root[key]; ok {
			switch t := v.(type) {
			case map[string]interface{}:
				return t
			case string:
				if strings.TrimSpace(t) == "" {
					continue
				}
				out := map[string]interface{}{}
				if json.Unmarshal([]byte(t), &out) == nil {
					return out
				}
			}
		}
	}
	return extractCompatInnerPayload(root)
}

func extractCompatInnerPayload(root map[string]interface{}) map[string]interface{} {
	for _, key := range []string{"data", "Data"} {
		if raw, ok := root[key]; ok {
			switch v := raw.(type) {
			case map[string]interface{}:
				return v
			case string:
				out := map[string]interface{}{}
				if strings.TrimSpace(v) != "" && json.Unmarshal([]byte(v), &out) == nil {
					return out
				}
			}
		}
	}
	return root
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

func parseCompatJSON(body []byte) map[string]interface{} {
	out := map[string]interface{}{}
	if len(body) == 0 {
		return out
	}
	_ = json.Unmarshal(body, &out)
	return out
}

func compatFirstNonEmpty(m map[string]interface{}, keys ...string) string {
	for _, k := range keys {
		if v, ok := m[k]; ok {
			if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
				return strings.TrimSpace(s)
			}
		}
	}
	return ""
}
