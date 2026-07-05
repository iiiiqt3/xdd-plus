package yybportal

import (
	"time"

	"gorm.io/gorm"
)

// PortalYybBinding portal 用户与应用宝账号绑定
type PortalYybBinding struct {
	ID           int64          `gorm:"primaryKey" json:"id"`
	UserNumber   int            `gorm:"index;not null;uniqueIndex:idx_portal_yyb_user_openid" json:"userNumber"`
	YybAccountID int64          `gorm:"index;not null" json:"yybAccountId"`
	OpenID       string         `gorm:"size:128;not null;uniqueIndex:idx_portal_yyb_user_openid" json:"openid"`
	Nickname     string         `gorm:"size:128" json:"nickname"`
	Status       string         `gorm:"size:32" json:"status"`
	CreatedAt    time.Time      `json:"createdAt"`
	UpdatedAt    time.Time      `json:"updatedAt"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

func (PortalYybBinding) TableName() string { return "portal_yyb_bindings" }

// PortalAccountView 门户展示
type PortalAccountView struct {
	BindingID    int64  `json:"bindingId"`
	YybAccountID int64  `json:"yybAccountId"`
	OpenID       string `json:"openid"`
	UIN          *int64 `json:"uin,omitempty"`
	Nickname     string `json:"nickname"`
	AvatarURL    string `json:"avatarUrl"`
	Status       string `json:"status"`
	LastChecked  *int64 `json:"lastCheckedAt,omitempty"`
	CreatedAt    int64  `json:"createdAt"`
}

// AdminAccountView 管理后台展示
type AdminAccountView struct {
	PortalAccountView
	UserNumber int    `json:"userNumber"`
	UserNick   string `json:"userNickname"`
}
