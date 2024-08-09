package models

import (
	"encoding/base64"
	"fmt"
	"github.com/beego/beego/v2/client/httplib"
	"github.com/beego/beego/v2/core/logs"
	"github.com/buger/jsonparser"
	"io/ioutil"
	"strings"
	"time"
)

func BBKGetJdQrImg(sender *Sender) {
	if sysConfig.BBKJdUrl == "" {
		logs.Error("BBKJdUrl is empty")
		return
	}
	get := httplib.Get(fmt.Sprintf("%s/d/getQR?t=%d", sysConfig.BBKJdUrl, time.Now().Unix()))
	response, _ := get.Response()
	cookies := response.Cookies()
	ck := cookies[0].Name + "=" + cookies[0].Value
	all, _ := ioutil.ReadAll(response.Body)

	key, _ := jsonparser.GetString(all, "data", "qrUrl")
	sender.Reply(NolanLJToKL(key, "京东快捷登录"))

	val, _ := jsonparser.GetString(all, "data", "qr")
	replaceAll := strings.ReplaceAll(val, "data:image/jpeg;base64,", "")
	decodeStr, _ := base64.StdEncoding.DecodeString(replaceAll)
	sender.SendImg(decodeStr)
	sender.Reply("请使用京东APP扫描,有效期为160秒")
	go BBKGetJdQrStatus(ck, sender)
}

func BBKGetJdQrStatus(cookie string, sender *Sender) {
	for {
		time.Sleep(time.Second * time.Duration(2))
		get := httplib.Get(fmt.Sprintf("%s/d/status?t=%d", sysConfig.BBKJdUrl, time.Now().Unix()))
		get.Header("Cookie", cookie)
		logs.Info(cookie)
		bytes, _ := get.Bytes()
		logs.Info(string(bytes))
		code, _ := jsonparser.GetInt(bytes, "code")
		errorMsg, _ := jsonparser.GetString(bytes, "errorMsg")
		data, _ := jsonparser.GetString(bytes, "data", "wskey")
		if code == 500 || code == 202 {
			sender.Reply(errorMsg)
			return
		} else if code == 408 {
			sender.Reply("已超时，扫码结束")
			return
		} else if code == 410 && data != "" {
			_, _, appck := BBKGetCookie(data)
			ptkey := FetchJdCookieValue("pt_key", appck)
			pin := FetchJdCookieValue("pt_pin", appck)
			rwskey := FetchJdCookieValue("wskey", data)
			ck := JdCookie{
				PtPin:  pin,
				PtKey:  ptkey,
				RWskey: rwskey,
			}
			if nck, err := GetJdCookie(ck.PtPin); err == nil {
				nck.Updates(JdCookie{RWskey: rwskey, QQ: sender.UserID, PtKey: ptkey, WeiXin: sender.WxId})
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
		} else if code == 429 {
			return
		}
	}

}

func BBKGetCookie(cookie string) (bool, string, string) {
	if sysConfig.BBKToken == "" || sysConfig.BBKJdUrl == "" {
		logs.Error("BBKToken or BBKJdUrl is empty")
		return false, "", ""
	}
	//http://192.168.195.53:5016/env/wskey
	//http://你的IP:3081/d/convert?pin=xxx&wskey=xxx&token=机器人token
	pin := FetchJdCookieValue("pin", cookie)
	rwskey := FetchJdCookieValue("wskey", cookie)
	get := httplib.Get(fmt.Sprintf("%s/d/convert?pin=%s&wskey=%s&token=%s", sysConfig.BBKJdUrl, pin, rwskey, sysConfig.BBKToken))
	bytes, _ := get.Bytes()
	logs.Info(string(bytes))
	data, _ := jsonparser.GetString(bytes, "data")
	code, _ := jsonparser.GetInt(bytes, "code")
	msg, _ := jsonparser.GetString(bytes, "msg")
	errorMsg, _ := jsonparser.GetString(bytes, "errorMsg")
	if code == 200 {
		return true, msg, data
	} else {
		logs.Info(string(bytes))
		time.Sleep(time.Second * 3)
		return false, errorMsg, data
	}
}
