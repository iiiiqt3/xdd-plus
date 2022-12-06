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

func DtoC(url string) string {
	get := httplib.Get("https://u.jd.com/jda?e=99_1|1_10_1|||&p=JF8BARQJK1olXwUAXVhcCU0XA18IGV8TWg4FXG4ZVxNJXF9RXh5UHw0cSgYYXBcIX3BTTkRHA1ocFR0DXQ9FRnEIGV8TWg4FXEEETRdKDS1SX1cVXwIEU1ZaAFxUVy9MTxlQJVMOCxoAVXtUdjZwWFxlH2R1VxUhCFVBVA9RWh9DUQoyVW5eCUkXA28MEl0WbTYCUG4PZpOhtbetqkfB94nX3MltCXsXBGkJGFMRWAMEU19eOEsRM28BGl0SWQAKU19cZgonM18LK1MdMwZPVDBdCSUXTiJFK2sVbQUyVFZdCUoQA2sNB1sUXAYHSF5aDkoUC2sNHl8SVQ4yV19cCUsnM7GFqyBiImdLIR1UbQtvVDALYBDL0LY&a=fCg9UgoiAwwHO1BcXkQYFFljcXh2c1FcRl4zVRBSUll+AQAPDSwjLw==&refer=norefer&d=eMDOeNw")
	resp, _ := get.DoRequest()
	location, _ := resp.Location()
	return location.Path
}
