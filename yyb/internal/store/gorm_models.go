package store

// GormWechatAccount 应用宝微信账号（存 xdd 主库）
type GormWechatAccount struct {
	ID            int64   `gorm:"primaryKey"`
	OpenID        string  `gorm:"column:open_id;size:128;uniqueIndex;not null"`
	UIN           *int64  `gorm:"index"`
	Alias         *string `gorm:"size:256"`
	Nickname      *string `gorm:"size:256"`
	Avatar        *string `gorm:"size:512"`
	UserInfo      string  `gorm:"type:longtext"`
	LoginBuffer   string  `gorm:"type:longtext;not null"`
	Credentials   string  `gorm:"type:longtext"`
	Status        *string `gorm:"size:32"`
	LastCheckedAt *int64
	CreatedAt     int64 `gorm:"not null"`
	UpdatedAt     int64 `gorm:"not null"`
}

func (GormWechatAccount) TableName() string { return "yyb_wechat_accounts" }

// GormSession WMPF 会话缓存
type GormSession struct {
	ID              int64  `gorm:"primaryKey"`
	WechatAccountID int64  `gorm:"not null;uniqueIndex:idx_yyb_sess_acc_proxy"`
	UIN             *int64
	TCPProxy        string `gorm:"size:256;not null;default:'';uniqueIndex:idx_yyb_sess_acc_proxy"`
	SessionBlob     string `gorm:"type:longtext;not null"`
	ExpiresAt       int64  `gorm:"not null;index"`
	CreatedAt       int64  `gorm:"not null"`
	UpdatedAt       int64  `gorm:"not null"`
}

func (GormSession) TableName() string { return "yyb_sessions" }

// GormFeature 能力定义
type GormFeature struct {
	Code        int     `gorm:"primaryKey"`
	Name        string  `gorm:"size:64;uniqueIndex;not null"`
	Description *string `gorm:"size:256"`
	Enabled     bool    `gorm:"not null;default:true"`
}

func (GormFeature) TableName() string { return "yyb_features" }
