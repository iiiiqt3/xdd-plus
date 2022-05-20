package main

import (
	"fmt"
	"github.com/beego/beego/v2/client/httplib"
	"github.com/beego/beego/v2/core/logs"
	"github.com/beego/beego/v2/server/web"
	"github.com/beego/beego/v2/server/web/context"
	"github.com/beego/beego/v2/server/web/filter/cors"
	"github.com/cdle/xdd/controllers"
	"github.com/cdle/xdd/models"
	"github.com/cdle/xdd/qbot"
	"io/ioutil"
	"os"
	"strings"
	"time"
)

var theme = ""

func main() {
	go func() {
		models.Save <- &models.JdCookie{}
	}()
	web.Get("/count", func(ctx *context.Context) {
		ctx.WriteString(models.Count())
	})
	web.Get("/", func(ctx *context.Context) {
		if models.Config.Theme == "" {
			models.Config.Theme = models.GhProxy + "https://ghproxy.com/https://raw.githubusercontent.com/764763903a/xdd-plus/main/theme/admin.html"
		}
		if theme != "" {
			ctx.WriteString(theme)
			return
		}
		if strings.Contains(models.Config.Theme, "http") {
			logs.Info("下载最新主题")
			s, _ := httplib.Get(models.Config.Theme).String()
			if s != "" {
				theme = s
				ctx.WriteString(s)
				return
			}
			logs.Warn("主题下载失败，使用默认主题")
		}
		f, err := os.Open(models.Config.Theme)
		if err == nil {
			d, _ := ioutil.ReadAll(f)
			theme = string(d)
			ctx.WriteString(string(d))
			return
		}
	})

	web.Router("/api/login/qrcode", &controllers.LoginController{}, "get:GetQrcode")
	web.Router("/api/login/qrcode.png", &controllers.LoginController{}, "get:GetQrcode")
	web.Router("/api/login/qrcode1", &controllers.LoginController{}, "get:GetQrcode1")
	web.Router("/api/login/query", &controllers.LoginController{}, "get:Query")
	web.Router("/api/login/cookie", &controllers.LoginController{}, "get:Cookie")
	web.Router("/api/login/admin", &controllers.LoginController{}, "post:IsAdmin")
	web.Router("/api/login/cklogin", &controllers.LoginController{}, "post:CkLogin")
	web.Router("/api/login/smslogin", &controllers.LoginController{}, "post:SMSLogin")
	web.Router("/api/getUserInfo", &controllers.LoginController{}, "post:GetUserInfo")
	web.Router("/api/getUserInfo", &controllers.LoginController{}, "get:GetUserInfo")
	web.Router("/api/getUserPin", &controllers.LoginController{}, "post:GetUserPin")
	web.Router("/api/getUserPin", &controllers.LoginController{}, "get:GetUserPin")
	web.Router("/api/log", &controllers.LoginController{}, "post:GetLogs")
	web.Router("/api/ua", &controllers.LoginController{}, "get:GetUS")
	web.Router("/api/akl", &controllers.LoginController{}, "get:GetLog199")
	web.Router("/api/account", &controllers.AccountController{}, "get:List")
	web.Router("/api/account", &controllers.AccountController{}, "post:CreateOrUpdate")
	web.Router("/admin", &controllers.AccountController{}, "get:Admin")
	web.Router("/admin", &controllers.AccountController{}, "post:Admin")
	web.Router("/userCenter", &controllers.AccountController{}, "get:UserCenter")
	web.Router("/userCenter", &controllers.AccountController{}, "post:UserCenter")

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
	go func() {
		time.Sleep(time.Second * 4)
		(&models.JdCookie{}).Push(fmt.Sprintf("小滴滴已启动，版本号:%s", models.Config.Version))

	}()
	if models.Config.QQID != 0 || models.Config.QQGroupID != 0 {
		go qbot.Main()
	}
	web.Run()

}
