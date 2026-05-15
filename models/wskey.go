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
    for {
        select {
        case n, ok := <-msg:
            if !ok {
                sender.Reply("通道已关闭，退出登录流程")
                loginList[sender.UserID] = nil
                return
            }

            // 判断是否输入了 "q" 退出
            if n == "q" {
                loginList[sender.UserID] = nil
                sender.Reply("已退出登录流程")
                close(msg)
                return
            }

            // num, err := strconv.Atoi(n)
            _, err := strconv.Atoi(n)
            if err != nil {
                sender.Reply("当前仅支持短信登录，请直接发送【短信登录】或【登录】进行操作")
                loginList[sender.UserID] = nil
                return
            }

            // if Config.QQID == 694738267 {
            //     switch num {
            //     case 1:
            //         loginList[sender.UserID] = nil
            //         Auto := &UserSession{}
            //         Autojdck(sender, Auto)
            //     case 2:
            //         loginList[sender.UserID] = nil
            //         value := GetEnv("grouplogin")
            //         if value == "" && (sender.Type == "qqg" || sender.Type == "wxg") {
            //             logs.Error("se grouplogin")
            //             sender.Reply("短信登录请添加本机器人好友后，私聊登录，避免信息泄露")
            //             return
            //         }
            //         c2 := make(chan string)
            //         smsList[sender.UserID] = c2

            //         sender.Reply("上车后请到京东-我的-支付设置，关闭小额免密，同时开启虚拟资产验密")

            //         go SmsSelect(sender, c2, "Nolan")
            //     case 3:
            //         loginList[sender.UserID] = nil
            //         NolanGetJdQrImg(sender)
            //     default:
            //         sender.Reply("无效输入，请输入数字 1, 2, 3 或 'q' 退出登录流程")
            //     }
            // } else {
            //     sender.Reply("无效QQID，无法处理登录请求")
            //     close(msg)
            //     return
            // }
            loginList[sender.UserID] = nil
            sender.Reply("当前仅支持短信登录，请直接发送【短信登录】或【登录】进行操作")
        }
    }
}
