package wxface

import (
	"wechatdll/Cilent/mm"
	"wechatdll/models"
	"wechatdll/models/Msg"
)

// IWXConnect 微信链接接口
type IWXModels interface {
	// 消息同步接口
	MsgSync(Data Msg.SyncParam) models.ResponseResult
	// 短链接心跳接口
	LoginHeartBeat(Wxid string) (models.ResponseResult, *mm.HeartBeatResponse)
	// 长连接心跳接口
	LoginHeartBeatLong(Wxid string) (models.ResponseResult, *mm.HeartBeatResponse)
	// 二次登录接口
	LoginSecautoauth(Wxid string) (models.ResponseResult, *mm.UnifyAuthResponse)
}
