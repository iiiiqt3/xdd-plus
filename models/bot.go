package models

import (
	"crypto/md5"
	//	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/buger/jsonparser"
	//	"github.com/skip2/go-qrcode"
	"io"
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
				UserId  int    `json:"user_id"`
				GroupID int    `json:"group_id"`
				Message string `json:"message"`
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
				UserId  int    `json:"user_id"`
				GroupID int    `json:"group_id"`
				Message string `json:"message"`
			}{
				GroupID: gid,
				Message: msg.(string),
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

var ListenQQGroupMessage = func(uid int, gid int, msg string) {
	if strings.Contains(Config.QQGroupID, strconv.Itoa(gid)) {
		if Config.QbotPublicMode {
			SendQQGroup(gid, uid, handleMessage(msg, "qqg", uid, gid))
		} else {
			SendQQ(uid, handleMessage(msg, "qq", uid))
		}
	}
}

var pcodes = make(map[int]string)
var replies = map[string]string{}
var riskcodes = make(map[int]string)
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
	if msgs[1].(string) == "wx" {
		sender.UserID = getWxId(msgs[2].(string))
	} else {
		sender.UserID = msgs[2].(int)
	}
	if len(msgs) >= 4 {
		sender.ChatID = msgs[3].(int)
	}
	if sender.Type == "wx" {
		sender.WxId = msgs[2].(string)
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
					sender.Reply(NolanKl(msg))
				}
			}

			//运行
			{
				if strings.Contains(msg, "运行") {
					rsp := httplib.Post("http://jd.zack.xin/api/jd/ulink.php")
					rsp.Param("url", msg)
					rsp.Param("type", "hy")
					//rsp.Body(fmt.Sprintf(`url=%s&type=hy`, msg))
					data, err := rsp.Response()

					if err != nil {
						return "口令转换失败"
					}
					body, _ := ioutil.ReadAll(data.Body)
					if strings.Contains(string(body), "口令转换失败") {
						return "口令转换失败"
					} else {
						msg = string(body)
					}
				}
			}

			//验证码
			{
				regex := "^\\d{5}(\\d|X|x)$"
				reg := regexp.MustCompile(regex)
				if reg.MatchString(msg) {
					logs.Info("进入验证码阶段")
					addr := Config.Jdcurl
					phone := pcodes[sender.UserID]
					if len(addr) > 0 {
						//若兰登录

						risk := riskcodes[sender.UserID]
						logs.Info(sender.UserID)
						if strings.EqualFold(risk, "true") {
							logs.Info("进入风险验证阶段")
							if phone != "" {
								req := httplib.Post(addr + "/api/VerifyCardCode")
								req.Header("content-type", "application/json")
								data, _ := req.Body(`{"Phone":"` + phone + `","QQ":"` + strconv.Itoa(sender.UserID) + `","qlkey":0,"Code":"` + msg + `"}`).Bytes()
								var arkRes ArkRes
								json.Unmarshal(data, &arkRes)
								if arkRes.Success || strings.Contains(arkRes.Message, "添加xdd成功") {
									sender.Reply("登录成功。可以继续登录下一个账号")
									go func() {
										Save <- &JdCookie{}
									}()
								} else if !arkRes.Success {
									sender.Reply("验证失败,可能填写错误")
								}
							}
							riskcodes[sender.UserID] = "false"
						} else {
							logs.Info("进入验证码阶段")
							if phone != "" {
								req := httplib.Post(addr + "/api/VerifyCode")
								req.Header("content-type", "application/json")
								data, _ := req.Body(`{"Phone":"` + phone + `","QQ":"` + strconv.Itoa(sender.UserID) + `","qlkey":0,"Code":"` + msg + `"}`).Bytes()
								var arkRes ArkRes
								json.Unmarshal(data, &arkRes)
								if !arkRes.Success && arkRes.Data.Status == 555 {

									switch arkRes.Data.Mode {
									case "USER_ID":
										//验证
										sender.Reply("你的账号需要验证才能登陆，请输入你的京东账号绑定的身份证前两位和后四位，最后一位如果是X，请输入大写X\n例如：31122X")
										//做个标记
										riskcodes[sender.UserID] = "true"
										if arkRes.Message != "" {
											sender.Reply(arkRes.Message)
										}
									case "HISTORY_DEVICE":
										sender.Reply("新设备登录需要验证，前往京东APP-我的-设置-账户与安全-新设备登录确认中确认，好了对我说:000000")
										riskcodes[sender.UserID] = "true"
									}
									//验证
									sender.Reply("你的账号需要验证才能登陆，请输入你的京东账号绑定的身份证前两位和后四位，最后一位如果是X，请输入大写X\n例如：31122X")
									//做个标记
									riskcodes[sender.UserID] = "true"
									if arkRes.Message != "" {
										sender.Reply(arkRes.Message)
									}
								} else if strings.Contains(arkRes.Message, "添加xdd成功") {
									sender.Reply("登录成功。可以继续登录下一个账号")
									go func() {
										Save <- &JdCookie{}
									}()
								} else {
									if arkRes.Message != "" {
										sender.Reply(arkRes.Message)
									} else {
										sender.Reply("登陆失败，请重新登录，多次尝试失败请联系管理员")
									}
								}
							}
						}
					} else if len(Config.Madurl) > 0 {

					}
				}
			}

			//手机号
			{
				ist := pcodes[(sender.UserID)]
				if strings.EqualFold(ist, "true") {
					regular := `^(13[0-9]|14[01456879]|15[0-35-9]|16[2567]|17[0-8]|18[0-9]|19[0-35-9])\d{8}$`
					reg := regexp.MustCompile(regular)
					if reg.MatchString(msg) {
						//诺兰登录
						if len(Config.Jdcurl) > 0 {
							sender.Reply("请耐心等待...")
							addr := Config.Jdcurl
							req := httplib.Post(addr + "/api/SendSMS")
							req.Header("content-type", "application/json")
							data, _ := req.Body(`{"Phone":"` + msg + `","qlkey":0}`).Bytes()
							message, _ := jsonparser.GetString(data, "message")
							success, _ := jsonparser.GetBoolean(data, "success")
							status, _ := jsonparser.GetInt(data, "data", "status")
							captcha, _ := jsonparser.GetInt(data, "data", "captcha")
							if captcha == 0 {
								captcha = 1
							}
							if message != "" && status != 666 {
								sender.Reply(message)
							}
							i := 1

							if success {
								pcodes[sender.UserID] = msg
								logs.Info(strconv.Itoa(sender.UserID))
								sender.Reply("请输入6位验证码：")
								break
							}
							//{"success":true,"message":"","data":{"ckcount":0,"tabcount":3}}
							if !success && status == 666 && captcha == 2 {

								sender.Reply("正在进行验证...")
								for {
									req = httplib.Post(addr + "/api/AutoCaptcha")
									req.Header("content-type", "application/json")
									data, _ := req.Body(`{"Phone":"` + msg + `"}`).Bytes()
									message, _ := jsonparser.GetString(data, "message")
									success, _ := jsonparser.GetBoolean(data, "success")
									status, _ := jsonparser.GetInt(data, "data", "status")
									if !success {
										//s.Reply("滑块验证失败：" + string(data))
									}
									if success {
										pcodes[sender.UserID] = msg
										sender.Reply("请输入6位验证码：")
										break
									}
									if i > 5 {
										sender.Reply("滑块验证失败,请尝试重新登录")
										break
									}
									if status == 666 {
										i++
										sender.Reply(fmt.Sprintf("正在进行第%d次滑块验证...", i))
										continue
									}
									if strings.Contains(message, "上限") {
										i = 6
										sender.Reply(message)
										break
									}
								}

							} else {

								sender.Reply("滑块失败，请网页登录")
							}
							//{"success":true,"message":"","data":{"ckcount":0,"tabcount":3}}
						} else if len(Config.Madurl) > 0 {
							sender.Reply("请耐心等待...")
							addr := Config.Madurl
							req := httplib.Post(addr + "/api/SendSMS")
							req.Header("content-type", "application/json")
							data, _ := req.Body(`{"Phone":"` + msg + `","qlkey":0}`).Bytes()
							message, _ := jsonparser.GetString(data, "message")
							success, _ := jsonparser.GetBoolean(data, "success")
							status, _ := jsonparser.GetInt(data, "data", "status")
							if message != "" && status != 666 {
								sender.Reply(message)
							}
							i := 1

							if success {
								pcodes[sender.UserID] = msg
								logs.Info(string(sender.UserID))
								sender.Reply("请输入6位验证码：")
								break
							}
							//{"success":true,"message":"","data":{"ckcount":0,"tabcount":3}}
							if !success && status == 666 && i < 5 {

								sender.Reply("正在进行验证...")
								for {
									req = httplib.Post(addr + "/api/AutoCaptcha")
									req.Header("content-type", "application/json")
									data, _ := req.Body(`{"Phone":"` + msg + `"}`).Bytes()
									message, _ := jsonparser.GetString(data, "message")
									success, _ := jsonparser.GetBoolean(data, "success")
									status, _ := jsonparser.GetInt(data, "data", "status")
									if success {
										pcodes[sender.UserID] = msg
										sender.Reply("请输入6位验证码：")
										break
									}
									if i > 5 {
										//pcodes[sender.UserID] = msg
										//s := Config.Jdcurl + "/Captcha/" + msg
										//sender.Reply(fmt.Sprintf("请访问网址进行手动验证%s", s))
										sender.Reply("滑块验证失败,请尝试重新登录")
										break
									}
									if status == 666 {
										i++
										sender.Reply(fmt.Sprintf("正在进行第%d次滑块验证...", i))
										continue
									}
									if strings.Contains(message, "上限") {
										i = 6
										sender.Reply(message)
										break
									}
								}
							} else {
								sender.Reply("滑块失败，请网页登录")
							}
						}
					}
				}
			}

			//识别登录
			{
				if strings.Contains(msg, "登录") || strings.Contains(msg, "登陆") {

					//if len(Config.Jdcurl) > 0 {
					//	var tabcount int64
					//	addr := Config.Jdcurl
					//	if addr == "" {
					//		return "若兰很忙，请稍后再试。"
					//	}
					//	logs.Info(addr + "/api/Config")
					//	if addr != "" {
					//		data, _ := httplib.Get(addr + "/api/Config").Bytes()
					//		logs.Info(string(data) + "返回数据")
					//		tabcount, _ = jsonparser.GetInt(data, "data", "autocount")
					//		if tabcount != 0 {
					//			pcodes[sender.UserID] = "true"
					//			riskcodes[sender.UserID] = "false"
					//			sender.Reply("若兰为您服务，请输入11位手机号：")
					//		} else {
					//			sender.Reply("服务忙，请稍后再试。")
					//		}
					//	}
					//} else {
					//	//pcodes[sender.UserID] = "true"
					//	//sender.R
					//	//eply("小滴滴")
					//}

					//sender.Reply("服务升级中，目前登录请私聊群主谢谢")

					msg := make(chan string)
					loginList[sender.UserID] = msg
					go LoginSelect(sender, msg)
					sender.Reply("请选择登录渠道: \r\n 1:京东扫码  \r\n 2:微信扫码(渠道适配中)  \r\n 3:短信登录（渠道升级中）")

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
					rsp, err := getKey(wkey)
					if err != nil {
						logs.Error(err)
					}
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

//随机slice数组
func randShuffle(slice []JdCookie) {
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(slice), func(i, j int) {
		slice[i], slice[j] = slice[j], slice[i]
	})
}

func getMd5String1(str string) string {
	m := md5.New()
	io.WriteString(m, str)
	arr := m.Sum(nil)
	return fmt.Sprintf("%x", arr)
}

func FetchJdCookieValue(key string, cookies string) string {
	match := regexp.MustCompile(key + `=([^;]*);{0,1}`).FindStringSubmatch(cookies)
	if len(match) == 2 {
		return match[1]
	} else {
		return ""
	}
}

func LoginSelect(sender *Sender, msg chan string) {
	for {
		n, ok := <-msg
		//说明发送方关闭了channel
		if !ok {
			break
		}
		switch n {
		case "京东扫码", "1":
			NolanGetJdQrImg(sender)
			loginList[sender.UserID] = nil
		case "微信扫码", "2":
			sender.Reply("渠道适配中")
		case "短信登录", "3":
			sender.Reply("渠道升级，等待后续开放")
		case "q":
			loginList[sender.UserID] = nil
			close(msg)
		default:
			sender.Reply("无匹配渠道，如需回复'q'退出登录流程")
		}
	}
}
