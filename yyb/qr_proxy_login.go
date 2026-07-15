package yyb

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/cdle/xdd/yyb/internal/protocol"
	"github.com/cdle/xdd/yyb/internal/qr"
)

// ProxyScanOption 门户扫码代理选项（由 yybportal 从 models 映射）
type ProxyScanOption struct {
	Enabled    bool
	PackID     string
	RegionCode string
	RegionName string
}

// ProxyLoginBuilder 提取/复用 51 SOCKS5（由 yybportal 注入 models.YybBuildProxyForLogin）
type ProxyLoginBuilder func(opt ProxyScanOption) (tcpProxy string, meta map[string]interface{}, err error)

var (
	proxyLoginBuild   ProxyLoginBuilder
	proxyLoginEnabled func() bool
)

// SetProxyLoginHooks 注入代理构建与开关（避免 yyb 依赖 models 形成循环引用）
func SetProxyLoginHooks(build ProxyLoginBuilder, enabled func() bool) {
	proxyLoginBuild = build
	proxyLoginEnabled = enabled
}

type portalQRState struct {
	session          *qr.Session
	client           *qr.Client
	ownerKey         int
	bornAt           time.Time
	tcpProxy         string
	proxyMeta        map[string]interface{}
	pollProxyRetries int
	cost             int
	deductCoin       bool
}

var portalQRStore = struct {
	sync.Mutex
	m map[string]*portalQRState
}{m: map[string]*portalQRState{}}

const portalQRProxyRotateLimit = 5

func portalQRTimeout() time.Duration { return 45 * time.Second }

func buildProxy(opt ProxyScanOption) (string, map[string]interface{}, error) {
	if proxyLoginBuild == nil {
		return "", nil, nil
	}
	return proxyLoginBuild(opt)
}

func proxyEnabled() bool {
	if proxyLoginEnabled == nil {
		return false
	}
	return proxyLoginEnabled()
}

// StartPortalScanLogin 创建带 51 代理的扫码会话；ownerKey 通常为门户用户编号
func StartPortalScanLogin(ownerKey int, cost int, deduct bool, proxyOpt ProxyScanOption) (imageDataURI, sessionID string, meta map[string]interface{}, err error) {
	proxyOpt.Enabled = proxyOpt.Enabled || proxyEnabled()
	maxAttempts := 1
	if proxyEnabled() && proxyOpt.Enabled {
		maxAttempts = portalQRProxyRotateLimit
	}
	timeout := portalQRTimeout()
	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		tcpProxy, proxyMeta, perr := buildProxy(proxyOpt)
		if perr != nil {
			lastErr = perr
			continue
		}
		client := qr.NewClientWithProxy(timeout, tcpProxy, false)
		img, err := client.GetQRCodeImage(context.Background())
		if err != nil {
			lastErr = err
			if strings.TrimSpace(tcpProxy) != "" && attempt < maxAttempts {
				continue
			}
			break
		}
		if proxyMeta == nil {
			proxyMeta = map[string]interface{}{}
		}
		proxyMeta["yyb_proxy_qr_create_attempt"] = attempt
		state := &portalQRState{
			session:    img.Session,
			client:     client,
			ownerKey:   ownerKey,
			bornAt:     time.Now(),
			tcpProxy:   tcpProxy,
			proxyMeta:  proxyMeta,
			cost:       cost,
			deductCoin: deduct,
		}
		portalQRStore.Lock()
		portalQRStore.m[img.Session.ID] = state
		for id, st := range portalQRStore.m {
			if time.Since(st.bornAt) > 10*time.Minute {
				delete(portalQRStore.m, id)
			}
		}
		portalQRStore.Unlock()
		return qr.DataURIJPEG(img.ImageBytes), img.Session.ID, proxyMeta, nil
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("未知错误")
	}
	return "", "", nil, fmt.Errorf("应用宝二维码创建失败，已自动重试 %d 次，请换地区或稍后重试：%v", maxAttempts, lastErr)
}

// PollPortalScanLogin 轮询扫码（含代理异常自动换 IP）
func PollPortalScanLogin(sessionID string, ownerKey int) (map[string]interface{}, error) {
	sessionID = strings.TrimSpace(sessionID)
	portalQRStore.Lock()
	state := portalQRStore.m[sessionID]
	portalQRStore.Unlock()
	if state == nil || state.ownerKey != ownerKey {
		return nil, fmt.Errorf("扫码会话无效或已过期")
	}
	var poll qr.PollResult
	for {
		var err error
		poll, err = state.client.PollQRCode(context.Background(), state.session)
		if err == nil {
			break
		}
		if !portalQRCodePollShouldRotate(err, state) {
			return nil, err
		}
		if rotateErr := portalRotateQRCodeProxy(state); rotateErr != nil {
			return nil, rotateErr
		}
		if state.pollProxyRetries < portalQRProxyRotateLimit {
			continue
		}
		poll, err = state.client.PollQRCode(context.Background(), state.session)
		if err != nil {
			return nil, fmt.Errorf("扫码确认代理连续 %d 次异常，请换地区或稍后重试：%v", portalQRProxyRotateLimit, err)
		}
		break
	}
	out := map[string]interface{}{"status": poll.Status}
	switch poll.Status {
	case "pending", "scanned":
		return out, nil
	case "cancelled", "expired", "unknown":
		portalQRStore.Lock()
		delete(portalQRStore.m, sessionID)
		portalQRStore.Unlock()
		if poll.Message != "" {
			out["message"] = poll.Message
		}
		return out, nil
	case "authorized", "confirmed":
		out["status"] = "authorized"
		return out, nil
	default:
		return out, nil
	}
}

// FinishPortalScanLogin 获取 login_buffer 并结束会话；调用方负责写库与绑定
func FinishPortalScanLogin(sessionID string, ownerKey int) (loginBuffer string, creds protocol.LoginBufferCredentials, proxyMeta map[string]interface{}, cost int, deduct bool, err error) {
	sessionID = strings.TrimSpace(sessionID)
	portalQRStore.Lock()
	state := portalQRStore.m[sessionID]
	portalQRStore.Unlock()
	if state == nil || state.ownerKey != ownerKey {
		return "", protocol.LoginBufferCredentials{}, nil, 0, false, fmt.Errorf("扫码会话无效或已过期")
	}
	result, err := portalGetLoginBufferWithProxyRetry(state)
	if err != nil {
		return "", protocol.LoginBufferCredentials{}, nil, 0, false, err
	}
	cost = state.cost
	deduct = state.deductCoin
	proxyMeta = state.proxyMeta
	portalQRStore.Lock()
	delete(portalQRStore.m, sessionID)
	portalQRStore.Unlock()
	return result.LoginBuffer, result.Credentials, proxyMeta, cost, deduct, nil
}

func portalQRCodePollShouldRotate(err error, state *portalQRState) bool {
	if err == nil || state == nil || state.pollProxyRetries >= portalQRProxyRotateLimit || strings.TrimSpace(state.tcpProxy) == "" {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "SOCKS5 connect failed") ||
		strings.Contains(msg, "05020001") ||
		strings.Contains(strings.ToLower(msg), "proxy")
}

func portalProxyOptionFromState(state *portalQRState) ProxyScanOption {
	if state == nil || state.proxyMeta == nil {
		return ProxyScanOption{}
	}
	if !metaBool(state.proxyMeta["yyb_proxy_enabled"]) {
		return ProxyScanOption{Enabled: proxyEnabled()}
	}
	return ProxyScanOption{
		Enabled:    true,
		PackID:     metaString(state.proxyMeta["yyb_proxy_packid"]),
		RegionCode: metaString(state.proxyMeta["yyb_proxy_region_code"]),
		RegionName: metaString(state.proxyMeta["yyb_proxy_region_name"]),
	}
}

func portalRotateQRCodeProxy(state *portalQRState) error {
	if state == nil {
		return fmt.Errorf("扫码会话为空")
	}
	opt := portalProxyOptionFromState(state)
	opt.Enabled = true
	state.pollProxyRetries++
	tcpProxy, proxyMeta, err := buildProxy(opt)
	if err != nil {
		return fmt.Errorf("扫码确认代理异常，自动换代理失败（第 %d/%d 次）：%v", state.pollProxyRetries, portalQRProxyRotateLimit, err)
	}
	if strings.TrimSpace(tcpProxy) == "" {
		return fmt.Errorf("扫码确认代理异常，自动换代理失败（第 %d/%d 次）：未提取到代理", state.pollProxyRetries, portalQRProxyRotateLimit)
	}
	state.client.ReplaceProxyForSession(state.session, tcpProxy, false)
	state.tcpProxy = tcpProxy
	if state.proxyMeta == nil {
		state.proxyMeta = map[string]interface{}{}
	}
	for k, v := range proxyMeta {
		state.proxyMeta[k] = v
	}
	state.proxyMeta["yyb_proxy_poll_rotate_count"] = state.pollProxyRetries
	state.proxyMeta["yyb_proxy_poll_rotate_at"] = time.Now().Unix()
	return nil
}

func portalGetLoginBufferWithProxyRetry(state *portalQRState) (protocol.LoginBufferResult, error) {
	if state == nil || state.client == nil {
		return protocol.LoginBufferResult{}, fmt.Errorf("扫码会话为空")
	}
	for {
		result, err := state.client.GetLoginBuffer(context.Background(), state.session)
		if err == nil {
			return result, nil
		}
		if strings.TrimSpace(state.tcpProxy) != "" && state.pollProxyRetries >= portalQRProxyRotateLimit &&
			(strings.Contains(err.Error(), "SOCKS5 connect failed") || strings.Contains(err.Error(), "05020001")) {
			return protocol.LoginBufferResult{}, fmt.Errorf("应用宝授权成功，但获取登录缓存时代理连续 %d 次异常，请换地区或稍后重试：%v", portalQRProxyRotateLimit, err)
		}
		if !portalQRCodePollShouldRotate(err, state) {
			return protocol.LoginBufferResult{}, err
		}
		if rotateErr := portalRotateQRCodeProxy(state); rotateErr != nil {
			return protocol.LoginBufferResult{}, rotateErr
		}
		if state.pollProxyRetries >= portalQRProxyRotateLimit {
			result, err = state.client.GetLoginBuffer(context.Background(), state.session)
			if err != nil {
				return protocol.LoginBufferResult{}, fmt.Errorf("应用宝授权成功，但获取登录缓存时代理连续 %d 次异常，请换地区或稍后重试：%v", portalQRProxyRotateLimit, err)
			}
			return result, nil
		}
	}
}

func metaBool(v interface{}) bool {
	switch x := v.(type) {
	case bool:
		return x
	case string:
		return strings.EqualFold(strings.TrimSpace(x), "true") || strings.TrimSpace(x) == "1"
	case float64:
		return x != 0
	default:
		return false
	}
}

func metaString(v interface{}) string {
	switch x := v.(type) {
	case string:
		return strings.TrimSpace(x)
	case float64:
		if x == float64(int64(x)) {
			return fmt.Sprintf("%d", int64(x))
		}
		return fmt.Sprintf("%v", x)
	default:
		return ""
	}
}
