package models

import (
	"encoding/base64"
	"fmt"
	"github.com/beego/beego/v2/client/httplib"
	"github.com/buger/jsonparser"
	"io/ioutil"
	"strings"
	"time"
)

func BBKGetWxQrImg(sender *Sender) {
	if sysConfig.BBKWxUrl == "" {
		Error("BBKWxUrl is empty")
		return
	}
	get := httplib.Get(fmt.Sprintf("%s/d/getQR?t=%d", sysConfig.BBKWxUrl, time.Now().Unix()))
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
		time.Sleep(time.Second * time.Duration(2))
		get := httplib.Get(fmt.Sprintf("%s/d/status?t=%d", sysConfig.BBKWxUrl, time.Now().Unix()))
		get.Header("Cookie", cookie)
		Info(cookie)
		bytes, _ := get.Bytes()
		Info(string(bytes))
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
			appck := getKey(data)
			ptKey := FetchJdCookieValue("pt_key", appck)
			ptPin := FetchJdCookieValue("pt_pin", appck)
			ck := JdCookie{
				PtKey: ptKey,
				PtPin: ptPin,
			}
			if strings.Contains(appck, "fake") {
				//todo 失效账号处理
				ck.Updates(JdCookie{Available: False})
			} else {
				if ptPin != "" || ptKey != "" {
					if nck, err := GetJdCookie(ck.PtPin); err == nil {
						nck.Updates(JdCookie{QQ: sender.UserID, PtKey: ptKey})
						sender.Reply(fmt.Sprintf("登录成功:%s", ptPin))
						(&JdCookie{}).Push(fmt.Sprintf("登录成功:%s", ptPin))
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
				} else {
					(&JdCookie{}).Push(fmt.Sprintf("转换失败，请求超时，账号:%s", ptPin))
				}
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
	if sysConfig.BBKJdUrl == "" {
		Error("BBKJdUrl is empty")
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
		Info(cookie)
		bytes, _ := get.Bytes()
		Info(string(bytes))
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

func BBKGetCookie(cookie string) (bool, string, string) {
	if sysConfig.BBKToken == "" || sysConfig.BBKJdUrl == "" {
		Error("BBKToken or BBKJdUrl is empty")
		return false, "", ""
	}
	//http://你的IP:3081/d/convert?pin=xxx&wskey=xxx&token=机器人token
	pin := FetchJdCookieValue("pin", cookie)
	rwskey := FetchJdCookieValue("wskey", cookie)
	get := httplib.Get(fmt.Sprintf("%s/d/convert?pin=%s&wskey=%s&token=%s", sysConfig.BBKJdUrl, pin, rwskey, sysConfig.BBKToken))
	bytes, _ := get.Bytes()
	Info(string(bytes))
	data, _ := jsonparser.GetString(bytes, "data")
	code, _ := jsonparser.GetInt(bytes, "code")
	msg, _ := jsonparser.GetString(bytes, "msg")
	errorMsg, _ := jsonparser.GetString(bytes, "errorMsg")
	if code == 200 {
		return true, msg, data
	} else {
		Info(string(bytes))
		time.Sleep(time.Second * 3)
		return false, errorMsg, data
	}
}
