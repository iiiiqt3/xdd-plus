package models

import (
	"github.com/beego/beego/v2/client/httplib"
	"github.com/beego/beego/v2/core/logs"
	"io/ioutil"
	"net/http"
	"strings"
)

func LJtoKL(url string) string {
	rsp := httplib.Post("http://jd.zack.xin/api/jd/ulink.php")
	rsp.Param("url", url)
	rsp.Param("type", "kl")
	//rsp.Body(fmt.Sprintf(`url=%s&type=hy`, msg))
	data, err := rsp.Response()
	if err != nil {
		return "口令转换失败"
	}
	body, _ := ioutil.ReadAll(data.Body)
	logs.Info(string(body))
	if strings.Contains(string(body), "口令转换失败") {
		return "口令转换失败"
	} else {
		return string(body)
	}
}

func KLtoLJ(kl string) string {
	rsp := httplib.Post("http://jd.zack.xin/api/jd/ulink.php")
	rsp.Param("url", kl)
	rsp.Param("type", "hy")
	//rsp.Body(fmt.Sprintf(`url=%s&type=hy`, msg))
	data, err := rsp.Response()

	if err != nil {
		return "口令转换失败"
	}
	body, _ := ioutil.ReadAll(data.Body)
	if strings.Contains(string(body), "口令转换失败") {
		return "口令转换失败"
	} else {
		return string(body)
	}
}

func NolanKl(kl string) string {
	//rsp := httplib.Post("https://api.nolanstore.top/JComExchange")
	//rsp.Header("Content-Type", "application/json")
	//rsp.Header("Accept", "*/*")
	//rsp.Body("{\n  \"code\": \"15:/缝纫机教师😁，⇝𝒥𝓲𝓲𝓲𝓷𝓰◗凍(Q6BXW0mhbly)\"\n}")
	//body, _ := rsp.Bytes()
	//logs.Info(string(body))
	//val, _ := jsonparser.GetString(body, "data", "jumpUrl")
	//return val

	resp, err := http.Post("https://api.nolanstore.top/JComExchange", "application/json", strings.NewReader("{\n  \"code\": \"15:/缝纫机教师😁，⇝𝒥𝓲𝓲𝓲𝓷𝓰◗凍(Q6BXW0mhbly)\"\n}"))
	if err != nil {
		logs.Info("post请求失败 error: %+v", err)

	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logs.Info("读取Body失败 error: %+v", err)

	}
	logs.Info(string(body))
	return string(body)

}
