package models

import (
	"encoding/json"
	"fmt"
	"github.com/beego/beego/v2/client/httplib"
	"github.com/beego/beego/v2/core/logs"
	"github.com/buger/jsonparser"
	"github.com/skip2/go-qrcode"
	"time"
)

var NolanUrl string
var NolanToken string

func NolanGetJdQrImg(sender *Sender) {
	NolanUrl = GetEnv("NolanUrl")
	NolanToken = GetEnv("NolanToken")
	if NolanUrl == "" || NolanToken == "" {
		logs.Error("NolanUrl or NolanToken is empty")
		return
	}

	//http://192.168.195.53:5016/qr/GetQRKey
	//https://qr.m.jd.com/p?k=${qrcode_info.value.QRCodeKey
	get := httplib.Post(fmt.Sprintf("%s/qr/GetQRKey", NolanUrl))
	get.Header("Content-Type", "application/json")
	get.Body(fmt.Sprintf("{\n  \"botApitoken\": \"%s\"\n}", NolanToken))
	bytes, _ := get.Bytes()
	logs.Info(string(bytes))

	code, _ := jsonparser.GetBoolean(bytes, "success")
	if code {
		key, _ := jsonparser.GetString(bytes, "data", "key")
		var png []byte
		png, _ = qrcode.Encode("https://qr.m.jd.com/p?k="+key, qrcode.Medium, 256)
		sender.SendImg(png)
		logs.Info(key)
		sender.Reply("请使用京东APP扫码，150秒失效")
		go NolanGetJDQrStatus(key, sender)
	} else {
		logs.Info(string(bytes))
		sender.Reply("获取扫码失败")
	}
}

func NolanGetJDQrStatus(cookie string, sender *Sender) {

	type NolanRWskey struct {
		Status       int           `json:"status"`
		Mode         string        `json:"mode"`
		Ck           string        `json:"ck"`
		Rwskey       string        `json:"rwskey"`
		AccessToken  string        `json:"accessToken"`
		RefreshToken string        `json:"refreshToken"`
		Roles        []interface{} `json:"roles"`
		Img          string        `json:"img"`
		Username     string        `json:"username"`
		Expires      time.Time     `json:"expires"`
	}
	for {
		time.Sleep(time.Second * time.Duration(5))
		//http://192.168.195.53:5016/qr/CheckQRKey
		get := httplib.Post(fmt.Sprintf("%s/qr/CheckQRKey", NolanUrl))
		get.Header("Content-Type", "application/json")
		get.Body(fmt.Sprintf("{\n  \"qrkey\": \"%s\",\n  \"botApitoken\": \"%s\"\n}", cookie, NolanToken))
		bytes, _ := get.Bytes()
		code, _ := jsonparser.GetBoolean(bytes, "success")
		logs.Info(string(bytes))
		if code {
			data, _ := jsonparser.GetString(bytes, "data")
			logs.Info(data)
			nolan := &NolanRWskey{}
			json.Unmarshal([]byte(data), nolan)

			//pin, _ := jsonparser.GetString(bytes, "pin")
			//pin = url.QueryEscape(pin)
			//var pinky = fmt.Sprintf("pin=%s;wskey=%s;", pin, data)
			//_, _, appck := RabbitGetCookie(pinky)
			//ptkey := FetchJdCookieValue("pt_key", appck)
			//ck := JdCookie{
			//	PtPin:  pin,
			//	PtKey:  ptkey,
			//	RWskey: data,
			//}
			//if nck, err := GetJdCookie(ck.PtPin); err == nil {
			//	nck.Update(RWSKEY, data)
			//	nck.Update(QQ, sender.UserID)
			//	nck.Update(PtKey, ptkey)
			//	sender.Reply(fmt.Sprintf("登录成功:%s", pin))
			//	(&JdCookie{}).Push(fmt.Sprintf("登录成功:%s", pin))
			//} else {
			//	NewJdCookie(&ck)
			//	msg := fmt.Sprintf("添加账号，账号名:%s", ck.PtPin)
			//	if sender.IsQQ() || sender.IsQQ() {
			//		ck.Update(QQ, sender.UserID)
			//	}
			//	sender.Reply(fmt.Sprintf(msg))
			//	sender.Reply(ck.Query())
			//	(&JdCookie{}).Push(msg)
			//}
			return
		} else {
			msg, _ := jsonparser.GetString(bytes, "message")
			if msg != "二维码未扫描，请扫描二维码" && msg != "请手机客户端确认登录" {
				sender.Reply(msg)
				return
			}
		}
	}
}
