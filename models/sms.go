package models

import (
	"github.com/beego/beego/v2/core/logs"
	"regexp"
)

var smsList = make(map[int]chan string)
var phoneList = make(map[int]string)
var Nark string

func SmsSelect(sender *Sender, msg chan string, smsSelect string) {

	switch smsSelect {
	case "Nolan":
		if sysConfig.NolanUrl == "" || sysConfig.NolanToken == "" {
			logs.Error("NolanUrl or NolanToken is empty")
			return
		}
	case "Rabbit":
		if sysConfig.RabbitUrl == "" {
			logs.Error("RabbitUrl is empty")
			return
		} else {
			sender.Reply("请输入11位手机号")
		}
	}

	for {
		n, ok := <-msg
		//说明发送方关闭了channel
		if !ok {
			break
		}

		regex := "^\\d{5}(\\d|X|x)$"
		reg := regexp.MustCompile(regex)
		if reg.MatchString(n) {
			logs.Info("进入验证码阶段")
			switch smsSelect {
			case "Nolan":
				phone := phoneList[sender.UserID]
				phoneList[sender.UserID] = ""
				go NolanSendCode(phone, n, sender)
				return
			case "Rabbit":
				phone := phoneList[sender.UserID]
				phoneList[sender.UserID] = ""
				go RabbitSendCode(phone, n, sender)
				return
			default:
				logs.Info("报错了")
				return
			}
		}

		regular := `^(13[0-9]|14[01456879]|15[0-35-9]|16[2567]|17[0-8]|18[0-9]|19[0-35-9])\d{8}$`
		reg = regexp.MustCompile(regular)
		if reg.MatchString(n) {
			logs.Info("进入手机号阶段")
			switch smsSelect {
			case "Nolan":
				logs.Info("Pro短信登录")
				phoneList[sender.UserID] = n
				go NolanSendSMS(n, sender)
				return
			case "Rabbit":
				phoneList[sender.UserID] = n
				go RabbitSendSMS(n, sender)
				return
			default:
				logs.Info("报错了")
				return
			}
		}

		if n != "q" {
			sender.Reply("当前处于登录流程，回复‘q’可退出流程")
		} else {
			sender.Reply("退出登录流程")
			smsList[sender.UserID] = nil
			return
		}

	}
}
