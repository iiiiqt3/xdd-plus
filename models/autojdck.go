package models

//
//import (
//    "regexp"
//	"bytes"
//	"encoding/json"
//	"fmt"
//	"net/http"
//	"time"
//	"math/rand"
//
//	"github.com/beego/beego/v2/core/logs"
//	"gorm.io/gorm"
//)
//
//
//type UserSession struct {
//	apiBackend string
//	account    string
//	password   string
//	uid        string
//	appck      string
//	smsCode    string
//	smsRetry   int
//	isAuto     bool
//	isUser     bool
//}
//
//// Autojdck 开始流程，提示用户输入手机号
//func Autojdck(sender *Sender, Auto *UserSession) {
//	Auto.apiBackend = GetEnv("apiBackend")
//	if Auto.apiBackend == "" {
//		sender.Reply("管理员未开启密码登录")
//		return
//	}
//	Auto.smsRetry = 0
//	Auto.isAuto = false
//	Auto.isUser = true
//    sender.Reply("密码登录ck不掉线，掉线了机器人会自动提交ck，请关注机器人推送验证消息")
//    sender.Reply("请输入手机号：")
//    c2 := make(chan string)
//    smsList[sender.UserID] = c2
//    accountInput(sender, c2, Auto)
//}
//
//// 账号输入流程
//func accountInput(sender *Sender, msg chan string, Auto *UserSession) {
//    for {
//        // 设置定时器
//        timeout := time.After(60 * time.Second)
//        select {
//        case n, ok := <-msg:
//            if !ok {
//                break
//            }
//            deal := false
//            regex := `^(13[0-9]|14[01456879]|15[0-35-9]|16[2567]|17[0-8]|18[0-9]|19[0-35-9])\d{8}$`
//            reg := regexp.MustCompile(regex)
//            if reg.MatchString(n) {
//                logs.Info("输入账号阶段")
//                Auto.account = n
//                deal = true
//                cks := GetJdCookies(func(sb *gorm.DB) *gorm.DB {
//                    return sb.Where("Account = ?", Auto.account)
//                })
//                if len(cks) > 0 {
//                    Auto.password = cks[0].Password
//                    if (Auto.password != "") {
//                        sender.Reply("登录中，请稍等，请勿重复操作...")
//                        loginAPI(sender, Auto)
//                        smsList[sender.UserID] = nil
//                        return
//                    }
//                }
//                sender.Reply("请输入密码：")
//                passwordInput(sender, msg, Auto)
//                return
//            }
//            if n == "q" {
//                sender.Reply("退出登录流程")
//                smsList[sender.UserID] = nil
//                return
//            }
//            if !deal {
//                sender.Reply("手机号输入格式错误，请重新输入，或回复‘q’退出流程")
//            }
//        case <-timeout:
//            // 超时处理
//            sender.Reply("操作超时，退出登录流程！")
//            smsList[sender.UserID] = nil
//            return
//        }
//    }
//}
//
//// 密码输入流程
//func passwordInput(sender *Sender, msg chan string, Auto *UserSession) {
//    for {
//        n, ok := <-msg
//        if !ok {
//            break
//        }
//        deal := false
//        if n != "" && n != "q" {
//            logs.Info("输入密码阶段")
//            Auto.password = n
//            sender.Reply("登录中，请稍等，请勿重复操作...")
//            deal = true
//			loginAPI(sender, Auto)
//			smsList[sender.UserID] = nil
//			return
//        }
//        if n == "q" {
//            sender.Reply("退出登录流程")
//            smsList[sender.UserID] = nil
//            return
//        }
//        if !deal {
//            sender.Reply("密码不能为空，请重新输入，或回复‘q’退出流程")
//        }
//    }
//}
//
//// 调用api登录
//func loginAPI(sender *Sender, Auto *UserSession) {
//	logs.Info("登录阶段")
//    url := Auto.apiBackend + "/login"
//	params := map[string]interface{}{
//		"id":     Auto.account,
//		"pw":     Auto.password,
//		"isAuto": Auto.isAuto,
//	}
//	jsonData, err := json.Marshal(params)
//	if err != nil {
//		fmt.Printf("JSON序列化失败: %s\n", err)
//		return
//	}
//
//	client := &http.Client{Timeout: 10 * time.Second}
//	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
//	if err != nil {
//		fmt.Printf("创建请求失败: %s\n", err)
//		return
//	}
//	req.Header.Set("Content-Type", "application/json")
//	resp, err := client.Do(req)
//	if err != nil {
//		sender.Reply("服务器失联啦，过会再试")
//		return
//	}
//	defer resp.Body.Close()
//	if resp.StatusCode != http.StatusOK {
//		fmt.Printf("请求失败，状态码: %d\n", resp.StatusCode)
//		return
//	}
//	var result map[string]interface{}
//	if err = json.NewDecoder(resp.Body).Decode(&result); err != nil {
//		fmt.Printf("解析响应失败: %s\n", err)
//		return
//	}
//	fmt.Printf("响应: %v\n", result)
//
//
//	if status, ok := result["status"].(string); !ok || status != "pass" {
//		sender.Reply("服务器失联啦，过会再试")
//		return
//	}
//	Auto.uid = result["uid"].(string)
//	fmt.Printf("uid: %v\n", Auto.uid)
//	time.Sleep(1 * time.Second)
//    check(sender, Auto)
//
//}
//
//
//// 调用api检测状态
//func check(sender *Sender, Auto *UserSession) {
//	logs.Info("检测状态阶段")
//	time.Sleep(5 * time.Second)
//    url := Auto.apiBackend + "/check"
//	params := map[string]interface{}{
//		"uid":     Auto.uid,
//	}
//	jsonData, err := json.Marshal(params)
//	if err != nil {
//		fmt.Printf("JSON序列化失败: %s\n", err)
//		return
//	}
//	client := &http.Client{Timeout: 5 * time.Second}
//
//	for i := 0; i < 18; i++ {
//		time.Sleep(7 * time.Second)
//		req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
//		if err != nil {
//			fmt.Printf("创建请求失败: %s\n", err)
//			return
//		}
//		req.Header.Set("Content-Type", "application/json")
//		resp, err := client.Do(req)
//		if err != nil {
//			sender.Reply("服务器失联啦，过会再试")
//			return
//		}
//		defer resp.Body.Close()
//
//		if resp.StatusCode != http.StatusOK {
//			fmt.Printf("请求失败，状态码: %d\n", resp.StatusCode)
//			return
//		}
//		var result map[string]interface{}
//		if err = json.NewDecoder(resp.Body).Decode(&result); err != nil {
//			fmt.Printf("解析响应失败: %s\n", err)
//			return
//		}
//		fmt.Printf("响应: %v\n", result)
//
//		if status, ok := result["status"].(string); ok {
//			switch status {
//			case "pending":
//				continue
//			case "error":
//
//
//			if msg, exists := result["msg"].(string); exists {
//					if msg == "登录失败，请在十秒后重试：账号或密码不正确" {
//						cks := GetJdCookies(func(sb *gorm.DB) *gorm.DB {
//							return sb.Where("Account = ?", Auto.account)
//						})
//						if len(cks) > 0 {
//							cks[0].Update(Password, "")
//						}
//						sender.Reply("登录失败，请账号或密码不正确，如果你忘记密码，请到京东app修改密码后，再次尝试密码登录！")
//						return
//					}
//					if msg == "登录失败，请在十秒后重试：登录超时" {
//						sender.Reply("登录超时，请在十秒后重试！\n如多次超时，请先发送【短信登录】登录一次后再使用【密码登录】尝试！")
//						return
//					}
//					if msg == "登录失败，请在十秒后重试：自动续期时不能使用短信验证" {
//						cks := GetJdCookies(func(sb *gorm.DB) *gorm.DB {
//							return sb.Where("Account = ?", Auto.account)
//						})
//						if len(cks) > 0 {
//							cks[0].Update(Smsverify, "true")
//						}
//						logs.Info("自动续期时不能使用短信验证")
//						return
//					}
//					logs.Info(msg)
//					sender.Reply(msg)
//				}
//				return
//			case "SMS":
//				time.Sleep(5 * time.Second)
//				sender.Reply("需要短信验证，请输入短信验证码：")
//				smsChan := make(chan string)
//				smsList[sender.UserID] = smsChan
//				SmsInput(sender, smsChan, Auto)
//				return
//			case "wrongSMS":
//				if Auto.smsRetry <= 2 {
//					sender.Reply("验证码错误，请检查后重新输入：")
//					smsChan := make(chan string)
//					smsList[sender.UserID] = smsChan
//					SmsInput(sender, smsChan, Auto)
//					return
//				} else {
//					sender.Reply("验证码连续3次错误，退出流程，请重新输入口令登录！")
//					return
//				}
//			case "pass":
//				if cookie, exists := result["cookie"].(string); exists {
//					Auto.appck = cookie
//				}
//				fmt.Printf("登录成功: %s\n", result)
//				Autockup(Auto.appck, sender, Auto)
//				return
//			}
//		}
//	}
//    sender.Reply("处理账号超时，请稍后再试，可能此账号需要「短信登录」指令登录一次，然后再试尝试密码登录")
//}
//
////获取验证码流程
//func SmsInput(sender *Sender, msg chan string, Auto *UserSession) {
//	for {
//        n, ok := <-msg
//        if !ok {
//            break
//        }
//        deal := false
//        regex := "^\\d{5}(\\d|X|x)$"
//        reg := regexp.MustCompile(regex)
//        if reg.MatchString(n) {
//            logs.Info("短信验证码阶段")
//            Auto.smsCode = n
//            deal = true
//			SmsAPI(sender, Auto)
//			smsList[sender.UserID] = nil
//			check(sender, Auto)
//			return
//        }
//        if n == "q" {
//            sender.Reply("退出登录流程")
//            smsList[sender.UserID] = nil
//            return
//        }
//        if !deal {
//            sender.Reply("验证码输入格式错误，请重新输入6位数字验证码，或回复‘q’退出流程")
//        }
//    }
//}
//
//// 调用api短信验证
//func SmsAPI(sender *Sender, Auto *UserSession) {
//	logs.Info("验证码验证阶段")
//    url := Auto.apiBackend + "/sms"
//	params := map[string]interface{}{
//		"uid":     Auto.uid,
//		"code":     Auto.smsCode,
//	}
//	jsonData, err := json.Marshal(params)
//	if err != nil {
//		fmt.Printf("JSON序列化失败: %s\n", err)
//		return
//	}
//
//	client := &http.Client{Timeout: 10 * time.Second}
//
//	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
//	if err != nil {
//		fmt.Printf("创建请求失败: %s\n", err)
//		return
//	}
//
//	req.Header.Set("Content-Type", "application/json")
//	resp, err := client.Do(req)
//	if err != nil {
//		sender.Reply("服务器失联啦，过会再试")
//		return
//	}
//	defer resp.Body.Close()
//
//	if resp.StatusCode != http.StatusOK {
//		fmt.Printf("请求失败，状态码: %d\n", resp.StatusCode)
//		return
//	}
//	var result map[string]interface{}
//	if err = json.NewDecoder(resp.Body).Decode(&result); err != nil {
//		fmt.Printf("解析响应失败: %s\n", err)
//		return
//	}
//	fmt.Printf("响应: %v\n", result)
//
//	if status, ok := result["status"].(string); ok {
//		if status == "error" {
//			if msg, ok := result["msg"].(string); ok {
//				sender.Reply(msg)
//			}
//		}
//	}
//	Auto.smsRetry++
//}
//
////上传ck
//func Autockup(cookie string, sender *Sender, Auto *UserSession) {
//	logs.Info("上传CK阶段")
//	pin := FetchJdCookieValue("pin", Auto.appck)
//	ptkey := FetchJdCookieValue("pt_key", Auto.appck)
//	ck := JdCookie{
//		PtPin:     pin,
//		PtKey:     ptkey,
//		Available: True,
//		QQ: sender.UserID,
//		Account: Auto.account,
//		Password: Auto.password,
//	}
//	if nck, err := GetJdCookie(ck.PtPin); err == nil {
//		date := Date()
//		UpdateAt2 := nck.UpdateAt
//		if Auto.isUser {
//			result6 := Addcoin(UpdateAt2, sender)
//			cookie := JdCookie{RWskey: "null", QQ: sender.UserID, PtKey: ptkey, Available: True, Account: Auto.account, Password: Auto.password, Smsverify: "false"}
//			if result6 {
//				cookie.UpdateAt = date
//			}
//			switch sender.Type {
//			case "wx", "wxg":
//				cookie.WeiXin = sender.WxId
//			case "tg", "tgg":
//				cookie.Telegram = sender.UserID
//			}
//			nck.Updates(cookie)
//			sender.Reply(fmt.Sprintf("登录成功:%s", pin))
//			(&JdCookie{}).Push(fmt.Sprintf("来自密码登录成功:%s", pin))
//		}else {
//			result7 := AutoAddcoin(UpdateAt2, pin)
//			cookie := JdCookie{RWskey: "null", PtKey: ptkey, Available: True}
//			if result7 {
//				cookie.UpdateAt = date
//				(&JdCookie{}).Push(fmt.Sprintf("来自密码自动登录成功:%s，登录奖励积分已发放！", pin))
//			} else {
//				(&JdCookie{}).Push(fmt.Sprintf("来自密码自动登录成功:%s，三日内登录奖励积分已发放！", pin))
//			}
//			nck.Updates(cookie)
//		}
//
//	} else {
//		NewJdCookie(&ck)
//		msg := fmt.Sprintf("来自密码登录的添加账号，账号名:%s", ck.PtPin)
//		switch sender.Type {
//		case "wx", "wxg":
//			ck.Update("WeiXin", sender.WxId)
//		case "tg", "tgg":
//			ck.Update("Telegram", sender.UserID)
//		}
//		sender.Reply(fmt.Sprintf(msg))
//		Recoin(sender)
//		sender.Reply(ck.Query())
//		(&JdCookie{}).Push(msg)
//	}
//	go func() {
//		Save <- &JdCookie{}
//	}()
//	return
//}
//
//func AutoAddcoin(updateAt2 string, pin string) bool {
//	cks := GetJdCookies(func(sb *gorm.DB) *gorm.DB {
//		return sb.Where("PtPin = ?", pin)
//	})
//	if len(cks) == 0 {
//		return false
//	}
//	coninid := cks[0].QQ
//	var u User
//
//	err := db.Where("number = ?", coninid).First(&u).Error
//	if err != nil {
//		return false
//	}
//	result := CompareDates(updateAt2)
//	switch result {
//	case -1, 0:
//		coin := 20 //奖励积分数量
//		db.Model(&u).Updates(map[string]interface{}{
//			"coin": gorm.Expr(fmt.Sprintf("coin+%d", coin)),
//		})
//		u.Coin += coin
//		return true
//	default:
//		return false
//	}
//}
//
//
//
//
//
//func UpAutoCookie() {
//	logs.Info("开始密码自动登录检测")
//	cks := GetJdCookies(func(sb *gorm.DB) *gorm.DB {
//		return sb.Where("Account IS NOT NULL AND Account != '' AND Password IS NOT NULL AND Password != '' AND Smsverify = ?", False)
//	})
//	logs.Info(fmt.Sprintf("需要自动登录账号数量%d个", len(cks)))
//	for _, ck := range cks {
//		time.Sleep(time.Second * time.Duration(Config.Later))
//		if !CookieOK(&ck) {
//			time.Sleep(time.Second * time.Duration(Config.Later))
//			time.Sleep(time.Duration(rand.Intn(1000)+2000) * time.Millisecond)
//			Auto := &UserSession{}
//			sender2 := &Sender{
//				UserID: 1,
//				Type:   "tg",
//			}
//			Auto.isAuto = true
//			Auto.isUser = false
//			Auto.account = ck.Account
//			Auto.password =ck.Password
//			Auto.apiBackend = GetEnv("apiBackend")
//			loginAPI(sender2, Auto)
//		}
//	}
//	logs.Info("自动登录检测完成")
//	go func() {
//		Save <- &JdCookie{}
//	}()
//}
//
//
//
//
//
//
//
//
//func initAutoCookie() {
//	(&JdCookie{}).Push("开始密码登录检测")
//	cks := GetJdCookies(func(sb *gorm.DB) *gorm.DB {
//		return sb.Where("Available = ? AND Account IS NOT NULL AND Account != ''", True)
//	})
//	xj := 0
//	for _, ck := range cks {
//		time.Sleep(time.Second * time.Duration(Config.Later))
//		if !CookieOK(&ck) {
//			//todo 通知账号失效
//			ck.Updates(JdCookie{Available: False})
//			time.Sleep(time.Second * time.Duration(Config.Later))
//			time.Sleep(time.Duration(rand.Intn(1000)+2000) * time.Millisecond)
//			ck.Push(fmt.Sprintf("====尊贵的密码登录用户====\n您的账号：%s，临时失效，需要短信验证！\n请重新发送【密码登录】验证！", ck.PtPin))
//			(&JdCookie{}).Push(fmt.Sprintf("需要验证账号：%s", ck.PtPin))
//			xj++
//		}
//	}
//	(&JdCookie{}).Push(fmt.Sprintf("密码登录检测结束，检测账号数量%d个，需要验证账号%d个", len(cks), xj))
//	go func() {
//		Save <- &JdCookie{}
//	}()
//}
