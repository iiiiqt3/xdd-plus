package models

import (
	"github.com/beego/beego/v2/client/httplib"
	"github.com/beego/beego/v2/core/logs"
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

