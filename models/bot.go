package models

import (
	"encoding/json"
	"fmt"
	browser "github.com/EDDYCJY/fake-useragent"
	"net/url"

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

var replies = map[string]string{}
var loginList = make(map[int]chan string)
var meituanList = make(map[int]chan string)
var ckList = make(map[int]chan string)

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
	time.Sleep(time.Second * time.Duration(rand.Intn(3)))
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
		sender.WxId = msgs[2].(string)
	} else {
		sender.UserID = msgs[2].(int)
	}

	if len(msgs) >= 4 && sender.Type != "wxg" {
		sender.ChatID = msgs[3].(int)
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

	if meituanList[sender.UserID] != nil {
		c2 := meituanList[sender.UserID]
		c2 <- msg
		return nil
	}
	if ckList[sender.UserID] != nil {
		c2 := ckList[sender.UserID]
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
					if codeSignals[i].Coin > 0 {
						RemCoin(sender.UserID, codeSignals[i].Coin)
					}

					return codeSignals[i].Handle(sender)
				}()
			}
		}
	}

	if Config.VIP {
		switch msg {
		default:

			//美团UUID识别
			{
				//识别 http://dpurl.cn/
				if strings.Contains(msg, "http://dpurl.cn/") {
					//提取出url
					re := regexp.MustCompile(`http[s]?://(?:[a-zA-Z]|[0-9]|[$-_@.&+]|[!*\\(\\),]|(?:%[0-9a-fA-F][0-9a-fA-F]))+`)
					match := re.FindString(msg)
					realUrl := Meituan_getRealUrl(match)
					//提取UUID
					uuid := Meituan_getUUID(realUrl)
					//UUID绑定 模糊绑定
					if uuid != "" {
						//从realUrl提取inviterGameNickName
						inviterGameNickName := regexp.MustCompile(`inviterGameNickName=(.*?)&`)
						inviterGameNickNameMatch := inviterGameNickName.FindStringSubmatch(realUrl)
						if len(inviterGameNickNameMatch) > 1 {
							re := regexp.MustCompile(`(.*?)\*\*\*`)
							pre_name := re.FindStringSubmatch(inviterGameNickNameMatch[1])
							if len(pre_name) > 1 {
								bind := Meituan_Bind(sender, pre_name[1], uuid)
								if !bind {
									//模糊匹配失败，进入手动匹配模式
									meiTuans := GetMeiTuan(sender)
									if len(meiTuans) > 0 {
										//进入队列
										msg := make(chan string)
										meituanList[sender.UserID] = msg
										sender.Contents = []string{uuid}
										go MeituanSelect(sender, msg, 3, meiTuans)

										msgs := []string{
											"请回复以下序列号指定账号绑定UUID:",
										}
										for i, tuan := range meiTuans {
											msgs = append(msgs, fmt.Sprintf("%d、%s", i, tuan.Nickname))
										}
										sender.Reply(strings.Join(msgs, "\n"))
									} else {
										return "查无美团账号"
									}
								}
							}
						}
					}
				}
			}

			//返利识别
			{
				matched, _ := regexp.MatchString("^https://item(.m)?.jd.com/(product/)?([0-9]+).html", msg)
				b2 := matched || strings.Contains(msg, "https://u.jd.com/")
				if b2 {
					return Get_powerful_link(msg)
				}
			}

			//校验卡密
			{
				if strings.HasPrefix(msg, "XDD") {
					return useKey(msg, sender.UserID)
				}
			}

			//校验美团
			{
				if strings.HasPrefix(msg, "Ag") {
					UpLine(msg, sender)
				}
			}

			//美团登录
			{
				if strings.Contains(msg, "http://meishi.meituan.com/i/") || strings.Contains(msg, "https://i.meituan.com/") {
					//http://meishi.meituan.com/i/?ci=290&stid_b=1&cevent=imt%2Fhomepage%2Fcategory1%2F1&userId=87393719&token=AgEZKZO6f0O42_Yv8fH8iQLjfGnvuXs0z-WJlvqLj6whocvwVMm4IjBX-POW0mr-FMVynKz1PNO4xAAAAACpGQAAfzgL2jpMT-rFhkb51bFwZ2f1aiXNK7cetPz3H4_mWyifxiIRo_Tm_NXS1YW4zeby
					// 解析URL
					if sender.Type == "qq" || sender.Type == "qqg" {
						msg = strings.ReplaceAll(msg, "&amp;", "&")

					}
					parsedURL, err := url.Parse(msg)
					if err != nil {
						fmt.Println("解析URL出错:", err)
						return fmt.Sprintf("解析URL出错:%s", err)
					}
					// 获取指定参数的值
					token := parsedURL.Query().Get("token")
					UpLine(token, sender)
				}
			}

			//查询美团详细
			{
				if msg == "查询美团" {
					meiTuans := GetMeiTuan(sender)
					if len(meiTuans) > 0 {
						for _, meituan := range meiTuans {
							sender.Reply(meituan.Query())
						}
					} else {
						return "查无美团账号"
					}
				}
			}

			//美团50
			{
				if msg == "美团50" {
					meiTuans := GetMeiTuan(sender)
					if len(meiTuans) > 0 {

						//进入队列
						msg := make(chan string)
						meituanList[sender.UserID] = msg
						go MeituanSelect(sender, msg, 1, meiTuans)

						msgs := []string{
							"请回复以下序列号指定账号运行任务:",
						}
						for i, tuan := range meiTuans {
							msgs = append(msgs, fmt.Sprintf("%d、%s", i, tuan.Nickname))
						}
						sender.Reply(strings.Join(msgs, "\n"))
					} else {
						return "查无美团账号"
					}
				}
			}

			//美团领卷
			{
				if msg == "美团领卷" || msg == "美团领券" {
					//return "今日接口维护，请下午重试"
					meiTuans := GetMeiTuan(sender)
					if len(meiTuans) > 0 {

						//进入队列
						msg := make(chan string)
						meituanList[sender.UserID] = msg

						go MeituanSelect(sender, msg, 2, meiTuans)
						msgs := []string{
							"请回复以下序列号指定账号运行任务:",
						}
						for i, tuan := range meiTuans {
							msgs = append(msgs, fmt.Sprintf("%d、%s", i, tuan.Nickname))
						}
						sender.Reply(strings.Join(msgs, "\n"))
					} else {
						return "查无美团账号"
					}
				}
			}

			//口令转换
			{
				if strings.Contains(msg, "口令") {
					sender.Reply(KLtoLJ(msg))
					sender.Reply(LJtoKL("https://lzkj-isv.isvjcloud.com/lzclient/cjwx/common/openJDApp.html?actlink=openapp.jdmobile://virtual?params={\"category\":\"jump\",\"des\":\"scanLogin\",\"key\":\"AAEAIC7o7uvtQDO6vdYl4liag5G4fngqZbK2Vt83LyAbnmhF\",\"sourceType\":\"JSHOP_SOURCE_TYPE\",\"sourceValue\":\"JSHOP_SOURCE_VALUE\",\"M_sourceFrom\":\"mxz\",\"msf_type\":\"auto\"}"))
				}
			}

			//运行
			{
				if strings.Contains(msg, "运行") {
					lj := KLtoLJ(msg)
					msg = lj
				}
			}

			//识别登录
			{
				if msg == "登录" || msg == "登陆" {
					msg := make(chan string)
					loginList[sender.UserID] = msg
					go LoginSelect(sender, msg)

					msgs := []string{
						fmt.Sprintf("请选择登录渠道:"),
						fmt.Sprintf("1:扫码登录"),
						fmt.Sprintf("2:短信登录"),
					}
					msgs = append(msgs, "如需退出请回复'q'退出登录流程")
					sender.Reply(strings.Join(msgs, "\n"))
				}
			}

		}
	}

	switch msg {
	default:
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

func AutoCollection(autocollect map[string]string) {
	type AutoGenerated1 struct {
		Token      string `json:"token"`
		API        string `json:"api"`
		RobotWxid  string `json:"robot_wxid"`
		ToWxid     string `json:"from_wxid"`
		Money      string `json:"money"`
		PayerId    string `json:"payer_pay_id"`
		ReceiverId string `json:"receiver_pay_id"`
		PayType    string `json:"paysubtype"`
	}

	req := httplib.Post(Config.Wx.Url)
	reply := &AutoGenerated1{
		Token:      Config.Wx.Token,
		API:        "AccepteTransfer",
		RobotWxid:  Config.Wx.Robotid,
		ToWxid:     autocollect["to_wxid"],
		Money:      autocollect["money"],
		PayerId:    autocollect["payer_pay_id"],
		ReceiverId: autocollect["receiver_pay_id"],
		PayType:    "1",
	}

	random := browser.Random()
	req.Header("User-Agent", random)
	marshal, _ := json.Marshal(reply)
	logs.Info(string(marshal))
	req.Body(string(marshal))
	s, _ := req.String()
	logs.Info(s)
	//增加积分
	id := getWxId(autocollect["to_wxid"])
	if id != 0 {
		money, _ := strconv.ParseFloat(autocollect["money"], 64)
		logs.Info(money)
		AdddCoin(id, int(100.0*money))
	}
	SendWxMsg(autocollect["to_wxid"], fmt.Sprintf("充值成功！\n账户余额：%d\n发送“菜单”获取更多功能", GetCoin(id)))

}
