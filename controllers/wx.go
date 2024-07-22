package controllers

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	browser "github.com/EDDYCJY/fake-useragent"
	"github.com/beego/beego/v2/client/httplib"
	"github.com/beego/beego/v2/core/logs"
	"github.com/buger/jsonparser"
	"github.com/cdle/xdd/models"
	"os"
	"strconv"
	"strings"
)

type WxController struct {
	BaseController
}

type MyFriendVerifyMsg struct {
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

type MyAgreeFriend struct {
	API       string `json:"api"`        // API名
	RobotWxid string `json:"robot_wxid"` // 机器人ID
	Token     string `json:"token"`      // 验证密钥
	Type      int    `json:"type"`       // 收到好友验证消息中（json）的type属性
	V1        string `json:"v1"`         // 收到好友验证消息中（json）的v1属性
	V2        string `json:"v2"`         // 收到好友验证消息中（json）的v2属性

}

type AutocollectMessageBody struct {
	PayerPayId    string `json:"payer_pay_id"`
	ReceiverPayId string `json:"receiver_pay_id"`
	Paysubtype    int    `json:"paysubtype"`
	Money         string `json:"money"`
	PayMemo       string `json:"pay_memo"`
}

func (c *WxController) HandleWxMessage() {
	data := c.Ctx.Input.RequestBody
	logs.Info(string(data))
	event, _ := jsonparser.GetString(data, "Event")
	switch event {
	case "EventFrieneVerify":
		ag := &MyFriendVerifyMsg{}
		err := json.Unmarshal(data, ag)
		logs.Info(err)
		auto := models.IsAutoAgreeFriendVerify()
		if auto {
			if models.UseAgreeMsg() {
				AgreeMsg := models.GetEnv("AgreeMsg")
				if !strings.Contains(ag.Content.JSONMsg.Content, AgreeMsg) {
					return
				}
			}
			args := make(map[string]string)
			args["model"] = "my"
			args["type"] = strconv.Itoa(ag.Content.Type)
			args["v1"] = ag.Content.V1
			args["v2"] = ag.Content.V2
			args["content"] = ag.Content.JSONMsg.Content
			args["uid"] = ag.Content.FromWxid
			AgreeFriendVerify(args)
		}

	case "EventPrivateChat":
		ag := &WxMessage{}
		json.Unmarshal(data, ag)
		switch ag.Content.Type {
		case 1:
			if ag.Content.RobotWxid == models.Config.Wx.Robotid {
				//logs.Info(err)
				logs.Info("接收到信息" + ag.Content.Msg)
				models.ListenWXTempPrivateMessage(ag.Content.FromWxid, ag.Content.Msg)
			}
		case 2000:
			if ag.Content.RobotWxid == models.Config.Wx.Robotid {
				if models.ElmList[models.GetWxid(ag.Content.FromWxid)] != nil {
					logs.Info("进入饿充值队列")
					autocollect := &AutocollectMessageBody{}
					err := json.Unmarshal([]byte(ag.Content.Msg), autocollect)
					if err == nil && autocollect.PayerPayId != "" && autocollect.ReceiverPayId != "" && autocollect.Paysubtype == 1 {
						receiveMoney(autocollect, ag, 1)
						switch autocollect.Money {
						case "1.00":
							switchMoney(ag, "1.txt")
							return
						case "5.00":
							switchMoney(ag, "5.txt")
							return
						case "10.00":
							switchMoney(ag, "10.txt")
						default:
							//不予处理
						}
						return
					}
				}
				logs.Info("接收到转账" + ag.Content.Msg)
				if models.IsAutoAgreeAutocollection() {
					autocollect := &AutocollectMessageBody{}
					err := json.Unmarshal([]byte(ag.Content.Msg), autocollect)
					if err == nil && autocollect.PayerPayId != "" && autocollect.ReceiverPayId != "" && autocollect.Paysubtype == 1 {
						receiveMoney(autocollect, ag, 0)
					}
				}
			}
		}

	case "EventGroupChat":
		ag := &WxMessage{}
		json.Unmarshal(data, ag)
		if ag.Content.RobotWxid == models.Config.Wx.Robotid {
			logs.Info("接收到微信群信息" + ag.Content.Msg)
			models.ListenWXGroupMessage(ag.Content.FromWxid, ag.Content.FromGroup, ag.Content.Msg)

		}
	}
}

func switchMoney(ag *WxMessage, filePath string) {
	line, err := handleElm(filePath)
	if err != nil {
		logs.Error(err)
		models.JdCookie{}.Push(fmt.Sprintf("卡密为空请及时维护:%s", filePath))
		models.SendWxMsg(ag.Content.FromWxid, "卡密库存不足,请联系管理员处理")
		return
	}
	models.SendWxMsg(ag.Content.FromWxid, line)
	models.ElmList[models.GetWxid(ag.Content.FromWxid)] = nil
}

func receiveMoney(autocollect *AutocollectMessageBody, ag *WxMessage, typ int) {
	args := make(map[string]string)
	args["money"] = autocollect.Money
	args["payer_pay_id"] = autocollect.PayerPayId
	args["receiver_pay_id"] = autocollect.ReceiverPayId
	args["paysubtype"] = strconv.Itoa(autocollect.Paysubtype)
	args["to_wxid"] = ag.Content.FromWxid
	if typ == 0 {
		models.AutoCollectionAndAddCoin(args)
	} else if typ == 1 {
		models.AutoCollection(args)
	}
}

func AgreeFriendVerify(args interface{}) {
	arg := args.(map[string]string)
	req := httplib.Post(models.Config.Wx.Url)
	type1, _ := strconv.Atoi(arg["type"])
	agree := &MyAgreeFriend{
		Token:     models.Config.Wx.Token,
		API:       "AgreeFriendVerify",
		RobotWxid: models.Config.Wx.Robotid,
		Type:      type1,
		V1:        arg["v1"],
		V2:        arg["v2"],
	}
	random := browser.Random()
	req.Header("User-Agent", random)
	marshal, _ := json.Marshal(agree)
	req.Body(string(marshal))
	s, _ := req.Bytes()
	val, _ := jsonparser.GetString(s, "Result")
	if val == "OK" {
		welcome := models.GetEnv("Welcome")
		if welcome != "" {
			models.SendWxMsg(arg["uid"], welcome)
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

func handleElm(filePath string) (firstLine string, err error) {
	// 打开文件进行读取
	file, err := os.Open(models.ExecPath + "/elm/" + filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	// 读取第一行
	if scanner.Scan() {
		firstLine = scanner.Text()
	} else {
		return "", errors.New("文件为空")
	}

	// 读取文件的其余部分
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	// 库存提醒
	if len(lines) <= 3 {
		// Perform the special reminder action here
		models.JdCookie{}.Push(fmt.Sprintf("卡密库存不足，请及时补充:%s", filePath))
	}

	// 关闭文件后重新打开进行写入
	file.Close()
	err = os.WriteFile(filePath, []byte(strings.Join(lines, "\n")), 0644)
	if err != nil {
		return "", err
	}

	// 返回第一行
	return firstLine, nil
}
