package models

import (
	"fmt"
	"github.com/beego/beego/v2/adapter/logs"
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
		logs.Warn("解析备注日期失败：%s，备注：%s", err.Error(), remarks)
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
// - 删除前二次验证CK仍为禁用状态，防止误删

func NotifyDeleteExpiredCKs(sender *Sender) {
	NotifyDeleteExpiredCKsWithChannels(sender, NotifyChannels{Web: false, App: false, Robot: true}, nil)
}

func NotifyDeleteExpiredCKsWithChannels(sender *Sender, channels NotifyChannels, activityIDs []string) {
	if sender != nil {
		logs.Info("===== 用户【%d】触发过期CK通知/删除检查 =====", sender.UserID)
	} else {
		logs.Info("===== 开始过期CK通知/删除检查（定时任务） =====")
	}

	now := time.Now()

	totalNotified := 0
	totalDeleted := 0
	notifyCount := 0

	for _, activityConfig := range ActivityConfigs {
		if !activityConfig.IsMonthlyDeduct {
			continue
		}

		if len(activityIDs) > 0 {
			found := false
			for _, aid := range activityIDs {
				if aid == activityConfig.ID {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		currentActivityID := activityConfig.ID
		qlConfig := getQingLongConfigForActivity(currentActivityID)
		if qlConfig == nil {
			continue
		}

		client := NewQingLongClient(qlConfig)
		envs, err := client.QueryEnvs(activityConfig.EnvKey)
		if err != nil || len(envs) == 0 {
			continue
		}

		for _, env := range envs {
			// 只处理已禁用的CK（Status != 0，即被 DisableExpiredCKs 禁用的）
			if env.Status == 0 {
				continue
			}

			// 解析备注中的过期日期
			expireDate, ok := ParseRemarksDate(env.Remarks)
			if !ok {
				continue
			}

			// 计算过期阈值（到期日次日0点，与 CheckRemarksExpired 一致）
			expireThreshold := time.Date(
				expireDate.Year(), expireDate.Month(), expireDate.Day(),
				0, 0, 0, 0, time.Local,
			).AddDate(0, 0, 1)

			// 已过期的天数
			expiredDuration := now.Sub(expireThreshold)
			if expiredDuration < 0 {
				continue // 还没过期
			}
			expiredDays := int(expiredDuration.Hours() / 24)

			// 解析备注中的用户ID
			parts := strings.Split(env.Remarks, "/")
			accountAlias := "未知账号"
			userID := ""
			if len(parts) >= 1 && parts[0] != "" {
				accountAlias = strings.TrimSpace(parts[0])
			}
			if len(parts) >= 2 {
				userID = strings.TrimSpace(parts[1])
			}

			// === 过期第30天：通知用户即将删除 ===
			if expiredDays == 30 {
				msg := fmt.Sprintf(
					"🔴【授权过期删除提醒】\n"+
						"活动：%s\n"+
						"账号备注：%s\n"+
						"到期日期：%s\n"+
						"已过期：%d 天\n\n"+
						"⚠️ 您的CK已被禁用超过30天，将在明天自动删除！\n"+
						"如需保留，请尽快发送【记录授权】续费，删除后无法恢复！",
					activityConfig.Name, accountAlias,
					expireDate.Format(DateLayout), expiredDays)
				logs.Info(">>> 过期删除通知 -> 用户:[%s] 账号:[%s] 活动:[%s] 过期[%d天]",
					userID, accountAlias, activityConfig.Name, expiredDays)
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
				logs.Info(">>> 过期CK超过30天，准备删除青龙变量 -> 账号:[%s] 活动:[%s] ENV ID:[%d] 过期[%d天]",
					accountAlias, activityConfig.Name, env.ID, expiredDays)
				deleteOK := false
				deleteErr := error(nil)
				if env.Status != 0 {
					deleteErr = client.DeleteEnv(env.ID)
					deleteOK = deleteErr == nil
				}
				if userID != "" {
					delMsg := fmt.Sprintf(
						"📦【授权过期删除提醒】\n"+
							"活动：%s\n"+
							"账号备注：%s\n"+
							"到期日期：%s\n"+
							"已过期：%d 天\n\n"+
							"您的CK已过期超过30天，系统已直接删除青龙变量。如需继续使用请重新发送【记录授权】续费。",
						activityConfig.Name, accountAlias, expireDate.Format(DateLayout), expiredDays)
					if !deleteOK {
						delMsg = fmt.Sprintf(
							"📦【授权过期删除失败提醒】\n"+
								"活动：%s\n"+
								"账号备注：%s\n"+
								"到期日期：%s\n"+
								"已过期：%d 天\n\n"+
								"系统尝试删除过期CK失败，请联系管理员处理。失败原因：%v",
							activityConfig.Name, accountAlias, expireDate.Format(DateLayout), expiredDays, deleteErr)
					}
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
				if deleteOK {
					totalDeleted++
				} else {
					logs.Error("删除过期CK失败 -> 账号:[%s] 活动:[%s] ENV ID:[%d] 错误:%v", accountAlias, activityConfig.Name, env.ID, deleteErr)
				}
			}
		}
	}

	resultMsg := fmt.Sprintf("过期CK检查完成：通知 %d 个，删除 %d 个", totalNotified, totalDeleted)
	if sender != nil {
		sender.Reply(resultMsg)
		logs.Info("===== 用户【%d】过期CK通知/删除检查完成 =====", sender.UserID)
	} else {
		logs.Info("===== 过期CK检查完成（定时任务）: %s =====", resultMsg)
	}
}

// CalculateDeductCoin 计算积分（修复负数月数漏洞）
// 🔴 修复点：ActivityConfigs 现在是切片，不能直接用 [] 访问，需使用 getActivityByID
func CalculateDeductCoin(activityID string, months int) (int, error) {
	// 旧代码: activityConfig, ok := ActivityConfigs[activityID]
	// 新代码:
	activityConfig := getActivityByID(activityID)
	if activityConfig == nil {
		return 0, fmt.Errorf("活动ID【%s】不存在", activityID)
	}

	if months < 1 {
		return 0, fmt.Errorf("月数必须≥1")
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
	// 如果是指令触发，打印触发用户信息；如果是定时任务，sender为nil
	if sender != nil {
		logs.Info("===== 用户【%d】触发禁用授权过期CK指令 =====", sender.UserID)
	} else {
		logs.Info("===== 开始检查并禁用过期CK（定时任务） =====")
	}

	// 🔴 修复点：ActivityConfigs 现在是切片，range 返回的是 index 和 value，没有 key
	// 旧代码: for activityID, activityConfig := range ActivityConfigs
	// 新代码:
	for _, activityConfig := range ActivityConfigs {
		if !activityConfig.IsMonthlyDeduct {
			continue
		}

		// 使用结构体中的 ID 字段（初始化时已生成字符串 "1", "2"...）
		currentActivityID := activityConfig.ID

		logs.Info("检查活动【%s】（ID：%s）的过期CK...", activityConfig.Name, currentActivityID)

		qlConfig := getQingLongConfigForActivity(currentActivityID)
		if qlConfig == nil {
			logs.Warn("活动【%s】无法获取青龙配置，跳过", activityConfig.Name)
			continue
		}

		client := NewQingLongClient(qlConfig)
		envs, err := client.QueryEnvs(activityConfig.EnvKey)
		if err != nil {
			logs.Error("查询活动【%s】环境变量失败：%v", activityConfig.Name, err)
			continue
		}
		if len(envs) == 0 {
			logs.Info("活动【%s】无环境变量，跳过", activityConfig.Name)
			continue
		}

		var expiredIDs []int
		// 核心修复：适配青龙状态规则（0=启用，1=禁用）
		// 原逻辑：env.Status == 1 才禁用 → 修正为 env.Status == 0（启用状态的过期CK才需要禁用）
		for _, env := range envs {
			expired, expireDate := CheckRemarksExpired(env.Remarks)
			// 修复：过期 && 当前是启用状态（Status=0），才需要禁用
			if expired && env.Status == 0 {
				expiredIDs = append(expiredIDs, env.ID)
				logs.Info("CK将被禁用：备注【%s】，到期日【%s】，ENV ID【%d】，当前状态【%d】",
					env.Remarks, expireDate, env.ID, env.Status)
			}
		}

		if len(expiredIDs) > 0 {
			err := client.DisableEnvs(expiredIDs)
			if err != nil {
				logs.Error("禁用活动【%s】过期CK失败：%v", activityConfig.Name, err)
				if sender != nil {
					sender.Reply(fmt.Sprintf("禁用活动【%s】过期CK失败：%v", activityConfig.Name, err))
				}
			} else {
				logs.Info("成功禁用活动【%s】的%d个过期CK", activityConfig.Name, len(expiredIDs))
				if sender != nil {
					sender.Reply(fmt.Sprintf("成功禁用活动【%s】的%d个授权过期CK", activityConfig.Name, len(expiredIDs)))
				}
			}
		} else {
			logs.Info("活动【%s】无过期CK", activityConfig.Name)
			if sender != nil {
				sender.Reply(fmt.Sprintf("活动【%s】无授权过期CK需要禁用", activityConfig.Name))
			}
		}
	}

	if sender != nil {
		logs.Info("===== 用户【%d】触发的禁用授权过期CK指令执行完成 =====", sender.UserID)
		sender.Reply("授权过期CK检查禁用完成！")
	} else {
		logs.Info("===== 授权过期CK检查禁用完成（定时任务） =====")
	}
}

func CheckExpiringCKs(expireThresholdDays int, sender *Sender) {
	// 1. 入口日志
	if sender != nil {
		logs.Info("===== [手动触发] 用户【%d】开始检查即将过期CK (阈值: %d天) =====", sender.UserID, expireThresholdDays)
	} else {
		logs.Info("===== [定时任务] 开始检查即将过期CK (阈值: %d天) =====", expireThresholdDays)
	}

	now := time.Now()
	var totalReminded int
	var totalScanned int

	// 2. 遍历所有活动配置
	for _, activityConfig := range ActivityConfigs {
		// 只检查开启了按月扣费的活动
		if !activityConfig.IsMonthlyDeduct {
			continue
		}

		logs.Debug("正在检查活动：【%s】 (ID: %s)", activityConfig.Name, activityConfig.ID)

		currentActivityID := activityConfig.ID
		qlConfig := getQingLongConfigForActivity(currentActivityID)
		if qlConfig == nil {
			logs.Warn("跳过活动【%s】：无法获取青龙配置", activityConfig.Name)
			continue
		}

		client := NewQingLongClient(qlConfig)

		// 3. 查询环境变量
		envs, err := client.QueryEnvs(activityConfig.EnvKey)
		if err != nil {
			logs.Error("查询活动【%s】环境变量失败：%v", activityConfig.Name, err)
			continue
		}

		if len(envs) == 0 {
			continue
		}

		// 4. 遍历每个环境变量 (CK)
		for _, env := range envs {
			// 解析备注中的日期 (格式：账号名称/用户ID/日期)
			expireDate, ok := ParseRemarksDate(env.Remarks)
			if !ok {
				// 无法解析日期则跳过
				continue
			}
			totalScanned++

			// 计算过期时间点 (到期日次日 0:00)
			expireThreshold := time.Date(
				expireDate.Year(),
				expireDate.Month(),
				expireDate.Day(),
				0, 0, 0, 0,
				time.Local,
			).AddDate(0, 0, 1)

			// 计算剩余时间
			durationLeft := expireThreshold.Sub(now)

			// 核心判断逻辑
			isExpired := durationLeft <= 0
			isExpiringSoon := durationLeft > 0 && durationLeft <= time.Duration(expireThresholdDays)*24*time.Hour

			if isExpired {
				// 已过期，跳过预警（由禁用逻辑处理）
				continue
			}

			if isExpiringSoon {
				// --- 解析备注结构 ---
				// 预期格式：账号名称 / 用户ID / 日期
				// 索引：      0       1      2
				parts := strings.Split(env.Remarks, "/")

				var accountAlias string
				var userID string

				// 提取账号名称 (第一个元素)
				if len(parts) >= 1 {
					accountAlias = strings.TrimSpace(parts[0])
				} else {
					accountAlias = "未知账号"
				}

				// 【修改点】提取用户 ID (正数第二个元素，索引 1)
				if len(parts) >= 2 {
					userID = strings.TrimSpace(parts[1])
				}

				if userID == "" {
					logs.Warn("ENV ID[%d] 备注格式错误，无法提取用户ID (需至少2部分): [%s]", env.ID, env.Remarks)
					continue
				}

				// 计算剩余天数显示
				daysLeft := int(durationLeft.Hours() / 24)
				if daysLeft == 0 {
					daysLeft = 1
				}

				// --- 构造消息 ---
				msg := fmt.Sprintf("⚠️【授权即将过期提醒】\n"+
					"活动：%s\n"+
					"账号备注：%s\n"+
					"到期日期：%s\n"+
					"剩余时间：约 %d 天\n"+
					"请及时发送【记录授权】续费，以免服务中断！",
					activityConfig.Name,
					accountAlias,
					expireDate.Format(DateLayout),
					daysLeft)

				logs.Info(">>> 触发推送 -> 用户ID:[%s] | 账号:[%s] | 活动:[%s] | 到期:[%s] | 剩余:[%d天]",
					userID, accountAlias, activityConfig.Name, expireDate.Format(DateLayout), daysLeft)

				// 执行推送
				PushByQQ(userID, msg)

				totalReminded++
			}
		}
	}

	// 5. 总结
	resultMsg := fmt.Sprintf("检查完成！扫描有效授权 %d 个，发现 %d 个即将过期并已推送。", totalScanned, totalReminded)

	if sender != nil {
		logs.Info("===== [手动触发] 用户【%d】检查结束: %s =====", sender.UserID, resultMsg)
		(&JdCookie{}).Push(resultMsg)
	} else {
		logs.Info("===== [定时任务] 检查结束: %s =====", resultMsg)
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
