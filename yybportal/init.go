package yybportal

import (
	"sync"
	"sync/atomic"
	"time"

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
}

var (
	mu              sync.RWMutex
	cfg             ModuleConfig
	yybSvc          *yyb.Service
	ready           bool
	initErr         error
	initRunning     atomic.Bool
	registerHookOnce sync.Once
)

// Init 初始化应用宝模块；失败不 panic
func Init(c ModuleConfig) error {
	if c.MaxAccountsPerUser <= 0 {
		c.MaxAccountsPerUser = 5
	}

	registerHookOnce.Do(func() {
		models.RegisterConfigReloadHook(RefreshConfigFromModels)
	})

	if !c.Enabled {
		mu.Lock()
		cfg = c
		ready = false
		yybSvc = nil
		initErr = nil
		mu.Unlock()
		models.Yyb().Infof("应用宝模块未启用（请在 conf/config.yaml 设置 yyb.enabled: true）")
		return nil
	}

	initRunning.Store(true)
	defer initRunning.Store(false)

	models.Yyb().Infof("应用宝 migrate 开始 resource=%s", c.ResourceRoot)
	t0 := time.Now()
	if err := migrate(); err != nil {
		mu.Lock()
		cfg = c
		ready = false
		yybSvc = nil
		initErr = err
		mu.Unlock()
		models.Yyb().Errorf("应用宝 migrate 失败(%v): %v", time.Since(t0).Round(time.Millisecond), err)
		return err
	}
	models.Yyb().Infof("应用宝 migrate 完成 耗时=%v", time.Since(t0).Round(time.Millisecond))

	models.Yyb().Infof("应用宝核心服务启动中 db=%s proxy51=%v", c.DBFilename, models.Config.Yyb.Proxy51Enabled)
	t1 := time.Now()
	s, err := yyb.Start(yyb.Config{
		Enabled:           true,
		ResourceRoot:      c.ResourceRoot,
		DBFilename:        c.DBFilename,
		GormDB:            models.GormDB(),
		TCPProxy:          c.TCPProxy,
		Proxy51Enabled:    models.Config.Yyb.Proxy51Enabled,
	})
	if err != nil {
		mu.Lock()
		cfg = c
		ready = false
		yybSvc = nil
		initErr = err
		mu.Unlock()
		models.Yyb().Errorf("应用宝核心服务启动失败(%v): %v", time.Since(t1).Round(time.Millisecond), err)
		return err
	}
	models.Yyb().Infof("应用宝核心服务启动完成 耗时=%v", time.Since(t1).Round(time.Millisecond))

	mu.Lock()
	cfg = c
	yybSvc = s
	ready = true
	initErr = nil
	mu.Unlock()

	RegisterAccountProxyResolver()
	RegisterProxyLoginHooks()
	RegisterProtocolGatewayHandlers()
	startJdCron()
	models.Yyb().Infof("应用宝模块初始化完成 ready=%v 总耗时=%v", Ready(), time.Since(t0).Round(time.Millisecond))
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

// RefreshConfigFromModels 热更新可即时生效的配置项（扫码积分、账号上限等）
func RefreshConfigFromModels() {
	if initRunning.Load() {
		return
	}
	c := ModuleConfigFromModels()
	mu.Lock()
	defer mu.Unlock()
	cfg.ScanLoginCost = c.ScanLoginCost
	cfg.MaxAccountsPerUser = c.MaxAccountsPerUser
	cfg.APIToken = c.APIToken
	cfg.TCPProxy = c.TCPProxy
	cfg.ResourceRoot = c.ResourceRoot
	cfg.DBFilename = c.DBFilename
	cfg.Enabled = c.Enabled
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
	if initRunning.Load() {
		out["initializing"] = true
	}
	return out
}
