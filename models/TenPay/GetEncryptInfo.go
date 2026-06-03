package TenPay

import (
	"fmt"
	"net/url"
	"wechatdll/comm"
	"wechatdll/models"
)

type GetEncryptInfoParam struct {
	Encrypt_Userinfo string
	Encrypt_Key      string
}

func GetEncryptInfo(Wxid string) models.ResponseResult {
	D, err := comm.GetLoginata(Wxid, nil)
	if err != nil {
		return models.ResponseResult{
			Code:    -8,
			Success: false,
			Message: fmt.Sprintf("异常：%v", err.Error()),
			Data:    nil,
		}
	}
	encrypt_userinfo := url.QueryEscape(D.Deviceid_str)
	encrypt_key := url.QueryEscape(string(D.Aeskey))

	EncryptInfoParam := GetEncryptInfoParam{
		Encrypt_Userinfo: encrypt_userinfo,
		Encrypt_Key:      encrypt_key,
	}

	return models.ResponseResult{
		Code:    0,
		Success: true,
		Message: "成功",
		Data:    EncryptInfoParam,
	}
}
