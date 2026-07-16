package yybportal

import (
	"context"
	"fmt"
	"strings"

	"github.com/cdle/xdd/yyb"
)

// InternalWxappGetCode 内部取小程序 code（cron/自动刷新，不校验门户绑定）
func InternalWxappGetCode(ref, appID string) (map[string]any, error) {
	a, err := svc()
	if err != nil {
		return nil, err
	}
	return a.WxappGetCode(context.Background(), strings.TrimSpace(ref), strings.TrimSpace(appID))
}

// InternalWxappGetPhone 内部取小程序手机号
func InternalWxappGetPhone(ref, appID string) (map[string]any, error) {
	a, err := svc()
	if err != nil {
		return nil, err
	}
	return a.WxappGetPhoneNumber(context.Background(), strings.TrimSpace(ref), strings.TrimSpace(appID))
}

// InternalWxappOperate 内部云函数调用
func InternalWxappOperate(ref, appID string, payload map[string]any) (map[string]any, error) {
	a, err := svc()
	if err != nil {
		return nil, err
	}
	return a.WxappOperateWXData(context.Background(), strings.TrimSpace(ref), strings.TrimSpace(appID), payload)
}

// IsYybAccountAlive 协议账号是否可用
func IsYybAccountAlive(ref string) bool {
	st := accountStatus(ref)
	return st == "alive" || st == "online"
}

func accountStatus(ref string) string {
	a, err := svc()
	if err != nil {
		return ""
	}
	acc, err := a.GetAccountPublic(context.Background(), strings.TrimSpace(ref))
	if err != nil || acc == nil || acc.Status == nil {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(*acc.Status))
}

// ListUserAliveAccounts 用户已绑定且可用的应用宝账号
func ListUserAliveAccounts(userNumber int) ([]PortalAccountView, error) {
	if !Ready() {
		return nil, fmt.Errorf("应用宝服务不可用")
	}
	rows, err := PortalListAccounts(userNumber)
	if err != nil {
		return nil, err
	}
	out := make([]PortalAccountView, 0, len(rows))
	for _, row := range rows {
		st := strings.ToLower(strings.TrimSpace(row.Status))
		if st == "alive" || st == "online" {
			out = append(out, row)
		}
	}
	return out, nil
}

// ListAllBindings 全站门户绑定（掉线检测用）
func ListAllBindings() ([]PortalYybBinding, error) {
	var rows []PortalYybBinding
	err := db().Order("id asc").Find(&rows).Error
	return rows, err
}

// RefreshYybAccountLiveness 刷新协议账号存活状态
func RefreshYybAccountLiveness(ref string) (string, error) {
	a, err := svc()
	if err != nil {
		return "", err
	}
	data, err := a.RefreshAccount(context.Background(), strings.TrimSpace(ref))
	if err != nil {
		return "", err
	}
	if st, ok := data["status"].(string); ok && st != "" {
		return st, nil
	}
	return accountStatus(ref), nil
}

// ServiceReady 模块是否可用
func ServiceReady() bool { return Ready() }

// AccountPublic 查询协议账号
func AccountPublic(ref string) (*yyb.AccountPublic, error) {
	a, err := svc()
	if err != nil {
		return nil, err
	}
	return a.GetAccountPublic(context.Background(), strings.TrimSpace(ref))
}
