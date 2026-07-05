package yybportal

import (
	"path/filepath"

	"github.com/cdle/xdd/models"
)

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
	token := c.APIToken
	if token == "" {
		token = models.Config.ApiToken
	}
	return ModuleConfig{
		Enabled:            c.Enabled,
		ResourceRoot:       root,
		DBFilename:         dbFile,
		TCPProxy:           c.TCPProxy,
		ScanLoginCost:      c.ScanLoginCost,
		MaxAccountsPerUser: c.MaxAccountsPerUser,
		APIToken:           token,
		ExposeInternalAPI:  c.ExposeInternalAPI,
	}
}

// AdminConfigView 管理后台配置展示
func AdminConfigView() map[string]any {
	c := Config()
	return map[string]any{
		"enabled":            c.Enabled,
		"ready":              Ready(),
		"resourceRoot":       c.ResourceRoot,
		"dbFilename":         c.DBFilename,
		"tcpProxy":           c.TCPProxy,
		"scanLoginCost":      getScanLoginCost(),
		"maxAccountsPerUser": getMaxAccountsPerUser(),
		"hasApiToken":        c.APIToken != "",
		"exposeInternalApi":  c.ExposeInternalAPI,
	}
}
