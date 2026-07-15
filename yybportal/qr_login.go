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

func portalConfirmScanLogin(sessionID string, userNumber int, clientCtx models.ClientContext) (map[string]interface{}, error) {
	loginBuffer, creds, proxyMeta, cost, deduct, err := yyb.FinishPortalScanLogin(sessionID, userNumber)
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

	everBound := hasEverBoundOpenID(userNumber, acc.OpenID)
	costCharged := 0
	if deduct && !everBound {
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
		"alreadyBound": everBound,
	}, nil
}
