package models

import (
	"fmt"
	"github.com/beego/beego/v2/client/httplib"
	"github.com/beego/beego/v2/core/logs"
	"github.com/buger/jsonparser"
)

var powerful = "https://api.jingpinku.com/get_powerful_link/api"

func Get_powerful_link(content string) string {
	url := fmt.Sprintf("https://api.jingpinku.com/get_powerful_link/api?appid=%s&appkey=%s&union_id=%s&content=%s", Config.FanLis.Appid, Config.FanLis.Appkey, Config.FanLis.Union_id, content)
	logs.Informational(url)
	get := httplib.Get(url)
	s, _ := get.Bytes()
	val, _ := jsonparser.GetString(s, "official")
	return val
}
