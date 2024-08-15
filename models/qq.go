package models

import (
	"encoding/json"
	"fmt"
	"github.com/beego/beego/v2/core/logs"
	"strconv"
	"strings"
)

type QQMessage struct {
	Action string `json:"action"`
	QQMsg  struct {
		MessageType string `json:"message_type"`
		UserId      int    `json:"user_id"`
		GroupID     int    `json:"group_id"`
		Message     string `json:"message"`
	} `json:"params"`
	Echo      string `json:"echo"`
	MessageId int    `json:"message_id"`
}

var SendQQ = func(qq int, msg interface{}) {

	switch msg.(type) {
	case string:
		SendQQMsg(QQMessage{
			Action: "send_msg",
			QQMsg: struct {
				MessageType string `json:"message_type"`
				UserId      int    `json:"user_id"`
				GroupID     int    `json:"group_id"`
				Message     string `json:"message"`
			}{
				UserId:  qq,
				Message: msg.(string),
			},
			Echo: "user_id",
		})
	}
}

var SendQQGroup = func(gid int, qq int, msg interface{}) {
	switch msg.(type) {
	case string:
		SendQQMsg(QQMessage{
			Action: "send_msg",
			QQMsg: struct {
				MessageType string `json:"message_type"`
				UserId      int    `json:"user_id"`
				GroupID     int    `json:"group_id"`
				Message     string `json:"message"`
			}{
				GroupID: gid,
				Message: fmt.Sprintf("[CQ:at,qq=%d]", qq) + msg.(string),
			},
			Echo: "user_id",
		})
	}
}

var ListenQQPrivateMessage = func(uid int, msg string) {
	SendQQ(uid, handleMessage(msg, "qq", uid))
}

var ListenQQGroupMessage = func(uid int, gid int, msg string) {
	if strings.Contains(Config.QQGroupID, strconv.Itoa(gid)) {
		if Config.QbotPublicMode {
			SendQQGroup(gid, uid, handleMessage(msg, "qqg", uid, gid))
		} else {
			SendQQ(uid, handleMessage(msg, "qq", uid))
		}
	}
}

func SendQQMsg(msg QQMessage) {
	marshal, _ := json.Marshal(msg)
	logs.Info(string(marshal))
	msgchan <- marshal

}

func DeleteQQMsg(msgid int) {
	marshal, _ := json.Marshal(struct {
		Action string `json:"action"`
		QQMsg  struct {
			MessageId int `json:"message_id"`
		} `json:"params"`
	}{
		Action: "delete_msg",
		QQMsg: struct {
			MessageId int `json:"message_id"`
		}{
			MessageId: msgid,
		},
	})
	logs.Info(string(marshal))
	msgchan <- marshal
}

func AutoAgreeFriedns(flag string) {
	marshal, _ := json.Marshal(struct {
		Action string `json:"action"`
		QQMsg  struct {
			Approve string `json:"approve"`
			Remark  string `json:"remark"`
			Flag    string `json:"flag"`
		} `json:"params"`
	}{
		Action: "set_friend_add_request",
		QQMsg: struct {
			Approve string `json:"approve"`
			Remark  string `json:"remark"`
			Flag    string `json:"flag"`
		}{
			Approve: "true",
			Remark:  "",
			Flag:    flag,
		},
	})
	logs.Info(string(marshal))
	msgchan <- marshal
}
