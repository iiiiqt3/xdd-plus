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

var BBKWxUrl string
var BBKJdUrl string

func BBKGetWxQrImg(sender *Sender) {
	BBKWxUrl = GetEnv("BbkWxUrl")
	if BBKWxUrl == "" {
		logs.Error("BbkWxUrl is empty")
		return
	}
	get := httplib.Get(fmt.Sprintf("%s/d/getQR?t=%d", BBKWxUrl, time.Now().Unix()))
	response, _ := get.Response()
	cookies := response.Cookies()
	ck := cookies[0].Name + "=" + cookies[0].Value
	all, _ := ioutil.ReadAll(response.Body)
	val, _ := jsonparser.GetString(all, "data", "qr")
	replaceAll := strings.ReplaceAll(val, "data:image/jpeg;base64,", "")
	decodeStr, _ := base64.StdEncoding.DecodeString(replaceAll)
	sender.SendImg(decodeStr)
	sender.Reply("请使用微信扫码，后摄像头,有效期为160秒")
	go BBKGetWxQrStatus(ck, sender)
}

func BBKGetWxQrStatus(cookie string, sender *Sender) {
	for {
		get := httplib.Get(fmt.Sprintf("%s/d/status?t=%d", BBKWxUrl, time.Now().Unix()))
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
			_, _, appck := RabbitGetCookie(data)
			ptkey := FetchJdCookieValue("pt_key", appck)
			pin := FetchJdCookieValue("pin", appck)
			rwskey := FetchJdCookieValue("wskey", data)
			ck := JdCookie{
				PtPin:  pin,
				PtKey:  ptkey,
				RWskey: rwskey,
			}
			if nck, err := GetJdCookie(ck.PtPin); err == nil {
				nck.Updates(JdCookie{RWskey: rwskey, QQ: sender.UserID, PtKey: ptkey})
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

func BBKGetJdQrImg(sender *Sender) {
	BBKJdUrl = GetEnv("BBKJdUrl")
	if BBKJdUrl == "" {
		logs.Error("BBKJdUrl is empty")
		return
	}
	get := httplib.Get(fmt.Sprintf("%s/d/getQR?t=%d", BBKJdUrl, time.Now().Unix()))
	response, _ := get.Response()
	cookies := response.Cookies()
	ck := cookies[0].Name + "=" + cookies[0].Value
	all, _ := ioutil.ReadAll(response.Body)
	val, _ := jsonparser.GetString(all, "data", "qr")
	replaceAll := strings.ReplaceAll(val, "data:image/jpeg;base64,", "")
	decodeStr, _ := base64.StdEncoding.DecodeString(replaceAll)
	sender.SendImg(decodeStr)
	sender.Reply("请使用京东APP扫码,有效期为160秒")
	go BBKGetJdQrStatus(ck, sender)
}

func BBKGetJdQrStatus(cookie string, sender *Sender) {
	for {
		get := httplib.Get(fmt.Sprintf("%s/d/status?t=%d", BBKJdUrl, time.Now().Unix()))
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
			_, _, appck := RabbitGetCookie(data)
			ptkey := FetchJdCookieValue("pt_key", appck)
			pin := FetchJdCookieValue("pin", appck)
			rwskey := FetchJdCookieValue("wskey", data)
			ck := JdCookie{
				PtPin:  pin,
				PtKey:  ptkey,
				RWskey: rwskey,
			}
			if nck, err := GetJdCookie(ck.PtPin); err == nil {
				nck.Updates(JdCookie{RWskey: rwskey, QQ: sender.UserID, PtKey: ptkey})
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
