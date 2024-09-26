package models

import (
	"fmt"
	"github.com/beego/beego/v2/client/httplib"
	"github.com/beego/beego/v2/core/logs"
	"github.com/buger/jsonparser"
)

type FanLi struct {
	Appid    string
	Appkey   string
	Union_id string
}

var powerful = "https://api.jingpinku.com/get_powerful_link/api"

func Get_powerful_link(content string) string {
	//url := fmt.Sprintf("https://api.jingpinku.com/get_wire_report_link/api?appid=%s&appkey=%s&union_id=%s&content=%s", Config.FanLis.Appid, Config.FanLis.Appkey, Config.FanLis.Union_id, content)
	url := fmt.Sprintf("https://api.jingpinku.com/get_wire_report_link/api?appid=%s&appkey=%s&union_id=%s&content=%s", "2204290013243761", "NwG3i8LfdTyVM557kiLpVCTjMKtU06eA", "2024487473", content)
	logs.Informational(url)
	get := httplib.Get(url)
	s, _ := get.Bytes()
	val, _ := jsonparser.GetString(s, "official")
	return val
}
