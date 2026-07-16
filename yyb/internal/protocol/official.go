package protocol

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
)

// OfficialCGIRequest 描述一次通过应用宝登录态发起的公众号/通用 CGI 实验请求。
// 注意：应用宝登录态和 861 手机协议登录态不同，部分 micromsg-bin 接口可能被服务端拒绝；
// 这里保留 raw_base64 入口，便于后续按抓包结果精确补齐请求体。
type OfficialCGIRequest struct {
	CGIURL       string         `json:"cgi_url"`
	CmdID        uint64         `json:"cmd_id"`
	AppID        string         `json:"app_id"`
	BodyKind     string         `json:"body_kind"`
	HostAppID    string         `json:"host_app_id"`
	Payload      map[string]any `json:"payload"`
	RawBase64    string         `json:"raw_base64"`
	CookieBase64 string         `json:"cookie_base64"`
}

// OfficialCGI 使用当前应用宝账号的 WMPF session，经 wxaruntime_transfer 尝试调用指定 CGI。
// 它是底层实验能力：成功与否以服务端真实响应为准，不在上层伪造成功。
func (p *Pool) OfficialCGI(ctx context.Context, loginBuffer string, accountID int64, tcpProxy string, req OfficialCGIRequest) (map[string]any, error) {
	return p.run(ctx, loginBuffer, accountID, tcpProxy, func(ctx context.Context, st WmpfSession) (map[string]any, error) {
		plain, err := buildOfficialCGIRequest(st.Session, req)
		if err != nil {
			return nil, err
		}
		envelope, err := buildTransferPacket(st.Session, plain)
		if err != nil {
			return nil, err
		}
		code, resp, err := p.sendEnvelope(ctx, st, envelope)
		if err != nil {
			return nil, err
		}
		out := parseRawResponse(code, resp)
		out["_official"] = map[string]any{
			"cgi_url":  req.CGIURL,
			"cmd_id":   req.CmdID,
			"app_id":   req.AppID,
			"code_hex": hex.EncodeToString(code),
			"resp_hex": truncateHex(resp, 4096),
		}
		return out, nil
	})
}

func buildOfficialCGIRequest(sess AppSession, req OfficialCGIRequest) ([]byte, error) {
	cgiURL := strings.TrimSpace(req.CGIURL)
	if cgiURL == "" {
		return nil, fmt.Errorf("cgi_url is required")
	}
	if !strings.HasPrefix(cgiURL, "/cgi-bin/") {
		return nil, fmt.Errorf("cgi_url must start with /cgi-bin/")
	}
	if req.CmdID == 0 || req.CmdID > 10000 {
		return nil, fmt.Errorf("cmd_id must be between 1 and 10000")
	}
	appID := strings.TrimSpace(req.AppID)
	hostAppID := probeHostAppID(req.HostAppID, sess.HostAppID)
	body, sessDevice, err := buildOfficialCGIBody(sess, req)
	if err != nil {
		return nil, err
	}
	return buildJSAPIPlaintext(sess.UIN, appID, []byte(cgiURL), req.CmdID, body, hostAppID, sessDevice), nil
}

func buildOfficialCGIBody(sess AppSession, req OfficialCGIRequest) ([]byte, []byte, error) {
	kind := strings.TrimSpace(req.BodyKind)
	if kind == "" {
		kind = "default"
	}
	if strings.TrimSpace(req.RawBase64) != "" {
		raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(req.RawBase64))
		if err != nil {
			return nil, nil, fmt.Errorf("raw_base64 decode failed: %w", err)
		}
		return raw, nil, nil
	}
	switch kind {
	case "default":
		return nil, nil, nil
	case "empty":
		return []byte{}, nil, nil
	case "appid-only":
		sessDevice := randomSessDevice()
		body := make([]byte, 0, 100)
		body = append(body, pbLen(1, sessionInfo(uint32(sess.UIN), unifiedPCWindows, sessDevice))...)
		body = append(body, pbLen(2, []byte(req.AppID))...)
		return body, sessDevice, nil
	case "json-field3", "operate-json":
		sessDevice := randomSessDevice()
		payload := req.Payload
		if payload == nil {
			payload = map[string]any{}
		}
		raw, err := json.Marshal(payload)
		if err != nil {
			return nil, nil, err
		}
		body := make([]byte, 0, 180+len(raw))
		body = append(body, pbLen(1, sessionInfo(uint32(sess.UIN), unifiedPCWindows, sessDevice))...)
		body = append(body, pbLen(2, []byte(req.AppID))...)
		body = append(body, pbLen(3, raw)...)
		body = append(body, pbLen(4, nil)...)
		body = append(body, pbVar(5, 0)...)
		body = append(body, pbVar(6, 0)...)
		return body, sessDevice, nil
	case "oauth-authorize":
		// Critical logic: OAuthAuthorize is a real protobuf request, not a JSON payload.
		// 861 uses mm.OauthAuthorizeReq{BaseRequest, OauthUrl, BizUsername, Scene=4}.
		body, err := buildOAuthAuthorizeBody(sess, req)
		return body, nil, err
	case "geta8key":
		return buildGetA8KeyLikeBody(sess, req), nil, nil
	default:
		return nil, nil, fmt.Errorf("unsupported body_kind: %s", kind)
	}
}

func buildGetA8KeyLikeBody(sess AppSession, req OfficialCGIRequest) []byte {
	payload := req.Payload
	if payload == nil {
		payload = map[string]any{}
	}
	reqURL := stringFromMapAny(payload, "req_url", "ReqUrl", "url", "Url")
	netType := stringFromMapAny(payload, "net_type", "NetType")
	if netType == "" {
		netType = "WIFI"
	}
	opCode := uint64FromMap(payload, 2, "op_code", "OpCode")
	scene := uint64FromMap(payload, 4, "scene", "Scene")
	codeType := uint64FromMap(payload, 19, "code_type", "CodeType")
	codeVersion := uint64FromMap(payload, 10, "code_version", "CodeVersion")
	cookie, _ := base64.StdEncoding.DecodeString(strings.TrimSpace(req.CookieBase64))

	base := make([]byte, 0, 80)
	base = append(base, pbLen(1, sess.Ticket)...)
	base = append(base, pbVar(2, uint64(uint32(sess.UIN)))...)
	base = append(base, pbLen(3, sess.DeviceID)...)
	base = append(base, pbVar(4, uint64(clientVersion))...)
	base = append(base, pbLen(5, windowsName)...)
	base = append(base, pbVar(6, 0)...)

	body := make([]byte, 0, 256+len(reqURL)+len(cookie))
	body = append(body, pbLen(1, base)...)
	body = append(body, pbVar(2, codeType)...)
	body = append(body, pbVar(3, codeVersion)...)
	body = append(body, pbVar(4, 0)...)
	body = append(body, pbVar(5, 100)...)
	body = append(body, pbLen(6, []byte(netType))...)
	body = append(body, pbVar(7, opCode)...)
	body = append(body, pbLen(8, []byte{})...)
	body = append(body, pbLen(9, []byte(reqURL))...)
	body = append(body, pbVar(10, 0)...)
	body = append(body, pbVar(11, scene)...)
	body = append(body, pbLen(12, cookie)...)
	return body
}


func buildOAuthAuthorizeBody(sess AppSession, req OfficialCGIRequest) ([]byte, error) {
	payload := req.Payload
	if payload == nil {
		payload = map[string]any{}
	}
	oauthURL := stringFromMapAny(payload, "oauth_url", "OauthUrl")
	redirectURL := stringFromMapAny(payload, "url", "Url")
	// 关键逻辑：url 可能是业务 redirect_uri，也可能已经是完整 OAuth 地址；
	// 只有确认是 open.weixin OAuth 地址时才直接当 oauth_url 使用，避免把业务 URL 错当授权 URL。
	if oauthURL == "" && strings.Contains(redirectURL, "/connect/oauth2/authorize") {
		oauthURL = redirectURL
	}
	if oauthURL == "" {
		appid := firstNonEmptyProtocol(stringFromMapAny(payload, "appid", "app_id", "Appid", "AppID"), req.AppID)
		redirectURI := firstNonEmptyProtocol(stringFromMapAny(payload, "redirect_uri", "redirectUri", "RedirectUri"), redirectURL)
		scope := firstNonEmptyProtocol(stringFromMapAny(payload, "scope", "Scope"), "snsapi_base")
		state := stringFromMapAny(payload, "state", "State")
		component := stringFromMapAny(payload, "component_appid", "component_app_id", "ComponentAppid", "ComponentAppID")
		if appid != "" && redirectURI != "" {
			oauthURL = "https://open.weixin.qq.com/connect/oauth2/authorize?appid=" + urlQueryEscape(appid) +
				"&redirect_uri=" + urlQueryEscape(redirectURI) +
				"&response_type=code&scope=" + urlQueryEscape(scope) +
				"&state=" + urlQueryEscape(state)
			if component != "" {
				oauthURL += "&component_appid=" + urlQueryEscape(component)
			}
			oauthURL += "#wechat_redirect"
		}
	}
	if strings.TrimSpace(oauthURL) == "" {
		return nil, fmt.Errorf("oauth_url is required, or provide appid/app_id and redirect_uri")
	}
	bizUsername := stringFromMapAny(payload, "biz_username", "bizUsername", "BizUsername")
	scene := uint64FromMap(payload, 4, "scene", "Scene")
	body := make([]byte, 0, 256+len(oauthURL))
	body = append(body, pbLen(1, buildOfficialBaseRequest(sess, "literal"))...)
	body = append(body, pbLen(2, []byte(oauthURL))...)
	body = append(body, pbLen(3, []byte(bizUsername))...)
	body = append(body, pbVar(4, scene)...)
	return body, nil
}

func buildOfficialBaseRequest(sess AppSession, mode string) []byte {
	sessionKey := sess.Ticket
	if strings.EqualFold(mode, "literal") || len(sessionKey) == 0 {
		sessionKey = sessionKeyLiteral
	}
	body := make([]byte, 0, 96)
	body = append(body, pbLen(1, sessionKey)...)
	body = append(body, pbVar(2, uint64(uint32(sess.UIN)))...)
	body = append(body, pbLen(3, sess.DeviceID)...)
	body = append(body, pbVar(4, uint64(clientVersion))...)
	body = append(body, pbLen(5, unifiedPCWindows)...)
	body = append(body, pbVar(6, 0)...)
	return body
}

func firstNonEmptyProtocol(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func urlQueryEscape(s string) string {
	var out strings.Builder
	const hex = "0123456789ABCDEF"
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_' || c == '.' || c == '~' {
			out.WriteByte(c)
		} else {
			out.WriteByte('%')
			out.WriteByte(hex[c>>4])
			out.WriteByte(hex[c&15])
		}
	}
	return out.String()
}

func stringFromMapAny(m map[string]any, keys ...string) string {
	for _, k := range keys {
		if v, ok := m[k].(string); ok && strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func uint64FromMap(m map[string]any, def uint64, keys ...string) uint64 {
	for _, k := range keys {
		switch v := m[k].(type) {
		case int:
			return uint64(v)
		case int64:
			return uint64(v)
		case uint64:
			return v
		case float64:
			return uint64(v)
		case json.Number:
			if n, err := v.Int64(); err == nil {
				return uint64(n)
			}
		}
	}
	return def
}
