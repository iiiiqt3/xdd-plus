package wxface

import "wechatdll/srv"

// IWXConnect 微信链接接口
type IWXConnect interface {
	// 开启
	Start() error
	// 发送心跳
	SendHeartBeat() error
	// 关闭
	Stop()
	// 获取微信帐号信息
	GetWXAccount() *srv.WXAccount
	// 等待 waitTimes后发送心跳包
	SendHeartBeatWaitingSeconds(seconds uint32)
}
