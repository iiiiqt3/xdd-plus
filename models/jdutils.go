package models

import (
	"compress/gzip"
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
	rsp := httplib.Post("http://api.nolanstore.top/JComExchange")
	rsp.Header("Content-Type", "application/json")
	rsp.Param("body", "{\r\n  \"code\": \"口令海量低价好物，新人享1分购噢！  http:/JEFc2BMGPfeew1副制这段话￥G9Q4kDK5Vh%⇝【⤴ιng▴栋特价】\"\r\n}")
	rsp.Body("{\r\n  \"code\": \"口令海量低价好物，新人享1分购噢！  http:/JEFc2BMGPfeew1副制这段话￥G9Q4kDK5Vh%⇝【⤴ιng▴栋特价】\"\r\n}")
	//rsp.Param("type", "hy")

	data, _ := rsp.Response()
	reader, _ := gzip.NewReader(data.Body)
	body, _ := ioutil.ReadAll(reader)

	//body, _ := rsp.Bytes()
	logs.Info(string(body))
	val, _ := jsonparser.GetString(body, "data", "jumpUrl")
	return val
}
