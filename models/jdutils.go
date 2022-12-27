package models

import (
	"github.com/beego/beego/v2/client/httplib"
	"github.com/beego/beego/v2/core/logs"
	"github.com/buger/jsonparser"
	"io/ioutil"
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
	rsp := httplib.Post("https://api.nolanstore.top/JComExchange")
	rsp.Param("code", kl)
	//rsp.Param("type", "hy")

	data, _ := rsp.Response()
	body, _ := ioutil.ReadAll(data.Body)
	val, _ := jsonparser.GetString(body, "data", "jumpUrl")
	return val
}
