package models

import (
	"fmt"
	"strings"
)

// Wechat08SendGroupMsg 群发文字（不 @）
func Wechat08SendGroupMsg(gid, msg string) {
	msg = strings.ReplaceAll(msg, "\r", "\n")
	_, err := wechat08CallType("Q0001", map[string]string{
		"wxid": gid,
		"msg":  msg,
	})
	if err != nil {
		Error("[wechat08] 群发失败:", err)
	}
}

// Wechat08IsEnabled 当前是否启用 wechat08 通道
func Wechat08IsEnabled() bool {
	return strings.EqualFold(strings.TrimSpace(Config.Wx.Model), "wechat08")
}

// Wechat08Describe 调试描述
func Wechat08Describe() string {
	return fmt.Sprintf("model=%s url=%s robot=%s tokenSet=%v",
		Config.Wx.Model, Config.Wx.Url, Config.Wx.Robotid, Config.Wx.Token != "")
}
