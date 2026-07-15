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
}
