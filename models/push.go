package models

import (
	"encoding/json"
	"github.com/beego/beego/v2/client/httplib"
	"time"
	"strconv" // 用于字符串转数字
	"strings"  // 用于字符串判断 (修复 undefined: strings)
)

// Push 推送消息到用户，根据用户配置推送到QQ、微信、TG等渠道
// 推送渠道说明：0所有，1QQ，2微信，3TG，12QQ+微信，13QQ+TG，23微信+TG
func (ck JdCookie) Push(msg string) {
	if ck.PtPin != "" {
		go SendQQ(ck.QQ, msg)
		go pushPlus(ck.PushPlus, msg)
		go SendTgMsg(ck.Telegram, msg)
		if ck.WeiXin != "" {
			go SendWxMsg(ck.WeiXin, msg)
		}
	} else {
		value := GetEnv("AdminPush")
		if value == "" {
			value = "0"
		}
		
		if value == "0" || value == "1" || value == "12" || value == "13" {
			go SendQQ(Config.QQID, msg) // QQ通知
		}
		
		if value == "0" || value == "2" || value == "12" || value == "23" {
			WeiXin := getWeiXinId(Config.QQID)
			if WeiXin != "找不到对应的微信ID" {
				go SendWxMsg(WeiXin, msg)  // 微信通知
			}
		}
		
		if value == "0" || value == "3" || value == "13" || value == "23" {
			go SendTgMsg(Config.TelegramUserID, msg) // TG通知
		}
		
		if value == "0" || value == "4" {
			go qywxNotify(&QywxConfig{QywxKey: Config.QywxKey, Content: msg}) // 企业微信通知
		}
	}
}


func PushByQQ(qq string, msg string) {
	if qq == "" {
		return
	}

	// 定义局部结构体匹配 users 表字段
	// 修改点：将 gorm:"column:qq" 改为 gorm:"column:number"
	type UserRecord struct {
		QQ     string `gorm:"column:number"` // 这里映射到数据库的 number 字段
		WxID   string `gorm:"column:wxid"`
	}

	var user UserRecord

	// 查询 users 表
	// 修改点：Where 条件中的字段名也要改为 "number"
	err := db.Table("users").Where("number = ?", qq).First(&user).Error
	if err != nil {
		// 没找到或出错，直接返回
		return
	}

	// --- 类型转换 ---
	// 将字符串类型的 number (原逻辑中的 QQ) 转换为 int
	qqInt, err := strconv.Atoi(user.QQ)
	if err != nil {
		return
	}

	// 推送 QQ (现在传入的是 int 类型)
	go SendQQ(qqInt, msg)

	// 推送微信 (仅当 wxid 存在且非空)
	if user.WxID != "" && user.WxID != "[NULL]" {
		// 简单的格式校验
		if strings.HasPrefix(user.WxID, "wxid_") || len(user.WxID) > 5 {
			go SendWxMsg(user.WxID, msg)
		}
	}
}


func pushPlus(token string, content string) {
	if token == "" {
		return
	}
	data, _ := json.Marshal(struct {
		Token    string `json:"token"`
		Content  string `json:"content"`
		Template string `json:"template"`
	}{
		Token:    token,
		Content:  content,
		Template: "txt",
	})
	req := httplib.Post("http://pushplus.hxtrip.com/send")
	req.Header("Content-Type", "application/json")
	req.Body(data)
	req.Response()
	req = httplib.Post("http://www.pushplus.plus/send")
	req.Header("Content-Type", "application/json")
	req.Body(data)
	req.Response()
}

type QywxConfig struct {
	QywxKey string
	Content string
}

type QywxNotifyMessage struct {
	Msgtype string `json:"msgtype"`
	Text    struct {
		Content string `json:"content"`
	} `json:"text"`
}

func qywxNotify(c *QywxConfig) {
	if c.QywxKey == "" {
		return
	}
	wx := QywxNotifyMessage{
		Msgtype: "text",
	}
	wx.Text.Content = c.Content
	req := httplib.Post("https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=" + c.QywxKey)
	req.Header("Content-Type", "application/json")
	req, _ = req.JSONBody(wx)
	req.SetTimeout(time.Second*2, time.Second*2)
	req.Response()
}
