package models

import (
	"fmt"
	"github.com/beego/beego/v2/client/httplib"
	"github.com/beego/beego/v2/core/logs"
	"github.com/buger/jsonparser"
)

var NolanUrl string
var NolanToken string

func NolanGetJdQrImg(sender *Sender) {
	RabbitUrl = GetEnv("NolanUrl")
	RabbitToken = GetEnv("NolanToken")
	if RabbitUrl == "" || RabbitToken == "" {
		logs.Error("NolanUrl or NolanToken is empty")
		return
	}

	//http://192.168.195.53:5016/qr/GetQRKey
	get := httplib.Post(fmt.Sprintf("%s/qr/GetQRKey", RabbitUrl))
	get.Header("Content-Type", "application/json")
	get.Body(fmt.Sprintf("{\n  \"botApitoken\": \"%s\"\n}", NolanToken))
	bytes, _ := get.Bytes()
	logs.Info(string(bytes))

	code, _ := jsonparser.GetBoolean(bytes, "success")
	if code {
		key, _ := jsonparser.GetString(bytes, "data", "key")
		logs.Info(key)
		//decodeStr, _ := base64.StdEncoding.DecodeString(qr)
		//sender.SendImg(decodeStr)
		//sender.Reply("请使用京东APP扫码，150秒失效")
		//go getJDQrStatus(key, sender)
	} else {
		logs.Info(string(bytes))
		sender.Reply("获取扫码失败")
	}
}
