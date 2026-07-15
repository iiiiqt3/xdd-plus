package models

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// 由 yybportal 在启动时注入，避免 models ↔ yybportal 循环依赖
var (
	protocolYybGetCode    func(openid, appID string) (map[string]interface{}, error)
	protocolYybOperate    func(openid, appID string, payload map[string]interface{}) (map[string]interface{}, error)
	protocolYybIsAlive    func(openid string) bool
	protocolYybRefresh    func(openid string) (string, error)
)

func SetProtocolYybHandlers(
	getCode func(openid, appID string) (map[string]interface{}, error),
	operate func(openid, appID string, payload map[string]interface{}) (map[string]interface{}, error),
	isAlive func(openid string) bool,
	refresh func(openid string) (string, error),
) {
	protocolYybGetCode = getCode
	protocolYybOperate = operate
	protocolYybIsAlive = isAlive
	protocolYybRefresh = refresh
}

func ProtocolYybAccountAlive(openid string) bool {
	if protocolYybIsAlive == nil {
		return false
	}
	return protocolYybIsAlive(strings.TrimSpace(openid))
}

func protocolYybGetCodeWithRetry(route ProtocolRoute, appID string) (map[string]interface{}, error) {
	if protocolYybGetCode == nil {
		return nil, fmt.Errorf("应用宝模块未就绪")
	}
	openid := strings.TrimSpace(route.OpenID)
	if openid == "" {
		return nil, fmt.Errorf("缺少应用宝 openid")
	}
	data, err := protocolYybGetCode(openid, appID)
	if err == nil {
		return data, nil
	}
	if !route.FromBind || strings.TrimSpace(route.WxWxid) == "" {
		return nil, err
	}
	if protocolYybRefresh != nil {
		_, _ = protocolYybRefresh(openid)
		if data2, err2 := protocolYybGetCode(openid, appID); err2 == nil {
			return data2, nil
		}
	}
	return nil, err
}

// ProtocolGetWxAppCode 统一取小程序 code：优先应用宝，绑定账号掉线可回退微信（选项 A）
func ProtocolGetWxAppCode(ref, appID string) (code string, err error) {
	ref = strings.TrimSpace(ref)
	appID = strings.TrimSpace(appID)
	if ref == "" {
		return "", fmt.Errorf("缺少账号标识")
	}
	if appID == "" {
		appID = WxJdAppID
	}
	route := ResolveProtocolRoute(ref)
	if route.Backend == "yyb" {
		data, yybErr := protocolYybGetCodeWithRetry(route, appID)
		if yybErr == nil {
			if c := extractCompatCode(data); c != "" {
				Yyb().Infof("[协议路由] getCode → 应用宝 成功 %s appid=%s", route.LogSummary(), appID)
				return c, nil
			}
			yybErr = fmt.Errorf("应用宝未返回 code")
		}
		if route.FromBind && strings.TrimSpace(route.WxWxid) != "" {
			if online, _ := checkWxDeviceOnline(route.WxWxid); online {
				Yyb().Warnf("[协议路由] getCode 应用宝失败，尝试 wechat08 回退 %s err=%v", route.LogSummary(), yybErr)
				code, err := protocolGetWxCodeViaWechat(route.WxWxid, appID)
				if err == nil {
					Yyb().Infof("[协议路由] getCode → wechat08回退 成功 wxid=%s appid=%s", protocolRefShort(route.WxWxid), appID)
				} else {
					Yyb().Warnf("[协议路由] getCode → wechat08回退 失败 wxid=%s err=%v", protocolRefShort(route.WxWxid), err)
				}
				return code, err
			}
		}
		Yyb().Warnf("[协议路由] getCode → 应用宝 失败 %s appid=%s err=%v", route.LogSummary(), appID, yybErr)
		return "", yybErr
	}
	Yyb().Infof("[协议路由] getCode → wechat08 %s appid=%s", route.LogSummary(), appID)
	return protocolGetWxCodeViaWechat(ref, appID)
}

func protocolGetWxCodeViaWechat(wxid, appID string) (string, error) {
	if strings.TrimSpace(wxid) == "" {
		return "", fmt.Errorf("缺少微信ID")
	}
	base := getWxJdServerForDevice(wxid)
	return wxJdGetWxCodeFromURL(base, wxid)
}

// ProtocolCallWxFunction 统一云函数 / operate
func ProtocolCallWxFunction(ref, appID string, payload map[string]interface{}) (map[string]interface{}, error) {
	ref = strings.TrimSpace(ref)
	if appID == "" {
		appID = WxJdAppID
	}
	route := ResolveProtocolRoute(ref)
	if route.Backend == "yyb" {
		if protocolYybOperate == nil {
			return nil, fmt.Errorf("应用宝模块未就绪")
		}
		data, err := protocolYybOperate(route.OpenID, appID, payload)
		if err == nil {
			Yyb().Infof("[协议路由] callFunction → 应用宝 成功 %s appid=%s", route.LogSummary(), appID)
			return data, nil
		}
		if route.FromBind && strings.TrimSpace(route.WxWxid) != "" {
			if online, _ := checkWxDeviceOnline(route.WxWxid); online {
				Yyb().Warnf("[协议路由] callFunction 应用宝失败，尝试 wechat08 回退 %s err=%v", route.LogSummary(), err)
				data2, err2 := protocolCallFunctionViaWechat(route.WxWxid, appID, payload)
				if err2 == nil {
					Yyb().Infof("[协议路由] callFunction → wechat08回退 成功 wxid=%s appid=%s", protocolRefShort(route.WxWxid), appID)
				} else {
					Yyb().Warnf("[协议路由] callFunction → wechat08回退 失败 wxid=%s err=%v", protocolRefShort(route.WxWxid), err2)
				}
				return data2, err2
			}
		}
		Yyb().Warnf("[协议路由] callFunction → 应用宝 失败 %s appid=%s err=%v", route.LogSummary(), appID, err)
		return nil, err
	}
	Yyb().Infof("[协议路由] callFunction → wechat08 %s appid=%s", route.LogSummary(), appID)
	return protocolCallFunctionViaWechat(ref, appID, payload)
}

func protocolCallFunctionViaWechat(wxid, appID string, payload map[string]interface{}) (map[string]interface{}, error) {
	base := getWxJdServerForDevice(wxid)
	body := map[string]interface{}{
		"wxid":  wxid,
		"appid": appID,
		"data":  mustJSON(payload),
	}
	data, err := wxJdPostToURL(base, []string{"/api/v1/wx/app/call/function", "/wx/app/call/function"}, body, 15)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// LocalGatewayBaseURL 内置调用本机兼容网关
func LocalGatewayBaseURL() string {
	port := strings.TrimSpace(webHTTPPort())
	if port == "" {
		port = "8080"
	}
	return "http://127.0.0.1:" + port
}

func webHTTPPort() string {
	// beego 默认 HTTPPort，运行时可从环境变量覆盖
	if p := strings.TrimSpace(GetEnv("xdd_http_port")); p != "" {
		return p
	}
	return "8080"
}

// ForwardWxProtoRequest 转发到 wechat08
func ForwardWxProtoRequest(method, path, rawQuery string, body []byte) (respBody []byte, status int, contentType string, err error) {
	base := strings.TrimRight(getWxLoginBaseURL(), "/")
	url := base + path
	if rawQuery != "" {
		url += "?" + rawQuery
	}
	req, err := http.NewRequest(strings.ToUpper(method), url, strings.NewReader(string(body)))
	if err != nil {
		return nil, 0, "", err
	}
	if len(body) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		if alt := strings.TrimRight(getOldWxLoginBaseURL(), "/"); alt != "" && alt != base {
			url2 := alt + path
			if rawQuery != "" {
				url2 += "?" + rawQuery
			}
			req2, _ := http.NewRequest(strings.ToUpper(method), url2, strings.NewReader(string(body)))
			if len(body) > 0 {
				req2.Header.Set("Content-Type", "application/json")
			}
			resp, err = client.Do(req2)
		}
		if err != nil {
			return nil, 0, "", err
		}
	}
	defer resp.Body.Close()
	respBody, _ = io.ReadAll(resp.Body)
	contentType = resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/json; charset=utf-8"
	}
	return respBody, resp.StatusCode, contentType, nil
}

func extractCompatCode(data map[string]interface{}) string {
	if data == nil {
		return ""
	}
	if c, ok := data["code"].(string); ok && c != "" {
		return c
	}
	for _, key := range []string{"Data", "data"} {
		if inner, ok := data[key].(map[string]interface{}); ok {
			if c, ok := inner["code"].(string); ok && c != "" {
				return c
			}
		}
	}
	return ""
}

func ExtractCompatRef(rawQuery string, body []byte) string {
	if rawQuery != "" {
		for _, part := range strings.Split(rawQuery, "&") {
			kv := strings.SplitN(part, "=", 2)
			if len(kv) != 2 {
				continue
			}
			key := strings.ToLower(strings.TrimSpace(kv[0]))
			if key == "wxid" || key == "openid" || key == "ref" {
				return strings.TrimSpace(kv[1])
			}
		}
	}
	if len(body) == 0 {
		return ""
	}
	var m map[string]interface{}
	if json.Unmarshal(body, &m) != nil {
		return ""
	}
	for _, key := range []string{"wxid", "Wxid", "WXID", "openid", "OpenID", "ref", "Ref"} {
		if v, ok := m[key].(string); ok && strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func BuildCompatCodeResponse(code string) []byte {
	payload := map[string]interface{}{
		"Code":    0,
		"Success": true,
		"Data": map[string]interface{}{
			"code": code,
		},
	}
	b, _ := json.Marshal(payload)
	return b
}

func BuildCompatErrorResponse(msg string) []byte {
	payload := map[string]interface{}{
		"Code":    1,
		"Success": false,
		"Message": msg,
		"Data":    nil,
	}
	b, _ := json.Marshal(payload)
	return b
}

// ShouldRouteToYyb 兼容网关是否走应用宝
func ShouldRouteToYyb(ref string) bool {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return false
	}
	if IsYybOpenIDRef(ref) {
		return true
	}
	route := ResolveProtocolRoute(ref)
	return route.Backend == "yyb"
}

func ProtocolGetEidTokenForRef(ref string) string {
	payload := map[string]interface{}{
		"api_name":         "webapi_getuserinfo",
		"data":             map[string]string{"lang": "zh_CN"},
		"with_credentials": true,
	}
	data, err := ProtocolCallWxFunction(ref, WxJdAppID, payload)
	if err != nil {
		return ""
	}
	inner, _ := data["Data"].(map[string]interface{})
	if inner == nil {
		inner, _ = data["data"].(map[string]interface{})
	}
	if inner == nil {
		return ""
	}
	b64Data, _ := inner["data"].(string)
	if b64Data == "" {
		return ""
	}
	raw, err := base64.StdEncoding.DecodeString(b64Data)
	if err != nil {
		return ""
	}
	var parsed map[string]interface{}
	if json.Unmarshal(raw, &parsed) != nil {
		return ""
	}
	if v, ok := parsed["eid_token"].(string); ok {
		return v
	}
	if v, ok := parsed["eid"].(string); ok {
		return v
	}
	return ""
}
