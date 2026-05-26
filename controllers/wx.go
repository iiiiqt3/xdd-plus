package controllers

import (
	"bufio"
	"bytes"
	"encoding/base64"
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
	"math/rand"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// WxController 微信机器人控制器，处理微信消息接收和响应
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
type QXMoneyMessage struct {
	Event int    `json:"event"`
	Wxid  string `json:"wxid"`
	Data  struct {
		Type string `json:"type"`
		Des  string `json:"des"`
		Data struct {
			FromWxid    string `json:"fromWxid"`
			MsgSource   int    `json:"msgSource"`
			TransType   int    `json:"transType"`
			Money       string `json:"money"`
			Memo        string `json:"memo"`
			Transferid  string `json:"transferid"`
			Invalidtime string `json:"invalidtime"`
		} `json:"data"`
		Timestamp string `json:"timestamp"`
		Wxid      string `json:"wxid"`
		Port      int    `json:"port"`
		Pid       int    `json:"pid"`
		Flag      string `json:"flag"`
	} `json:"data"`
}

type QxFriendVerifyMsg struct {
	Event int    `json:"event"`
	Wxid  string `json:"wxid"`
	Data  struct {
		Type string `json:"type"`
		Des  string `json:"des"`
		Data struct {
			Wxid         string `json:"wxid"`
			WxNum        string `json:"wxNum"`
			Nick         string `json:"nick"`
			NickBrief    string `json:"nickBrief"`
			NickWhole    string `json:"nickWhole"`
			V3           string `json:"v3"`
			V4           string `json:"v4"`
			Sign         string `json:"sign"`
			Country      string `json:"country"`
			Province     string `json:"province"`
			City         string `json:"city"`
			AvatarMinURL string `json:"avatarMinUrl"`
			AvatarMaxURL string `json:"avatarMaxUrl"`
			Sex          string `json:"sex"`
			Content      string `json:"content"`
			Scene        string `json:"scene"`
		} `json:"data"`
		Timestamp string `json:"timestamp"`
		Wxid      string `json:"wxid"`
		Port      int    `json:"port"`
		Pid       int    `json:"pid"`
		Flag      string `json:"flag"`
	} `json:"data"`
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

type QxAgreeFriend struct {
	Type string `json:"type"`
	Data struct {
		Scene string `json:"scene"`
		V3    string `json:"v3"`
		V4    string `json:"v4"`
	} `json:"data"`
}

// HandleWxMessage 处理微信消息入口，根据消息事件类型分发到不同的处理逻辑
func (c *WxController) HandleWxMessage() {
	data := c.Ctx.Input.RequestBody
	logs.Info(string(data))

	if models.Config.Wx.Model == "qx" {
		event, _ := jsonparser.GetInt(data, "event")
		val, _ := jsonparser.GetString(data, "wxid")
		if val == models.Config.Wx.Robotid {
			switch event {

			case 10009:
				ag := &QXMessage{}
				err := json.Unmarshal(data, ag)
				logs.Info(err)
				msgType := ag.Data.Data.MsgType
				if msgType == 49 {
					decodedMsg, decodeErr := base64.StdEncoding.DecodeString(ag.Data.Data.MsgBase64)
					if decodeErr != nil {
						logs.Error("解密失败:", decodeErr)
						return
					}
					decodedStr := string(decodedMsg)
					if matches := regexp.MustCompile(`<url>(.*?)</url>`).FindStringSubmatch(decodedStr); len(matches) > 1 {
						ag.Data.Data.Msg = strings.ReplaceAll(matches[1], "&amp;", "&")
					}
					logs.Info("接收到信息" + ag.Data.Data.Msg)
					if ag.Wxid == models.Config.Wx.Robotid {
						models.ListenWXTempPrivateMessage(ag.Data.Data.FromWxid, ag.Data.Data.Msg)
					}

				} else {
					logs.Info("接收到信息" + ag.Data.Data.Msg)
					if ag.Wxid == models.Config.Wx.Robotid {
						models.ListenWXTempPrivateMessage(ag.Data.Data.FromWxid, ag.Data.Data.Msg)
					}
				}

			case 10008:
				ag := &QXMessage{}
				err := json.Unmarshal(data, ag)
				logs.Info(err)
				msgType := ag.Data.Data.MsgType
				if msgType == 49 {
					decodedMsg, decodeErr := base64.StdEncoding.DecodeString(ag.Data.Data.MsgBase64)
					if decodeErr != nil {
						logs.Error("解密失败:", decodeErr)
						return
					}
					decodedStr := string(decodedMsg)
					if matches := regexp.MustCompile(`<url>(.*?)</url>`).FindStringSubmatch(decodedStr); len(matches) > 1 {
						ag.Data.Data.Msg = strings.ReplaceAll(matches[1], "&amp;", "&")
					}
					logs.Info("接收到信息" + ag.Data.Data.Msg)
					if ag.Wxid == models.Config.Wx.Robotid {
						models.ListenWXGroupMessage(ag.Data.Data.FinalFromWxid, ag.Data.Data.FromWxid, ag.Data.Data.Msg)
					}

				} else {
					logs.Info("接收到信息" + ag.Data.Data.Msg)
					if ag.Wxid == models.Config.Wx.Robotid {
						models.ListenWXGroupMessage(ag.Data.Data.FinalFromWxid, ag.Data.Data.FromWxid, ag.Data.Data.Msg)
					}
				}

				case 10006:
				ag := &QXMoneyMessage{}
				err := json.Unmarshal(data, ag)
				logs.Info(err)
				args := make(map[string]string)
				args["model"] = "qx"
				args["money"] = ag.Data.Data.Money
				args["transferid"] = ag.Data.Data.Transferid
				args["to_wxid"] = ag.Data.Data.FromWxid

				logs.Info("Money:", args["money"])
				logs.Info("Transfer ID:", args["transferid"])
				logs.Info("To WXID:", args["to_wxid"])
				if args["to_wxid"] == models.Config.Wx.Robotid {
					return
				}
				time.Sleep(time.Duration(rand.Intn(1000)+2000) * time.Millisecond)
				// 执行转账请求
				err = models.TransferRequest(args["to_wxid"], args["transferid"], args["money"])
				if err != nil {
					logs.Info("调用 TransferRequest 失败:", err)
				} else {
					// 转账成功后执行充值逻辑
					money, err := strconv.ParseFloat(args["money"], 64)
					if err != nil {
						logs.Info("Money 转换失败:", err)
						return
					}
					rechargePoints := int(100.0 * money) // 计算充值积分

					// 获取 to_wxid 的 int 值
					id := models.GetWxid(args["to_wxid"])
					// 调用 AdddCoin 函数充值
					models.AdddCoin(id, rechargePoints)
					// 发送充值成功消息
					models.SendWxMsg(args["to_wxid"], fmt.Sprintf("充值成功！充值积分：%d\n充值后账户余额：%d\n注意：没收到请联系群主\n发送“菜单”获取更多功能", rechargePoints, models.GetCoin(id)))
				}

			case 10011:
				ag := &QxFriendVerifyMsg{}
				err := json.Unmarshal(data, ag)
				logs.Info(err)
				auto := models.IsAutoAgreeFriendVerify()
				if auto {
					if models.UseAgreeMsg() {
						AgreeMsg := models.GetEnv("AgreeMsg")
						if !strings.Contains(ag.Data.Data.Content, AgreeMsg) {
							return
						}
					}
					args := make(map[string]string)
					args["model"] = "qx"
					args["v3"] = ag.Data.Data.V3
					args["v4"] = ag.Data.Data.V4
					args["content"] = ag.Data.Data.Content
					args["uid"] = ag.Data.Data.Wxid
					AgreeFriendVerify(args)
				}
			}

		}
	} else {
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
					logs.Info("接收到信息" + ag.Content.Msg)
					models.ListenWXTempPrivateMessage(ag.Content.FromWxid, ag.Content.Msg)
				}
			case 2000:
			if ag.Content.RobotWxid == models.Config.Wx.Robotid {
				logs.Info("接收到转账" + ag.Content.Msg)
				if models.IsAutoAgreeAutocollection() {
					autocollect := &AutocollectMessageBody{}
					err := json.Unmarshal([]byte(ag.Content.Msg), autocollect)
					if err == nil && autocollect.PayerPayId != "" && autocollect.ReceiverPayId != "" && autocollect.Paysubtype == 1 {
						receiveMoney(autocollect, ag, 0)
					}
				}

			}

			case 2002:
				if ag.Content.RobotWxid == models.Config.Wx.Robotid {
					if matches := regexp.MustCompile(`<url>(.*?)</url>`).FindStringSubmatch(ag.Content.Msg); len(matches) > 1 {
						ag.Content.Msg = strings.ReplaceAll(matches[1], "&amp;", "&")
					}
					logs.Info("接收到信息" + ag.Content.Msg)
					models.ListenWXTempPrivateMessage(ag.Content.FromWxid, ag.Content.Msg)

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
}



// receiveMoney 处理微信转账收款，支持自动收款和自动收款+充值两种模式
func receiveMoney(autocollect *AutocollectMessageBody, ag *WxMessage, typ int) {
	// 检查是否是机器人给用户转账（机器人发起的转账不应该给用户充值积分）
	if ag.Content.FromWxid == models.Config.Wx.Robotid {
		logs.Info("检测到机器人给用户转账，跳过充值逻辑")
		return
	}
	
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

// AgreeFriendVerify 自动同意好友验证请求，支持my和qx两种微信框架
func AgreeFriendVerify(args interface{}) {
	arg := args.(map[string]string)
	model := arg["model"]
	switch model {
	case "my":
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
	case "qx":
		req := httplib.Post(models.Config.Wx.Url + "DaenWxHook/httpapi/?wxid=" + models.Config.Wx.Robotid)
		agree := &QxAgreeFriend{
			Type: "Q0017",
			Data: struct {
				Scene string `json:"scene"`
				V3    string `json:"v3"`
				V4    string `json:"v4"`
			}{
				Scene: arg["scene"],
				V3:    arg["v3"],
				V4:    arg["v4"],
			},
		}
		random := browser.Random()
		req.Header("User-Agent", random)
		marshal, _ := json.Marshal(agree)
		req.Body(string(marshal))
		s, _ := req.Bytes()
		val, _ := jsonparser.GetString(s, "msg")
		if val == "现在可以开始聊天" {
			welcome := models.GetEnv("Welcome")
			if welcome != "" {
				models.SendWxMsg(arg["uid"], welcome)
			}
		}
	}
}

// u2s 将Unicode转义字符串转换为普通字符串
func u2s(form string) (to string, err error) {
	bs, err := hex.DecodeString(strings.Replace(form, `\u`, ``, -1))
	if err != nil {
		return
	}
	for i, bl, br, r := 0, len(bs), bytes.NewReader(bs), uint16(0); i < bl; i += 2 {
		binary.Read(br, binary.BigEndian, &r)
		to += string(rune(r))
	}
	return
}


