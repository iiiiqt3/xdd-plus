package models

//
//import (
//	"bytes"
//	"encoding/json"
//	"fmt"
//	"math/rand"
//	"net/http"
//	"net/url"
//	"regexp"
//	"time"
//
//	"github.com/beego/beego/v2/core/logs"
//	"gorm.io/gorm"
//)
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
//	sender.Reply("1、请多次执行【密码登录】，直到不在需要短信验证码\n2、上车后请到京东-我的-支付设置，关闭小额免密，同时开启虚拟资产验密\n3、由于微信机器人经常被举报封号即将停用，请尽快切换到QQ机器人，回复【QQ群】查看Q群和QQ机器人具体信息")
//	sender.Reply("请输入手机号：")
//	c2 := make(chan string)
//	smsList[sender.UserID] = c2
//	accountInput(sender, c2, Auto)
//}
//
//// 账号输入流程
//func accountInput(sender *Sender, msg chan string, Auto *UserSession) {
//	for {
//		// 设置定时器
//		timeout := time.After(60 * time.Second)
//		select {
//		case n, ok := <-msg:
//			if !ok {
//				break
//			}
//			deal := false
//			regex := `^(13[0-9]|14[01456879]|15[0-35-9]|16[2567]|17[0-8]|18[0-9]|19[0-35-9])\d{8}$`
//			reg := regexp.MustCompile(regex)
//			if reg.MatchString(n) {
//				logs.Info("输入账号阶段")
//				Auto.account = n
//				deal = true
//				cks := GetJdCookies(func(sb *gorm.DB) *gorm.DB {
//					return sb.Where("Account = ?", Auto.account)
//				})
//				if len(cks) > 0 {
//					Auto.password = cks[0].Password
//					if Auto.password != "" {
//						sender.Reply("登录中，请稍等，请勿重复操作...")
//						loginAPI(sender, Auto)
//						smsList[sender.UserID] = nil
//						return
//					}
//				}
//				sender.Reply("请输入密码：")
//				passwordInput(sender, msg, Auto)
//				return
//			}
//			if n == "q" {
//				sender.Reply("退出登录流程")
//				smsList[sender.UserID] = nil
//				return
//			}
//			if !deal {
//				sender.Reply("手机号输入格式错误，请重新输入，或回复‘q’退出流程")
//			}
//		case <-timeout:
//			// 超时处理
//			sender.Reply("操作超时，退出登录流程！")
//			smsList[sender.UserID] = nil
//			return
//		}
//	}
//}
//
//// 密码输入流程
//func passwordInput(sender *Sender, msg chan string, Auto *UserSession) {
//	for {
//		n, ok := <-msg
//		if !ok {
//			break
//		}
//
//		if n == "y" {
//			loginAPI(sender, Auto)
//			return
//		}
//
//		if n != "" && n != "q" {
//			logs.Info("输入密码阶段")
//			Auto.password = n
//			sender.Reply("登录中，请稍等，请勿重复操作...")
//			flag := loginAPI(sender, Auto)
//			if flag {
//				smsList[sender.UserID] = nil
//				return
//			}
//		}
//		if n == "q" {
//			sender.Reply("退出登录流程")
//			smsList[sender.UserID] = nil
//			return
//		}
//	}
//}
//
//// 调用api登录
//func loginAPI(sender *Sender, Auto *UserSession) bool {
//
//	logs.Info("登录阶段")
//	requesturl := Auto.apiBackend + "/api/encrypt"
//	params := map[string]interface{}{
//		"phone":  Auto.account,
//		"pwd":    Auto.password,
//		"isAuto": Auto.isAuto,
//	}
//	jsonData, err := json.Marshal(params)
//	if err != nil {
//		fmt.Printf("JSON序列化失败: %s\n", err)
//		return true
//	}
//
//	client := &http.Client{Timeout: 10 * time.Second}
//	req, err := http.NewRequest("POST", requesturl, bytes.NewBuffer(jsonData))
//	if err != nil {
//		fmt.Printf("创建请求失败: %s\n", err)
//		return true
//	}
//	req.Header.Set("Content-Type", "application/json")
//	resp, err := client.Do(req)
//	if err != nil {
//		sender.Reply("服务器失联啦，过会再试")
//		return true
//	}
//	defer resp.Body.Close()
//	if resp.StatusCode != http.StatusOK {
//		fmt.Printf("请求失败，状态码: %d\n", resp.StatusCode)
//		return true
//	}
//	var result map[string]interface{}
//	if err = json.NewDecoder(resp.Body).Decode(&result); err != nil {
//		fmt.Printf("解析响应失败: %s\n", err)
//		return true
//	}
//	logs.Info(fmt.Printf("响应: %v\n", result))
//
//	if status := result["err_code"].(float64); status == 128 {
//		sender.Reply(result["err_msg"].(string))
//		sender.Reply(result["jmp_url"].(string))
//		sender.Reply("验证后请回复【Y】重新登录")
//		return false
//	} else if status != 0 {
//		sender.Reply(result["err_msg"].(string))
//		return true
//	}
//
//	if cookie := result["pt_key"].(string); cookie != "" {
//		pin := url.QueryEscape(result["pt_pin"].(string))
//		logs.Info(pin)
//		cookie = fmt.Sprintf("pt_key=%s;pin=%s;", cookie, pin)
//		logs.Info(cookie)
//		Auto.appck = cookie
//	}
//	fmt.Printf("登录成功: %s\n", result["pt_pin"])
//	Autockup(Auto.appck, sender, Auto)
//	return true
//}
//
//// 上传ck
//func Autockup(cookie string, sender *Sender, Auto *UserSession) {
//	logs.Info("上传CK阶段")
//	pin := FetchJdCookieValue("pin", Auto.appck)
//	ptkey := FetchJdCookieValue("pt_key", Auto.appck)
//	ck := JdCookie{
//		PtPin:     pin,
//		PtKey:     ptkey,
//		Available: True,
//		QQ:        sender.UserID,
//		Account:   Auto.account,
//		Password:  Auto.password,
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
//		} else {
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
//func UpAutoCookie() {
//	logs.Info("开始密码自动登录检测")
//	cks := GetJdCookies(func(sb *gorm.DB) *gorm.DB {
//		return sb.Where("Account IS NOT NULL AND Account != '' AND Password IS NOT NULL AND Password != '' AND Smsverify = ?", False)
//		//	return sb.Where("Account IS NOT NULL AND Account != '' AND Password IS NOT NULL AND Password != ''")
//	})
//	logs.Info(fmt.Sprintf("需要自动登录账号数量%d个", len(cks)))
//	for _, ck := range cks {
//		time.Sleep(time.Second * time.Duration(Config.Later))
//		if !CookieOK(&ck) {
//			ck.Update(Available, True)
//			time.Sleep(time.Second * time.Duration(Config.Later))
//			time.Sleep(time.Duration(rand.Intn(1000)+1000) * time.Millisecond)
//			//		time.Sleep(time.Second)   //#时间改成1秒
//			Auto := &UserSession{}
//			sender2 := &Sender{
//				UserID: 1,
//				Type:   "tg",
//			}
//			Auto.isAuto = true
//			Auto.isUser = false
//			Auto.account = ck.Account
//			Auto.password = ck.Password
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
//func UpAutoCookie3() {
//	logs.Info("开始密码自动登录检测")
//	cks := GetJdCookies(func(sb *gorm.DB) *gorm.DB {
//		return sb.Where("Account IS NOT NULL AND Account != '' AND Password IS NOT NULL AND Password != '' AND Smsverify = ?", True)
//	})
//	(&JdCookie{}).Push(fmt.Sprintf("开始尝试登录被标记账密账号，数量%d个", len(cks)))
//	for _, ck := range cks {
//		time.Sleep(time.Second * time.Duration(Config.Later))
//		if !CookieOK(&ck) {
//			ck.Update(Available, True)
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
//			Auto.password = ck.Password
//			Auto.apiBackend = GetEnv("apiBackend")
//			loginAPI(sender2, Auto)
//			time.Sleep(30 * time.Second)
//		}
//	}
//	logs.Info("自动登录检测完成")
//	(&JdCookie{}).Push("登录标记账号运行完成！")
//	go func() {
//		Save <- &JdCookie{}
//	}()
//}
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
//			ck.Push(fmt.Sprintf("====尊贵的密码登录用户====\n1、您的账号：%s，已失效，由于账密登录需要短信验证码！已经无法自动续期，建议您多次执行【密码登录】，确保不再需要短信验证码\n2、回复【QQ群】查看Q群和QQ机器人具体信息", ck.PtPin))
//			(&JdCookie{}).Push(fmt.Sprintf("需要验证账号：%s", ck.PtPin))
//			xj++
//		}
//	}
//	(&JdCookie{}).Push(fmt.Sprintf("密码登录检测结束，检测账号数量%d个，需要验证账号%d个", len(cks), xj))
//	go func() {
//		Save <- &JdCookie{}
//	}()
//}
