package models

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

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
