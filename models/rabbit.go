package models

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/beego/beego/v2/client/httplib"
	"github.com/beego/beego/v2/core/logs"
	"github.com/buger/jsonparser"
	"gorm.io/gorm"
	"math/rand"
	"strings"
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

func GetCookie(cookie string) {
	get := httplib.Post(fmt.Sprintf("http://192.168.195.53:5800/api/wsck?RabbitToken=%s", "sad5d5s6c5d5e8w6r6t6uiopfghf5s6ew5ds8c12b"))
	marshal, _ := json.Marshal(struct {
		WSCK        string `json:"wsck"`
		RabbitToken string `json:"RabbitToken"`
	}{
		WSCK:        cookie,
		RabbitToken: "3cd2db5316374ebf885a3c421f370c34",
	})
	get.Body(marshal)
	bytes, _ := get.Bytes()
	logs.Info(string(bytes))
}

func UpdateRwskey() {
	cks := GetJdCookies(func(sb *gorm.DB) *gorm.DB {
		return sb.Where(fmt.Sprintf("%s != ?", RWSKEY), "")
	})
	xx := 0
	yy := 0
	(&JdCookie{}).Push("开始定时更新转换Wskey")

	for i, ck := range cks {
		if i == len(cks)/2 {
			(&JdCookie{}).Push("Wskey已更新二分一")
		}
		time.Sleep(time.Duration(rand.Int63n(1)) * time.Second)
		//JdCookie{}.Push(fmt.Sprintf("更新账号账号，%s", ck.Nickname))
		var pinky = fmt.Sprintf("pin=%s;wskey=%s;", ck.PtPin, ck.WsKey)
		rsp, _ := getKey(pinky)
		if strings.Contains(rsp, "错误") {
			yy++
			ck.Update(Available, False)
			//ck.Push(fmt.Sprintf("年费Wskey失效账号，%s，请联系管理员", ck.PtPin))
			(&JdCookie{}).Push(fmt.Sprintf("年费Wskey失效，%s", ck.PtPin))
		} else {
			ptKey := FetchJdCookieValue("pt_key", rsp)
			ptPin := FetchJdCookieValue("pt_pin", rsp)
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
				(&JdCookie{}).Push(fmt.Sprintf("转换失败，请求超时，账号:%s", ck.PtPin))
			}
		}

	}
	go func() {
		Save <- &JdCookie{}
	}()
	(&JdCookie{}).Push(fmt.Sprintf("所有CK转换完成，共%d个,转换失败个数共%d个", xx, yy))
}
