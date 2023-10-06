package models

import (
	"encoding/json"
	"fmt"
	"github.com/beego/beego/v2/client/httplib"
	"github.com/beego/beego/v2/core/logs"
	"github.com/buger/jsonparser"
	"github.com/skip2/go-qrcode"
	"strconv"
	"time"
)

func NolanGetJdQrImg(sender *Sender) {
	if sysConfig.NolanUrl == "" || sysConfig.NolanToken == "" {
		logs.Error("NolanUrl or NolanToken is empty")
		return
	}

	//http://192.168.195.53:5016/qr/GetQRKey
	//https://qr.m.jd.com/p?k=${qrcode_info.value.QRCodeKey
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

		//sender.Reply(NolanLJToKL("https://qr.m.jd.com/p?k="+key, "京东快捷登录"))

		//lj := LJtoLJ("https://qr.m.jd.com/p?k=" + key)
		//url, _ := jsonparser.GetString(lj, "code")
		//logs.Info(url)
		//sender.Reply(NolanLJToKL(url, "京东快捷登录"))

		logs.Info(key)
		//if Config.QQID == 764763903 {
		//	SendQQMsg(QQMessage{
		//		Action: "send_msg",
		//		QQMsg: struct {
		//			MessageType string `json:"message_type"`
		//			UserId      int    `json:"user_id"`
		//			GroupID     int    `json:"group_id"`
		//			Message     string `json:"message"`
		//		}{
		//			UserId:  sender.UserID,
		//			GroupID: 0,
		//			Message: fmt.Sprintf("[CQ:share,url=%s,title=京东快捷登录]", "https://qr.m.jd.com/p?k="+key),
		//		},
		//		Echo: "",
		//	})
		//
		//}
		sender.Reply("请使用京东APP扫描，150秒失效")
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
		//http://192.168.195.53:5016/qr/CheckQRKey
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
			}
			if nck, err := GetJdCookie(ck.PtPin); err == nil {
				nck.Updates(JdCookie{RWskey: rwskey, QQ: sender.UserID, PtKey: ptkey, Available: True})
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
		} else {
			msg, _ := jsonparser.GetString(bytes, "message")
			if msg == "请先获取二维码" {
				sender.Reply("key已失效，请重新获取")
				return
			} else if msg == "没有次数了!" {
				JdCookie{}.Push("Pro没有次数请及时签到")
				return
			} else if msg != "二维码未扫描，请扫描二维码" && msg != "请手机客户端确认登录" {
				sender.Reply("渠道维护，请使用其他登录方式。")
				return
			}
		}
	}
}

func NolanGetCookie(cookie string) (bool, string, string) {
	get := httplib.Post(fmt.Sprintf("%s/env/wskey", sysConfig.NolanUrl))
	get.Header("Content-Type", "application/json")
	//get.Body(fmt.Sprintf("{\n  \"botApiToken\": \"%s\",\n  \"wskey\": \"%s\"\n}", NolanToken, cookie))
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

func NolanSendSMS(phone string, sender *Sender) {
	if sysConfig.NolanUrl == "" || sysConfig.NolanToken == "" {
		logs.Error("NolanUrl or NolanToken is empty")
		return
	}

	sender.Reply("请耐心等待...")
	req := httplib.Post(sysConfig.NolanUrl + "/sms/SendSMS")
	req.Header("content-type", "application/json")
	data, _ := req.Body(`{"Phone":"` + phone + `","botApitoken":` + sysConfig.NolanToken + `}`).Bytes()
	message, _ := jsonparser.GetString(data, "message")
	success, _ := jsonparser.GetBoolean(data, "success")
	status, _ := jsonparser.GetInt(data, "data", "status")
	if success {
		logs.Info(strconv.Itoa(sender.UserID))
		sender.Reply("请输入6位验证码：")
		return
	} else {
		sender.Reply("滑块失败，请网页登录")
		return
	}
}

func NolanSendCode(phone string, code string, sender *Sender) {
	sender.Reply("请耐心等待...")
	req := httplib.Post(fmt.Sprintf("%s/sms/VerifyCode", sysConfig.RabbitUrl))
	req.Header("Content-Type", "application/json; charset=utf-8")
	data, _ := req.Body(fmt.Sprintf("{\n    \"phone\": %s,\n    \"code\": \"%s\",\n    \"botApitoken\": %s\n}", phone, code, sysConfig.NolanToken)).Bytes()
	logs.Info(string(data))

	message, _ := jsonparser.GetString(data, "message")
	success, _ := jsonparser.GetBoolean(data, "success")
	ck, _ := jsonparser.GetString(data, "data", "ck")
	state, _ := jsonparser.GetInt(data, "data", "status")
	if success {
		ptkey := FetchJdCookieValue("pt_key", ck)
		pin := FetchJdCookieValue("pt_pin", ck)
		ck := JdCookie{
			PtPin: pin,
			PtKey: ptkey,
		}
		if nck, err := GetJdCookie(ck.PtPin); err == nil {
			nck.Updates(JdCookie{QQ: sender.UserID, PtKey: ptkey})
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
		if state == 555 {
			mode, _ := jsonparser.GetString(data, "data", "mode")
			if mode == "USER_ID" {

			} else if mode == "HISTORY_DEVICE" {

			}
			sender.Reply(message)
			smsList[sender.UserID] = nil
		} else {
			smsList[sender.UserID] = nil
			sender.Reply(message)
		}
	}
}
