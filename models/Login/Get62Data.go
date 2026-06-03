package Login

import (
	"fmt"
	"wechatdll/clientsdk/baseutils"
	"wechatdll/comm"
)

func Get62Data(Wxid string) string {
	D, err := comm.GetLoginata(Wxid, nil)
	if err != nil || D == nil || D.Wxid == "" {
		errorMsg := fmt.Sprintf("异常：%v [%v]", "未找到登录信息", Wxid)
		if err != nil {
			errorMsg = fmt.Sprintf("异常：%v", err.Error())
		}
		return errorMsg
	}
	return baseutils.Get62Data(D.Deviceid_str)
}

func GetA16Data(Wxid string) string {
	D, err := comm.GetLoginata(Wxid, nil)
	if err != nil || D == nil || D.Wxid == "" {
		errorMsg := fmt.Sprintf("异常：%v [%v]", "未找到登录信息", Wxid)
		if err != nil {
			errorMsg = fmt.Sprintf("异常：%v", err.Error())
		}
		return errorMsg
	}
	return baseutils.GetA16Data(D.Deviceid_str)
}
