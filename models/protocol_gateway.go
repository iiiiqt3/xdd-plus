package models

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/beego/beego/v2/server/web"
)

// 由 yybportal 在启动时注入，避免 models ↔ yybportal 循环依赖
var (
	protocolYybGetCode           func(openid, appID string) (map[string]interface{}, error)
	protocolYybGetPhone          func(openid, appID string) (map[string]interface{}, error)
	protocolYybOperate           func(openid, appID string, payload map[string]interface{}) (map[string]interface{}, error)
	protocolYybIsAlive           func(openid string) bool
	protocolYybRefresh           func(openid string) (string, error)
	protocolYybAccountExistsFn   func(ref string) bool
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

// SetProtocolYybGetPhone 注册应用宝取手机号
func SetProtocolYybGetPhone(fn func(openid, appID string) (map[string]interface{}, error)) {
	protocolYybGetPhone = fn
}

// SetProtocolYybAccountExists 注册应用宝账号是否存在检查
func SetProtocolYybAccountExists(fn func(ref string) bool) {
	protocolYybAccountExistsFn = fn
}

func friendlyProtocolErr(err error) error {
	if err == nil {
		return nil
	}
	msg := strings.TrimSpace(err.Error())
	lower := strings.ToLower(msg)
	if errors.Is(err, sql.ErrNoRows) || strings.Contains(lower, "no rows") {
		return fmt.Errorf("应用宝账号不存在（库中无此 openid），请重新扫码登录后发送【记录授权】更新账号")
	}
	if strings.Contains(lower, "account expired") || strings.Contains(msg, "账号已失效") {
		return fmt.Errorf("应用宝账号已失效，请重新扫码登录")
	}
	if strings.Contains(lower, "account not found") {
		return fmt.Errorf("应用宝账号不存在，请重新扫码登录后发送【记录授权】更新账号")
	}
	return err
}

// EnsureQueryProtocolRef 查询脚本执行前校验应用宝账号是否在库
func EnsureQueryProtocolRef(ref string) error {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return fmt.Errorf("账号标识为空")
	}
	route := ResolveProtocolRoute(ref)
	if route.Backend != "yyb" {
		return nil
	}
	openid := strings.TrimSpace(route.OpenID)
	if openid == "" {
		openid = ref
	}
	if protocolYybAccountExistsFn != nil && !protocolYybAccountExistsFn(openid) {
		return fmt.Errorf("应用宝账号不存在（库中无此 openid），请重新扫码登录后发送【记录授权】更新账号")
	}
	return nil
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
	if protocolYybRefresh != nil {
		_, _ = protocolYybRefresh(openid)
		if data2, err2 := protocolYybGetCode(openid, appID); err2 == nil {
			return data2, nil
		}
	}
	if !route.FromBind || strings.TrimSpace(route.WxWxid) == "" {
		return nil, err
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
		return "", friendlyProtocolErr(yybErr)
	}
	Yyb().Infof("[协议路由] getCode → wechat08 %s appid=%s", route.LogSummary(), appID)
	return protocolGetWxCodeViaWechat(ref, appID)
}

// ProtocolGetWxAppPhone 统一取小程序手机号（授权 code / 手机号列表）
func ProtocolGetWxAppPhone(ref, appID string) (map[string]interface{}, error) {
	ref = strings.TrimSpace(ref)
	appID = strings.TrimSpace(appID)
	if ref == "" {
		return nil, fmt.Errorf("缺少账号标识")
	}
	if appID == "" {
		appID = WxJdAppID
	}
	route := ResolveProtocolRoute(ref)
	if route.Backend == "yyb" {
		if protocolYybGetPhone == nil {
			return nil, fmt.Errorf("应用宝模块未就绪")
		}
		data, err := protocolYybGetPhone(route.OpenID, appID)
		if err == nil {
			Yyb().Infof("[协议路由] getPhone → 应用宝 成功 %s appid=%s", route.LogSummary(), appID)
			return data, nil
		}
		if route.FromBind && strings.TrimSpace(route.WxWxid) != "" {
			if online, _ := checkWxDeviceOnline(route.WxWxid); online {
				Yyb().Warnf("[协议路由] getPhone 应用宝失败，尝试 wechat08 回退 %s err=%v", route.LogSummary(), err)
				data2, err2 := protocolGetPhoneViaWechat(route.WxWxid, appID)
				if err2 == nil {
					Yyb().Infof("[协议路由] getPhone → wechat08回退 成功 wxid=%s appid=%s", protocolRefShort(route.WxWxid), appID)
				}
				return data2, err2
			}
		}
		Yyb().Warnf("[协议路由] getPhone → 应用宝 失败 %s appid=%s err=%v", route.LogSummary(), appID, err)
		return nil, friendlyProtocolErr(err)
	}
	Yyb().Infof("[协议路由] getPhone → wechat08 %s appid=%s", route.LogSummary(), appID)
	return protocolGetPhoneViaWechat(ref, appID)
}

func protocolGetPhoneViaWechat(wxid, appID string) (map[string]interface{}, error) {
	if strings.TrimSpace(wxid) == "" {
		return nil, fmt.Errorf("缺少微信ID")
	}
	base := getWxJdServerForDevice(wxid)
	body := map[string]interface{}{
		"wxid":  wxid,
		"appid": appID,
	}
	return wxJdPostToURL(base, []string{"/api/Wxapp/GetAllMobile", "/api/v1/wx/app/get/all/mobile"}, body, 15)
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

// ScriptWechatGatewayURL 查询/记录等脚本应请求的协议网关（xdd 本机，自动分流应用宝/wechat08）
func ScriptWechatGatewayURL() string {
	if u := strings.TrimRight(strings.TrimSpace(LocalGatewayBaseURL()), "/"); u != "" {
		return u
	}
	return "http://127.0.0.1:8080"
}

// ScriptWechatGatewayEnvs 注入 WECHAT_SERVER，供 scripts/query 下脚本取 code
func ScriptWechatGatewayEnvs() []string {
	u := ScriptWechatGatewayURL()
	return []string{
		"WECHAT_SERVER=" + u,
		"WECHAT_SERVER_NEW=" + u,
	}
}

func webHTTPPort() string {
	if p := strings.TrimSpace(GetEnv("xdd_http_port")); p != "" {
		return p
	}
	if web.BConfig.Listen.HTTPPort > 0 {
		return strconv.Itoa(web.BConfig.Listen.HTTPPort)
	}
	return "8080"
}

// CheckWxDeviceOnline 供兼容网关判断双绑微信是否在线（可回退 wechat08）
func CheckWxDeviceOnline(wxid string) (bool, error) {
	return checkWxDeviceOnline(wxid)
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

func BuildCompatSuccessResponse(data interface{}) []byte {
	payload := map[string]interface{}{
		"Code":    0,
		"Success": true,
		"Message": "成功",
		"Data":    data,
	}
	b, _ := json.Marshal(payload)
	return b
}

// BuildCompatGetAllMobileResponse 将应用宝/wechat08 结果统一为旧脚本期望的 GetAllMobile 结构：
// {Code, Success, Data: {Data: "<json string>", ALLMobile: [...]}}
func BuildCompatGetAllMobileResponse(raw map[string]interface{}) []byte {
	if raw == nil {
		return BuildCompatErrorResponse("GetAllMobile 未返回数据")
	}
	if isCompatOKEnvelope(raw) {
		if inner, ok := raw["Data"].(map[string]interface{}); ok {
			if _, has := inner["Data"]; has {
				b, _ := json.Marshal(raw)
				return b
			}
		}
	}
	if inner, ok := raw["Data"].(map[string]interface{}); ok {
		if _, has := inner["Data"]; has {
			return BuildCompatSuccessResponse(inner)
		}
	}
	if inner, ok := raw["data"].(map[string]interface{}); ok {
		if _, has := inner["Data"]; has {
			return BuildCompatSuccessResponse(inner)
		}
		if _, has := inner["data"]; has {
			return BuildCompatSuccessResponse(inner)
		}
	}

	phoneCode, mobile := extractPhoneAuthCode(raw)
	if phoneCode == "" {
		return BuildCompatErrorResponse("GetAllMobile 未找到手机号授权 code")
	}
	itemData, _ := json.Marshal(map[string]string{"code": phoneCode})
	item := map[string]interface{}{
		"mobile":       mobile,
		"show_mobile":  mobile,
		"need_auth":    "0",
		"allow_send_sms": "0",
		"data":         string(itemData),
		"code":         phoneCode,
	}
	innerObj := map[string]interface{}{
		"custom_phone_list": []map[string]interface{}{item},
	}
	if mobile != "" {
		innerObj["wx_phone"] = item
	}
	innerBytes, _ := json.Marshal(innerObj)
	outerData := map[string]interface{}{
		"Data":      string(innerBytes),
		"ALLMobile": []map[string]interface{}{item},
	}
	return BuildCompatSuccessResponse(outerData)
}

func isCompatOKEnvelope(m map[string]interface{}) bool {
	if v, ok := m["Success"].(bool); ok && v {
		return true
	}
	switch c := m["Code"].(type) {
	case float64:
		return c == 0
	case int:
		return c == 0
	case int64:
		return c == 0
	}
	return false
}

func extractPhoneAuthCode(raw map[string]interface{}) (code, mobile string) {
	var walk func(interface{}) bool
	walk = func(v interface{}) bool {
		switch x := v.(type) {
		case map[string]interface{}:
			if m := strings.TrimSpace(stringFromAnyMap(x, "mobile", "phone", "show_mobile")); m != "" && mobile == "" {
				mobile = m
			}
			if c := strings.TrimSpace(stringFromAnyMap(x, "code", "phoneCode", "wxCode")); c != "" {
				code = c
				return true
			}
			if dataStr := strings.TrimSpace(stringFromAnyMap(x, "data", "Data")); dataStr != "" && strings.HasPrefix(dataStr, "{") {
				var inner map[string]interface{}
				if json.Unmarshal([]byte(dataStr), &inner) == nil {
					if c := strings.TrimSpace(stringFromAnyMap(inner, "code", "phoneCode")); c != "" {
						code = c
						return true
					}
				}
			}
			for _, key := range []string{"Data", "data", "result", "ALLMobile", "custom_phone_list", "wx_phone"} {
				if walk(x[key]) {
					return true
				}
			}
			for _, val := range x {
				if walk(val) {
					return true
				}
			}
		case []interface{}:
			for _, item := range x {
				if walk(item) {
					return true
				}
			}
		case string:
			text := strings.TrimSpace(x)
			if text == "" {
				return false
			}
			if strings.HasPrefix(text, "{") {
				var obj map[string]interface{}
				if json.Unmarshal([]byte(text), &obj) == nil && walk(obj) {
					return true
				}
			}
		}
		return false
	}
	walk(raw)
	return code, mobile
}

func stringFromAnyMap(m map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		if v, ok := m[key]; ok {
			if s := strings.TrimSpace(fmt.Sprint(v)); s != "" && s != "<nil>" {
				return s
			}
		}
	}
	return ""
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
