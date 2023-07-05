package models

import (
	"fmt"
	"gorm.io/gorm"

	"io/ioutil"
	"math/rand"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/beego/beego/v2/client/httplib"
	"github.com/beego/beego/v2/core/logs"
)

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
		//SendQQMsg(QQMessage{GroupID: gid, Message: msg.(string)})
	}
}

type ArkResData struct {
	Status uint   `json:"status"`
	Mode   string `json:"mode"`
}

type ArkRes struct {
	Success bool       `json:"success"`
	Message string     `json:"message"`
	Data    ArkResData `json:"data"`
}

type ViVoData struct {
	Autologin  int    `json:"autologin"`
	Gsalt      string `json:"gsalt"`
	GUID       string `json:"guid"`
	Lsid       string `json:"lsid"`
	NeedAuth   int    `json:"need_auth"`
	ReturnPage string `json:"return_page"`
	RsaModulus string `json:"rsa_modulus"`
}

type ViVoRes struct {
	Data    ViVoData `json:"data"`
	ErrCode int      `json:"err_code"`
	ErrMsg  string   `json:"err_msg"`
}

var ListenQQPrivateMessage = func(uid int, msg string) {
	//if strings.Contains(msg, "绑定微信") {
	//	SendQQ(uid, handleMessage(msg, "qq", int(uid)))
	//}
	SendQQ(uid, handleMessage(msg, "qq", uid))
}

var ListenWXTempPrivateMessage = func(uid string, msg string) {
	rt := handleMessage(msg, "wx", uid)

	switch rt.(type) {
	case string:
		SendWxMsg(uid, rt.(string))
	}
}

var ListenWXGroupMessage = func(uid string, gid string, msg string) {
	if strings.Contains(Config.WXGroupID, gid) || msg == "监听微信群" || msg == "取消监听" {
		rt := handleMessage(msg, "wxg", uid, gid)
		switch rt.(type) {
		case string:
			SendWxGroupMsg(uid, gid, rt.(string))
		}
	}

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

var replies = map[string]string{}
var tytlist = make(map[string]int)
var tytno = 0
var tytnum = 0
var loginList = make(map[int]chan string)

func InitReplies() {
	f, err := os.Open(ExecPath + "/conf/reply.php")
	if err == nil {
		defer f.Close()
		data, _ := ioutil.ReadAll(f)
		ss := regexp.MustCompile("`([^`]+)`\\s*=>\\s*`([^`]+)`").FindAllStringSubmatch(string(data), -1)
		for _, s := range ss {
			replies[s[1]] = s[2]
		}
	}
	if _, ok := replies["壁纸"]; !ok {
		replies["壁纸"] = "https://acg.toubiec.cn/random.php"
	}
}

var handleMessage = func(msgs ...interface{}) interface{} {
	time.Sleep(time.Second * time.Duration(rand.Intn(5)))
	logs.Info(msgs)
	msg := msgs[0].(string)
	args := strings.Split(msg, " ")
	head := args[0]
	contents := args[1:]
	sender := &Sender{
		UserID:   0,
		Type:     msgs[1].(string),
		Contents: contents,
	}
	//logs.Info(msgs[1])
	//logs.Info(msgs[2].(string))
	if msgs[1].(string) == "wx" || msgs[1].(string) == "wxg" {
		sender.UserID = getWxId(msgs[2].(string))
	} else {
		sender.UserID = msgs[2].(int)
	}
	if len(msgs) >= 4 && sender.Type != "wxg" {
		sender.ChatID = msgs[3].(int)
	}
	if sender.Type == "wx" {
		sender.WxId = msgs[2].(string)
	}

	if sender.Type == "wxg" {
		sender.WxId = msgs[2].(string)
		sender.WxGroupId = msgs[3].(string)
	}
	if sender.Type == "tgg" {
		sender.MessageID = msgs[4].(int)
		sender.Username = msgs[5].(string)
		sender.ReplySenderUserID = msgs[6].(int)
	}
	if sender.UserID == Config.TelegramUserID || sender.UserID == int(Config.QQID) {
		sender.IsAdmin = true
	}
	if sender.IsAdmin == false {
		if IsUserAdmin(strconv.Itoa(sender.UserID)) {
			sender.IsAdmin = true
		}
	}
	if loginList[sender.UserID] != nil {
		c2 := loginList[sender.UserID]
		c2 <- msg
		return nil
	}

	if smsList[sender.UserID] != nil {
		c2 := smsList[sender.UserID]
		c2 <- msg
		return nil
	}

	for i := range codeSignals {
		for j := range codeSignals[i].Command {
			if codeSignals[i].Command[j] == head {
				return func() interface{} {
					if codeSignals[i].Admin && !sender.IsAdmin {
						return "你没有权限操作"
					}
					return codeSignals[i].Handle(sender)
				}()
			}
		}
	}

	if Config.VIP {
		switch msg {
		default:

			//返利识别
			{
				matched, _ := regexp.MatchString("^https://item(.m)?.jd.com/(product/)?([0-9]+).html", msg)
				b2 := matched || strings.Contains(msg, "https://u.jd.com/")
				if b2 {
					return Get_powerful_link(msg)
				}
			}

			//绑定QQ
			{
				if strings.HasPrefix(msg, "DXWX") {
					return setWxId(msg, sender.WxId)
				}
			}

			//校验卡密
			{
				if strings.HasPrefix(msg, "XDD") {
					return useKey(msg, sender.UserID)
				}
			}

			//口令转换
			{
				if strings.Contains(msg, "口令") {
					sender.Reply(KLtoLJ(msg))
				}
			}

			//运行
			{
				if strings.Contains(msg, "运行") {
					lj := KLtoLJ(msg)
					msg = lj
				}
			}

			//验证码
			//{
			//	regex := "^\\d{5}(\\d|X|x)$"
			//	reg := regexp.MustCompile(regex)
			//	if reg.MatchString(msg) {
			//		logs.Info("进入验证码阶段")
			//		addr := Config.Jdcurl
			//		phone := pcodes[sender.UserID]
			//		if len(addr) > 0 {
			//			//若兰登录
			//
			//			risk := riskcodes[sender.UserID]
			//			logs.Info(sender.UserID)
			//			if strings.EqualFold(risk, "true") {
			//				logs.Info("进入风险验证阶段")
			//				if phone != "" {
			//					req := httplib.Post(addr + "/api/VerifyCardCode")
			//					req.Header("content-type", "application/json")
			//					data, _ := req.Body(`{"Phone":"` + phone + `","QQ":"` + strconv.Itoa(sender.UserID) + `","qlkey":0,"Code":"` + msg + `"}`).Bytes()
			//					var arkRes ArkRes
			//					json.Unmarshal(data, &arkRes)
			//					if arkRes.Success || strings.Contains(arkRes.Message, "添加xdd成功") {
			//						sender.Reply("登录成功。可以继续登录下一个账号")
			//						go func() {
			//							Save <- &JdCookie{}
			//						}()
			//					} else if !arkRes.Success {
			//						sender.Reply("验证失败,可能填写错误")
			//					}
			//				}
			//				riskcodes[sender.UserID] = "false"
			//			} else {
			//				logs.Info("进入验证码阶段")
			//				if phone != "" {
			//					req := httplib.Post(addr + "/api/VerifyCode")
			//					req.Header("content-type", "application/json")
			//					data, _ := req.Body(`{"Phone":"` + phone + `","QQ":"` + strconv.Itoa(sender.UserID) + `","qlkey":0,"Code":"` + msg + `"}`).Bytes()
			//					var arkRes ArkRes
			//					json.Unmarshal(data, &arkRes)
			//					if !arkRes.Success && arkRes.Data.Status == 555 {
			//
			//						switch arkRes.Data.Mode {
			//						case "USER_ID":
			//							//验证
			//							sender.Reply("你的账号需要验证才能登陆，请输入你的京东账号绑定的身份证前两位和后四位，最后一位如果是X，请输入大写X\n例如：31122X")
			//							//做个标记
			//							riskcodes[sender.UserID] = "true"
			//							if arkRes.Message != "" {
			//								sender.Reply(arkRes.Message)
			//							}
			//						case "HISTORY_DEVICE":
			//							sender.Reply("新设备登录需要验证，前往京东APP-我的-设置-账户与安全-新设备登录确认中确认，好了对我说:000000")
			//							riskcodes[sender.UserID] = "true"
			//						}
			//						//验证
			//						sender.Reply("你的账号需要验证才能登陆，请输入你的京东账号绑定的身份证前两位和后四位，最后一位如果是X，请输入大写X\n例如：31122X")
			//						//做个标记
			//						riskcodes[sender.UserID] = "true"
			//						if arkRes.Message != "" {
			//							sender.Reply(arkRes.Message)
			//						}
			//					} else if strings.Contains(arkRes.Message, "添加xdd成功") {
			//						sender.Reply("登录成功。可以继续登录下一个账号")
			//						go func() {
			//							Save <- &JdCookie{}
			//						}()
			//					} else {
			//						if arkRes.Message != "" {
			//							sender.Reply(arkRes.Message)
			//						} else {
			//							sender.Reply("登陆失败，请重新登录，多次尝试失败请联系管理员")
			//						}
			//					}
			//				}
			//			}
			//		} else if len(Config.Madurl) > 0 {
			//
			//		}
			//	}
			//}

			//手机号
			//{
			//	ist := pcodes[(sender.UserID)]
			//	if strings.EqualFold(ist, "true") {
			//		regular := `^(13[0-9]|14[01456879]|15[0-35-9]|16[2567]|17[0-8]|18[0-9]|19[0-35-9])\d{8}$`
			//		reg := regexp.MustCompile(regular)
			//		if reg.MatchString(msg) {
			//			//诺兰登录
			//			if len(Config.Jdcurl) > 0 {
			//				NolanSendSMS(msg, sender)
			//			} else if len(Config.Madurl) > 0 {
			//				RabbitSendSMS(msg, sender)
			//			}
			//		}
			//	}
			//}

			//识别登录
			{
				if msg == "登录" || msg == "登陆" {
					msg := make(chan string)
					loginList[sender.UserID] = msg
					go LoginSelect(sender, msg)

					msgs := []string{
						fmt.Sprintf("请选择登录渠道:"),
					}

					if sysConfig.RabbitUrl != "" && sysConfig.RabbitApiToken != "" && sysConfig.RabbitToken != "" {
						msgs = append(msgs, "1:兔子京东扫码")
					}

					if sysConfig.NolanUrl != "" && sysConfig.NolanToken != "" {
						msgs = append(msgs, "2:Nolan京东扫码")
					}

					if sysConfig.BBKJdUrl != "" && sysConfig.BBKToken != "" {
						msgs = append(msgs, "3:BBK京东扫码")
					}

					if sysConfig.BBKWxUrl != "" {
						msgs = append(msgs, "4:BBK微信扫码")
					}

					if sysConfig.RabbitUrl != "" && sysConfig.RabbitApiToken != "" && sysConfig.RabbitToken != "" {
						msgs = append(msgs, "5:兔子短信Wskey")
					}

					if Config.QQID == 764763903 {
						sender.Reply("请选择登录渠道: \r\n  2:Nolan京东扫码  \r\n  如需退出请回复'q'退出登录流程")
					} else {
						msgs = append(msgs, "如需退出请回复'q'退出登录流程")
						sender.Reply(strings.Join(msgs, "\n"))
					}

				}
			}

		}
	}

	switch msg {
	default:

		{
			if strings.Contains(msg, "wskey=") {
				logs.Info(msg + "开始WSKEY登录")
				wsKey := FetchJdCookieValue("wskey", msg)
				ptPin := FetchJdCookieValue("pin", msg)
				if len(ptPin) == 0 {
					ptPin = FetchJdCookieValue("pt_pin", msg)
				}
				if len(wsKey) > 0 && len(ptPin) > 0 {
					wkey := "pin=" + ptPin + ";wskey=" + wsKey + ";"
					rsp := getKey(wkey)
					if strings.Contains(rsp, "fake_") {
						logs.Error("wskey错误")
						sender.Reply(fmt.Sprintf("wskey错误 除京东APP皆不可用"))
					} else {
						ptKey := FetchJdCookieValue("pt_key", rsp)
						ptPin := FetchJdCookieValue("pt_pin", rsp)
						ck := JdCookie{
							PtPin: ptPin,
							PtKey: ptKey,
							WsKey: wsKey,
						}
						if CookieOK(&ck) {

							if sender.IsQQ() || sender.isWX() {
								ck.QQ = sender.UserID
							} else if sender.IsTG() {
								ck.Telegram = sender.UserID
							}
							if nck, err := GetJdCookie(ck.PtPin); err == nil {
								nck.Updates(JdCookie{PtKey: ptKey})
								if nck.WsKey == "" || len(nck.WsKey) == 0 {
									if sender.IsQQ() {
										ck.Update(QQ, ck.QQ)
									}
									nck.Update(WsKey, ck.WsKey)
									msg := fmt.Sprintf("写入WsKey，并更新账号%s", ck.PtPin)
									sender.Reply(fmt.Sprintf(msg))
									(&JdCookie{}).Push(msg)
									logs.Info(msg)
								} else {
									if nck.WsKey == ck.WsKey {
										msg := fmt.Sprintf("重复写入")
										sender.Reply(fmt.Sprintf(msg))
										(&JdCookie{}).Push(msg)
										logs.Info(msg)
									} else {
										nck.Updates(JdCookie{
											WsKey: ck.WsKey,
										})
										msg := fmt.Sprintf("更新WsKey，并更新账号%s", ck.PtPin)
										sender.Reply(fmt.Sprintf(msg))
										(&JdCookie{}).Push(msg)
										logs.Info(msg)
									}
								}

							} else {
								NewJdCookie(&ck)

								msg := fmt.Sprintf("添加账号，账号名:%s", ck.PtPin)

								if sender.IsQQ() {
									ck.Update(QQ, ck.QQ)
								}

								sender.Reply(fmt.Sprintf(msg))
								sender.Reply(ck.Query())
								(&JdCookie{}).Push(msg)
							}
						}
						go func() {
							Save <- &JdCookie{}
						}()
						return nil
					}
				}
			}
			//ss := regexp.MustCompile(`pin=([^;=\s]+);wskey=([^;=\s]+)`).FindAllStringSubmatch(msg, -1)
			//if len(ss) > 0 {
			//	for _, s := range ss {
			//		wkey := "pin=" + s[1] + ";wskey=" + s[2] + ";"
			//		//rsp := cmd(fmt.Sprintf(`python3 test.py "%s"`, wkey), &Sender{})
			//		rsp, err := getKey(wkey)
			//		if err != nil {
			//			logs.Error(err)
			//		}
			//		if strings.Contains(rsp, "fake_") {
			//			logs.Error("wskey错误")
			//			sender.Reply(fmt.Sprintf("wskey错误 除京东APP皆不可用"))
			//		} else {
			//			ptKey := FetchJdCookieValue("pt_key", rsp)
			//			ptPin := FetchJdCookieValue("pt_pin", rsp)
			//			ck := JdCookie{
			//				PtPin: ptPin,
			//				PtKey: ptKey,
			//				WsKey: s[2],
			//			}
			//			if CookieOK(&ck) {
			//
			//				if sender.IsQQ() {
			//					ck.QQ = sender.UserID
			//				} else if sender.IsTG() {
			//					ck.Telegram = sender.UserID
			//				}
			//				if nck, err := GetJdCookie(ck.PtPin); err == nil {
			//					nck.InPool(ck.PtKey)
			//					if nck.WsKey == "" || len(nck.WsKey) == 0 {
			//						if sender.IsQQ() {
			//							ck.Update(QQ, ck.QQ)
			//						}
			//						nck.Update(WsKey, ck.WsKey)
			//						msg := fmt.Sprintf("写入WsKey，并更新账号%s", ck.PtPin)
			//						sender.Reply(fmt.Sprintf(msg))
			//						(&JdCookie{}).Push(msg)
			//						logs.Info(msg)
			//					} else {
			//						if nck.WsKey == ck.WsKey {
			//							msg := fmt.Sprintf("重复写入")
			//							sender.Reply(fmt.Sprintf(msg))
			//							(&JdCookie{}).Push(msg)
			//							logs.Info(msg)
			//						} else {
			//							nck.Updates(JdCookie{
			//								WsKey: ck.WsKey,
			//							})
			//							msg := fmt.Sprintf("更新WsKey，并更新账号%s", ck.PtPin)
			//							sender.Reply(fmt.Sprintf(msg))
			//							(&JdCookie{}).Push(msg)
			//							logs.Info(msg)
			//						}
			//					}
			//
			//				} else {
			//					NewJdCookie(&ck)
			//
			//					msg := fmt.Sprintf("添加账号，账号名:%s", ck.PtPin)
			//
			//					if sender.IsQQ() {
			//						ck.Update(QQ, ck.QQ)
			//					}
			//
			//					sender.Reply(fmt.Sprintf(msg))
			//					sender.Reply(ck.Query())
			//					(&JdCookie{}).Push(msg)
			//				}
			//			}
			//			go func() {
			//				Save <- &JdCookie{}
			//			}()
			//			return nil
			//		}
			//	}
			//}
		}

		{ //tyt
			if strings.Contains(msg, "e6ad1ef1a55c440e9054e663be14e1d6") {
				no := tytno
				tytno += 1
				split := strings.Split(msg, "&amp;")
				for i := range split {
					if strings.Contains(split[i], "packetId=") {
						env := strings.Split(split[i], "=")
						if strings.Contains(env[1], "微信") {
							sender.Reply("微信渠道暂时无法识别")
						}
						if !sender.IsAdmin {
							coin := GetCoin(sender.UserID)
							if coin < Config.Tyt {
								return fmt.Sprintf("推一推需要%d个积分", Config.Tyt)
							}
							RemCoin(sender.UserID, Config.Tyt)

							sender.Reply(fmt.Sprintf("推一推即将开始，已扣除%d个积分,订单编号:%d，剩余%d", Config.Tyt, no, GetCoin(sender.UserID)))
						} else {
							sender.Reply(fmt.Sprintf("推一推即将开始，已扣除%d个积分，管理员通道", Config.Tyt))
						}
						//runTask(&Task{Path: "jd_tyt.js", Envs: []Env{
						//	{Name: "tytpacketId", Value: env[1]},
						//}}, sender)
						tytlist[env[1]] = no
						go runtyt(sender, env[1])
						//return fmt.Sprintf("订单编号：%d,推一推结束", no)
					}
				}
			}
		}

		{
			if Config.QQID == 764763903 {
				if msg == "导出助力账号" {
					var msgs []string
					cks := GetJdCookies(func(sb *gorm.DB) *gorm.DB {
						return sb.Where(fmt.Sprintf("%s >= ? and %s != ? and %s = ?", Priority, Hack, Available), 0, True, True)
					})
					for _, ck := range cks {
						msgs = append(msgs, fmt.Sprintf("pt_key=%s;pt_pin=%s;", ck.PtKey, ck.PtPin))
					}
					sender.Reply("导出所有账号")
					logs.Info("导出所有账号")
					f, err := os.OpenFile(ExecPath+"/scripts/ck.txt", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0777)
					if err != nil {
						logs.Warn("创建jdCookie.txt失败，", err)
					}
					join := strings.Join(msgs, "\n")
					f.WriteString(join)
					f.Close()
					return nil
				}
			}
		}

		{ //快递拆红包
			ss := regexp.MustCompile(`inviteId=(\S+)(&|&amp;)shareType`).FindStringSubmatch(msg)
			if len(ss) > 0 {
				if !sender.IsAdmin {
					coin := GetCoin(sender.UserID)
					if coin < 25 {
						return fmt.Sprintf("拆红包助力需要%d个互助值", 25)
					}
					RemCoin(sender.UserID, 25)
					order3 := &Order{id: branchHelpOrderNum, name: ss[1], Sender: sender}
					branchHelpOrderQueue.AddOrder(order3)
					sender.Reply(fmt.Sprintf("拆红包助力即将开始，已扣除%d个积分,剩余%d，订单ID:%d", 25, GetCoin(sender.UserID), branchHelpOrderNum))

				} else {
					order3 := &Order{id: branchHelpOrderNum, name: ss[1], Sender: sender}
					branchHelpOrderQueue.AddOrder(order3)
					sender.Reply(fmt.Sprintf("拆红包助力即将开始，已扣除%d个互助值，管理员通道,订单ID:%d", 25, branchHelpOrderNum))
				}
				branchHelpOrderNum++
				//runTask(&Task{Path: "jd_qmckd_branchHelp.js", Envs: []Env{
				//	{Name: "jd_qmckd_inviteIdArr_expand", Value: ss[1]},
				//}}, sender)

			}
		}

		{ //拆快递_任务助力
			ss := regexp.MustCompile(`taskHelp&inviteId=(\S+)(&|&amp;)mpin`).FindStringSubmatch(msg)
			if len(ss) > 0 {
				if !sender.IsAdmin {
					coin := GetCoin(sender.UserID)
					if coin < Config.Tyt {
						return fmt.Sprintf("拆快递_任务助力需要%d个互助值", Config.Tyt)
					}
					RemCoin(sender.UserID, Config.Tyt)
					sender.Reply(fmt.Sprintf("拆快递_任务助力即将开始，已扣除%d个积分,剩余%d", Config.Tyt, GetCoin(sender.UserID)))
				} else {
					sender.Reply(fmt.Sprintf("拆快递_任务助力即将开始，已扣除%d个互助值，管理员通道", Config.Tyt))
				}
				runTask(&Task{Path: "jd_qmckd_taskHelp.js", Envs: []Env{
					{Name: "jd_qmckd_inviteIdArr", Value: ss[1]},
				}}, sender)

				return "拆快递_任务助力已结束"
			}
		}

		{
			if strings.Contains(msg, "pt_key") {
				logs.Info(msg + "开始CK登录")
				ptKey := FetchJdCookieValue("pt_key", msg)
				ptPin := FetchJdCookieValue("pt_pin", msg)
				if len(ptPin) > 0 && len(ptKey) > 0 {
					ck := JdCookie{
						PtKey: ptKey,
						PtPin: ptPin,
					}
					if CookieOK(&ck) {
						if sender.IsQQ() || sender.isWX() {
							ck.QQ = sender.UserID
						} else if sender.IsTG() {
							ck.Telegram = sender.UserID
						}
						if HasKey(ck.PtKey) {
							sender.Reply(fmt.Sprintf("重复提交"))
						} else {
							if nck, err := GetJdCookie(ck.PtPin); err == nil {
								nck.Updates(JdCookie{PtKey: ptKey})
								msg := fmt.Sprintf("更新账号，%s", ck.PtPin)
								if sender.IsQQ() {
									ck.Update(QQ, ck.QQ)
								}
								sender.Reply(fmt.Sprintf(msg))
								(&JdCookie{}).Push(msg)
								logs.Info(msg)
							} else {
								if Cdle {
									ck.Hack = True
								}
								NewJdCookie(&ck)
								msg := fmt.Sprintf("添加账号，账号名:%s", ck.PtPin)
								if sender.IsQQ() {
									ck.Update(QQ, ck.QQ)
								}
								sender.Reply(fmt.Sprintf(msg))
								sender.Reply(ck.Query())
								(&JdCookie{}).Push(msg)
								logs.Info(msg)
							}
						}
					} else {
						sender.Reply(fmt.Sprintf("无效"))
					}
				}
				go func() {
					Save <- &JdCookie{}
				}()
				return nil
			}
		}

		for k, v := range replies {
			if regexp.MustCompile(k).FindString(msg) != "" {
				if strings.Contains(msg, "妹") && time.Now().Unix()%10 == 0 {
					v = "https://pics4.baidu.com/feed/d833c895d143ad4bfee5f874cfdcbfa9a60f069b.jpeg?token=8a8a0e1e20d4626cd31c0b838d9e4c1a"
				}
				if regexp.MustCompile(`^https{0,1}://[^\x{4e00}-\x{9fa5}\n\r\s]{3,}$`).FindString(v) != "" {
					url := v
					rsp, err := httplib.Get(url).Response()
					if err != nil {
						return nil
					}
					ctp := rsp.Header.Get("content-type")
					if ctp == "" {
						rsp.Header.Get("Content-Type")
					}
					if strings.Contains(ctp, "text") || strings.Contains(ctp, "json") {
						data, _ := ioutil.ReadAll(rsp.Body)
						return string(data)
					}
					return rsp
				}
				return v
			}
		}

	}
	return nil
}

func runtyt(sender *Sender, code string) {
	for {
		time.Sleep(time.Duration(rand.Intn(60)))
		if tytnum < 3 {
			tytnum++
			runTask(&Task{Path: "jd_tyt.js", Envs: []Env{
				{Name: "tytpacketId", Value: code},
			}}, sender)

			no := tytlist[code]
			sender.Reply(fmt.Sprintf("订单编号：%d,推一推结束", no))
			tytnum--
			return
		}
	}
}
