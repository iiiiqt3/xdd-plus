package yyb

import (
	"strings"
	"sync"

	"github.com/cdle/xdd/yyb/internal/httpapi"
)

var (
	globalMu sync.RWMutex
	global   *Service
)

// Service 应用宝独立服务实例
type Service struct {
	app *httpapi.App
	cfg Config
}

// Start 启动应用宝服务；失败时返回 error，调用方应捕获且不影响主程序
func Start(cfg Config) (*Service, error) {
	if cfg.ResourceRoot == "" {
		cfg.ResourceRoot = DefaultConfig().ResourceRoot
	}
	if cfg.DBFilename == "" {
		cfg.DBFilename = DefaultConfig().DBFilename
	}
	if cfg.SessionTTL == 0 {
		cfg.SessionTTL = DefaultConfig().SessionTTL
	}
	if cfg.RequestTimeout == 0 {
		cfg.RequestTimeout = DefaultConfig().RequestTimeout
	}
	if cfg.AvatarTimeout == 0 {
		cfg.AvatarTimeout = DefaultConfig().AvatarTimeout
	}
	if cfg.QRSessionTTL == 0 {
		cfg.QRSessionTTL = DefaultConfig().QRSessionTTL
	}

	app, err := httpapi.NewApp(httpapi.Config{
		ResourceRoot:           cfg.ResourceRoot,
		DBFilename:             cfg.DBFilename,
		GormDB:                 cfg.GormDB,
		TCPProxy:               cfg.TCPProxy,
		Proxy51Enabled:         cfg.Proxy51Enabled,
		Proxy51BusinessEnabled: cfg.Proxy51BusinessEnabled,
		SessionTTL:             cfg.SessionTTL,
		RequestTimeout:         cfg.RequestTimeout,
		AvatarTimeout:          cfg.AvatarTimeout,
		ScanTimeout:            cfg.ScanTimeout,
		QRSessionTTL:           cfg.QRSessionTTL,
	})
	if err != nil {
		return nil, err
	}
	svc := &Service{app: app, cfg: cfg}
	globalMu.Lock()
	global = svc
	globalMu.Unlock()
	return svc, nil
}

// Global 返回已启动的全局实例（可能为 nil）
func Global() *Service {
	globalMu.RLock()
	defer globalMu.RUnlock()
	return global
}

// App 内部 API 应用（仅 yyb 包内使用）
func (s *Service) appAPI() *httpapi.App {
	if s == nil {
		return nil
	}
	return s.app
}

// Close 释放资源
func (s *Service) Close() error {
	if s == nil || s.app == nil {
		return nil
	}
	err := s.app.Close()
	globalMu.Lock()
	if global == s {
		global = nil
	}
	globalMu.Unlock()
	return err
}

// Ready 模块是否可用
func (s *Service) Ready() bool {
	return s != nil && s.app != nil
}

// SetTCPProxy 运行时切换默认 TCP 代理
func (s *Service) SetTCPProxy(proxy string) {
	if s == nil || s.app == nil {
		return
	}
	s.app.SetTCPProxy(proxy)
	s.cfg.TCPProxy = strings.TrimSpace(proxy)
}

// SetProxy51BusinessEnabled 热更新脚本业务是否走 51 代理
func (s *Service) SetProxy51BusinessEnabled(enabled bool) {
	if s == nil || s.app == nil {
		return
	}
	s.app.SetProxy51BusinessEnabled(enabled)
	s.cfg.Proxy51BusinessEnabled = enabled
}
