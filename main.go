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
	"github.com/cdle/xdd/vweb"
	"github.com/eatmoreapple/openwechat"
	"time"
)

var query = ""
var friedns openwechat.Friends

// Result 通用API响应结构体
type Result struct {
	Code    int         `json:"code"`
	Data    interface{} `json:"data"`
	Message string      `json:"message"`
}

// AuthResult 认证结果响应结构体
type AuthResult struct {
	Flag    bool        `json:"flag"`
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

// main 程序入口函数，初始化日志、路由、定时任务并启动Web服务
func main() {

	// 设置日志输出到文件
	logs.SetLogger(logs.AdapterFile, "{\"filename\":\"logs/xdd.log\", \"level\":6}")

	// 启动定时保存任务
	go func() {
		models.Save <- &models.JdCookie{}
	}()

	// 微信消息推送API，通过token验证权限
	token := models.GetEnv("wx_msg_token")
	web.Post("/api/send_wx_msg", func(ctx *context.Context) {
		type RequestData struct {
			WxID string `json:"wxid"`
			Msg  string `json:"msg"`
		}
		requestToken := ctx.Input.Query("token") // 假设token是通过 query 参数传递的

		// 如果 token 不匹配，则返回 403 错误
		if requestToken != token {
			ctx.Output.SetStatus(403)
			ctx.Output.Body([]byte("禁止访问：token无效"))
			return
		}

		var data RequestData
		if err := json.Unmarshal(ctx.Input.RequestBody, &data); err != nil {
			ctx.Output.SetStatus(400)
			ctx.Output.Body([]byte("Invalid request payload"))
			return
		}

		if data.WxID == "" || data.Msg == "" {
			ctx.Output.SetStatus(400)
			ctx.Output.Body([]byte("wxid and msg are required"))
			return
		}

		models.SendWxMsg(data.WxID, data.Msg)

		ctx.Output.Body([]byte("Message sent successfully"))
	})
	// 手机号权限验证接口
	web.Get("/permisson", func(ctx *context.Context) {
		tel := ctx.Input.Query("phone")
		logs.Info(tel)
		auth := models.GetAuth(tel)
		if auth {
			result := AuthResult{
				Flag:    true,
				Data:    "null",
				Code:    20000,
				Message: "操作成功",
			}
			jsons, errs := json.Marshal(result) //转换成JSON返回的是byte[]
			if errs != nil {
				fmt.Println(errs.Error())
			}
			ctx.WriteString(string(jsons))
		} else {
			result := AuthResult{
				Flag:    false,
				Data:    "null",
				Code:    51000,
				Message: "操作失败",
			}
			jsons, errs := json.Marshal(result) //转换成JSON返回的是byte[]
			if errs != nil {
				fmt.Println(errs.Error())
			}
			ctx.WriteString(string(jsons))
		}
	})

	// 在线人数统计接口
	web.Get("/count", func(ctx *context.Context) {
		ctx.WriteString(models.Count())
	})

	// 公告查询接口
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

	// VIP用户查询页面，从远程下载最新版本
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

	// 默认首页，返回门户登录页面
	web.Get("/", func(ctx *context.Context) {
		file, err := vweb.WebFs.ReadFile("html/portal_login.html")
		if err != nil {
			ctx.WriteString("portal login page not found")
			return
		}
		ctx.WriteString(string(file))
	})

	// ===================== 登录相关路由 =====================
	web.Router("/api/login/qrcode", &controllers.LoginController{}, "get:GetQrcode")
	web.Router("/api/login/qrcode.png", &controllers.LoginController{}, "get:GetQrcode")
	web.Router("/api/login/qrcode1", &controllers.LoginController{}, "get:GetQrcode1")
	web.Router("/api/login/query", &controllers.LoginController{}, "get:Query")
	web.Router("/api/login/admin", &controllers.LoginController{}, "post:IsAdmin")
	web.Router("/api/login/register", &controllers.LoginController{}, "post:RegisterUser")
	web.Router("/api/login/reset/info", &controllers.LoginController{}, "post:GetResetPasswordInfo")
	web.Router("/api/login/reset/password", &controllers.LoginController{}, "post:ResetPassword")
	web.Router("/api/login/logout", &controllers.LoginController{}, "post:PortalLogout")
	web.Router("/api/login/status", &controllers.LoginController{}, "get:GetLoginStatus")
	web.Router("/api/login/cklogin", &controllers.LoginController{}, "post:CkLogin")
	web.Router("/api/login/smslogin", &controllers.LoginController{}, "post:SMSLogin")
	web.Router("/api/login/batch-smslogin", &controllers.LoginController{}, "post:BatchSMSLogin")
	web.Router("/api/getUserInfo", &controllers.LoginController{}, "post:GetUserInfo")
	web.Router("/api/getUserInfo", &controllers.LoginController{}, "get:GetUserInfo")
	web.Router("/api/getUserPin", &controllers.LoginController{}, "post:GetUserPin")
	web.Router("/api/getUserPin", &controllers.LoginController{}, "get:GetUserPin")
	web.Router("/api/account", &controllers.AccountController{}, "get:List")
	web.Router("/api/account", &controllers.AccountController{}, "post:CreateOrUpdate")
	// ===================== 门户页面路由 =====================
	web.Router("/portal/login", &controllers.LoginController{}, "get:PortalLoginPage")
	web.Router("/portal", &controllers.PortalController{}, "get:Index")
	web.Router("/api/portal/dashboard", &controllers.PortalController{}, "get:Dashboard")
	web.Router("/api/portal/profile", &controllers.PortalController{}, "get:Profile")
	web.Router("/api/portal/activities", &controllers.PortalController{}, "get:Activities")
	web.Router("/api/portal/projects", &controllers.PortalController{}, "get:Projects")
	web.Router("/api/portal/project", &controllers.PortalController{}, "post:CreateProject")
	web.Router("/api/portal/project/renew", &controllers.PortalController{}, "post:RenewProject")
	web.Router("/api/portal/project/update", &controllers.PortalController{}, "post:UpdateProject")
	web.Router("/api/portal/project/delete", &controllers.PortalController{}, "post:DeleteProject")
	web.Router("/api/portal/project/income", &controllers.PortalController{}, "post:QueryIncome")
	web.Router("/api/portal/checkin", &controllers.PortalController{}, "post:CheckIn")
	web.Router("/api/portal/pray", &controllers.PortalController{}, "post:Pray")
	web.Router("/api/portal/sign-params", &controllers.PortalController{}, "get:SignParams")
	web.Router("/api/portal/redeem-key", &controllers.PortalController{}, "post:RedeemKey")
	web.Router("/api/portal/wx/status", &controllers.PortalController{}, "get:WxStatus")
	web.Router("/api/portal/wx/scan-login", &controllers.PortalController{}, "post:WxScanLogin")
	web.Router("/api/portal/wx/relogin", &controllers.PortalController{}, "post:WxRelogin")
	web.Router("/api/portal/wx/wake-login", &controllers.PortalController{}, "post:WxWakeLogin")
	web.Router("/api/portal/wx/logout", &controllers.PortalController{}, "post:WxLogout")
	web.Router("/api/portal/wx/delete", &controllers.PortalController{}, "post:WxDelete")
	web.Router("/api/portal/wx/poll-login", &controllers.PortalController{}, "post:WxPollLogin")
	web.Router("/api/portal/wx/devices", &controllers.PortalController{}, "get:WxDevices")
	web.Router("/api/portal/wx/add-device", &controllers.PortalController{}, "post:WxAddDevice")
	web.Router("/api/portal/wx/remove-device", &controllers.PortalController{}, "post:WxRemoveDevice")
	web.Router("/api/portal/notifications", &controllers.PortalController{}, "get:Notifications")
	web.Router("/api/portal/notification", &controllers.PortalController{}, "get:NotificationDetail")
	web.Router("/api/portal/feedback", &controllers.PortalController{}, "post:SubmitFeedback")
	web.Router("/api/portal/coin-logs", &controllers.PortalController{}, "get:CoinLogs")
	// 管理员登录页面
	web.Get("/admin/login", func(ctx *context.Context) {
		file, err := vweb.WebFs.ReadFile("html/admin_login.html")
		if err != nil {
			ctx.WriteString("admin login page not found")
			return
		}
		ctx.WriteString(string(file))
	})

	web.Router("/admin", &controllers.AccountController{}, "get:Admin")
	web.Router("/admin", &controllers.AccountController{}, "post:Admin")

	// ===================== 后台管理 API =====================
	web.Router("/api/admin/activities", &controllers.AdminApiController{}, "get:GetActivities")
	web.Router("/api/admin/activities/save", &controllers.AdminApiController{}, "post:SaveActivities")
	web.Router("/api/admin/qlconfigs", &controllers.AdminApiController{}, "get:GetQingLongConfigs")
	web.Router("/api/admin/qlconfigs/test", &controllers.AdminApiController{}, "post:TestQLConnection")
	web.Router("/api/admin/jdcontainers", &controllers.AdminApiController{}, "get:GetJdContainers")
	web.Router("/api/admin/jdconfig", &controllers.AdminApiController{}, "get:GetJdConfig")
	web.Router("/api/admin/jdconfig/save", &controllers.AdminApiController{}, "post:SaveJdConfig")
	web.Router("/api/admin/gameconfig", &controllers.AdminApiController{}, "get:GetGameConfig")
	web.Router("/api/admin/gameconfig/save", &controllers.AdminApiController{}, "post:SaveGameConfig")
	web.Router("/api/admin/activity/pushgroups", &controllers.AdminApiController{}, "post:SendActivityToGroups")
	web.Router("/api/admin/container/test", &controllers.AdminApiController{}, "post:TestContainer")
	web.Router("/api/admin/notifications", &controllers.AdminApiController{}, "get:GetNotifications")
	web.Router("/api/admin/notifications/send", &controllers.AdminApiController{}, "post:SendNotification")
	web.Router("/api/admin/notifications/batch/delete", &controllers.AdminApiController{}, "post:BatchDeleteNotifications")
	web.Router("/api/admin/notifications/:id", &controllers.AdminApiController{}, "delete:DeleteNotification;put:UpdateNotification")
	web.Router("/api/admin/feedbacks", &controllers.AdminApiController{}, "get:GetFeedbacks")
	web.Router("/api/admin/feedbacks/:id", &controllers.AdminApiController{}, "get:GetFeedbackDetail;put:UpdateFeedbackStatus")
	web.Router("/api/admin/feedbacks/batch/delete", &controllers.AdminApiController{}, "post:BatchDeleteFeedbacks")
	web.Router("/api/admin/feedbacks/batch/reply", &controllers.AdminApiController{}, "post:BatchReplyFeedbacks")
	web.Router("/api/admin/feedbacks/batch/reward", &controllers.AdminApiController{}, "post:BatchRewardFeedbacks")
	web.Router("/api/admin/notifications/categories", &controllers.AdminApiController{}, "get:GetNotificationCategories")
	web.Router("/api/admin/envvars", &controllers.AdminApiController{}, "get:GetEnvVars")
	web.Router("/api/admin/envvars/create", &controllers.AdminApiController{}, "post:CreateEnvVar")
	web.Router("/api/admin/envvars/update", &controllers.AdminApiController{}, "post:UpdateEnvVar")
	web.Router("/api/admin/envvars/:id", &controllers.AdminApiController{}, "delete:DeleteEnvVar")
	web.Router("/api/admin/users", &controllers.AdminApiController{}, "get:GetUsers")
	web.Router("/api/admin/users/coin", &controllers.AdminApiController{}, "post:UpdateUserCoin")
	web.Router("/api/admin/users/coin-logs", &controllers.AdminApiController{}, "get:GetCoinLogs")
	web.Router("/api/admin/sysconfig", &controllers.AdminApiController{}, "get:GetSystemConfig")
	web.Router("/api/admin/sysconfig/save", &controllers.AdminApiController{}, "post:SaveSystemConfig")
	web.Router("/api/admin/sysconfig/image-token", &controllers.AdminApiController{}, "post:GetImageToken")
	web.Router("/api/admin/jdcookies", &controllers.AdminApiController{}, "get:GetJdCookies")
	web.Router("/api/admin/jdcookies/delete", &controllers.AdminApiController{}, "post:DeleteJdCookie")
	web.Router("/api/admin/jdcookies/update", &controllers.AdminApiController{}, "post:UpdateJdCookie")
	web.Router("/api/admin/qlenvs", &controllers.AdminApiController{}, "get:GetQLEnvs")
	web.Router("/api/admin/qlenvs/delete", &controllers.AdminApiController{}, "post:DeleteQLEnv")
	web.Router("/api/admin/qlenvs/enable", &controllers.AdminApiController{}, "post:EnableQLEnv")
	web.Router("/api/admin/qlenvs/disable", &controllers.AdminApiController{}, "post:DisableQLEnv")
	web.Router("/api/admin/qlenvs/update", &controllers.AdminApiController{}, "post:UpdateQLEnv")
	web.Router("/api/admin/users/delete", &controllers.AdminApiController{}, "post:DeleteUser")
	web.Router("/api/admin/jdcontainers/add", &controllers.AdminApiController{}, "post:AddJdContainer")
	web.Router("/api/admin/jdcontainers/update", &controllers.AdminApiController{}, "post:UpdateJdContainer")
	web.Router("/api/admin/jdcontainers/delete", &controllers.AdminApiController{}, "post:DeleteJdContainer")
	web.Router("/api/admin/qlconfigs/add", &controllers.AdminApiController{}, "post:AddQLConfig")
	web.Router("/api/admin/qlconfigs/update", &controllers.AdminApiController{}, "post:UpdateQLConfig")
	web.Router("/api/admin/qlconfigs/delete", &controllers.AdminApiController{}, "post:DeleteQLConfig")
	web.Router("/api/admin/users/create", &controllers.AdminApiController{}, "post:CreateUser")
	web.Router("/api/admin/users/update", &controllers.AdminApiController{}, "post:UpdateUser")
	web.Router("/api/admin/users/batch/delete", &controllers.AdminApiController{}, "post:BatchDeleteUsers")
	web.Router("/api/admin/users/batch/update", &controllers.AdminApiController{}, "post:BatchUpdateUserCoins")
	web.Router("/api/admin/jdcookies/batch/delete", &controllers.AdminApiController{}, "post:BatchDeleteJdCookies")
	web.Router("/api/admin/jdcookies/batch/update", &controllers.AdminApiController{}, "post:BatchUpdateJdCookies")
	web.Router("/api/admin/envvars/batch/delete", &controllers.AdminApiController{}, "post:BatchDeleteEnvVars")
	web.Router("/api/admin/qlenvs/create", &controllers.AdminApiController{}, "post:CreateQLEnv")
	web.Router("/api/admin/qlenvs/batch/delete", &controllers.AdminApiController{}, "post:BatchDeleteQLEnvs")
	web.Router("/api/admin/qlenvs/batch/update", &controllers.AdminApiController{}, "post:BatchUpdateQLEnvs")

	// ===================== 活动人数统计 API =====================
	web.Router("/api/admin/activity-stats", &controllers.AdminApiController{}, "get:GetActivityStats")
	web.Router("/api/admin/activity-stats/refresh", &controllers.AdminApiController{}, "post:RefreshActivityStats")
	web.Router("/api/admin/activity-stats/:id/refresh", &controllers.AdminApiController{}, "post:RefreshActivityStats")
	web.Router("/api/admin/notify-delete-expired", &controllers.AdminApiController{}, "post:TriggerNotifyDeleteExpiredCKs")

	// ===================== 青龙活动授权管理 API =====================
	web.Router("/api/admin/activity-auth-list", &controllers.AdminApiController{}, "get:GetActivityAuthList")
	web.Router("/api/admin/activity-auth-accounts", &controllers.AdminApiController{}, "get:GetActivityAuthAccounts")
	web.Router("/api/admin/activity-auth-delete-account", &controllers.AdminApiController{}, "post:DeleteActivityAuthAccount")
	web.Router("/api/admin/activity-auth-batch", &controllers.AdminApiController{}, "post:BatchUpdateActivityAuth")
	web.Router("/api/admin/activity-auth-convert-monthly", &controllers.AdminApiController{}, "post:ConvertActivityToMonthly")

	// ===================== 微信协议设备管理 API =====================
	web.Router("/api/admin/wx-devices", &controllers.AdminApiController{}, "get:GetWxDevices")
	web.Router("/api/admin/wx-devices-by-url", &controllers.AdminApiController{}, "get:GetWxDevicesByURL")
	web.Router("/api/admin/wx-device-stats", &controllers.AdminApiController{}, "get:GetWxDeviceStats")
	web.Router("/api/admin/user-online-stats", &controllers.AdminApiController{}, "get:GetUserOnlineStats")
	web.Router("/api/admin/wx-notify-offline", &controllers.AdminApiController{}, "post:NotifyWxOffline")

	// ===================== 微信协议配置管理 API =====================
	web.Router("/api/admin/wx-protocol-config", &controllers.AdminApiController{}, "get:GetWxProtocolConfig")
	web.Router("/api/admin/wx-protocol-config/save", &controllers.AdminApiController{}, "post:SaveWxProtocolConfig")

	// ===================== 青龙 Cron 任务管理 API =====================
	web.Router("/api/admin/crontasks", &controllers.AdminApiController{}, "get:GetCronTasks")
	web.Router("/api/admin/crontasks/logs", &controllers.AdminApiController{}, "get:GetCronTaskLogs")
	web.Router("/api/admin/crontasks/logcontent", &controllers.AdminApiController{}, "get:GetCronTaskLogContent")
	web.Router("/api/admin/crontasks/script", &controllers.AdminApiController{}, "get:GetCronTaskScript")
	web.Router("/api/admin/crontasks/script", &controllers.AdminApiController{}, "post:UpdateCronTaskScript")
	web.Router("/api/admin/crontasks/scriptfile", &controllers.AdminApiController{}, "get:GetCronTaskScriptFile")
	web.Router("/api/admin/crontasks/scriptfile", &controllers.AdminApiController{}, "post:UpdateCronTaskScriptFile")
	web.Router("/api/admin/crontasks/schedule", &controllers.AdminApiController{}, "post:UpdateCronTaskSchedule")
	web.Router("/api/admin/crontasks/enable", &controllers.AdminApiController{}, "post:EnableCronTask")
	web.Router("/api/admin/crontasks/disable", &controllers.AdminApiController{}, "post:DisableCronTask")
	web.Router("/api/admin/crontasks/delete", &controllers.AdminApiController{}, "post:DeleteCronTask")
	web.Router("/api/admin/crontasks/run", &controllers.AdminApiController{}, "post:RunCronTask")
	web.Router("/api/admin/crontasks/stop", &controllers.AdminApiController{}, "post:StopCronTask")
	web.Router("/api/admin/jdcontainers/names", &controllers.AdminApiController{}, "get:GetJdContainerNames")

	// ===================== 京东容器 Cron 任务管理 API =====================
	web.Router("/api/admin/jdcrontasks", &controllers.AdminApiController{}, "get:GetJdCronTasks")
	web.Router("/api/admin/jdcrontasks/logs", &controllers.AdminApiController{}, "get:GetJdCronTaskLogs")
	web.Router("/api/admin/jdcrontasks/logcontent", &controllers.AdminApiController{}, "get:GetJdCronTaskLogContent")
	web.Router("/api/admin/jdcrontasks/script", &controllers.AdminApiController{}, "get:GetJdCronTaskScript")
	web.Router("/api/admin/jdcrontasks/script", &controllers.AdminApiController{}, "post:UpdateJdCronTaskScript")
	web.Router("/api/admin/jdcrontasks/scriptfile", &controllers.AdminApiController{}, "get:GetJdCronTaskScriptFile")
	web.Router("/api/admin/jdcrontasks/scriptfile", &controllers.AdminApiController{}, "post:UpdateJdCronTaskScriptFile")
	web.Router("/api/admin/jdcrontasks/schedule", &controllers.AdminApiController{}, "post:UpdateJdCronTaskSchedule")
	web.Router("/api/admin/jdcrontasks/enable", &controllers.AdminApiController{}, "post:EnableJdCronTask")
	web.Router("/api/admin/jdcrontasks/disable", &controllers.AdminApiController{}, "post:DisableJdCronTask")
	web.Router("/api/admin/jdcrontasks/delete", &controllers.AdminApiController{}, "post:DeleteJdCronTask")
	web.Router("/api/admin/jdcrontasks/run", &controllers.AdminApiController{}, "post:RunJdCronTask")
	web.Router("/api/admin/jdcrontasks/stop", &controllers.AdminApiController{}, "post:StopJdCronTask")
	// VIP用户额外路由：微信消息接收、QQ机器人、环境变量管理、配置管理
	if models.Config.VIP {
		web.Router("/wx/receive", &controllers.WxController{}, "post:HandleWxMessage")
		web.Router("/api/login/wskeylogin", &controllers.LoginController{}, "post:WskeyLogin")
		web.Router("/qq", &controllers.QQController{}, "get,post:Echo")
		web.Router("/api/envs", &controllers.AccountController{}, "get:ListEnvs")
		web.Router("/api/envs", &controllers.AccountController{}, "post:CreateOrUpdateEnv")
		web.Router("/api/config", &controllers.ConfigController{}, "get:ListConfig")
		web.Router("/api/config", &controllers.ConfigController{}, "post:CreateOrUpdateConfig")

		}

	// 设置静态文件目录
	if models.Config.Static == "" {
		models.Config.Static = "./static"
	}
	web.BConfig.WebConfig.StaticDir["/static"] = models.Config.Static

	// 配置Web服务参数
	web.BConfig.AppName = models.AppName
	web.BConfig.WebConfig.AutoRender = false
	web.BConfig.CopyRequestBody = true
	web.BConfig.WebConfig.Session.SessionOn = true
	web.BConfig.WebConfig.Session.SessionGCMaxLifetime = 172800
	web.BConfig.WebConfig.Session.SessionCookieLifeTime = 172800
	web.BConfig.WebConfig.Session.SessionName = models.AppName
	// 配置CORS跨域访问
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

	// 启动后延迟发送启动通知
	go func() {
		time.Sleep(time.Second * 4)
		(&models.JdCookie{}).Push(fmt.Sprintf("小滴滴已启动，版本号:%s", models.Config.Version))

	}()

	// 初始化微信机器人（桌面模式）
	bot := openwechat.DefaultBot(openwechat.Desktop) // 桌面模式

	// 注册微信消息处理函数
	bot.MessageHandler = func(msg *openwechat.Message) {
		if msg.IsText() && msg.Content == "ping" {

			msg.ReplyText("pong")

			name := msg.FromUserName
			id := friedns.GetByUsername(name).User
			id.Detail()

		}
	}
	// 启动Web服务，阻塞主线程
	web.Run()

}
