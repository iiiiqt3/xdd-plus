package models

import (
	"encoding/json"
	"fmt"
	"github.com/beego/beego/v2/client/httplib"
	"github.com/beego/beego/v2/core/logs"
	"github.com/buger/jsonparser"
	"gorm.io/gorm"
	"net/url"
	"strconv"
	"strings"
	"time"
)

var RabbitUrl string
var RabbitApiToken string
var RabbitToken string

func RabbitGetJdQrImg(sender *Sender) {
	RabbitUrl = GetEnv("RabbitUrl")
	RabbitApiToken = GetEnv("RabbitApiToken")
	RabbitToken = GetEnv("RabbitToken")
	if RabbitUrl == "" || RabbitApiToken == "" || RabbitToken == "" {
		logs.Error("RabbitUrl or RabbitToken is empty")
		return
	}
	get := httplib.Post(fmt.Sprintf("%s/api/BeanQrCode?token=%s", RabbitUrl, RabbitApiToken))
	bytes, _ := get.Bytes()
	logs.Info(string(bytes))
	code, _ := jsonparser.GetInt(bytes, "code")
	if code == 0 {

		key, _ := jsonparser.GetString(bytes, "QRCodeKey")

		//返回口令
		jcommond, _ := jsonparser.GetString(bytes, "jcommond")
		sender.Reply(jcommond)

		//sender.Reply(NolanLJToKL("https://qr.m.jd.com/p?k="+key, "京东快捷登录"))

		//qr, _ := jsonparser.GetString(bytes, "qr")
		//decodeStr, _ := base64.StdEncoding.DecodeString(qr)
		//sender.SendImg(decodeStr)

		sender.Reply("请复制口令到京东APP登录，150秒失效")
		go RabbitGetJDQrStatus(key, sender)
	} else {
		logs.Info(string(bytes))
		sender.Reply("获取扫码失败")
	}
}

func RabbitGetJDQrStatus(cookie string, sender *Sender) {
	for {
		time.Sleep(time.Second * time.Duration(5))
		get := httplib.Post(fmt.Sprintf("%s/api/QrCheck?token=%s", RabbitUrl, RabbitApiToken))
		marshal, _ := json.Marshal(struct {
			QRCodeKey string `json:"QRCodeKey"`
			Qlkey     string `json:"qlkey"`
		}{
			QRCodeKey: cookie,
			Qlkey:     strconv.Itoa(0),
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
				nck.Updates(JdCookie{RWskey: data, QQ: sender.UserID, PtKey: ptkey})
				sender.Reply(fmt.Sprintf("登录成功:%s", pin))
				(&JdCookie{}).Push(fmt.Sprintf("登录成功:%s", pin))
			} else {
				NewJdCookie(&ck)
				msg := fmt.Sprintf("添加账号，账号名:%s", ck.PtPin)
				if sender.IsQQ() || sender.IsQQ() {
					ck.Update(QQ, sender.UserID)
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
	get := httplib.Post(fmt.Sprintf("%s/api/wsck?RabbitToken=%s", RabbitUrl, RabbitApiToken))
	marshal, _ := json.Marshal(struct {
		WSCK        string `json:"wsck"`
		RabbitToken string `json:"RabbitToken"`
	}{
		WSCK:        cookie,
		RabbitToken: RabbitToken,
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
		return val, "", ""
	}
}

func UpdateRwskey() {
	RabbitUrl = GetEnv("RabbitUrl")
	RabbitApiToken = GetEnv("RabbitApiToken")
	RabbitToken = GetEnv("RabbitToken")

	NolanUrl = GetEnv("NolanUrl")
	NolanToken = GetEnv("NolanToken")

	BBKToken = GetEnv("BBKToken")
	BBKJdUrl = GetEnv("BBKJdUrl")

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
		//自动切换转换渠道，默认nolan
		if NolanUrl != "" && NolanToken != "" {
			rsp, _, appck = NolanGetCookie(pinky)
		} else if RabbitUrl != "" && RabbitApiToken != "" && RabbitToken != "" {
			pin, _ := url.QueryUnescape(ck.PtPin)
			var pinky = fmt.Sprintf("pin=%s;wskey=%s;", pin, ck.RWskey)
			rsp, _, appck = RabbitGetCookie(pinky)
		} else if BBKToken != "" && BBKJdUrl != "" {
			rsp, _, appck = BBKGetCookie(pinky)
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
			(&JdCookie{}).Push(fmt.Sprintf("转换失败，请求超时，账号:%s", ck.PtPin))
		}
	}
	go func() {
		Save <- &JdCookie{}
	}()
	(&JdCookie{}).Push(fmt.Sprintf("所有CK转换完成，共%d个,转换失败个数共%d个", xx, yy))
}

func RabbitSendSMS(phone string, sender *Sender) {
	sender.Reply("请耐心等待...")
	logs.Info(RabbitUrl)
	req := httplib.Post(fmt.Sprintf("%s/api/sendSMS?token=%s", RabbitUrl, RabbitApiToken))
	req.Header("content-type", "application/json")
	data, _ := req.Body(`{"Phone":"` + phone + `","qlkey":1}`).Bytes()
	logs.Info(string(data))
	message, _ := jsonparser.GetString(data, "message")
	success, _ := jsonparser.GetBoolean(data, "success")
	status, _ := jsonparser.GetInt(data, "data", "status")
	if message != "" && status != 666 {
		sender.Reply(message)
	}
	i := 1
	if success {
		logs.Info(strconv.Itoa(sender.UserID))
		sender.Reply("请输入6位验证码：")
		return
	}
	//{"success":true,"message":"","data":{"ckcount":0,"tabcount":3}}
	if !success && status == 666 {

		sender.Reply("正在进行验证...")
		for {
			i++
			req = httplib.Post(fmt.Sprintf("%s/api/AutoCaptcha?token=%s", RabbitUrl, RabbitApiToken))
			req.Header("content-type", "application/json")
			data, _ := req.Body(`{"Phone":"` + phone + `"}`).Bytes()
			message, _ := jsonparser.GetString(data, "message")
			success, _ := jsonparser.GetBoolean(data, "success")
			status, _ := jsonparser.GetInt(data, "data", "status")
			if success {
				sender.Reply("请输入6位验证码：")
				break
			}
			if i > 5 {
				//pcodes[sender.UserID] = msg
				//s := Config.Jdcurl + "/Captcha/" + msg
				//sender.Reply(fmt.Sprintf("请访问网址进行手动验证%s", s))
				sender.Reply("滑块验证失败,请尝试重新登录")
				break
			}
			if status == 666 {
				i++
				sender.Reply(fmt.Sprintf("正在进行第%d次滑块验证...", i))
				continue
			}
			if strings.Contains(message, "上限") {
				i = 6
				sender.Reply(message)
				break
			}
		}
	} else {
		smsList[sender.UserID] = nil
		sender.Reply("滑块失败，请网页登录")
	}
}

func RabbitSendCode(phone string, code string, sender *Sender) {
	sender.Reply("请耐心等待...")
	req := httplib.Post(fmt.Sprintf("%s/api/VerifyCode?token=%s", RabbitUrl, RabbitApiToken))
	req.Header("content-type", "application/json")
	data, _ := req.Body(fmt.Sprintf("{\n    \"Phone\": %s,\n    \"Code\": \"%s\",\n    \"qlkey\": 1\n}", phone, code)).Bytes()
	logs.Info(string(data))

	message, _ := jsonparser.GetString(data, "msg")
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
			nck.Updates(JdCookie{WsKey: wskey, QQ: sender.UserID, PtKey: ptkey})
			sender.Reply(fmt.Sprintf("登录成功:%s", pin))
			(&JdCookie{}).Push(fmt.Sprintf("登录成功:%s", pin))
		} else {
			NewJdCookie(&ck)
			msg := fmt.Sprintf("添加账号，账号名:%s", ck.PtPin)
			if sender.IsQQ() || sender.IsQQ() {
				ck.Update(QQ, sender.UserID)
			}
			sender.Reply(fmt.Sprintf(msg))
			sender.Reply(ck.Query())
			(&JdCookie{}).Push(msg)
		}

		sender.Reply(fmt.Sprintf("登录成功:%s", pin))
		(&JdCookie{}).Push(fmt.Sprintf("登录成功:%s", pin))
		smsList[sender.UserID] = nil
	} else {
		sender.Reply(message)
	}
}
