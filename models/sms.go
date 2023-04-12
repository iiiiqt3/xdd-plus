package models

import (
	"github.com/beego/beego/v2/core/logs"
	"regexp"
)

var smsList = make(map[int]chan string)
var phoneList = make(map[int]string)

func SmsSelect(sender *Sender, msg chan string, smsSelect string) {

	var Nark string
	var Rabbit string

	switch smsSelect {

	case "Nolan":
		Nark = GetEnv("Nark")
		if Nark == "" {
			logs.Error("Nark is empty")
			return
		}
	case Rabbit:
		Rabbit = GetEnv("Rabbit")
		if Nark == "" {
			logs.Error("Rabbit is empty")
			return
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

			case "Rabbit":
				phone := phoneList[sender.UserID]
				phoneList[sender.UserID] = ""
				go RabbitSendCode(phone, n, sender)
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

			case "Rabbit":
				phoneList[sender.UserID] = n
				go RabbitSendSMS(n, sender)
			default:
				logs.Info("报错了")
				return
			}
		}

		sender.Reply("无法识别")

	}
}
