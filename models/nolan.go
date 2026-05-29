package models

import (
	"encoding/json"
	"fmt"
	//	"strconv"
	"time"

	"github.com/beego/beego/v2/client/httplib"
	"github.com/beego/beego/v2/core/logs"
	"github.com/buger/jsonparser"
	"github.com/skip2/go-qrcode"
	"gorm.io/gorm"
)

func NolanGetJdQrImg(sender *Sender) {
	if sysConfig.NolanUrl == "" || sysConfig.NolanToken == "" {
		logs.Error("NolanUrl or NolanToken is empty")
		return
	}

	get := httplib.Post(fmt.Sprintf("%s/qr/GetQRKey", sysConfig.NolanUrl))
	get.Header("Content-Type", "application/json")
	get.Body(fmt.Sprintf("{\n  \"botApitoken\": \"%s\"\n}", sysConfig.NolanToken))
	bytes, _ := get.Bytes()
	logs.Info(string(bytes))

	code, _ := jsonparser.GetBoolean(bytes, "success")
	if code {
		key, _ := jsonparser.GetString(bytes, "data", "key")
		var png []byte
		png, _ = qrcode.Encode("https://qr.m.jd.com/p?k="+key, qrcode.Medium, 256)
		sender.SendImg(png)
		value := GetEnv("dlkl") // 变量设置登录变量值，export dlkl 开
		if value == "开" {
			sender.Reply("请使用京东APP扫描登录或复制链接用浏览器打开或复制口令打开京东app，150秒失效\n口令模式只支持安卓用户，苹果用户无法使用")
			sender.Reply(LJtoKL("https://lzkj-isv.isvjcloud.com/lzclient/cjwx/common/openJDApp.html?actlink=openapp.jdmobile://virtual?params={\"category\":\"jump\",\"des\":\"scanLogin\",\"key\":\"AAEAIC7o7uvtQDO6vdYl4liag5G4fngqZbK2Vt83LyAbnmhF\",\"sourceType\":\"JSHOP_SOURCE_TYPE\",\"sourceValue\":\"JSHOP_SOURCE_VALUE\",\"M_sourceFrom\":\"mxz\",\"msf_type\":\"auto\"}"))
			sender.Reply(fmt.Sprintf("https://qr.m.jd.com/p?k=%s", key))
		} else {
			sender.Reply(fmt.Sprintf("https://qr.m.jd.com/p?k=%s", key))
			sender.Reply("请使用京东APP扫描或复制链接用浏览器打开，150秒失效")
		}

		logs.Info(key)
		go NolanGetJDQrStatus(key, sender)
	} else {
		logs.Info(string(bytes))
		sender.Reply("获取扫码失败")
	}
}

func NolanGetJDQrStatus(cookie string, sender *Sender) {

	type NolanRWskey struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
		Data    struct {
			Ck           string        `json:"ck"`
			Rwskey       string        `json:"rwskey"`
			AccessToken  string        `json:"accessToken"`
			RefreshToken string        `json:"refreshToken"`
			Roles        []interface{} `json:"roles"`
			Img          interface{}   `json:"img"`
			Username     string        `json:"username"`
			Expires      time.Time     `json:"expires"`
			Status       int           `json:"status"`
			Mode         interface{}   `json:"mode"`
		} `json:"data"`
	}
	for {
		time.Sleep(time.Second * time.Duration(5))
		get := httplib.Post(fmt.Sprintf("%s/qr/CheckQRKey", sysConfig.NolanUrl))
		get.Header("Content-Type", "application/json")
		get.Body(fmt.Sprintf("{\n  \"qrkey\": \"%s\",\n  \"botApitoken\": \"%s\"\n}", cookie, sysConfig.NolanToken))
		bytes, _ := get.Bytes()
		code, _ := jsonparser.GetBoolean(bytes, "success")
		logs.Info(string(bytes))
		if code {
			nolan := &NolanRWskey{}
			json.Unmarshal(bytes, nolan)
			logs.Info(nolan.Data.Rwskey)
			var pinky = nolan.Data.Rwskey
			_, _, appck := NolanGetCookie(pinky)
			pin := FetchJdCookieValue("pin", appck)
			ptkey := FetchJdCookieValue("pt_key", appck)
			rwskey := FetchJdCookieValue("wskey", pinky)
			ck := JdCookie{
				PtPin:     pin,
				PtKey:     ptkey,
				RWskey:    rwskey,
				Available: True,
				QQ:        sender.UserID,
			}
			if nck, err := GetJdCookie(ck.PtPin); err == nil {
				cookie := JdCookie{
					RWskey:    rwskey,
					QQ:        sender.UserID,
					PtKey:     ptkey,
					Available: True,
				}

				if nck.Password == "" {
					// #如果 Password 字段为空，不设置 Smsverify
					cookie.Smsverify = ""
				} else {
					// #如果 Password 字段不为空，保持 Smsverify: "false"
					cookie.Smsverify = "false"
				}

				switch sender.Type {
				case "wx", "wxg":
					cookie.WeiXin = sender.WxId
				case "tg", "tgg":
					cookie.Telegram = sender.UserID
				}
				nck.Updates(cookie)
				sender.Reply(fmt.Sprintf("扫码登录成功:%s", pin))
				(&JdCookie{}).Push(fmt.Sprintf("来自扫码登录的更新:%s", pin))
			} else {
				NewJdCookie(&ck)
				msg := fmt.Sprintf("来自扫码添加账号，账号名:%s", ck.PtPin)
				switch sender.Type {
				case "wx", "wxg":
					ck.Update("WeiXin", sender.WxId)
				case "tg", "tgg":
					ck.Update("Telegram", sender.UserID)
				}
				sender.Reply(fmt.Sprintf(msg))
				sender.Reply(ck.Query())
				(&JdCookie{}).Push(msg)
			}
			go func() {
				Save <- &JdCookie{}
			}()
			return
		} else {
			msg, _ := jsonparser.GetString(bytes, "message")
			if msg == "请先获取二维码" {
				sender.Reply("key已失效，请重新获取")
				return
			} else if msg == "" {
				sender.Reply("新账号首次扫码登录失败，请使用【登陆】，登录自动续期！")
				return
			} else if msg == "没有次数了!" {
				JdCookie{}.Push("Pro没有次数请及时签到")
				return
			} else if msg != "二维码未扫描，请扫描二维码" && msg != "请手机客户端确认登录" {
				sender.Reply(msg)
				return
			}
		}
	}
}

func NolanGetCookie(cookie string) (bool, string, string) {
	get := httplib.Post(fmt.Sprintf("%s/env/wskey", sysConfig.NolanUrl))
	get.Header("Content-Type", "application/json")
	get.Body(fmt.Sprintf("{\n  \"botApiToken\": \"%s\",\n  \"wskey\": \"%s\"\n}", sysConfig.NolanToken, cookie))
	bytes, _ := get.Bytes()
	msg, _ := jsonparser.GetString(bytes, "msg")
	appck, _ := jsonparser.GetString(bytes, "data", "appck")
	val, _ := jsonparser.GetBoolean(bytes, "success")
	if appck != "" {
		return val, msg, appck
	} else {
		time.Sleep(time.Second * 3)
		return false, "", ""
	}
}



func NolanSendSMS(phone string, sender *Sender) bool {
	sender.Reply("请耐心等待发送验证码...")
	req := httplib.Post(sysConfig.NolanUrl + "/sms/SendSMS")
	req.Header("content-type", "application/json")
	payload := fmt.Sprintf(`{"phone":"%s","botApitoken":"%s"}`, phone, sysConfig.NolanToken)

	data, err := req.Body(payload).Bytes()
	if err != nil {
		logs.Error("NolanSendSMS 请求失败: %v", err)
		sender.Reply("验证码请求失败，请稍后重试。")
		return false
	}

	success, _ := jsonparser.GetBoolean(data, "success")
	message, _ := jsonparser.GetString(data, "message")

	if success {
		sender.Reply("验证码已发送，请输入6位验证码：")
		return true
	} else {
		// 发送失败，立即退出流程
		if message == "" {
			message = "发送失败，未知错误"
		}
		sender.Reply(fmt.Sprintf("验证码发送失败：%s", message))
		return false
	}
}

func NolanSendCode(phone string, code string, sender *Sender) {
	req := httplib.Post(fmt.Sprintf("%s/sms/VerifyCode", sysConfig.NolanUrl))
	req.Header("Content-Type", "application/json; charset=utf-8")
	sprintf := fmt.Sprintf("{\n  \"phone\": \"%s\",\n  \"code\": \"%s\",\n  \"botApitoken\": \"%s\"\n}", phone, code, sysConfig.NolanToken)
	data, _ := req.Body(sprintf).Bytes()
	logs.Info(string(data))

	message, _ := jsonparser.GetString(data, "message")
	success, _ := jsonparser.GetBoolean(data, "success")
	ck, _ := jsonparser.GetString(data, "data", "ck")
	state, _ := jsonparser.GetInt(data, "data", "status")
	if success {
		ptkey := FetchJdCookieValue("pt_key", ck)
		pin := FetchJdCookieValue("pt_pin", ck)
		ck := JdCookie{
			PtPin:     pin,
			PtKey:     ptkey,
			Available: True, // 按你的要求保留原写法不修改
			QQ:        sender.UserID,
		}
		if nck, err := GetJdCookie(ck.PtPin); err == nil {
			cookie := JdCookie{
				QQ:        sender.UserID,
				PtKey:     ptkey,
				Available: True, // 按你的要求保留原写法不修改
			}

			if nck.Password == "" {
				// #如果 Password 字段为空，不设置 Smsverify
				cookie.Smsverify = ""
			} else {
				// #如果 Password 字段不为空，保持 Smsverify: "false"
				cookie.Smsverify = "false"
			}
			switch sender.Type {
			case "wx", "wxg":
				cookie.WeiXin = sender.WxId
			case "tg", "tgg":
				cookie.Telegram = sender.UserID
			}
			nck.Updates(cookie)
			sender.Reply(fmt.Sprintf("来自短信登录成功:%s", pin))
			(&JdCookie{}).Push(fmt.Sprintf("来自短信登录成功:%s", pin))
		} else {
			NewJdCookie(&ck)
			msg := fmt.Sprintf("来自短信添加账号，账号名:%s", ck.PtPin)
			switch sender.Type {
			case "wx", "wxg":
				ck.Update("WeiXin", sender.WxId)
			case "tg", "tgg":
				ck.Update("Telegram", sender.UserID)
			}
			sender.Reply(fmt.Sprintf(msg))
			sender.Reply(ck.Query())
			(&JdCookie{}).Push(msg)
		}
		smsList[sender.UserID] = nil
		go func() {
			Save <- &JdCookie{}
		}()
		return
	} else {
		if state == 555 {
			mode, _ := jsonparser.GetString(data, "data", "mode")
			if mode == "USER_ID" {
				RiskList[sender.UserID] = true
				phoneList[sender.UserID] = phone
				sender.Reply("你的账号需要验证才能登陆，请输入你的京东账号绑定的身份证前两位和后四位，最后一位如果是X，请输入大写X\n例如：31122X")
			} else if mode == "HISTORY_DEVICE" {
				sender.Reply("请使用手机进行验证后重新登录")
				smsList[sender.UserID] = nil
				return
			}
		} else if state == 404 {
			if message == "验证码输入错误" {
				sender.Reply("验证码输入错误,请重新输入")
				//smsList[sender.UserID] = nil
			} else {
				smsList[sender.UserID] = nil
				sender.Reply("你的账号可能触发了电话语音验证，请在京东官方app登录验证后再次尝试，建议直接使用【登陆】")
			}
		} else {
			smsList[sender.UserID] = nil
			sender.Reply(message)
		}
	}
}

func NolanAuthCode(phone string, code string, sender *Sender) {
	req := httplib.Post(fmt.Sprintf("%s/sms/VerifyCard", sysConfig.NolanUrl))
	req.Header("Content-Type", "application/json; charset=utf-8")
	sprintf := fmt.Sprintf("{\n  \"phone\": \"%s\",\n  \"code\": \"%s\",\n  \"botApitoken\": \"%s\"\n}", phone, code, sysConfig.NolanToken)
	data, _ := req.Body(sprintf).Bytes()
	logs.Info(string(data))

	message, _ := jsonparser.GetString(data, "message")
	success, _ := jsonparser.GetBoolean(data, "success")
	ck, _ := jsonparser.GetString(data, "data", "ck")
	state, _ := jsonparser.GetInt(data, "data", "status")

	if success {
		ptkey := FetchJdCookieValue("pt_key", ck)
		pin := FetchJdCookieValue("pt_pin", ck)
		ck := JdCookie{
			PtPin:     pin,
			PtKey:     ptkey,
			Available: True,
			QQ:        sender.UserID,
		}
		if nck, err := GetJdCookie(ck.PtPin); err == nil {
			cookie := JdCookie{
				QQ:        sender.UserID,
				PtKey:     ptkey,
				Available: True,
			}

			if nck.Password == "" {
				cookie.Smsverify = ""
			} else {
				cookie.Smsverify = "false"
			}

			switch sender.Type {
			case "wx", "wxg":
				cookie.WeiXin = sender.WxId
			case "tg", "tgg":
				cookie.Telegram = sender.UserID
			}
			nck.Updates(cookie)
			sender.Reply(fmt.Sprintf("登录成功:%s", pin))
			(&JdCookie{}).Push(fmt.Sprintf("登录成功:%s", pin))
		} else {
			NewJdCookie(&ck)
			msg := fmt.Sprintf("添加账号，账号名:%s", ck.PtPin)
			switch sender.Type {
			case "wx", "wxg":
				ck.Update("WeiXin", sender.WxId)
			case "tg", "tgg":
				ck.Update("Telegram", sender.UserID)
			}
			sender.Reply(fmt.Sprintf(msg))
			sender.Reply(ck.Query())
			(&JdCookie{}).Push(msg)

		}
		RiskList[sender.UserID] = false
		smsList[sender.UserID] = nil
		go func() {
			Save <- &JdCookie{}
		}()
		return
	} else {
		if state == 404 {
			sender.Reply(message)
		} else {
			sender.Reply(message)
			smsList[sender.UserID] = nil
			(&JdCookie{}).Push("Pro短信登录异常" + message)
			return
		}
	}
}