package yyb

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

// AccountProxyRegion 返回账号当前保存的代理城市（无代理时 name 为「直连」）
func (s *Service) AccountProxyRegion(ctx context.Context, ref string) (code string, name string) {
	if s == nil || s.app == nil {
		return "", "直连"
	}
	acc, err := s.app.GetAccount(ctx, ref)
	if err != nil || acc == nil {
		return "", "直连"
	}
	return proxyRegionFromCredentials(acc.Credentials)
}

func proxyRegionFromCredentials(cred map[string]any) (string, string) {
	if cred == nil {
		return "", "直连"
	}
	name := strings.TrimSpace(yybAnyString(cred["yyb_proxy_region_name"]))
	code := strings.TrimSpace(yybAnyString(cred["yyb_proxy_region_code"]))
	if yybAnyBool(cred["yyb_proxy_bypass"]) {
		if name != "" {
			return code, name
		}
		return "", "直连"
	}
	if !yybAnyBool(cred["yyb_proxy_enabled"]) && name == "" {
		return "", "直连"
	}
	if name == "" {
		if code != "" {
			return code, code
		}
		return "", "直连"
	}
	return code, name
}

func yybAnyString(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case []byte:
		return string(t)
	case float64:
		return strconv.FormatInt(int64(t), 10)
	case int:
		return strconv.Itoa(t)
	case int64:
		return strconv.FormatInt(t, 10)
	default:
		if v == nil {
			return ""
		}
		return strings.TrimSpace(fmt.Sprint(v))
	}
}

func yybAnyBool(v any) bool {
	switch t := v.(type) {
	case bool:
		return t
	case string:
		s := strings.TrimSpace(strings.ToLower(t))
		return s == "1" || s == "true" || s == "yes"
	case float64:
		return t != 0
	case int:
		return t != 0
	case int64:
		return t != 0
	default:
		return false
	}
}
