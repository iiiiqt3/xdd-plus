package yybportal

import (
	"path/filepath"

	"github.com/cdle/xdd/models"
)

func yybScanLoginCostFromConfig(c models.YybConfig) int {
	if c.ScanLoginCost != nil {
		return *c.ScanLoginCost
	}
	if models.Config.WxProtocol.ScanLoginCost > 0 {
		return models.Config.WxProtocol.ScanLoginCost
	}
	return 2000
}

// ModuleConfigFromModels 从全局配置构建模块配置
func ModuleConfigFromModels() ModuleConfig {
	c := models.Config.Yyb
	root := c.ResourceRoot
	if root == "" {
		root = filepath.Join(models.ExecPath, "yyb/resource")
	}
	dbFile := c.DBFilename
	if dbFile == "" {
		dbFile = filepath.Join(models.ExecPath, "yyb/resource", "yyb.db")
	} else if !filepath.IsAbs(dbFile) {
		dbFile = filepath.Join(models.ExecPath, dbFile)
	}
	return ModuleConfig{
		Enabled:            c.Enabled,
		ResourceRoot:       root,
		DBFilename:         dbFile,
		TCPProxy:           c.TCPProxy,
		ScanLoginCost:      yybScanLoginCostFromConfig(c),
		MaxAccountsPerUser: c.MaxAccountsPerUser,
		APIToken:           c.APIToken,
	}
}

// AdminConfigView 管理后台配置展示
func AdminConfigView() map[string]any {
	c := Config()
	yyb := models.Config.Yyb
	models.NormalizeYybConfig(&yyb)
	return map[string]any{
		"enabled":            c.Enabled,
		"ready":              Ready(),
		"resourceRoot":       c.ResourceRoot,
		"dbFilename":         "xdd 主库 (yyb_wechat_accounts / yyb_sessions / yyb_features)",
		"tcpProxy":           c.TCPProxy,
		"scanLoginCost":      getScanLoginCost(),
		"maxAccountsPerUser": getMaxAccountsPerUser(),
		"hasApiToken":        c.APIToken != "",
		"proxy51BusinessEnabled": yyb.Proxy51BusinessEnabled,
		"livenessCheckEnabled":        models.YybLivenessCheckEnabled(yyb),
		"livenessCheckTime":           yyb.LivenessCheckTime,
		"livenessCheckIntervalSec":    yyb.LivenessCheckIntervalSec,
		"livenessManualCooldownMin":   yyb.LivenessManualCooldownMin,
		"livenessSkipIfCheckedWithinHours": yyb.LivenessSkipIfCheckedWithinHours,
	}
}
