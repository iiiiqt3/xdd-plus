package models

import (
	"github.com/beego/beego/v2/client/httplib"
	"github.com/beego/beego/v2/core/logs"
	"github.com/buger/jsonparser"
	"net/http"
	"net/url"
	"strings"
	"time"
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

/*
下面是sign接口
*/

//func getSelfSign(sign string) string {
//	req := httplib.Post(sign)
//	req.Param("body", "{}")
//	req.Param("functionId", "genToken")
//	data, _ := req.Bytes()
//	getString, _ := jsonparser.GetString(data, "data", "convertUrl")
//	return getString
//}

func getNewSign(sign string) string {
	req := httplib.Post(sign)
	req.Body("{\n  \"body\": {\"url\": \"https://plogin.m.jd.com/jd-mlogin/static/html/appjmp_blank.html\"},\n  \"fn\": \"genToken\"\n}")
	req.Header("Content-Type", "application/json")
	data, _ := req.Bytes()
	getString, _ := jsonparser.GetString(data, "body")
	return getString

}

func getTokenKey(sign string, WSCK string) (string, error) {

	var tokenKey string
	var i = 0
	for {
		i++
		//s := getSelfSign(sign)
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
	//ptKey := FetchJdCookieValue("pt_key", cookies)
	//logs.Info(cookies)
	return cookies, nil
}

func LoginSelect(sender *Sender, msg chan string) {
	for {
		n, ok := <-msg
		//说明发送方关闭了channel
		if !ok {
			break
		}

		if Config.QQID == 764763902 {
			if n == "q" {
				//todo退出流程
				loginList[sender.UserID] = nil
				close(msg)
				return
			}

			//var login LoginSelectType
			//if len(msg) > 1 {
			//	login = GetLoginSelectTypeByName(n)
			//} else {
			//	itoa, _ := strconv.Atoi(n)
			//	login = GetLoginSelectType(itoa)
			//}

			//请选择登录渠道:
			// 1:兔子京东扫码
			// 2:Nolan京东扫码
			// 3:BBK京东扫码
			// 4:BBK微信扫码

			//switch login.Type {
			//case 1:
			//	loginList[sender.UserID] = nil
			//	RabbitUrl = GetEnv("RabbitUrl")
			//	RabbitApiToken = GetEnv("RabbitApiToken")
			//	RabbitToken = GetEnv("RabbitToken")
			//	if RabbitUrl == "" || RabbitApiToken == "" || RabbitToken == "" {
			//		logs.Error("RabbitUrl or RabbitToken is empty")
			//		sender.Reply("渠道尚未配置")
			//		return
			//	}
			//	RabbitGetJdQrImg(sender)
			//case 2:
			//	loginList[sender.UserID] = nil
			//	NolanUrl = GetEnv("NolanUrl")
			//	NolanToken = GetEnv("NolanToken")
			//	if NolanUrl == "" || NolanToken == "" {
			//		logs.Error("NolanUrl or NolanToken is empty")
			//		sender.Reply("渠道尚未配置")
			//		return
			//	}
			//	NolanGetJdQrImg(sender)
			//case 3:
			//	loginList[sender.UserID] = nil
			//	BBKJdUrl = GetEnv("BBKJdUrl")
			//	if BBKJdUrl == "" {
			//		logs.Error("BBKJdUrl is empty")
			//		sender.Reply("渠道尚未配置")
			//		return
			//	}
			//	BBKGetJdQrImg(sender)
			//case 4:
			//	loginList[sender.UserID] = nil
			//	BBKWxUrl = GetEnv("BBKWxUrl")
			//	if BBKWxUrl == "" {
			//		logs.Error("BBKWxUrl is empty")
			//		sender.Reply("渠道尚未配置")
			//		return
			//	}
			//	BBKGetWxQrImg(sender)
			//default:
			//	sender.Reply("无匹配渠道，如需回复'q'退出登录流程")
			//}
		} else if Config.QQID == 413255735 {
			//请选择登录渠道:
			// 1:兔子京东扫码
			// 2:Nolan京东扫码
			// 3:BBK京东扫码
			// 4:BBK微信扫码

			switch n {
			case "扫码登录", "1":
				loginList[sender.UserID] = nil
				if sysConfig.RabbitUrl == "" || sysConfig.RabbitApiToken == "" || sysConfig.RabbitToken == "" {
					logs.Error("RabbitUrl or RabbitToken is empty")
					sender.Reply("渠道尚未配置")
					return
				}
				RabbitGetJdQrImg(sender)
			case "2", "短信登录":
				loginList[sender.UserID] = nil
				c2 := make(chan string)
				smsList[sender.UserID] = c2
				go SmsSelect(sender, c2, "Rabbit")
			case "q":
				loginList[sender.UserID] = nil
				close(msg)
			default:
				sender.Reply("无匹配渠道，如需回复'q'退出登录流程")
			}
		} else {
			//请选择登录渠道:
			// 1:兔子京东扫码
			// 2:Nolan京东扫码
			// 3:BBK京东扫码
			// 4:BBK微信扫码

			switch n {
			case "兔子京东扫码", "1":
				loginList[sender.UserID] = nil
				if sysConfig.RabbitUrl == "" || sysConfig.RabbitApiToken == "" || sysConfig.RabbitToken == "" {
					logs.Error("RabbitUrl or RabbitToken is empty")
					sender.Reply("渠道尚未配置")
					return
				}
				RabbitGetJdQrImg(sender)
			case "Nolan京东扫码", "2":
				loginList[sender.UserID] = nil
				if sysConfig.NolanUrl == "" || sysConfig.NolanToken == "" {
					logs.Error("NolanUrl or NolanToken is empty")
					sender.Reply("渠道尚未配置")
					return
				}
				NolanGetJdQrImg(sender)
			case "BBK京东扫码", "3":
				loginList[sender.UserID] = nil
				if sysConfig.BBKJdUrl == "" {
					logs.Error("BBKJdUrl is empty")
					sender.Reply("渠道尚未配置")
					return
				}
				BBKGetJdQrImg(sender)
			case "BBK微信扫码", "4":
				loginList[sender.UserID] = nil
				if sysConfig.BBKWxUrl == "" {
					logs.Error("BBKWxUrl is empty")
					sender.Reply("渠道尚未配置")
					return
				}
				BBKGetWxQrImg(sender)
			case "5":
				loginList[sender.UserID] = nil
				c2 := make(chan string)
				smsList[sender.UserID] = c2
				go SmsSelect(sender, c2, "Rabbit")
			case "6", "Pro短信":
				loginList[sender.UserID] = nil
				c2 := make(chan string)
				smsList[sender.UserID] = c2
				sender.Reply("请输入手机号")
				go SmsSelect(sender, c2, "Nolan")
			case "q":
				loginList[sender.UserID] = nil
				sender.Reply("您已退出登录流程")
				close(msg)
			default:
				sender.Reply("无匹配渠道，如需回复'q'退出登录流程")
			}
		}

	}
}
