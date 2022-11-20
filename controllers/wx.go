package controllers

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	browser "github.com/EDDYCJY/fake-useragent"
	"github.com/beego/beego/v2/client/httplib"
	"github.com/beego/beego/v2/core/logs"
	"github.com/buger/jsonparser"
	"github.com/cdle/xdd/models"
	"strconv"
	"strings"
)

type WxController struct {
	BaseController
}

type QXMessage struct {
	Event int    `json:"event"`
	Wxid  string `json:"wxid"`
	Data  struct {
		Type string `json:"type"`
		Des  string `json:"des"`
		Data struct {
			TimeStamp     string        `json:"timeStamp"`
			FromType      int           `json:"fromType"`
			MsgType       int           `json:"msgType"`
			MsgSource     int           `json:"msgSource"`
			FromWxid      string        `json:"fromWxid"`
			FinalFromWxid string        `json:"finalFromWxid"`
			AtWxidList    []interface{} `json:"atWxidList"`
			Silence       int           `json:"silence"`
			Membercount   int           `json:"membercount"`
			Signature     string        `json:"signature"`
			Msg           string        `json:"msg"`
			MsgBase64     string        `json:"msgBase64"`
		} `json:"data"`
		Timestamp string `json:"timestamp"`
		Wxid      string `json:"wxid"`
		Port      int    `json:"port"`
		Pid       int    `json:"pid"`
		Flag      string `json:"flag"`
	} `json:"data"`
}

type FriendVerifyMsg struct {
	SdkVer  int    `json:"sdkVer"`
	Event   string `json:"Event"`
	Content struct {
		RobotWxid string `json:"robot_wxid"`
		Type      int    `json:"type"`
		FromWxid  string `json:"from_wxid"`
		FromName  string `json:"from_name"`
		V1        string `json:"v1"`
		V2        string `json:"v2"`
		JSONMsg   struct {
			Scene         int    `json:"scene"`
			Headimgurl    string `json:"headimgurl"`
			FromContent   string `json:"from_content"`
			FromGroupWxid string `json:"from_group_wxid"`
			ShareWxid     string `json:"share_wxid"`
			ShareNickname string `json:"share_nickname"`
			V1            string `json:"v1"`
			V2            string `json:"v2"`
			Sex           int    `json:"sex"`
			Content       string `json:"content"`
		} `json:"json_msg"`
		RobotType int `json:"robot_type"`
	} `json:"content"`
}

type WxMessage struct {
	SdkVer  int    `json:"sdkVer"`
	Event   string `json:"Event"`
	Content struct {
		RobotWxid     string `json:"robot_wxid"`
		Type          int    `json:"type"`
		FromGroup     string `json:"from_group"`
		FromGroupName string `json:"from_group_name"`
		FromWxid      string `json:"from_wxid"`
		FromName      string `json:"from_name"`
		Msg           string `json:"msg"`
		Clientid      int    `json:"clientid"`
		RobotType     int    `json:"robot_type"`
	} `json:"content"`
}

type AgreeFriend struct {
	API       string `json:"api"`        // API名
	RobotWxid string `json:"robot_wxid"` // 机器人ID
	Token     string `json:"token"`      // 验证密钥
	Type      int    `json:"type"`       // 收到好友验证消息中（json）的type属性
	V1        string `json:"v1"`         // 收到好友验证消息中（json）的v1属性
	V2        string `json:"v2"`         // 收到好友验证消息中（json）的v2属性

}

func (c *WxController) HandleMessage() {
	data := c.Ctx.Input.RequestBody
	logs.Info(string(data))
	event, err := jsonparser.GetString(data, "Event")
	if err != nil {
		logs.Info("进入测试")
		ev, _ := jsonparser.GetInt(data, "event")
		logs.Info(ev)
		event = strconv.FormatInt(ev, 10)
	}
	logs.Info(event)
	switch event {
	case "EventFrieneVerify":
		ag := &FriendVerifyMsg{}
		err := json.Unmarshal(data, ag)
		logs.Info(err)
		auto := models.IsAutoAgreeFriendVerify()
		if auto {
			AgreeFriendVerify(ag.Content.Type, ag.Content.V1, ag.Content.V2, ag.Content.JSONMsg.Content, ag.Content.FromWxid)
		}

	case "EventPrivateChat":
		ag := &WxMessage{}
		err := json.Unmarshal(data, ag)
		logs.Info(err)
		logs.Info("接收到信息" + ag.Content.Msg)
		models.ListenWXTempPrivateMessage(ag.Content.FromWxid, ag.Content.Msg)

	case "10009":
		ag := &QXMessage{}
		err := json.Unmarshal(data, ag)
		logs.Info(err)
		logs.Info("接收到信息" + ag.Data.Data.Msg)
		models.ListenWXTempPrivateMessage(ag.Data.Data.FromWxid, ag.Data.Data.Msg)

	}

}

func AgreeFriendVerify(type1 int, v1 string, v2 string, content string, uid string) {

	req := httplib.Post(models.Config.Wx.Url)
	agree := &AgreeFriend{
		Token:     "1",
		API:       "AgreeFriendVerify",
		RobotWxid: models.Config.Wx.Robotid,
		Type:      type1,
		V1:        v1,
		V2:        v2,
	}
	random := browser.Random()
	req.Header("User-Agent", random)
	marshal, _ := json.Marshal(agree)
	logs.Info(string(marshal))

	req.Body(string(marshal))
	s, _ := req.Bytes()
	val, _ := jsonparser.GetString(s, "Result")
	if val == "OK" {
		welcome := models.GetEnv("Welcome")
		if welcome != "" {
			models.SendWxMsg(uid, welcome)
		}
	}
}

func u2s(form string) (to string, err error) {
	bs, err := hex.DecodeString(strings.Replace(form, `\u`, ``, -1))
	if err != nil {
		return
	}
	for i, bl, br, r := 0, len(bs), bytes.NewReader(bs), uint16(0); i < bl; i += 2 {
		binary.Read(br, binary.BigEndian, &r)
		to += string(r)
	}
	return
}
