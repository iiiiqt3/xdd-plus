package srv

import (
	"sync"
	"wechatdll/comm"
)

// WXAccount 代表微信帐号
type WXAccount struct {
	UserInfo *comm.LoginData
	Lock     sync.RWMutex
}

// NewWXAccount 生成一个新的账户
func NewWXAccount(D *comm.LoginData) *WXAccount {
	wxAccount := &WXAccount{
		UserInfo: D,
	}
	return wxAccount
}

// GetUserInfo 获取UserInfo
func (wx *WXAccount) GetUserInfo() *comm.LoginData {
	wx.Lock.RLock()
	defer wx.Lock.RUnlock()
	return wx.UserInfo
}

// SetUserInfo 设置用户信息
func (wxAccount *WXAccount) SetUserInfo(info *comm.LoginData) {
	wxAccount.UserInfo = info
}
