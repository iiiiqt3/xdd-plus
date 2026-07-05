package yybportal

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/cdle/xdd/models"
	"github.com/cdle/xdd/yyb"
)

// AdminStatusExtras 管理端状态扩展（绑定数、协议账号数、可用数）
func AdminStatusExtras() map[string]any {
	out := map[string]any{
		"bindingCount":  0,
		"protocolCount": 0,
		"aliveCount":    0,
	}
	if !Ready() {
		return out
	}
	var bindingCount int64
	_ = db().Model(&PortalYybBinding{}).Count(&bindingCount).Error
	out["bindingCount"] = bindingCount
	a, err := svc()
	if err != nil {
		return out
	}
	accounts, err := a.ListAccounts(context.Background())
	if err != nil {
		out["protocolCount"] = 0
		return out
	}
	out["protocolCount"] = len(accounts)
	alive := 0
	for _, acc := range accounts {
		if acc.Status == nil {
			continue
		}
		st := strings.ToLower(strings.TrimSpace(*acc.Status))
		if st == "alive" || st == "online" {
			alive++
		}
	}
	out["aliveCount"] = alive
	return out
}

// AdminListProtocolAccounts 协议库全部账号（用于管理端调试，不依赖门户绑定）
func AdminListProtocolAccounts() ([]yyb.AccountPublic, error) {
	if !Ready() {
		return nil, fmt.Errorf("应用宝服务不可用")
	}
	a, err := svc()
	if err != nil {
		return nil, err
	}
	return a.ListAccounts(context.Background())
}

// AdminServeAvatar 管理端头像（不校验门户绑定）
func AdminServeAvatar(w http.ResponseWriter, r *http.Request, ref string) error {
	if !Ready() {
		return fmt.Errorf("应用宝服务不可用")
	}
	a, err := svc()
	if err != nil {
		return err
	}
	return a.ServeAccountAvatar(w, r, ref)
}

// AdminListAccounts 全站门户绑定
func AdminListAccounts() ([]AdminAccountView, error) {
	if !Ready() {
		return nil, fmt.Errorf("应用宝服务不可用")
	}
	var bindings []PortalYybBinding
	if err := db().Order("id desc").Find(&bindings).Error; err != nil {
		return nil, err
	}
	a, err := svc()
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	out := make([]AdminAccountView, 0, len(bindings))
	nickCache := map[int]string{}
	for _, b := range bindings {
		view := AdminAccountView{
			PortalAccountView: toPortalView(ctx, b, a),
			UserNumber:        b.UserNumber,
		}
		if nick, ok := nickCache[b.UserNumber]; ok {
			view.UserNick = nick
		} else {
			if acc, err := models.GetWebUserAccountByUserNumber(b.UserNumber); err == nil {
				view.UserNick = acc.Username
				nickCache[b.UserNumber] = acc.Username
			}
		}
		out = append(out, view)
	}
	return out, nil
}

// AdminDeleteAccount 管理员删除
func AdminDeleteAccount(ref string) error {
	if !Ready() {
		return fmt.Errorf("应用宝服务不可用")
	}
	a, err := svc()
	if err != nil {
		return err
	}
	acc, err := a.GetAccountPublic(context.Background(), ref)
	if err == nil && acc != nil {
		_ = a.DeleteAccount(context.Background(), strconv.FormatInt(acc.ID, 10))
		db().Unscoped().Where(&PortalYybBinding{YybAccountID: acc.ID}).Or(&PortalYybBinding{OpenID: acc.OpenID}).Delete(&PortalYybBinding{})
		return nil
	}
	db().Unscoped().Where("open_id = ? OR id = ? OR yyb_account_id = ?", ref, ref, ref).Delete(&PortalYybBinding{})
	return nil
}

// AdminRefreshAccount 刷新
func AdminRefreshAccount(ref string) (map[string]any, error) {
	a, err := svc()
	if err != nil {
		return nil, err
	}
	return a.RefreshAccount(context.Background(), ref)
}

// AdminResyncAccount 同步
func AdminResyncAccount(ref string) (map[string]any, error) {
	a, err := svc()
	if err != nil {
		return nil, err
	}
	acc, err := a.ResyncAccount(context.Background(), ref)
	if err != nil {
		return nil, err
	}
	pub := acc
	return map[string]any{"account": pub}, nil
}

// AdminWxappGetCode 调试
func AdminWxappGetCode(ref, appID string) (map[string]any, error) {
	a, err := svc()
	if err != nil {
		return nil, err
	}
	return a.WxappGetCode(context.Background(), ref, appID)
}

// AdminWxappGetPhone 调试
func AdminWxappGetPhone(ref, appID string) (map[string]any, error) {
	a, err := svc()
	if err != nil {
		return nil, err
	}
	return a.WxappGetPhoneNumber(context.Background(), ref, appID)
}

// AdminWxappOperate 调试
func AdminWxappOperate(ref, appID string, payload map[string]any) (map[string]any, error) {
	a, err := svc()
	if err != nil {
		return nil, err
	}
	return a.WxappOperateWXData(context.Background(), ref, appID, payload)
}

// AdminCreateQR 管理员扫码（不扣积分、不写绑定，仅测试）
func AdminCreateQR() (map[string]any, error) {
	a, err := svc()
	if err != nil {
		return nil, err
	}
	qr, err := a.CreateQR(context.Background(), true)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"sessionId":   qr.SessionID,
		"status":      qr.Status,
		"imageBase64": qr.ImageB64,
	}, nil
}

// AdminPollQR 轮询
func AdminPollQR(sessionID string) (map[string]any, error) {
	a, err := svc()
	if err != nil {
		return nil, err
	}
	return a.PollQR(context.Background(), sessionID)
}

// AdminConfirmQR 确认（仅保存到 yyb 本地库，不写 portal 绑定）
func AdminConfirmQR(sessionID string) (map[string]any, error) {
	a, err := svc()
	if err != nil {
		return nil, err
	}
	acc, err := a.ConfirmQR(context.Background(), sessionID)
	if err != nil {
		return nil, err
	}
	return map[string]any{"account": acc}, nil
}
