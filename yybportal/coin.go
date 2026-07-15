package yybportal

import (
	"fmt"
	"strings"

	"github.com/cdle/xdd/models"
)

func getScanLoginCost() int {
	// 优先读热更新后的内存配置；未单独配置时回退微信协议积分
	if models.Config.Yyb.ScanLoginCost != nil {
		return *models.Config.Yyb.ScanLoginCost
	}
	if c := Config().ScanLoginCost; c > 0 {
		return c
	}
	if models.Config.WxProtocol.ScanLoginCost > 0 {
		return models.Config.WxProtocol.ScanLoginCost
	}
	return 2000
}

func getMaxAccountsPerUser() int {
	if models.Config.Yyb.MaxAccountsPerUser > 0 {
		return models.Config.Yyb.MaxAccountsPerUser
	}
	if Config().MaxAccountsPerUser > 0 {
		return Config().MaxAccountsPerUser
	}
	return 5
}

func ensureCoin(userNumber, cost int) error {
	if cost <= 0 {
		return nil
	}
	coin := models.GetCoin(userNumber)
	if coin < cost {
		return fmt.Errorf("积分不足，无法登录（需要 %d 积分，当前 %d 积分）", cost, coin)
	}
	return nil
}

func ensureCoinForScanQR(userNumber, cost int) error {
	if cost <= 0 {
		return nil
	}
	coin := models.GetCoin(userNumber)
	if coin < cost {
		return fmt.Errorf("积分不足，无法生成二维码（需要 %d 积分，当前 %d 积分）", cost, coin)
	}
	return nil
}

func deductCoin(userNumber, cost int, clientCtx models.ClientContext, remark string) error {
	if cost <= 0 {
		return nil
	}
	if err := models.DeductCoinChecked(userNumber, cost); err != nil {
		if strings.Contains(err.Error(), "积分不足") {
			return fmt.Errorf("积分不足，无法登录（需要 %d 积分，当前 %d 积分）", cost, models.GetCoin(userNumber))
		}
		return err
	}
	models.RecordCoinLog(userNumber, -cost, "应用宝登录", remark, clientCtx.WithDefault())
	return nil
}
