package yybportal

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/cdle/xdd/models"
)

// PortalStatus 门户状态；autoCheck 为 true 时自动刷新全部绑定账号存活状态
func PortalStatus(userNumber int, autoCheck bool) (map[string]any, error) {
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
	if !Ready() {
		st["message"] = "应用宝服务暂不可用，请稍后再试"
		st["accounts"] = []PortalAccountView{}
		return st, nil
	}
	if autoCheck {
		models.Yyb().Infof("门户用户 %d 进入应用宝页，开始自动检测账号存活", userNumber)
		summary, err := portalCheckAllAccounts(userNumber)
		if err != nil {
			models.Yyb().Errorf("门户用户 %d 应用宝自动检测失败: %v", userNumber, err)
			return nil, err
		}
		st["checkSummary"] = summary
		models.Yyb().Infof("门户用户 %d 应用宝检测完成: total=%v alive=%v dead=%v failed=%v",
			userNumber, summary["total"], summary["alive"], summary["dead"], summary["failed"])
	}
	accounts, err := PortalListAccounts(userNumber)
	if err != nil {
		return nil, err
	}
	st["accounts"] = accounts
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
	rows = dedupeBindings(rows)
	ctx := context.Background()
	out := make([]PortalAccountView, 0, len(rows))
	for _, b := range rows {
		out = append(out, toPortalView(ctx, b, a))
	}
	return out, nil
}

// PortalCreateQR 创建扫码（用户选择 51 代理省市区）
// 积分预检：仅「库内尚无应用宝绑定时」按新增账号要求余额。
// 已有绑定（含失效）允许先出码——确认时旧 OpenID 续登不扣费，新 OpenID 再按名额扣费。
func PortalCreateQR(userNumber int, proxyOpt models.YybProxyLoginOption) (map[string]any, error) {
	if !Ready() {
		return nil, fmt.Errorf("应用宝服务不可用")
	}
	cost, _, hint := models.CalcYybScanLoginCost(userNumber)
	hasBinding := models.CountPortalYybBindings(userNumber) > 0
	if cost > 0 && !hasBinding {
		if err := ensureCoinForScanQR(userNumber, cost); err != nil {
			return nil, err
		}
	}
	if models.Config.Yyb.Proxy51Enabled {
		proxyOpt.Enabled = true
		if strings.TrimSpace(proxyOpt.RegionCode) == "" {
			return nil, fmt.Errorf("请选择登录地区（省/市）")
		}
	} else {
		proxyOpt.Enabled = false
	}
	imageB64, sessionID, meta, err := portalStartScanLogin(userNumber, cost, cost > 0, proxyOpt)
	if err != nil {
		return nil, err
	}
	out := map[string]any{
		"sessionId":     sessionID,
		"status":        "pending",
		"imageBase64":   strings.TrimPrefix(imageB64, "data:image/jpeg;base64,"),
		"scanLoginCost": cost,
		"scanCostHint":  hint,
	}
	if meta != nil {
		out["proxyMeta"] = meta
		if models.YybProxyShouldBypass(proxyOpt.RegionCode, proxyOpt.RegionName) {
			out["proxyBypass"] = true
		}
		out["proxyRegionCode"] = proxyOpt.RegionCode
		out["proxyRegionName"] = proxyOpt.RegionName
	}
	return out, nil
}

// PortalPollQR 轮询扫码（含代理异常自动换 IP）
func PortalPollQR(userNumber int, sessionID string) (map[string]any, error) {
	if !Ready() {
		return nil, fmt.Errorf("应用宝服务不可用")
	}
	return portalPollScanLogin(sessionID, userNumber)
}

// PortalConfirmQR 确认扫码并绑定
func PortalConfirmQR(userNumber int, sessionID string, clientCtx models.ClientContext) (map[string]any, error) {
	if !Ready() {
		return nil, fmt.Errorf("应用宝服务不可用")
	}
	return portalConfirmScanLogin(sessionID, userNumber, clientCtx)
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
	return db().Unscoped().Delete(b).Error
}

// PortalRefreshAccount 刷新存活
func PortalRefreshAccount(userNumber int, ref string) (map[string]any, error) {
	if err := ensureYybManualRefreshAllowed(fmt.Sprintf("portal:%d", userNumber), ref); err != nil {
		return nil, err
	}
	b, err := resolveBinding(userNumber, ref)
	if err != nil {
		return nil, err
	}
	a, err := svc()
	if err != nil {
		return nil, err
	}
	data, err := a.RefreshAccount(context.Background(), strconv.FormatInt(b.YybAccountID, 10))
	if err != nil {
		return nil, err
	}
	syncBindingFromRefresh(b, data)
	return data, nil
}

func syncBindingFromRefresh(b *PortalYybBinding, data map[string]any) {
	if b == nil || data == nil {
		return
	}
	changed := false
	if st, ok := data["status"].(string); ok && st != "" && b.Status != st {
		b.Status = st
		changed = true
	}
	if nick, ok := data["nickname"].(string); ok && nick != "" && b.Nickname != nick {
		b.Nickname = nick
		changed = true
	}
	if changed {
		_ = db().Save(b).Error
	}
}

// portalCheckAllAccounts 检测用户全部绑定账号存活状态
func portalCheckAllAccounts(userNumber int) (map[string]any, error) {
	rows, err := listBindings(userNumber)
	if err != nil {
		return nil, err
	}
	rows = dedupeBindings(rows)
	summary := map[string]any{
		"total": len(rows),
	}
	alive, dead, failed := 0, 0, 0
	if len(rows) == 0 {
		summary["alive"] = 0
		summary["dead"] = 0
		summary["failed"] = 0
		return summary, nil
	}
	for _, b := range rows {
		ref := strconv.FormatInt(b.ID, 10)
		if strings.TrimSpace(b.OpenID) != "" {
			ref = b.OpenID
		}
		if _, err := PortalRefreshAccount(userNumber, ref); err != nil {
			failed++
			continue
		}
		var fresh PortalYybBinding
		if db().Where("id = ?", b.ID).First(&fresh).Error == nil {
			st := strings.ToLower(strings.TrimSpace(fresh.Status))
			if st == "alive" || st == "online" {
				alive++
			} else {
				dead++
			}
		}
	}
	summary["alive"] = alive
	summary["dead"] = dead
	summary["failed"] = failed
	return summary, nil
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

// PortalClaimAccount 认领已存在于应用宝库、但未绑定门户的账号
func PortalClaimAccount(userNumber int, ref string) (PortalAccountView, error) {
	if !Ready() {
		return PortalAccountView{}, fmt.Errorf("应用宝服务不可用")
	}
	a, err := svc()
	if err != nil {
		return PortalAccountView{}, err
	}
	ctx := context.Background()
	acc, err := a.GetAccountPublic(ctx, strings.TrimSpace(ref))
	if err != nil || acc == nil {
		return PortalAccountView{}, fmt.Errorf("应用宝中未找到该账号，请确认 openid 或账号 ID")
	}
	if isOpenIDBoundToOther(userNumber, acc.OpenID) {
		return PortalAccountView{}, fmt.Errorf("该账号已被其他门户用户绑定")
	}
	var count int64
	db().Model(&PortalYybBinding{}).Where(&PortalYybBinding{UserNumber: userNumber}).Count(&count)
	if int(count) >= getMaxAccountsPerUser() {
		return PortalAccountView{}, fmt.Errorf("已达账号上限（%d 个）", getMaxAccountsPerUser())
	}
	binding, err := bindAccount(userNumber, acc, "alive")
	if err != nil {
		return PortalAccountView{}, err
	}
	return toPortalView(ctx, *binding, a), nil
}

func PortalWxappGetCode(userNumber int, ref, appID string) (map[string]any, error) {
	b, err := resolveBinding(userNumber, ref)
	if err != nil {
		return nil, fmt.Errorf("%w，请从账号列表选择", err)
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
		return nil, fmt.Errorf("%w，请从账号列表选择", err)
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
		return nil, fmt.Errorf("%w，请从账号列表选择", err)
	}
	a, err := svc()
	if err != nil {
		return nil, err
	}
	return a.WxappOperateWXData(context.Background(), strconv.FormatInt(b.YybAccountID, 10), strings.TrimSpace(appID), payload)
}
