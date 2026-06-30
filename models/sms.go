package models

import (
	"regexp"
   "time"
	"gorm.io/gorm"
)

var smsList = make(map[int]chan string)
var phoneList = make(map[int]string)
var RiskList = make(map[int]bool)

// SmsSelect 短信验证码登录流程，支持Nolan和Rabbit两种平台
func SmsSelect(sender *Sender, msg chan string, smsSelect string) {
	// 检查配置
	switch smsSelect {
	case "Nolan":
		if sysConfig.NolanUrl == "" || sysConfig.NolanToken == "" {
			Error("NolanUrl or NolanToken is empty")
			sender.Reply("配置错误：Nolan 接口信息不完整。")
			return
		}
	case "Rabbit":
		if sysConfig.RabbitUrl == "" {
			Error("RabbitUrl is empty")
			sender.Reply("配置错误：Rabbit 接口信息不完整。")
			return
		}
	default:
		sender.Reply("不支持的平台类型。")
		return
	}

	// 等待手机号输入
	phone, ok := WaitForUserInput(sender, msg, 60, "请输入11位手机号")
	if !ok {
		smsList[sender.UserID] = nil
		return
	}

	// 校验手机号
	phoneRegex := regexp.MustCompile(`^(13[0-9]|14[01456879]|15[0-35-9]|16[2567]|17[0-8]|18[0-9]|19[0-35-9])\d{8}$`)
	if !phoneRegex.MatchString(phone) {
		sender.Reply("手机号格式错误，已退出流程。")
		smsList[sender.UserID] = nil
		return
	}
	phoneList[sender.UserID] = phone

	// Nolan 平台先发验证码
	switch GetEnv("短信登录") {
     case "pro":
         smsSelect = "Nolan"
     case "兔子":
         smsSelect = "Rabbit"
     }
	if smsSelect == "Nolan" {
		Info("Nolan 短信发送中")
		sent := NolanSendSMS(phone, sender)  // 同步等待返回
          if !sent {
              // 发送失败时立即退出，不再等待验证码
              smsList[sender.UserID] = nil
              return
          }
	} else if smsSelect == "Rabbit" {
		// 检查是否已保存密码
		Auto := &UserSession{account: phone}
		value := GetEnv("SMStoPwd")
		cks := GetJdCookies(func(sb *gorm.DB) *gorm.DB {
			return sb.Where("Account = ?", Auto.account)
		})
		if len(cks) > 0 && value == "" {
			Auto.password = cks[0].Password
			if Auto.password != "" && CookieOK(&cks[0]) {
				sender.Reply("账号有效，无需重复登录")
				smsList[sender.UserID] = nil
				return
			}
			sender.Reply("登录中，请稍等...")
			Auto.smsRetry = 0
			Auto.isAuto = false
			Auto.isUser = true
			loginAPI(sender, Auto)
			return
		}
		Info("Rabbit 短信发送中")
		sent := RabbitSendSMS("mck", phone, sender)
		if !sent {
			smsList[sender.UserID] = nil
			return
		}
	}

	// 等待验证码输入
	code, ok := WaitForUserInput(sender, msg, 240, "")
	if !ok {
		smsList[sender.UserID] = nil
		return
	}

	// 校验验证码格式
	codeRegex := regexp.MustCompile(`^\d{5}(\d|X|x)$`)
	if !codeRegex.MatchString(code) {
		sender.Reply("验证码格式错误，退出流程。")
		smsList[sender.UserID] = nil
		return
	}

	// 调用验证码提交接口
	switch smsSelect {
	case "Nolan":
		if RiskList[sender.UserID] {
			go NolanAuthCode(phone, code, sender)
		} else {
			go NolanSendCode(phone, code, sender)
		}
	case "Rabbit":
		phoneList[sender.UserID] = ""
		go RabbitSendCode("mck", phone, code, sender)
	}

	smsList[sender.UserID] = nil
}


//等待用户输入通用逻辑
func WaitForUserInput(sender *Sender, msg chan string, timeoutSec int, prompt string) (string, bool) {
	timeout := time.After(time.Duration(timeoutSec) * time.Second)
	sender.Reply(prompt)

	select {
	case input := <-msg:
		if input == "q" || input == "Q" {
			sender.Reply("您已退出流程。")
			return "", false
		}
		return input, true
	case <-timeout:
		sender.Reply("操作超时，自动退出。")
		return "", false
	}
}