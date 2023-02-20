package models

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/beego/beego/v2/client/httplib"
	"github.com/beego/beego/v2/core/logs"
	"github.com/buger/jsonparser"
	"time"
)

func getJdQrImg(sender *Sender) {
	get := httplib.Post(fmt.Sprintf("http://192.168.195.53:5800/api/BeanQrCode?token=%s", "sad5d5s6c5d5e8w6r6t6uiopfghf5s6ew5ds8c12b"))
	bytes, _ := get.Bytes()
	code, _ := jsonparser.GetInt(bytes, "code")
	if code == 0 {
		qr, _ := jsonparser.GetString(bytes, "qr")
		key, _ := jsonparser.GetString(bytes, "QRCodeKey")
		decodeStr, _ := base64.StdEncoding.DecodeString(qr)
		sender.SendImg(decodeStr)
		sender.Reply("请使用京东APP扫码")
		go getJDQrStatus(key, sender)
	} else {
		logs.Info(string(bytes))
		sender.Reply("获取扫码失败")
	}
}

func getJDQrStatus(cookie string, sender *Sender) {
	for {
		time.Sleep(time.Second * time.Duration(5))
		get := httplib.Post(fmt.Sprintf("http://192.168.195.53:5800/api/QrCheck?token=%s", "sad5d5s6c5d5e8w6r6t6uiopfghf5s6ew5ds8c12b"))
		marshal, _ := json.Marshal(struct {
			QRCodeKey string `json:"QRCodeKey"`
			Qlkey     string `json:"qlkey"`
		}{
			QRCodeKey: cookie,
			Qlkey:     string(0),
		},
		)
		get.Body(marshal)
		bytes, _ := get.Bytes()
		code, _ := jsonparser.GetInt(bytes, "code")
		data, _ := jsonparser.GetString(bytes, "wskey")
		pin, _ := jsonparser.GetString(bytes, "pin")
		msg, _ := jsonparser.GetString(bytes, "msg")
		logs.Info(string(bytes))
		if code == 502 || code == 503 || code == 403 || code == 54 {
			sender.Reply(msg)
			return
		} else if code == 200 {
			//cookie := fmt.Sprintf("pin=%s;wskey=%s;\n", pin, data)
			ck := JdCookie{
				PtPin:  pin,
				RWskey: data,
			}
			if nck, err := GetJdCookie(ck.PtPin); err == nil {
				nck.Update(RWSKEY, data)
				nck.Update(QQ, sender.UserID)
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
			return
		}

	}
}
