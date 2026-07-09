package controllers

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/rand"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/buger/jsonparser"
	"github.com/cdle/xdd/models"
)

// HandleWechat08Message 处理 wecha08 机器人推送（千寻兼容事件结构）
// 与 qx/my 分支隔离，仅在 Config.Wx.Model == "wechat08" 时由入口调用
func (c *WxController) HandleWechat08Message(data []byte) {
	event, _ := jsonparser.GetInt(data, "event")
	val, _ := jsonparser.GetString(data, "wxid")
	if val != "" && val != models.Config.Wx.Robotid {
		return
	}

	switch event {
	case 10009:
		ag := &QXMessage{}
		if err := json.Unmarshal(data, ag); err != nil {
			models.Bot().Infof("[wechat08] 私聊解析失败: %v", err)
			return
		}
		msgType := ag.Data.Data.MsgType
		msg := ag.Data.Data.Msg
		if msgType == 49 {
			if ag.Data.Data.MsgBase64 != "" {
				if decoded, err := base64.StdEncoding.DecodeString(ag.Data.Data.MsgBase64); err == nil {
					decodedStr := string(decoded)
					if matches := regexp.MustCompile(`(?s)<url>(.*?)</url>`).FindStringSubmatch(decodedStr); len(matches) > 1 {
						msg = strings.ReplaceAll(matches[1], "&amp;", "&")
					}
				}
			}
		}
		models.UserLog().Infof("[微信wechat08] %s", msg)
		models.ListenWXTempPrivateMessage(ag.Data.Data.FromWxid, msg)

	case 10008:
		ag := &QXMessage{}
		if err := json.Unmarshal(data, ag); err != nil {
			models.Bot().Infof("[wechat08] 群聊解析失败: %v", err)
			return
		}
		msgType := ag.Data.Data.MsgType
		msg := ag.Data.Data.Msg
		if msgType == 49 {
			if ag.Data.Data.MsgBase64 != "" {
				if decoded, err := base64.StdEncoding.DecodeString(ag.Data.Data.MsgBase64); err == nil {
					decodedStr := string(decoded)
					if matches := regexp.MustCompile(`(?s)<url>(.*?)</url>`).FindStringSubmatch(decodedStr); len(matches) > 1 {
						msg = strings.ReplaceAll(matches[1], "&amp;", "&")
					}
				}
			}
		}
		models.UserLog().Infof("[微信群wechat08] %s", msg)
		models.ListenWXGroupMessage(ag.Data.Data.FinalFromWxid, ag.Data.Data.FromWxid, msg)

	case 10006:
		handleWechat08Transfer(data)

	case 10011:
		ag := &QxFriendVerifyMsg{}
		if err := json.Unmarshal(data, ag); err != nil {
			models.Bot().Infof("[wechat08] 好友验证解析失败: %v", err)
			return
		}
		if !models.IsAutoAgreeFriendVerify() {
			return
		}
		if models.UseAgreeMsg() {
			agreeMsg := models.GetEnv("AgreeMsg")
			if !strings.Contains(ag.Data.Data.Content, agreeMsg) {
				return
			}
		}
		scene := ag.Data.Data.Scene
		ok := models.Wechat08AgreeFriend(ag.Data.Data.V3, ag.Data.Data.V4, scene, ag.Data.Data.Wxid)
		models.Bot().Infof("[wechat08] 同意好友 result=%v uid=%s", ok, ag.Data.Data.Wxid)

	default:
		models.Bot().Infof("[wechat08] 未处理事件: %d", event)
	}
}

func handleWechat08Transfer(data []byte) {
	// 兼容扩展字段 transactionid / invalidtime
	type w08Money struct {
		Event int    `json:"event"`
		Wxid  string `json:"wxid"`
		Data  struct {
			Data struct {
				FromWxid      string `json:"fromWxid"`
				Money         string `json:"money"`
				Transferid    string `json:"transferid"`
				Transactionid string `json:"transactionid"`
				Invalidtime   string `json:"invalidtime"`
				Memo          string `json:"memo"`
			} `json:"data"`
		} `json:"data"`
	}
	ag := &w08Money{}
	if err := json.Unmarshal(data, ag); err != nil {
		models.Bot().Infof("[wechat08] 转账解析失败: %v", err)
		return
	}
	toWxid := ag.Data.Data.FromWxid
	if toWxid == "" || toWxid == models.Config.Wx.Robotid {
		return
	}
	moneyStr := ag.Data.Data.Money
	transferID := ag.Data.Data.Transferid
	transactionID := ag.Data.Data.Transactionid
	invalidTime := ag.Data.Data.Invalidtime

	models.Bot().Infof("[wechat08] 转账 money=%s transferid=%s from=%s", moneyStr, transferID, toWxid)
	time.Sleep(time.Duration(rand.Intn(1000)+2000) * time.Millisecond)

	err := models.Wechat08TransferRequest(toWxid, transferID, transactionID, invalidTime, moneyStr)
	if err != nil {
		models.Bot().Infof("[wechat08] 收款失败: %v", err)
		return
	}
	money, err := strconv.ParseFloat(moneyStr, 64)
	if err != nil {
		models.Bot().Infof("[wechat08] Money 转换失败: %v", err)
		return
	}
	rechargePoints := int(100.0 * money)
	id := models.GetWxid(toWxid)
	models.AdddCoin(id, rechargePoints)
	models.RecordCoinLog(id, rechargePoints, "充值", fmt.Sprintf("微信转账充值 %.2f元", money), models.WxBotContext())
	models.SendWxMsg(toWxid, fmt.Sprintf("充值成功！充值积分：%d\n充值后账户余额：%d\n注意：没收到请联系群主\n发送“菜单”获取更多功能", rechargePoints, models.GetCoin(id)))
}
