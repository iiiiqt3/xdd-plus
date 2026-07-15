package yyb

import (
	"context"
	"fmt"

	"github.com/cdle/xdd/yyb/internal/store"
)

// AccountProxyResolver 按账号 credentials 解析 SOCKS5（由 yybportal 注入 models 逻辑）
type AccountProxyResolver func(credentials map[string]any) (proxy string, updated map[string]any, changed bool)

// AccountProxyForceRefresher 强制重提账号同地区短效代理（代理重试循环用）
type AccountProxyForceRefresher func(credentials map[string]any) (proxy string, updated map[string]any, err error)

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

// SetAccountProxyForceRefresher 注册强制换代理（与 xdd-g yybForceRefreshAccountProxy 一致）
func (s *Service) SetAccountProxyForceRefresher(r AccountProxyForceRefresher) {
	if s == nil || s.app == nil {
		return
	}
	s.app.SetAccountProxyForceRefresher(func(credentials map[string]any) (string, map[string]any, error) {
		if r == nil {
			return "", nil, fmt.Errorf("代理强制刷新未注册")
		}
		return r(credentials)
	})
}
