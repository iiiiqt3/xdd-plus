package models

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"
)

// ===================== 时间常量定义 =====================
const (
	DateLayout         = "2006-01-02"
	DefaultMonthlyCoin = 200
)

// ParseRemarksDate 从备注解析日期
func ParseRemarksDate(remarks string) (time.Time, bool) {
	if remarks == "" {
		return time.Time{}, false
	}
	parts := strings.Split(remarks, "/")
	if len(parts) < 3 {
		return time.Time{}, false
	}
	dateStr := strings.TrimSpace(parts[len(parts)-1])
	date, err := time.ParseInLocation(DateLayout, dateStr, time.Local)
	if err != nil {
	TaskLog().Warnf("解析备注日期失败：%s，备注：%s", err.Error(), remarks)
		return time.Time{}, false
	}
	return date, true
}

// GenerateExpireDate 从当前时间加N个月（新开通用）
func GenerateExpireDate(months int) string {
	now := time.Now()
	expireDate := AddMonthsKeepLastDay(now, months)
	return expireDate.Format(DateLayout)
}

// AddMonthsKeepLastDay 加月并保持月末最后一天（修复 1-31 → 3-02 问题）
func AddMonthsKeepLastDay(t time.Time, months int) time.Time {
	year := t.Year()
	month := t.Month()
	day := t.Day()

	target := time.Date(year, month+time.Month(months), 1, 0, 0, 0, 0, time.Local)
	lastDay := LastDayOfMonth(target)
	if day > lastDay {
		day = lastDay
	}
	return time.Date(target.Year(), target.Month(), day, 0, 0, 0, 0, time.Local)
}

// LastDayOfMonth 获取当月最后一天
func LastDayOfMonth(t time.Time) int {
	first := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
	nextMonth := first.AddDate(0, 1, 0)
	return nextMonth.AddDate(0, 0, -1).Day()
}

// GenerateExpireDateFromBase 从基准日期加月（续费核心！）
// 规则：
// 1. 若原授权仍在有效期内，则从原到期日继续顺延
// 2. 若原授权已过期，则按“当前续费当天”起算 months 个月
func GenerateExpireDateFromBase(baseDate time.Time, months int) string {
	if baseDate.IsZero() {
		return GenerateExpireDate(months)
	}
	if months <= 0 {
		months = 1
	}

	currentBase := time.Date(baseDate.Year(), baseDate.Month(), baseDate.Day(), 0, 0, 0, 0, time.Local)
	now := time.Now()
	currentThreshold := time.Date(currentBase.Year(), currentBase.Month(), currentBase.Day(), 0, 0, 0, 0, time.Local).AddDate(0, 0, 1)

	// 原授权尚在有效期内：直接从原到期日顺延 months 个月
	if now.Before(currentThreshold) {
		return AddMonthsKeepLastDay(currentBase, months).Format(DateLayout)
	}

	// 原授权已过期：从当前续费日期开始重新计算
	return AddMonthsKeepLastDay(now, months).Format(DateLayout)
}

// BuildMonthDeductRemarks 构建备注
func BuildMonthDeductRemarks(baseRemarks string, expireDate string) string {
	parts := strings.Split(baseRemarks, "/")
	if len(parts) >= 3 {
		baseRemarks = strings.Join(parts[:len(parts)-1], "/")
	}
	return fmt.Sprintf("%s/%s", baseRemarks, expireDate)
}

// CheckRemarksExpired 到期日当天可用，次日0点过期
func CheckRemarksExpired(remarks string) (bool, string) {
	date, ok := ParseRemarksDate(remarks)
	if !ok {
		return false, ""
	}

	now := time.Now()
	expireThreshold := time.Date(
		date.Year(),
		date.Month(),
		date.Day(),
		0, 0, 0, 0,
		time.Local,
	).AddDate(0, 0, 1)

	if now.After(expireThreshold) || now.Equal(expireThreshold) {
		return true, date.Format(DateLayout)
	}
	return false, date.Format(DateLayout)
}

// ===================== 过期30天通知+删除逻辑 =====================
// 规则：
// - 授权到期日次日0点，CK 被 DisableExpiredCKs 禁用（已有逻辑）
// - 过期第30天：通知用户即将删除CK
// - 过期第31天及以上：执行删除并通知用户
// 示例：5月1号过期 → 5/2禁用 → 5/31通知即将删除 → 6/1执行删除
//
// 安全保护：
// - 删除前验证 DB 仍为禁用状态（Status!=0），防止误删已续费账号

func NotifyDeleteExpiredCKs(sender *Sender) {
	NotifyDeleteExpiredCKsWithChannels(sender, NotifyChannels{Web: false, App: false, Robot: true}, nil)
}

func NotifyDeleteExpiredCKsWithChannels(sender *Sender, channels NotifyChannels, activityIDs []string) {
	if sender != nil {
		UserLog().Infof("===== 用户【%d】触发过期CK通知/删除检查 =====", sender.UserID)
	} else {
		TaskLog().Infof("===== 开始过期CK通知/删除检查（定时任务） =====")
	}

	now := time.Now()

	totalNotified := 0
	totalDeleted := 0
	notifyCount := 0

	disabledExpiredProjects, err := GetDisabledExpiredProjects()
	if err != nil {
		Error("查询过期已禁用项目失败：%v", err)
		return
	}

	for _, project := range disabledExpiredProjects {
		if len(activityIDs) > 0 {
			found := false
			for _, aid := range activityIDs {
				if aid == project.ActivityID {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		cfg := getActivityByID(project.ActivityID)
		if cfg == nil {
			continue
		}

		expireDate, _ := time.Parse(DateLayout, project.ExpireDate)
		expireThreshold := time.Date(
			expireDate.Year(), expireDate.Month(), expireDate.Day(),
			0, 0, 0, 0, time.Local,
		).AddDate(0, 0, 1)

		expiredDuration := now.Sub(expireThreshold)
		if expiredDuration < 0 {
			continue
		}
		expiredDays := int(expiredDuration.Hours() / 24)

		accountAlias := project.RemarkAlias
		if accountAlias == "" {
			accountAlias = "未知账号"
		}
		userID := fmt.Sprintf("%d", project.UserNumber)

		if expiredDays == 30 {
			msg := fmt.Sprintf(
				"🔴【授权过期删除提醒】\n"+
					"活动：%s\n"+
					"账号备注：%s\n"+
					"到期日期：%s\n"+
					"已过期：%d 天\n\n"+
					"⚠️ 您的CK已被禁用超过30天，将在明天自动删除！\n"+
					"如需保留，请尽快发送【记录授权】续费，删除后无法恢复！",
				project.ActivityName, accountAlias,
				expireDate.Format(DateLayout), expiredDays)
			TaskLog().Infof(">>> 过期删除通知 -> 用户:[%s] 账号:[%s] 活动:[%s] 过期[%d天]",
				userID, accountAlias, project.ActivityName, expiredDays)
			if userID != "" {
				if channels.Robot {
					PushByQQ(userID, msg)
					notifyCount++
					if notifyCount >= 2 {
						delay := 3 + rand.Intn(3)
						time.Sleep(time.Duration(delay) * time.Second)
					}
				}
				if userNumber, err := strconv.Atoi(userID); err == nil {
					CreateSystemWebNotification("授权过期删除提醒", msg, NotifyCategoryAuth, NotifySourceAuth, userNumber, channels)
				}
			}
			totalNotified++
		}

		if expiredDays >= 31 {
			fresh, freshErr := GetActivityProjectByID(project.ID)
			if freshErr != nil || fresh == nil {
				Error("删除前查询项目失败 ID=%d: %v", project.ID, freshErr)
				continue
			}
			if fresh.Status == 0 {
				TaskLog().Infof(">>> 跳过自动删除：账号已重新启用 -> 账号:[%s] 活动:[%s] DB ID:[%d]",
					accountAlias, project.ActivityName, project.ID)
				continue
			}
			if fresh.DeletedAt != nil {
				continue
			}

			TaskLog().Infof(">>> 过期CK超过30天，准备删除 -> 账号:[%s] 活动:[%s] DB ID:[%d] 过期[%d天]",
				accountAlias, project.ActivityName, project.ID, expiredDays)

			if err := DeleteProjectWithQinglongSync(project.ID); err != nil {
				Error("删除过期项目失败 ID=%d: %v", project.ID, err)
			} else {
				totalDeleted++
			}

			if userID != "" {
				delMsg := fmt.Sprintf(
					"📦【授权过期删除提醒】\n"+
						"活动：%s\n"+
						"账号备注：%s\n"+
						"到期日期：%s\n"+
						"已过期：%d 天\n\n"+
						"您的CK已过期超过30天，系统已直接删除。如需继续使用请重新发送【记录授权】续费。",
					project.ActivityName, accountAlias, expireDate.Format(DateLayout), expiredDays)
				if channels.Robot {
					PushByQQ(userID, delMsg)
					notifyCount++
					if notifyCount >= 2 {
						delay := 3 + rand.Intn(3)
						time.Sleep(time.Duration(delay) * time.Second)
					}
				}
				if userNumber, err := strconv.Atoi(userID); err == nil {
					CreateSystemWebNotification("授权过期删除提醒", delMsg, NotifyCategoryAuth, NotifySourceAuth, userNumber, channels)
				}
			}
		}
	}

	resultMsg := fmt.Sprintf("过期CK检查完成：通知 %d 个，删除 %d 个", totalNotified, totalDeleted)
	if sender != nil {
		sender.Reply(resultMsg)
		UserLog().Infof("===== 用户【%d】过期CK通知/删除检查完成 =====", sender.UserID)
	} else {
		TaskLog().Infof("===== 过期CK检查完成（定时任务）: %s =====", resultMsg)
	}
}

// CalculateDeductCoin 计算积分（修复负数月数漏洞）
// 🔴 修复点：ActivityConfigs 现在是切片，不能直接用 [] 访问，需使用 getActivityByID
func CalculateDeductCoin(activityID string, months int) (int, error) {
	activityConfig := getActivityByID(activityID)
	if activityConfig == nil {
		return 0, fmt.Errorf("活动ID【%s】不存在", activityID)
	}

	if months < 1 {
		return 0, fmt.Errorf("月数必须≥1")
	}

	if activityConfig.IsDailyDeduct {
		dailyCoin := activityConfig.DailyCoin
		if dailyCoin <= 0 {
			return 0, fmt.Errorf("每天积分配置错误")
		}
		return months * dailyCoin, nil
	}
	if activityConfig.IsMonthlyDeduct {
		monthlyCoin := activityConfig.MonthlyCoin
		if monthlyCoin <= 0 {
			monthlyCoin = DefaultMonthlyCoin
		}
		return months * monthlyCoin, nil
	}
	return activityConfig.NeedCoin, nil
}

// DisableExpiredCKs 每天禁用过期CK

func DisableExpiredCKsCronWrapper() {
	DisableExpiredCKs(nil) // 定时任务触发时，sender传nil
}

func DisableExpiredCKs(sender *Sender) {
	if sender != nil {
		UserLog().Infof("===== 用户【%d】触发禁用授权过期CK指令 =====", sender.UserID)
	} else {
		TaskLog().Infof("===== 开始检查并禁用过期CK（定时任务） =====")
	}

	expiredProjects, err := GetExpiredProjects()
	if err != nil {
		Error("查询过期项目失败：%v", err)
		if sender != nil {
			sender.Reply(fmt.Sprintf("查询过期项目失败：%v", err))
		}
		return
	}

	if len(expiredProjects) == 0 {
		TaskLog().Infof("无过期CK需要禁用")
		if sender != nil {
			sender.Reply("无授权过期CK需要禁用")
		}
		return
	}

	for _, project := range expiredProjects {
		cfg := getActivityByID(project.ActivityID)
		if cfg == nil {
			continue
		}

		TaskLog().Infof("CK将被禁用：备注【%s】，到期日【%s】，DB ID【%d】，当前状态【%d】",
			project.Remarks, project.ExpireDate, project.ID, project.Status)

		project.Status = 1
		project.SyncStatus = "pending_disable"
		project.SyncError = ""
		if err := UpdateActivityProject(&project); err != nil {
			Error("更新数据库状态失败 ID=%d: %v", project.ID, err)
			continue
		}

		if err := SyncProjectNow(project.ID); err != nil {
			Error("青龙禁用同步失败 ID=%d（保持 pending_disable 待重试）: %v", project.ID, err)
			continue
		}

		TaskLog().Infof("成功禁用过期CK ID=%d，备注=%s", project.ID, project.Remarks)
	}

	TaskLog().Infof("成功禁用 %d 个过期CK", len(expiredProjects))
	if sender != nil {
		sender.Reply(fmt.Sprintf("成功禁用 %d 个授权过期CK", len(expiredProjects)))
	}

	if sender != nil {
		UserLog().Infof("===== 用户【%d】触发的禁用授权过期CK指令执行完成 =====", sender.UserID)
		sender.Reply("授权过期CK检查禁用完成！")
	} else {
		TaskLog().Infof("===== 授权过期CK检查禁用完成（定时任务） =====")
	}
}

func GenerateExpireDateFromDays(days int) string {
	return time.Now().AddDate(0, 0, days).Format(DateLayout)
}

func CheckExpiringCKs(expireThresholdDays int, sender *Sender) {
	if sender != nil {
		UserLog().Infof("===== [手动触发] 用户【%d】开始检查即将过期CK (阈值: %d天) =====", sender.UserID, expireThresholdDays)
	} else {
		TaskLog().Infof("===== [定时任务] 开始检查即将过期CK (阈值: %d天) =====", expireThresholdDays)
	}

	now := time.Now()
	var totalReminded int
	var totalScanned int

	expiringProjects, err := GetExpiringProjects(expireThresholdDays)
	if err != nil {
		Error("查询即将过期项目失败：%v", err)
		return
	}

	for _, project := range expiringProjects {
		cfg := getActivityByID(project.ActivityID)
		if cfg == nil {
			continue
		}

		expireDate, _ := time.Parse(DateLayout, project.ExpireDate)
		expireThreshold := time.Date(
			expireDate.Year(),
			expireDate.Month(),
			expireDate.Day(),
			0, 0, 0, 0,
			time.Local,
		).AddDate(0, 0, 1)

		durationLeft := expireThreshold.Sub(now)
		daysLeft := int(durationLeft.Hours() / 24)
		if daysLeft == 0 {
			daysLeft = 1
		}

		accountAlias := project.RemarkAlias
		if accountAlias == "" {
			accountAlias = "未知账号"
		}

		userID := fmt.Sprintf("%d", project.UserNumber)

		msg := fmt.Sprintf("⚠️【授权即将过期提醒】\n"+
			"活动：%s\n"+
			"账号备注：%s\n"+
			"到期日期：%s\n"+
			"剩余时间：约 %d 天\n"+
			"请及时发送【记录授权】续费，以免服务中断！",
			project.ActivityName,
			accountAlias,
			expireDate.Format(DateLayout),
			daysLeft)

		TaskLog().Infof(">>> 触发推送 -> 用户ID:[%s] | 账号:[%s] | 活动:[%s] | 到期:[%s] | 剩余:[%d天]",
			userID, accountAlias, project.ActivityName, expireDate.Format(DateLayout), daysLeft)

		PushByQQ(userID, msg)

		totalScanned++
		totalReminded++
	}

	// 5. 总结
	resultMsg := fmt.Sprintf("检查完成！扫描有效授权 %d 个，发现 %d 个即将过期并已推送。", totalScanned, totalReminded)

	if sender != nil {
		UserLog().Infof("===== [手动触发] 用户【%d】检查结束: %s =====", sender.UserID, resultMsg)
		(&JdCookie{}).Push(resultMsg)
	} else {
		TaskLog().Infof("===== [定时任务] 检查结束: %s =====", resultMsg)
	}
}

// GetUserAuthMonths 输入1-12月
func GetUserAuthMonths(sender *Sender, msgChannel chan string) (int, bool) {
	input, exit := getUserInputByField(sender, msgChannel, InputField{
		Key:    "auth_months",
		Prompt: "请输入授权月数（1-12个月，输入q退出）：",
		Validator: func(s string) (bool, string) {
			if checkExit(s) {
				return false, "程序已退出"
			}
			months, err := strconv.Atoi(s)
			if err != nil {
				return false, "月数必须是数字，请重新输入！"
			}
			if months < 1 || months > 12 {
				return false, "月数必须在1-12之间，请重新输入！"
			}
			return true, ""
		},
		Required:   true,
		TrimSpace:  true,
		TimeoutSec: 60,
		ErrorMsg:   "月数输入错误，请输入1-12之间的数字！",
	})
	if exit {
		return 0, true
	}
	months, _ := strconv.Atoi(input)
	return months, false
}
