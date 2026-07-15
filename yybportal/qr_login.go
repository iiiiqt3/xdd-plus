package yybportal

import (
	"context"
	"fmt"
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

// resolveYybScanCharge 确认时实时计算是否扣费：
// - 库内已有 OpenID 续登录 → 不扣
// - 新 OpenID 且库内数 < 在线微信数 → 不扣（含掉线账号占名额）
// - 新 OpenID 且名额已满 → 扣费
func resolveYybScanCharge(userNumber int, openid string) (cost int, needCharge bool) {
	openid = strings.TrimSpace(openid)
	if openid != "" && isUserBoundOpenID(userNumber, openid) {
		return 0, false
	}
	cost, free, _ := models.CalcYybScanLoginCost(userNumber)
	if free || cost <= 0 {
		return 0, false
	}
	return cost, true
}

func portalConfirmScanLogin(sessionID string, userNumber int, clientCtx models.ClientContext) (map[string]interface{}, error) {
	loginBuffer, creds, proxyMeta, _, _, err := yyb.FinishPortalScanLogin(sessionID, userNumber)
	if err != nil {
		return nil, err
	}

	openid := strings.TrimSpace(creds.OpenID)
	alreadyBound := openid != "" && isUserBoundOpenID(userNumber, openid)
	cost, needCharge := resolveYybScanCharge(userNumber, openid)
	if needCharge {
		if err := ensureCoin(userNumber, cost); err != nil {
			return nil, err
		}
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

	costCharged := 0
	if needCharge {
		nick := ""
		if acc.Nickname != nil {
			nick = *acc.Nickname
		}
		if err := deductCoin(userNumber, cost, clientCtx, fmt.Sprintf("应用宝扫码登录 %s", nick)); err != nil {
			return nil, err
		}
		models.RecordClientSourceEvent(userNumber, "yyb_scan_login", clientCtx)
		costCharged = cost
	}

	binding, err := bindAccount(userNumber, acc, "alive")
	if err != nil {
		return nil, err
	}
	view := toPortalView(ctx, *binding, a)
	return map[string]interface{}{
		"account":      view,
		"cost":         costCharged,
		"alreadyBound": alreadyBound,
	}, nil
}
