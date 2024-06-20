package models

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/beego/beego/v2/client/httplib"
	"github.com/beego/beego/v2/core/logs"
	"github.com/buger/jsonparser"
	"github.com/skip2/go-qrcode"
	"gorm.io/gorm"
)

func RabbitGetJdQrImg(sender *Sender) {
	if sysConfig.RabbitUrl == "" || sysConfig.RabbitApiToken == "" || sysConfig.RabbitToken == "" {
		logs.Error("RabbitUrl or RabbitToken is empty")
		return
	}
	get := httplib.Post(fmt.Sprintf("%s/bot/GenQrCode?BotApiToken=%s", sysConfig.RabbitUrl, sysConfig.RabbitApiToken))
	bytes, _ := get.Bytes()
	logs.Info(string(bytes))
	code, _ := jsonparser.GetInt(bytes, "code")
	if code == 0 {

		key, _ := jsonparser.GetString(bytes, "QRCodeKey")

		//返回口令
		jcommond, _ := jsonparser.GetString(bytes, "jcommond")
		sender.Reply(jcommond + " 请复制口令到京东APP打开")

		qr, _ := jsonparser.GetString(bytes, "qr")
		decodeStr, _ := base64.StdEncoding.DecodeString(qr)
		sender.SendImg(decodeStr)

		sender.Reply("请使用京东APP扫描，150秒失效")
		go RabbitGetJDQrStatus(key, sender)
	} else {
		logs.Info(string(bytes))
		sender.Reply("获取扫码失败")
	}
}

func RabbitGetJDQrStatus(cookie string, sender *Sender) {
	// {"success":false,"message":"","data":{"ck":"","rwskey":"","accessToken":"","refreshToken":"","roles":[],"img":null,"username":"user","expires":"2023-09-26T15:45:13.3731483+08:00","status":0,"mode":null}}

	for {
		time.Sleep(time.Second * time.Duration(5))
		get := httplib.Post(fmt.Sprintf("%s/bot/QrCheck?BotApiToken=%s", sysConfig.RabbitUrl, sysConfig.RabbitApiToken))
		marshal, _ := json.Marshal(struct {
			QRCodeKey string `json:"QRCodeKey"`
		}{
			QRCodeKey: cookie,
		},
		)
		get.Body(marshal)
		bytes, _ := get.Bytes()
		code, _ := jsonparser.GetInt(bytes, "code")
		msg, _ := jsonparser.GetString(bytes, "msg")
		logs.Info(string(bytes))
		if code == 502 || code == 503 || code == 403 || code == 54 {
			sender.Reply(msg)
			return
		} else if code == 200 {
			data, _ := jsonparser.GetString(bytes, "wskey")
			pin, _ := jsonparser.GetString(bytes, "pin")
			if data == "" || pin == "" {
				sender.Reply("渠道维护，请使用其他登录方式。")
				return
			}
			var pinky = fmt.Sprintf("pin=%s;wskey=%s;", pin, data)
			_, _, appck := RabbitGetCookie(pinky)
			pin = url.QueryEscape(pin)
			ptkey := FetchJdCookieValue("pt_key", appck)
			ck := JdCookie{
				PtPin:  pin,
				PtKey:  ptkey,
				RWskey: data,
			}
			if nck, err := GetJdCookie(ck.PtPin); err == nil {
				nck.Updates(JdCookie{RWskey: data, QQ: sender.UserID, PtKey: ptkey, WeiXin: sender.WxId})
				sender.Reply(fmt.Sprintf("登录成功:%s", pin))
				(&JdCookie{}).Push(fmt.Sprintf("登录成功:%s", pin))
			} else {
				NewJdCookie(&ck)
				msg := fmt.Sprintf("添加账号，账号名:%s", ck.PtPin)
				if sender.IsQQ() || sender.isWX() {
					ck.Update(QQ, sender.UserID)
					ck.Update("WeiXin", sender.WxId)
				}
				sender.Reply(fmt.Sprintf(msg))
				sender.Reply(ck.Query())
				(&JdCookie{}).Push(msg)
			}
			go func() {
				Save <- &JdCookie{}
			}()
			return
		}

	}
}

func RabbitGetCookie(cookie string) (bool, string, string) {
	get := httplib.Post(fmt.Sprintf("%s/bot/wsck?BotApiToken=%s", sysConfig.RabbitUrl, sysConfig.RabbitApiToken))
	marshal, _ := json.Marshal(struct {
		WSCK        string `json:"wsck"`
		RabbitToken string `json:"RabbitToken"`
	}{
		WSCK:        cookie,
		RabbitToken: sysConfig.RabbitToken,
	})
	get.Body(marshal)
	bytes, _ := get.Bytes()
	val, _ := jsonparser.GetBoolean(bytes, "success")
	if val {
		msg, _ := jsonparser.GetString(bytes, "msg")
		appck, _ := jsonparser.GetString(bytes, "data", "appck")
		return val, msg, appck
	} else {
		logs.Info(string(bytes))
		time.Sleep(time.Second * 3)
		return val, "", ""
	}
}

func UpdateRwskey() {

	cks := GetJdCookies(func(sb *gorm.DB) *gorm.DB {
		return sb.Where(fmt.Sprintf("%s != ? and %s !=?", RWSKEY, RWSKEY), "null", "")
	})
	xx := 0
	yy := 0
	(&JdCookie{}).Push("开始定时更新转换RWskey")

	for i, ck := range cks {
		if i == len(cks)/2 {
			(&JdCookie{}).Push("RWskey已更新二分一")
		}

		//JdCookie{}.Push(fmt.Sprintf("更新账号账号，%s", ck.Nickname))
		var pinky = fmt.Sprintf("pin=%s;wskey=%s;", ck.PtPin, ck.RWskey)
		//rsp, _, appck := NolanGetCookie(pinky)

		var rsp bool
		var appck string
		var retry int
		//自动切换转换渠道，默认nolan
		if sysConfig.NolanUrl != "" && sysConfig.NolanToken != "" {
			for !rsp {
				rsp, _, appck = NolanGetCookie(pinky)
				retry++
				if retry == 5 {
					logs.Info("退出尝试")
					break
				}
			}
		} else if sysConfig.RabbitUrl != "" && sysConfig.RabbitApiToken != "" && sysConfig.RabbitToken != "" {
			pin, _ := url.QueryUnescape(ck.PtPin)
			var pinky = fmt.Sprintf("pin=%s;wskey=%s;", pin, ck.RWskey)

			for !rsp {
				rsp, _, appck = RabbitGetCookie(pinky)
				retry++
				if retry == 5 {
					break
				}
			}

		} else if sysConfig.BBKToken != "" && sysConfig.BBKJdUrl != "" {
			for !rsp {
				rsp, _, appck = BBKGetCookie(pinky)
				retry++
				if retry == 5 {
					break
				}
			}
		}

		if rsp {
			ptKey := FetchJdCookieValue("pt_key", appck)
			ptPin := FetchJdCookieValue("pt_pin", appck)
			ck1 := JdCookie{
				PtKey: ptKey,
				PtPin: ptPin,
			}
			if ptPin != "" || ptKey != "" {
				if nck, err := GetJdCookie(ck1.PtPin); err == nil {
					xx++
					nck.Updates(JdCookie{PtKey: ptKey, Available: True})
					msg := fmt.Sprintf("定时更新账号，%s", ck.PtPin)
					////不再发送成功提醒
					//(&JdCookie{}).Push(msg)
					logs.Info(msg)
				} else {
					yy++
					ck1.Update(Available, False)
					(&JdCookie{}).Push(fmt.Sprintf("查无匹配得ptpin，%s", ck.PtPin))
				}
			} else {
				yy++
				logs.Info(appck)
				(&JdCookie{}).Push(fmt.Sprintf("转换失败，请求超时，账号:%s", ck.PtPin))
			}

		} else {
			ck.Updates(JdCookie{RWskey: "null", Available: False})
			time.Sleep(time.Second * time.Duration(Config.Later))
			ck.Push(fmt.Sprintf("RWskey失效账号，%s，请稍后重新登录", ck.PtPin))
			(&JdCookie{}).Push(fmt.Sprintf("RWskey失效，%s", ck.PtPin))
		}
	}
	go func() {
		Save <- &JdCookie{}
	}()
	(&JdCookie{}).Push(fmt.Sprintf("所有CK转换完成，共%d个,转换失败个数共%d个", xx, yy))
}

func RabbitSendSMS(ty string, phone string, sender *Sender) {
	sender.Reply("正在验证...")
	logs.Info(sysConfig.RabbitUrl)
	var req *httplib.BeegoHTTPRequest
	if ty == "mck" {
		req = httplib.Post(fmt.Sprintf("%s/bot/mck/sendSMS?BotApiToken=%s", sysConfig.RabbitUrl, sysConfig.RabbitApiToken))
	} else if ty == "wskey" {
		req = httplib.Post(fmt.Sprintf("%s/bot/wskey/sendSMS?BotApiToken=%s", sysConfig.RabbitUrl, sysConfig.RabbitApiToken))
	}
	req.Header("content-type", "application/json; charset=utf-8")
	data, _ := req.Body(`{"Phone":"` + phone + `"}`).Bytes()
	logs.Info(string(data))
	message, _ := jsonparser.GetString(data, "message")
	success, _ := jsonparser.GetBoolean(data, "success")
	status, _ := jsonparser.GetInt(data, "data", "status")
	if message != "" && status != 666 {
		sender.Reply(message)
	}
	if success {
		logs.Info(strconv.Itoa(sender.UserID))
		sender.Reply("请输入6位验证码：")
		return
	} else {
		sender.Reply("正在进行验证...")
		i := 1
		for {
			i++
			if ty == "mck" {
				req = httplib.Post(fmt.Sprintf("%s/bot/mck/AutoCaptcha?BotApiToken=%s", sysConfig.RabbitUrl, sysConfig.RabbitApiToken))
			} else if ty == "wskey" {
				req = httplib.Post(fmt.Sprintf("%s/bot/wskey/AutoCaptcha?BotApiToken=%s", sysConfig.RabbitUrl, sysConfig.RabbitApiToken))
			}
			req.Header("content-type", "application/json; charset=utf-8")
			data, _ := req.Body(`{"Phone":"` + phone + `"}`).Bytes()
			message, _ := jsonparser.GetString(data, "message")
			success, _ := jsonparser.GetBoolean(data, "success")
			status, _ := jsonparser.GetInt(data, "data", "status")
			if success {
				sender.Reply("请输入6位验证码：")
				break
			}
			if i > 5 {
				smsList[sender.UserID] = nil
				sender.Reply("滑块验证失败,请尝试重新登录")
				break
			}
			if status == 666 || status == 505 {
				i++
				sender.Reply(fmt.Sprintf("正在进行第%d次滑块验证...", i))
				//休眠2秒钟
				time.Sleep(time.Second * 2)
				continue
			} else {
				sender.Reply(message)
				smsList[sender.UserID] = nil
				break
			}

		}
	}
}

func RabbitSendCode(ty string, phone string, code string, sender *Sender) {
	sender.Reply("请耐心等待...")

	var req *httplib.BeegoHTTPRequest
	if ty == "mck" {
		req = httplib.Post(fmt.Sprintf("%s/bot/mck/VerifyCode?BotApiToken=%s", sysConfig.RabbitUrl, sysConfig.RabbitApiToken))
	} else if ty == "wskey" {
		req = httplib.Post(fmt.Sprintf("%s/bot/wskey/VerifyCode?BotApiToken=%s", sysConfig.RabbitUrl, sysConfig.RabbitApiToken))
	}
	req.Header("Content-Type", "application/json; charset=utf-8")
	data, _ := req.Body(fmt.Sprintf("{\n    \"Phone\": %s,\n    \"Code\": \"%s\" \n}", phone, code)).Bytes()

	logs.Info(string(data))
	message, _ := jsonparser.GetString(data, "message")
	pin, _ := jsonparser.GetString(data, "pin")
	state, _ := jsonparser.GetInt(data, "code")
	wskey, _ := jsonparser.GetString(data, "wskey")
	appck, _ := jsonparser.GetString(data, "ck")

	if state == 200 {
		ptkey := FetchJdCookieValue("pt_key", appck)
		ck := JdCookie{
			PtPin: pin,
			PtKey: ptkey,
			WsKey: wskey,
		}
		if nck, err := GetJdCookie(ck.PtPin); err == nil {
			nck.Updates(JdCookie{WsKey: wskey, QQ: sender.UserID, PtKey: ptkey, WeiXin: sender.WxId})
			sender.Reply(fmt.Sprintf("登录成功:%s", pin))
			(&JdCookie{}).Push(fmt.Sprintf("登录成功:%s", pin))
		} else {
			NewJdCookie(&ck)
			msg := fmt.Sprintf("添加账号，账号名:%s", ck.PtPin)
			if sender.IsQQ() || sender.IsQQ() {
				ck.Update(QQ, sender.UserID)
				ck.Update("WeiXin", sender.WxId)
			}
			sender.Reply(fmt.Sprintf(msg))
			sender.Reply(ck.Query())
			(&JdCookie{}).Push(msg)
		}
		smsList[sender.UserID] = nil
	} else if state == 555 {
		RiskUrl, _ := jsonparser.GetString(data, "RiskUrl")
		var png []byte
		png, _ = qrcode.Encode(RiskUrl, qrcode.Medium, 256)
		sender.SendImg(png)
		sender.Reply(message)
		smsList[sender.UserID] = nil
	} else if state == 505 {
		sender.Reply(message)
	} else {
		smsList[sender.UserID] = nil
		sender.Reply(message)
	}
}
