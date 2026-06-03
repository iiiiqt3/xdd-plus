package Login

import (
	"fmt"
	"wechatdll/comm"
	"wechatdll/models"
)

func CacheInfo(Wxid string) models.ResponseResult {
	D, err := comm.GetLoginata(Wxid, nil)
	if err != nil || D == nil || D.Wxid == "" {
		errorMsg := fmt.Sprintf("异常：%v [%v]", "未找到登录信息", Wxid)
		if err != nil {
			errorMsg = fmt.Sprintf("异常：%v", err.Error())
		}
		return models.ResponseResult{
			Code:    -8,
			Success: false,
			Message: errorMsg,
			Data:    nil,
		}
	}

	return models.ResponseResult{
		Code:    1,
		Success: true,
		Message: "成功",
		Data:    D,
	}
}
