package yyb

// AccountPublic 对外账号视图
type AccountPublic struct {
	ID            int64   `json:"id"`
	OpenID        string  `json:"openid"`
	UIN           *int64  `json:"uin,omitempty"`
	Alias         *string `json:"alias,omitempty"`
	Nickname      *string `json:"nickname,omitempty"`
	Avatar        *string `json:"avatar,omitempty"`
	Status        *string `json:"status,omitempty"`
	LastCheckedAt *int64  `json:"last_checked_at,omitempty"`
	CreatedAt     int64   `json:"created_at"`
	UpdatedAt     int64   `json:"updated_at"`
}

// QRCreateResult 扫码会话
type QRCreateResult struct {
	SessionID string `json:"session_id"`
	Status    string `json:"status"`
	ImageURL  string `json:"image_url"`
	ImageB64  string `json:"image_base64,omitempty"`
}
