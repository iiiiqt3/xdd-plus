package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/beego/beego/v2/server/web"
	"github.com/beego/beego/v2/server/web/context"
	"github.com/beego/beego/v2/server/web/filter/cors"
	"github.com/cdle/xdd/controllers"
	"github.com/cdle/xdd/models"
	"github.com/cdle/xdd/vweb"
	"github.com/cdle/xdd/yybportal"
)

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

	models.System().Infof("XDD 服务启动")

	// 应用宝模块后台初始化，避免阻塞 HTTP / 机器人 WS（失败不影响主服务）
	go func() {
		models.Yyb().Infof("应用宝模块后台初始化中...")
		if err := yybportal.Init(yybportal.ModuleConfigFromModels()); err != nil {
			models.Yyb().Errorf("应用宝启动失败(主服务不受影响): %v", err)
			return
		}
		if models.Config.Yyb.Enabled {
			models.SetYybJdLoginHandler(yybportal.HandleBotJdLogin)
		}
	}()

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
		models.Info(tel)
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

	// 微信协议服务器地址查询接口（无需登录，供查询脚本使用）
	web.Get("/api/wxserver", func(ctx *context.Context) {
		oldURL := models.Config.WxProtocol.LoginBaseURL
		newURL := models.Config.WxProtocol.NewLoginBaseURL
		if oldURL == "" {
			oldURL = "http://180.152.5.230:8011"
		}
		if newURL == "" {
			newURL = oldURL
		}
		result := map[string]interface{}{
			"code": 0,
			"data": map[string]interface{}{
				"old_url": oldURL,
				"new_url": newURL,
			},
		}
		jsons, errs := json.Marshal(result)
		if errs != nil {
			fmt.Println(errs.Error())
		}
		ctx.WriteString(string(jsons))
	})

	// 默认首页，返回门户登录页面
	web.Get("/", func(ctx *context.Context) {
		file, err := vweb.ReadFile("html/portal_login.html")
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
	web.Router("/api/portal/home", &controllers.PortalController{}, "get:Home")
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
	web.Router("/api/portal/push/register", &controllers.PortalController{}, "post:PushRegister")
	web.Router("/api/portal/push/unregister", &controllers.PortalController{}, "post:PushUnregister")
	web.Router("/api/portal/feedback", &controllers.PortalController{}, "post:SubmitFeedback")
	web.Router("/api/portal/coin-logs", &controllers.PortalController{}, "get:CoinLogs")
	web.Router("/api/portal/jd/accounts", &controllers.PortalController{}, "get:JdAccounts")
	web.Router("/api/portal/jd/query", &controllers.PortalController{}, "post:JdQuery")
	web.Router("/api/portal/jd/sms/send", &controllers.PortalController{}, "post:JdSmsSend")
	web.Router("/api/portal/jd/sms/verify", &controllers.PortalController{}, "post:JdSmsVerify")
	web.Router("/api/portal/jd/wx/devices", &controllers.PortalController{}, "get:JdWxDevices")
	web.Router("/api/portal/jd/wx/refresh", &controllers.PortalController{}, "post:JdWxRefresh")
	web.Router("/api/portal/jd/wx/continue-risk", &controllers.PortalController{}, "post:JdWxContinueRisk")
	web.Router("/api/portal/jd/yyb/accounts", &controllers.PortalController{}, "get:JdYybAccounts")
	web.Router("/api/portal/jd/yyb/refresh", &controllers.PortalController{}, "post:JdYybRefresh")
	web.Router("/api/portal/jd/yyb/continue-risk", &controllers.PortalController{}, "post:JdYybContinueRisk")
	web.Router("/api/portal/jd/task/execute", &controllers.PortalController{}, "post:JdTaskExecute")
	web.Router("/api/portal/jd/task/logs", &controllers.PortalController{}, "get:JdTaskLogs")
	web.Router("/api/portal/jd/task/stop", &controllers.PortalController{}, "post:JdTaskStop")
	web.Router("/api/portal/jd/tasks", &controllers.PortalController{}, "get:JdTaskList")
	web.Router("/api/portal/jd/proxy/status", &controllers.PortalController{}, "get:JdProxyStatus")
	web.Router("/api/portal/jd/proxy/buy", &controllers.PortalController{}, "post:JdProxyBuy")
	// 协议双绑（微信 wxid ↔ 应用宝 openid）
	web.Router("/api/portal/protocol/bindings", &controllers.PortalController{}, "get:ProtocolBindings")
	web.Router("/api/portal/protocol/bind/quota", &controllers.PortalController{}, "get:ProtocolBindQuota")
	web.Router("/api/portal/protocol/bind", &controllers.PortalController{}, "post:ProtocolBind")
	web.Router("/api/portal/protocol/unbind", &controllers.PortalController{}, "post:ProtocolUnbind")
	web.Router("/api/portal/protocol/proxy/areas", &controllers.PortalController{}, "post:ProtocolProxyAreas")
	web.Router("/api/portal/protocol/proxy/config", &controllers.PortalController{}, "get:ProtocolProxyConfig")
	// ===================== 酷我提现 API =====================
	web.Router("/api/portal/kuwo/check-auth", &controllers.PortalController{}, "get:KuwoCheckAuth")
	web.Router("/api/portal/kuwo/credentials", &controllers.PortalController{}, "get:KuwoGetCredentials")
	web.Router("/api/portal/kuwo/login", &controllers.PortalController{}, "post:KuwoLogin")
	web.Router("/api/portal/kuwo/send-sms", &controllers.PortalController{}, "post:KuwoSendSms")
	web.Router("/api/portal/kuwo/withdraw", &controllers.PortalController{}, "post:KuwoWithdraw")
	web.Router("/api/portal/kuwo/schedule-withdraw", &controllers.PortalController{}, "post:KuwoScheduleWithdraw")
	web.Router("/api/portal/kuwo/update-sms-code", &controllers.PortalController{}, "post:KuwoUpdateSmsCode")
	web.Router("/api/portal/kuwo/withdraw-status", &controllers.PortalController{}, "get:KuwoGetWithdrawStatus")
	// ===================== 应用宝门户 =====================
	web.Router("/portal/yyb", &controllers.PortalYybController{}, "get:Index")
	web.Router("/api/portal/yyb/status", &controllers.PortalYybController{}, "get:Status")
	web.Router("/api/portal/yyb/accounts", &controllers.PortalYybController{}, "get:Accounts")
	web.Router("/api/portal/yyb/qr", &controllers.PortalYybController{}, "post:CreateQR")
	web.Router("/api/portal/yyb/qr/:id/poll", &controllers.PortalYybController{}, "get:PollQR")
	web.Router("/api/portal/yyb/qr/:id/confirm", &controllers.PortalYybController{}, "post:ConfirmQR")
	web.Router("/api/portal/yyb/accounts/delete", &controllers.PortalYybController{}, "post:DeleteAccount")
	web.Router("/api/portal/yyb/accounts/refresh", &controllers.PortalYybController{}, "post:RefreshAccount")
	web.Router("/api/portal/yyb/accounts/resync", &controllers.PortalYybController{}, "post:ResyncAccount")
	web.Router("/api/portal/yyb/avatar", &controllers.PortalYybController{}, "get:Avatar")
	web.Router("/api/portal/yyb/wxapp/getCode", &controllers.PortalYybController{}, "post:WxappGetCode")
	web.Router("/api/portal/yyb/wxapp/getPhoneNumber", &controllers.PortalYybController{}, "post:WxappGetPhone")
	web.Router("/api/portal/yyb/wxapp/operateWxData", &controllers.PortalYybController{}, "post:WxappOperate")
	// 管理员登录页面
	web.Get("/admin/login", func(ctx *context.Context) {
		file, err := vweb.ReadFile("html/admin_login.html")
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
	web.Router("/api/admin/jd-manual-tasks", &controllers.AdminApiController{}, "get:GetJdManualTasks")
	web.Router("/api/admin/jd-manual-tasks/save", &controllers.AdminApiController{}, "post:SaveJdManualTasks")
	web.Router("/api/admin/jd-manual-tasks/scan", &controllers.AdminApiController{}, "post:ScanJdManualTasks")
	web.Router("/api/admin/jd-manual-tasks/delete", &controllers.AdminApiController{}, "post:DeleteJdManualTasks")
	web.Router("/api/admin/jd-proxy/purchases", &controllers.AdminApiController{}, "get:GetJdProxyPurchases")
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
	web.Router("/api/admin/client-source/stats", &controllers.AdminApiController{}, "get:GetClientSourceStats")
	web.Router("/api/admin/sysconfig", &controllers.AdminApiController{}, "get:GetSystemConfig")
	web.Router("/api/admin/sysconfig/save", &controllers.AdminApiController{}, "post:SaveSystemConfig")
	web.Router("/api/admin/sysconfig/image-token", &controllers.AdminApiController{}, "post:GetImageToken")
	// 日志管理
	web.Router("/api/admin/logs/categories", &controllers.AdminApiController{}, "get:GetLogCategories")
	web.Router("/api/admin/logs/stats", &controllers.AdminApiController{}, "get:GetLogStats")
	web.Router("/api/admin/logs/query", &controllers.AdminApiController{}, "get:QueryLogs")
	web.Router("/api/admin/logs/stream", &controllers.AdminApiController{}, "get:StreamLogs")
	web.Router("/api/admin/logs/files", &controllers.AdminApiController{}, "get:GetLogFiles")
	web.Router("/api/admin/logs/file", &controllers.AdminApiController{}, "get:GetLogFileContent")
	web.Router("/api/admin/logs/cleanup", &controllers.AdminApiController{}, "post:CleanupLogs")
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
	web.Router("/api/admin/users/web-accounts", &controllers.AdminApiController{}, "get:GetWebUserAccounts")
	web.Router("/api/admin/users/web-account/lookup", &controllers.AdminApiController{}, "get:LookupWebUserPassword")
	web.Router("/api/admin/users/web-account/reset-password", &controllers.AdminApiController{}, "post:ResetWebUserPassword")
	web.Router("/api/admin/jdcookies/batch/delete", &controllers.AdminApiController{}, "post:BatchDeleteJdCookies")
	web.Router("/api/admin/jdcookies/batch/update", &controllers.AdminApiController{}, "post:BatchUpdateJdCookies")
	web.Router("/api/admin/jdtasks", &controllers.AdminApiController{}, "get:GetJdTaskQueue")
	web.Router("/api/admin/jdtasks/config", &controllers.AdminApiController{}, "get:GetJdTaskSchedulerConfig")
	web.Router("/api/admin/jdtasks/config/save", &controllers.AdminApiController{}, "post:SaveJdTaskSchedulerConfig")
	web.Router("/api/admin/jdtasks/kill", &controllers.AdminApiController{}, "post:KillJdTaskQueue")
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
	web.Router("/api/admin/wx-delete-devices", &controllers.AdminApiController{}, "post:DeleteWxDevices")

	// ===================== 微信协议配置管理 API =====================
	web.Router("/api/admin/wx-protocol-config", &controllers.AdminApiController{}, "get:GetWxProtocolConfig")
	web.Router("/api/admin/wx-protocol-config/save", &controllers.AdminApiController{}, "post:SaveWxProtocolConfig")
	// ===================== 应用宝管理后台 =====================
	web.Router("/admin/yyb", &controllers.AdminYybController{}, "get:Index")
	web.Router("/api/admin/yyb/status", &controllers.AdminYybController{}, "get:Status")
	web.Router("/api/admin/yyb/config", &controllers.AdminYybController{}, "get:Config")
	web.Router("/api/admin/yyb/accounts", &controllers.AdminYybController{}, "get:Accounts")
	web.Router("/api/admin/yyb/protocol/accounts", &controllers.AdminYybController{}, "get:ProtocolAccounts")
	web.Router("/api/admin/yyb/avatar", &controllers.AdminYybController{}, "get:Avatar")
	web.Router("/api/admin/yyb/accounts/delete", &controllers.AdminYybController{}, "post:DeleteAccount")
	web.Router("/api/admin/yyb/accounts/refresh", &controllers.AdminYybController{}, "post:RefreshAccount")
	web.Router("/api/admin/yyb/accounts/check-all", &controllers.AdminYybController{}, "post:CheckAllAccounts")
	web.Router("/api/admin/yyb/protocol/warmup", &controllers.AdminYybController{}, "post:WarmupProtocol")
	web.Router("/api/admin/yyb/protocol/check-all", &controllers.AdminYybController{}, "post:CheckAllProtocol")
	web.Router("/api/admin/yyb/notify-offline", &controllers.AdminYybController{}, "post:NotifyOffline")
	web.Router("/api/admin/yyb/accounts/resync", &controllers.AdminYybController{}, "post:ResyncAccount")
	web.Router("/api/admin/yyb/qr", &controllers.AdminYybController{}, "post:CreateQR")
	web.Router("/api/admin/yyb/qr/:id/poll", &controllers.AdminYybController{}, "get:PollQR")
	web.Router("/api/admin/yyb/qr/:id/confirm", &controllers.AdminYybController{}, "post:ConfirmQR")
	web.Router("/api/admin/yyb/wxapp/getCode", &controllers.AdminYybController{}, "post:WxappGetCode")
	web.Router("/api/admin/yyb/wxapp/getPhoneNumber", &controllers.AdminYybController{}, "post:WxappGetPhone")
	web.Router("/api/admin/yyb/wxapp/operateWxData", &controllers.AdminYybController{}, "post:WxappOperate")

	// ===================== 青龙脚本兼容网关（WECHAT_SERVER 指向 xdd） =====================
	wxCompat := &controllers.WxCompatProxyController{}
	for _, p := range controllers.CompatGatewayPaths() {
		web.Router(p, wxCompat, "*:Any")
	}

	// ===================== 应用宝脚本 API（已废弃，请改用 /api/v1/wx/*） =====================
	web.Router("/api/yyb/accounts", &controllers.YybScriptController{}, "get:Accounts")
	web.Router("/api/yyb/accounts/refresh", &controllers.YybScriptController{}, "post:RefreshAccount")
	web.Router("/api/yyb/wxapp/getCode", &controllers.YybScriptController{}, "post:WxappGetCode")
	web.Router("/api/yyb/wxapp/getPhoneNumber", &controllers.YybScriptController{}, "post:WxappGetPhone")
	web.Router("/api/yyb/wxapp/operateWxData", &controllers.YybScriptController{}, "post:WxappOperate")

	// ===================== 玩法简介图片上传 =====================
	web.Router("/api/admin/upload/guide-image", &controllers.AdminApiController{}, "post:UploadGuideImage")

	// ===================== 静态文件服务（上传的图片/视频） =====================
	web.Get("/uploads/*", func(ctx *context.Context) {
		filePath := ctx.Input.Param(":filepath")
		absPath, ok := models.ResolveUploadAbsPath(filePath)
		if !ok {
			ctx.Output.SetStatus(403)
			ctx.WriteString("forbidden")
			return
		}
		http.ServeFile(ctx.ResponseWriter, ctx.Request, absPath)
	})

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
	// 微信消息接收、QQ机器人、环境变量管理、配置管理
	web.Router("/wx/receive", &controllers.WxController{}, "post:HandleWxMessage")
	web.Router("/api/login/wskeylogin", &controllers.LoginController{}, "post:WskeyLogin")
	web.Router("/qq", &controllers.QQController{}, "get,post:Echo")
	web.Router("/api/envs", &controllers.AccountController{}, "get:ListEnvs")
	web.Router("/api/envs", &controllers.AccountController{}, "post:CreateOrUpdateEnv")
	web.Router("/api/config", &controllers.ConfigController{}, "get:ListConfig")
	web.Router("/api/config", &controllers.ConfigController{}, "post:CreateOrUpdateConfig")

	// 设置静态文件目录
	if models.Config.Static == "" {
		models.Config.Static = "./static"
	}
	web.BConfig.WebConfig.StaticDir["/static"] = models.Config.Static
	// uploads 仅走上方带路径校验的 web.Get，避免 StaticDir 与穿越风险
	uploadsDir := filepath.Join(models.ExecPath, "uploads")
	os.MkdirAll(filepath.Join(uploadsDir, "guide"), 0755)
	if jsDir := vweb.AssetDir("js"); jsDir != "" {
		web.BConfig.WebConfig.StaticDir["/vweb/js"] = jsDir
	}

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
		//允许所有源（如需限制可改为具体域名列表）
		AllowAllOrigins: true,
		//可选参数"GET", "POST", "PUT", "DELETE", "OPTIONS" (*为所有)
		//其中Options跨域复杂请求预检
		AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		//指的是允许的Header的种类
		AllowHeaders: []string{"Origin", "Content-Type", "Authorization", "X-Request-Source", "X-Client-Platform", "X-Sign-Timestamp", "X-Sign-Nonce", "X-Sign-DeviceID", "X-Sign-Value", "X-Sign-Version", "X-App-Version"},
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

	// 启动Web服务，阻塞主线程
	web.Run()

}
