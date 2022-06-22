package models

import (
	"errors"
	"fmt"
	"github.com/beego/beego/v2/client/httplib"
	"github.com/beego/beego/v2/core/logs"
	"gorm.io/gorm"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type CodeSignal struct {
	Command []string
	Admin   bool
	Handle  func(sender *Sender) interface{}
}

type Sender struct {
	UserID            int
	ChatID            int
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
		SendQQ(int64(sender.UserID), msg)
		//if strings.Contains(msg, "账号昵称：") && Config.VIP {
		//	SendQQ(int64(sender.UserID), strtoimg(msg))
		//} else {
		//	SendQQ(int64(sender.UserID), msg)
		//}
	case "qqg":
		SendQQGroup(int64(sender.ChatID), int64(sender.UserID), msg)
	}
}

func (sender *Sender) JoinContens() string {
	return strings.Join(sender.Contents, " ")
}

func (sender *Sender) IsQQ() bool {
	return strings.Contains(sender.Type, "qq")
}

func (sender *Sender) IsTG() bool {
	return strings.Contains(sender.Type, "tg")
}

var wb = false
var qj = false

func (sender *Sender) handleJdCookies(handle func(ck *JdCookie)) error {
	cks := GetJdCookies()
	a := sender.JoinContens()
	ok := false
	if !sender.IsAdmin && a == "" {
		for i := range cks {
			if strings.Contains(sender.Type, "qq") {
				if cks[i].QQ == sender.UserID {
					if !ok {
						ok = true
					}
					handle(&cks[i])
				}
			} else if strings.Contains(sender.Type, "tg") {
				if cks[i].Telegram == sender.UserID {
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
				sender.Reply("你尚未绑定🐶东账号，请发送教程获取最新上车方法。")
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

	{
		Command: []string{"获取临时CK"},
		Handle: func(sender *Sender) interface{} {
			if Config.VIP == true {
				sender.handleJdCookies(func(ck *JdCookie) {
					time.Sleep(time.Second * time.Duration(Config.Later))
					cookie := fmt.Sprintf("pt_key=%s;pt_pin=%s;", ck.PtKey, ck.PtPin)
					if strings.Contains(cookie, "app_open") {
						sender.Reply(cookie)
					}
				})
			}
			return nil
		},
	},

	{
		Command: []string{"status", "状态"},
		Admin:   true,
		Handle: func(sender *Sender) interface{} {
			return Count()
		},
	},

	{
		Command: []string{"清空WCK"},
		Admin:   true,
		Handle: func(sender *Sender) interface{} {
			cleanWck()
			return nil
		},
	},

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

	//{
	//	Command: []string{"sign", "打卡", "签到"},
	//	Handle: func(sender *Sender) interface{} {
	//		//if sender.Type == "tgg" {
	//		//	sender.Type = "tg"
	//		//}
	//		//if sender.Type == "qqg" {
	//		//	sender.Type = "qq"
	//		//}
	//		zero, _ := time.ParseInLocation("2006-01-02", time.Now().Local().Format("2006-01-02"), time.Local)
	//		var u User
	//		var ntime = time.Now()
	//		var first = false
	//		total := []int{}
	//		err := db.Where("number = ?", sender.UserID).First(&u).Error
	//		if err != nil {
	//			first = true
	//			u = User{
	//				Class:    sender.Type,
	//				Number:   sender.UserID,
	//				Coin:     1,
	//				ActiveAt: ntime,
	//				Womail:   "",
	//			}
	//			if err := db.Create(&u).Error; err != nil {
	//				return err.Error()
	//			}
	//		} else {
	//			if zero.Unix() > u.ActiveAt.Unix() {
	//				first = true
	//			} else {
	//				return fmt.Sprintf("你打过卡了，积分余额%d。", u.Coin)
	//			}
	//		}
	//		if first {
	//			db.Model(User{}).Select("count(id) as total").Where("active_at > ?", zero).Pluck("total", &total)
	//			coin := 1
	//			if total[0]%3 == 0 {
	//				coin = 2
	//			}
	//			if total[0]%13 == 0 {
	//				coin = 8
	//			}
	//			db.Model(&u).Updates(map[string]interface{}{
	//				"active_at": ntime,
	//				"coin":      gorm.Expr(fmt.Sprintf("coin+%d", coin)),
	//			})
	//			u.Coin += coin
	//			if u.Womail != "" {
	//				rsp := cmd(fmt.Sprintf(`python3 womail.py "%s"`, u.Womail), &Sender{})
	//				sender.Reply(fmt.Sprintf("%s", rsp))
	//			}
	//			sender.Reply(fmt.Sprintf("你是打卡第%d人，奖励%d个积分，积分余额%d。", total[0]+1, coin, u.Coin))
	//			ReturnCoin(sender)
	//			return ""
	//		}
	//		return nil
	//	},
	//},

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

	{
		Command: []string{"coin", "积分"},
		Handle: func(sender *Sender) interface{} {
			return fmt.Sprintf("积分:%d", GetCoin(sender.UserID))
		},
	},

	{
		Command: []string{"推一推状态"},
		Admin:   true,
		Handle: func(sender *Sender) interface{} {
			cks := []JdCookie{}
			db.Where(fmt.Sprintf("%s = 'true' and %s = 'true'", Tyt, Available)).Order("RAND()").Find(&cks)
			return fmt.Sprintf("推一推正在运行线程:%d,闲置线程:%d,剩余推一推个数:%d", tytnum, 3-tytnum, len(cks))
		},
	},

	{
		Command: []string{"大赢家状态"},
		Admin:   true,
		Handle: func(sender *Sender) interface{} {
			cks := []JdCookie{}
			db.Where(fmt.Sprintf("%s = 'true' and %s = 'true'", Dig, Available)).Order("RAND()").Find(&cks)
			return fmt.Sprintf("推一推正在运行线程:%d,闲置线程:%d,剩余大赢家个数:%d", tytnum, 3-tytnum, len(cks))
		},
	},

	{
		Command: []string{"开始检测"},
		Admin:   true,
		Handle: func(sender *Sender) interface{} {
			initCookie()
			return "检测完成"
		},
	},

	{
		Command: []string{"升级", "更新", "update", "upgrade"},
		Admin:   true,
		Handle: func(sender *Sender) interface{} {
			if err := Update(sender); err != nil {
				return err.Error()
			}
			sender.Reply("重启程序")
			Daemon()
			return nil
		},
	},

	{
		Command: []string{"重启", "reload", "restart", "reboot"},
		Admin:   true,
		Handle: func(sender *Sender) interface{} {
			sender.Reply("重启程序")
			Daemon()
			return nil
		},
	},

	{
		Command: []string{"更新账号", "Whiskey更新", "给老子更新"},
		Admin:   true,
		Handle: func(sender *Sender) interface{} {
			sender.Reply("更新所有账号")
			logs.Info("更新所有账号")
			updateCookie()
			return nil
		},
	},

	{
		Command: []string{"导出所有账号"},
		Admin:   true,
		Handle: func(sender *Sender) interface{} {
			var msgs []string
			cks := GetJdCookies(func(sb *gorm.DB) *gorm.DB {
				return sb.Where(fmt.Sprintf("%s >= ? and %s != ? and %s = ?", Priority, Hack, Available), 0, True, True)
			})
			for _, ck := range cks {
				msgs = append(msgs, fmt.Sprintf("pt_key=%s;pt_pin=%s;", ck.PtKey, ck.PtPin))
			}
			sender.Reply("导出所有账号")
			logs.Info("导出所有账号")
			f, err := os.OpenFile(ExecPath+"/jdCookie.txt", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0777)
			if err != nil {
				logs.Warn("创建jdCookie.txt失败，", err)
			}
			join := strings.Join(msgs, "\n")
			f.WriteString(join)
			f.Close()
			return nil
		},
	},

	{
		Command: []string{"导出所有W账号"},
		Admin:   true,
		Handle: func(sender *Sender) interface{} {
			var msgs []string
			cks := GetJdCookies(func(sb *gorm.DB) *gorm.DB {
				return sb.Where(fmt.Sprintf("%s >= ? and %s != ? and %s = ? and WsKey != '' ", Priority, Hack, Available), 0, True, True)
			})
			for _, ck := range cks {
				msgs = append(msgs, fmt.Sprintf("pt_key=%s;pt_pin=%s;", ck.PtKey, ck.PtPin))
			}
			sender.Reply("导出所有账号")
			logs.Info("导出所有账号")
			f, err := os.OpenFile(ExecPath+"/jdCookie.txt", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0777)
			if err != nil {
				logs.Warn("创建jdCookie.txt失败，", err)
			}
			join := strings.Join(msgs, "\n")
			f.WriteString(join)
			f.Close()
			return nil
		},
	},

	{
		Command: []string{"开启wb"},
		Admin:   true,
		Handle: func(sender *Sender) interface{} {
			wb = true
			sender.Reply("开启wb")
			return nil
		},
	},
	{
		Command: []string{"关闭wb"},
		Admin:   true,
		Handle: func(sender *Sender) interface{} {
			wb = false
			sender.Reply("关闭wb")
			return nil
		},
	},
	{
		Command: []string{"开启qj"},
		Admin:   true,
		Handle: func(sender *Sender) interface{} {
			qj = true
			sender.Reply("开启qj")
			return nil
		},
	},
	{
		Command: []string{"关闭qj"},
		Admin:   true,
		Handle: func(sender *Sender) interface{} {
			qj = false
			sender.Reply("关闭qj")
			return nil
		},
	},
	{
		Command: []string{"微博", "wb"},
		Handle: func(sender *Sender) interface{} {
			if wb != true {
				sender.Reply("项目未开启，如有需求请联系群主。")
				return nil
			}
			f, err := os.OpenFile(ExecPath+"/wb.txt", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0777)
			if err != nil {
				logs.Warn("wb.txt失败，", err)
			}
			sender.handleJdCookies(func(ck *JdCookie) {
				if GetCoin(sender.UserID) > 34 {
					if ck.WsKey != "" {
						f.WriteString(fmt.Sprintf("ptkey=%s;pin=%s;\n", ck.PtKey, ck.PtPin))
						RemCoin(sender.UserID, 35)
						sender.Reply(fmt.Sprintf("已提交订单：账号：%s，扣除积分35，剩余积分：%d", ck.PtPin, GetCoin(sender.UserID)))

					} else {
						f.WriteString(fmt.Sprintf("pt_key=%s;pt_pin=%s;\n", ck.PtKey, ck.PtPin))
						RemCoin(sender.UserID, 35)
						sender.Reply(fmt.Sprintf("已提交订单：账号：%s，扣除积分35，剩余积分：%d", ck.PtPin, GetCoin(sender.UserID)))

					}
				} else {
					sender.Reply("积分不足")
				}
			})
			f.Close()
			return nil
		},
	},
	{
		Command: []string{"qj"},
		Handle: func(sender *Sender) interface{} {
			if qj != true {
				sender.Reply("项目未开启，如有需求请联系群主。")
				return nil
			}
			f, err := os.OpenFile(ExecPath+"/qj.txt", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0777)
			if err != nil {
				logs.Warn("qj.txt失败，", err)
			}
			sender.handleJdCookies(func(ck *JdCookie) {
				if GetCoin(sender.UserID) > 1 {
					f.WriteString(fmt.Sprintf("pt_key=%s;pt_pin=%s;\n", ck.PtKey, ck.PtPin))
					//RemCoin(sender.UserID, 5)
					sender.Reply(fmt.Sprintf("已提交订单：账号：%s，扣除积分5，剩余积分：%d", ck.PtPin, GetCoin(sender.UserID)))
				} else {
					sender.Reply("积分不足")
				}
			})
			f.Close()
			return nil
		},
	},
	{
		Command: []string{"重置推一推"},
		Admin:   true,
		Handle: func(sender *Sender) interface{} {
			cks := GetJdCookies()
			for _, ck := range cks {
				ck.Update(Tyt, True)
			}
			sender.Reply("重置推一推")
			return nil
		},
	},

	{
		Command: []string{"推一推"},
		Admin:   true,
		Handle: func(sender *Sender) interface{} {
			if len(sender.Contents) > 0 {
				no := tytno
				tytno += 1
				code := sender.JoinContens()
				tytlist[code] = no
				go runtyt(sender, code)
				logs.Info(code)
				sender.Reply(fmt.Sprintf("开始Code：%s", code))
			}
			return nil
		},
	},
	//{
	//	Command: []string{"挖宝1"},
	//	Handle: func(sender *Sender) interface{} {
	//		f, err := os.OpenFile(ExecPath+"/wb1.txt", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0777)
	//		if err != nil {
	//			logs.Warn("wb.txt失败，", err)
	//		}
	//		sender.handleJdCookies(func(ck *JdCookie) {
	//			sender.Reply(fmt.Sprintf("已记录账号：账号：%s", ck.PtPin))
	//		})
	//		f.Close()
	//		return nil
	//	},
	//},
	{
		Command: []string{"查询", "query"},
		Handle: func(sender *Sender) interface{} {
			sender.Reply("请扫码完成网页查询")
			if sender.IsAdmin {
				sender.handleJdCookies(func(ck *JdCookie) {
					time.Sleep(time.Second * time.Duration(Config.Later))
					sender.Reply(ck.Query())
				})
			} else {
				list := getUserNameList(strconv.Itoa(sender.UserID))
				str := "在线账号:\n"
				for _, s := range list {
					str = str + fmt.Sprintf("账号：%s  \n", s)
				}
				sender.Reply(str)
				url := "http://h5img.smxy.xyz/qrcode.png"
				rsp, err := httplib.Get(url).Response()
				if err != nil {
					return nil
				}
				return rsp
			}
			//else {
			//	if getLimit(sender.UserID, 1) {
			//		sender.handleJdCookies(func(ck *JdCookie) {
			//			time.Sleep(timew.Second * time.Duration(Config.Later))
			//			sender.Reply(ck.Query())
			//		})
			//	} else {
			//		sender.Reply(fmt.Sprintf("鉴于东哥对接口限流，为了不影响大家的任务正常运行，即日起每日限流%d次，已超过今日限制", Config.Lim))
			//	}
			//}
			//sender.Reply("今日查询接口维护，请明日再来")

			return nil
		},
	},
	{
		Command: []string{"查Q", "CQ"},
		Admin:   true,
		Handle: func(sender *Sender) interface{} {
			str := ""
			sender.Contents = sender.Contents[0:]
			sender.handleJdCookies(func(ck *JdCookie) {
				str = str + fmt.Sprintf("账号：%s (%s) QQ：%d \n", ck.Nickname, ck.PtPin, ck.QQ)
			})
			return str
		},
	},
	{
		Command: []string{"详细查询", "query"},
		Handle: func(sender *Sender) interface{} {
			if sender.IsAdmin {
				sender.handleJdCookies(func(ck *JdCookie) {
					time.Sleep(time.Second * time.Duration(Config.Later))
					sender.Reply(ck.Query1())
				})
			} else {
				if getLimit(sender.UserID, 1) {
					time.Sleep(time.Second * time.Duration(Config.Later))
					sender.handleJdCookies(func(ck *JdCookie) {
						sender.Reply(ck.Query1())
					})
				} else {
					sender.Reply(fmt.Sprintf("鉴于东哥对接口限流，为了不影响大家的任务正常运行，即日起每日限流%d次，已超过今日限制", Config.Lim))
				}
			}

			return nil
		},
	},

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

	{
		Command: []string{"设置管理员"},
		Admin:   true,
		Handle: func(sender *Sender) interface{} {
			ctt := sender.JoinContens()
			db.Create(&UserAdmin{Content: ctt})
			return "已设置管理员"
		},
	},

	{
		Command: []string{"取消管理员"},
		Admin:   true,
		Handle: func(sender *Sender) interface{} {
			ctt := sender.JoinContens()
			RemoveUserAdmin(ctt)
			return "已取消管理员"
		},
	},

	{
		Command: []string{"run", "执行", "运行"},
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
			envs := []Env{}
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

	{
		Command: []string{"绑定"},
		Handle: func(sender *Sender) interface{} {
			qq := Int(sender.Contents[0])
			if len(sender.Contents) > 1 {
				sender.Contents = sender.Contents[1:]
				sender.handleJdCookies(func(ck *JdCookie) {
					ck.Update(QQ, qq)
					sender.Reply(fmt.Sprintf("已设置账号%s的QQ为%v。", ck.Nickname, ck.QQ))
				})
			}
			return nil
		},
	},

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

	{
		Command: []string{"屏蔽", "hack"},
		Admin:   true,
		Handle: func(sender *Sender) interface{} {
			sender.handleJdCookies(func(ck *JdCookie) {
				ck.Update(Priority, -1)
				sender.Reply(fmt.Sprintf("已屏蔽账号%s", ck.Nickname))
			})
			return nil
		},
	},

	{
		Command: []string{"更新指定"},
		Admin:   true,
		Handle: func(sender *Sender) interface{} {
			sender.handleJdCookies(func(ck *JdCookie) {
				if len(ck.WsKey) > 0 {
					var pinky = fmt.Sprintf("pin=%s;wskey=%s;", ck.PtPin, ck.WsKey)
					rsp, err := getKey(pinky)
					if err != nil {
						logs.Error(err)
					}
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
							nck.InPool(ck.PtKey)
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

	{
		Command: []string{"删除", "clean"},
		Admin:   true,
		Handle: func(sender *Sender) interface{} {
			sender.handleJdCookies(func(ck *JdCookie) {
				ck.Removes(ck)
				ck.OutPool()
				sender.Reply(fmt.Sprintf("已删除账号%s", ck.Nickname))
			})
			return nil
		},
	},

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
}

var mx = map[int]bool{}

func GetPinList(qq string) []string {
	cks := []JdCookie{}
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
	cks := []JdCookie{}
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
						names = append(names, fmt.Sprintf("%s\n尊贵的年费用户，您距离失效还有：%d天 \n", ck.Nickname, 365-i))
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
	ncks := []JdCookie{}
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
