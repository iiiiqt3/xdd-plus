package yybportal

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/cdle/xdd/models"
)

// PortalStatus 门户状态
func PortalStatus(userNumber int) (map[string]any, error) {
	if !Config().Enabled {
		return map[string]any{
			"enabled": false,
			"ready":   false,
			"message": "应用宝模块未启用",
		}, nil
	}
	st := StatusPayload()
	st["scanLoginCost"] = getScanLoginCost()
	st["maxAccounts"] = getMaxAccountsPerUser()
	st["coin"] = models.GetCoin(userNumber)
	accounts, err := PortalListAccounts(userNumber)
	if err != nil {
		return nil, err
	}
	st["accounts"] = accounts
	if !Ready() {
		st["message"] = "应用宝服务暂不可用，请稍后再试"
	}
	return st, nil
}

// PortalListAccounts 用户绑定账号列表
func PortalListAccounts(userNumber int) ([]PortalAccountView, error) {
	if !Ready() {
		return nil, fmt.Errorf("应用宝服务不可用")
	}
	a, err := svc()
	if err != nil {
		return nil, err
	}
	rows, err := listBindings(userNumber)
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	out := make([]PortalAccountView, 0, len(rows))
	for _, b := range rows {
		out = append(out, toPortalView(ctx, b, a))
	}
	return out, nil
}

// PortalCreateQR 创建扫码
func PortalCreateQR(userNumber int) (map[string]any, error) {
	if !Ready() {
		return nil, fmt.Errorf("应用宝服务不可用")
	}
	cost := getScanLoginCost()
	if err := ensureCoin(userNumber, cost); err != nil {
		return nil, err
	}
	a, err := svc()
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	qr, err := a.CreateQR(ctx, true)
	if err != nil {
		return nil, err
	}
	putPendingScan(userNumber, qr.SessionID, cost, true)
	return map[string]any{
		"sessionId":     qr.SessionID,
		"status":        qr.Status,
		"imageBase64":   qr.ImageB64,
		"scanLoginCost": cost,
	}, nil
}

// PortalPollQR 轮询扫码
func PortalPollQR(userNumber int, sessionID string) (map[string]any, error) {
	if !Ready() {
		return nil, fmt.Errorf("应用宝服务不可用")
	}
	scanMu.Lock()
	p, ok := scanStore[sessionID]
	scanMu.Unlock()
	if !ok || p.UserNumber != userNumber {
		return nil, fmt.Errorf("扫码会话无效或已过期")
	}
	a, err := svc()
	if err != nil {
		return nil, err
	}
	return a.PollQR(context.Background(), sessionID)
}

// PortalConfirmQR 确认扫码并绑定
func PortalConfirmQR(userNumber int, sessionID string, clientCtx models.ClientContext) (map[string]any, error) {
	if !Ready() {
		return nil, fmt.Errorf("应用宝服务不可用")
	}
	p, err := popPendingScan(sessionID, userNumber)
	if err != nil {
		return nil, err
	}
	a, err := svc()
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	acc, err := a.ConfirmQR(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if p.DeductCoin {
		nick := ""
		if acc.Nickname != nil {
			nick = *acc.Nickname
		}
		if err := deductCoin(userNumber, p.Cost, clientCtx, fmt.Sprintf("应用宝扫码登录 %s", nick)); err != nil {
			return nil, err
		}
		models.RecordClientSourceEvent(userNumber, "yyb_scan_login", clientCtx)
	}
	binding, err := bindAccount(userNumber, acc, "alive")
	if err != nil {
		return nil, err
	}
	view := toPortalView(ctx, *binding, a)
	return map[string]any{
		"account": view,
		"cost":    p.Cost,
	}, nil
}

// PortalDeleteAccount 删除绑定与本地账号
func PortalDeleteAccount(userNumber int, ref string) error {
	if !Ready() {
		return fmt.Errorf("应用宝服务不可用")
	}
	b, err := resolveBinding(userNumber, ref)
	if err != nil {
		return err
	}
	a, err := svc()
	if err != nil {
		return err
	}
	refID := strconv.FormatInt(b.YybAccountID, 10)
	if err := a.DeleteAccount(context.Background(), refID); err != nil {
		// 本地账号可能已不存在，继续删绑定
	}
	return db().Delete(b).Error
}

// PortalRefreshAccount 刷新存活
func PortalRefreshAccount(userNumber int, ref string) (map[string]any, error) {
	b, err := resolveBinding(userNumber, ref)
	if err != nil {
		return nil, err
	}
	a, err := svc()
	if err != nil {
		return nil, err
	}
	return a.RefreshAccount(context.Background(), strconv.FormatInt(b.YybAccountID, 10))
}

// PortalResyncAccount 同步资料
func PortalResyncAccount(userNumber int, ref string) (PortalAccountView, error) {
	b, err := resolveBinding(userNumber, ref)
	if err != nil {
		return PortalAccountView{}, err
	}
	a, err := svc()
	if err != nil {
		return PortalAccountView{}, err
	}
	acc, err := a.ResyncAccount(context.Background(), strconv.FormatInt(b.YybAccountID, 10))
	if err != nil {
		return PortalAccountView{}, err
	}
	if acc.Nickname != nil {
		b.Nickname = *acc.Nickname
	}
	if acc.Status != nil {
		b.Status = *acc.Status
	}
	_ = db().Save(b)
	return toPortalView(context.Background(), *b, a), nil
}

// PortalServeAvatar 输出头像
func PortalServeAvatar(w http.ResponseWriter, r *http.Request, userNumber int, ref string) error {
	if !Ready() {
		return fmt.Errorf("应用宝服务不可用")
	}
	if _, err := resolveBinding(userNumber, ref); err != nil {
		// 管理/脚本可能用 openid 直接访问，门户必须绑定
		return err
	}
	a, err := svc()
	if err != nil {
		return err
	}
	return a.ServeAccountAvatar(w, r, ref)
}

// PortalWxappGetCode 获取小程序 code
func PortalWxappGetCode(userNumber int, ref, appID string) (map[string]any, error) {
	b, err := resolveBinding(userNumber, ref)
	if err != nil {
		return nil, err
	}
	a, err := svc()
	if err != nil {
		return nil, err
	}
	return a.WxappGetCode(context.Background(), strconv.FormatInt(b.YybAccountID, 10), strings.TrimSpace(appID))
}

// PortalWxappGetPhone 获取手机号
func PortalWxappGetPhone(userNumber int, ref, appID string) (map[string]any, error) {
	b, err := resolveBinding(userNumber, ref)
	if err != nil {
		return nil, err
	}
	a, err := svc()
	if err != nil {
		return nil, err
	}
	return a.WxappGetPhoneNumber(context.Background(), strconv.FormatInt(b.YybAccountID, 10), strings.TrimSpace(appID))
}

// PortalWxappOperate 云函数
func PortalWxappOperate(userNumber int, ref, appID string, payload map[string]any) (map[string]any, error) {
	b, err := resolveBinding(userNumber, ref)
	if err != nil {
		return nil, err
	}
	a, err := svc()
	if err != nil {
		return nil, err
	}
	return a.WxappOperateWXData(context.Background(), strconv.FormatInt(b.YybAccountID, 10), strings.TrimSpace(appID), payload)
}
