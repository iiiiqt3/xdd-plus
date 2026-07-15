package yybportal

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/cdle/xdd/models"
	"github.com/cdle/xdd/yyb/internal/protocol"
	"github.com/cdle/xdd/yyb/internal/qr"
)

type portalQRState struct {
	Session          *qr.Session
	Client           *qr.Client
	UserNumber       int
	BornAt           time.Time
	TCPProxy         string
	ProxyMeta        map[string]interface{}
	PollProxyRetries int
	Cost             int
	DeductCoin       bool
}

var portalQRStore = struct {
	sync.Mutex
	m map[string]*portalQRState
}{m: map[string]*portalQRState{}}

const portalQRProxyRotateLimit = 5

func portalQRTimeout() time.Duration {
	if models.Config.Yyb.Enabled {
		return 45 * time.Second
	}
	return 45 * time.Second
}

// portalStartScanLogin 创建带 51 代理的扫码会话
func portalStartScanLogin(userNumber int, cost int, deduct bool, proxyOpt models.YybProxyLoginOption) (imageB64, sessionID string, meta map[string]interface{}, err error) {
	proxyOpt.Enabled = proxyOpt.Enabled || models.Config.Yyb.Proxy51Enabled
	maxAttempts := 1
	if models.Config.Yyb.Proxy51Enabled && proxyOpt.Enabled {
		maxAttempts = portalQRProxyRotateLimit
	}
	timeout := portalQRTimeout()
	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		tcpProxy, proxyMeta, perr := models.YybBuildProxyForLogin(proxyOpt)
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
			Session:    img.Session,
			Client:     client,
			UserNumber: userNumber,
			BornAt:     time.Now(),
			TCPProxy:   tcpProxy,
			ProxyMeta:  proxyMeta,
			Cost:       cost,
			DeductCoin: deduct,
		}
		portalQRStore.Lock()
		portalQRStore.m[img.Session.ID] = state
		for id, st := range portalQRStore.m {
			if time.Since(st.BornAt) > 10*time.Minute {
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

func portalQRCodePollShouldRotate(err error, state *portalQRState) bool {
	if err == nil || state == nil || state.PollProxyRetries >= portalQRProxyRotateLimit || strings.TrimSpace(state.TCPProxy) == "" {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "SOCKS5 connect failed") ||
		strings.Contains(msg, "05020001") ||
		strings.Contains(strings.ToLower(msg), "proxy")
}

func portalProxyOptionFromQRState(state *portalQRState) models.YybProxyLoginOption {
	if state == nil || state.ProxyMeta == nil {
		return models.YybProxyLoginOption{}
	}
	if !boolFromMap(state.ProxyMeta["yyb_proxy_enabled"]) {
		return models.YybProxyLoginOption{Enabled: models.Config.Yyb.Proxy51Enabled}
	}
	return models.YybProxyLoginOption{
		Enabled:    true,
		PackID:     stringFromMap(state.ProxyMeta["yyb_proxy_packid"]),
		RegionCode: stringFromMap(state.ProxyMeta["yyb_proxy_region_code"]),
		RegionName: stringFromMap(state.ProxyMeta["yyb_proxy_region_name"]),
	}
}

func portalRotateQRCodeProxy(state *portalQRState) error {
	if state == nil {
		return fmt.Errorf("扫码会话为空")
	}
	opt := portalProxyOptionFromQRState(state)
	opt.Enabled = true
	state.PollProxyRetries++
	tcpProxy, proxyMeta, err := models.YybBuildProxyForLogin(opt)
	if err != nil {
		return fmt.Errorf("扫码确认代理异常，自动换代理失败（第 %d/%d 次）：%v", state.PollProxyRetries, portalQRProxyRotateLimit, err)
	}
	if strings.TrimSpace(tcpProxy) == "" {
		return fmt.Errorf("扫码确认代理异常，自动换代理失败（第 %d/%d 次）：未提取到代理", state.PollProxyRetries, portalQRProxyRotateLimit)
	}
	state.Client.ReplaceProxyForSession(state.Session, tcpProxy, false)
	state.TCPProxy = tcpProxy
	if state.ProxyMeta == nil {
		state.ProxyMeta = map[string]interface{}{}
	}
	for k, v := range proxyMeta {
		state.ProxyMeta[k] = v
	}
	state.ProxyMeta["yyb_proxy_poll_rotate_count"] = state.PollProxyRetries
	state.ProxyMeta["yyb_proxy_poll_rotate_at"] = time.Now().Unix()
	return nil
}

func portalGetLoginBufferWithProxyRetry(state *portalQRState) (protocol.LoginBufferResult, error) {
	if state == nil || state.Client == nil {
		return protocol.LoginBufferResult{}, fmt.Errorf("扫码会话为空")
	}
	for {
		result, err := state.Client.GetLoginBuffer(context.Background(), state.Session)
		if err == nil {
			return result, nil
		}
		if strings.TrimSpace(state.TCPProxy) != "" && state.PollProxyRetries >= portalQRProxyRotateLimit &&
			(strings.Contains(err.Error(), "SOCKS5 connect failed") || strings.Contains(err.Error(), "05020001")) {
			return protocol.LoginBufferResult{}, fmt.Errorf("应用宝授权成功，但获取登录缓存时代理连续 %d 次异常，请换地区或稍后重试：%v", portalQRProxyRotateLimit, err)
		}
		if !portalQRCodePollShouldRotate(err, state) {
			return protocol.LoginBufferResult{}, err
		}
		if rotateErr := portalRotateQRCodeProxy(state); rotateErr != nil {
			return protocol.LoginBufferResult{}, rotateErr
		}
		if state.PollProxyRetries >= portalQRProxyRotateLimit {
			result, err = state.Client.GetLoginBuffer(context.Background(), state.Session)
			if err != nil {
				return protocol.LoginBufferResult{}, fmt.Errorf("应用宝授权成功，但获取登录缓存时代理连续 %d 次异常，请换地区或稍后重试：%v", portalQRProxyRotateLimit, err)
			}
			return result, nil
		}
	}
}

func portalPollScanLogin(sessionID string, userNumber int) (map[string]interface{}, error) {
	sessionID = strings.TrimSpace(sessionID)
	portalQRStore.Lock()
	state := portalQRStore.m[sessionID]
	portalQRStore.Unlock()
	if state == nil || state.UserNumber != userNumber {
		return nil, fmt.Errorf("扫码会话无效或已过期")
	}
	var poll qr.PollResult
	for {
		var err error
		poll, err = state.Client.PollQRCode(context.Background(), state.Session)
		if err == nil {
			break
		}
		if !portalQRCodePollShouldRotate(err, state) {
			return nil, err
		}
		if rotateErr := portalRotateQRCodeProxy(state); rotateErr != nil {
			return nil, rotateErr
		}
		if state.PollProxyRetries < portalQRProxyRotateLimit {
			continue
		}
		poll, err = state.Client.PollQRCode(context.Background(), state.Session)
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

func portalConfirmScanLogin(sessionID string, userNumber int, clientCtx models.ClientContext) (map[string]interface{}, error) {
	sessionID = strings.TrimSpace(sessionID)
	portalQRStore.Lock()
	state := portalQRStore.m[sessionID]
	portalQRStore.Unlock()
	if state == nil || state.UserNumber != userNumber {
		return nil, fmt.Errorf("扫码会话无效或已过期")
	}
	result, err := portalGetLoginBufferWithProxyRetry(state)
	if err != nil {
		return nil, err
	}
	a, err := svc()
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	acc, err := a.StoreScanAccount(ctx, result.LoginBuffer, result.Credentials, state.ProxyMeta)
	if err != nil {
		return nil, err
	}
	portalQRStore.Lock()
	delete(portalQRStore.m, sessionID)
	portalQRStore.Unlock()

	everBound := hasEverBoundOpenID(userNumber, acc.OpenID)
	costCharged := 0
	if state.DeductCoin && !everBound {
		nick := ""
		if acc.Nickname != nil {
			nick = *acc.Nickname
		}
		if err := deductCoin(userNumber, state.Cost, clientCtx, fmt.Sprintf("应用宝扫码登录 %s", nick)); err != nil {
			return nil, err
		}
		models.RecordClientSourceEvent(userNumber, "yyb_scan_login", clientCtx)
		costCharged = state.Cost
	}
	binding, err := bindAccount(userNumber, acc, "alive")
	if err != nil {
		return nil, err
	}
	view := toPortalView(ctx, *binding, a)
	return map[string]interface{}{
		"account":      view,
		"cost":         costCharged,
		"alreadyBound": everBound,
	}, nil
}

func boolFromMap(v interface{}) bool {
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

func stringFromMap(v interface{}) string {
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
