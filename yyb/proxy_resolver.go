package yyb

import (
	"context"

	"github.com/cdle/xdd/yyb/internal/store"
)

// AccountProxyResolver 按账号 credentials 解析 SOCKS5（由 yybportal 注入 models 逻辑）
type AccountProxyResolver func(credentials map[string]any) (proxy string, updated map[string]any, changed bool)

// SetAccountProxyResolver 注册账号级 51 代理解析
func (s *Service) SetAccountProxyResolver(r AccountProxyResolver) {
	if s == nil || s.app == nil {
		return
	}
	s.app.SetAccountTCPProxyResolver(func(ctx context.Context, acc *store.WechatAccount) (string, bool, map[string]any) {
		if r == nil || acc == nil {
			if s.cfg.Proxy51Enabled {
				return "", false, nil
			}
			return s.cfg.TCPProxy, false, nil
		}
		proxy, updated, changed := r(acc.Credentials)
		return proxy, changed, updated
	})
}
