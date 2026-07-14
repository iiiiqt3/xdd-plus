package models

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
)

var jdProxyPurchaseDetailRe = regexp.MustCompile(`京东任务代理\s*(\d+)\s*个月.*?到期\s*([0-9:\-\s]+)`)

// PortalJdProxySubscription 网页端京东任务代理订阅（按用户）
type PortalJdProxySubscription struct {
	ID         int       `gorm:"primaryKey;autoIncrement"`
	UserNumber int       `gorm:"uniqueIndex;not null"`
	ExpireAt   time.Time `gorm:"index;not null"`
	UpdatedAt  time.Time
}

func (PortalJdProxySubscription) TableName() string {
	return "portal_jd_proxy_subscription"
}

// GetJdTaskProxyMonthlyCoin 每月代理订阅积分价格
func GetJdTaskProxyMonthlyCoin() int {
	if sysConfig.JdTaskProxyMonthlyCoin < 0 {
		return 0
	}
	return sysConfig.JdTaskProxyMonthlyCoin
}

// IsPortalJdProxyActive 用户是否拥有有效的京东任务代理订阅
func IsPortalJdProxyActive(userNumber int) bool {
	if userNumber <= 0 {
		return false
	}
	var sub PortalJdProxySubscription
	if err := db.Where("user_number = ?", userNumber).First(&sub).Error; err != nil {
		return false
	}
	return sub.ExpireAt.After(time.Now())
}

// PortalJdProxyStatus 门户展示用代理订阅状态
type PortalJdProxyStatus struct {
	Active      bool   `json:"active"`
	ExpireAt    string `json:"expireAt"`
	MonthlyCoin int    `json:"monthlyCoin"`
	UserCoin    int    `json:"userCoin"`
	ProxyReady  bool   `json:"proxyReady"`
}

// ResolvePortalJdTaskProxy 门户订阅用户：手动代理开关开 → 用手动配置；关 → 用系统设置代理
func ResolvePortalJdTaskProxy() (enabled bool, url, renum, redelay string) {
	if IsJdManualProxySwitchEnabled() {
		return resolveManualOnlyJdTaskProxy()
	}
	return ResolveSystemJdTaskProxy()
}

// ResolvePortalJdTaskProxyReady 门户是否可购买/使用代理（按当前代理来源判断配置是否齐全）
func ResolvePortalJdTaskProxyReady() bool {
	ok, url, _, _ := ResolvePortalJdTaskProxy()
	return ok && url != ""
}

// GetPortalJdProxyStatus 查询用户代理订阅状态
func GetPortalJdProxyStatus(userNumber int) PortalJdProxyStatus {
	st := PortalJdProxyStatus{
		MonthlyCoin: GetJdTaskProxyMonthlyCoin(),
		UserCoin:    GetCoin(userNumber),
		ProxyReady:  ResolvePortalJdTaskProxyReady(),
	}
	var sub PortalJdProxySubscription
	if err := db.Where("user_number = ?", userNumber).First(&sub).Error; err == nil {
		if sub.ExpireAt.After(time.Now()) {
			st.Active = true
			st.ExpireAt = sub.ExpireAt.Format("2006-01-02 15:04:05")
		}
	}
	return st
}

// ResolveManualJdTaskProxyReady 后台手动任务是否已配置可用代理（含回退系统代理）
func ResolveManualJdTaskProxyReady() bool {
	ok, url, _, _ := ResolveManualJdTaskProxy()
	return ok && url != ""
}

// PurchasePortalJdProxy 使用积分购买/续费京东任务代理
func PurchasePortalJdProxy(userNumber int, months int, clientCtx ClientContext) (PortalJdProxyStatus, error) {
	if months < 1 || months > 36 {
		return PortalJdProxyStatus{}, fmt.Errorf("购买月数需在 1-36 之间")
	}
	monthly := GetJdTaskProxyMonthlyCoin()
	if monthly <= 0 {
		return PortalJdProxyStatus{}, fmt.Errorf("代理订阅暂未开放，请联系管理员")
	}
	if !ResolvePortalJdTaskProxyReady() {
		return PortalJdProxyStatus{}, fmt.Errorf("管理员尚未配置任务代理，暂不可购买")
	}
	total := monthly * months
	if err := DeductCoinChecked(userNumber, total); err != nil {
		return PortalJdProxyStatus{}, err
	}

	now := time.Now()
	var sub PortalJdProxySubscription
	err := db.Where("user_number = ?", userNumber).First(&sub).Error
	base := now
	if err == nil && sub.ExpireAt.After(now) {
		base = sub.ExpireAt
	}
	newExpire := base.AddDate(0, months, 0)

	if err != nil {
		sub = PortalJdProxySubscription{
			UserNumber: userNumber,
			ExpireAt:   newExpire,
			UpdatedAt:  now,
		}
		if createErr := db.Create(&sub).Error; createErr != nil {
			db.Model(&User{}).Where("number = ?", userNumber).Update("coin", gorm.Expr(fmt.Sprintf("coin+%d", total)))
			return PortalJdProxyStatus{}, createErr
		}
	} else {
		if updateErr := db.Model(&sub).Updates(map[string]interface{}{
			"expire_at":  newExpire,
			"updated_at": now,
		}).Error; updateErr != nil {
			db.Model(&User{}).Where("number = ?", userNumber).Update("coin", gorm.Expr(fmt.Sprintf("coin+%d", total)))
			return PortalJdProxyStatus{}, updateErr
		}
	}

	detail := fmt.Sprintf("京东任务代理 %d 个月，到期 %s", months, newExpire.Format("2006-01-02"))
	RecordCoinLogEx(userNumber, -total, "代理订阅", detail, clientCtx)
	return GetPortalJdProxyStatus(userNumber), nil
}

// ApplyPortalJdTaskProxyEnvs 门户任务：订阅有效时按手动/系统代理来源注入
func ApplyPortalJdTaskProxyEnvs(userNumber int, envs map[string]string) {
	if envs == nil || !IsPortalJdProxyActive(userNumber) {
		return
	}
	ok, url, renum, redelay := ResolvePortalJdTaskProxy()
	if !ok || url == "" {
		return
	}
	applyJdProxyEnvsFromConfig(envs, url, renum, redelay)
}

// AdminJdProxyPurchaseStats 门户任务代理购买汇总
type AdminJdProxyPurchaseStats struct {
	TotalPurchases int   `json:"totalPurchases"`
	TotalCoin      int   `json:"totalCoin"`
	UniqueUsers    int   `json:"uniqueUsers"`
	ActiveCount    int   `json:"activeCount"`
}

// AdminJdProxyPurchaseItem 单次代理购买记录（来自 coin_log）
type AdminJdProxyPurchaseItem struct {
	ID              int    `json:"id"`
	UserNumber      int    `json:"userNumber"`
	Username        string `json:"username"`
	Nickname        string `json:"nickname"`
	QQ              string `json:"qq"`
	Coin            int    `json:"coin"`
	Months          int    `json:"months"`
	ExpireAt        string `json:"expireAt"`
	CurrentExpireAt string `json:"currentExpireAt"`
	CurrentlyActive bool   `json:"currentlyActive"`
	ClientSource    string `json:"clientSource"`
	ClientPlatform  string `json:"clientPlatform"`
	SourceLabel     string `json:"sourceLabel"`
	Detail          string `json:"detail"`
	PurchasedAt     string `json:"purchasedAt"`
}

func parseJdProxyPurchaseDetail(detail string) (months int, expireAt string) {
	m := jdProxyPurchaseDetailRe.FindStringSubmatch(strings.TrimSpace(detail))
	if len(m) < 3 {
		return 0, ""
	}
	months, _ = strconv.Atoi(m[1])
	expireAt = strings.TrimSpace(m[2])
	if len(expireAt) > 10 {
		expireAt = expireAt[:10]
	}
	return months, expireAt
}

func buildAdminJdProxyPurchaseQuery(userNumber int, days int) *gorm.DB {
	query := db.Model(&CoinLog{}).Where("type = ?", "代理订阅")
	if userNumber > 0 {
		query = query.Where("user_number = ?", userNumber)
	}
	if days > 0 {
		since := time.Now().AddDate(0, 0, -days)
		query = query.Where("created_at >= ?", since)
	}
	return query
}

// GetAdminJdProxyPurchaseStats 代理购买汇总（可按用户/天数筛选）
func GetAdminJdProxyPurchaseStats(userNumber int, days int) AdminJdProxyPurchaseStats {
	stats := AdminJdProxyPurchaseStats{}
	base := buildAdminJdProxyPurchaseQuery(userNumber, days)
	var total int64
	base.Count(&total)
	stats.TotalPurchases = int(total)

	var sumCoin int64
	buildAdminJdProxyPurchaseQuery(userNumber, days).Select("COALESCE(SUM(ABS(amount)),0)").Scan(&sumCoin)
	stats.TotalCoin = int(sumCoin)

	var unique int64
	buildAdminJdProxyPurchaseQuery(userNumber, days).Distinct("user_number").Count(&unique)
	stats.UniqueUsers = int(unique)

	now := time.Now()
	subQuery := db.Model(&PortalJdProxySubscription{}).Where("expire_at > ?", now)
	if userNumber > 0 {
		subQuery = subQuery.Where("user_number = ?", userNumber)
	}
	var active int64
	subQuery.Count(&active)
	stats.ActiveCount = int(active)
	return stats
}

// GetAdminJdProxyPurchases 代理购买记录列表
func GetAdminJdProxyPurchases(userNumber int, days int, page int, limit int) ([]AdminJdProxyPurchaseItem, int64, AdminJdProxyPurchaseStats) {
	if page < 1 {
		page = 1
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	stats := GetAdminJdProxyPurchaseStats(userNumber, days)

	var total int64
	buildAdminJdProxyPurchaseQuery(userNumber, days).Count(&total)
	if total == 0 {
		return []AdminJdProxyPurchaseItem{}, 0, stats
	}

	var logs []CoinLog
	buildAdminJdProxyPurchaseQuery(userNumber, days).
		Order("id desc").
		Offset((page - 1) * limit).
		Limit(limit).
		Find(&logs)

	userNumbers := make([]int, 0, len(logs))
	seen := map[int]bool{}
	for _, log := range logs {
		if log.UserNumber > 0 && !seen[log.UserNumber] {
			seen[log.UserNumber] = true
			userNumbers = append(userNumbers, log.UserNumber)
		}
	}

	usernameMap := map[int]string{}
	if len(userNumbers) > 0 {
		var accounts []WebUserAccount
		db.Where("user_number IN ?", userNumbers).Find(&accounts)
		for _, acc := range accounts {
			usernameMap[acc.UserNumber] = acc.Username
		}
	}

	userMap := map[int]*User{}
	if len(userNumbers) > 0 {
		var users []User
		db.Where("number IN ?", userNumbers).Find(&users)
		for i := range users {
			userMap[users[i].Number] = &users[i]
		}
	}

	subMap := map[int]PortalJdProxySubscription{}
	if len(userNumbers) > 0 {
		var subs []PortalJdProxySubscription
		db.Where("user_number IN ?", userNumbers).Find(&subs)
		for _, sub := range subs {
			subMap[sub.UserNumber] = sub
		}
	}

	now := time.Now()
	items := make([]AdminJdProxyPurchaseItem, 0, len(logs))
	for _, log := range logs {
		ctx, detail := ResolveCoinLogContext(log)
		months, expireAt := parseJdProxyPurchaseDetail(detail)
		item := AdminJdProxyPurchaseItem{
			ID:             log.ID,
			UserNumber:     log.UserNumber,
			Username:       usernameMap[log.UserNumber],
			Coin:           -log.Amount,
			Months:         months,
			ExpireAt:       expireAt,
			Detail:         detail,
			ClientSource:   ctx.Source,
			ClientPlatform: ctx.Platform,
			SourceLabel:    ctx.AdminLabel(),
			PurchasedAt:    log.CreatedAt.Format("2006-01-02 15:04:05"),
		}
		if u := userMap[log.UserNumber]; u != nil {
			item.Nickname = u.Nickname
			item.QQ = u.QQ
		}
		if sub, ok := subMap[log.UserNumber]; ok {
			item.CurrentExpireAt = sub.ExpireAt.Format("2006-01-02 15:04:05")
			item.CurrentlyActive = sub.ExpireAt.After(now)
		}
		items = append(items, item)
	}
	return items, total, stats
}
