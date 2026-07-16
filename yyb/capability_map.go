package yyb

import (
	"context"
	"strings"

	"github.com/cdle/xdd/yyb/internal/protocol"
)

// OfficialCGIMap 公众号/通用 CGI（map 入参，供网关层调用）
func (s *Service) OfficialCGIMap(ctx context.Context, ref string, body map[string]any) (map[string]any, error) {
	if s == nil || s.app == nil {
		return nil, errNotReady
	}
	p, _ := body["payload"].(map[string]any)
	req := protocol.OfficialCGIRequest{
		CGIURL:       pickStr(body, "cgi_url"),
		CmdID:        pickUint64(body["cmd_id"]),
		AppID:        pickStr(body, "app_id", "appid"),
		BodyKind:     pickStr(body, "body_kind"),
		HostAppID:    pickStr(body, "host_app_id"),
		Payload:      p,
		RawBase64:    pickStr(body, "raw_base64"),
		CookieBase64: pickStr(body, "cookie_base64"),
	}
	if req.Payload == nil {
		req.Payload = body
	}
	return s.app.OfficialCGI(ctx, strings.TrimSpace(ref), req)
}

// TenPayCGIMap TenPay CGI
func (s *Service) TenPayCGIMap(ctx context.Context, ref string, body map[string]any) (map[string]any, error) {
	if s == nil || s.app == nil {
		return nil, errNotReady
	}
	p, _ := body["payload"].(map[string]any)
	if p == nil {
		p = body
	}
	req := protocol.TenPayRequest{
		CGICmd:      pickUint64(body["cgi_cmd"]),
		ReqText:     pickStr(body, "req_text", "ReqText", "tenpay_url", "TenpayUrl"),
		ReqTextWx:   pickStr(body, "req_text_wx", "ReqTextWx"),
		RawBase64:   pickStr(body, "raw_base64"),
		HostAppID:   pickStr(body, "host_app_id"),
		SessionMode: pickStr(body, "session_mode"),
		Payload:     p,
	}
	return s.app.TenPayCGI(ctx, strings.TrimSpace(ref), req)
}

// RuntimeSessionMap runtime session
func (s *Service) RuntimeSessionMap(ctx context.Context, ref, appID string, payload map[string]any) (map[string]any, error) {
	if s == nil || s.app == nil {
		return nil, errNotReady
	}
	return s.app.RuntimeSession(ctx, strings.TrimSpace(ref), strings.TrimSpace(appID), payload)
}

// UpdateStepMap 刷步
func (s *Service) UpdateStepMap(ctx context.Context, ref string, body map[string]any) (map[string]any, error) {
	if s == nil || s.app == nil {
		return nil, errNotReady
	}
	return s.app.UpdateStep(ctx, strings.TrimSpace(ref), stepReqFromMap(body))
}

// ReportMotionMap 上报运动
func (s *Service) ReportMotionMap(ctx context.Context, ref string, body map[string]any) (map[string]any, error) {
	if s == nil || s.app == nil {
		return nil, errNotReady
	}
	return s.app.ReportMotion(ctx, strings.TrimSpace(ref), stepReqFromMap(body))
}

// GetBoundHardDevicesMap 绑定硬件
func (s *Service) GetBoundHardDevicesMap(ctx context.Context, ref string, body map[string]any) (map[string]any, error) {
	if s == nil || s.app == nil {
		return nil, errNotReady
	}
	return s.app.GetBoundHardDevices(ctx, strings.TrimSpace(ref), stepReqFromMap(body))
}

// GetWeRunDataMap 微信运动
func (s *Service) GetWeRunDataMap(ctx context.Context, ref, appID string) (map[string]any, error) {
	if s == nil || s.app == nil {
		return nil, errNotReady
	}
	return s.app.GetWeRunData(ctx, strings.TrimSpace(ref), strings.TrimSpace(appID))
}

func stepReqFromMap(body map[string]any) protocol.StepRequest {
	payload, _ := body["payload"].(map[string]any)
	if payload == nil {
		payload = body
	}
	return protocol.StepRequest{
		Number:      pickInt64(body, payload, "number", "Number", "step", "Step", "step_count", "StepCount"),
		DeviceID:    pickStrMaps(body, payload, "device_id", "DeviceID", "DeviceId", "deviceId"),
		DeviceType:  pickStrMaps(body, payload, "device_type", "DeviceType", "deviceType"),
		AppName:     pickStrMaps(body, payload, "app_name", "AppName", "appname"),
		BundleID:    pickStrMaps(body, payload, "bundle_id", "BundleID", "bundleid"),
		RawBase64:   pickStrMaps(body, payload, "raw_base64", "RawBase64"),
		HostAppID:   pickStrMaps(body, payload, "host_app_id", "HostAppID"),
		SessionMode: pickStrMaps(body, payload, "session_mode", "SessionMode"),
		Payload:     payload,
		SkipLookup:  pickBoolMaps(body, payload, "skip_lookup", "SkipLookup"),
	}
}

func pickStr(m map[string]any, keys ...string) string {
	for _, k := range keys {
		if v, ok := m[k].(string); ok && strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func pickStrMaps(maps ...any) string {
	var keys []string
	var bodies []map[string]any
	for _, item := range maps {
		switch x := item.(type) {
		case map[string]any:
			bodies = append(bodies, x)
		case string:
			keys = append(keys, x)
		}
	}
	for _, k := range keys {
		for _, m := range bodies {
			if v, ok := m[k].(string); ok && strings.TrimSpace(v) != "" {
				return strings.TrimSpace(v)
			}
		}
	}
	return ""
}

func pickInt64(maps ...any) int64 {
	var keys []string
	var bodies []map[string]any
	for _, item := range maps {
		switch x := item.(type) {
		case map[string]any:
			bodies = append(bodies, x)
		case string:
			keys = append(keys, x)
		}
	}
	for _, k := range keys {
		for _, m := range bodies {
			switch v := m[k].(type) {
			case int:
				return int64(v)
			case int64:
				return v
			case float64:
				return int64(v)
			}
		}
	}
	return 0
}

func pickBoolMaps(maps ...any) bool {
	var keys []string
	var bodies []map[string]any
	for _, item := range maps {
		switch x := item.(type) {
		case map[string]any:
			bodies = append(bodies, x)
		case string:
			keys = append(keys, x)
		}
	}
	for _, k := range keys {
		for _, m := range bodies {
			switch v := m[k].(type) {
			case bool:
				return v
			case string:
				if strings.EqualFold(strings.TrimSpace(v), "true") || strings.TrimSpace(v) == "1" {
					return true
				}
			}
		}
	}
	return false
}

func pickUint64(v any) uint64 {
	switch x := v.(type) {
	case int:
		return uint64(x)
	case int64:
		return uint64(x)
	case float64:
		return uint64(x)
	default:
		return 0
	}
}
