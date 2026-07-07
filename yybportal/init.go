package yybportal

import (
	"sync"

	"github.com/cdle/xdd/models"
	"github.com/cdle/xdd/yyb"
)

// ModuleConfig 门户层配置（从 models.Config 映射）
type ModuleConfig struct {
	Enabled            bool
	ResourceRoot       string
	DBFilename         string
	TCPProxy           string
	ScanLoginCost      int
	MaxAccountsPerUser int
	APIToken           string
	ExposeInternalAPI  bool
}

var (
	mu       sync.RWMutex
	cfg      ModuleConfig
	yybSvc   *yyb.Service
	ready    bool
	initErr  error
)

// Init 初始化应用宝模块；失败不 panic
func Init(c ModuleConfig) error {
	mu.Lock()
	defer mu.Unlock()
	cfg = c
	if !c.Enabled {
		ready = false
		yybSvc = nil
		initErr = nil
		return nil
	}
	if c.MaxAccountsPerUser <= 0 {
		c.MaxAccountsPerUser = 5
	}
	cfg = c

	if err := migrate(); err != nil {
		ready = false
		initErr = err
		return err
	}

	s, err := yyb.Start(yyb.Config{
		Enabled:           true,
		ResourceRoot:      c.ResourceRoot,
		DBFilename:        c.DBFilename,
		GormDB:            models.GormDB(),
		TCPProxy:          c.TCPProxy,
		ExposeInternalAPI: c.ExposeInternalAPI,
	})
	if err != nil {
		ready = false
		yybSvc = nil
		initErr = err
		return err
	}
	yybSvc = s
	ready = true
	initErr = nil
	return nil
}

// Ready 模块是否可用
func Ready() bool {
	mu.RLock()
	defer mu.RUnlock()
	return ready && yybSvc != nil && yybSvc.Ready()
}

// Service 获取服务实例
func Service() *yyb.Service {
	mu.RLock()
	defer mu.RUnlock()
	return yybSvc
}

// Config 当前配置
func Config() ModuleConfig {
	mu.RLock()
	defer mu.RUnlock()
	return cfg
}

// LastError 启动错误
func LastError() error {
	mu.RLock()
	defer mu.RUnlock()
	return initErr
}

// StatusPayload 状态信息
func StatusPayload() map[string]any {
	mu.RLock()
	defer mu.RUnlock()
	out := map[string]any{
		"enabled": cfg.Enabled,
		"ready":   ready && yybSvc != nil,
	}
	if initErr != nil {
		out["error"] = initErr.Error()
	}
	return out
}
