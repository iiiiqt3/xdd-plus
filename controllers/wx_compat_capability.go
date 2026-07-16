package controllers

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/cdle/xdd/models"
	"github.com/cdle/xdd/yybportal"
)

func compatYybRef(ref string) string {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return ""
	}
	route := models.ResolveProtocolRoute(ref)
	if strings.TrimSpace(route.OpenID) != "" {
		return strings.TrimSpace(route.OpenID)
	}
	return strings.TrimPrefix(ref, models.YybOpenIDPrefix)
}

func handleCompatWxOAuth(rawQuery string, body []byte) ([]byte, int) {
	payload := map[string]any{}
	if len(body) > 0 {
		_ = json.Unmarshal(body, &payload)
	}
	if rawQuery != "" {
		if vals, err := url.ParseQuery(rawQuery); err == nil {
			for key, values := range vals {
				if _, exists := payload[key]; !exists && len(values) > 0 && strings.TrimSpace(values[0]) != "" {
					payload[key] = strings.TrimSpace(values[0])
				}
			}
		}
	}
	ref := capFirstString(payload, "ref", "wxid", "Wxid", "openid", "OpenID")
	if ref == "" {
		ref = models.ExtractCompatRef(rawQuery, body)
	}
	if ref == "" {
		return models.BuildCompatErrorResponse("缺少 openid/ref"), 200
	}
	appid := capFirstString(payload, "appid", "app_id", "Appid", "AppID")
	if appid == "" {
		return models.BuildCompatErrorResponse("缺少 appid"), 200
	}
	req := map[string]any{
		"app_id":    appid,
		"cgi_url":   "/cgi-bin/mmbiz-bin/oauth_authorize",
		"cmd_id":    1254,
		"body_kind": "oauth-authorize",
		"payload":   payload,
	}
	data, err := yybportal.InternalOfficialCGIMap(compatYybRef(ref), req)
	if err != nil {
		return models.BuildCompatErrorResponse("请求应用宝 OAuth 失败：" + err.Error()), 200
	}
	return wrapCompatOfficialResponse(data), 200
}

func handleCompatSessionID(path, rawQuery string, body []byte) ([]byte, int) {
	payload := map[string]any{}
	if len(body) > 0 {
		_ = json.Unmarshal(body, &payload)
	}
	ref := capFirstString(payload, "ref", "wxid", "Wxid", "openid", "OpenID")
	if ref == "" {
		ref = models.ExtractCompatRef(rawQuery, body)
	}
	appid := capFirstString(payload, "app_id", "appid", "Appid", "AppID")
	if ref == "" {
		return models.BuildCompatErrorResponse("缺少 wxid/ref"), 200
	}
	if appid == "" {
		return models.BuildCompatErrorResponse("缺少 appid"), 200
	}
	data, err := yybportal.InternalRuntimeSession(compatYybRef(ref), appid, payload)
	if err != nil {
		return models.BuildCompatErrorResponse("请求应用宝失败：" + err.Error()), 200
	}
	return wrapCompatSessionIDResponse(data), 200
}

func handleCompatOfficial(path, rawQuery string, body []byte) ([]byte, int) {
	payload := map[string]any{}
	if len(body) > 0 {
		_ = json.Unmarshal(body, &payload)
	}
	ref := capFirstString(payload, "ref", "wxid", "Wxid", "openid", "OpenID")
	if ref == "" {
		ref = models.ExtractCompatRef(rawQuery, body)
	}
	if ref == "" {
		return models.BuildCompatErrorResponse("缺少 wxid/ref"), 200
	}
	req := map[string]any{
		"app_id":  capFirstString(payload, "app_id", "appid", "Appid", "AppID"),
		"payload": payload,
	}
	switch path {
	case "/api/Tools/GetA8Key", "/api/v1/wx/offical/get/a8key", "/api/v1/wx/c":
		req["cgi_url"] = "/cgi-bin/micromsg-bin/geta8key"
		req["cmd_id"] = 233
		req["body_kind"] = "geta8key"
		req["cookie_base64"] = capFirstString(payload, "CookieBase64", "cookie_base64")
	case "/api/OfficialAccounts/MpGetA8Key", "/api/OfficialAccounts/GetAppMsgExt", "/api/OfficialAccounts/GetAppMsgExtLike":
		req["cgi_url"] = "/cgi-bin/micromsg-bin/mp-geta8key"
		req["cmd_id"] = 238
		req["body_kind"] = "geta8key"
	case "/api/OfficialAccounts/OauthAuthorize":
		req["cgi_url"] = "/cgi-bin/mmbiz-bin/oauth_authorize"
		req["cmd_id"] = 1254
		req["body_kind"] = "oauth-authorize"
	case "/api/OfficialAccounts/JSAPIPreVerify":
		req["cgi_url"] = "/cgi-bin/mmbiz-bin/jsapi-preverify"
		req["cmd_id"] = 1093
		req["body_kind"] = "json-field3"
	case "/api/OfficialAccounts/Follow":
		req["cgi_url"] = "/cgi-bin/micromsg-bin/verifyuser"
		req["cmd_id"] = 137
		req["body_kind"] = "json-field3"
	case "/api/OfficialAccounts/Quit":
		req["cgi_url"] = "/cgi-bin/micromsg-bin/oplog"
		req["cmd_id"] = 681
		req["body_kind"] = "json-field3"
	default:
		return models.BuildCompatUnsupportedResponse(path, ""), 200
	}
	data, err := yybportal.InternalOfficialCGIMap(compatYybRef(ref), req)
	if err != nil {
		return models.BuildCompatErrorResponse("请求应用宝失败：" + err.Error()), 200
	}
	return wrapCompatOfficialResponse(data), 200
}

func handleCompatTenPay(path, rawQuery string, body []byte) ([]byte, int) {
	payload := map[string]any{}
	if len(body) > 0 {
		_ = json.Unmarshal(body, &payload)
	}
	ref := capFirstString(payload, "ref", "wxid", "Wxid", "openid", "OpenID")
	if ref == "" {
		ref = models.ExtractCompatRef(rawQuery, body)
	}
	if ref == "" {
		return models.BuildCompatErrorResponse("缺少 wxid/ref"), 200
	}
	p := payload
	if p == nil {
		p = map[string]any{}
	}
	req := map[string]any{
		"cgi_cmd":     85,
		"req_text":    capFirstString(payload, "req_text", "ReqText", "tenpay_url", "TenpayUrl"),
		"req_text_wx": capFirstString(payload, "req_text_wx", "ReqTextWx"),
		"raw_base64":  capFirstString(payload, "raw_base64"),
		"payload":     p,
	}
	data, err := yybportal.InternalTenPayCGIMap(compatYybRef(ref), req)
	if err != nil {
		return models.BuildCompatErrorResponse("请求应用宝 TenPay 失败：" + err.Error()), 200
	}
	return wrapCompatTenPayResponse(data), 200
}

func handleCompatStep(path, rawQuery string, body []byte) ([]byte, int) {
	payload := map[string]any{}
	if len(body) > 0 {
		_ = json.Unmarshal(body, &payload)
	}
	ref := capFirstString(payload, "ref", "wxid", "Wxid", "openid", "OpenID")
	if ref == "" {
		ref = models.ExtractCompatRef(rawQuery, body)
	}
	if ref == "" {
		return models.BuildCompatErrorResponse("missing wxid/ref"), 200
	}
	payload["ref"] = compatYybRef(ref)
	var data map[string]any
	var err error
	switch path {
	case "/api/v1/wx/tools/report/motion", "/api/User/ReportMotion":
		data, err = yybportal.InternalReportMotionMap(compatYybRef(ref), payload)
	case "/api/v1/wx/tools/get/werun", "/api/Tools/GetWeRunData":
		appID := capFirstString(payload, "app_id", "appid", "Appid", "AppID")
		if appID == "" {
			appID = models.WxJdAppID
		}
		data, err = yybportal.InternalGetWeRunData(compatYybRef(ref), appID)
	case "/api/Tools/GetBoundHardDevices":
		data, err = yybportal.InternalGetBoundHardDevicesMap(compatYybRef(ref), payload)
	default:
		data, err = yybportal.InternalUpdateStepMap(compatYybRef(ref), payload)
	}
	if err != nil {
		return models.BuildCompatErrorResponse(err.Error()), 200
	}
	return wrapCompatCapabilityResult(path, data), 200
}

func wrapCompatOfficialResponse(data map[string]any) []byte {
	if msg := compatBackendResponseError(data); msg != "" {
		return models.BuildCompatErrorResponse(msg)
	}
	result := data["result"]
	if result == nil {
		result = data
	}
	b, _ := json.Marshal(map[string]any{
		"Code": 0, "Success": true, "Message": "成功", "Data": result, "Data62": "", "Debug": "应用宝 official/cgi",
	})
	return b
}

func wrapCompatTenPayResponse(data map[string]any) []byte {
	if msg := compatBackendResponseError(data); msg != "" {
		return models.BuildCompatErrorResponse(msg)
	}
	result := data["result"]
	if result == nil {
		result = data
	}
	b, _ := json.Marshal(map[string]any{
		"Code": 0, "Success": true, "Message": "成功", "Data": result, "Data62": "", "Debug": "应用宝 TenPay CGI",
	})
	return b
}

func wrapCompatSessionIDResponse(data map[string]any) []byte {
	if msg := compatBackendResponseError(data); msg != "" {
		return models.BuildCompatErrorResponse(msg)
	}
	session := findLikelySessionString(data["result"])
	if session == "" {
		session = findLikelySessionString(data)
	}
	if session == "" {
		raw, _ := json.Marshal(data)
		return models.BuildCompatErrorResponse("应用宝未返回可用 sessionid/session_key：" + string(raw))
	}
	inner := map[string]any{
		"sessionid": session, "Sessionid": session, "sessionId": session,
		"session_key": session, "sessionKey": session, "SessionKey": session, "value": session,
	}
	b, _ := json.Marshal(map[string]any{
		"Code": 0, "Success": true, "Message": "成功", "Data": inner, "data": inner,
		"Data62": "", "Debug": "应用宝 runtimeSession",
	})
	return b
}

func wrapCompatCapabilityResult(path string, data map[string]any) []byte {
	if msg := compatBackendResponseError(data); msg != "" {
		return models.BuildCompatErrorResponse(msg)
	}
	result := data["result"]
	if result == nil {
		result = data
	}
	b, _ := json.Marshal(map[string]any{
		"Code": 0, "Success": true, "Message": "成功", "Data": result, "Data62": "", "Debug": path,
	})
	return b
}

func compatBackendResponseError(data map[string]any) string {
	if data == nil {
		return ""
	}
	if numericNonZero(data["code"]) || numericNonZero(data["Code"]) {
		msg := capFirstString(data, "message", "Message", "msg", "Msg")
		if msg == "" {
			raw, _ := json.Marshal(data)
			msg = string(raw)
		}
		return msg
	}
	for _, key := range []string{"success", "Success", "status"} {
		if v, ok := data[key].(bool); ok && !v {
			return capFirstString(data, "message", "Message", "msg", "Msg")
		}
	}
	return ""
}

func findLikelySessionString(v any) string {
	keys := map[string]bool{
		"sessionid": true, "Sessionid": true, "sessionId": true,
		"session_key": true, "sessionKey": true, "SessionKey": true,
		"runtimeSessionId": true, "RuntimeSessionId": true, "value": true,
	}
	var fallback string
	var walk func(any)
	walk = func(x any) {
		if fallback != "" {
			return
		}
		switch t := x.(type) {
		case map[string]any:
			for k, v := range t {
				if keys[k] {
					if s := strings.TrimSpace(fmt.Sprint(v)); s != "" && s != "<nil>" {
						fallback = s
						return
					}
				}
			}
			for _, v := range t {
				walk(v)
			}
		case []any:
			for _, v := range t {
				walk(v)
			}
		case string:
			s := strings.TrimSpace(t)
			if len(s) >= 16 && len(s) <= 256 {
				fallback = s
			}
		}
	}
	walk(v)
	return fallback
}

func capFirstString(m map[string]any, keys ...string) string {
	for _, k := range keys {
		if v, ok := m[k].(string); ok && strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func numericNonZero(v any) bool {
	switch x := v.(type) {
	case int:
		return x != 0
	case int64:
		return x != 0
	case float64:
		return x != 0
	case json.Number:
		n, _ := x.Int64()
		return n != 0
	case string:
		return strings.TrimSpace(x) != "" && strings.TrimSpace(x) != "0"
	default:
		return false
	}
}
