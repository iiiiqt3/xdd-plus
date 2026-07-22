package yybportal

import "github.com/cdle/xdd/models"

// RegisterProtocolGatewayHandlers 向 models 注入应用宝协议回调
func RegisterProtocolGatewayHandlers() {
	models.SetProtocolYybHandlers(
		func(openid, appID string) (map[string]interface{}, error) {
			data, err := InternalWxappGetCode(openid, appID)
			if err != nil {
				return nil, err
			}
			return data, nil
		},
		func(openid, appID string, payload map[string]interface{}) (map[string]interface{}, error) {
			p := map[string]any{}
			for k, v := range payload {
				p[k] = v
			}
			return InternalWxappOperate(openid, appID, p)
		},
		func(openid string) bool {
			return IsYybAccountAlive(openid)
		},
		func(openid string) (string, error) {
			return RefreshYybAccountLiveness(openid)
		},
	)
	models.SetProtocolYybGetPhone(func(openid, appID string) (map[string]interface{}, error) {
		data, err := InternalWxappGetPhone(openid, appID)
		if err != nil {
			return nil, err
		}
		return data, nil
	})
	models.SetProtocolYybAccountExists(func(ref string) bool {
		_, err := AccountPublic(ref)
		return err == nil
	})
	models.SetProtocolListYybBriefsFn(func(userNumber int) ([]models.ProtocolYybAccountBrief, error) {
		accounts, err := PortalListAccounts(userNumber)
		if err != nil {
			return nil, err
		}
		out := make([]models.ProtocolYybAccountBrief, 0, len(accounts))
		for _, a := range accounts {
			out = append(out, models.ProtocolYybAccountBrief{
				OpenID:   a.OpenID,
				Nickname: a.Nickname,
				Status:   a.Status,
			})
		}
		return out, nil
	})
}
