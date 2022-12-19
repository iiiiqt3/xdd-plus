package models

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	browser "github.com/EDDYCJY/fake-useragent"
	"github.com/buger/jsonparser"
	"github.com/tinyhubs/tinydom"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/beego/beego/v2/client/httplib"
	"github.com/beego/beego/v2/core/logs"
)

var SendQQ = func(a int64, b interface{}) {

}
var SendQQGroup = func(a int64, b int64, c interface{}) {

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

var ListenQQPrivateMessage = func(uid int64, msg string) {
	//if strings.Contains(msg, "绑定微信") {
	//	SendQQ(uid, handleMessage(msg, "qq", int(uid)))
	//}
	SendQQ(uid, handleMessage(msg, "qq", int(uid)))
}

var ListenQQTempPrivateMessage = func(uid int64, msg string) {
	if strings.Contains(msg, "绑定微信") {
		SendQQ(uid, handleMessage(msg, "qq", int(uid)))
	}
}

var ListenWXTempPrivateMessage = func(uid string, msg string) {
	rt := handleMessage(msg, "wx", uid)

	switch rt.(type) {
	case string:
		SendWxMsg(uid, rt.(string))
	}
}

var ListenQQGroupMessage = func(gid int64, uid int64, msg string) {
	if gid == Config.QQGroupID {
		if Config.QbotPublicMode {
			SendQQGroup(gid, uid, handleMessage(msg, "qqg", int(uid), int(gid)))
		} else {
			SendQQ(uid, handleMessage(msg, "qq", int(uid)))
		}
	}
}

var ua = "Mozilla/5.0 (iPhone; CPU iPhone OS 13_3_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/15E148 SP-engine/2.14.0 main%2F1.0 baiduboxapp/11.18.0.16 (Baidu; P2 13.3.1) NABar/0.0"

var pcodes = make(map[int]string)
var replies = map[string]string{}
var riskcodes = make(map[int]string)
var tytlist = make(map[string]int)
var tytno = 0
var tytnum = 0
var pzlist = make(map[string]int)
var pz = 0
var pzno = 0
var zdlist = make(map[string]int)
var zd = 0
var zdno = 0

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

			//赚钱大赢家
			{
				if sender.IsAdmin {
					if strings.Contains(msg, "赚钱大赢家") {
						rsp := httplib.Post("http://jd.zack.xin/api/jd/ulink.php")
						rsp.Param("url", msg)
						rsp.Param("type", "hy")
						data, err := rsp.Response()

						if err != nil {
							return "口令转换失败"
						}
						body, _ := ioutil.ReadAll(data.Body)
						if strings.Contains(string(body), "口令转换失败") {
							return "口令转换失败"
						} else {
							//inviterCode := regexp.MustCompile(`shareId=(\S+)(&|&amp;)bridgeType`).FindStringSubmatch(string(body))
							inviterCode := ""
							split := strings.Split(string(body), "&")
							for i := range split {
								if strings.Contains(split[i], "shareId=") {
									logs.Info(split[i])
									env := strings.Split(split[i], "=")
									logs.Info(env[1])
									inviterCode = env[1]
								}
							}
							no := pzno
							pzno += 1
							pzlist[inviterCode] = no
							sender.Reply(fmt.Sprintf("订单编号：%d,已进入队列", no))
							go rundyj(sender, inviterCode)

							//split := strings.Split(string(body), "【链接】")
							//f, err := os.OpenFile(ExecPath+"/zqdyj.txt", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0777)
							//if err != nil {
							//	logs.Warn("zqdyj.txt失败，", err)
							//}
							//sender.Reply("已提交")
							//f.WriteString(split[1])
							//f.Close()
						}
						//}
					} else if strings.Contains(msg, "https://wqs.jd.com/sns/") {
						//inviterCode := regexp.MustCompile(`shareId=(\S+)(&|&amp;)bridgeType`).FindStringSubmatch(msg)
						inviterCode := ""
						split := strings.Split(msg, "&")
						logs.Info(msg)
						for i := range split {
							if strings.Contains(split[i], "shareId=") {
								logs.Info(split[i])
								env := strings.Split(split[i], "=")
								logs.Info(env[1])
								inviterCode = env[1]
							}
						}
						no := pzno
						pzno += 1
						pzlist[inviterCode] = no
						sender.Reply(fmt.Sprintf("订单编号：%d,已进入队列", no))
						go rundyj(sender, inviterCode)
						//f, err := os.OpenFile(ExecPath+"/zqdyj.txt", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0777)
						//if err != nil {
						//	logs.Warn("zqdyj.txt失败，", err)
						//}
						//sender.Reply("已提交")
						//f.WriteString(msg + "\n")
						//f.Close()
					}
				}

			}

			//膨胀
			//{
			//	if sender.IsAdmin {
			//		if strings.Contains(msg, "膨胀") {
			//			rsp := httplib.Post("http://jd.zack.xin/api/jd/ulink.php")
			//			rsp.Param("url", msg)
			//			rsp.Param("type", "hy")
			//			data, err := rsp.Response()
			//
			//			if err != nil {
			//				return "口令转换失败"
			//			}
			//			body, _ := ioutil.ReadAll(data.Body)
			//			if strings.Contains(string(body), "口令转换失败") {
			//				return "口令转换失败"
			//			} else {
			//				if strings.Contains(string(body), "shareType=expandHelp") {
			//					sender.Reply("开始助力")
			//					inviterCode := regexp.MustCompile(`inviteId=(\S+)(&|&amp;)mpin`).FindStringSubmatch(string(body))
			//					f, err := os.OpenFile(ExecPath+"/zqdyj.txt", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0777)
			//					if err != nil {
			//						logs.Warn("zqdyj.txt失败，", err)
			//					}
			//					sender.Reply("已提交")
			//					f.WriteString(inviterCode[1] + "&")
			//					f.Close()
			//
			//					//runTask(&Task{Path: "jd_racxj_expandHelp.js", Envs: []Env{
			//					//	{Name: "jd_racxj_inviteIdArr_expand", Value: inviterCode[1]}, {Name: "gua_racxj_token", Value: GetEnv("token")},
			//					//}}, sender)
			//					//go runpz(sender, inviterCode[1])
			//				}
			//			}
			//		}
			//	}
			//
			//}

			//金币助力
			//{
			//	if sender.IsAdmin {
			//		if strings.Contains(msg, "助力") {
			//			rsp := httplib.Post("http://jd.zack.xin/api/jd/ulink.php")
			//			rsp.Param("url", msg)
			//			rsp.Param("type", "hy")
			//			data, err := rsp.Response()
			//
			//			if err != nil {
			//				return "口令转换失败"
			//			}
			//			body, _ := ioutil.ReadAll(data.Body)
			//			if strings.Contains(string(body), "口令转换失败") {
			//				return "口令转换失败"
			//			} else {
			//				if strings.Contains(string(body), "shareType=taskHelp") {
			//					sender.Reply("开始助力")
			//					inviterCode := regexp.MustCompile(`inviteId=(\S+)(&|&amp;)mpin`).FindStringSubmatch(string(body))
			//					sender.Reply("开始助力，管理员")
			//
			//					runTask(&Task{Path: "jd_racxj_taskHelp.js", Envs: []Env{
			//						{Name: "jd_racxj_inviteIdArr", Value: inviterCode[1]}, {Name: "gua_racxj_token", Value: GetEnv("token")},
			//					}}, sender)
			//					//flag := nianhelp(inviterCode[1])
			//					//if flag {
			//					//	return "助力完成"
			//					//} else {
			//					//	return "助力失败"
			//					//}
			//				}
			//			}
			//		}
			//	}
			//}

			//组队
			//{
			//	if sender.IsAdmin {
			//		if strings.Contains(msg, "加入") || strings.Contains(msg, "咖叺") {
			//			rsp := httplib.Post("http://jd.zack.xin/api/jd/ulink.php")
			//			rsp.Param("url", msg)
			//			rsp.Param("type", "hy")
			//			data, err := rsp.Response()
			//
			//			if err != nil {
			//				return "口令转换失败"
			//			}
			//			body, _ := ioutil.ReadAll(data.Body)
			//			if strings.Contains(string(body), "口令转换失败") {
			//				return "口令转换失败"
			//			} else {
			//				if strings.Contains(string(body), "shareType=team") {
			//					sender.Reply("已提交")
			//					f, err := os.OpenFile(ExecPath+"/zdzl.txt", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0777)
			//					if err != nil {
			//						logs.Warn("zdzl.txt失败，", err)
			//					}
			//					f.WriteString(string(body) + "\n")
			//					f.Close()
			//					//inviterCode := regexp.MustCompile(`inviteId=(\S+)(&|&amp;)mpin`).FindStringSubmatch(string(body))
			//					//
			//					//flag := zdhelp(inviterCode[1])
			//					//if flag {
			//					//	return "助力完成"
			//					//} else {
			//					//	return "助力失败"
			//					//}
			//				}
			//			}
			//		}
			//	}
			//}

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

			//挖宝统计
			{
				if strings.Contains(msg, "https://bnzf.jd.com/") {
					if strings.Contains(msg, "xml version=") {
						doc, _ := tinydom.LoadDocument(strings.NewReader(msg))
						talk := doc.FirstChildElement("msg").FirstChildElement("appmsg").FirstChildElement("url").Text()
						fmt.Print(talk)
						msg = talk
					}

					split := strings.Split(msg, "&")
					inviterId := ""
					inviterCode := ""
					for i := range split {
						if strings.Contains(split[i], "inviterId=") {
							env := strings.Split(split[i], "=")
							inviterId = env[1]
						}
						if strings.Contains(split[i], "inviterCode=") {
							env := strings.Split(split[i], "=")
							inviterCode = env[1]
						}
					}
					f, err := os.OpenFile(ExecPath+"/table.txt", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0777)
					if err != nil {
						logs.Warn("table.txt失败，", err)
					}
					str := fmt.Sprintf("https://bnzf.jd.com/?activityId=pTTvJeSTrpthgk9ASBVGsw&inviterId=%s&inviterCode=%s&utm_user=plusmember&ad_od=share&utm_source=androidapp&utm_medium=appshare&utm_campaign=t_335139774&utm_term=Wxfriends", inviterId, inviterCode)
					sender.Reply("已提交")
					f.WriteString(str + "\n")
					f.Close()

					//msg = strings.ReplaceAll(msg, "&amp;", "&")
					//f, err := os.OpenFile(ExecPath+"/wblj.txt", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0777)
					//if err != nil {
					//	logs.Warn("wb.txt失败，", err)
					//}
					//if GetCoin(sender.UserID) > 19 {
					//	f.WriteString(msg + "\n")
					//	RemCoin(sender.UserID, 20)
					//	sender.Reply(fmt.Sprintf("已提交转订单，扣除积分20，剩余积分：%d", GetCoin(sender.UserID)))
					//} else {
					//	sender.Reply("积分不足")
					//}
					//f.Close()
				}
			}

			//锦鲤统计
			{
				if strings.Contains(msg, "红包") {
					if GetCoin(sender.UserID) > 24 {
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
							s := string(body)
							if strings.Contains(s, "3ugedFa7yA6NhxLN5gw2L3PF9sQC") {
								split := strings.Split(s, "index.html?asid=")
								f, err := os.OpenFile(ExecPath+"/jl.txt", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0777)
								if err != nil {
									logs.Warn("wb.txt失败，", err)
								}
								f.WriteString(split[1] + "&")
								RemCoin(sender.UserID, 25)
								sender.Reply("已提交")
							} else {
								return "非锦鲤口令"
							}
						}
					} else {
						return "积分不足"
					}
				}
			}

			//口令转换
			{
				if strings.Contains(msg, "口令") {
					KLtoLJ(msg)
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
									//做个标记
									riskcodes[sender.UserID] = "true"
									if arkRes.Message != "" {
										if strings.Contains(arkRes.Message, "Object reference not set to an instance of an object.") {
											sender.Reply("登录失败,请使用APP登录。")
										} else {
											sender.Reply(arkRes.Message)
										}

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
									}
									//sender.Reply(message)
								}
								//} else if !success && captcha == 2 {
								//	pcodes[string(sender.UserID)] = msg
								//	s := Config.Jdcurl + "/Captcha/" + msg
								//	sender.Reply(fmt.Sprintf("请访问网址进行手动验证%s", s))

							} else {

								sender.Reply("滑块失败，请网页登录")
							}
							//{"success":true,"message":"","data":{"ckcount":0,"tabcount":3}}
						}
					}
				}
			}

			//识别登录
			{
				if strings.Contains(msg, "登录") || strings.Contains(msg, "登陆") {

					if len(Config.Jdcurl) > 0 {
						var tabcount int64
						addr := Config.Jdcurl
						if addr == "" {
							return "若兰很忙，请稍后再试。"
						}
						logs.Info(addr + "/api/Config")
						if addr != "" {
							data, _ := httplib.Get(addr + "/api/Config").Bytes()
							logs.Info(string(data) + "返回数据")
							tabcount, _ = jsonparser.GetInt(data, "data", "autocount")
							if tabcount != 0 {
								pcodes[sender.UserID] = "true"
								riskcodes[sender.UserID] = "false"
								sender.Reply("若兰为您服务，请输入11位手机号：")
							} else {
								sender.Reply("服务忙，请稍后再试。")
							}
						}
					} else {
						//pcodes[sender.UserID] = "true"
						//sender.Reply("小滴滴")
					}

					//sender.Reply("服务升级中，目前登录请私聊群主谢谢")
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

							if sender.IsQQ() {
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
									msg := fmt.Sprintf("年费新增成功，账号%s", ck.PtPin)
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
										msg := fmt.Sprintf("年费更新成功，账号%s", ck.PtPin)
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

		//{ //
		//	ss := regexp.MustCompile(`pt_key=([^;=\s]+);pt_pin=([^;=\s]+)`).FindAllStringSubmatch(msg, -1)
		//
		//	if len(ss) > 0 {
		//
		//		xyb := 0
		//		for _, s := range ss {
		//			ck := JdCookie{
		//				PtKey: s[1],
		//				PtPin: s[2],
		//			}
		//			xyb++
		//			if sender.IsQQ() {
		//				ck.QQ = sender.UserID
		//			} else if sender.IsTG() {
		//				ck.Telegram = sender.UserID
		//			}
		//			if HasKey(ck.PtKey) {
		//				sender.Reply(fmt.Sprintf("重复提交"))
		//			} else {
		//				if nck, err := GetJdCookie(ck.PtPin); err == nil {
		//					nck.InPool(ck.PtKey)
		//					msg := fmt.Sprintf("更新账号，%s", ck.PtPin)
		//					(&JdCookie{}).Push(msg)
		//					logs.Info(msg)
		//				} else {
		//					if Cdle {
		//						ck.Hack = True
		//					}
		//					NewJdCookie(&ck)
		//					msg := fmt.Sprintf("添加账号，%s", ck.PtPin)
		//					sender.Reply(fmt.Sprintf("很棒，许愿币+1，余额%d", AddCoin(sender.UserID)))
		//					logs.Info(msg)
		//				}
		//			}
		//
		//		}
		//		go func() {
		//			Save <- &JdCookie{}
		//		}()
		//		return nil
		//	}
		//}

		//{
		//	//k1k
		//	ss := regexp.MustCompile(`launchid=(\S+)(&|&amp;)ptag`).FindStringSubmatch(msg)
		//	if len(ss) > 0 {
		//		if !sender.IsAdmin {
		//			sender.Reply("仅管理员可用")
		//		} else {
		//			sender.Reply(fmt.Sprintf("砍价开始，管理员通道"))
		//			runTask(&Task{Path: "jd_kanjia.js", Envs: []Env{
		//				{Name: "launchid", Value: ss[1]},
		//			}}, sender)
		//		}
		//		return nil
		//	}
		//}

		{ //tyt
			if strings.Contains(msg, "3075b6eab065464dad1c4042d345ac97") {
				no := tytno
				tytno += 1
				split := strings.Split(msg, "&amp;")
				for i := range split {
					if strings.Contains(split[i], "packetId=") {
						f, err := os.OpenFile(ExecPath+"/tytlj.txt", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0777)
						if err != nil {
							logs.Warn("tytlj.txt失败，", err)
						}
						logs.Info(split[i])
						env := strings.Split(split[i], "=")
						if strings.Contains(env[1], "微信") {
							sender.Reply("微信渠道暂时无法识别")
						}
						f.WriteString(env[1] + "\n")
						f.Close()
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
						//return "推一推已结束"
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
						if sender.IsQQ() {
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
					data, _ := ioutil.ReadAll(rsp.Body)
					return data
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
			//runTask(&Task{Path: "jd_tyt.js", Envs: []Env{
			//	{Name: "tytpacketId", Value: code},
			//}}, sender)
			//
			num, f := starttyt(code)
			no := tytlist[code]
			if f {
				sender.Reply(fmt.Sprintf("订单编号：%d,推一推结束共用:%d个账号", no, num))
			} else {
				sender.Reply(fmt.Sprintf("订单编号：%d,推一推异常，请联系群主，或自行检查", no))
			}

			tytnum--
			//sender.Reply("推一推已结束，请检查是否完成，未完成请联系群主")
			return
		}
	}
}

func nianhelp(invited string) (flag bool) {
	logs.Info("开始金币助力")
	k := 0
	cks := GetJdCookies()
	db.Where(fmt.Sprintf("%s = 'true' and %s = 'true'", Tyt, Available)).Order("RAND()").Find(&cks)
	for _, ck := range cks {
		time.Sleep(time.Second * time.Duration(3))
		cookie := "pt_key=" + ck.PtKey + ";pt_pin=" + ck.PtPin + ";"
		sc := getScKey(cookie)
		if sc != "" {
			url := "https://api.m.jd.com/client.action?functionId=promote_collectScore"
			//body := fmt.Sprintf(`{"ss":"{\"extraData\":{\"log\":\"\",\"sceneid\":\"HYGJZYh5\"},\"secretp\":\"%s\",\"random\":\"%d\"}","inviteId":"%s"}`, sc, rand.Intn(99999999), invited)
			body := fmt.Sprintf(`{"random":"3m5QtABC","log":"1666267341263~194Cl2EtLOeMDFqSmpmSzAxMQ==.W3xcUHlcfVlVclx+WRgBKBAYLQwBegJXNVtmKUoIRnsUVzVbNActOBsyM1IGBwAIJRgvIz4HJQB8Gl8aFHtaSnpaNA==.6c8730f2~C,1~22CD2662C9991565879D915A12D2A9A085468D19~07bzbsz~C~SRJAWBANam0cFkQPWxAOaxYFBhQLcE10AxgEfX4dURxAEk0UVgMdByMffwseUGAEGFQeQxMcElABTAVwGHYHTQJqbR4UHkQWaB4VVkJeFgpQGhBHQxZbEQgOBFcABgMGAQ8HBQINBlcEEBgSQwRXGwIQFUZEQEFUQlcSHBZHBFcQDhJSB0dNTEYUU1EWGRBHVV4SDmtYGgYFHAZNABUJHlRvHBZfWBULARwWUxIUCBYJBVhQClwCVQsGAgYKUVNUVQ0IUQYLDQhRAlUBDABWVxIYF1xHEwoSXWAJWVxREhhDRxsCA1cEBgYDBQQFAQICA00UWF8SDkMKWwtXAwUIUAZQBgEEA1FTAFQGUgUFBwEBXlRWB1QBVAMEBwlWBgcDFB4WVkQDEQMaXypBQUxsBnpcelJ3YyRfZlVeXldDAGkQTRBeQhcIFXBAQFhVQXVdWUBBFVZLFBIoXFMaFx4VX1FGFgpDBwQMAgVRERUaQQJAEg5uCgMFHAMNADwaEEZfFltoG1FiCV1eUQQBGwMSHBZZLmUQGBIFVR0PGh5DAwEaBBwDExwSBQZZBAMEEhhDClsLVwMFCFAGUAYBBANRUwBUBlIFBQcBAV5UVgdUAVQDBAcJVgYHAxQeFlEWPB8bUV0AEAoWU1RRV1ZWQERDGhBVWhZbEUwaHkNRWRYPEEAFHgAaBUMaEFdWaxcRAxoCUBAcFldWFQsSQlVeBVlfCQNZVGJNeXAiEBwWWFgVC2sBGABNBm8YElYNXF4aCEMDBgIDAAEGAwMMBlMOTAVhZRcEYAhFEEFEeXF4elReXFxmG3dLenEJXB1fbUoxZAJiA2ZiRWpqBwkqYnAMZFA1aWltWFQLfmZgSnxlQGd1AFAPa3FDfDdKVQFwI2djcU8KTnpicVd5MA5kcEkAJHgJAEguSABNcXMDVHNaDEQ0fkVyfV0KZ3F8AiBkXHV6dWJFcX9ncVdjY0dzbhtcTGxYMAd+cQ1/ellhXHkABm9/XX1iMGJeT1ckQnF5YgYAZ2RYY3YlYwJ4Wm1VfGttYSdfQEx3c1QMHgcHBQJVVFFSShgfCEZMH3BOZ118ZGZidXl1N2NxfFdxMntBbmMkRVtmYlp+VXVcTHEkZ11gdUxRYGtAYyt0cV9yc2ZWYndMdjJgZUx2cQZAanljKGB4dnt2ckZfYnJyWWNlcXlmIlt8cHoWd3R1Qn9keWJ6ZXUGXWFxAENcTQlcRwNZBloXHhVcQ1cWCkMUHhZIVxMRAxoCUE5IDUlIW1ZXV0ZXUUxYWBJJ~1uk1y6y","actionType":"0","inviteId":"%s"}`, invited)
			req := httplib.Post(url)
			random := browser.Random()
			req.Param("clientVersion", "-1")
			req.Param("appid", "signed_wh5")
			req.Param("functionId", "promote_collectScore")
			req.Param("body", body)
			req.Header("User-Agent", random)
			req.Header("Accept", "application/json, text/plain, */*")
			req.Header("Connection", "keep-alive")
			req.Header("Accept-Language", "zh-cn")
			req.Header("Accept-Encoding", "gzip, deflate, br")
			req.Header("Origin", "https://bunearth.m.jd.com")
			req.Header("Cookie", cookie)
			s, _ := req.String()
			bizCode, _ := jsonparser.GetInt([]byte(s), "data", "bizCode")
			if bizCode == 0 {
				k++
				logs.Info("助力成功")
				logs.Info(s)
			} else {
				logs.Info("助力失败")
				logs.Info(s)
				if strings.Contains(s, "好友人气爆棚") {
					return true
				} else if strings.Contains(s, "火爆") {
					ck.Update(Tyt, False)
				} else {
					ck.Update(Tyt, s)
				}
			}
		}
	}
	return false
}

func zdhelp(invited string) (flag bool) {
	logs.Info("开始组队")
	k := 0
	cks := GetJdCookies()
	db.Where(fmt.Sprintf("%s = 'true' and %s = 'true'", Dig, Available)).Order("RAND()").Find(&cks)
	for _, ck := range cks {
		time.Sleep(time.Second * time.Duration(3))
		cookie := "pt_key=" + ck.PtKey + ";pt_pin=" + ck.PtPin + ";"
		sc := getScKey(cookie)
		if sc != "" {
			url := "https://api.m.jd.com/client.action?functionId=promote_pk_joinGroup"
			body := fmt.Sprintf(`{"random":"3m5QtABC","log":"1666267341263~194Cl2EtLOeMDFqSmpmSzAxMQ==.W3xcUHlcfVlVclx+WRgBKBAYLQwBegJXNVtmKUoIRnsUVzVbNActOBsyM1IGBwAIJRgvIz4HJQB8Gl8aFHtaSnpaNA==.6c8730f2~C,1~22CD2662C9991565879D915A12D2A9A085468D19~07bzbsz~C~SRJAWBANam0cFkQPWxAOaxYFBhQLcE10AxgEfX4dURxAEk0UVgMdByMffwseUGAEGFQeQxMcElABTAVwGHYHTQJqbR4UHkQWaB4VVkJeFgpQGhBHQxZbEQgOBFcABgMGAQ8HBQINBlcEEBgSQwRXGwIQFUZEQEFUQlcSHBZHBFcQDhJSB0dNTEYUU1EWGRBHVV4SDmtYGgYFHAZNABUJHlRvHBZfWBULARwWUxIUCBYJBVhQClwCVQsGAgYKUVNUVQ0IUQYLDQhRAlUBDABWVxIYF1xHEwoSXWAJWVxREhhDRxsCA1cEBgYDBQQFAQICA00UWF8SDkMKWwtXAwUIUAZQBgEEA1FTAFQGUgUFBwEBXlRWB1QBVAMEBwlWBgcDFB4WVkQDEQMaXypBQUxsBnpcelJ3YyRfZlVeXldDAGkQTRBeQhcIFXBAQFhVQXVdWUBBFVZLFBIoXFMaFx4VX1FGFgpDBwQMAgVRERUaQQJAEg5uCgMFHAMNADwaEEZfFltoG1FiCV1eUQQBGwMSHBZZLmUQGBIFVR0PGh5DAwEaBBwDExwSBQZZBAMEEhhDClsLVwMFCFAGUAYBBANRUwBUBlIFBQcBAV5UVgdUAVQDBAcJVgYHAxQeFlEWPB8bUV0AEAoWU1RRV1ZWQERDGhBVWhZbEUwaHkNRWRYPEEAFHgAaBUMaEFdWaxcRAxoCUBAcFldWFQsSQlVeBVlfCQNZVGJNeXAiEBwWWFgVC2sBGABNBm8YElYNXF4aCEMDBgIDAAEGAwMMBlMOTAVhZRcEYAhFEEFEeXF4elReXFxmG3dLenEJXB1fbUoxZAJiA2ZiRWpqBwkqYnAMZFA1aWltWFQLfmZgSnxlQGd1AFAPa3FDfDdKVQFwI2djcU8KTnpicVd5MA5kcEkAJHgJAEguSABNcXMDVHNaDEQ0fkVyfV0KZ3F8AiBkXHV6dWJFcX9ncVdjY0dzbhtcTGxYMAd+cQ1/ellhXHkABm9/XX1iMGJeT1ckQnF5YgYAZ2RYY3YlYwJ4Wm1VfGttYSdfQEx3c1QMHgcHBQJVVFFSShgfCEZMH3BOZ118ZGZidXl1N2NxfFdxMntBbmMkRVtmYlp+VXVcTHEkZ11gdUxRYGtAYyt0cV9yc2ZWYndMdjJgZUx2cQZAanljKGB4dnt2ckZfYnJyWWNlcXlmIlt8cHoWd3R1Qn9keWJ6ZXUGXWFxAENcTQlcRwNZBloXHhVcQ1cWCkMUHhZIVxMRAxoCUE5IDUlIW1ZXV0ZXUUxYWBJJ~1uk1y6y","actionType":"0","inviteId":"%s"}`, invited)

			req := httplib.Post(url)
			random := browser.Random()
			req.Param("clientVersion", "-1")
			req.Param("appid", "signed_wh5")
			req.Param("functionId", "promote_pk_joinGroup")
			req.Param("body", body)
			req.Header("User-Agent", random)
			req.Header("Accept", "application/json, text/plain, */*")
			req.Header("Connection", "keep-alive")
			req.Header("Accept-Language", "zh-cn")
			req.Header("Accept-Encoding", "gzip, deflate, br")
			req.Header("Origin", "https://bunearth.m.jd.com")
			req.Header("Cookie", cookie)
			s, _ := req.String()
			bizCode, _ := jsonparser.GetInt([]byte(s), "data", "bizCode")
			if bizCode == 0 {
				k++
				logs.Info("助力成功")
				ck.Update(Dig, False)
				logs.Info(s)
			} else {
				logs.Info("助力失败")
				logs.Info(s)
				//你已经有团队了
				if strings.Contains(s, "该团队已经满员了") {
					return true
				} else if strings.Contains(s, "火爆") {
					ck.Update(Dig, False)
				} else if strings.Contains(s, "已结束") {
					return false
				} else if strings.Contains(s, "你已经有团队了") {
					ck.Update(Dig, False)
				} else {
					ck.Update(Dig, s)
				}
			}
		}
	}
	return false
}

func starttyt(red string) (num int, f bool) {
	k := 0
	//cks := GetJdCookies(func(sb *gorm.DB) *gorm.DB {
	//	return sb.Where(fmt.Sprintf("%s != ? and %s = ? ORDER BY RAND()", Tyt, Available), False, True)
	//})
	cks := []JdCookie{}
	db.Where(fmt.Sprintf("%s = 'true' and %s = 'true'", Tyt, Available)).Order("RAND()").Find(&cks)
	logs.Info(len(cks))
	if len(cks) < 50 {
		(&JdCookie{}).Push("推一推账号不足  注意补单")
		//return k, false
	}
	for _, ck := range cks {
		time.Sleep(time.Second * 10)
		logs.Info(ck.PtPin)
		cookie := "pt_key=" + ck.PtKey + ";pt_pin=" + ck.PtPin + ";"
		sprintf := fmt.Sprintf(`https://api.m.jd.com/?functionId=helpCoinDozer&appid=station-soa-h5&client=H5&clientVersion=1.0.0&t=1641900500241&body={"actId":"3075b6eab065464dad1c4042d345ac97","channel":"coin_dozer","referer":"-1","frontendInitStatus":"s","packetId":"%s","helperStatus":"0"}&_ste=1`, red)
		req := httplib.Post(sprintf)
		random := browser.Random()
		req.Header("User-Agent", random)
		req.Header("Host", "api.m.jd.com")
		req.Header("Accept", "application/json, text/plain, */*")
		req.Header("Origin", "https://pushgold.jd.com")
		req.Header("Cookie", cookie)
		data, _ := req.String()
		code, _ := jsonparser.GetInt([]byte(data), "code")
		logs.Info(data)
		if code == 0 {
			k++
			logs.Info(jsonparser.GetString([]byte(data), "data", "amount"))
		} else {
			if strings.Contains(data, "完成") {
				logs.Info("返回完成")
				return k, true
			} else if strings.Contains(data, "帮砍机会已用完") {
				ck.Update(Tyt, False)
			} else if strings.Contains(data, "火爆") {
				ck.Update(Tyt, False)
			} else if strings.Contains(data, "帮砍排队") {
				return k, false
			} else if strings.Contains(data, "need") {
				ck.Update(Tyt, "need verity")
			} else if strings.Contains(data, "未登录") {
				CookieOK(&ck)
				go func() {
					Save <- &JdCookie{}
				}()
			} else {
				getString, _ := jsonparser.GetString([]byte(data), "msg")
				ck.Update(Tyt, getString)
				logs.Info(getString)
			}
		}
	}
	(&JdCookie{}).Push(fmt.Sprintf("补单：%s", red))
	return k, false
}

func getMd5String(b []byte) string {
	return fmt.Sprintf("%x", md5.Sum(b))
}

func getScKey(ck string) (key string) {
	url := "https://api.m.jd.com/client.action?functionId=promote_getHomeData"
	req := httplib.Get(url)
	random := browser.Random()
	req.Param("clientVersion", "-1")
	req.Param("functionId", "promote_pk_getHomeData")
	req.Param("appid", "signed_wh5")
	req.Header("User-Agent", random)
	req.Header("Host", "api.m.jd.com")
	req.Header("Accept", "application/json, text/plain, */*")
	req.Header("Connection", "keep-alive")
	req.Header("Accept-Language", "zh-cn")
	req.Header("Accept-Encoding", "gzip, deflate, br")
	req.Header("Origin", "https://bunearth.m.jd.com")
	req.Header("Cookie", ck)
	data, _ := req.String()
	if strings.Contains(data, "secretp") {
		index := strings.Index(data, "\"secretp\":") + 11
		i := strings.Index(data, "shareCopywriting") - 3
		s := data[index:i]
		return s
	}
	return ""
}

var mu sync.Mutex

func rundyj(sender *Sender, code string) {
	for {
		for mu.TryLock() {
			pz++
			no := pzlist[code]
			get := httplib.Get(fmt.Sprintf("http://192.168.195.40:8066/api/dyj?shareId=%s", code)).SetTimeout(time.Duration(500)*time.Second, time.Duration(500)*time.Second)
			s, _ := get.Bytes()
			logs.Info(string(s))
			getInt, _ := jsonparser.GetInt(s, "code")
			if getInt == 200 {
				sender.Reply(fmt.Sprintf("订单编号：%d,邀请码:%s已完成", no, code))
			} else if getInt == 100 {
				sender.Reply(fmt.Sprintf("订单编号：%d,地址:%s,等待管理员通知重发", no, fmt.Sprintf("https://wqs.jd.com/sns/202210/20/make-money-shop/bridge.html?type=sign&activeId=63526d8f5fe613a6adb48f03&shareId=%s&bridgeType=sign&sharefromapp=jdltapp&channel=superjd-m-makemoneyking-hudong&utm_user=plusmember&ad_od=share&utm_source=androidapp&utm_medium=appshare&utm_campaign=t_335139774&utm_term=Wxfriends", code)))
			} else {
				sender.Reply(fmt.Sprintf("订单编号：%d,邀请异常，%s,请联系管理员", no, fmt.Sprintf("https://wqs.jd.com/sns/202210/20/make-money-shop/bridge.html?type=sign&activeId=63526d8f5fe613a6adb48f03&shareId=%s&bridgeType=sign&sharefromapp=jdltapp&channel=superjd-m-makemoneyking-hudong&utm_user=plusmember&ad_od=share&utm_source=androidapp&utm_medium=appshare&utm_campaign=t_335139774&utm_term=Wxfriends", code)))
			}
			pz--
			mu.Unlock()
			return
		}
	}
}

func getShare(ck string) []byte {
	req := httplib.Get("https://api.m.jd.com/api?g_ty=h5&g_tk=&appCode=msc588d6d5&body=%7B%22activeId%22%3A%2263526d8f5fe613a6adb48f03%22%2C%22isFirst%22%3A0%2C%22operType%22%3A1%7D&appid=jdlt_h5&client=jxh5&functionId=makemoneyshop_home&clientVersion=1.2.5")
	random := browser.Random()
	req.Header("User-Agent", random)
	req.Header("Host", "api.m.jd.com")
	req.Header("Accept", "application/json, text/plain, */*")
	req.Header("Connection", "keep-alive")
	req.Header("Accept-Language", "zh-cn")
	req.Header("Accept-Encoding", "gzip, deflate, br")
	req.Header("Origin", "https://wqs.jd.com")
	req.Header("Cookie", ck)
	bytes, _ := req.Bytes()
	return bytes

}

func getmoney(ck string) {
	for i := 0; i < 20; i++ {
		time.Sleep(time.Second * time.Duration(3))
		req := httplib.Get("https://wq.jd.com/newtasksys/newtasksys_front/Award?g_ty=h5&g_tk=&appCode=msc588d6d5&__t=1671472063174&source=makemoneyshop&isSecurity=true&taskId=3533&bizCode=makemoneyshop&_stk=__t%2CbizCode%2CisSecurity%2Csource%2CtaskId&_ste=1&h5st=20221220014743179%3B6897072897615020%3Bd06f1%3Btk02w82a31b4c18nga3Pz1Kz96NMEr3qiA7Sq2jBGN3TTsD5TNtGiZpnMQYLUDsBinJAAOCNpDJrs6lakAdMForE53CQ%3Bea1ef35ef9bf605358bd2e17db13dd12beedfee60207bdb504dc5e50db351aff%3B3.1%3B1671472063179%3B62f4d401ae05799f14989d31956d3c5f09633e3696c2d8f838f2f363c4fcd57aac5270f01d53f14fa132771b2ca2231d02ed6f7b7db0024e9cb202a6780fc384bd416eff2590a9d13da456a774455c08157834bb91f3bdecc83ba0b720cb0a55fe184c27e70f4601ce9ad75cda15667fc1bfc38db3622e572b77dfa341d713780cc4b8d74f5892500b744858073ed8ed45f2cfa685d2a838af862f0361dae479fb2a5830375c52a9b627eb8a14f180fc&sceneval=2&callback=__jsonp1671472045318")
		req.Header("User-Agent", "jdltapp;android;4.5.0;;;appBuild/2370;ef/1;ep/%7B%22hdid%22%3A%22JM9F1ywUPwflvMIpYPok0tt5k9kW4ArJEU3lfLhxBqw%3D%22%2C%22ts%22%3A1671472044033%2C%22ridx%22%3A-1%2C%22cipher%22%3A%7B%22sv%22%3A%22CJC%3D%22%2C%22ad%22%3A%22YJTvEQCnEJvvY2DtEWZrDG%3D%3D%22%2C%22od%22%3A%22EJC1ENU0YJU2ZQVsDJqyZtU3CWC4DWU3CtC3EWC5CwSmEJSnZJKmYWHtYJHwYwPvDJPsCwUzYJDtDzO1Y2VvYG%3D%3D%22%2C%22ov%22%3A%22CzC%3D%22%2C%22ud%22%3A%22YJTvEQCnEJvvY2DtEWZrDG%3D%3D%22%7D%2C%22ciphertype%22%3A5%2C%22version%22%3A%221.2.0%22%2C%22appname%22%3A%22com.jd.jdlite%22%7D;Mozilla/5.0 (Linux; Android 13; V2218A Build/TP1A.220624.003; wv) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/107.0.5304.105 Mobile Safari/537.36")
		req.Header("Host", "wq.jd.com")
		req.Header("Sec-Fetch-Mode", "no-cors")
		req.Header("Content-Type", "text/html;")
		req.Header("Referer", "https://wqs.jd.com")
		req.Header("Cookie", ck)
		bytes, _ := req.Bytes()
		logs.Info(string(bytes))
	}

}

func runpz(sender *Sender, code string) {
	for {
		time.Sleep(time.Duration(rand.Intn(60)))
		if pz < 3 {
			pz++
			num, f := startpz(code)
			no := pzlist[code]

			if f {
				sender.Reply(fmt.Sprintf("订单编号：%d,膨胀结束共用:%d个账号", no, num))
			} else {
				sender.Reply(fmt.Sprintf("订单编号：%d,膨胀异常，请联系群主，或自行检查", no))
			}
			pz--
			return
		}
	}
}

func startpz(invited string) (num int, flag bool) {
	logs.Info("开始膨胀助力")
	k := 0
	cks := GetJdCookies()
	db.Where(fmt.Sprintf("%s = 'true' and %s = 'true'", Dig, Available)).Order("RAND()").Find(&cks)
	for _, ck := range cks {
		time.Sleep(time.Second * time.Duration(3))
		cookie := "pt_key=" + ck.PtKey + ";pt_pin=" + ck.PtPin + ";"
		sc := getScKey(cookie)
		logs.Info(cookie)
		logs.Info(sc)
		if sc != "" {
			//https://wbbny.m.jd.com/pb/013349910/3rFiv8Sdkn7BPhk8Pw8xrgMWH6mT/index.html?babelChannel=shouyefuceng&shareType=expandHelp&inviteId=PKASTT0225KkcRkpP9VPQdhz9lf9cJgCTdXn4aRzTQjeQOc&mpin=RnFtkWRRYTOMmdRP--txCYtZA7-VliccLeHN&from=sc
			url := "https://api.m.jd.com/client.action?functionId=promote_pk_collectPkExpandScore"
			body := fmt.Sprintf(`{"random":"3m5QtABC","log":"1666267341263~194Cl2EtLOeMDFqSmpmSzAxMQ==.W3xcUHlcfVlVclx+WRgBKBAYLQwBegJXNVtmKUoIRnsUVzVbNActOBsyM1IGBwAIJRgvIz4HJQB8Gl8aFHtaSnpaNA==.6c8730f2~C,1~22CD2662C9991565879D915A12D2A9A085468D19~07bzbsz~C~SRJAWBANam0cFkQPWxAOaxYFBhQLcE10AxgEfX4dURxAEk0UVgMdByMffwseUGAEGFQeQxMcElABTAVwGHYHTQJqbR4UHkQWaB4VVkJeFgpQGhBHQxZbEQgOBFcABgMGAQ8HBQINBlcEEBgSQwRXGwIQFUZEQEFUQlcSHBZHBFcQDhJSB0dNTEYUU1EWGRBHVV4SDmtYGgYFHAZNABUJHlRvHBZfWBULARwWUxIUCBYJBVhQClwCVQsGAgYKUVNUVQ0IUQYLDQhRAlUBDABWVxIYF1xHEwoSXWAJWVxREhhDRxsCA1cEBgYDBQQFAQICA00UWF8SDkMKWwtXAwUIUAZQBgEEA1FTAFQGUgUFBwEBXlRWB1QBVAMEBwlWBgcDFB4WVkQDEQMaXypBQUxsBnpcelJ3YyRfZlVeXldDAGkQTRBeQhcIFXBAQFhVQXVdWUBBFVZLFBIoXFMaFx4VX1FGFgpDBwQMAgVRERUaQQJAEg5uCgMFHAMNADwaEEZfFltoG1FiCV1eUQQBGwMSHBZZLmUQGBIFVR0PGh5DAwEaBBwDExwSBQZZBAMEEhhDClsLVwMFCFAGUAYBBANRUwBUBlIFBQcBAV5UVgdUAVQDBAcJVgYHAxQeFlEWPB8bUV0AEAoWU1RRV1ZWQERDGhBVWhZbEUwaHkNRWRYPEEAFHgAaBUMaEFdWaxcRAxoCUBAcFldWFQsSQlVeBVlfCQNZVGJNeXAiEBwWWFgVC2sBGABNBm8YElYNXF4aCEMDBgIDAAEGAwMMBlMOTAVhZRcEYAhFEEFEeXF4elReXFxmG3dLenEJXB1fbUoxZAJiA2ZiRWpqBwkqYnAMZFA1aWltWFQLfmZgSnxlQGd1AFAPa3FDfDdKVQFwI2djcU8KTnpicVd5MA5kcEkAJHgJAEguSABNcXMDVHNaDEQ0fkVyfV0KZ3F8AiBkXHV6dWJFcX9ncVdjY0dzbhtcTGxYMAd+cQ1/ellhXHkABm9/XX1iMGJeT1ckQnF5YgYAZ2RYY3YlYwJ4Wm1VfGttYSdfQEx3c1QMHgcHBQJVVFFSShgfCEZMH3BOZ118ZGZidXl1N2NxfFdxMntBbmMkRVtmYlp+VXVcTHEkZ11gdUxRYGtAYyt0cV9yc2ZWYndMdjJgZUx2cQZAanljKGB4dnt2ckZfYnJyWWNlcXlmIlt8cHoWd3R1Qn9keWJ6ZXUGXWFxAENcTQlcRwNZBloXHhVcQ1cWCkMUHhZIVxMRAxoCUE5IDUlIW1ZXV0ZXUUxYWBJJ~1uk1y6y","actionType":"0","inviteId":"%s"}`, invited)
			req := httplib.Post(url)
			random := browser.Random()
			req.Param("clientVersion", "-1")
			//req.Param("functionId", "promote_pk_getHomeData")
			req.Param("appid", "signed_wh5")
			req.Param("functionId", "promote_pk_collectPkExpandScore")
			req.Param("body", body)
			req.Header("User-Agent", random)
			req.Header("Accept", "application/json, text/plain, */*")
			req.Header("Connection", "keep-alive")
			req.Header("Accept-Language", "zh-cn")
			req.Header("Accept-Encoding", "gzip, deflate, br")
			req.Header("Origin", "https://wbbny.m.jd.com")
			req.Header("Cookie", cookie)
			s, _ := req.String()
			bizCode, _ := jsonparser.GetInt([]byte(s), "data", "bizCode")
			bizMsg, _ := jsonparser.GetString([]byte(s), "data", "bizMsg")
			if bizCode == 0 {
				k++
				logs.Info("助力成功")
				logs.Info(bizMsg)
			} else {
				logs.Info("助力失败")
				logs.Info(s)
				if strings.Contains(bizMsg, "TA已经获得足够的助力了") {
					return k, true
				} else if strings.Contains(bizMsg, "火爆") {
					ck.Update(Dig, False)
				} else if strings.Contains(bizMsg, "已结束") {
					return k, false
				} else if strings.Contains(bizMsg, "次数") {
					ck.Update(Dig, False)
				} else {
					ck.Update(Dig, bizMsg)
				}
			}
		} else {
			CookieOK(&ck)
			go func() {
				Save <- &JdCookie{}
			}()
		}
	}
	return k, false

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
