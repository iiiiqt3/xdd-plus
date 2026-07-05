package yybportal

import (
	"context"
	"fmt"
	"strings"
)

// CheckScriptToken 校验青龙/脚本 Token
func CheckScriptToken(token string) bool {
	cfg := Config()
	if cfg.APIToken == "" {
		return false
	}
	return strings.TrimSpace(token) == cfg.APIToken
}

// ScriptListAccounts 脚本列出全部 yyb 账号（不含 portal 用户映射）
func ScriptListAccounts() (any, error) {
	if !Ready() {
		return nil, fmt.Errorf("应用宝服务不可用")
	}
	a, err := svc()
	if err != nil {
		return nil, err
	}
	return a.ListAccounts(context.Background())
}

// ScriptWxappGetCode 脚本获取 code
func ScriptWxappGetCode(ref, appID string) (map[string]any, error) {
	a, err := svc()
	if err != nil {
		return nil, err
	}
	return a.WxappGetCode(context.Background(), ref, strings.TrimSpace(appID))
}

// ScriptWxappGetPhone 脚本获取手机号
func ScriptWxappGetPhone(ref, appID string) (map[string]any, error) {
	a, err := svc()
	if err != nil {
		return nil, err
	}
	return a.WxappGetPhoneNumber(context.Background(), ref, strings.TrimSpace(appID))
}

// ScriptWxappOperate 脚本云函数
func ScriptWxappOperate(ref, appID string, payload map[string]any) (map[string]any, error) {
	a, err := svc()
	if err != nil {
		return nil, err
	}
	return a.WxappOperateWXData(context.Background(), ref, strings.TrimSpace(appID), payload)
}

// ScriptRefreshAccount 脚本刷新存活
func ScriptRefreshAccount(ref string) (map[string]any, error) {
	a, err := svc()
	if err != nil {
		return nil, err
	}
	return a.RefreshAccount(context.Background(), ref)
}
