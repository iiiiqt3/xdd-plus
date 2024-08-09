package models

import (
	"github.com/beego/beego/v2/core/logs"
)

func LoginSelect(sender *Sender, msg chan string) {
	for {
		n, ok := <-msg
		//说明发送方关闭了channel
		if !ok {
			break
		}
		//请选择登录渠道:
		// 1:兔子京东扫码
		// 2:Nolan京东扫码
		// 3:BBK京东扫码
		// 4:BBK微信扫码
		switch n {
		case "扫码登录", "1":
			loginList[sender.UserID] = nil
			if sysConfig.RabbitUrl == "" || sysConfig.RabbitApiToken == "" || sysConfig.RabbitToken == "" {
				logs.Error("RabbitUrl or RabbitToken is empty")
				sender.Reply("渠道尚未配置")
				return
			}
			RabbitGetJdQrImg(sender)
		//case "Nolan京东扫码", "2":
		//	loginList[sender.UserID] = nil
		//	if sysConfig.NolanUrl == "" || sysConfig.NolanToken == "" {
		//		logs.Error("NolanUrl or NolanToken is empty")
		//		sender.Reply("渠道尚未配置")
		//		return
		//	}
		//	NolanGetJdQrImg(sender)
		//case "BBK京东扫码", "3":
		//	loginList[sender.UserID] = nil
		//	if sysConfig.BBKJdUrl == "" {
		//		logs.Error("BBKJdUrl is empty")
		//		sender.Reply("渠道尚未配置")
		//		return
		//	}
		//	BBKGetJdQrImg(sender)
		//case "BBK微信扫码", "4":
		//	loginList[sender.UserID] = nil
		//	if sysConfig.BBKWxUrl == "" {
		//		logs.Error("BBKWxUrl is empty")
		//		sender.Reply("渠道尚未配置")
		//		return
		//	}
		//	BBKGetWxQrImg(sender)
		case "2", "短信登录":
			loginList[sender.UserID] = nil
			c2 := make(chan string)
			smsList[sender.UserID] = c2
			go SmsSelect(sender, c2, "Rabbit")
		//case "6", "Pro短信":
		//	loginList[sender.UserID] = nil
		//	c2 := make(chan string)
		//	smsList[sender.UserID] = c2
		//	sender.Reply("请输入手机号")
		//	go SmsSelect(sender, c2, "Nolan")
		case "q":
			loginList[sender.UserID] = nil
			sender.Reply("您已退出登录流程")
			close(msg)
		default:
			sender.Reply("无匹配渠道，如需回复'q'退出登录流程")
		}

	}
}
