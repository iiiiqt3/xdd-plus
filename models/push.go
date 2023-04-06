package models

import (
	"encoding/json"
	"github.com/beego/beego/v2/client/httplib"
	"time"
)

func (ck JdCookie) Push(msg string) {
	if ck.PtPin != "" {
		go SendQQ(ck.QQ, msg)
		//go SendWxMsg()
		go pushPlus(ck.PushPlus, msg)
		go SendTgMsg(ck.Telegram, msg)
	} else {
		go SendQQ(Config.QQID, msg)
		go qywxNotify(&QywxConfig{QywxKey: Config.QywxKey, Content: msg})
		go SendTgMsg(Config.TelegramUserID, msg)
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
