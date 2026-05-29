package models

import (
	"fmt"
	"log"
	"math"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"gorm.io/gorm"
)

type PortalProjectItem struct {
	ActivityID      string                `json:"activityId"`
	ActivityName    string                `json:"activityName"`
	EnvKey          string                `json:"envKey"`
	EnvID           int                   `json:"envId"`
	EnvValue        string                `json:"envValue"`
	QingLongConfig  string                `json:"qingLongConfig"`
	Remark          string                `json:"remark"`
	DisplayName     string                `json:"displayName"`
	ExpireDate      string                `json:"expireDate"`
	Status          int                   `json:"status"`
	StatusText      string                `json:"statusText"`
	UpdatedAt       string                `json:"updatedAt"`
	CreatedAt       string                `json:"createdAt"`
	IsMonthlyDeduct bool                  `json:"isMonthlyDeduct"`
	MonthlyCoin     int                   `json:"monthlyCoin"`
	IsDailyDeduct   bool                  `json:"isDailyDeduct"`
	DailyCoin       int                   `json:"dailyCoin"`
	NeedCoin        int                   `json:"needCoin"`
	GrantExpireDate string                `json:"grantExpireDate"`
	BizStatus       string                `json:"bizStatus"`
	BizStatusText   string                `json:"bizStatusText"`
	DaysLeft        int                   `json:"daysLeft"`
	PriceText       string                `json:"priceText"`
	InputFields     []PortalActivityField `json:"inputFields"`
	CKTemplate      string                `json:"ckTemplate"`
}

type PortalActivityField struct {
	Key        string `json:"key"`
	Prompt     string `json:"prompt"`
	Required   bool   `json:"required"`
	TrimSpace  bool   `json:"trimSpace"`
	TimeoutSec int    `json:"timeoutSec"`
	ErrorMsg   string `json:"errorMsg"`
}

type PortalActivityItem struct {
	ID              string                `json:"id"`
	Name            string                `json:"name"`
	EnvKey          string                `json:"envKey"`
	NeedCoin        int                   `json:"needCoin"`
	IsMonthlyDeduct bool                  `json:"isMonthlyDeduct"`
	MonthlyCoin     int                   `json:"monthlyCoin"`
	IsDailyDeduct   bool                  `json:"isDailyDeduct"`
	DailyCoin       int                   `json:"dailyCoin"`
	MinDays         int                   `json:"minDays"`
	QingLongConfig  string                `json:"qingLongConfig"`
	Guide           string                `json:"guide"`
	InputFields     []PortalActivityField `json:"inputFields"`
	CKTemplate      string                `json:"ckTemplate"`
	Enabled         bool                  `json:"enabled"`
}

type PortalDashboard struct {
	Number                    int    `json:"number"`
	QQ                        string `json:"qq"`
	Wxid                      string `json:"wxid"`
	Coin                      int    `json:"coin"`
	Nickname                  string `json:"nickname"`
	AccountID                 int    `json:"accountId"`
	Username                  string `json:"username"`
	BoundAt                   string `json:"boundAt"`
	LastLoginAt               string `json:"lastLoginAt"`
	AvailableCount            int    `json:"availableCount"`
	ProjectCount              int    `json:"projectCount"`
	JoinedCount               int    `json:"joinedCount"`
	ActiveCount               int    `json:"activeCount"`
	ValidCkCount              int    `json:"validCkCount"`
	ExpiringCount             int    `json:"expiringCount"`
	ExpiredCount              int    `json:"expiredCount"`
	NotificationTotal         int64  `json:"notificationTotal"`
	NotificationUnread        int64  `json:"notificationUnread"`
	CheckedInToday            bool   `json:"checkedInToday"`
	ContinuousDays            int    `json:"continuousDays"`
	NextCheckInBonus          int    `json:"nextCheckInBonus"`
	DaysUntilNextCheckInBonus int    `json:"daysUntilNextCheckInBonus"`
	PrayedToday               bool   `json:"prayedToday"`
}

type PortalPrayRecord struct {
	ID         int    `gorm:"primaryKey"`
	UserNumber int    `gorm:"uniqueIndex:idx_portal_pray_user_day,priority:1"`
	PrayDate   string `gorm:"size:10;uniqueIndex:idx_portal_pray_user_day,priority:2"`
	CreatedAt  time.Time
}

type PortalProfile struct {
	User    *User           `json:"user"`
	Account *WebUserAccount `json:"account"`
}

func GetPortalProfile(accountID int) (*PortalProfile, error) {
	account, err := GetWebUserAccountByID(accountID)
	if err != nil {
		return nil, fmt.Errorf("未找到登录账号，请重新登录；如果刚注册过，请确认注册时填写的是机器人【用户信息】里的 UserID")
	}
	var user User
	if err := db.Where("number = ?", account.UserNumber).First(&user).Error; err != nil {
		return nil, fmt.Errorf("未找到绑定用户")
	}
	return &PortalProfile{User: &user, Account: account}, nil
}

func GetPortalDashboard(accountID int) (*PortalDashboard, error) {
	profile, err := GetPortalProfile(accountID)
	if err != nil {
		return nil, err
	}

	notificationTotal, notificationUnread := GetPortalNotificationCounts(profile.User.Number)
	availableCount := CountPortalAvailableActivities()
	projectCount, joinedCount, activeCount, expiringCount, expiredCount := CountPortalProjectStats(profile.User.Number)
	checkedInToday, continuousDays := getPortalCheckInStatus(profile.User)
	nextBonus, daysUntilNextBonus := getNextCheckInBonus(continuousDays)
	prayedToday := hasPrayedToday(profile.User.Number)

	return &PortalDashboard{
		Number:                    profile.User.Number,
		QQ:                        profile.User.QQ,
		Wxid:                      profile.User.Wxid,
		Coin:                      profile.User.Coin,
		Nickname:                  profile.User.Nickname,
		AccountID:                 profile.Account.ID,
		Username:                  profile.Account.Username,
		BoundAt:                   formatPortalTime(profile.Account.BoundAt),
		LastLoginAt:               formatPortalTime(profile.Account.LastLoginAt),
		AvailableCount:            availableCount,
		ProjectCount:              projectCount,
		JoinedCount:               joinedCount,
		ActiveCount:               activeCount,
		ValidCkCount:              activeCount + expiringCount,
		ExpiringCount:             expiringCount,
		ExpiredCount:              expiredCount,
		NotificationTotal:         notificationTotal,
		NotificationUnread:        notificationUnread,
		CheckedInToday:            checkedInToday,
		ContinuousDays:            continuousDays,
		NextCheckInBonus:          nextBonus,
		DaysUntilNextCheckInBonus: daysUntilNextBonus,
		PrayedToday:               prayedToday,
	}, nil
}

func isPortalActivityAvailable(cfg *ActivityConfig) bool {
	if cfg == nil || !cfg.Enabled || strings.TrimSpace(cfg.EnvKey) == "" || strings.TrimSpace(cfg.CKTemplate) == "" || len(cfg.InputFields) == 0 {
		return false
	}
	return getQingLongConfigForActivity(cfg.ID) != nil
}

func CountPortalAvailableActivities() int {
	activityConfigsMu.RLock()
	configs := make([]*ActivityConfig, 0, len(ActivityConfigs))
	configs = append(configs, ActivityConfigs...)
	activityConfigsMu.RUnlock()

	count := 0
	for _, cfg := range configs {
		if isPortalActivityAvailable(cfg) {
			count++
		}
	}
	return count
}

func CountPortalProjectStats(userNumber int) (int, int, int, int, int) {
	projects, err := GetPortalProjects(userNumber)
	if err != nil {
		return 0, 0, 0, 0, 0
	}
	joined := map[string]bool{}
	activeCount := 0
	expiringCount := 0
	expiredCount := 0
	for _, project := range projects {
		if project.ActivityID != "" && project.BizStatus != "expired" {
			joined[project.ActivityID] = true
		}
		switch project.BizStatus {
		case "expired":
			expiredCount++
		case "expiring":
			expiringCount++
		default:
			activeCount++
		}
	}
	return len(projects), len(joined), activeCount, expiringCount, expiredCount
}

func getNextCheckInBonus(days int) (int, int) {
	switch {
	case days < 10:
		return 50, 10 - days
	case days < 20:
		return 150, 20 - days
	case days < 30:
		return 300, 30 - days
	default:
		return 0, 0
	}
}

func getPortalCheckInStatus(user *User) (bool, int) {
	if user == nil {
		return false, 0
	}
	today, _ := time.ParseInLocation("2006-01-02", time.Now().Local().Format("2006-01-02"), time.Local)
	return user.SignInDate.Unix() >= today.Unix(), user.ContinuousSignIns
}

func hasPrayedToday(userNumber int) bool {
	var count int64
	today := time.Now().Format("2006-01-02")
	db.Model(&PortalPrayRecord{}).Where("user_number = ? AND pray_date = ?", userNumber, today).Count(&count)
	return count > 0
}

func GetPortalActivities() []PortalActivityItem {
	activityConfigsMu.RLock()
	defer activityConfigsMu.RUnlock()

	result := make([]PortalActivityItem, 0, len(ActivityConfigs))
	for _, cfg := range ActivityConfigs {
		if !isPortalActivityAvailable(cfg) {
			continue
		}
		item := PortalActivityItem{
			ID:              cfg.ID,
			Name:            cfg.Name,
			EnvKey:          cfg.EnvKey,
			NeedCoin:        cfg.NeedCoin,
			IsMonthlyDeduct: cfg.IsMonthlyDeduct,
			MonthlyCoin:     cfg.MonthlyCoin,
			IsDailyDeduct:   cfg.IsDailyDeduct,
			DailyCoin:       cfg.DailyCoin,
			MinDays:         cfg.MinDays,
			QingLongConfig:  cfg.QingLongConfigName,
			Guide:           strings.TrimSpace(cfg.Guide),
			CKTemplate:      cfg.CKTemplate,
			Enabled:         cfg.Enabled,
		}
		for _, field := range cfg.InputFields {
			prompt := field.Prompt
			if strings.TrimSpace(prompt) == "" {
				prompt = field.Key
			}
			pfield := PortalActivityField{
				Key:        field.Key,
				Prompt:     prompt,
				Required:   field.Required,
				TrimSpace:  field.TrimSpace,
				TimeoutSec: field.TimeoutSec,
				ErrorMsg:   field.ErrorMsg,
			}
			if pfield.TimeoutSec == 0 {
				pfield.TimeoutSec = 60
			}
			item.InputFields = append(item.InputFields, pfield)
		}
		log.Printf("[门户API] 活动[%s] ID=%s, 字段数=%d, 字段详情=%v", cfg.Name, cfg.ID, len(item.InputFields), item.InputFields)
		result = append(result, item)
	}
	return result
}

func GetPortalProjects(userNumber int) ([]PortalProjectItem, error) {
	var projects []PortalProjectItem

	dbProjects, err := GetActivityProjectsByUser(userNumber)
	if err != nil {
		return nil, err
	}

	activityConfigsMu.RLock()
	configMap := make(map[string]*ActivityConfig)
	for _, cfg := range ActivityConfigs {
		if cfg != nil {
			configMap[cfg.ID] = cfg
		}
	}
	activityConfigsMu.RUnlock()

	for _, dbProj := range dbProjects {
		cfg, exists := configMap[dbProj.ActivityID]
		if !exists {
			continue
		}
		if !isPortalActivityAvailable(cfg) {
			continue
		}

		projectFields := make([]PortalActivityField, 0, len(cfg.InputFields))
		for _, field := range cfg.InputFields {
			projectFields = append(projectFields, PortalActivityField{
				Key:        field.Key,
				Prompt:     field.Prompt,
				Required:   field.Required,
				TrimSpace:  field.TrimSpace,
				TimeoutSec: field.TimeoutSec,
				ErrorMsg:   field.ErrorMsg,
			})
		}

		expireDate := ""
		bizStatus := "active"
		bizStatusText := "授权有效中"
		daysLeft := 0
		if dbProj.ExpireDate != "" {
			expireDate = dbProj.ExpireDate
			if t, err := time.Parse(DateLayout, dbProj.ExpireDate); err == nil {
				threshold := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.Local).AddDate(0, 0, 1)
				durationLeft := threshold.Sub(time.Now())
				daysLeft = int(durationLeft.Hours() / 24)
				if daysLeft == 0 && durationLeft > 0 {
					daysLeft = 1
				}
				if durationLeft <= 0 || dbProj.Status != 0 {
					bizStatus = "expired"
					bizStatusText = "已失效"
					if daysLeft < 0 {
						daysLeft = 0
					}
				} else if durationLeft > 0 && durationLeft <= 3*24*time.Hour {
					bizStatus = "expiring"
					bizStatusText = "快到期"
				}
			}
		} else if dbProj.Status != 0 {
			bizStatus = "expired"
			bizStatusText = "已失效"
		}

		statusText := "已禁用"
		if dbProj.Status == 0 {
			statusText = "已启用"
		}
		priceText := fmt.Sprintf("一次性 %d 积分", dbProj.NeedCoin)
		if dbProj.IsDailyDeduct {
			priceText = fmt.Sprintf("每天 %d 积分", dbProj.DailyCoin)
		} else if dbProj.IsMonthlyDeduct {
			priceText = fmt.Sprintf("每月 %d 积分", dbProj.MonthlyCoin)
		}

		projects = append(projects, PortalProjectItem{
			ActivityID:      dbProj.ActivityID,
			ActivityName:    dbProj.ActivityName,
			EnvKey:          dbProj.EnvKey,
			EnvID:           dbProj.QingLongEnvID,
			EnvValue:        dbProj.EnvValue,
			QingLongConfig:  dbProj.QingLongConfigName,
			Remark:          dbProj.Remarks,
			DisplayName:     dbProj.RemarkAlias,
			ExpireDate:      expireDate,
			Status:          dbProj.Status,
			StatusText:      statusText,
			UpdatedAt:       dbProj.UpdatedAt.Format("2006-01-02 15:04:05"),
			CreatedAt:       dbProj.CreatedAt.Format("2006-01-02 15:04:05"),
			IsMonthlyDeduct: dbProj.IsMonthlyDeduct,
			MonthlyCoin:     dbProj.MonthlyCoin,
			IsDailyDeduct:   dbProj.IsDailyDeduct,
			DailyCoin:       dbProj.DailyCoin,
			NeedCoin:        dbProj.NeedCoin,
			GrantExpireDate: dbProj.GrantExpireDate,
			BizStatus:       bizStatus,
			BizStatusText:   bizStatusText,
			DaysLeft:        daysLeft,
			PriceText:       priceText,
			InputFields:     projectFields,
			CKTemplate:      cfg.CKTemplate,
		})
	}

	sort.Slice(projects, func(i, j int) bool {
		if projects[i].ActivityID != projects[j].ActivityID {
			return projects[i].ActivityID < projects[j].ActivityID
		}
		return projects[i].DisplayName < projects[j].DisplayName
	})
	return projects, nil
}

func PortalCreateProject(userNumber int, activityID string, inputs map[string]string, userRemarks string, months int) (string, error) {
	cfg := getActivityByID(activityID)
	if cfg == nil {
		return "", fmt.Errorf("活动不存在")
	}
	if strings.TrimSpace(userRemarks) == "" {
		return "", fmt.Errorf("备注名不能为空")
	}
	if strings.Contains(userRemarks, "/") {
		return "", fmt.Errorf("备注名不能包含/")
	}

	cleanedInputs := make(map[string]string)
	for _, field := range cfg.InputFields {
		value := strings.TrimSpace(inputs[field.Key])
		if field.TrimSpace {
			value = strings.TrimSpace(value)
		}
		if field.Required && value == "" {
			return "", fmt.Errorf("请填写%s", field.Prompt)
		}
		if field.Validator != nil && value != "" {
			if ok, msg := field.Validator(value); !ok {
				if msg == "" {
					msg = field.ErrorMsg
				}
				return "", fmt.Errorf(msg)
			}
		}
		cleanedInputs[field.Key] = value
	}

	if cfg.IsDailyDeduct {
		minDays := cfg.MinDays
		if minDays < 1 {
			minDays = 1
		}
		if months < minDays || months > 365 {
			return "", fmt.Errorf("授权天数需在%d-365之间", minDays)
		}
	} else if cfg.IsMonthlyDeduct {
		if months < 1 || months > 12 {
			return "", fmt.Errorf("授权月数需在1-12之间")
		}
	} else {
		months = 0
	}

	totalCoin := cfg.NeedCoin
	expireDate := ""
	if cfg.IsDailyDeduct {
		totalCoin = cfg.DailyCoin * months
		expireDate = GenerateExpireDateFromDays(months)
	} else if cfg.IsMonthlyDeduct {
		totalCoin = cfg.MonthlyCoin * months
		expireDate = GenerateExpireDate(months)
	}
	userCoin := GetCoin(userNumber)
	if userCoin < totalCoin {
		return "", fmt.Errorf("积分不足，当前%d，需要%d", userCoin, totalCoin)
	}

	ckValue := cfg.CKBuilder(cleanedInputs)
	finalRemarks := cfg.RemarksBuilder(userNumber, strings.TrimSpace(userRemarks), cleanedInputs)
	if cfg.IsMonthlyDeduct || cfg.IsDailyDeduct {
		finalRemarks = fmt.Sprintf("%s/%s", finalRemarks, expireDate)
	}

	project := &ActivityProject{
		ActivityID:         cfg.ID,
		ActivityName:       cfg.Name,
		EnvKey:             cfg.EnvKey,
		EnvValue:           ckValue,
		Remarks:            finalRemarks,
		UserNumber:         userNumber,
		QingLongConfigName: cfg.QingLongConfigName,
		Status:             0,
		ExpireDate:         expireDate,
		IsMonthlyDeduct:    cfg.IsMonthlyDeduct,
		MonthlyCoin:        cfg.MonthlyCoin,
		IsDailyDeduct:      cfg.IsDailyDeduct,
		DailyCoin:          cfg.DailyCoin,
		SyncStatus:         "pending",
	}
	if cfg.IsDailyDeduct && cfg.MinDays > 0 {
		minDays := cfg.MinDays
		project.MinDays = &minDays
	}
	if !cfg.IsMonthlyDeduct && !cfg.IsDailyDeduct {
		project.NeedCoin = cfg.NeedCoin
	}

	if err := CreateActivityProject(project); err != nil {
		return "", fmt.Errorf("保存到数据库失败：%v", err)
	}

	RemCoin(userNumber, totalCoin)

	go TriggerSync(project.ID)

	if cfg.IsDailyDeduct {
		return fmt.Sprintf("添加%s成功，扣除%d积分，有效期至%s", cfg.Name, totalCoin, expireDate), nil
	} else if cfg.IsMonthlyDeduct {
		return fmt.Sprintf("添加%s成功，扣除%d积分，有效期至%s", cfg.Name, totalCoin, expireDate), nil
	}
	return fmt.Sprintf("添加%s成功，扣除%d积分", cfg.Name, totalCoin), nil
}

func PortalRenewProject(userNumber int, activityID, remarks string, months int) (string, error) {
	cfg := getActivityByID(activityID)
	if cfg == nil {
		return "", fmt.Errorf("活动不存在")
	}
	if !cfg.IsMonthlyDeduct && !cfg.IsDailyDeduct {
		return "", fmt.Errorf("该活动不支持网页端授权续费")
	}

	var totalCoin int
	if cfg.IsDailyDeduct {
		minDays := cfg.MinDays
		if minDays < 1 {
			minDays = 1
		}
		if months < minDays || months > 365 {
			return "", fmt.Errorf("授权天数需在%d-365之间", minDays)
		}
		totalCoin = cfg.DailyCoin * months
	} else {
		if months < 1 || months > 12 {
			return "", fmt.Errorf("授权月数需在1-12之间")
		}
		totalCoin = cfg.MonthlyCoin * months
	}

	userCoin := GetCoin(userNumber)
	if userCoin < totalCoin {
		return "", fmt.Errorf("积分不足，当前%d，需要%d", userCoin, totalCoin)
	}

	project, err := GetActivityProjectByRemarks(activityID, remarks, cfg.EnvKey)
	if err != nil {
		return "", fmt.Errorf("未找到待续费账号")
	}

	baseTime, hasOldDate := ParseRemarksDate(remarks)
	var newExpireDate string
	if cfg.IsDailyDeduct {
		if hasOldDate {
			now := time.Now()
			expireThreshold := time.Date(baseTime.Year(), baseTime.Month(), baseTime.Day(), 0, 0, 0, 0, time.Local).AddDate(0, 0, 1)
			if now.Before(expireThreshold) {
				newExpireDate = baseTime.AddDate(0, 0, months).Format(DateLayout)
			} else {
				newExpireDate = GenerateExpireDateFromDays(months)
			}
		} else {
			newExpireDate = GenerateExpireDateFromDays(months)
		}
	} else {
		newExpireDate = GenerateExpireDate(months)
		if hasOldDate {
			newExpireDate = GenerateExpireDateFromBase(baseTime, months)
		}
	}
	newRemarks := BuildMonthDeductRemarks(remarks, newExpireDate)

	project.Remarks = newRemarks
	project.ExpireDate = newExpireDate
	project.NeedCoin = 0
	project.Status = 0
	if cfg.IsDailyDeduct {
		project.IsDailyDeduct = true
		project.DailyCoin = cfg.DailyCoin
	}
	project.SyncStatus = "pending_update"
	project.SyncError = ""
	if err := UpdateActivityProject(project); err != nil {
		return "", fmt.Errorf("更新数据库失败：%v", err)
	}

	RemCoin(userNumber, totalCoin)

	go TriggerSync(project.ID)

	return fmt.Sprintf("授权成功，扣除%d积分，有效期至%s", totalCoin, newExpireDate), nil
}

func PortalDeleteProject(userNumber int, activityID, remarks string) (string, error) {
	cfg := getActivityByID(activityID)
	if cfg == nil {
		return "", fmt.Errorf("活动不存在")
	}

	returnCoin := 0
	project, err := GetActivityProjectByRemarks(activityID, remarks, cfg.EnvKey)
	if err != nil {
		return "", fmt.Errorf("未找到对应项目记录")
	}

	if (cfg.IsMonthlyDeduct || cfg.IsDailyDeduct) && project.NeedCoin == 0 {
		paidDays := CalcPaidRemainingDays(project)
		if cfg.IsDailyDeduct && project.DailyCoin > 0 {
			returnCoin = project.DailyCoin * paidDays
		} else if project.MonthlyCoin > 0 {
			returnCoin = int(math.Round(float64(project.MonthlyCoin) * float64(paidDays) / 30))
		}
	}

	if err := SoftDeleteActivityProject(project.ID); err != nil {
		return "", fmt.Errorf("删除失败：%v", err)
	}

	go TriggerSync(project.ID)

	if returnCoin > 0 {
		AdddCoin(userNumber, returnCoin)
		return fmt.Sprintf("删除成功，已退还 %d 积分", returnCoin), nil
	}
	return "删除成功", nil
}

func PortalUpdateProject(userNumber int, activityID, remarks, newCkValue string) (string, error) {
	cfg := getActivityByID(activityID)
	if cfg == nil {
		return "", fmt.Errorf("活动不存在")
	}
	if strings.TrimSpace(newCkValue) == "" {
		return "", fmt.Errorf("CK 值不能为空")
	}

	project, err := GetActivityProjectByRemarks(activityID, remarks, cfg.EnvKey)
	if err != nil {
		return "", fmt.Errorf("未找到对应项目记录")
	}

	project.EnvValue = newCkValue
	project.SyncStatus = "pending_update"
	project.SyncError = ""
	if err := UpdateActivityProject(project); err != nil {
		return "", fmt.Errorf("更新 CK 失败：%v", err)
	}

	go TriggerSync(project.ID)

	return "CK 更新成功", nil
}

func PortalQueryProjectIncome(userNumber int, activityID, remarks string) (string, error) {
	cfg := getActivityByID(activityID)
	if cfg == nil {
		return "", fmt.Errorf("活动不存在")
	}
	if cfg.ScriptPaths.Query == "" {
		return "", fmt.Errorf("该项目暂无查询脚本")
	}

	project, err := GetActivityProjectByRemarks(activityID, remarks, cfg.EnvKey)
	if err != nil {
		return "", fmt.Errorf("未找到对应项目记录（数据库中不存在）")
	}

	if project.UserNumber != userNumber {
		return "", fmt.Errorf("无权查询此账号")
	}

	if project.Status != 0 {
		return "", fmt.Errorf("该项目当前已禁用或已过期，请先续费/重新授权")
	}

	if (cfg.IsMonthlyDeduct || cfg.IsDailyDeduct) && project.ExpireDate != "" {
		expireTimeObj, parseErr := time.Parse("2006-01-02", project.ExpireDate)
		if parseErr == nil && time.Now().After(expireTimeObj) {
			return "", fmt.Errorf("该项目授权已过期（过期时间：%s），请先发送【记录授权】续费", project.ExpireDate)
		}
	}

	ckValue := project.EnvValue
	if ckValue == "" {
		return "", fmt.Errorf("CK数据为空（可能同步异常），请联系管理员")
	}

	scriptPath := cfg.ScriptPaths.Query
	scriptExt := strings.ToLower(filepath.Ext(scriptPath))
	execCmd := ""
	switch scriptExt {
	case ".js":
		execCmd = "node"
	case ".py":
		execCmd = "python3"
	default:
		return "", fmt.Errorf("暂不支持该查询脚本类型")
	}

	sender := &Sender{UserID: userNumber}
	output, err := executeScript(sender, execCmd, scriptPath, ckValue)
	if err != nil {
		return "", fmt.Errorf("查询收入失败：%v", err)
	}
	if strings.TrimSpace(output) == "" {
		return "暂无查询结果", nil
	}
	return strings.TrimSpace(output), nil
}

func PortalRedeemKey(userNumber int, token string) (string, int, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return "", 0, fmt.Errorf("请输入卡密")
	}
	var result string
	if strings.HasPrefix(token, "ZSKM") {
		result = use_ZSKey(token, userNumber)
	} else if strings.HasPrefix(token, "XDD") {
		result = useKey(token, userNumber)
	} else {
		// 两种都尝试：先试普通卡密，再试赠送卡密
		result = useKey(token, userNumber)
		if result == "查无此卡" {
			result = use_ZSKey(token, userNumber)
		}
	}

	balance := GetCoin(userNumber)

	// 判断是否成功：机器人逻辑中成功返回 "使用成功"
	if strings.Contains(result, "使用成功") {
		// 提取增加积分数
		msg := fmt.Sprintf("✅ %s\n当前余额：%d 积分\n如有异常请联系大师", result, balance)
		return msg, balance, nil
	}

	// 失败场景，返回友好提示 + 联系方式
	errMsg := fmt.Sprintf("❌ %s\n如有异常请联系大师", result)
	return errMsg, balance, fmt.Errorf(result)
}

func PortalCheckIn(userNumber int) (string, error) {
	return portalCheckIn(userNumber)
}

func PortalPray(userNumber int) (string, error) {
	return portalPray(userNumber)
}

func portalCheckIn(userNumber int) (string, error) {
	var u User
	ntime := time.Now()
	zero, _ := time.ParseInLocation("2006-01-02", ntime.Local().Format("2006-01-02"), time.Local)
	total := []int{}

	err := db.Where("number = ?", userNumber).First(&u).Error
	if err != nil {
		u = User{Class: "portal", Number: userNumber, Coin: 1, ActiveAt: ntime, LastSignIn: ntime, ContinuousSignIns: 1, SignInDate: ntime}
		if err := db.Create(&u).Error; err != nil {
			return "创建用户失败", err
		}
	} else {
		if zero.Unix() > u.SignInDate.Unix() {
			if u.LastSignIn.Day()+1 == ntime.Day() && u.LastSignIn.Month() == ntime.Month() && u.LastSignIn.Year() == ntime.Year() {
				u.ContinuousSignIns += 1
			} else {
				u.ContinuousSignIns = 1
			}
			u.SignInDate = ntime
		} else {
			return fmt.Sprintf("你今天已经打卡过了，当月已连续打卡 %d 天，积分余额%d。", u.ContinuousSignIns, u.Coin), nil
		}
	}

	bonus := 0
	if u.ContinuousSignIns == 10 {
		bonus = 50
	} else if u.ContinuousSignIns == 20 {
		bonus = 150
	} else if u.ContinuousSignIns == 30 {
		bonus = 300
	}

	db.Model(User{}).Select("count(id) as total").Where("sign_in_date > ?", zero).Pluck("total", &total)
	coin := 0
	switch total[0] {
	case 0:
		coin = 15
	case 1:
		coin = 14
	case 2:
		coin = 13
	case 3:
		coin = 12
	case 4:
		coin = 11
	case 5:
		coin = 10
	case 6:
		coin = 9
	case 7:
		coin = 8
	case 50:
		coin = 10
	case 100:
		coin = 20
	default:
		if total[0]%14 == 13 {
			coin = 10
		}
	}
	if total[0] > 7 {
		coin = time.Now().Nanosecond()%5 + 1
	}

	db.Model(&u).Updates(map[string]interface{}{
		"last_sign_in":        ntime,
		"coin":                gorm.Expr(fmt.Sprintf("coin+%d", coin+bonus)),
		"continuous_sign_ins": u.ContinuousSignIns,
		"sign_in_date":        ntime,
	})
	u.Coin += coin + bonus

	nextBonus := 0
	daysUntilNextBonus := 0
	switch {
	case u.ContinuousSignIns == 9:
		nextBonus = 50
		daysUntilNextBonus = 1
	case u.ContinuousSignIns == 19:
		nextBonus = 150
		daysUntilNextBonus = 1
	case u.ContinuousSignIns == 29:
		nextBonus = 300
		daysUntilNextBonus = 1
	case u.ContinuousSignIns < 9:
		nextBonus = 50
		daysUntilNextBonus = 10 - u.ContinuousSignIns
	case u.ContinuousSignIns < 19:
		nextBonus = 150
		daysUntilNextBonus = 20 - u.ContinuousSignIns
	case u.ContinuousSignIns < 29:
		nextBonus = 300
		daysUntilNextBonus = 30 - u.ContinuousSignIns
	}

	return fmt.Sprintf("今日打卡人数：%d人\n奖励积分：%d个\n您已连续打卡：%d天\n积分余额总数：%d个\n额外奖励：继续连续打卡%d天，可获得%d积分奖励。", total[0]+1, coin, u.ContinuousSignIns, u.Coin, daysUntilNextBonus, nextBonus), nil
}

func portalPray(userNumber int) (string, error) {
	today := time.Now().Format("2006-01-02")
	if hasPrayedToday(userNumber) {
		return "你今天已经祈福过了，明天再来吧。", nil
	}
	if err := db.Create(&PortalPrayRecord{UserNumber: userNumber, PrayDate: today, CreatedAt: time.Now()}).Error; err != nil {
		return "你今天已经祈福过了，明天再来吧。", nil
	}

	if time.Now().Unix()%2 == 0 {
		return "祈福诚意不足，祈福失败，不增加积分。", nil
	}
	if db.Model(User{}).Where("number = ?", userNumber).Update("coin", gorm.Expr("coin + 3")).RowsAffected == 0 {
		return "先去打卡吧你。", nil
	}
	return "祈福成功，愿你事事顺心如意，积分 + 3。", nil
}

func formatPortalTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("2006-01-02 15:04:05")
}
