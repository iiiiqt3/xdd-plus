package yybportal

import (
	"time"

	"gorm.io/gorm"
)

// PortalYybBinding portal 用户与应用宝账号绑定（列名均显式声明，避免 GORM 命名歧义）
type PortalYybBinding struct {
	ID           int64          `gorm:"column:id;primaryKey" json:"id"`
	UserNumber   int            `gorm:"column:user_number;index;not null;uniqueIndex:idx_portal_yyb_user_openid" json:"userNumber"`
	YybAccountID int64          `gorm:"column:yyb_account_id;index;not null" json:"yybAccountId"`
	OpenID       string         `gorm:"column:open_id;size:128;not null;uniqueIndex:idx_portal_yyb_user_openid" json:"openid"`
	Nickname     string         `gorm:"column:nickname;size:128" json:"nickname"`
	Remark       string         `gorm:"column:remark;size:128" json:"remark"`
	Status       string         `gorm:"column:status;size:32" json:"status"`
	LoginAt      time.Time      `gorm:"column:login_at" json:"loginAt"`
	CreatedAt    time.Time      `gorm:"column:created_at" json:"createdAt"`
	UpdatedAt    time.Time      `gorm:"column:updated_at" json:"updatedAt"`
	DeletedAt    gorm.DeletedAt `gorm:"column:deleted_at;index" json:"-"`
}

func (PortalYybBinding) TableName() string { return "portal_yyb_bindings" }

// PortalAccountView 门户展示
type PortalAccountView struct {
	BindingID    int64  `json:"bindingId"`
	YybAccountID int64  `json:"yybAccountId"`
	OpenID       string `json:"openid"`
	UIN          *int64 `json:"uin,omitempty"`
	Nickname     string `json:"nickname"`
	Remark       string `json:"remark"`
	AvatarURL    string `json:"avatarUrl"`
	Status       string `json:"status"`
	LoginAt      int64  `json:"loginAt"`
	ExpiresAt    int64  `json:"expiresAt"`
	LastChecked  *int64 `json:"lastCheckedAt,omitempty"`
	CreatedAt    int64  `json:"createdAt"`
	ProxyRegionCode string `json:"proxyRegionCode,omitempty"`
	ProxyRegionName string `json:"proxyRegionName,omitempty"`
}

// AdminAccountView 管理后台展示
type AdminAccountView struct {
	PortalAccountView
	UserNumber int    `json:"userNumber"`
	UserNick   string `json:"userNickname"`
}
