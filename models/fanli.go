package models

import (
	"fmt"
	"github.com/beego/beego/v2/client/httplib"
	"github.com/buger/jsonparser"
)

var powerful = "https://api.jingpinku.com/get_powerful_link/api"

func Get_powerful_link(content string, union_id string) string {
	url := fmt.Sprintf("https://api.jingpinku.com/get_powerful_link/api?appid=%s&appkey=%s&union_id=%s&content=%s", Config.FanLis.appid, Config.FanLis.appkey, Config.FanLis.union_id, content)
	get := httplib.Get(url)
	s, _ := get.Bytes()
	val, _ := jsonparser.GetString(s, "official")
	return val

}
