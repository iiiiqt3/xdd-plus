package models

import (
	"github.com/beego/beego/v2/client/httplib"
	// "github.com/beego/beego/v2/core/logs"
	"github.com/buger/jsonparser"
	"net/http"
	"net/url"
	"strings"
	"time"
	 "strconv"

	
)

func getKey(WSCK string) string {
	var ptKey = ""
	sign := GetEnv("sign")
	if sign == "" {
		//ptKey, _ = getTokenKey("http://nsign.smxy.xyz/sign", WSCK)
		ptKey, _ = getTokenKey("http://log.smxy.xyz", WSCK)
	} else {
		ptKey, _ = getTokenKey(sign, WSCK)
	}
	if strings.Contains(ptKey, "fake") {
		return "Wskey错误"
	}
	return ptKey
}

// getNewSign 获取新的签名token，用于京东wskey登录
func getNewSign(sign string) string {
	req := httplib.Post(sign)
	req.Body("{\n  \"body\": {\"url\": \"https://plogin.m.jd.com/jd-mlogin/static/html/appjmp_blank.html\"},\n  \"fn\": \"genToken\"\n}")
	req.Header("Content-Type", "application/json")
	data, _ := req.Bytes()
	getString, _ := jsonparser.GetString(data, "body")
	return getString

}

// getTokenKey 通过wskey获取京东tokenKey，用于生成Cookie
func getTokenKey(sign string, WSCK string) (string, error) {

	var tokenKey string
	var i = 0
	for {
		i++
		s := getNewSign(sign)
		str := `https://api.m.jd.com/client.action?` + s + "&functionId=genToken"
		req := httplib.Post(str)
		req.Header("cookie", WSCK)
		req.Header("User-Agent", ua)
		req.Header("content-type", `application/x-www-form-urlencoded; charset=UTF-8`)
		req.Header("charset", `UTF-8`)
		req.Header("accept-encoding", `br,gzip,deflate`)
		req.Body(`%7B%22to%22%3A%20%22https%3A//m.jd.com%22%2C%20%22action%22%3A%20%22to%22%7D`)
		data, err := req.Bytes()
		if err != nil {
			return "", err
		}
		tokenKey, _ = jsonparser.GetString(data, "tokenKey")
		if tokenKey != "xxx" || i == 7 {
			break
		} else {
			time.Sleep(time.Duration(20) * time.Second)
		}
	}
	ptKey, _ := appjmp(tokenKey)
	return ptKey, nil
}

func appjmp(tokenKey string) (string, error) {

	v := url.Values{}
	v.Add("tokenKey", tokenKey)
	v.Add("to", `https://plogin.m.jd.com/jd-mlogin/static/html/appjmp_blank.html`)
	v.Add("client_type", "android")
	v.Add("appid", "879")
	v.Add("appup_type", "1")
	req := httplib.Get(`https://un.m.jd.com/cgi-bin/app/appjmp?` + v.Encode())
	req.Header("User-Agent", ua)
	req.Header("accept", `accept:text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9`)
	req.Header("x-requested-with", "com.jingdong.app.mall")
	req.SetCheckRedirect(func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	})
	rsp, err := req.Response()
	if err != nil {
		return "", err
	}
	cookies := strings.Join(rsp.Header.Values("Set-Cookie"), " ")
	return cookies, nil
}



func LoginSelect(sender *Sender, msg chan string) {
	defer func() {
		delete(loginList, sender.UserID)
	}()

	timeout := time.After(30 * time.Second)

	for {
		select {
		case <-timeout:
			sender.Reply("⏰ 操作超时，已自动退出登录流程。")
			close(msg)
			return

		case n, ok := <-msg:
			if !ok {
				sender.Reply("通道已关闭，退出登录流程")
				return
			}

			if n == "q" {
				sender.Reply("已退出登录流程")
				close(msg)
				return
			}

			num, err := strconv.Atoi(n)
			if err != nil {
				sender.Reply("请输入数字 1 或 2，或输入 q 退出")
				continue
			}

			switch num {
			case 1:
				close(msg)
				c2 := make(chan string)
				smsList[sender.UserID] = c2
				sender.Reply("请输入手机号...\n上车后请到京东-我的-支付设置，关闭小额免密，同时开启虚拟资产验密\n回复'q'退出登录流程")
				go SmsSelect(sender, c2, "Nolan")
				return
			case 2:
				close(msg)
				c2 := make(chan string)
				wxJdList[sender.UserID] = c2
				go handleWxJdLogin(sender, c2)
				return
			default:
				sender.Reply("无效输入，请输入数字 1 或 2，或输入 q 退出")
			}
		}
	}
}
