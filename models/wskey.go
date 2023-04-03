package models

import (
	"github.com/beego/beego/v2/client/httplib"
	"github.com/beego/beego/v2/core/logs"
	"github.com/buger/jsonparser"
	"net/http"
	"net/url"
	"strings"
)

func getKey(WSCK string) string {
	var ptKey = ""
	sign := GetEnv("sign")
	if sign == "" {
		ptKey, _ = getTokenKey("https://sign.smxy.xyz/jd/sign", WSCK)
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

func getSelfSign(sign string) string {
	req := httplib.Post(sign)
	req.Param("body", "{}")
	req.Param("functionId", "genToken")
	data, _ := req.Bytes()
	getString, _ := jsonparser.GetString(data, "data", "convertUrl")
	return getString
}

func getTokenKey(sign string, WSCK string) (string, error) {
	s := getSelfSign(sign)
	logs.Info(s)
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
	logs.Info(string(data))
	tokenKey, _ := jsonparser.GetString(data, "tokenKey")
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
