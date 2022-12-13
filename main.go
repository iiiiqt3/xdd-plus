package main

import (
	"encoding/json"
	"fmt"
	"github.com/beego/beego/v2/client/httplib"
	"github.com/beego/beego/v2/core/logs"
	"github.com/beego/beego/v2/server/web"
	"github.com/beego/beego/v2/server/web/context"
	"github.com/beego/beego/v2/server/web/filter/cors"
	"github.com/cdle/xdd/controllers"
	"github.com/cdle/xdd/models"
	"github.com/cdle/xdd/qbot"
	"time"
)

var theme = ""

var query = ""

type Result struct {
	Code    int         `json:"code"`
	Data    interface{} `json:"data"`
	Message string      `json:"message"`
}

func main() {
	go func() {
		models.Save <- &models.JdCookie{}
	}()

	web.Get("/count", func(ctx *context.Context) {
		ctx.WriteString(models.Count())
	})

	web.Get("/announcement", func(ctx *context.Context) {
		result := Result{
			Data:    models.Config.Title,
			Code:    0,
			Message: "查询成功",
		}
		jsons, errs := json.Marshal(result) //转换成JSON返回的是byte[]
		if errs != nil {
			fmt.Println(errs.Error())
		}
		ctx.WriteString(string(jsons))
	})

	if models.Config.VIP {
		web.Get("/query", func(ctx *context.Context) {
			if query != "" {
				ctx.WriteString(query)
				return
			}
			logs.Info("下载最新网页查询版本")
			s, _ := httplib.Get("https://git.smxy.xyz/jia_yuan/xdd-html/raw/branch/xdd/version/index.html").String()
			if s != "" {
				query = s
				ctx.WriteString(s)
				return
			}
			logs.Warn("主题下载失败，请注意查看网络环境")
		})
	}

	//web.Get("/", func(ctx *context.Context) {
	//	//if models.Config.Theme == "" {
	//	//	models.Config.Theme = "http://xdd.smxy.xyz/admin.html"
	//	//}
	//	models.Config.Theme = "http://xdd.smxy.xyz/admin.html"
	//	if theme != "" {
	//		ctx.WriteString(theme)
	//		return
	//	}
	//	if strings.Contains(models.Config.Theme, "http") {
	//		logs.Info("下载最新主题")
	//		s, _ := httplib.Get(models.Config.Theme).String()
	//		if s != "" {
	//			theme = s
	//			ctx.WriteString(s)
	//			return
	//		}
	//		logs.Warn("主题下载失败，使用默认主题")
	//	}
	//	f, err := os.Open(models.Config.Theme)
	//	if err == nil {
	//		d, _ := ioutil.ReadAll(f)
	//		theme = string(d)
	//		ctx.WriteString(string(d))
	//		return
	//	}
	//})

	web.Router("/api/login/admin", &controllers.LoginController{}, "post:IsAdmin")
	web.Router("/api/login/cklogin", &controllers.LoginController{}, "post:CkLogin")
	web.Router("/api/login/smslogin", &controllers.LoginController{}, "post:SMSLogin")
	web.Router("/api/login/wskeylogin", &controllers.LoginController{}, "post:WskeyLogin")
	web.Router("/api/getUserInfo", &controllers.UserController{}, "post:GetUserInfo")
	web.Router("/api/getUserInfo", &controllers.UserController{}, "get:GetUserInfo")
	web.Router("/api/getUserPin", &controllers.UserController{}, "post:GetUserPin")
	web.Router("/api/getUserPin", &controllers.UserController{}, "get:GetUserPin")
	web.Router("/api/account", &controllers.AccountController{}, "get:List")
	web.Router("/api/account", &controllers.AccountController{}, "post:CreateOrUpdate")
	web.Router("/api/envs", &controllers.AccountController{}, "get:ListEnvs")
	web.Router("/api/envs", &controllers.AccountController{}, "post:CreateOrUpdateEnv")
	web.Router("/admin", &controllers.AccountController{}, "get:Admin")
	web.Router("/wx/receive", &controllers.WxController{}, "get,post:HandleMessage")

	if models.Config.Static == "" {
		models.Config.Static = "./static"
	}
	web.BConfig.WebConfig.StaticDir["/static"] = models.Config.Static
	web.BConfig.AppName = models.AppName
	web.BConfig.WebConfig.AutoRender = false
	web.BConfig.CopyRequestBody = true
	web.BConfig.WebConfig.Session.SessionOn = true
	web.BConfig.WebConfig.Session.SessionGCMaxLifetime = 3600
	web.BConfig.WebConfig.Session.SessionName = models.AppName
	web.InsertFilter("*", web.BeforeRouter, cors.Allow(&cors.Options{
		//允许访问所有源
		AllowAllOrigins: true,
		//可选参数"GET", "POST", "PUT", "DELETE", "OPTIONS" (*为所有)
		//其中Options跨域复杂请求预检
		AllowMethods: []string{"*"},
		//指的是允许的Header的种类
		AllowHeaders: []string{"*"},
		//公开的HTTP标头列表
		ExposeHeaders: []string{"Content-Length"},
		//如果设置，则允许共享身份验证凭据，例如cookie
		AllowCredentials: true,
	}))

	if models.Config.QQID != 0 || models.Config.QQGroupID != 0 {
		go qbot.Main()
	}
	go func() {
		time.Sleep(time.Second * 4)
		(&models.JdCookie{}).Push(fmt.Sprintf("小滴滴已启动，版本号:%s", models.Config.Version))

	}()

	web.Run()
}
