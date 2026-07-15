package yybportal

import (
	"fmt"

	"github.com/cdle/xdd/models"
)

// RegisterAccountProxyResolver 注册账号级 51 代理解析（与 xdd-g 一致）
func RegisterAccountProxyResolver() {
	s := Service()
	if s == nil {
		return
	}
	s.SetAccountProxyResolver(func(credentials map[string]any) (string, map[string]any, bool) {
		if !models.Config.Yyb.Proxy51Enabled {
			return models.Config.Yyb.TCPProxy, nil, false
		}
		return models.YybProxyTCPForCredentials(credentials)
	})
	s.SetAccountProxyForceRefresher(func(credentials map[string]any) (string, map[string]any, error) {
		if !models.Config.Yyb.Proxy51Enabled {
			return "", nil, fmt.Errorf("51 代理未启用")
		}
		return models.YybForceRefreshAccountProxy(credentials)
	})
}
