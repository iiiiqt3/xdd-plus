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
	"net/url"
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
		//[CQ:image,file=http://baidu.com/1.jpg,type=show,id=40004]

		SendQQMsg(QQMessage{
			UserId:  764763903,
			GroupID: 0,
			Message: fmt.Sprintf("[CQ:image,file=base64://%s,type=show,id=40004]", qr),
		})
		sender.SendImg(decodeStr)

		sender.Reply("请使用京东APP扫码，150秒失效")
		go getJDQrStatus(key, sender)
	} else {
		logs.Info(string(bytes))
		sender.Reply("获取扫码失败，目前登录人数过多，请一两分钟后再试")
	}
}

func getJDQrStatus(cookie string, sender *Sender) {
	for {
		time.Sleep(time.Second * time.Duration(5))
		get := httplib.Post(fmt.Sprintf("http://192.168.195.53:5800/api/QrCheck?token=%s", "sad5d5s6c5d5e8w6r6t6uiopfghf5s6ew5ds8c12b"))
		marshal, _ := json.Marshal(struct {
			QRCodeKey string `json:"QRCodeKey"`
			Qlkey     int    `json:"qlkey"`
		}{
			QRCodeKey: cookie,
			Qlkey:     0,
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
			ptPin := url.QueryEscape(pin)
			var pinky = fmt.Sprintf("pin=%s;wskey=%s;", pin, data)
			_, _, appck := GetCookie(pinky)
			ptkey := FetchJdCookieValue("pt_key", appck)
			ck := JdCookie{
				PtPin:  ptPin,
				PtKey:  ptkey,
				RWskey: data,
			}
			if nck, err := GetJdCookie(ck.PtPin); err == nil {
				nck.Updates(JdCookie{RWskey: data, QQ: sender.UserID, PtKey: ptkey})
				sender.Reply(fmt.Sprintf("登录成功:%s", ptPin))
				(&JdCookie{}).Push(fmt.Sprintf("登录成功:%s", ptPin))
			} else {
				NewJdCookie(&ck)
				msg := fmt.Sprintf("添加账号，账号名:%s", ptPin)
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

func GetCookie(cookie string) (bool, string, string) {
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
	val, err := jsonparser.GetBoolean(bytes, "success")
	if err != nil {
		logs.Info(val)
		logs.Info(err)
		return false, "请求异常", ""
	}
	if val {
		msg, _ := jsonparser.GetString(bytes, "msg")
		appck, _ := jsonparser.GetString(bytes, "data", "appck")
		return val, msg, appck
	} else {
		logs.Info(string(bytes))
		return val, "Wskey失效", ""
	}
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
		time.Sleep(time.Duration(rand.Int63n(2)) * time.Second)
		//JdCookie{}.Push(fmt.Sprintf("更新账号账号，%s", ck.Nickname))
		pin, _ := url.QueryUnescape(ck.PtPin)
		var pinky = fmt.Sprintf("pin=%s;wskey=%s;", pin, ck.RWskey)
		rsp, msg, appck := GetCookie(pinky)
		if rsp {
			ptKey := FetchJdCookieValue("pt_key", appck)
			if ptKey != "" {
				xx++
				ck.Updates(JdCookie{PtKey: ptKey, Available: True})
				msg := fmt.Sprintf("定时更新账号，%s", ck.PtPin)
				logs.Info(msg)
			} else {
				yy++
				logs.Info(appck)
				(&JdCookie{}).Push(fmt.Sprintf("Wskey失效，账号:%s", ck.PtPin))
			}

		} else {
			yy++
			switch msg {
			case "Wskey失效":
				ck.Updates(JdCookie{RWskey: "null", Available: False})
				//ck.Push()
				(&JdCookie{}).Push(fmt.Sprintf("Wskey失效，账号:%s", ck.PtPin))
			case "请求异常":
				(&JdCookie{}).Push(fmt.Sprintf("请求异常，账号:%s", ck.PtPin))
			default:
				(&JdCookie{}).Push(fmt.Sprintf("特殊异常，账号:%s", ck.PtPin))
			}
		}

	}
	go func() {
		Save <- &JdCookie{}
	}()
	(&JdCookie{}).Push(fmt.Sprintf("所有CK转换完成，共%d个,转换失败个数共%d个", xx, yy))
}
