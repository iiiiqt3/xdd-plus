// cron.go
package models

import (
	
	"github.com/beego/beego/v2/adapter/logs"
	"github.com/robfig/cron/v3"
	"math/rand"
	"strconv"
)

var c *cron.Cron

// initCron 初始化定时任务（新增禁用过期CK任务）
func initCron() {
	c = cron.New()
	// 随机时间执行getAuthFlag
	c.AddFunc(strconv.Itoa(rand.Intn(59))+" "+strconv.Itoa(rand.Intn(24))+" * * ?", getAuthFlag)
	// 随机分钟、每7小时5点开始执行GetAuthKey
	c.AddFunc(strconv.Itoa(rand.Intn(59))+" 10 5/7 * ?", GetAuthKey)

	// 原有定时任务
//	c.AddFunc("59 23 * * ?", Daemon)             // 自动重启xdd（每天23点59分）
//	c.AddFunc("0 8-20/1 * * ?", GetNewVersion)   // 检查新版本（每天8-20点每小时0分）
//	c.AddFunc("0 0 * * ?", ResetOrderNumber)     // 重置编号（每天0点）
//	c.AddFunc("5 0,6,12,18 * * ?", UpAutoCookie) // 账密自动登录更新ck（每天0、6、12、18点5分）
	c.AddFunc("0 */6 * * ?", initCookie)       // 账号检测（每6小时执行一次）
	c.AddFunc("59 58 23 L * ?", ClearAllContinuousSignIns) // 连续打卡次数清0（每月最后一日23点58分59秒）
	c.AddFunc("10 9 * * ?",HandleNews) // 新闻推送
	c.AddFunc("0 10 * * ?", CheckWxOfflineAndNotify) // 微信掉线检测推送（每天早上10点检测一次，掉线后仅通知一次）
	
	
	c.AddFunc("15 10 * * ?", func() {	CheckExpiringCKs(2, nil)})    //记录ck2天开始通知
	
	//c.AddFunc("5 */4 * * *", func() {            // wskey转换（每4小时5分）
	//	fmt.Println("开始wskey转换")
	//	updateCookie()
		//UpdateRwskey()
	//})
	c.AddFunc("58 23,8 * * *", func() {          // 导出指定账号（每天23、8点58分）
		logs.Info("开始导出 jd_fcwb_help 账号")
		Exportck("jd_fcwb_help")
		Exportck("jd_joyzbj_help")
		Exportck("jd_zzhb_new_help")
		Exportck("jd_farmnew_code_help")
	})
	c.AddFunc("59 23 * * *", func() {            // 导出农场共享账号（每天23点59分）
		logs.Info("开始导出 jd_farmshare.js 账号")
		Exportck("jd_farmshare")
		Exportck_huanjing("jd_zlyhl")
	})

	// ===== 核心新增：每天凌晨0点1分检查并禁用过期CK =====
		c.AddFunc("1 0 * * ?", DisableExpiredCKsCronWrapper)

	// ===== 过期30天通知+删除：每天12点30分检查一次 =====
	c.AddFunc("30 12 * * ?", func() { NotifyDeleteExpiredCKs(nil) })

	// 启动所有定时任务
	c.Start()
	logs.Info("所有定时任务已启动，包含过期CK禁用任务")
}

// StopCron 停止定时任务（备用）
func StopCron() {
	if c != nil {
		c.Stop()
		logs.Info("定时任务已停止")
	}
}