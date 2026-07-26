package yybportal

import (
	"context"
	"strings"

	"github.com/cdle/xdd/models"
	"github.com/cdle/xdd/yyb"
)

// RegisterProxyLoginHooks 将 models 层 51 代理逻辑注入 yyb 扫码模块
func RegisterProxyLoginHooks() {
	yyb.SetProxyLoginHooks(
		func(opt yyb.ProxyScanOption) (string, map[string]interface{}, error) {
			return models.YybBuildProxyForLogin(models.YybProxyLoginOption{
				Enabled:    opt.Enabled,
				PackID:     opt.PackID,
				RegionCode: opt.RegionCode,
				RegionName: opt.RegionName,
			})
		},
		func() bool { return models.Config.Yyb.Proxy51Enabled },
	)
}

func toProxyScanOpt(opt models.YybProxyLoginOption) yyb.ProxyScanOption {
	return yyb.ProxyScanOption{
		Enabled:    opt.Enabled,
		PackID:     opt.PackID,
		RegionCode: opt.RegionCode,
		RegionName: opt.RegionName,
	}
}

func portalStartScanLogin(userNumber int, cost int, deduct bool, proxyOpt models.YybProxyLoginOption) (imageB64, sessionID string, meta map[string]interface{}, err error) {
	imageURI, sid, meta, err := yyb.StartPortalScanLogin(userNumber, cost, deduct, toProxyScanOpt(proxyOpt))
	if err != nil {
		return "", "", nil, err
	}
	return strings.TrimPrefix(imageURI, "data:image/jpeg;base64,"), sid, meta, nil
}

func portalPollScanLogin(sessionID string, userNumber int) (map[string]interface{}, error) {
	return yyb.PollPortalScanLogin(sessionID, userNumber)
}

// resolveYybScanCharge 确认时实时计算是否扣费（需在 StoreScanAccount 之后调用，以便使用 yyb_account_id）：
// - 库内已有 open_id / yyb_account_id → 续登，不扣
// - 新账号且库内数 < 在线微信数 → 不扣（含掉线账号占名额）
// - 新账号且名额已满 → 扣费
func resolveYybScanCharge(userNumber int, openid string, yybAccountID int64) (cost int, needCharge bool, chargeKind string) {
	if isKnownYybAccount(userNumber, openid, yybAccountID) {
		return 0, false, "relogin"
	}
	cost, free, _ := models.CalcYybScanLoginCost(userNumber)
	if free || cost <= 0 {
		return 0, false, "free_slot"
	}
	return cost, true, "new_account"
}

func portalConfirmScanLogin(sessionID string, userNumber int, clientCtx models.ClientContext) (map[string]interface{}, error) {
	loginBuffer, creds, proxyMeta, _, _, err := yyb.FinishPortalScanLogin(sessionID, userNumber)
	if err != nil {
		return nil, err
	}

	a, err := svc()
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	acc, err := a.StoreScanAccount(ctx, loginBuffer, creds, proxyMeta)
	if err != nil {
		return nil, err
	}

	openid := strings.TrimSpace(acc.OpenID)
	alreadyBound := isKnownYybAccount(userNumber, openid, acc.ID)
	cost, needCharge, chargeKind := resolveYybScanCharge(userNumber, openid, acc.ID)
	if needCharge {
		if err := ensureCoin(userNumber, cost); err != nil {
			return nil, err
		}
	}

	binding, err := bindAccount(userNumber, acc, "alive")
	if err != nil {
		return nil, err
	}

	costCharged := 0
	if needCharge {
		nick := ""
		if acc.Nickname != nil {
			nick = *acc.Nickname
		}
		remark := formatYybScanCoinRemark(chargeKind, nick, openid, acc.ID)
		if err := deductCoin(userNumber, cost, clientCtx, remark); err != nil {
			return nil, err
		}
		models.RecordClientSourceEvent(userNumber, "yyb_scan_login", clientCtx)
		costCharged = cost
	}

	view := toPortalView(ctx, *binding, a)
	return map[string]interface{}{
		"account":      view,
		"cost":         costCharged,
		"alreadyBound": alreadyBound,
	}, nil
}
