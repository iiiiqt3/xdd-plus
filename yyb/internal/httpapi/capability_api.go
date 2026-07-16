package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cdle/xdd/yyb/internal/protocol"
	"github.com/cdle/xdd/yyb/internal/store"
)

// OfficialCGI 公众号/通用 CGI
func (a *App) OfficialCGI(ctx context.Context, ref string, req protocol.OfficialCGIRequest) (map[string]any, error) {
	acc, err := a.db.ResolveAccount(ctx, ref)
	if err != nil {
		return nil, err
	}
	result, used, err := a.runBusinessWithProxyRetry(ctx, acc, func(current *store.WechatAccount, tcpProxy string) (map[string]any, error) {
		return a.pool.OfficialCGI(ctx, current.LoginBuffer, current.ID, tcpProxy, req)
	})
	if err != nil {
		return nil, err
	}
	return map[string]any{"openid": used.OpenID, "result": result}, nil
}

// TenPayCGI 微信支付 CGI
func (a *App) TenPayCGI(ctx context.Context, ref string, req protocol.TenPayRequest) (map[string]any, error) {
	acc, err := a.db.ResolveAccount(ctx, ref)
	if err != nil {
		return nil, err
	}
	if req.CGICmd == 0 {
		req.CGICmd = 85
	}
	result, used, err := a.runBusinessWithProxyRetry(ctx, acc, func(current *store.WechatAccount, tcpProxy string) (map[string]any, error) {
		return a.pool.TenPayCGI(ctx, current.LoginBuffer, current.ID, tcpProxy, req)
	})
	if err != nil {
		return nil, err
	}
	return map[string]any{"openid": used.OpenID, "result": result}, nil
}

// RuntimeSession 小程序 runtime session
func (a *App) RuntimeSession(ctx context.Context, ref, appID string, payload map[string]any) (map[string]any, error) {
	if strings.TrimSpace(appID) == "" {
		return nil, fmt.Errorf("app_id is required")
	}
	req := protocol.OfficialCGIRequest{
		CGIURL:   "/cgi-bin/mmbiz-bin/wxabusiness/getruntimesession",
		CmdID:    3540,
		AppID:    appID,
		BodyKind: "appid-only",
		Payload:  payload,
	}
	return a.OfficialCGI(ctx, ref, req)
}

// UpdateStep 上传微信运动步数
func (a *App) UpdateStep(ctx context.Context, ref string, req protocol.StepRequest) (map[string]any, error) {
	acc, err := a.db.ResolveAccount(ctx, ref)
	if err != nil {
		return nil, err
	}
	result, used, err := a.runBusinessWithProxyRetry(ctx, acc, func(current *store.WechatAccount, tcpProxy string) (map[string]any, error) {
		return a.pool.UpdateStep(ctx, current.LoginBuffer, current.ID, tcpProxy, req)
	})
	if err != nil {
		return nil, err
	}
	return map[string]any{"openid": used.OpenID, "result": result}, nil
}

// ReportMotion 上报运动（同 UpdateStep）
func (a *App) ReportMotion(ctx context.Context, ref string, req protocol.StepRequest) (map[string]any, error) {
	acc, err := a.db.ResolveAccount(ctx, ref)
	if err != nil {
		return nil, err
	}
	result, used, err := a.runBusinessWithProxyRetry(ctx, acc, func(current *store.WechatAccount, tcpProxy string) (map[string]any, error) {
		return a.pool.ReportMotion(ctx, current.LoginBuffer, current.ID, tcpProxy, req)
	})
	if err != nil {
		return nil, err
	}
	return map[string]any{"openid": used.OpenID, "result": result}, nil
}

// GetBoundHardDevices 查询绑定硬件设备
func (a *App) GetBoundHardDevices(ctx context.Context, ref string, req protocol.StepRequest) (map[string]any, error) {
	acc, err := a.db.ResolveAccount(ctx, ref)
	if err != nil {
		return nil, err
	}
	result, used, err := a.runBusinessWithProxyRetry(ctx, acc, func(current *store.WechatAccount, tcpProxy string) (map[string]any, error) {
		return a.pool.GetBoundHardDevices(ctx, current.LoginBuffer, current.ID, tcpProxy, req)
	})
	if err != nil {
		return nil, err
	}
	return map[string]any{"openid": used.OpenID, "result": result}, nil
}

// GetWeRunData 获取微信运动 encryptedData/iv
func (a *App) GetWeRunData(ctx context.Context, ref, appID string) (map[string]any, error) {
	if strings.TrimSpace(appID) == "" {
		return nil, fmt.Errorf("app_id is required")
	}
	acc, err := a.db.ResolveAccount(ctx, ref)
	if err != nil {
		return nil, err
	}
	run := func(current *store.WechatAccount, tcpProxy string) (map[string]any, error) {
		var last map[string]any
		for _, apiName := range []string{"webapi_getwerundata", "getWeRunData", "webapi_getuserwerundata", "webapi_getwerundataextra"} {
			payload := map[string]any{
				"api_name":         apiName,
				"data":             map[string]any{},
				"with_credentials": true,
				"from_component":   true,
			}
			result, err := a.pool.OperateWXData(ctx, current.LoginBuffer, appID, payload, current.ID, tcpProxy)
			if err != nil {
				last = map[string]any{"api_name": apiName, "error": err.Error()}
				continue
			}
			last = result
			if encrypted, iv := findWeRunEncryptedData(result); encrypted != "" && iv != "" {
				return map[string]any{
					"encryptedData": encrypted,
					"iv":            iv,
					"api_name":      apiName,
					"raw":           result,
				}, nil
			}
		}
		if last == nil {
			return nil, fmt.Errorf("应用宝底层暂未返回真实 wx.getWeRunData encryptedData/iv")
		}
		return nil, fmt.Errorf("应用宝底层暂未返回真实 wx.getWeRunData encryptedData/iv，最后响应：%v", last)
	}
	result, used, err := a.runBusinessWithProxyRetry(ctx, acc, run)
	if err != nil {
		return nil, err
	}
	return map[string]any{"openid": used.OpenID, "result": result}, nil
}

func findWeRunEncryptedData(v map[string]any) (string, string) {
	encrypted := firstNonEmptyMapString(v, "encryptedData", "encrypted_data")
	iv := firstNonEmptyMapString(v, "iv", "IV")
	if encrypted != "" && iv != "" {
		return encrypted, iv
	}
	if raw, ok := v["data"].(string); ok && strings.TrimSpace(raw) != "" {
		var inner map[string]any
		if json.Unmarshal([]byte(raw), &inner) == nil {
			return findWeRunEncryptedData(inner)
		}
	}
	for _, child := range v {
		if m, ok := child.(map[string]any); ok {
			if encrypted, iv := findWeRunEncryptedData(m); encrypted != "" && iv != "" {
				return encrypted, iv
			}
		}
	}
	return "", ""
}

func firstNonEmptyMapString(m map[string]any, keys ...string) string {
	for _, k := range keys {
		if v, ok := m[k].(string); ok && strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
