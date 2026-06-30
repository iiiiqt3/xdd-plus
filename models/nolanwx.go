package models

import (
	"encoding/json"
	"fmt"
	"github.com/beego/beego/v2/client/httplib"
	"time"
)

var NolanWxUrl = "http://192.168.195.41:9191"
var TempToken string
var ClientId string

func getChatToken() {
	for {
		Info("获取session token...")
		password := "NolanWeChatApi"
		get := httplib.Get(NolanWxUrl + "/api/Token/Create?password=" + password)
		TempToken, _ = get.String()
		Info("获取Token成功，休眠30分钟")
		time.Sleep(time.Duration(30) * time.Minute)
		Info("更新Token")
	}
}

func CreateClient() {
	env := GetEnv("ClientId")
	if env != "" {
		//todo使用实例
		Info("使用实列Guid:" + env)
	} else {
		type WxClient struct {
			Flag int `json:"flag"`
			Data struct {
				Guid string `json:"Guid"`
			} `json:"data"`
			Code    int         `json:"code"`
			Message string      `json:"message"`
			Url     interface{} `json:"url"`
			Remark  string      `json:"remark"`
		}
		var client WxClient
		post := httplib.Post(NolanWxUrl + "/api/Client/WXCreate")
		post.Header("Authorization", TempToken)
		post.Header("content-type", "application/json")
		Info("默认使用Ipad协议")
		post.Body("{\n  \"Terminal\": 2\n}").ToJSON(client)
		Info("获取到Guid:" + client.Data.Guid)
		ExportEnv(&Env{
			Name:  "ClientId",
			Value: client.Data.Guid,
		})
		//todo创建实例
	}
}

type NolanResult struct {
	Flag    int         `json:"flag"`
	Data    interface{} `json:"data"`
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Url     interface{} `json:"url"`
	Remark  interface{} `json:"remark"`
}

func AutoLogin() {
	var result NolanResult
	env := GetEnv("ClientId")
	Info("开始使用自动登录")
	post := httplib.Post(NolanWxUrl + "/api/Login/WXSecLoginAuto")
	post.Header("Authorization", TempToken)
	post.Header("content-type", "application/json")
	post.Body(fmt.Sprintf("{\n  \"Guid\": \"%s\"\n}", env)).ToJSON(result)
	if result.Code == -1 {
		Info("自动登录失败")
		WxLogin()
		//todo 执行人工登录
	} else {
		Info("自动登录成功")
		//todo 执行心跳
	}
}

func WxLogin() {
	env := GetEnv("ClientId")
	type WxUser struct {
		Guid     string `json:"Guid"`
		Channel  int    `json:"Channel"`
		UserName string `json:"UserName"`
		Password string `json:"Password"`
		Slider   bool   `json:"Slider"`
		Init     bool   `json:"Init"`
	}
	var result NolanResult

	wxuser := WxUser{
		Guid:     env,
		Channel:  0,
		UserName: GetEnv("UserName"),
		Password: GetEnv("Password"),
		Slider:   false,
		Init:     true,
	}
	Info("开始使用微信登录")
	post := httplib.Post(NolanWxUrl + "/api/Login/WXSecLoginManual")
	post.Header("Authorization", TempToken)
	post.Header("content-type", "application/json")
	marshal, _ := json.Marshal(wxuser)
	post.Body(marshal).ToJSON(result)
	if result.Code == -1 {
		Info("微信登录失败")
		//todo 执行人工登录
	} else {
		Info("微信登录成功")
		//todo 执行心跳
	}
}
