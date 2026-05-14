package models

import (
	"encoding/json"
	"fmt"
	browser "github.com/EDDYCJY/fake-useragent"
	"github.com/beego/beego/v2/client/httplib"
	"github.com/beego/beego/v2/core/logs"
	"gorm.io/gorm"
	"math/rand"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

var orderCounter int32

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

var pcodes = make(map[int]string)
var replies = map[string]string{}
var riskcodes = make(map[int]string)
var tytlist = make(map[string]int)
var tytno = 0
var tytnum = 0
var loginList = make(map[int]chan string)
var pzlist = make(map[string]int)
var pz = 0
var pzno = 0
var jbzllist = make(map[string]int)
var jbzl = 0
var jbzlno = 0
var zdlist = make(map[string]int)
var zd = 0
var zdno = 0
var meituanList = make(map[int]chan string)
var ElmList = make(map[int]chan string)
var ckList = make(map[int]chan string)

var inputList = make(map[int]chan string)

var AiinputList = make(map[int]chan string)



var handleMessage = func(msgs ...interface{}) interface{} {
	time.Sleep(time.Second * time.Duration(rand.Intn(3)))
	msg := msgs[0].(string)
	args := strings.Split(msg, " ")
	head := args[0]
	contents := args[1:]
	sender := &Sender{
		UserID:   0,
		Type:     msgs[1].(string),
		RawMessage: msg, 
		Contents: contents,
	}

	if msgs[1].(string) == "wx" || msgs[1].(string) == "wxg" {
		sender.UserID = GetWxid(msgs[2].(string))
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
	if ElmList[sender.UserID] != nil {
		c2 := ElmList[sender.UserID]
		c2 <- msg
		return nil
	}
	if ckList[sender.UserID] != nil {
		c2 := ckList[sender.UserID]
		c2 <- msg
		return nil
	}
	if inputList[sender.UserID] != nil {
		c2 := inputList[sender.UserID]
		c2 <- msg
		return nil
	}

	if AiinputList[sender.UserID] != nil {
		c2 := AiinputList[sender.UserID]
		c2 <- msg
		return nil
	}

if TryHandleSshMessage(sender) {
    return nil
}
	for i := range codeSignals {
		for j := range codeSignals[i].Command {
			if codeSignals[i].Command[j] == head {
				return func() interface{} {
					if codeSignals[i].Admin && !sender.IsAdmin {
						return nil
					}
					return codeSignals[i].Handle(sender)
				}()
			}
		}
	}

	if Config.VIP {
		switch msg {
		default:


			

			{
				matched, _ := regexp.MatchString("https://item(.m)?.jd.com/(product/)?([0-9]+).html", msg)
				b2 := matched || strings.Contains(msg, "https://u.jd.com/") || strings.Contains(msg, "https://3.cn/")
				if b2 {
					GetUniversal(sender, msg)
				}
			}

			{
				if strings.HasPrefix(msg, "DXWX") {
					return setWxId(msg, sender.WxId)
				}
			}

			{
				if strings.HasPrefix(msg, "XDD") {
					return useKey(msg, sender.UserID)
				}
			}
			{
				if strings.HasPrefix(msg, "ZSKM") {
					return use_ZSKey(msg, sender.UserID)
				}
			}

			{
				if strings.HasPrefix(msg, "Ag") {
					UpLine(msg, sender)
				}
			}
			{
				if strings.Contains(msg, "http://meishi.meituan.com/i/") || strings.Contains(msg, "https://i.meituan.com/mttouch/") {
					if sender.Type == "qq" || sender.Type == "qqg" {
						msg = strings.ReplaceAll(msg, "&amp;", "&")

					}
					parsedURL, err := url.Parse(msg)
					if err != nil {
						fmt.Println("解析URL出错:", err)
						return fmt.Sprintf("解析URL出错:%s", err)
					}
					token := parsedURL.Query().Get("token")
					UpLine(token, sender)
				}
			}

			{
				if strings.Contains(msg, "http://dpurl.cn/") {
					re := regexp.MustCompile(`http[s]?://(?:[a-zA-Z]|[0-9]|[$-_@.&+]|[!*\\(\\),]|(?:%[0-9a-fA-F][0-9a-fA-F]))+`)
					match := re.FindString(msg)
					realUrl := Meituan_getRealUrl(match)
					uuid := Meituan_getUUID(realUrl)
					if uuid != "" {
						inviterGameNickName := regexp.MustCompile(`inviterGameNickName=(.*?)&`)
						inviterGameNickNameMatch := inviterGameNickName.FindStringSubmatch(realUrl)
						if len(inviterGameNickNameMatch) > 1 {
							re := regexp.MustCompile(`(.*?)\*\*\*`)
							pre_name := re.FindStringSubmatch(inviterGameNickNameMatch[1])
							if len(pre_name) > 1 {
								bind := Meituan_Bind(sender, pre_name[1], uuid)
								if !bind {
									meiTuans := GetMeiTuan(sender)
									if len(meiTuans) > 0 {
										msg := make(chan string)
										meituanList[sender.UserID] = msg
										sender.Contents = []string{uuid}
										go MeituanSelect(sender, msg, 2, meiTuans)

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

			{
				if strings.Contains(msg, "https://i.meituan.com/c/") || strings.Contains(msg, "https://w.dianping.com/c/") {
					parsedURL, err := url.Parse(msg)
					if err != nil {
						fmt.Println("解析URL出错:", err)
						return fmt.Sprintf("解析URL出错:%s", err)
					}

					startIndex := strings.Index(parsedURL.String(), "0000")
					if startIndex < 0 {
						fmt.Println("未找到以0000开头的信息")
						return ""
					}

					endIndex := startIndex + 64
					urlStr := parsedURL.String()
					if endIndex > len(urlStr) {
						fmt.Println("0000索引64长度不够")
						return ""
					}

					uuid := urlStr[startIndex:endIndex]
					fmt.Println("提取的60位信息:", uuid)

					meiTuans := GetMeiTuan(sender)
					if len(meiTuans) > 0 {

						msg := make(chan string)
						meituanList[sender.UserID] = msg
						go MeituanUuid(sender, msg, uuid, meiTuans)

						msgs := []string{
							"请回复以下序列号更新账号uuid，如需退出请回复'q'退出登录流程：",
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

			{
				if msg == "查询美团" {
					meiTuans := GetMeiTuan(sender)
					if len(meiTuans) > 0 {
						for _, meituan := range meiTuans {
							sender.Reply(meituan.Query())
						}
					} else {
						return "查无美团账号，请使用 '美团登录'口令，按要求提交ck"
					}
				}
			}

			{
				if msg == "美团领券" || msg == "美团领卷" || msg == "美团领劵" {

					meiTuans := GetMeiTuan(sender)
					if len(meiTuans) > 0 {

						msg := make(chan string)
						meituanList[sender.UserID] = msg
						go MeituanSelect(sender, msg, 1, meiTuans)

						var msgs []string // 修改为[]string类型
						value := GetEnv("mtlq")
						jbcoin, _ := strconv.Atoi(value)

						message := fmt.Sprintf("请输入 ' 、' （顿号）前面的数字序号，选择后将扣除%d积分，输入小写的 q 退出流程", jbcoin)

						for i, tuan := range meiTuans {
							msgs = append(msgs, fmt.Sprintf("%d、%s", i, tuan.Nickname))
						}

						combinedMessage := fmt.Sprintf("%s\n%s", message, strings.Join(msgs, "\n"))
						sender.Reply(combinedMessage)

					} else {
						return "查无美团账号，请先发送 '美团登录'口令，按要求提交ck，随后在执行指令"
					}
				}
			}

			

			{
				if strings.Contains(msg, "我要转口令") {
					sender.Reply(KLtoLJ(msg))

				}
			}
			{
				if strings.Contains(msg, "口令") {
					sender.Reply("请求失败，返回错误码: 1002")

				}
			}
			

			{
				if strings.Contains(msg, "运行") {
					lj := KLtoLJ(msg)
					msg = lj
				}
			}

			//
			//
			//


	{
				if msg == "登录" || msg == "登陆" {
					msg := make(chan string)
					loginList[sender.UserID] = msg

					go LoginSelect(sender, msg)

					msgs := []string{
						"请选择登录渠道:",
					}

					if Config.QQID == 694738267 {
						sender.Reply("请回复【】里面的数字序号选择登录渠道:\n----------------------- \r\n 【1】 密码登录 （账号不掉线） \r\n 【2】 短信登录 （不定时掉线） \r\n 【3】 扫码登录 （不定时掉线） \r\n \n推荐使用密码登录，省心省力不错过任务，回复'q'退出登录流程\r\n \n 上车后请到京东-我的-支付设置，关闭小额免密，同时开启虚拟资产验密")
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
								nck.Updates(JdCookie{PtKey: ptKey, WeiXin: sender.WxId})
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

		{
			if strings.Contains(msg, "Bkfj1KTXRrTkmpwkQsmRf33WZbC") { // 判断信息中是否包含"挖宝"
				ss := regexp.MustCompile(`inviterId=([^&]+)(?:&|&amp;)inviterCode=([^&]+)`).FindStringSubmatch(msg)
				if len(ss) > 0 {
					inviterId := fmt.Sprintf("%s&%s", ss[1], ss[2])
					zlMatch := regexp.MustCompile(`助力(\d+)个`).FindStringSubmatch(msg)
					zl_NUM := 20
					if len(zlMatch) > 1 {
						zl_NUM, _ = strconv.Atoi(zlMatch[1])
					}
					if !sender.IsAdmin {
						coin := GetCoin(sender.UserID)
						if coin < 2000 {
							return fmt.Sprintf("挖宝不对积分少于%d的用户开放", 2000)
						}
						sender.Reply(fmt.Sprintf("即将开始，助力成功1头扣10积分"))
					} else {
					}
					queuePosition := atomic.LoadInt32(&taskCounter)
					atomic.AddInt32(&taskCounter, 1)
					orderID := atomic.AddInt32(&orderCounter, 1)
					sender.Reply(fmt.Sprintf("生成挖宝订单编号：%d \n助力码：%s \n前面还有%d个任务在等待", orderID, inviterId, queuePosition))
					envVars := map[string]string{
						"JD_FCWB_InviterId": inviterId,
						"JD_FCWB_NUM":       fmt.Sprintf("%d", zl_NUM),
						"DY_PROXY":          "http://api2.xkdaili.com/tools/XApi.ashx?apikey=XKF1492ACFB36F981667&qty=1&format=txt&split=0&iv=0&sign=fa46d34ef4c8da28ce2b2cfff4f03061",
					}
					taskQueue <- func() {
						sender.Reply(fmt.Sprintf("挖宝订单编号：%d 开始运行", orderID))
						result := run_fcwb_help_Task(sender, envVars, "jd_fcwb_help")
						sender.Reply(fmt.Sprintf("订单%d挖宝任务结果如下：\n%s", orderID, result))
					}
				}
			}
		}

		

		if strings.Contains(msg, "Bc9WX7MpCW7nW9QjZ5N3fFeJXMH") || strings.Contains(msg, "42HV4J3Q87B2xFQMJk81PCc1mEs3") {

			ss := regexp.MustCompile("inviteCode=([^&]+)").FindStringSubmatch(msg) // 提取助力码
			if len(ss) > 0 {
				inviterId := ss[1]

				 {
					id := sender.UserID
					var idType string
					if sender.Type == "tg" {
						idType = "Telegram"
					} else {
						idType = "QQ"
					}

					cks := GetJdCookies(func(sb *gorm.DB) *gorm.DB {
						return sb.Where(fmt.Sprintf("%s = ? and %s = ?", idType, "Available"), id, "True")
					})

					if len(cks) == 0 {
						sender.Reply("无法助力，您没有挂机京东账号或者账号已失效。上车请发送【密码登录】")
						return nil
					}

					currentTime := time.Now()
					currentHour := currentTime.Hour()
					
				      if currentHour < 9 || currentHour >= 23 {
		                   sender.Reply("只能在9:00到23:00期间执行助力任务。")
		                   return nil
		                }
					if IsUserInFarmNewTask(sender.UserID) {
						sender.Reply("你当前已有任务在排队或者正在执行，请等待完成后再提交新任务。")
						return nil
					}
				}

				if !sender.IsAdmin {
				value := GetEnv("xnczl")
				if value == "" {
					sender.Reply("群主未开启新农场种树")
					return nil
				}
			}
				go func() {

					defer delete(ckList, sender.UserID) // 确保在函数退出时清理通道
					msgChan := make(chan string)
					ckList[sender.UserID] = msgChan

					sender.Reply("请输入需要助力的次数，每天最大助力次数为11（当天超过11次会火爆）：")

					zl_NUM_Str := <-msgChan

					zl_NUM, err := strconv.Atoi(zl_NUM_Str)
					if err != nil || zl_NUM <= 0 || zl_NUM > 11 {
						sender.Reply("输入的次数不符合要求，请重新执行指令输入正确的次数，当前已退出程序")
						return
					}

					jbcoinStr := GetEnv("jd_farmnew_code_help") // #从环境变量获取的字符串
					jbcoin, err := strconv.Atoi(jbcoinStr)      // #将字符串转换为整数
					if err != nil {
						sender.Reply("获取积分值失败，请检查环境配置。")
						return
					}

					requiredCoins := zl_NUM * jbcoin

					coin := GetCoin(sender.UserID)
					if coin < requiredCoins {
						sender.Reply(fmt.Sprintf("积分不足，助力需要 %d 积分，当前积分 %d。", requiredCoins, coin))
						time.Sleep(time.Second * 1)
						sender.Reply("通过以下方法获取积分：\n1、签到、祈福\n2、直接私聊微信机器人转账1R=100积分\n3、复制网址：http://180.152.5.230:8005 到浏览器打开购买京东积分")
						return
					}

					if coin < 1000 {
						sender.Reply(fmt.Sprintf("总积分不能低于1000，当前积分 %d。", coin))
						time.Sleep(time.Second * 1)

						sender.Reply("通过以下方法获取积分：\n1、签到、祈福\n2、直接私聊微信机器人转账1R=100积分\n3、复制网址：http://180.152.5.230:8005 到浏览器打开购买京东积分")
						return
					}

					queuePosition := atomic.LoadInt32(&taskCounter)
					atomic.AddInt32(&taskCounter, 1)             // 增加任务计数
					orderID := atomic.AddInt32(&orderCounter, 1) // 生成订单编号
					SetFarmNewTaskStatus(sender.UserID, true)

					sender.Reply(fmt.Sprintf("生成新农场助力订单【%d】 \n助力码：%s \n前面还有%d个任务在等待", orderID, inviterId, queuePosition))
					sender.Reply(fmt.Sprintf("即将开始，助力成功1头扣%d积分，本次助力需要 %d 积分。", jbcoin, requiredCoins))

					envs := map[string]string{
						"NEWFRUITCODES": inviterId,
						"FRUIT_HELPNUM": fmt.Sprintf("%d", zl_NUM),
						"FRUIT_HOTNUM" : "4",

					}
					taskQueue <- func() {
						sender.Reply(fmt.Sprintf("新农场助力订单%d开始运行", orderID))
						result := run_fcwb_help_Task(sender, envs, "jd_farmnew_code_help")
						remainingTasks := atomic.LoadInt32(&taskCounter) - 1
						(&JdCookie{}).Push(fmt.Sprintf("===管理员通知===\n用户：%d的订单%d新农场助力结果如下：\n\n%s\n\n----------------------------\n后面一共还有%d个任务在等待执行", sender.UserID, orderID, result, remainingTasks))
						sender.Reply(fmt.Sprintf("订单【%d】新农场助力任务结果如下：\n\n%s \n\n-------------------------------\n你后面还有%d个任务在等待执行", orderID, result, remainingTasks))
						SetFarmNewTaskStatus(sender.UserID, false)
					}
				}()
			}
		}

		{
			if strings.Contains(msg, "4SJgvbvkFsTLRLhZqGU5ASjt1ahB") { // 判断信息中是否包含"心动乐园助力"
				ss := regexp.MustCompile(`wegameInviteId=([^&]+)`).FindStringSubmatch(msg) // 修复正则表达式引号问题
				zlMatch := regexp.MustCompile(`助力(\d+)[^0-9].*`).FindStringSubmatch(msg)
				zl_NUM := 3
				if len(zlMatch) > 1 {
					zl_NUM, _ = strconv.Atoi(zlMatch[1])
				}

				if len(ss) > 0 {
					inviterId := ss[1]
					if !sender.IsAdmin {
						value := GetEnv("gfjd")
						if value == "" {
							sender.Reply("助力瓜分100京豆已关闭，等群主开启")
							return nil
						}
					}

					coin := GetCoin(sender.UserID)

					jbcoin := GetEnv("jd_zlyhl")
					var requiredCoins int
					if jbcoin == "" {
						jbcoin = "20"
					}
					jbcoinValue, _ := strconv.Atoi(jbcoin)
					requiredCoins = zl_NUM * jbcoinValue

					if coin < 100 || requiredCoins > coin {
						return fmt.Sprintf("积分不足，总积分不能低于100，当前积分 %d，助力需要 %d 积分。", coin, requiredCoins)
					}

					sender.Reply(fmt.Sprintf("即将开始，助力成功1头扣%d", jbcoinValue))
					queuePosition := atomic.LoadInt32(&taskCounter)
					atomic.AddInt32(&taskCounter, 1)
					orderID := atomic.AddInt32(&orderCounter, 1)
					sender.Reply(fmt.Sprintf("助力瓜分100京豆订单%d \n助力码：%s \n预期助力%d个\n前面还有%d个任务在等待", orderID, inviterId, zl_NUM, queuePosition))

					envs := map[string]string{
						"jd_zlyhl_code":       inviterId,
						"jd_zlyhl_num":        fmt.Sprintf("%d", zl_NUM),
						"PRO_API_PROXY_URL":   "http://api2.xkdaili.com/tools/XApi.ashx?apikey=XKF1492ACFB36F981667&qty=1&format=txt&split=0&iv=0&sign=fa46d34ef4c8da28ce2b2cfff4f03061",
						"PRO_PROXY_WHITELIST": "jd",
					}

					taskQueue <- func() {
						sender.Reply(fmt.Sprintf("助力瓜分100京豆订单%d开始运行", orderID))
						result := run_fcwb_help_Task1(sender, envs, "jd_zlyhl")
						(&JdCookie{}).Push(fmt.Sprintf("===管理员通知===\n用户：%d的订单%d助力瓜分100京豆结果如下：\n%s", sender.UserID, orderID, result))
						sender.Reply(fmt.Sprintf("订单%d助力瓜分100京豆任务结果如下：\n%s", orderID, result))
					}
				}
			}
		}


if strings.Contains(msg, "B2Y13x641hwWfpsoRenCzfbz4jR") { // 判断信息中是否包含"赚赚记助力"
    ss := regexp.MustCompile(`inviterId=([^&]+)`).FindStringSubmatch(msg) // 提取助力码
    if len(ss) > 0 {
        inviterId := ss[1]

        if !sender.IsAdmin {
            value := GetEnv("zzzl")
            if value == "" {
                sender.Reply("赚赚助力活动没水，等有水群主会开启")
                return nil
            }
        }

        go func() {
            defer delete(ckList, sender.UserID) // 函数退出时清理通道
            msgChan := make(chan string)
            ckList[sender.UserID] = msgChan


            sender.Reply(fmt.Sprintf("您当前需要助力的助力码是：%s\n\n请输入需要助力的次数：", inviterId))

            zl_NUM_Str := <-msgChan

            zl_NUM, err := strconv.Atoi(zl_NUM_Str)
            if err != nil || zl_NUM <= 0 || zl_NUM > 400 {
                sender.Reply("输入的次数不符合要求，请重新执行指令输入正确的次数，当前已退出程序")
                return
            }

            if !sender.IsAdmin {
                jbcoinStr := GetEnv("jd_zzhb_code_help") // 赚赚助力单头积分环境变量
                jbcoin, err := strconv.Atoi(jbcoinStr)
                if err != nil {
                    sender.Reply("获取积分值失败，请检查环境配置。")
                    return
                }
                requiredCoins := zl_NUM * jbcoin

                coin := GetCoin(sender.UserID)
                if coin < requiredCoins {
                    sender.Reply(fmt.Sprintf("积分不足，助力需要 %d 积分，当前积分 %d。", requiredCoins, coin))
                    time.Sleep(time.Second * 1)
                    sender.Reply("通过以下方法获取积分：\n1、签到、祈福\n2、直接私聊微信机器人转账1R=100积分\n3、复制网址：http://180.152.5.230:8005 到浏览器打开购买京东积分")
                    return
                }

                if coin < 500 {
                    sender.Reply(fmt.Sprintf("总积分不能低于500，当前积分 %d。", coin))
                    time.Sleep(time.Second * 1)
                    sender.Reply("通过以下方法获取积分：\n1、签到、祈福\n2、直接私聊微信机器人转账1R=100积分\n3、复制网址：http://180.152.5.230:8005 到浏览器打开购买京东积分")
                    return
                }
            }

            queuePosition := atomic.LoadInt32(&taskCounter)
            atomic.AddInt32(&taskCounter, 1)
            orderID := atomic.AddInt32(&orderCounter, 1)
            sender.Reply(fmt.Sprintf("生成赚赚助力订单%d \n助力码：%s \n预期助力%d个\n前面还有%d个任务在等待", orderID, inviterId, zl_NUM, queuePosition))

            if sender.IsAdmin {
                sender.Reply("您是管理员，已跳过zzzl环境变量检查和积分限制")
            }

            envs := map[string]string{
                "ZZHB2CODE": inviterId,
                "JDZHB2NUM": fmt.Sprintf("%d", zl_NUM),
            }

            taskQueue <- func() {
                sender.Reply(fmt.Sprintf("赚赚助力订单%d开始运行", orderID))
                result := run_fcwb_help_Task_zz(sender, envs, "jd_zzhb_new_help")
                (&JdCookie{}).Push(fmt.Sprintf("===管理员通知===\n用户：%d的订单%d赚赚助力结果如下：\n%s", sender.UserID, orderID, result))
                sender.Reply(fmt.Sprintf("订单%d赚赚助力任务结果如下：\n%s", orderID, result))
            }
        }()
    }
}



		{
			if strings.Contains(msg, "3ABYwYuC87Dcx4gZYGKw6fqtE8WN") { // 判断信息中是否包含"3c记助力"


				ss := regexp.MustCompile(`inviterId=([^&]+)`).FindStringSubmatch(msg) // 修复正则表达式引号问题
				zlMatch := regexp.MustCompile(`助力(\d+)[^0-9].*`).FindStringSubmatch(msg)
				zl_NUM := 50
				if len(zlMatch) > 1 {
					zl_NUM, _ = strconv.Atoi(zlMatch[1])
				}
				if len(ss) > 0 {
					inviterId := ss[1]
					if !sender.IsAdmin {
						value := GetEnv("3czl")
						if value == "" {
							sender.Reply("3c助力活动没水，等有水群主会开启")
							return nil
						}
						coin := GetCoin(sender.UserID)

						requiredCoins := zl_NUM * 8

						if coin < 500 || requiredCoins > coin {
							return fmt.Sprintf("积分不足，总积分不能低于500，当前积分 %d，助力需要 %d 积分。", coin, requiredCoins)
						}

						sender.Reply(fmt.Sprintf("即将开始，助力成功1头扣8积分"))
					}

					queuePosition := atomic.LoadInt32(&taskCounter)
					atomic.AddInt32(&taskCounter, 1)
					orderID := atomic.AddInt32(&orderCounter, 1)
					sender.Reply(fmt.Sprintf("生成3c助力订单%d \n助力码：%s \n预期助力%d个\n前面还有%d个任务在等待", orderID, inviterId, zl_NUM, queuePosition))

					envs := map[string]string{
						"SMQCODE": inviterId,
						"SMQNUM":  fmt.Sprintf("%d", zl_NUM),
					}

					taskQueue <- func() {
						sender.Reply(fmt.Sprintf("3c助力订单%d开始运行", orderID))
						result := run_fcwb_help_Task(sender, envs, "jd_smq_help")
						(&JdCookie{}).Push(fmt.Sprintf("===管理员通知===\n用户：%d的订单%d3c助力结果如下：\n%s", sender.UserID, orderID, result))
						sender.Reply(fmt.Sprintf("订单%d3c助力任务结果如下：\n%s", orderID, result))
					}
				}
			}
		}

		if strings.Contains(msg, "【东东1农场】") { // 判断信息中是否包含"东东农场"
			value := GetEnv("ncxcx")
			if value == "" {
				return "脚本有问题，已关闭小程序助力"
			}

			ss := regexp.MustCompile(`inviteCode=([^&]+)`).FindStringSubmatch(msg)

			if len(ss) > 0 {
				inviterId := ss[1] // 们只提取了 inviteCode

				if !sender.IsAdmin {
					coin := GetCoin(sender.UserID)
					if coin < 100 {
						return fmt.Sprintf("小程序助力不对积分少于%d的用户开放，请发送【充值】，到网站购买", 100)
					}
					sender.Reply(fmt.Sprintf("即将开始，助力成功1头扣2积分"))
				} else {
				}

				queuePosition := atomic.LoadInt32(&taskCounter)
				atomic.AddInt32(&taskCounter, 1)
				orderID := atomic.AddInt32(&orderCounter, 1)
				sender.Reply(fmt.Sprintf("生成新农场小程序助力订单编号：%d \n助力码：%s \n前面还有%d个任务在等待", orderID, inviterId, queuePosition))

				envVars := map[string]string{
					"NEWFRUITCODES": inviterId,
					"DY_PROXY":      "http://api2.xkdaili.com/tools/XApi.ashx?apikey=XKF1492ACFB36F981667&qty=1&format=txt&split=0&iv=0&sign=fa46d34ef4c8da28ce2b2cfff4f03061",
				}

				taskQueue <- func() {
					sender.Reply(fmt.Sprintf("新农场小程序助力订单编号：%d 开始运行", orderID))
					result := run_ncxcx_help_Task(sender, envVars, "jd_farmshare")
					sender.Reply(fmt.Sprintf("订单%d新农场小程序助力任务结果如下：\n%s", orderID, result))
				}
			}
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
						tytlist[env[1]] = no
						go runtyt(sender, env[1])
					}
				}
			}
		}

	



						{
							if Config.QQID == 694738267 {
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


{
	if strings.Contains(msg, "pt_key") && strings.Contains(msg, "pt_pin") { // 双关键字前置判断，减少无效校验
		logs.Info(msg + "开始CK登录")

		chineseSymbolRegex := regexp.MustCompile(`[；，。：“”‘’（）【】、＝￥～]`)
		if chineseSymbolRegex.MatchString(msg) {
			sender.Reply("你提交的ck里面包含中文字符")
			return nil
		}

		dealMsg := regexp.MustCompile(`;( +)`).ReplaceAllString(msg, ";")

		ckRegex := regexp.MustCompile(`.*(pt_key=([^;]+);pt_pin=([^;]+);).*|.*(pt_pin=([^;]+);pt_key=([^;]+);).*`)
		match := ckRegex.FindStringSubmatch(dealMsg)
		if match == nil {
			sender.Reply("CK格式错误，正确的ck有效格式是：pt_key=xxxx;pt_pin=xxxx; 或者 pt_pin=xxxx;pt_key=xxxx;")
			return nil
		}

		var ptKey, ptPin string
		if match[2] != "" && match[3] != "" {
			ptKey = match[2]
			ptPin = match[3]
		} else if match[5] != "" && match[6] != "" {
			ptPin = match[5]
			ptKey = match[6]
		}

		if len(ptPin) == 0 || len(ptKey) == 0 {
			sender.Reply("CK格式错误，正确的ck有效格式是：pt_key=xxxx;pt_pin=xxxx; 或者 pt_pin=xxxx;pt_key=xxxx;")
			return nil
		}

		ck := JdCookie{
			PtKey: ptKey,
			PtPin: ptPin,
		}
		if CookieOK(&ck) {
			if sender.IsQQ() || sender.isWX() { // 保留原有isWX首字母小写写法
				ck.QQ = sender.UserID
			} else if sender.IsTG() {
				ck.Telegram = sender.UserID
			}
			if HasKey(ck.PtKey) {
				sender.Reply(fmt.Sprintf("重复提交"))
			} else {
				if nck, err := GetJdCookie(ck.PtPin); err == nil {
					nck.Updates(JdCookie{PtKey: ptKey, WeiXin: sender.WxId})
					msg := fmt.Sprintf("更新账号，%s", ck.PtPin)
					if sender.IsQQ() {
						ck.Update(QQ, ck.QQ)
						ck.Update(Available, True) // 完全保留原有大写True，不做任何修正
					}
					sender.Reply(fmt.Sprintf(msg))
					(&JdCookie{}).Push(msg)
					logs.Info(msg)
				} else {
					NewJdCookie(&ck)
					msg := fmt.Sprintf("添加账号，账号名:%s", ck.PtPin)
					if sender.IsQQ() {
						ck.Update(QQ, ck.QQ)
						ck.Update(Available, True) // 完全保留原有大写True，不做任何修正
					}
					sender.Reply(fmt.Sprintf(msg))
					sender.Reply(ck.Query())
					(&JdCookie{}).Push(msg)
					logs.Info(msg)
				}
			}
		} else {
			sender.Reply(fmt.Sprintf("无效的ck"))
		}
		go func() {
			Save <- &JdCookie{}
		}()
		return nil
	}
}

		if content, ok := GetReplyContent(msg); ok {
			replyStr, isString := content.(string)
			
			if isString && replyStr != "" {
				if (sender.Type == "wx" || sender.Type == "wxg") && strings.Contains(replyStr, "[CQ:image") {
					
					re := regexp.MustCompile(`\[CQ:image,file=([^\]]+)\]`)
					matches := re.FindStringSubmatch(replyStr)
					
					if len(matches) > 1 {
						imgUrl := matches[1]
						logs.Info("【微信图片适配】拦截 CQ 码，直接发送图片:", imgUrl)
						
						targetId := ""
						if sender.Type == "wxg" {
							targetId = sender.WxGroupId
						} else {
							targetId = sender.WxId
						}

						if targetId != "" {
							SendWxImg2(targetId, imgUrl)
						} else {
							logs.Error("发送失败：未获取到有效的微信接收者ID (Type:", sender.Type, ")")
						}
						
						return nil
					}
				}
				
				return content
			}
			return content
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
}




func AutoCollectionAndAddCoin(autocollect map[string]string) {
	AutoCollection(autocollect)
	id := GetWxid(autocollect["to_wxid"])
	rechargePoints := 0 // 将 rechargePoints 的声明移到这里，并初始化为 0
	if id != 0 {
		money, _ := strconv.ParseFloat(autocollect["money"], 64)
		rechargePoints = int(100.0 * money) // 更新 rechargePoints 的值
		logs.Info(money)
		AdddCoin(id, rechargePoints)
	}
	SendWxMsg(autocollect["to_wxid"], fmt.Sprintf("充值成功！充值积分：%d\n充值后账户余额：%d\n注意：没收到请联系群主\n发送“菜单”获取更多功能", rechargePoints, GetCoin(id)))
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
}

func setupOrderResetTask() {
	now := time.Now()
	nextMidnight := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
	duration := nextMidnight.Sub(now)

	time.AfterFunc(duration, func() {
		atomic.StoreInt32(&orderCounter, 0)
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			atomic.StoreInt32(&orderCounter, 0)
		}
	})
}


func LoadFarmNewTasks() map[string]bool {
	file := "userid_farmNew.json"
	m := make(map[string]bool)
	data, err := os.ReadFile(file)
	if err == nil {
		json.Unmarshal(data, &m)
	}
	return m
}

func SaveFarmNewTasks(m map[string]bool) {
	file := "userid_farmNew.json"
	data, _ := json.MarshalIndent(m, "", "  ")
	os.WriteFile(file, data, 0644)
}

func SetFarmNewTaskStatus(uid int, status bool) {

	m := LoadFarmNewTasks()
	m[strconv.Itoa(uid)] = status
	SaveFarmNewTasks(m)
}

func IsUserInFarmNewTask(uid int) bool {

	m := LoadFarmNewTasks()
	return m[strconv.Itoa(uid)]
}
