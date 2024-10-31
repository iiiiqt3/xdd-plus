package models

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	browser "github.com/EDDYCJY/fake-useragent"
	"github.com/beego/beego/v2/client/httplib"
	"math/rand"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/beego/beego/v2/core/logs"
	"github.com/google/uuid"
	"github.com/skip2/go-qrcode"
	"gorm.io/gorm"
)

var ua = "Mozilla/5.0 (iPhone; CPU iPhone OS 13_3_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/15E148 SP-engine/2.14.0 main%2F1.0 baiduboxapp/11.18.0.16 (Baidu; P2 13.3.1) NABar/0.0"

type CodeSignal struct {
	Command []string
	Admin   bool
	Coin    int
	Handle  func(sender *Sender) interface{}
}

type Sender struct {
	UserID            string
	ChatID            int
	GroupId           int
	WxGroupId         string
	Type              string
	Contents          []string
	MessageID         int
	Username          string
	IsAdmin           bool
	ReplySenderUserID int
}

type QQuery struct {
	Code int `json:"code"`
	Data struct {
		LSid          string `json:"lSid"`
		QqLoginQrcode struct {
			Bytes string `json:"bytes"`
			Sig   string `json:"sig"`
		} `json:"qqLoginQrcode"`
		RedirectURL string `json:"redirectUrl"`
		State       string `json:"state"`
		TempCookie  string `json:"tempCookie"`
	} `json:"data"`
	Message string `json:"message"`
}

func (sender *Sender) Reply(msg string) {
	switch sender.Type {
	case "tg":
		SendTgMsg(sender.UserID, msg)
	case "tgg":
		SendTggMsg(sender.ChatID, sender.UserID, msg, sender.MessageID, sender.Username)
	case "qq":
		if strings.Contains(msg, "账号昵称：") && Config.VIP && isOpenImg() {
			SendQQ(sender.UserID, strtoimg(msg))
		} else {
			SendQQ(sender.UserID, msg)
		}
	case "qqg":
		SendQQGroup(sender.ChatID, sender.UserID, msg)
	case "wx":
		SendWxMsg(sender.UserID, msg)
	case "wxg":
		SendWxGroupMsg(sender.UserID, sender.WxGroupId, msg)
	}
}

func (sender *Sender) SendImg(msg []byte) {
	switch sender.Type {
	case "qq":
		SendQQMsg(QQMessage{
			Action: "send_msg",
			QQMsg: struct {
				MessageType string `json:"message_type"`
				UserId      string `json:"user_id"`
				GroupID     int    `json:"group_id"`
				Message     string `json:"message"`
			}{
				UserId:  sender.UserID,
				GroupID: 0,
				Message: fmt.Sprintf("[CQ:image,file=base64://%s,type=show]", base64.StdEncoding.EncodeToString(msg)),
			},
			Echo: "",
		})
	case "qqg":
		SendQQMsg(QQMessage{
			Action: "send_msg",
			QQMsg: struct {
				MessageType string `json:"message_type"`
				UserId      string `json:"user_id"`
				GroupID     int    `json:"group_id"`
				Message     string `json:"message"`
			}{
				MessageType: "group",
				GroupID:     sender.ChatID,
				Message:     fmt.Sprintf("[CQ:at,qq=%s][CQ:image,file=base64://%s,type=show]", sender.UserID, base64.StdEncoding.EncodeToString(msg)),
			},
			Echo: "",
		})
	case "tg":
		SendTgImg(sender.UserID, msg)
	case "wx":
		SendWxImg(sender.UserID, msg)
	case "wxg":
		SendWxImg(sender.WxGroupId, msg)

	}

}

func (sender *Sender) JoinContens() string {
	return strings.Join(sender.Contents, " ")
}

func (sender *Sender) IsQQ() bool {
	return strings.Contains(sender.Type, "qq")
}

func (sender *Sender) isWX() bool {
	return strings.Contains(sender.Type, "wx")
}

func (sender *Sender) IsTG() bool {
	return strings.Contains(sender.Type, "tg")
}

func (sender *Sender) handleJdCookies(handle func(ck *JdCookie)) error {
	cks := GetJdCookies()
	a := sender.JoinContens()
	ok := false
	if !sender.IsAdmin || a == "" {
		for i := range cks {
			if strings.Contains(sender.Type, "qq") || strings.Contains(sender.Type, "wx") {
				if cks[i].UserId == sender.UserID {
					if !ok {
						ok = true
					}
					handle(&cks[i])
				}
			} else if strings.Contains(sender.Type, "tg") {
				if cks[i].UserId == sender.UserID {
					if !ok {
						ok = true
					}
					handle(&cks[i])
				}
			}
		}
		if !ok {
			if Config.Query != "" {
				sender.Reply(Config.Query)
				return errors.New(Config.Query)
			} else {
				if sender.Type == "wx" {
					sender.Reply("你尚未绑定🐶东账号，请对QQ机器人发送绑定微信获取Code绑定QQ数据，或者是新用户请直接发送登录。")
				} else {
					sender.Reply("你尚未绑定🐶东账号，请发送教程获取最新上车方法。")
				}
				return errors.New("你尚未绑定🐶东账号，请发送教程获取最新上车方法。")
			}
		}
	} else {
		cks = LimitJdCookie(cks, a)
		if len(cks) == 0 {
			sender.Reply("没有匹配的账号")
			return errors.New("没有匹配的账号")
		} else {
			for i := range cks {
				handle(&cks[i])
			}
		}
	}
	return nil
}

var codeSignals = []CodeSignal{

	//获取我的userid
	{
		Command: []string{"获取我的userid"},
		Handle: func(sender *Sender) interface{} {
			return sender.UserID
		},
	},

	// https://pp.iaka.cn/api/ajax.php?act=search&name=短剧名称 通过api写出搜剧代码，识别命令搜据，并解析空格后的剧名进行api查询并返回查询结果
	{
		Command: []string{"搜剧"},
		Handle: func(sender *Sender) interface{} {
			if len(sender.Contents) == 0 {
				return "请输入剧名"
			}
			name := sender.Contents[0]
			url := fmt.Sprintf("https://pp.iaka.cn/api/ajax.php?act=search&name=%s", url.QueryEscape(name))
			req := httplib.Get(url)
			req.Header("User-Agent", browser.Random())
			bytes, err := req.Bytes()
			if err != nil {
				return err.Error()
			}
			return string(bytes)
		},
	},

	//获取我的userid
	{
		Command: []string{"获取我的群Id"},
		Admin:   true,
		Handle: func(sender *Sender) interface{} {
			return sender.WxGroupId
		},
	},

	//拉人进微信群
	{
		Command: []string{"拉群"},
		Handle: func(sender *Sender) interface{} {
			if sender.IsAdmin && sender.Type == "wxg" {
				ExportEnv(&Env{
					Name:  "InviteWxGroupID",
					Value: sender.WxGroupId,
				})
				return "已将此群设为拉群目标"
			} else if sender.Type == "wx" {
				env := GetEnv("InviteWxGroupID")
				if env != "" {
					InviteGroup(sender.UserID, env)
					return nil
				} else {
					return "未设置拉群目标！"
				}
			} else {
				return "错误指令"
			}
		},
	},

	//监听微信群
	{
		Command: []string{"监听微信群"},
		Admin:   true,
		Handle: func(sender *Sender) interface{} {
			if !strings.Contains(Config.WXGroupID, sender.WxGroupId) {
				logs.Info(sender.WxGroupId)
				env := GetEnv("WxGroupID")
				if strings.Contains(env, sender.WxGroupId) {
					return "已在监听列表"
				} else {
					env1 := &Env{
						Name:  "WxGroupID",
						Value: env + "," + sender.WxGroupId,
					}
					ExportEnv(env1)
					initWX()
					return "监听成功"
				}
			} else {
				return "已在监听列表"
			}
		},
	},

	//取消监听
	{
		Command: []string{"取消监听"},
		Admin:   true,
		Handle: func(sender *Sender) interface{} {

			if strings.Contains(Config.WXGroupID, sender.WxGroupId) {
				env := GetEnv("WxGroupID")
				if !strings.Contains(env, sender.WxGroupId) {
					return "不在监听范围"
				} else {
					replace := strings.ReplaceAll(env, sender.WxGroupId, "")
					env1 := &Env{
						Name:  "WxGroupID",
						Value: replace,
					}
					ExportEnv(env1)
					initWX()
					return "取消监听成功"
				}
			} else {
				return "不在监听范围"
			}
		},
	},

	//短信登陆
	{
		Command: []string{"短信登录", "短信登陆"},
		Handle: func(sender *Sender) interface{} {
			c2 := make(chan string)
			smsList[sender.UserID] = c2
			sender.Reply("请输入手机号")
			go SmsSelect(sender, c2, "Nolan")
			return nil
		},
	},

	//停止助力
	{
		Command: []string{"停助力", "停止助力"},
		Handle: func(sender *Sender) interface{} {
			if sender.UserID == "995336676" || sender.IsAdmin {
				rsp := cmd(fmt.Sprintf(`bash stop.sh`), &Sender{})
				return rsp
			} else {
				sender.Reply("无权操作")
			}
			return nil
		},
	},

	//生成卡密
	{
		Command: []string{"生成卡密"},
		Admin:   true,
		Handle: func(sender *Sender) interface{} {
			if Config.VIP == true {
				contents := sender.Contents
				logs.Info(contents[0])
				num, _ := strconv.Atoi(contents[0])
				value, _ := strconv.Atoi(contents[1])
				return createKey(num, value)
			}
			return "非VIP用户"
		},
	},

	//状态
	{
		Command: []string{"status", "状态"},
		Admin:   true,
		Handle: func(sender *Sender) interface{} {
			return Count()
		},
	},

	//py执行口令
	{
		Command: []string{"跑"},
		Admin:   true,
		Handle: func(sender *Sender) interface{} {
			if len(sender.Contents) >= 1 {
				var head = sender.Contents[0]
				var args = strings.Join(sender.Contents[1:], " ")
				rsp := cmd(fmt.Sprintf(`python3 ./runcommand.py check "%s" "%s"`, head, args), &Sender{})
				sender.Reply(rsp)
				if strings.Index(rsp, "开始") >= 0 {
					rsp := cmd(fmt.Sprintf(`python3 ./runcommand.py run "%s" "%s"`, head, args), &Sender{})
					sender.Reply(rsp)
				}
			} else {
				sender.Reply("请配置开始信息")
			}
			return nil
		},
	},

	//R京东扫码
	{
		Command: []string{"R京东扫码"},
		Handle: func(sender *Sender) interface{} {
			RabbitGetJdQrImg(sender)
			return nil
		},
	},

	//N京东扫码
	{
		Command: []string{"N京东扫码"},
		Handle: func(sender *Sender) interface{} {
			NolanGetJdQrImg(sender)
			return nil
		},
	},

	//打卡
	{
		Command: []string{"sign", "打卡", "签到"},
		Handle: func(sender *Sender) interface{} {
			zero, _ := time.ParseInLocation("2006-01-02", time.Now().Local().Format("2006-01-02"), time.Local)
			var u User
			var ntime = time.Now()
			var first = false
			var total []int
			err := db.Where("number = ?", sender.UserID).First(&u).Error
			if err != nil {
				first = true
				u = User{
					Class:    sender.Type,
					UserId:   sender.UserID,
					Coin:     1,
					ActiveAt: ntime,
				}
				if err := db.Create(&u).Error; err != nil {
					return err.Error()
				}
			} else {
				if zero.Unix() > u.ActiveAt.Unix() {
					first = true
				} else {
					return fmt.Sprintf("你打过卡了，积分余额%d。", u.Coin)
				}
			}
			if first {
				db.Model(User{}).Select("count(id) as total").Where("active_at > ?", zero).Pluck("total", &total)
				coin := 1
				if total[0]%3 == 0 {
					coin = 2
				}
				if total[0]%13 == 0 {
					coin = 8
				}
				db.Model(&u).Updates(map[string]interface{}{
					"active_at": ntime,
					"coin":      gorm.Expr(fmt.Sprintf("coin+%d", coin)),
				})
				u.Coin += coin
				sender.Reply(fmt.Sprintf("你是打卡第%d人，奖励%d个积分，积分余额%d。", total[0]+1, coin, u.Coin))
				ReturnCoin(sender)
				return ""
			}
			return nil
		},
	},

	//清零
	{
		Command: []string{"清零"},
		Admin:   true,
		Handle: func(sender *Sender) interface{} {
			sender.handleJdCookies(func(ck *JdCookie) {
				ck.Update(Priority, 1)
			})
			sender.Reply("优先级已清零")
			return nil
		},
	},

	//更新优先级
	{
		Command: []string{"更新优先级", "更新车位"},
		Handle: func(sender *Sender) interface{} {
			coin := GetCoin(sender.UserID)
			t := time.Now()
			if t.Weekday().String() == "Monday" && int(t.Hour()) <= 10 {
				sender.handleJdCookies(func(ck *JdCookie) {
					ck.Update(Priority, coin)
				})
				sender.Reply("优先级已更新")
				ClearCoin(sender.UserID)
			} else {
				sender.Reply("你错过时间了呆瓜,下周一10点前再来吧.")
			}
			return nil
		},
	},

	//XDD专用还愿CK指令
	{
		Command: []string{"XDD专用还愿CK指令，慎用！"},
		Admin:   true,
		Handle: func(sender *Sender) interface{} {
			sender.Reply("开始还原CK，后悔请马上重启")
			Reduction()
			sender.Reply("已还原完成")
			return nil
		},
	},

	//备份CK
	{
		Command: []string{"备份CK"},
		Admin:   true,
		Handle: func(sender *Sender) interface{} {
			sender.Reply("开始备份CK")
			AutoBak()
			sender.Reply("备份完成")
			return nil
		},
	},

	//查询积分
	{
		Command: []string{"coin", "积分"},
		Handle: func(sender *Sender) interface{} {
			return fmt.Sprintf("积分:%d", GetCoin(sender.UserID))
		},
	},

	//授权
	{
		Command: []string{"授权"},
		Admin:   true,
		Handle: func(sender *Sender) interface{} {
			ctt := sender.JoinContens()
			auth := AddAuth(ctt)
			if auth {
				return "授权成功"
			} else {
				return "授权失败"
			}
		},
	},

	//取消授权
	{
		Command: []string{"取消授权"},
		Admin:   true,
		Handle: func(sender *Sender) interface{} {
			ctt := sender.JoinContens()
			auth := Remove(ctt)
			if auth {
				return "操作成功"
			} else {
				return "操作失败"
			}
		},
	},

	//开始检测
	{
		Command: []string{"开始检测"},
		Admin:   true,
		Handle: func(sender *Sender) interface{} {
			initCookie()
			return "检测完成"
		},
	},

	//检测美团
	{
		Command: []string{"美团检测"},
		Admin:   true,
		Handle: func(sender *Sender) interface{} {
			CheckMTList()
			return "检测完成"
		},
	},

	//更新
	//{
	//	Command: []string{"升级", "更新", "update", "upgrade"},
	//	Admin:   true,
	//	Handle: func(sender *Sender) interface{} {
	//		if err := Update(sender); err != nil {
	//			return err.Error()
	//		}
	//		sender.Reply("重启程序")
	//		Daemon()
	//		return nil
	//	},
	//},

	//重启
	{
		Command: []string{"重启", "reload", "restart", "reboot"},
		Admin:   true,
		Handle: func(sender *Sender) interface{} {
			sender.Reply("重启程序")
			Daemon()
			return nil
		},
	},

	//更新Rwskey
	{
		Command: []string{"Rwskey更新", "更新Rwskey"},
		Admin:   true,
		Handle: func(sender *Sender) interface{} {
			sender.Reply("更新所有账号")
			logs.Info("更新所有账号")
			UpdateRwskey()
			return nil
		},
	},

	//导出所有账号
	{
		Command: []string{"导出所有账号"},
		Admin:   true,
		Handle: func(sender *Sender) interface{} {
			var msgs []string
			cks := GetJdCookies(func(sb *gorm.DB) *gorm.DB {
				return sb.Where(fmt.Sprintf("%s >= ? and %s = ?", Priority, Available), 0, True)
			})
			for _, ck := range cks {
				msgs = append(msgs, fmt.Sprintf("pt_key=%s;pt_pin=%s;", ck.PtKey, ck.PtPin))
			}
			sender.Reply("导出所有账号")
			logs.Info("导出所有账号")
			f, err := os.OpenFile(ExecPath+"/scripts/jdCookie.txt", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0777)
			if err != nil {
				logs.Warn("创建jdCookie.txt失败，", err)
			}
			join := strings.Join(msgs, "\n")
			f.WriteString(join)
			f.Close()
			return nil
		},
	},

	//查Q
	{
		Command: []string{"查Q", "CQ"},
		Admin:   true,
		Handle: func(sender *Sender) interface{} {
			str := ""
			sender.Contents = sender.Contents[0:]
			sender.handleJdCookies(func(ck *JdCookie) {
				str = str + fmt.Sprintf("账号：%s (%s) QQ：%s \n", ck.Nickname, ck.PtPin, ck.UserId)
			})
			return str
		},
	},

	//备注
	{
		Command: []string{"备注", "bz"},
		Handle: func(sender *Sender) interface{} {
			if len(sender.Contents) > 1 {
				note := sender.Contents[0]
				sender.Contents = sender.Contents[1:]
				str := sender.Contents[0]
				number, err := strconv.Atoi(str)
				count := 0
				sender.handleJdCookies(func(ck *JdCookie) {
					count++
					if (err == nil && number == count) || ck.PtPin == str || sender.IsAdmin {
						ck.Update("Note", note)
						sender.Reply(fmt.Sprintf("已设置账号%s(%s)的备注为%s。", ck.PtPin, ck.Nickname, note))
					}
				})
			}
			return nil
		},
	},

	//设置掉线通知
	{
		Command: []string{"设置掉线通知", "掉线通知"},
		Handle: func(sender *Sender) interface{} {
			if len(sender.Contents) > 1 {
				PushPlus := sender.Contents[0]
				sender.Contents = sender.Contents[1:]
				str := sender.Contents[0]
				number, err := strconv.Atoi(str)
				count := 0
				sender.handleJdCookies(func(ck *JdCookie) {
					count++
					if (err == nil && number == count) || ck.PtPin == str || sender.IsAdmin {
						ck.Update("PushPlus", PushPlus)
						sender.Reply(fmt.Sprintf("已设置账号%s(%s)的通知token为%s。", ck.PtPin, ck.Nickname, PushPlus))
					}
				})
			}
			return nil
		},
	},

	//通知过期账号
	{
		Command: []string{"通知过期账号", "通知失效账号"},
		Admin:   true,
		Handle: func(sender *Sender) interface{} {
			sender.Reply("好的长官。")
			cks := GetJdCookies(func(sb *gorm.DB) *gorm.DB {
				return sb.Where(fmt.Sprintf("%s = ? ", Available), False)
			})
			xk := 0
			for _, ck := range cks {
				rt := fmt.Sprintf("你的账号【%s】已过期，请对机器人发（登录）重新上车", ck.Nickname)
				time.Sleep(time.Second * time.Duration(Config.Later))
				time.Sleep(time.Duration(rand.Intn(1000)+2000) * time.Millisecond)
				ck.Push(rt)
				time.Sleep(500)
				xk++
			}
			(&JdCookie{}).Push(fmt.Sprintf("发送完成，已通知过期账号%d个", xk))
			return nil
		},
	},

	//查询
	{
		Command: []string{"查询", "query"},
		Handle: func(sender *Sender) interface{} {
			sender.Reply("正在为您查询，请耐心等待")
			switch sender.Type {
			case "wx":
				sender.handleJdCookies(func(ck *JdCookie) {
					time.Sleep(time.Second * time.Duration(Config.Later))
					sender.Reply(ck.Query())
				})
			case "qq":
				value := GetEnv("qq")
				if value == "" || sender.IsAdmin {
					sender.handleJdCookies(func(ck *JdCookie) {
						time.Sleep(time.Second * time.Duration(Config.Later))
						sender.Reply(ck.Query())
					})
				} else {
					list := getUserNameList(sender.UserID)
					if list == nil {
						if Config.Query != "" {
							return Config.Query
						} else {
							return "你尚未绑定🐶东账号，请发送教程获取最新上车方法。"
						}
					} else {
						str := "在线账号:\n"
						for _, s := range list {
							str = str + fmt.Sprintf("账号：%s  \n", s)
						}
						sender.Reply(str)
						query := GetEnv("query")
						//https://qladmin.smxy.xyz/query#/?id=1095916117
						url := fmt.Sprintf("%squery#/?id=%s", query, sender.UserID)
						if query != "" {
							sender.Reply("请扫描二维码查看")
							var png []byte
							png, _ = qrcode.Encode(url, qrcode.Medium, 256)
							sender.SendImg(png)
						}
					}

				}
			case "qqg":
				value := GetEnv("qqg")
				if value == "" || sender.IsAdmin {
					sender.handleJdCookies(func(ck *JdCookie) {
						time.Sleep(time.Second * time.Duration(Config.Later))
						sender.Reply(ck.Query())
					})
				} else {
					list := getUserNameList(sender.UserID)
					if list == nil {
						if Config.Query != "" {
							return Config.Query
						} else {
							return "你尚未绑定🐶东账号，请发送教程获取最新上车方法。"
						}
					} else {
						str := "在线账号:\n"
						for _, s := range list {
							str = str + fmt.Sprintf("账号：%s  \n", s)
						}
						sender.Reply(str)
						//query := GetEnv("query")
						//url := fmt.Sprintf("%squery#/?id=%d", query, sender.UserID)
						//if query != "" {
						//	sender.Reply("请扫描二维码查看")
						//	var png []byte
						//	png, _ = qrcode.Encode(url, qrcode.Medium, 256)
						//	//logs.Info(Config.QQGroupID)
						//	//SendQQGroup(, sender.UserID, png)
						//}
					}
				}
			default:
				sender.handleJdCookies(func(ck *JdCookie) {
					time.Sleep(time.Second * time.Duration(Config.Later))
					sender.Reply(ck.Query())
				})
			}

			return nil
		},
	},

	//发送通知
	{
		Command: []string{"发送", "通知", "notify", "send"},
		Admin:   true,
		Handle: func(sender *Sender) interface{} {
			if len(sender.Contents) < 2 {
				sender.Reply("发送指令格式错误")
			} else {
				rt := strings.Join(sender.Contents[1:], " ")
				sender.Contents = sender.Contents[0:1]
				if sender.handleJdCookies(func(ck *JdCookie) {
					ck.Push(rt)
				}) == nil {
					return "操作成功"
				}
			}
			return nil
		},
	},

	//QQ转账
	{
		Command: []string{"QQ转账"},
		Admin:   true,
		Handle: func(sender *Sender) interface{} {
			qq := sender.Contents[0]
			logs.Info(qq)
			if len(sender.Contents) > 1 {
				//sender.Contents = sender.Contents[1:]
				logs.Info(sender.Contents[1:])
				AdddCoin(qq, Int(sender.Contents[1]))
				sender.Reply(fmt.Sprintf("%s已增加%d枚互助值。", qq, Int(sender.Contents[1])))
			}
			return nil
		},
	},

	//运行脚本
	{
		Command: []string{"run", "执行"},
		Admin:   true,
		Handle: func(sender *Sender) interface{} {
			name := sender.Contents[0]
			pins := ""
			if len(sender.Contents) > 1 {
				sender.Contents = sender.Contents[1:]
				err := sender.handleJdCookies(func(ck *JdCookie) {
					pins += "&" + ck.PtPin
				})
				if err != nil {
					return nil
				}
			}
			var envs []Env
			if pins != "" {
				envs = append(envs, Env{
					Name:  "pins",
					Value: pins,
				})
			}
			runTask(&Task{Path: name, Envs: envs}, sender)
			return nil
		},
	},

	//设置优先级
	{
		Command: []string{"优先级", "priority"},
		Admin:   true,
		Handle: func(sender *Sender) interface{} {
			priority := Int(sender.Contents[0])
			if len(sender.Contents) > 1 {
				sender.Contents = sender.Contents[1:]
				sender.handleJdCookies(func(ck *JdCookie) {
					ck.Update(Priority, priority)
					sender.Reply(fmt.Sprintf("已设置账号%s(%s)的优先级为%d。", ck.PtPin, ck.Nickname, priority))
				})
			}
			return nil
		},
	},

	//绑定QQ
	{
		Command: []string{"绑定"},
		Handle: func(sender *Sender) interface{} {
			qq := Int(sender.Contents[0])
			if len(sender.Contents) > 1 {
				sender.Contents = sender.Contents[1:]
				sender.handleJdCookies(func(ck *JdCookie) {
					ck.Update("UserId", qq)
					sender.Reply(fmt.Sprintf("已设置账号%s的QQ为%v。", ck.Nickname, ck.UserId))
				})
			}
			return nil
		},
	},

	//命令
	{
		Command: []string{"cmd", "command", "命令"},
		Admin:   true,
		Handle: func(sender *Sender) interface{} {
			ct := sender.JoinContens()
			if regexp.MustCompile(`rm\s+-rf`).FindString(ct) != "" {
				return "over"
			}
			cmd(ct, sender)
			return nil
		},
	},

	//管理后台
	{
		Command: []string{"管理后台"},
		Admin:   true,
		Handle: func(sender *Sender) interface{} {
			u := uuid.New()
			s := u.String()
			SaveCache("AdminToken", s)
			return "你的临时授权码为：" + s
		},
	},

	//获取环境变量
	{
		Command: []string{"环境变量", "environments", "envs"},
		Admin:   true,
		Handle: func(_ *Sender) interface{} {
			var rt []string
			envs := GetEnvs()
			if len(envs) == 0 {
				return "未设置任何环境变量"
			}
			for _, env := range envs {
				rt = append(rt, fmt.Sprintf(`%s="%s"`, env.Name, env.Value))
			}
			return strings.Join(rt, "\n")
		},
	},

	//获取环境变量
	{
		Command: []string{"get-env", "env", "e"},
		Handle: func(sender *Sender) interface{} {
			ct := sender.JoinContens()
			if ct == "" {
				return "未指定变量名"
			}
			value := GetEnv(ct)
			if value == "" {
				return "未设置环境变量"
			}
			return fmt.Sprintf("环境变量的值为：" + value)
		},
	},

	//设置环境变量
	{
		Command: []string{"set-env", "se", "export"},
		Admin:   true,
		Handle: func(sender *Sender) interface{} {
			env := &Env{}
			if len(sender.Contents) >= 2 {
				env.Name = sender.Contents[0]
				env.Value = strings.Join(sender.Contents[1:], " ")
			} else if len(sender.Contents) == 1 {
				ss := regexp.MustCompile(`^([^'"=]+)=['"]?([^=]+?)['"]?$`).FindStringSubmatch(sender.Contents[0])
				if len(ss) != 3 {
					return "无法解析"
				}
				env.Name = ss[1]
				env.Value = ss[2]
			} else {
				return "???"
			}
			ExportEnv(env)
			if strings.EqualFold(env.Name, "jd_zdjr_activityId") {
				runTask(&Task{Path: "jd_zdjr_activityId.js", Envs: []Env{
					{Name: "jd_zdjr_activityId", Value: env.Value},
				}}, sender)
			}
			return "操作成功"
		},
	},

	//删除环境变量
	{
		Command: []string{"unset-env", "ue", "unexport", "de"},
		Admin:   true,
		Handle: func(sender *Sender) interface{} {
			UnExportEnv(&Env{
				Name: sender.JoinContens(),
			})
			return "操作成功"
		},
	},

	//回复
	{
		Command: []string{"reply", "回复"},
		Admin:   true,
		Handle: func(sender *Sender) interface{} {
			if len(sender.Contents) >= 2 {
				replies[sender.Contents[0]] = strings.Join(sender.Contents[1:], " ")
			} else {
				return "操作失败"
			}
			return "操作成功"
		},
	},

	//更新指定R
	{
		Command: []string{"更新指定R"},
		Admin:   true,
		Handle: func(sender *Sender) interface{} {
			sender.handleJdCookies(func(ck *JdCookie) {
				if len(ck.WsKey) > 0 {
					var pinky = fmt.Sprintf("pin=%s;wskey=%s;", ck.PtPin, ck.RWskey)
					logs.Info(pinky)
					_, _, rsp := NolanGetCookie(pinky)
					_, _, rsp = BBKGetCookie(pinky)

					pin, _ := url.QueryUnescape(ck.PtPin)
					pinky = fmt.Sprintf("pin=%s;wskey=%s;", pin, ck.RWskey)
					_, _, rsp = RabbitGetCookie(pinky)

					if len(rsp) > 0 {
						if strings.Contains(rsp, "fake") {
							sender.Reply(fmt.Sprintf("Wskey失效，%s", ck.Nickname))
						}
						ptKey := FetchJdCookieValue("pt_key", rsp)
						ptPin := FetchJdCookieValue("pt_pin", rsp)
						ck := JdCookie{
							PtKey: ptKey,
							PtPin: ptPin,
						}
						if nck, err := GetJdCookie(ck.PtPin); err == nil {
							nck.Updates(JdCookie{PtKey: ptKey, Available: True})
							msg := fmt.Sprintf("更新账号，%s", ck.PtPin)
							sender.Reply(msg)
							logs.Info(msg)
						} else {
							sender.Reply("转换失败")
						}
					} else {
						sender.Reply("转换失败")
						//sender.Reply(fmt.Sprintf("Wskey失效，%s", ck.Nickname))
					}
				} else {
					sender.Reply(fmt.Sprintf("Wskey为空，%s", ck.Nickname))
				}

			})
			return nil
		},
	},

	//删除
	{
		Command: []string{"删除", "clean"},
		Admin:   true,
		Handle: func(sender *Sender) interface{} {
			sender.handleJdCookies(func(ck *JdCookie) {
				ck.Removes(ck)
				sender.Reply(fmt.Sprintf("已删除账号%s", ck.Nickname))
			})
			return nil
		},
	},

	//删除WCK
	{
		Command: []string{"删除WCK"},
		Admin:   true,
		Handle: func(sender *Sender) interface{} {
			sender.handleJdCookies(func(ck *JdCookie) {
				ck.Update(WsKey, "")
				sender.Reply(fmt.Sprintf("已删除WCK,%s", ck.Nickname))
			})
			return nil
		},
	},

	//清空WCK
	{
		Command: []string{"清空WCK"},
		Admin:   true,
		Handle: func(sender *Sender) interface{} {
			cleanWck()
			return nil
		},
	},

	//清理过期账号
	{
		Command: []string{"清理过期账号"},
		Admin:   true,
		Handle: func(sender *Sender) interface{} {
			sender.Reply(fmt.Sprintf("删除所有false账号，请慎用"))
			sender.handleJdCookies(func(ck *JdCookie) {
				cleanCookie()
			})
			return nil
		},
	},

	//转账
	{
		Command: []string{"转账"},
		Handle: func(sender *Sender) interface{} {
			cost := 1
			if sender.ReplySenderUserID == 0 {
				return "没有转账目标。"
			}
			amount := Int(sender.JoinContens())
			if !sender.IsAdmin {
				if amount <= 0 {
					return "转账金额必须大于等于1。"
				}
			}
			if Int(sender.UserID) == sender.ReplySenderUserID {
				db.Model(User{}).Where("number = ?", sender.UserID).Updates(map[string]interface{}{
					"coin": gorm.Expr(fmt.Sprintf("coin - %d", cost)),
				})
				return fmt.Sprintf("转账成功，扣除手续费%d枚互助值。", cost)
			}
			if amount > 10000 {
				return "单笔转账限额10000。"
			}
			tx := db.Begin()
			s := &User{}
			if err := db.Where("number = ?", sender.UserID).First(&s).Error; err != nil {
				tx.Rollback()
				return "你还没有开通钱包功能。"
			}
			if s.Coin < amount {
				tx.Rollback()
				return "余额不足。"
			}
			real := amount
			if !sender.IsAdmin {
				if amount <= cost {
					tx.Rollback()
					return fmt.Sprintf("转账失败，手续费需要%d个互助值。", cost)
				}
				real = amount - cost
			} else {
				cost = 0
			}
			r := &User{}
			if err := db.Where("number = ?", sender.ReplySenderUserID).First(&r).Error; err != nil {
				tx.Rollback()
				return "他还没有开通钱包功能"
			}
			if tx.Model(User{}).Where("number = ?", sender.UserID).Updates(map[string]interface{}{
				"coin": gorm.Expr(fmt.Sprintf("coin - %d", amount)),
			}).RowsAffected == 0 {
				tx.Rollback()
				return "转账失败"
			}
			if tx.Model(User{}).Where("number = ?", sender.ReplySenderUserID).Updates(map[string]interface{}{
				"coin": gorm.Expr(fmt.Sprintf("coin + %d", real)),
			}).RowsAffected == 0 {
				tx.Rollback()
				return "转账失败"
			}
			tx.Commit()
			return fmt.Sprintf("转账成功，你的余额%d，他的余额%d，手续费%d。", s.Coin-amount, r.Coin+real, cost)
		},
	},

	//导出
	{
		Command: []string{"献祭", "导出"},
		Admin:   true,
		Handle: func(sender *Sender) interface{} {
			sender.handleJdCookies(func(ck *JdCookie) {
				sender.Reply(fmt.Sprintf("pt_key=%s;pt_pin=%s;", ck.PtKey, ck.PtPin))
			})
			return nil
		},
	},

	//关闭私聊查询
	{
		Command: []string{"关闭私聊查询"},
		Admin:   true,
		Handle: func(sender *Sender) interface{} {
			env := &Env{
				Name:  "qq",
				Value: "1",
			}
			ExportEnv(env)
			sender.Reply("操作成功")
			return nil
		},
	},

	//开启私聊查询
	{
		Command: []string{"开启私聊查询"},
		Admin:   true,
		Handle: func(sender *Sender) interface{} {
			UnExportEnv(&Env{
				Name: "qq",
			})
			sender.Reply("操作成功")
			return nil
		},
	},

	//关闭群聊查询
	{
		Command: []string{"关闭群聊查询"},
		Admin:   true,
		Handle: func(sender *Sender) interface{} {
			env := &Env{
				Name:  "qqg",
				Value: "1",
			}
			ExportEnv(env)
			sender.Reply("操作成功")
			return nil
		},
	},

	//开启私聊查询
	{
		Command: []string{"开启私聊查询"},
		Admin:   true,
		Handle: func(sender *Sender) interface{} {
			UnExportEnv(&Env{
				Name: "qqg",
			})
			sender.Reply("操作成功")
			return nil
		},
	},

	//开启微信自动好友
	{
		Command: []string{"开启微信自动好友"},
		Admin:   true,
		Handle: func(sender *Sender) interface{} {
			env := &Env{
				Name:  "AutoAgree",
				Value: "1",
			}
			ExportEnv(env)
			sender.Reply("操作成功")
			return nil
		},
	},

	//关闭微信自动好友
	{
		Command: []string{"关闭微信自动好友"},
		Admin:   true,
		Handle: func(sender *Sender) interface{} {
			UnExportEnv(&Env{
				Name: "AutoAgree",
			})
			sender.Reply("操作成功")
			return nil
		},
	},

	//开启微信自动收款
	{
		Command: []string{"开启微信自动收款"},
		Admin:   true,
		Handle: func(sender *Sender) interface{} {
			env := &Env{
				Name:  "f",
				Value: "1",
			}
			ExportEnv(env)
			sender.Reply("操作成功")
			return nil
		},
	},

	//关闭微信自动收款
	{
		Command: []string{"关闭微信自动收款"},
		Admin:   true,
		Handle: func(sender *Sender) interface{} {
			UnExportEnv(&Env{
				Name: "Autocollection",
			})
			sender.Reply("操作成功")
			return nil
		},
	},

	//设置微信口令
	{
		Command: []string{"设置微信口令"},
		Admin:   true,
		Handle: func(sender *Sender) interface{} {
			ctt := sender.JoinContens()
			env := &Env{
				Name:  "AgreeMsg",
				Value: ctt,
			}
			ExportEnv(env)
			sender.Reply("操作成功")
			return nil
		},
	},

	//关闭微信口令
	{
		Command: []string{"关闭微信口令"},
		Admin:   true,
		Handle: func(sender *Sender) interface{} {
			UnExportEnv(&Env{
				Name: "AgreeMsg",
			})
			sender.Reply("操作成功")
			return nil
		},
	},

	//用户信息
	{
		Command: []string{"用户信息"},
		Handle: func(sender *Sender) interface{} {
			return fmt.Sprintf("用户ID：%s", sender.UserID)
		},
	},

	//设置欢迎语
	{
		Command: []string{"设置欢迎语"},
		Admin:   true,
		Handle: func(sender *Sender) interface{} {
			ctt := sender.JoinContens()
			env := &Env{
				Name:  "Welcome",
				Value: ctt,
			}
			ExportEnv(env)
			sender.Reply("操作成功")
			return nil
		},
	},

	//取消欢迎语
	{
		Command: []string{"取消欢迎语"},
		Admin:   true,
		Handle: func(sender *Sender) interface{} {
			UnExportEnv(&Env{
				Name: "Welcome",
			})
			sender.Reply("操作成功")
			return nil
		},
	},

	//设置signUrl
	{
		Command: []string{"设置signUrl"},
		Admin:   true,
		Handle: func(sender *Sender) interface{} {
			ctt := sender.JoinContens()
			env := &Env{
				Name:  "sign",
				Value: ctt,
			}
			ExportEnv(env)
			sender.Reply("操作成功")
			return nil
		},
	},

	//取消signUrl
	{
		Command: []string{"取消signUrl"},
		Admin:   true,
		Handle: func(sender *Sender) interface{} {
			UnExportEnv(&Env{
				Name: "sign",
			})
			sender.Reply("操作成功")
			return nil
		},
	},

	//设置代理
	{
		Command: []string{"设置代理"},
		Admin:   true,
		Handle: func(sender *Sender) interface{} {
			ctt := sender.JoinContens()
			env := &Env{
				Name:  "proxy",
				Value: ctt,
			}
			ExportEnv(env)
			sender.Reply("代理设置成功")
			return nil
		},
	},

	//导出wskey
	{
		Command: []string{"导出wskey"},
		Admin:   true,
		Handle: func(sender *Sender) interface{} {
			sender.handleJdCookies(func(ck *JdCookie) {
				sender.Reply(fmt.Sprintf("pin=%s;wskey=%s;", ck.PtPin, ck.WsKey))
			})
			return nil
		},
	},

	//美团登录
	{
		Command: []string{"美团登录", "登录美团", "美团登陆"},
		Handle: func(sender *Sender) interface{} {
			Meituan_getck(sender)
			return nil
		},
	},
	{
		Command: []string{"美团登录", "登录美团"},
		Handle: func(sender *Sender) interface{} {
			Meituan_getck(sender)
			sender.Reply("点击链接：\n https://passport.meituan.com/useraccount/ilogin")
			return nil
		},
	},

	//{
	//	Command: []string{"农场浇水", "浇水农场"},
	//	Handle: func(sender *Sender) interface{} {
	//		id := sender.UserID
	//		var idType string
	//		if sender.Type == "tg" {
	//			idType = Telegram
	//		} else {
	//			idType = QQ
	//		}
	//		cks := GetJdCookies(func(sb *gorm.DB) *gorm.DB {
	//			return sb.Where(fmt.Sprintf("%s = ? and %s = ?", idType, Available), id, True)
	//		})
	//
	//		if len(cks) > 0 {
	//			//进入队列
	//			msg := make(chan string)
	//			ckList[sender.UserID] = msg
	//			go Jd_fruit_watering(sender, msg, cks)
	//			msgs := []string{
	//				"请回复以下序列号指定账号运行任务，如需退出请回复'q'退出运行流程：",
	//			}
	//			for i, ck := range cks {
	//				msgs = append(msgs, fmt.Sprintf("%d、%s", i, ck.Nickname))
	//			}
	//			sender.Reply(strings.Join(msgs, "\n"))
	//		} else {
	//			sender.Reply("在线账号已全部失效，请对机器人发送“登录”")
	//			return nil
	//		}
	//		return nil
	//	},
	//},
	//{
	//	Command: []string{"一键保价", "一键价保"},
	//	Handle: func(sender *Sender) interface{} {
	//		id := sender.UserID
	//		var idType string
	//		if sender.Type == "tg" {
	//			idType = Telegram
	//		} else {
	//			idType = QQ
	//		}
	//		cks := GetJdCookies(func(sb *gorm.DB) *gorm.DB {
	//			return sb.Where(fmt.Sprintf("%s = ? and %s = ?", idType, Available), id, True)
	//		})
	//
	//		if len(cks) > 0 {
	//
	//			//进入队列
	//			msg := make(chan string)
	//			ckList[sender.UserID] = msg
	//			go Jd_price(sender, msg, cks)
	//
	//			msgs := []string{
	//				"请回复以下序列号指定账号运行任务，如需退出请回复'q'退出登录流程：",
	//			}
	//			for i, ck := range cks {
	//				msgs = append(msgs, fmt.Sprintf("%d、%s", i, ck.Nickname))
	//			}
	//			sender.Reply(strings.Join(msgs, "\n"))
	//		} else {
	//			sender.Reply("在线账号已全部失效，请对机器人发送“登录”")
	//			return nil
	//		}
	//		return nil
	//	},
	//},
	//{
	//	Command: []string{"一键评价", "自动评价"},
	//	Handle: func(sender *Sender) interface{} {
	//		id := sender.UserID
	//		var idType string
	//		if sender.Type == "tg" {
	//			idType = Telegram
	//		} else {
	//			idType = QQ
	//		}
	//		cks := GetJdCookies(func(sb *gorm.DB) *gorm.DB {
	//			return sb.Where(fmt.Sprintf("%s = ? and %s = ?", idType, Available), id, True)
	//		})
	//
	//		if len(cks) > 0 {
	//
	//			//进入队列
	//			msg := make(chan string)
	//			ckList[sender.UserID] = msg
	//			go Jd_AutoEval(sender, msg, cks)
	//
	//			msgs := []string{
	//				"请回复以下序列号指定账号运行任务，如需退出请回复'q'退出登录流程：",
	//			}
	//			for i, ck := range cks {
	//				msgs = append(msgs, fmt.Sprintf("%d、%s", i, ck.Nickname))
	//			}
	//			sender.Reply(strings.Join(msgs, "\n"))
	//		} else {
	//			sender.Reply("在线账号已全部失效，请对机器人发送“登录”")
	//			return nil
	//		}
	//		return nil
	//	},
	//},

	//{
	//	Command: []string{"失效账号", "账号删除"},
	//	Handle: func(sender *Sender) interface{} {
	//		id := sender.UserID
	//		var idType string
	//		if sender.Type == "tg" {
	//			idType = Telegram
	//		} else {
	//			idType = QQ
	//		}
	//		cks := GetJdCookies(func(sb *gorm.DB) *gorm.DB {
	//			//return sb.Where(fmt.Sprintf("%s = ?", idType), id)
	//			return sb.Where(fmt.Sprintf("%s = ? and %s = ?", idType, Available), id, False)
	//		})
	//
	//		if len(cks) > 0 {
	//			msg := make(chan string)
	//			ckList[sender.UserID] = msg
	//			go Delete_jdck(sender, msg, cks)
	//			msgs := []string{
	//				"请回复以下序列号删除指定失效账号，如需退出请回复'q'退出登录流程：",
	//			}
	//			for i, ck := range cks {
	//				msgs = append(msgs, fmt.Sprintf("%d、%s", i, ck.Nickname))
	//			}
	//			sender.Reply(strings.Join(msgs, "\n"))
	//		} else {
	//			sender.Reply("无失效账号，新增账号请对机器人发送“登录”")
	//			return nil
	//		}
	//		return nil
	//	},
	//},
	//{
	//	Command: []string{"删除美团", "美团删除"},
	//	Handle: func(sender *Sender) interface{} {
	//		meiTuans := GetMeiTuan(sender)
	//		if len(meiTuans) > 0 {
	//			// 进入队列
	//			msg := make(chan string)
	//			meituanList[sender.UserID] = msg
	//			go Delete_meituan(sender, msg, meiTuans)
	//			msgs := []string{
	//				"请回复以下序列号删除指定账号，如需退出请回复'q'退出登录流程：",
	//			}
	//			for i, tuan := range meiTuans {
	//				msgs = append(msgs, fmt.Sprintf("%d、%s", i, tuan.Nickname))
	//			}
	//				sender.Reply(strings.Join(msgs, "\n"))
	//		} else {
	//			sender.Reply("查无美团账号，请使用 '美团登录'口令，按要求提交ck")
	//			return nil
	//		}
	//		return nil
	//	},
	//},

}

func InviteGroup(uid string, gid string) {
	type AutoGenerated1 struct {
		Token     string `json:"token"`
		API       string `json:"api"`
		RobotWxid string `json:"robot_wxid"`
		ToWxid    string `json:"friend_wxid"`
		Msg       string `json:"group_wxid"`
	}

	req := httplib.Post(Config.Wx.Url)
	reply := &AutoGenerated1{
		Token:     Config.Wx.Token,
		API:       "InviteInGroupByLink",
		RobotWxid: Config.Wx.Robotid,
		ToWxid:    uid,
		Msg:       gid,
	}
	random := browser.Random()
	req.Header("User-Agent", random)
	marshal, _ := json.Marshal(reply)
	logs.Info(string(marshal))
	req.Body(string(marshal))
	s, _ := req.String()
	logs.Info(s)
}

func GetPinList(qq string) []string {
	var cks []JdCookie
	var pins []string
	db.Where(fmt.Sprintf("QQ = %s", qq)).Find(&cks)
	if len(cks) > 0 {
		for _, ck := range cks {
			pins = append(pins, ck.PtPin)
		}
	} else {
		return nil
	}
	return pins
}

func getUserNameList(qq string) []string {
	var cks []JdCookie
	var names []string
	db.Where(fmt.Sprintf("QQ = %s", qq)).Find(&cks)
	t, _ := time.Parse("2006-01-02", time.Now().Format("2006-01-02"))
	if len(cks) > 0 {
		for _, ck := range cks {
			if CookieOK(&ck) {
				if ck.UpdateAt != "" {
					parse1, _ := time.Parse("2006-01-02", ck.UpdateAt)
					f := t.Sub(parse1).Hours() / 24
					i, _ := strconv.Atoi(fmt.Sprintf("%1.0f", f))
					if !strings.Contains(ck.PtKey, "app_open") {
						names = append(names, fmt.Sprintf("%s\n距离失效还有：%d天\n", ck.Nickname, 28-i))
					} else {
						names = append(names, fmt.Sprintf("%s\n尊贵的年费用户，您距离失效还有：%d天 \n", ck.Nickname, 90-i))
					}
				}
			} else {
				names = append(names, fmt.Sprintf("%s\n账号已过期\n", ck.Nickname))
			}

		}
	} else {
		return nil
	}
	return names
}

func LimitJdCookie(cks []JdCookie, a string) []JdCookie {
	var ncks []JdCookie
	if s := strings.Split(a, "-"); len(s) == 2 {
		for i := range cks {
			if i+1 >= Int(s[0]) && i+1 <= Int(s[1]) {
				ncks = append(ncks, cks[i])
			}
		}
	} else if x := regexp.MustCompile(`^[\s\d,]+$`).FindString(a); x != "" {
		xx := regexp.MustCompile(`(\d+)`).FindAllStringSubmatch(a, -1)
		for i := range cks {
			for _, x := range xx {
				if fmt.Sprint(i+1) == x[1] {
					ncks = append(ncks, cks[i])
				} else if cks[i].UserId == x[1] {
					ncks = append(ncks, cks[i])
				}
			}

		}
	} else if a != "" {
		a = strings.Replace(a, " ", "", -1)
		for i := range cks {
			if strings.Contains(cks[i].Note, a) || strings.Contains(cks[i].Nickname, a) || strings.Contains(cks[i].PtPin, a) {
				ncks = append(ncks, cks[i])
			}
		}
	}
	return ncks
}

func ReturnCoin(sender *Sender) {
	tx := db.Begin()
	var ws []Wish
	if err := tx.Where("status = 0 and user_number = ?", sender.UserID).Find(&ws).Error; err != nil {
		tx.Rollback()
		sender.Reply(err.Error())
	}
	for _, w := range ws {
		if tx.Model(User{}).Where("number = ? ", sender.UserID).Update(
			"coin", gorm.Expr(fmt.Sprintf("coin + %d", w.Coin)),
		).RowsAffected == 0 {
			tx.Rollback()
			sender.Reply("愿望未达成退还互助值失败。")
			return
		}
		sender.Reply(fmt.Sprintf("愿望未达成退还%d枚互助值。", w.Coin))
		if tx.Model(&w).Update(
			"status", 1,
		).RowsAffected == 0 {
			tx.Rollback()
			sender.Reply("愿望未达成退还互助值失败。")
			return
		}
	}
	tx.Commit()
}

func Count() string {
	zs := 0
	yx := 0
	wx := 0
	tl := 0
	ts := 0
	tc := 0
	dt := Date()
	cks := GetJdCookies()
	for _, ck := range cks {
		zs++
		if ck.Available == True {
			yx++
		} else {
			wx++
		}
		if ck.CreateAt == dt {
			tc++
		}
		if ck.LoseAt == dt {
			tl++
		}
		if ck.UpdateAt == dt {
			ts++
		}
	}

	return fmt.Sprintf("总数%d,有效%d,无效%d,今日失效%d,今日更新%d,今日新增%d", zs, yx, wx, tl, ts, tc)
}
