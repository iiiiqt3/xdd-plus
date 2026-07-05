package store

// 所有字段显式 column 标签，避免 GORM 将 UIN/OpenID 等映射成 ui_n、open_id 与手写 SQL 不一致。
// 新增字段时务必带 column 标签；查询优先用结构体 Where/Updates，不要手写列名。

// GormWechatAccount 应用宝微信账号（存 xdd 主库）
type GormWechatAccount struct {
	ID            int64   `gorm:"column:id;primaryKey"`
	OpenID        string  `gorm:"column:open_id;size:128;uniqueIndex;not null"`
	UIN           *int64  `gorm:"column:uin;index"`
	Alias         *string `gorm:"column:alias;size:256"`
	Nickname      *string `gorm:"column:nickname;size:256"`
	Avatar        *string `gorm:"column:avatar;size:512"`
	UserInfo      string  `gorm:"column:user_info;type:longtext"`
	LoginBuffer   string  `gorm:"column:login_buffer;type:longtext;not null"`
	Credentials   string  `gorm:"column:credentials;type:longtext"`
	Status        *string `gorm:"column:status;size:32"`
	LastCheckedAt *int64  `gorm:"column:last_checked_at"`
	CreatedAt     int64   `gorm:"column:created_at;not null"`
	UpdatedAt     int64   `gorm:"column:updated_at;not null"`
}

func (GormWechatAccount) TableName() string { return "yyb_wechat_accounts" }

// GormSession WMPF 会话缓存
type GormSession struct {
	ID              int64  `gorm:"column:id;primaryKey"`
	WechatAccountID int64  `gorm:"column:wechat_account_id;not null;uniqueIndex:idx_yyb_sess_acc_proxy"`
	UIN             *int64 `gorm:"column:uin"`
	TCPProxy        string `gorm:"column:tcp_proxy;size:256;not null;default:'';uniqueIndex:idx_yyb_sess_acc_proxy"`
	SessionBlob     string `gorm:"column:session_blob;type:longtext;not null"`
	ExpiresAt       int64  `gorm:"column:expires_at;not null;index"`
	CreatedAt       int64  `gorm:"column:created_at;not null"`
	UpdatedAt       int64  `gorm:"column:updated_at;not null"`
}

func (GormSession) TableName() string { return "yyb_sessions" }

// GormFeature 能力定义
type GormFeature struct {
	Code        int     `gorm:"column:code;primaryKey"`
	Name        string  `gorm:"column:name;size:64;uniqueIndex;not null"`
	Description *string `gorm:"column:description;size:256"`
	Enabled     bool    `gorm:"column:enabled;not null;default:true"`
}

func (GormFeature) TableName() string { return "yyb_features" }
