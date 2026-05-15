package models

import (
	"encoding/base64"
	"errors"
	"fmt"
	"math/rand"
	"net/url"
//	"path/filepath" 
	"os"
	//	"math"
	//	"bufio"
	"sort"
 	"sync"
	"bytes"
	"encoding/json"
	browser "github.com/EDDYCJY/fake-useragent"
	"github.com/beego/beego/v2/client/httplib"
	"github.com/beego/beego/v2/core/logs"
	//	"sync"
	"compress/gzip"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"io"
	"net/http"
	
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
//	"log"
	
	
)
var (
    duelMutex   sync.Mutex       // 决斗房间互斥锁
    duels       map[string]*Duel // 存储所有决斗房间，key为决斗ID
    duelsByTime []*Duel          // 按时间排序的决斗列表
    
)



// 游戏结构体定义
type Game struct {
    ID       string            // 游戏ID
    Creator  int               // 创建人ID
    Platform string            // 平台类型（"wxg"或"qqg"）
    Players  map[int]int       // 玩家ID到数字的映射
    Status   string            // 游戏状态（waiting/playing/finished）
    Scores   map[int]int       // 玩家ID到奖励积分的映射
    CreateTime time.Time         // 游戏创建时间
}

type CodeSignal struct {
	Command []string
	Admin   bool
	Handle  func(sender *Sender) interface{}
}

type Sender struct {
	WxId              string
	UserID            int
	ChatID            int
	GroupId           int
	WxGroupId         string
	Type              string
	Contents          []string
     RawMessage string  // ← 新增字段
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
		SendWxMsg(sender.WxId, msg)
	case "wxg":
		SendWxGroupMsg(sender.WxId, sender.WxGroupId, msg)
	}
}

func (sender *Sender) SendImg(msg []byte) {
	switch sender.Type {
	case "qq":
		SendQQMsg(QQMessage{
			Action: "send_msg",
			QQMsg: struct {
				MessageType string `json:"message_type"`
				UserId      int    `json:"user_id"`
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
				UserId      int    `json:"user_id"`
				GroupID     int    `json:"group_id"`
				Message     string `json:"message"`
			}{
				MessageType: "group",
				GroupID:     sender.ChatID,
				Message:     fmt.Sprintf("[CQ:at,qq=%d][CQ:image,file=base64://%s,type=show]", sender.UserID, base64.StdEncoding.EncodeToString(msg)),
			},
			Echo: "",
		})
	case "tg":
		SendTgImg(sender.UserID, msg)
	case "tgg":
		SendTggImg(sender.ChatID, sender.UserID, msg, sender.MessageID, sender.Username)
	case "wx":
		SendWxImg(sender.WxId, msg)
	case "wxg":
		SendWxImg(sender.WxGroupId, msg)

	}

}
func (sender *Sender) SendImg2(msg string) {
	imageFormats := []string{".jpg", ".jpeg", ".png", ".gif", ".bmp"}
	isImage := false
	for _, format := range imageFormats {
		pattern := fmt.Sprintf(`^https?://.*%s$`, regexp.QuoteMeta(format))
		if match, _ := regexp.MatchString(pattern, msg); match {
			isImage = true
			break
		}
	}

	if !isImage {
		logs.Info("传递信息非纯地址URL")
		return
	}
	switch sender.Type {
	case "qq":
		SendQQMsg(QQMessage{
			Action: "send_msg",
			QQMsg: struct {
				MessageType string `json:"message_type"`
				UserId      int    `json:"user_id"`
				GroupID     int    `json:"group_id"`
				Message     string `json:"message"`
			}{
				UserId:  sender.UserID,
				GroupID: 0,
				Message: fmt.Sprintf("[CQ:image,file=%s]", msg),
			},
			Echo: "",
		})
	case "qqg":
		SendQQMsg(QQMessage{
			Action: "send_msg",
			QQMsg: struct {
				MessageType string `json:"message_type"`
				UserId      int    `json:"user_id"`
				GroupID     int    `json:"group_id"`
				Message     string `json:"message"`
			}{
				MessageType: "group",
				GroupID:     sender.ChatID,
				Message:     fmt.Sprintf("[CQ:at,qq=%d][CQ:image,file=%s]", sender.UserID, msg),
			},
			Echo: "",
		})
	case "tg":
		SendTgImg2(sender.UserID, msg)
	case "tgg":
		SendTggImg2(sender.ChatID, sender.UserID, msg, sender.MessageID, sender.Username)
	case "wx":
		SendWxImg2(sender.WxId, msg)
	case "wxg":
		SendWxImg2(sender.WxGroupId, msg)
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
				if sender.Type == "wx" {
					sender.Reply("你尚未绑定🐶东账号，请对QQ机器人发送绑定微信获取Code绑定QQ数据，或者是新用户请直接发送登录。")
				} else {
					sender.Reply("你尚未绑定🐶东账号，请发送【登录】上车。")
				}
				return errors.New("你尚未绑定🐶东账号，请发送【登录】上车。")
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

	//拉人进微信群
	{
		Command: []string{"拉1群"},
		Handle: func(sender *Sender) interface{} {
			if sender.Type == "wxg" { // 如果是群聊消息
				if sender.IsAdmin { // 如果是管理员
					ExportEnv(&Env{
						Name:  "InviteWxGroupID",
						Value: sender.WxGroupId,
					})
					return "已将此群设为拉群目标"
				} else { // 如果不是管理员
					return "只有管理员可以设置拉群目标"
				}
			} else if sender.Type == "wx" { // 如果是私聊消息
				if sender.IsAdmin { // 如果是管理员
					return "管理员请不要对机器人私聊此命令"
				} else { // 如果不是管理员
					env := GetEnv("InviteWxGroupID")
					if env != "" {
						InviteGroup(sender.WxId, env)
						return nil
					} else {
						return "未设置拉群目标！"
					}
				}
			} else { // 如果不是微信群聊或私聊消息
				sender.Reply("请添加微信机器人：Shi0718-c 后，在回复 拉群，加入群聊 ")
				return nil
			}
		},
	},

	{
		Command: []string{"拉群"},
		Handle: func(sender *Sender) interface{} {
			// 确定用户ID和类型

			if sender.Type == "tg" { // 如果是Telegram用户
			} else { // 如果是QQ用户
			}
			if sender.Type == "wxg" { // 如果是群聊消息
				if sender.IsAdmin { // 如果是管理员
					ExportEnv(&Env{
						Name:  "InviteWxGroupID",
						Value: sender.WxGroupId,
					})
					return "已将此群设为拉群目标"
				} else { // 如果不是管理员
					return "只有管理员可以设置拉群目标"
				}
			} else if sender.Type == "wx" { // 如果是私聊消息
				if sender.IsAdmin { // 如果是管理员
					return "管理员请不要对机器人私聊此命令"
				} else { // 如果不是管理员
					env := GetEnv("InviteWxGroupID")
					if env != "" {
						if Config.Wx.Model == "qx" {
							QxInviteGroup(sender.WxId, env)
						} else if Config.Wx.Model == "my" {
							InviteGroup(sender.WxId, env)
						} else {
							return "未知的拉群模式，请检查配置"
						}
						return nil
					} else {
						return "未设置拉群目标！"
					}
				}
			} else { // 如果不是微信群聊或私聊消息
				sender.Reply("请添加微信机器人：Shi0718-c 后，在回复 拉群，加入群聊")
				return nil
			}
		},
	},

	{
		Command: []string{"关闭拉群"},
		Admin:   true,
		Handle: func(sender *Sender) interface{} {
			UnExportEnv(&Env{
				Name: "InviteWxGroupID",
			})
			sender.Reply("已关闭自动拉群，如果需要开启，请在监听的微信群里面回复拉群")
			return nil
		},
	},

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

	{
		Command: []string{"对话Grok", "对话ai", "ai", "AI"},
		Handle: func(sender *Sender) interface{} {
			sender.Reply("您已进入与 Ai 连续对话模式，发送您的问题开始交流，回复 'q' 退出。")
			c2 := make(chan string)
			AiinputList[sender.UserID] = c2
			go handleGrokChat(sender, c2)
			return nil
		},
	},

	// 更新CK命令（修复语法，和你的结构对齐）
	{
		Command: []string{"更新ck", "更新CK","记录更新"},
		Handle: func(sender *Sender) interface{} {
			return HandleUpdateCK(sender)
		},
	},

	// 记录CK命令（修复语法）
	{
		Command: []string{"记录ck", "提交ck", "记录CK", "提交CK", "记录账号"},
		Handle: func(sender *Sender) interface{} {
			return HandleRecordCK(sender)
		},
	},
{
    Command: []string{"授权禁用了"},
    	Admin:   true,    
    Handle: func(sender *Sender) interface{} {
        // 1. 先给用户返回响应，告知指令已触发
        sender.Reply("开始检查并禁用过期CK，请稍候...")
        // 2. 异步执行禁用逻辑（避免阻塞指令响应）
        go func() {
            DisableExpiredCKs(sender) // 传入sender，便于日志关联用户
        }()
        return nil
    },
},
{
	Command: []string{"检查过期", "check_expire", "push_expire"}, // 设置多个触发命令，方便记忆
	Admin:   true,                                                // 限制仅管理员可用
	Handle: func(sender *Sender) interface{} {
		// 定义提前提醒的天数，例如：3天
		thresholdDays := 3
		
		logs.Info("管理员【%d】手动触发检查即将过期的授权...", sender.UserID)
		
		// 调用之前编写的 CheckExpiringCKs 函数
		// 参数1: 提前多少天提醒 (int)
		// 参数2: 发送者对象 (*Sender)，用于接收执行结果的反馈消息
		CheckExpiringCKs(thresholdDays, sender)
		
		// 注意：CheckExpiringCKs 内部已经包含了 sender.Reply() 逻辑，
		// 所以这里不需要再额外回复，除非你想覆盖默认消息。
		
		return nil
	},
},

	// 查询记录命令（修复语法）
	{
		Command: []string{"记录查询", "查询记录"},
		Handle: func(sender *Sender) interface{} {
			return HandleQueryRecord(sender)
		},
	},


{
	Command: []string{"记录授权", "授权续期"},
	Handle: func(sender *Sender) interface{} {
		return HandleAuthorizeCK(sender)
	},
},

{
    Command: []string{"记录删除", "删除记录"},
    Handle: func(sender *Sender) interface{} {
        return HandleDeleteCK(sender)
    },
},



{
	Command: []string{"店铛铛更新", "更新店铛铛"}, // 双命令触发，贴合原有习惯
	Handle: func(sender *Sender) interface{} {
		qq := sender.UserID // 获取整型用户ID（后续拼接备注用）
		msgChannel := make(chan string)
		ckList[qq] = msgChannel // 注册用户消息通道，复用原有ckList

		// 独立协程处理交互，避免阻塞主逻辑
		go func() {
			// 退出清理：释放通道+删除注册，避免内存泄漏
			defer func() {
				close(msgChannel)
				delete(ckList, qq)
			}()

			// 通用输入函数：支持q退出、1分钟超时、自动去空格，复用原有逻辑
			getInput := func(prompt string) string {
				sender.Reply(prompt)
				select {
				case input := <-msgChannel:
					input = strings.TrimSpace(input)
					if input == "q" {
						sender.Reply("你已选择退出，操作终止。")
						return ""
					}
					return input
				case <-time.After(time.Minute):
					sender.Reply("输入超时，程序已结束。")
					return ""
				}
			}

			// 步骤1：引导输入原有手机号（青龙中为DDD_ACCOUNTS变量，格式手机号#密码）
			oldPhone := getInput("请输入你店铛铛原有手机号（用于校验青龙已存在账号）：")
			if oldPhone == "" {
				return
			}
			sender.Reply(fmt.Sprintf("正在校验手机号【%s】是否存在于青龙面板...", oldPhone))

			// 步骤2：调用Python脚本校验手机号（参数：手机号、目标变量名DDD_ACCOUNTS）
			chkCmd := exec.Command("python3", "scripts/check_ddd_phone.py", oldPhone, "DDD_ACCOUNTS")
			var chkStdout, chkStderr bytes.Buffer
			chkCmd.Stdout = &chkStdout
			chkCmd.Stderr = &chkStderr
			if err := chkCmd.Run(); err != nil {
				sender.Reply(fmt.Sprintf("手机号校验失败：%s", chkStderr.String()))
				return
			}
			chkResult := strings.TrimSpace(chkStdout.String())
			if chkResult != "found" { // Python脚本返回found表示校验通过
				sender.Reply(fmt.Sprintf("校验失败：手机号【%s】未在青龙找到请确认后重试", oldPhone))
				return
			}
			sender.Reply("手机号校验通过，该账号已存在于青龙面板！本次操作不会更改账号和密码")

			// 步骤3：引导输入新的备注名（后续拼接备注名/用户ID）
			newRemarks := getInput("请输入新的备注名，会以备注名+自动读取用户id 组合的形式记录用户备注")
			if newRemarks == "" {
				return
			}

			// 步骤4：拼接最终备注格式 备注名/用户ID（用户ID转字符串，避免类型问题）
			uidStr := strconv.Itoa(qq)
			finalRemarks := fmt.Sprintf("%s/%s", newRemarks, uidStr)
			sender.Reply(fmt.Sprintf("即将更新青龙备注为：【%s】，开始执行更新操作...", finalRemarks))

			// 步骤5：调用Python脚本更新青龙备注（参数：手机号、最终备注、目标变量名）
			updCmd := exec.Command("python3", "scripts/update_ddd_remarks.py", oldPhone, finalRemarks, "DDD_ACCOUNTS")
			var updStdout, updStderr bytes.Buffer
			updCmd.Stdout = &updStdout
			updCmd.Stderr = &updStderr
			if err := updCmd.Run(); err != nil {
				sender.Reply(fmt.Sprintf("店铛铛备注更新失败：%s", updStderr.String()))
				return
			}

			// 步骤6：更新成功反馈
			updResult := strings.TrimSpace(updStdout.String())
			sender.Reply(fmt.Sprintf("店铛铛更新成功！青龙面板【DDD_ACCOUNTS】变量备注已更新为：%s", finalRemarks))
			if updResult != "" {
				sender.Reply(fmt.Sprintf("更新详情：%s", updResult))
			}
		}()

		return nil
	},
},
{
	Command: []string{"更新茄皇", "茄皇更新"},
	Handle: func(sender *Sender) interface{} {
		activityCodes := map[string]string{
			"1": "TYQH", // 茄皇
		}
		qq := sender.UserID
		qqStr := strconv.Itoa(qq)
		msgChannel := make(chan string)
		ckList[qq] = msgChannel

		go func() {
			defer func() {
				close(msgChannel)
				delete(ckList, qq)
			}()

			// 通用输入获取函数（保留原逻辑：超时1分钟、q退出、去空格）
			getInput := func(prompt string) string {
				sender.Reply(prompt)
				select {
				case input := <-msgChannel:
					input = strings.TrimSpace(input)
					if input == "q" {
						sender.Reply("你已选择退出程序，操作终止。")
						return ""
					}
					return input
				case <-time.After(time.Minute):
					sender.Reply("输入超时，程序已结束。")
					return ""
				}
			}

			// 步骤1：选择活动代号（仅茄皇，保留原逻辑）
			envSelect := getInput("请选择活动代号（输入数字），任意地方输入q可退出程序：\n1. TYQH（茄皇）\n")
			if envSelect == "" {
				return
			}
			envName, ok := activityCodes[envSelect]
			if !ok {
				sender.Reply("活动代号选择错误")
				return
			}

			// 步骤2：输入WID（核心调整：原先输旧CK，现直接输WID）
			wid := getInput("请输入茄皇的WID（小程序个人中心的客户编号）：")
			if wid == "" {
				return
			}

			// 步骤3：调用Python脚本【通过WID查询青龙CK并校验是否含手机号】
chkPhoneCmd := exec.Command("python3", "scripts/check_ck_has_phone.py", wid, envName)
var chkPhoneStdout, chkPhoneStderr bytes.Buffer
chkPhoneCmd.Stdout = &chkPhoneStdout
chkPhoneCmd.Stderr = &chkPhoneStderr
var hasPhone bool // 仅保留：标记CK是否包含手机号（后续逻辑需使用）
if err := chkPhoneCmd.Run(); err != nil {
    sender.Reply(fmt.Sprintf("CK校验失败：%s", chkPhoneStderr.String()))
    return
}
chkPhoneResult := strings.TrimSpace(chkPhoneStdout.String())
// 解析脚本返回结果：格式为「状态|当前CK值」，状态包括found_has_phone/found_no_phone/not_found
resultParts := strings.SplitN(chkPhoneResult, "|", 2)
if len(resultParts) != 2 {
    sender.Reply(fmt.Sprintf("CK校验异常：%s", chkPhoneResult))
    return
}
status, _ := resultParts[0], resultParts[1] // 第二个值未使用，用_忽略
switch status {
case "found_has_phone":
    hasPhone = true
    sender.Reply("检测到该WID对应的CK已包含手机号（格式：手机号#WID），无需重复输入手机号！")
case "found_no_phone":
    hasPhone = false
    sender.Reply("检测到该WID对应的CK未包含手机号，按流程输入手机号完成更新...")
case "not_found":
    sender.Reply("你没有挂过茄皇，无法进行更新操作，终止！上车请发【记录ck】")
    return
default:
    sender.Reply(fmt.Sprintf("CK校验异常：%s", status))
    return
}

			var newCk string // 存储构造后的新CK（仅无手机号时赋值）
			if !hasPhone {
				// 步骤4：无手机号时，提示输入手机号（非空校验）
				phone := getInput("请输入手机号：")
				if phone == "" {
					return
				}
				// 校验手机号格式（可选：简单校验11位数字，避免无效输入）
				if len(phone) != 11 {
					sender.Reply("手机号格式错误，请输入11位数字！")
					return
				}
				// 构造新CK：手机号#WID 格式
				newCk = fmt.Sprintf("%s#%s", phone, wid)
				sender.Reply(fmt.Sprintf("新CK已构造完成：%s", newCk))
			}

			// 步骤5：统一输入备注（无论是否含手机号，均执行此步骤）
			remarks := getInput("请输入备注名：")
			if remarks == "" {
				return
			}
			// 拼接最终备注：备注/用户ID（核心格式，不变）
			finalRemarks := fmt.Sprintf("%s/%s", remarks, qqStr)
			sender.Reply(fmt.Sprintf("备注已处理完成：%s，开始执行更新流程...", finalRemarks))

			// 步骤6：积分校验（保留原逻辑，未修改）
			value3 := GetEnv("up" + envName)
			if value3 == "" {
				sender.Reply(fmt.Sprintf("%s未开启更新功能", envName))
				return
			}
			coin := GetCoin(sender.UserID)
			jbcoin, _ := strconv.Atoi(value3)
			if coin < jbcoin {
				sender.Reply(fmt.Sprintf("积分不足，%s更新需要%d个积分，请直接私聊微信机器人转账，1元=100积分，转账成功即可完成积分充值，或者复制网址http://180.152.5.230:8005/到其他浏览器打开购买卡密充值", envName, jbcoin))
				return
			}

			// 步骤7：执行更新脚本（核心适配：传参随是否含手机号变化）
			// 传参规则：有手机号→newCk传空（仅更备注），无手机号→传新CK（更CK+备注）
			var cmd *exec.Cmd
			if hasPhone {
				cmd = exec.Command("python3", "scripts/updata_tyqh.py", "", finalRemarks, envName, wid)
			} else {
				cmd = exec.Command("python3", "scripts/updata_tyqh.py", newCk, finalRemarks, envName, wid)
			}
			var stdout, stderr bytes.Buffer
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr
			err := cmd.Run()
			if err == nil {
				outputStr := stdout.String()
				if strings.Contains(outputStr, "更新成功") {
					RemCoin(sender.UserID, jbcoin)
					sender.Reply(fmt.Sprintf("更新%s账号成功，已扣除%d个积分，剩余积分%d", envName, jbcoin, GetCoin(sender.UserID)))
					sender.Reply("更新成功！")
				} else {
					sender.Reply(fmt.Sprintf("更新提示：%s", outputStr))
				}
			} else {
				sender.Reply(fmt.Sprintf("更新失败：%s", stderr.String()))
			}
		}()

		return nil
	},
},




	

	// {
	// 	Command: []string{"密码登陆", "密码登录"},
	// 	Handle: func(sender *Sender) interface{} {
	// 		value := GetEnv("grouplogin")
	// 		if value == "" && (sender.Type == "qqg" || sender.Type == "wxg") {
	// 			// 如果是群聊且没有登录
	// 			Auto := &UserSession{}
	// 			go Autojdck(sender, Auto)
	// 		} else {
	// 			Auto := &UserSession{}
	// 			go Autojdck(sender, Auto)
	// 		}
	// 		return nil
	// 	},
	// },

	// {
	// 	Command: []string{"账密更新"},
	// 	Admin:   true,
	// 	Handle: func(sender *Sender) interface{} {
	// 		sender.Reply("开始检测ck，并更新ck")

	// 		go UpAutoCookie()

			
	// 	return nil
	// 	},
	// },

	// {
	// 	Command: []string{"所有账密更新", "更新所有账密"},
	// 	Admin:   true,
	// 	Handle: func(sender *Sender) interface{} {
	// 		sender.Reply("开始更新所有账密账号")

	// 		all_UpAutoCookie()

	// 		sender.Reply("所有账号账密登录任务执行完毕")

	// 		return nil
	// 	},
	// },

	

	// {
	// 	Command: []string{"账密检测", "密码检测"},
	// 	Handle: func(sender *Sender) interface{} {
	// 		sender.Reply("开始密码检测...")
	// 		go initAutoCookie()
	// 		return nil
	// 	},
	// },

	// {
	// 	Command: []string{"推送所有失效账密"},
	// 	Handle: func(sender *Sender) interface{} {
	// 		sender.Reply("开始密码检测...")
	// 		go all_initAutoCookie()
	// 		return nil
	// 	},
	// },

	{
		Command: []string{"短信登录", "短信登陆"},
		Handle: func(sender *Sender) interface{} {
			value := GetEnv("grouplogin")
			if value == "" && (sender.Type == "qqg" || sender.Type == "wxg") {
				sender.Reply("\n1、请添加本机器人私聊登录，避免信息泄露，建议使用【密码登录】，不掉线\n2、上车后请到京东-我的-支付设置，关闭小额免密，同时开启虚拟资产验密")
			} else {
				c2 := make(chan string)
				smsList[sender.UserID] = c2
				sender.Reply("请输入手机号...")
				go SmsSelect(sender, c2, "Nolan")
			}
			return nil
		},
	},

	{
		Command: []string{"删掉"},
		Admin:   true,
		Handle: func(sender *Sender) interface{} {
			cost := sender.Contents[0]

			if Config.QQID == 764763903 {
				DeleteCk(cost, "ck")
				return "删除成功"
			}
			if len(sender.Contents) >= 1 {
				sender.Reply(fmt.Sprintf("开始删除%s行", cost))
				rsp := cmd(fmt.Sprintf("python3 ./tou_ck.py %s", cost), &Sender{})
				sender.Reply(rsp)
			} else {
				sender.Reply("请配置开始信息")
			}
			return nil
		},
	},

	{
		Command: []string{"停助力", "停止助力"},
		Handle: func(sender *Sender) interface{} {
			if sender.UserID == 995336676 || sender.IsAdmin {
				rsp := cmd(fmt.Sprintf(`bash stop.sh`), &Sender{})
				return rsp
			} else {
				sender.Reply("无权操作")
			}
			return nil
		},
	},

	
	{
		Command: []string{"创建1卡密"},
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
		Command: []string{"赠送11卡密"},
		Admin:   true,
		Handle: func(sender *Sender) interface{} {
			if Config.VIP == true {
				contents := sender.Contents
				logs.Info(contents[0])
				num, _ := strconv.Atoi(contents[0])
				value, _ := strconv.Atoi(contents[1])
				return create_ZSKey(num, value)
			}
			return "非VIP用户"
		},
	},


{
    Command: []string{"赠送卡密"},
    Admin:   true,
    Handle: func(sender *Sender) interface{} {
        // 1. 首先校验VIP权限
        if !Config.VIP {
            return "非VIP用户"
        }

        // 2. 创建通道存储用户输入，并注册到ckList中
        msg := make(chan string)
        ckList[sender.UserID] = msg

        // 3. 启动goroutine处理引导式交互
        go func() {
            // 确保函数结束时从ckList中删除用户通道，防止内存泄漏
            defer delete(ckList, sender.UserID)

            // 步骤1：提示用户输入要赠送的卡密数量
            sender.Reply("请输入要赠送的卡密数量：")
            numInput := <-msg
            numInput = strings.TrimSpace(numInput)
            num, err := strconv.Atoi(numInput)
            // 校验数量输入的合法性：必须是正整数
            if err != nil || num <= 0 {
                sender.Reply("输入的数量不合法，请输入正整数！")
                return
            }

            // 步骤2：提示用户输入卡密对应的面额（数值）
            sender.Reply("请输入卡密对应的面额（可输入负数）：")
            valueInput := <-msg
            valueInput = strings.TrimSpace(valueInput)
            value, err := strconv.Atoi(valueInput)
            // 校验数值输入的合法性：仅校验是否为整数（允许负数、0、正数）
            if err != nil {
                sender.Reply("输入的面额不合法，请输入整数（可负数）！")
                return
            }

            // 步骤3：调用赠送卡密函数并获取结果（字符串类型）
            result := create_ZSKey(num, value)
            
            // 步骤4：回复赠送结果给用户（避免nil比较，适配字符串类型）
       
                sender.Reply(fmt.Sprintf("成功赠送 %d 个卡密，面额为 %d\n赠送结果：\n\n%s", num, value, result))
   
        }()

        // 返回nil表示由goroutine处理后续回复，无需立即返回内容
        return nil
    },
},

{
    Command: []string{"创建卡密"},
    Admin:   true,
    Handle: func(sender *Sender) interface{} {
        // 1. 首先校验VIP权限
        if !Config.VIP {
            return "非VIP用户"
        }

        // 2. 创建通道存储用户输入，并注册到ckList中
        msg := make(chan string)
        ckList[sender.UserID] = msg

        // 3. 启动goroutine处理引导式交互
        go func() {
            // 确保函数结束时从ckList中删除用户通道，防止内存泄漏
            defer delete(ckList, sender.UserID)

            // 步骤1：提示用户输入要创建的卡密数量
            sender.Reply("请输入要创建的卡密数量：")
            numInput := <-msg
            numInput = strings.TrimSpace(numInput)
            num, err := strconv.Atoi(numInput)
            // 校验数量输入的合法性：必须是正整数
            if err != nil || num <= 0 {
                sender.Reply("输入的数量不合法，请输入正整数！")
                return
            }

            // 步骤2：提示用户输入卡密对应的面额（数值）
            sender.Reply("请输入卡密对应的面额：")
            valueInput := <-msg
            valueInput = strings.TrimSpace(valueInput)
            value, err := strconv.Atoi(valueInput)
            // 校验数值输入的合法性：仅校验是否为整数（允许负数、0、正数）
            if err != nil {
                sender.Reply("请输入正确的面额")
                return
            }

            // 步骤3：调用创建卡密函数并返回结果
            result := createKey(num, value)
            
            // 步骤4：回复创建结果给用户
  
                sender.Reply(fmt.Sprintf("成功创建 %d 个卡密，面额为 %d\n创建结果：\n\n%s", num, value, result))
  
        }()

        // 返回nil表示由goroutine处理后续回复，无需立即返回内容
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

	{

		Command: []string{"我的plus", "我的PLUS"},
		Handle: func(sender *Sender) interface{} {
			sender.handleJdCookies(func(ck *JdCookie) {
				cc := fmt.Sprintf("pt_key=%s;pt_pin=%s;", ck.PtKey, ck.PtPin)
				rsp := cmd(fmt.Sprintf(`python3 ./jd_plus_score.py "%s"`, cc), &Sender{})
				sender.Reply(rsp)
			})

			return nil
		},
	},


{
	Command: []string{"京豆排名", "富豪榜", "京豆榜"},
	Handle: func(sender *Sender) interface{} {
		// 定义存储排名信息的结构体
		type BeanRanking struct {
			Rank     int    // 排名
			Nickname string // 昵称
			BeanNum  int    // 京豆数量（整型）
		}
		var beanRankings []BeanRanking
		var allCookies []JdCookie // 假设你的Cookie结构体类型为JdCookie

		// 1. 查询数据库所有JD Cookie（去掉Available条件，不做数据库排序）
		allCookies = GetJdCookies(func(sb *gorm.DB) *gorm.DB {
			return sb // 全量查询，无过滤条件
		})

		// 2. 先将所有Cookie转换为带数值BeanNum的结构，方便排序
		type CookieWithNum struct {
			JdCookie
			Num int // 京豆数量的数值类型
		}
		var cookieWithNums []CookieWithNum

		for _, cookie := range allCookies {
			// 将字符串类型的BeanNum转换为整型，失败则设为0
			num, err := strconv.Atoi(cookie.BeanNum)
			if err != nil {
				num = 0 // 转换失败默认设为0，也可改为continue跳过
			}
			cookieWithNums = append(cookieWithNums, CookieWithNum{
				JdCookie: cookie,
				Num:      num,
			})
		}

		// 3. 按京豆数量数值降序排序（避免字符串字典序排序错误）
		sort.Slice(cookieWithNums, func(i, j int) bool {
			return cookieWithNums[i].Num > cookieWithNums[j].Num
		})

		// 4. 取前20名构造排名列表
		for i, c := range cookieWithNums {
			if i >= 20 { // 仅保留前20名
				break
			}
			beanRankings = append(beanRankings, BeanRanking{
				Rank:     i + 1,
				Nickname: c.Nickname,
				BeanNum:  c.Num,
			})
		}

		// 5. 构造回复消息
		var replyMessage string
		// 存储前三名的恭喜语（后续拼接在末尾）
		var congratulateList []string
		// 数字对应的emoji序号映射（4-20，前三名不再使用）
		numEmojis := map[int]string{
			4:  "4️⃣", 5:  "5️⃣", 6:  "6️⃣", 7:  "7️⃣", 8:  "8️⃣",
			9:  "9️⃣", 10: "🔟", 11: "⑪", 12: "⑫", 13: "⑬",
			14: "⑭", 15: "⑮", 16: "⑯", 17: "⑰", 18: "⑱",
			19: "⑲", 20: "⑳",
		}
		// 奖牌图标（前3名特殊标识）
		medalEmojis := map[int]string{
			1: "🏅", 2: "🥈", 3: "🥉",
		}
		// 前三名恭喜语模板（动态填充昵称）
		congratulateTpl := map[int]string{
			1: "🎉 恭喜【%s】荣获京豆富豪榜冠军！太厉害了～",
			2: "🥳 恭喜【%s】荣获京豆富豪榜亚军！继续加油～",
			3: "👏 恭喜【%s】荣获京豆富豪榜季军！表现超棒～",
		}

		if len(beanRankings) == 0 {
			replyMessage = "暂无京豆数据可统计～"
		} else {
			replyMessage = "🏆 京豆富豪榜 Top20 🏆\n\n"
			for _, ranking := range beanRankings {
				// 前3名：仅奖牌图标 + 昵称 ------ 京豆（移除数字emoji）
				if medal, ok := medalEmojis[ranking.Rank]; ok {
					// 季军后增加一个空格，保证与后续排名的视觉对齐
					if ranking.Rank == 3 {
						replyMessage += fmt.Sprintf("%s  %s ------ %d 京豆\n", medal, ranking.Nickname, ranking.BeanNum)
					} else {
						replyMessage += fmt.Sprintf("%s %s ------ %d 京豆\n", medal, ranking.Nickname, ranking.BeanNum)
					}
					// 动态生成恭喜语并存储
					congratulateList = append(congratulateList, fmt.Sprintf(congratulateTpl[ranking.Rank], ranking.Nickname))
				} else {
					// 4-20名：数字emoji + 昵称 ------ 京豆
					numEmoji := numEmojis[ranking.Rank]
					replyMessage += fmt.Sprintf("%s %s ------ %d 京豆\n", numEmoji, ranking.Nickname, ranking.BeanNum)
				}
			}

			// 若有前三名恭喜语，添加分割线并拼接
			if len(congratulateList) > 0 {
				replyMessage += "\n------------------------\n"
				for _, msg := range congratulateList {
					replyMessage += msg + "\n"
				}
			}
		}

		// 发送回复
		sender.Reply(replyMessage)

		return nil
	},
},


	{
		Command: []string{"我的排名", "我的优先级"},
		Handle: func(sender *Sender) interface{} {
			// 检查限流次数是否超过
			if getLimit(sender.UserID, 1) || sender.IsAdmin {
				// 从发送者中获取用户的 QQ 号码
				userQQ := sender.UserID

				// 初始化变量，用于存储用户的排名、用户名和优先级，以及用户是否拥有 JD Cookie
				var userRankings []struct {
					Rank     int
					Nickname string
					Username string
					Priority int
				}
				var hasCookie bool

				// 获取所有的 JD Cookie
				allCookies := GetJdCookies(func(sb *gorm.DB) *gorm.DB {
					return sb.Where(fmt.Sprintf("%s = ? ", Available), True)
				})

				// 遍历所有的 JD Cookie
				for i, cookie := range allCookies {
					// 检查当前的 JD Cookie 是否属于该用户且可用
					if cookie.QQ == userQQ {
						// 如果用户有可用的 JD Cookie，添加排名、用户名和优先级到用户排名列表
						userRankings = append(userRankings, struct {
							Rank     int
							Nickname string
							Username string
							Priority int
						}{
							Rank:     i + 1,
							Nickname: cookie.Nickname,
							Username: cookie.PtPin,
							Priority: cookie.Priority,
						})
						hasCookie = true
					}
				}

				// 检查用户是否有 JD Cookie
				if hasCookie {
					// 如果用户有 JD Cookie，构造回复消息
					var replyMessage string
					if len(userRankings) == 1 {
						replyMessage = fmt.Sprintf("您的昵称：%s，您的用户名：%s，优先级：%d ，排名是第 %d 位", userRankings[0].Nickname, userRankings[0].Username, userRankings[0].Priority, userRankings[0].Rank)
					} else {
						replyMessage = "您账号排名信息："
						for idx, ranking := range userRankings {
							replyMessage += fmt.Sprintf("\n%d、昵称：%s，用户名：%s，优先级：%d ，排名：%d 位", idx+1, ranking.Nickname, ranking.Username, ranking.Priority, ranking.Rank)
						}
					}
					sender.Reply(replyMessage)
				} else {
					// 如果用户没有 JD Cookie，通知他们
					sender.Reply("您账号可能失效，或者尚未登录，无法查询排名。")
				}
			} else {
				// 超过限制，回复消息
				sender.Reply(fmt.Sprintf("每日限流%d次，已超过今日限制", Config.Lim))
			}

			return nil // 返回 nil，因为没有需要返回的结果
		},
	},

	{
		Command: []string{"优先级排名", "总排名", "排行榜"},
		Handle: func(sender *Sender) interface{} {
			if getLimit(sender.UserID, 1) || sender.IsAdmin {
				// 获取所有的 JD Cookie，并按照优先级降序排列，取前30名
				allCookies := GetJdCookies(func(sb *gorm.DB) *gorm.DB {
					return sb.Where("Available = ?", "true").Order("priority DESC").Limit(40)
				})

				// 构造回复消息
				var replyMessage string
				for i, cookie := range allCookies {
					replyMessage += fmt.Sprintf("\n%d、用户名：%s，优先级：%d", i+1, cookie.Nickname, cookie.Priority)
				}

				// 如果没有任何账号信息，则返回相应的消息
				if replyMessage == "" {
					replyMessage = "没有可用的账号信息。"
				} else {
					replyMessage = "前40名有效账号的优先级如下：\n--------------------------------" + replyMessage
				}

				sender.Reply(replyMessage)
			} else {
				// 超过限制，回复消息
				sender.Reply(fmt.Sprintf("每日限流%d次，已超过今日限制", Config.Lim))
			}

			return nil // 返回 nil，因为没有需要返回的结果
		},
	},

{
    Command: []string{"微信协议掉线推送"},
    Admin:   true,
    Handle: func(sender *Sender) interface{} {
        // 调用上面编写的函数
       CheckWxOfflineAndNotify() 
        return nil
    },
},


{
    Command: []string{"微信扫码登录","微信扫码登陆"},
    Handle: func(sender *Sender) interface{} {
            if sender.Type != "wx" {
            sender.Reply("此功能仅支持微信私聊使用")
            return nil
        }
        WXID_CODE(sender)
        return nil
    },
},


{
    Command: []string{"微信重新登录","微信重新登陆"},
    Handle: func(sender *Sender) interface{} {
        if sender.Type != "wx" {
            sender.Reply("此功能仅支持微信私聊使用")
            return nil
        }
        WXID_RELOGIN(sender)
        return nil
    },
},
{
    Command: []string{"微信唤醒登录","微信唤醒登陆"},
    Handle: func(sender *Sender) interface{} {
        if sender.Type != "wx" {
            sender.Reply("此功能仅支持微信私聊使用")
            return nil
        }
        WXID_WAKE_LOGIN(sender)
        return nil
    },
},
{
    Command: []string{"微信登出"},
    Handle: func(sender *Sender) interface{} {
        if sender.Type != "wx" && sender.Type != "wxg" {
            sender.Reply("此功能仅支持微信/群聊使用")
            return nil
        }
        WXID_LOGOUT(sender)
        return nil
    },
},
{
    Command: []string{"微信删除","删除微信"},
    Handle: func(sender *Sender) interface{} {
        if sender.Type != "wx" && sender.Type != "wxg" {
            sender.Reply("此功能仅支持微信/群聊使用")
            return nil
        }
        WXID_DELETE(sender)
        return nil
    },
},
{
    Command: []string{"微信设备状态"},
    Admin:   true,
    Handle: func(sender *Sender) interface{} {
        WXID_USER_STATUS(sender)
        return nil
    },
},
{
    Command: []string{"微信状态"},
    Handle: func(sender *Sender) interface{} {
        // 允许 wx  + wxg 群聊
        if sender.Type != "wx" && sender.Type != "wxg" {
            sender.Reply("此功能仅支持微信/群聊使用")
            return nil
        }
        WXID_MY_STATUS(sender)
        return nil
    },
},



	// {
	// 	Command: []string{"扫码", "扫码登录"},

	// 	Handle: func(sender *Sender) interface{} {
	// 		if sender.IsAdmin {
	// 			sender.Reply("开始京东扫码登录")
	// 		} else {
	// 			jbcoin := getWxScanLoginCost()
	// 			if jbcoin <= 0 {
	// 				return "未开启扫码登录"
	// 			}
	// 			coin := GetCoin(sender.UserID)
	// 			if coin < jbcoin {
	// 				return fmt.Sprintf("扫码登录需要%d个积分,当前积分%d不足，请直接微信机器人转账，1元=100积分，转账成功即可完成积分充值，或者复制网址http://180.152.5.230:8005/到其他浏览器打开购买卡密充值", jbcoin, coin)
	// 			}

	// 			actualDeduct := RemCoin(sender.UserID, jbcoin)
	// 			if actualDeduct > coin {
	// 				sender.Reply(fmt.Sprintf("系统异常：积分扣除失败，请联系管理员"))
	// 				return nil
	// 			}
	// 			sender.Reply(fmt.Sprintf("扫码即将开始，已扣除%d个积分,剩余%d", jbcoin, GetCoin(sender.UserID)))
	// 			sender.Reply("上车后请到京东-我的-支付设置，关闭小额免密，同时开启虚拟资产验密")
	// 		}

	// 		NolanGetJdQrImg(sender)
	// 		return nil
	// 	},
	// },

	{
		Command: []string{"任务菜单", "农场浇水", "新农场浇水", "一键保价", "自动挖宝", "一键评价", "话费签到", "调查问卷"},
		Handle: func(sender *Sender) interface{} {
			sender.Reply("任务菜单适用于任务漏跑的，自己可以手动执行任务补跑")
			id := sender.UserID
			var idType string
			if sender.Type == "tg" {
				idType = Telegram
			} else {
				idType = QQ
			}
			cks := GetJdCookies(func(sb *gorm.DB) *gorm.DB {
				return sb.Where(fmt.Sprintf("%s = ? and %s = ?", idType, Available), id, True)
			})

			if len(cks) > 0 {
				// 显示功能菜单
				msgs := []string{
					"请输入序号选择要执行的任务：",
					//				"1. 农场浇水",
					"1. 新农场浇水",
					"2. 种豆得豆任务",
					"3. 话费积分任务",
					//		"5. 特价APP签到提现",
					"4. 一键保价",
					"5. 一键评价",
					"6. 删除垃圾券（慎用，会误删，到已删除券恢复）",
					"7. 问卷调查得豆",
				//	"8. 京东外卖券",

					//     "7. 自动挖宝",
					"如需退出请回复'q'退出流程：",
				}
				sender.Reply(strings.Join(msgs, "\n\n"))

				// 创建一个通道用于接收用户的选择
				msg := make(chan string)
				inputList[sender.UserID] = msg
				go handleUserChoice(sender, msg, cks)
			} else {
				sender.Reply("在线账号已全部失效，请对机器人发送“密码登录”")
				return nil
			}
			return nil
		},
	},

{
    Command: []string{"sign", "打卡", "签到"},
    Handle: func(sender *Sender) interface{} {
        sender.Reply("🔄 机器人端打卡功能升级中\n\n📱 请前往以下渠道完成打卡：\n\n💻 电脑网页用户\n请复制以下地址到浏览器打开：\nhttp://180.152.5.230:5701\n\n📲 手机用户\n请回复【狗东下载】，下载大师全新开发的软件\n（支持上车项目查询、打卡查询等各种功能）\n\n⏳ 升级完成后将第一时间通知大家，感谢理解！")
        return nil
    },
},

	

	



	{
		Command: []string{"账号管理", "任务屏蔽", "取消屏蔽", "账号状态管理", "加入内测", "屏蔽任务"},
	Admin:   true,
		Handle: func(sender *Sender) interface{} {
			// 获取用户的账号列表
			id := sender.UserID
			var idType string
			if sender.Type == "tg" {
				idType = Telegram
			} else {
				idType = QQ
			}

			// 获取用户的有效账号
			cks := GetJdCookies(func(sb *gorm.DB) *gorm.DB {
				return sb.Where(fmt.Sprintf("%s = ? and %s = ?", idType, Available), id, True)
			})

			// 如果账号存在，继续流程
			if len(cks) > 0 {
				msg := make(chan string)
				ckList[sender.UserID] = msg

				// 提示用户选择操作
				msgs := []string{
					"请输入序号选择你要执行的操作，输入q可退出程序：\n",
					"1、任务屏蔽----不跑任何脚本等账号变白没有收入",
					"2、正常状态----跑全部脚本全员公测据群友反馈不黑了",
					"3、加入内测----只跑新农场任务和种豆任务",
				}
				sender.Reply(strings.Join(msgs, "\n"))
				time.Sleep(time.Second * 1)

				go func() {
					defer delete(ckList, sender.UserID) // 删除消息通道

					// 设置超时等待用户选择
					select {
					case response := <-msg:
						if response == "q" {
							sender.Reply("已退出程序。")
							return
						}

						// 根据用户选择执行相应的操作
						switch response {
						case "1":
							// 任务屏蔽
							msgs := []string{
								"请回复下面账号序号，选择后将屏蔽任务，如需退出请回复'q'退出登录流程：\n --------------------------------",
							}
							msgs = append(msgs, "0、所有账号") // 添加选择所有账号的选项
							for i, ck := range cks {
								status := "任务正常运行"
								if ck.Hack == True {
									status = "屏蔽状态"
								}
								if ck.Hack == True && ck.Appoint == True {
									status = "内测状态"
								}
								if ck.Hack == False {
									status = "正常状态"
								}
								msgs = append(msgs, fmt.Sprintf("%d、%s 【%s】", i+1, ck.Nickname, status)) // 序号从1开始
							}
							sender.Reply(strings.Join(msgs, "\n"))

							// 等待用户输入序号或退出
							select {
							case response = <-msg:
								if response == "q" {
									sender.Reply("已退出屏蔽流程。")
									return
								}

								index, err := strconv.Atoi(response)
								if err != nil || (index < 0 || index > len(cks)) {
									sender.Reply("输入的序列号无效，已退出，请重新执行命令。")
									return
								}

								if index == 0 {
									// 选择0时，所有账号都执行屏蔽
									for _, ck := range cks {
										ck.Update(Hack, True)     // 设置账号屏蔽
										ck.Update(Appoint, False) // 设置取消内测
									}
									sender.Reply("所有账号的任务已更新为屏蔽状态。")
								} else {
									// 选择账号并更新其状态为屏蔽
									ck := cks[index-1]
									ck.Update(Hack, True)     // 设置账号屏蔽
									ck.Update(Appoint, False) // 设置取消内测
									sender.Reply(fmt.Sprintf("账号:%s的任务状态更新为：屏蔽状态", ck.Nickname))
								}
							case <-time.After(time.Minute):
								sender.Reply("超时未回复，程序已退出。")
								return
							}

						case "2":
							// 任务恢复
							msgs := []string{
								"请回复下面账号序号，选择后将恢复任务，如需退出请回复'q'退出登录流程：\n --------------------------------",
							}
							msgs = append(msgs, "0、所有账号") // 添加选择所有账号的选项
							for i, ck := range cks {
								status := "任务正常运行"
								if ck.Hack == True {
									status = "屏蔽状态"
								}
								if ck.Hack == True && ck.Appoint == True {
									status = "内测状态"
								}
								if ck.Hack == False {
									status = "正常状态"
								}

								msgs = append(msgs, fmt.Sprintf("%d、%s 【%s】", i+1, ck.Nickname, status)) // 序号从1开始
							}
							sender.Reply(strings.Join(msgs, "\n"))

							// 等待用户输入序号或退出
							select {
							case response = <-msg:
								if response == "q" {
									sender.Reply("已退出恢复流程。")
									return
								}

								index, err := strconv.Atoi(response)
								if err != nil || (index < 0 || index > len(cks)) {
									sender.Reply("输入的序列号无效，已退出，请重新执行命令。")
									return
								}

								if index == 0 {
									// 选择0时，所有账号恢复任务
									for _, ck := range cks {
										ck.Update(Hack, False)    // 恢复任务
										ck.Update(Appoint, False) // 恢复账号
									}
									sender.Reply("所有账号的任务已恢复为正常状态。")
								} else {
									// 选择账号并更新其状态为恢复
									ck := cks[index-1]
									ck.Update(Hack, False)    // 恢复任务
									ck.Update(Appoint, False) // 恢复账号
									sender.Reply(fmt.Sprintf("账号:%s的任务状态已恢复为正常状态", ck.Nickname))
								}
							case <-time.After(time.Minute):
								sender.Reply("超时未回复，程序已退出。")
								return
							}

						case "3":
							// 加入内测
							msgs := []string{
								"请回复下面账号序号，选择后将加入内测，如需退出请回复'q'退出登录流程：\n --------------------------------",
							}
							msgs = append(msgs, "0、所有账号") // 添加选择所有账号的选项
							for i, ck := range cks {
								status := "任务正常运行"
								if ck.Hack == True {
									status = "屏蔽状态"
								}
								if ck.Hack == True && ck.Appoint == True {
									status = "内测状态"
								}
								if ck.Hack == False {
									status = "正常状态"
								}
								msgs = append(msgs, fmt.Sprintf("%d、%s 【%s】", i+1, ck.Nickname, status)) // 序号从1开始
							}
							sender.Reply(strings.Join(msgs, "\n"))

							// 等待用户输入序号或退出
							select {
							case response = <-msg:
								if response == "q" {
									sender.Reply("已退出内测流程。")
									return
								}

								index, err := strconv.Atoi(response)
								if err != nil || (index < 0 || index > len(cks)) {
									sender.Reply("输入的序列号无效，已退出，请重新执行命令。")
									return
								}

								if index == 0 {
									// 选择0时，所有账号加入内测
									for _, ck := range cks {
										ck.Update(Hack, True)    // 设置账号为屏蔽状态
										ck.Update(Appoint, True) // 设置账号为内测状态
									}
									sender.Reply("所有账号的任务已更新为加入内测状态。")
								} else {
									// 选择账号并更新其状态为内测
									ck := cks[index-1]
									ck.Update(Hack, True)    // 设置账号为屏蔽状态
									ck.Update(Appoint, True) // 设置账号为内测状态
									sender.Reply(fmt.Sprintf("账号:%s的任务状态更新为：内测状态", ck.Nickname))
								}
							case <-time.After(time.Minute):
								sender.Reply("超时未回复，程序已退出。")
								return
							}

						default:
							sender.Reply("选择错误，已退出程序，请重新发送命令")
						}

					case <-time.After(time.Minute): // 超过1分钟没有回复
						sender.Reply("超时未回复，程序已退出。")
						return
					}
				}()

				return nil
			} else {
				sender.Reply("账号全部失效，或者没有挂机，请发送【密码登录】。")
			}

			return nil
		},
	},



	{
		Command: []string{"更新优先级", "更新车位", "车位更新", "优先级更新", "排名更新", "更新排名"},
		Handle: func(sender *Sender) interface{} {
			value := GetEnv("车位更新")
			if value == "" {
				sender.Reply("优先级系统改革，暂不支持更新")
				return nil
			}

			msg := make(chan string)
			ckList[sender.UserID] = msg
			// 引导用户输入积分

			sender.Reply(fmt.Sprintf("请回复您想要使用的积分数量（当前积分：%d）：", GetCoin(sender.UserID)))

			// 处理用户输入的积分
			go func() {
				defer delete(ckList, sender.UserID) // 清理消息通道
				for {
					select {
					case response := <-msg:
						userCoinStr := response
						userCoin, err := strconv.ParseInt(userCoinStr, 10, 64)
						if err != nil || userCoin <= 0 {
							sender.Reply("输入的积分值无效，请确保输入的是一个正整数。已退出流程。")
							return
						}

						// 获取当前用户积分
						coin := GetCoin(sender.UserID)
						if int(userCoin) > int(coin) {
							sender.Reply("你输入的数字超过了你拥有的积分。")
							return
						}

						// 确定用户ID类型
						id := sender.UserID
						var idType string
						if sender.Type == "tg" {
							idType = Telegram
						} else {
							idType = QQ
						}

						// 获取可用的账号
						cks := GetJdCookies(func(sb *gorm.DB) *gorm.DB {
							return sb.Where(fmt.Sprintf("%s = ? and %s = ?", idType, Available), id, True)
						})

						// 检查是否有可更新的账号
						if len(cks) > 0 {
							// 引导用户选择账号
							msgs := []string{
								"请回复下面账号序号，选择后将更新优先级，并扣除积分。如需退出请回复'q'：\n --------------------------------",
							}
							for i, ck := range cks {
								msgs = append(msgs, fmt.Sprintf("%d、%s", i, ck.Nickname))
							}

							sender.Reply(strings.Join(msgs, "\n"))

							// 处理用户响应
							for {
								select {
								case response := <-msg:
									if response == "q" {
										sender.Reply("已退出更新流程。")
										return
									}

									index, err := strconv.Atoi(response)
									if err != nil || index < 0 || index >= len(cks) {
										sender.Reply("输入的序列号无效，已退出，请重新执行命令。")
										return
									}

									// 更新优先级
									ck := cks[index]
									ck.Update(Priority, ck.Priority+int(userCoin))
									sender.Reply(fmt.Sprintf("账号:%s的优先级已更新为：%d", ck.Nickname, ck.Priority))

									// 扣除积分
									RemCoin(sender.UserID, int(userCoin))
									time.Sleep(time.Second * 2)

									tcoin := GetCoin(sender.UserID)
									sender.Reply(fmt.Sprintf("已扣除%d的积分，当前剩余积分%d", int(userCoin), tcoin))
									return
								}
							}
						} else {
							sender.Reply("未找到可更新的账号。")
						}
						return
					}
				}
			}()
			return nil
		},
	},

	{
		Command: []string{"转移优先级", "优先级转移", "转移车位", "车位转移"},
		Handle: func(sender *Sender) interface{} {
			// 引导用户输入优先级
			sender.Reply("请继续输入你准备要转移的数字，例：100")

			// 等待用户输入优先级
			msg := make(chan string)
			ckList[sender.UserID] = msg

			go func() {
				defer delete(ckList, sender.UserID) // 删除消息通道

				for response := range msg {
					userPriorityStr := response
					userPriority, err := strconv.ParseInt(userPriorityStr, 10, 64)
					if err != nil || userPriority <= 0 {
						sender.Reply("输入的优先级值无效，请确保输入的是一个正整数，已退出")
						return
					}

					id := sender.UserID
					var idType string
					if sender.Type == "tg" {
						idType = Telegram
					} else {
						idType = QQ
					}

					cks := GetJdCookies(func(sb *gorm.DB) *gorm.DB {
						return sb.Where(fmt.Sprintf("%s = ?", idType), id)
					})

					if len(cks) > 0 {
						msgs := []string{
							"请选择要转出账号序号，如需退出请回复'q'：\n --------------------------------",
						}
						for i, ck := range cks {
							msgs = append(msgs, fmt.Sprintf("%d、%s (优先级: %d)", i, ck.Nickname, ck.Priority))
						}

						sender.Reply(strings.Join(msgs, "\n"))

						var fromIndex, toIndex int
						var firstSelection bool = true

						for {
							select {
							case response := <-msg:
								if response == "q" {
									sender.Reply("已退出转移流程。")
									return
								}

								if firstSelection {
									// 处理第一次选择
									fromIndex, err = strconv.Atoi(response)
									if err != nil || fromIndex < 0 || fromIndex >= len(cks) {
										sender.Reply("输入的序列号无效，已退出，请重新执行命令。")
										return
									}

									// 检查源账号是否有足够的优先级
									fromCk := cks[fromIndex]
									if fromCk.Priority <= int(userPriority) {
										sender.Reply("源账号的优先级不足，请检查后重试。")
										return
									}

									firstSelection = false
									sender.Reply("请输入优先级要转入账号序号：")
								} else {
									// 处理第二次选择
									toIndex, err = strconv.Atoi(response)
									if err != nil || toIndex < 0 || toIndex >= len(cks) || toIndex == fromIndex {
										sender.Reply("输入的目标序列号无效，已退出，请重新执行命令。")
										return
									}

									toCk := cks[toIndex]

									// 更新优先级
									fromCk := cks[fromIndex]
									fromCk.Update(Priority, fromCk.Priority-int(userPriority))
									toCk.Update(Priority, toCk.Priority+int(userPriority))
									sender.Reply(fmt.Sprintf("优先级已从账号:%s 转移到账号:%s。", fromCk.Nickname, toCk.Nickname))
									return
								}
							}
						}
					} else {
						sender.Reply("未找到可转移的账号。")
					}
					return
				}
			}()

			return nil
		},
	},

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

	{
		Command: []string{"余额", "积分", "我的积分", "积分查询", "查询积分"},
		Handle: func(sender *Sender) interface{} {
			return fmt.Sprintf("积分:%d", GetCoin(sender.UserID))
		},
	},

	{
		Command: []string{"推一推状态"},
		Admin:   true,
		Handle: func(sender *Sender) interface{} {
			return fmt.Sprintf("推一推正在运行线程:%d,闲置线程:%d", tytnum, 3-tytnum)
		},
	},

	{
		Command: []string{"绑定微信", "微信绑定"},
		Handle: func(sender *Sender) interface{} {
			if sender.Type == "wx" || sender.Type == "wxg" || sender.Type == "qqg" {
				sender.Reply("请对QQ机器人发送指令，得到绑定码，把绑定码发给微信机器人，打通QQ、微信 ，app客户端，三端查询和积分打卡系统")
				return nil
			} else {
				sender.Reply("请复制发送给Wx机器人完成绑定，微信机器人号：Shi0718-c ，添加后可回复 拉群 入群加入微信群聊，绑定后打通QQ、微信 ，app客户端，三端查询和积分打卡系统")
				sender.Reply("友情提醒：绑定后数据将以QQ机器人为准，如果原本微信用户还有积分，请先使用微信机器人【转账】功能，将微信积分转移到QQ上面，否则绑定后微信积分丢失")
			}
			return makeWxId(sender.UserID, "DXWX"+getMd5String1(strconv.Itoa(sender.UserID)))
		},
	},
{
	Command: []string{"账号注册", "注册账号", "网页注册","帐号注册","注册帐号"},
	Handle: func(sender *Sender) interface{} {
		// 群聊判断：和你短信登录逻辑保持一致
		if sender.Type == "qqg" || sender.Type == "wxg" {
			sender.Reply("为了账号安全，请私聊本机器人进行注册操作")
			return nil
		}

		// 检查用户是否已经有网页账号
		var accountCount int64
		db.Model(&WebUserAccount{}).Where("user_number = ?", sender.UserID).Count(&accountCount)
		if accountCount > 0 {
			return "您已经注册过账号，无需重复注册。如需找回密码，请发送【忘记密码】"
		}

		// 如果用户不存在，自动创建用户记录（类似打卡时的逻辑）
		var u User
		ntime := time.Now()
		err := db.Where("number = ?", sender.UserID).First(&u).Error
		if err != nil {
			// 用户不存在，自动创建
			u = User{
				Class:              sender.Type,
				Number:             sender.UserID,
				Coin:               0,
				ActiveAt:           ntime,
				LastSignIn:         ntime,
				ContinuousSignIns:  0,
				SignInDate:         ntime,
			}
			// 根据不同渠道类型设置对应字段
			switch sender.Type {
			case "wx", "wxg":
				u.Wxid = sender.WxId
			case "qq", "qqg":
				u.QQ = fmt.Sprintf("%d", sender.UserID)
			case "tg":
				u.Telegram = fmt.Sprintf("%d", sender.UserID)
			}
			if err := db.Create(&u).Error; err != nil {
				return "自动创建用户记录失败：" + err.Error()
			}
		}

		bind, err := CreateRegisterBindCode(sender.UserID)
		if err != nil {
			return "生成注册绑定ID失败：" + err.Error()
		}
		return fmt.Sprintf("你的APP/网页注册绑定ID如下：\n%s\n\n有效期：30分钟，仅可使用一次。\n请打开http://180.152.5.230:5701网页或者app注册时填写这个绑定ID完成绑定。\n打不开复制到浏览器打开", bind.Code)
	},
},
{
	Command: []string{"忘记密码", "重置密码", "找回密码"},
	Handle: func(sender *Sender) interface{} {
		// 群聊判断：和你短信登录逻辑保持一致
		if sender.Type == "qqg" || sender.Type == "wxg" {
			sender.Reply("为了账号安全，请私聊本机器人进行密码重置")
			return nil
		}

		resetCode, err := CreatePasswordResetCode(sender.UserID)
		if err != nil {
			return "生成重置验证码失败：" + err.Error()
		}
		return fmt.Sprintf("你的APP/网页重置验证码如下：\n%s\n\n有效期：30分钟，仅可使用一次。\n请打开APP/网页登录页，进入【忘记密码】界面后输入该验证码完成密码重置。", resetCode.Code)
	},
},

	{
		Command: []string{"授权"},
		Handle: func(sender *Sender) interface{} {
			value3 := GetEnv("sqelm")
			if value3 == "" {
				sender.Reply("管理员未开启授权添加功能sqelm")
			} else {
				coin := GetCoin(sender.UserID)
				jbcoin, _ := strconv.Atoi(value3)
				if coin < jbcoin {
					sender.Reply(fmt.Sprintf("积分不足，添加授权需要%d个积分,请登录京东账号获取奖励（或群主积分卡）", jbcoin))
				} else {
					RemCoin(sender.UserID, jbcoin)
					sender.Reply(fmt.Sprintf("添加授权，已扣除%d个积分，剩余积分%d", jbcoin, GetCoin(sender.UserID)))
					ctt := sender.JoinContens()
					auth := AddAuth(ctt)
					if auth {
						return "授权成功"
					} else {
						return "授权失败"
					}
				}
			}
			return nil
		},
	},

	
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
		Command: []string{"更新账号"},
		Admin:   true,
		Handle: func(sender *Sender) interface{} {
			sender.Reply("更新所有账号")
			logs.Info("更新所有账号")
			updateCookie()
			return nil
		},
	},

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

	{
		Command: []string{"导出挖宝账号", "导出挖宝帐号"},
		Admin:   true,
		Handle: func(sender *Sender) interface{} {
			Exportck("jd_fcwb_help")
			Exportck("jd_farmshare") // 传递文件名参数
			return "操作成功"
		},
	},

	{
		Command: []string{"导出追补记账号", "导出追补记帐号"},
		Admin:   true,
		Handle: func(sender *Sender) interface{} {
			Exportck("jd_joyzbj_help")
			return "操作成功"
		},
	},
	{
		Command: []string{"导出1亿账号", "导出瓜分京豆账号"},
		Admin:   true,
		Handle: func(sender *Sender) interface{} {
			Exportck_huanjing("jd_zlyhl")
			return "操作成功"
		},
	},

	{
		Command: []string{"导出赚赚账号", "导出农场账号"},
		Admin:   false,
		Handle: func(sender *Sender) interface{} {
			Exportck("jd_zzhb_new_help")
			Exportck("jd_farmnew_code_help")

			return "操作成功"
		},
	},

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
	{
		Command: []string{"查Q", "CQ"},
		Admin:   true,
		Handle: func(sender *Sender) interface{} {
			str := ""
			sender.Contents = sender.Contents[0:]
			sender.handleJdCookies(func(ck *JdCookie) {
				str = str + fmt.Sprintf("账号：%s (%s) QQ：%d 优先级：%d \n", ck.Nickname, ck.PtPin, ck.QQ, ck.Priority)
			})
			return str
		},
	},

	
{
    Command: []string{"ip统计"},
    Admin:   true,
    Handle: func(sender *Sender) interface{} {
        // 定义统计结构体，存储每个IP的账号总数和有效数
        type IpStat struct {
            total     int // 账号总数
            available int // 有效账号数
        }

        statMap := make(map[string]*IpStat) // key: Socks5_Ip（去空格后）, value: 统计数据
        var totalCookiesTraversed int       // 调试用：记录实际遍历的总账号数
        var unknownIpKey = "未绑定IP"          // 统一未绑定IP的key

        // 核心修复：使用全量Cookie获取方法
        allCookies := GetJdCookies(func(sb *gorm.DB) *gorm.DB {
            return sb // 获取全部Cookie
        })

        // 遍历所有全量Cookie，仅统计password非空的账号
        for _, ck := range allCookies {
            // 仅处理password字段非空的账号
            if strings.TrimSpace(ck.Password) == "" {
                continue // password为空则跳过该账号
            }

            totalCookiesTraversed++ // 累计有效遍历数（仅统计password非空的）

            // 处理IP：去空格+空IP归类
            ip := strings.TrimSpace(ck.Socks5_Ip)
            if ip == "" {
                ip = unknownIpKey
            }

            // 初始化当前IP的统计数据
            if _, exists := statMap[ip]; !exists {
                statMap[ip] = &IpStat{total: 0, available: 0}
            }

            statMap[ip].total++ // 累计该IP的账号总数

            // 有效账号判断
            validValues := map[string]bool{"true": true}
            availableVal := strings.TrimSpace(ck.Available)
            if availableVal != "" && validValues[availableVal] {
                statMap[ip].available++
            }
        }

        // 拼接统计结果（包含调试信息）
        var result strings.Builder
        result.WriteString(fmt.Sprintf("📊 IP统计结果（密码登录用户总数：%d）：\n", totalCookiesTraversed))

        if len(statMap) == 0 {
            result.WriteString("暂无符合条件的账号数据（未查询到password非空的京东Cookie）\n")
        } else {
            // 拆分IP列表：已绑定IP和未绑定IP，确保未绑定IP最后显示
            var boundIps []string       // 已绑定IP列表
            var unknownIpStat *IpStat   // 未绑定IP的统计数据
            var hasUnknownIp bool       // 是否存在未绑定IP的账号

            for ip, stat := range statMap {
                if ip == unknownIpKey {
                    hasUnknownIp = true
                    unknownIpStat = stat
                } else {
                    boundIps = append(boundIps, ip)
                }
            }

            // 输出已绑定IP的统计结果
            for _, ip := range boundIps {
                stat := statMap[ip]
                result.WriteString(fmt.Sprintf("%s --- 总计：%d  有效：%d\n", 
                    ip, stat.total, stat.available))
            }

            // 最后输出未绑定IP的统计结果（单独显示，不计入总IP数）
            if hasUnknownIp {
                result.WriteString(fmt.Sprintf("%s --- 总计：%d  有效：%d\n", 
                    unknownIpKey, unknownIpStat.total, unknownIpStat.available))
            }

            // 计算汇总数据（核心调整：总IP数 = 已绑定IP数量，未绑定IP不计入）
            totalIP := len(boundIps) // 总IP数 = 已绑定IP列表长度（未绑定IP不计入）
            totalAccounts := 0
            totalAvailable := 0
            // 计算所有账号的汇总（包含未绑定IP的账号）
            for _, stat := range statMap {
                totalAccounts += stat.total
                totalAvailable += stat.available
            }

            // 拼接汇总信息（总IP数仅统计已绑定IP）
            result.WriteString(fmt.Sprintf("\n📈 【汇总信息】\n"))
            result.WriteString(fmt.Sprintf("总IP数（已绑定）：%d\n", totalIP)) // 明确标注"已绑定"
            result.WriteString(fmt.Sprintf("总账号数：%d\n", totalAccounts))
            result.WriteString(fmt.Sprintf("总有效账号数：%d\n", totalAvailable))
            if totalAccounts > 0 {
                availabilityRate := float64(totalAvailable) / float64(totalAccounts) * 100
                result.WriteString(fmt.Sprintf("账号有效率：%.2f%%", availabilityRate))
            } else {
                result.WriteString("账号有效率：0.00%")
            }
        }

        return result.String()
    },
},
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
				rt := fmt.Sprintf("你的账号【%s】已过期，快发送【密码登录】提交账号把，全新的密码登录服务，体验直接拉满，快到你无法想象。", ck.Nickname)
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

	{
		Command: []string{"我的2资产", "query"},
		Handle: func(sender *Sender) interface{} {
			sender.Reply("正在为您查询，请耐心等待,如报错可使用 查询 口令确认ck是否失效")
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
	Command: []string{"查询", "我的资产"},
	Handle: func(sender *Sender) interface{} {
		sender.Reply("正在为您查询，请耐心等待，回复 手机卡 指令可办理超值流量卡，回复 密码登录，ck不掉线")

		// 获取用户ID和类型
		id := sender.UserID
		var idType string
		if sender.Type == "tg" {
			idType = Telegram
		} else {
			idType = QQ
		}

		// 获取用户的京东账号列表
		cks := GetJdCookies(func(sb *gorm.DB) *gorm.DB {
			return sb.Where(fmt.Sprintf("%s = ?", idType), id)
		})

		if len(cks) > 0 {
			msgChan := make(chan string)
			ckList[sender.UserID] = msgChan

			// 构建账号选择列表
			var msgs []string
			msgs = append(msgs, "请回复下面【】里面序号进行查询，0为查询全部：\n--------------------------------")
			msgs = append(msgs, "【0】全部")

			// 统计有效账号数量
			var validCount int
			for i, ck := range cks {
				// #调用独立的状态展示函数（核心修改点）
				statusText, isValid := GetAccountStatusText(&ck)
				if isValid {
					validCount++
				}
				// 拼接账号条目
				msgs = append(msgs, fmt.Sprintf("【%d】%s %s", i+1, ck.Nickname, statusText))
			}

			// 添加有效账号统计
			msgs = append(msgs, fmt.Sprintf("--------------------------------\n有效账号: %d/%d", validCount, len(cks)))

			// 拼接并发送列表消息
			accountListMsg := ""
			for _, msg := range msgs {
				accountListMsg += msg + "\n"
			}
			sender.Reply(accountListMsg)

			// 异步处理用户输入
			go func() {
				defer delete(ckList, sender.UserID) // 退出时清理通道

				select {
				case input := <-msgChan:
					// 处理退出指令
					if input == "Q" || input == "q" {
						sender.Reply("查询已退出。")
						return
					}

					// 处理查询指令
					switch input {
					case "0":
						for _, ck := range cks {
							sender.Reply(ck.Query())
						}
					default:
						index, err := strconv.Atoi(input)
						if err != nil || index < 1 || index > len(cks) {
							sender.Reply("无效的输入，已退出程序。")
						} else {
							selectedCk := cks[index-1]
							sender.Reply(selectedCk.Query())
						}
					}
				case <-time.After(30 * time.Second):
					sender.Reply("查询超时，程序自动退出。")
				}
			}()
		} else {
			sender.Reply("没有找到您的有效账号，请直接发送【密码登录】上车。")
		}

		return nil
	 },
	},

	{
		Command: []string{"大师查询"},

		Handle: func(sender *Sender) interface{} {
			sender.Reply("正在为您查询，请耐心等待，回复 手机卡 指令可办理超值流量卡，回复 密码登录，ck不掉线")

			sender.handleJdCookies(func(ck *JdCookie) {
				sender.Reply(ck.Query())
			})

			return nil
		},
	},

	{
		Command: []string{"京豆明细", "资产明细"},

		Handle: func(sender *Sender) interface{} {
			sender.Reply("正在为您查询，请耐心等待，回复 手机卡 指令可办理超值流量卡，回复 密码登录，ck不掉线")

			sender.handleJdCookies(func(ck *JdCookie) {
				sender.Reply(ck.Query3())
			})

			return nil
		},
	},

	

	{
		Command: []string{"发送", "通知", "notify", "send"},
		Admin:   true,
		Handle: func(sender *Sender) interface{} {
			// 创建一个新的通道，用于存储用户输入
			msg := make(chan string)
			ckList[sender.UserID] = msg
			// 启动一个 goroutine 来处理引导式输入
			go func() {
				defer delete(ckList, sender.UserID)

				// 步骤 1: 提示用户输入京东账号（多个账号用换行分隔）
				sender.Reply("请输入京东账号的 PtPin 或者昵称（多个账号用换行分隔）：")
				// 等待用户输入多个账号（换行分隔）
				input := <-msg
				accounts := strings.Split(input, "\n")

				// 去掉每个账号的多余空格
				for i := range accounts {
					accounts[i] = strings.TrimSpace(accounts[i])
				}

				// 步骤 2: 批量查询所有账号对应的数据库记录
				var cks []JdCookie
				err := db.Where("PtPin IN (?) OR Nickname IN (?)", accounts, accounts).Find(&cks).Error
				if err != nil {
					sender.Reply("数据库查询出错，请稍后再试。")
					return
				}

				// 查找没有匹配的账号
				foundAccounts := make(map[string]bool)
				for _, ck := range cks {
					foundAccounts[ck.PtPin] = true
					foundAccounts[ck.Nickname] = true
				}

				var notFound []string
				for _, account := range accounts {
					if !foundAccounts[account] {
						notFound = append(notFound, account)
					}
				}

				// 统计信息
				totalAccounts := len(accounts) // 需要通知的总账号数
				foundAccountsList := len(cks)  // 找到的账号数量
				notFoundCount := len(notFound) // 未找到的账号数量

				// 步骤 3: 提示哪些账号没有找到
				if len(notFound) > 0 {
					sender.Reply(fmt.Sprintf("以下账号未找到：\n%s", strings.Join(notFound, "\n")))
				}

				// 如果所有账号都没有找到，结束处理
				if len(foundAccounts) == 0 {
					sender.Reply("未找到任何匹配的京东账号。")
					return
				}

				// 步骤 4: 提示用户输入通知内容
				sender.Reply("请输入要发送的通知内容：")
				// 等待用户输入通知内容
				notifyMessage := <-msg

				// 步骤 5: 为每个账号生成单独的通知内容并发送
				for _, ck := range cks {
					// 生成带有签名的通知内容
					signedMessage := fmt.Sprintf(
						"【通知来自：%s】\n通知内容：\n%s",
						ck.PtPin, // 使用 PtPin 或者 Nickname 作为账号标识
						notifyMessage,
					)
					// 发送通知给当前账号
					ck.Push(signedMessage)
				}

				// 步骤 6: 生成统计报告并回复
				report := fmt.Sprintf(
					"本次操作统计：\n"+
						"需通知 %d 个账号。\n"+
						"共找到 %d 个账号。\n"+
						"未找到 %d 个账号：\n%s",
					totalAccounts, foundAccountsList, notFoundCount, strings.Join(notFound, "\n"),
				)

				sender.Reply(report)
			}()
			return nil // 返回 nil 表示没有需要返回的值
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
		Command: []string{"QQ转账"},
		Admin:   true,
		Handle: func(sender *Sender) interface{} {
			qq := Int(sender.Contents[0])
			logs.Info(qq)
			if len(sender.Contents) > 1 {
				logs.Info(sender.Contents[1:])
				AdddCoin(qq, Int(sender.Contents[1]))
				sender.Reply(fmt.Sprintf("%d已增加%d枚积分。", qq, Int(sender.Contents[1])))
			}
			return nil
		},
	},

	{
		Command: []string{"转账"},
		Admin:   false,
		Handle: func(sender *Sender) interface{} {
			// 获取发送者的QQ号
			qq := sender.UserID
			logs.Info(qq)

			// 引导用户输入对方QQ号
			sender.Reply("请输入对方的userid，让对方通过机器人发送【用户信息】获取（输入 'q' 退出流程）：")

			// 创建一个消息通道，等待用户输入
			msgChannel := make(chan string)
			ckList[qq] = msgChannel

			go func() {
				defer delete(ckList, qq) // 确保完成后删除该QQ的消息通道

				// 等待用户输入对方QQ号
				toQQStr := <-msgChannel

				// 检查是否退出
				if toQQStr == "q" {
					sender.Reply("已退出转账流程。")
					return
				}

				// 验证QQ号
				toQQ, err := strconv.Atoi(toQQStr)
				if err != nil || toQQ <= 0 {
					sender.Reply("无效的userid，已退出转账")
					return
				}

				// 查询数据库中是否存在该userid
				var user User
				result := db.Where("number = ?", toQQ).First(&user) // 假设User表中有number字段存储userid
				if result.Error != nil {
					// 如果没有找到用户
					sender.Reply(fmt.Sprintf("数据库中未找到你输入的userid：%d 已退出转账，请确认是否输入正确。", toQQ))
					return
				}

				// 引导用户输入积分数量
				sender.Reply(fmt.Sprintf("请输入转账的积分数量，输入 'q' 退出流程 （当前你的账号积分：%d）：", GetCoin(sender.UserID)))

				// 等待用户输入积分数量
				coinStr := <-msgChannel

				// 检查是否退出
				if coinStr == "q" {
					sender.Reply("已退出转账流程。")
					return
				}

				// 验证积分数量
				coin, err := strconv.Atoi(coinStr)
				if err != nil || coin <= 0 {
					sender.Reply("无效的积分数量，已退出转账")
					return
				}

				// 检查积分
				senderCoins := GetCoin(qq)
				if senderCoins < coin {
					sender.Reply("积分不足，无法完成转账。")
					return
				}

				// 执行转账操作
				RemCoin(qq, coin)
				receivedCoins := int(float64(coin) * 1) // 计算接收者实际得到的积分
				AdddCoin(toQQ, receivedCoins)

				senderCoinsAfter := GetCoin(qq)
				updatedReceivedCoins := GetCoin(toQQ)

				// 回复消息给发送者
				sender.Reply(fmt.Sprintf("你已向%d转账%d枚积分，剩余积分：%d；扣除手续费后对方获得：%d积分，对方积分余额为：%d", toQQ, coin, senderCoinsAfter, receivedCoins, updatedReceivedCoins))
			}()

			return nil
		},
	},

{
	Command: []string{"猜拳"},
		Handle: func(sender *Sender) interface{} {
			if !Config.Game.GameOpen || !Config.Game.RockPaperScissorsOpen {
				sender.Reply("管理员已关闭游戏")
				return nil
			}
			u := &User{}

			sender.Reply(fmt.Sprintf("请回复你的出拳选择：剪刀、石头或布（每次游戏固定%d积分）", Config.Game.RockPaperScissors))

		msgChan := make(chan string)
		ckList[sender.UserID] = msgChan
		go func() {
			defer close(msgChan)
			userChoice := <-msgChan

			// 输入有效性检查
			validChoices := map[string]bool{"剪刀": true, "石头": true, "布": true}
			if !validChoices[userChoice] {
				sender.Reply("无效的选择...已退出")
				delete(ckList, sender.UserID)
				return
			}

			// 积分检查
			if err := db.Where("number = ?", sender.UserID).First(u).Error; err != nil || u.Coin < Config.Game.RockPaperScissors {
				sender.Reply("积分不足...已退出")
				delete(ckList, sender.UserID)
				return
			}

			// **合并后的电脑出拳逻辑（原getComputerChoice函数）**
			computerChoice := func(uc string) int {
				r := time.Now().Nanosecond() % 10 // 10种可能性
				switch uc {
				case "剪刀":
					if r < 6 { return 1 } // 60%出石头（克制）
					if r < 9 { return 2 } // 30%出布
					return 0             // 10%出剪刀
				case "石头":
					if r < 6 { return 2 } // 60%出布（克制）
					if r < 9 { return 0 } // 30%出剪刀
					return 1             // 10%出石头
				case "布":
					if r < 6 { return 0 } // 60%出剪刀（克制）
					if r < 9 { return 1 } // 30%出石头
					return 2             // 10%出布
				default:
					return time.Now().Nanosecond() % 3
				}
			}(userChoice)

			choiceMap := []string{"剪刀", "石头", "布"}
			computerChoiceStr := choiceMap[computerChoice]
			
			sender.Reply(fmt.Sprintf("你出了：%s ✊，电脑出了：%s 🤖", userChoice, computerChoiceStr))
			time.Sleep(time.Second)
			
			// 判断结果...（同原代码）
			var resultMsg string
			var result int
			if userChoice == computerChoiceStr {
				result = 0
			} else if (userChoice == "剪刀" && computerChoiceStr == "布") || 
				(userChoice == "石头" && computerChoiceStr == "剪刀") || 
				(userChoice == "布" && computerChoiceStr == "石头") {
				result = 1
			} else {
				result = -1
			}
			
			// 积分处理（优化回复消息）
			var pointChange int
			if result == 1 {
				pointChange = Config.Game.RockPaperScissors
				resultMsg = fmt.Sprintf("恭喜你赢了！获得%d积分，当前积分余额", pointChange)
			} else if result == -1 {
				pointChange = -Config.Game.RockPaperScissors
				resultMsg = fmt.Sprintf("很遗憾你输了，扣除%d积分，当前积分余额", -pointChange)
			} else {
				resultMsg = "平局！积分不变，当前积分余额"
			}
			
			// 更新数据库积分
			db.Model(u).Update("coin", gorm.Expr(fmt.Sprintf("coin + %d", pointChange)))
			
			// 查询最新积分
			if err := db.Where("number = ?", sender.UserID).First(u).Error; err != nil {
				sender.Reply("查询积分失败...")
				delete(ckList, sender.UserID)
				return
			}
			
			// 发送最终结果（确保消息完整）
			sender.Reply(resultMsg + fmt.Sprintf("%d", u.Coin))
			delete(ckList, sender.UserID)
		}()

		return nil
	},
},

{
	Command: []string{"踩雷", "拼了"},
	Handle: func(sender *Sender) interface{} {
		u := &User{}

		// 提示用户输入积分
		currentCoin := GetCoin(sender.UserID)
		sender.Reply(fmt.Sprintf("请回复您想要使用的积分数量，理性踩雷，别上头 （当前积分：%d）：", currentCoin))

		msgChan := make(chan string)
		ckList[sender.UserID] = msgChan

		go func() {
			defer close(msgChan) // 确保在函数结束时关闭通道
			costStr := <-msgChan // 等待用户输入的积分数量

			// 输入有效性检查
			cost, err := strconv.Atoi(costStr)
			if err != nil {
				sender.Reply("无效的输入，请输入一个正整数。")
				delete(ckList, sender.UserID)
				return
			}

			if cost < 0 {
				sender.Reply("不允许输入负数。")
				delete(ckList, sender.UserID)
				return
			}

			if cost <= 0 || cost > 100000000000000 {
				sender.Reply("仿佛发生了点什么")
				delete(ckList, sender.UserID)
				return
			}

			// 检查用户积分
			if err := db.Where("number = ?", sender.UserID).First(u).Error; err != nil || u.Coin < cost {
				sender.Reply("哎呀积分不够了，快去搞点积分吧？=> 请直接微信机器人转账，1元=100积分，转账成功即可完成积分充值，或者复制网址http://180.152.5.230:8005/到其他浏览器打开购买卡密充值。")
				delete(ckList, sender.UserID)
				return
			}

			// 临时扣除积分（显示用）
			newBalance := GetCoin(sender.UserID) - cost
			sender.Reply(fmt.Sprintf("你使用%d枚积分，使用后积分余额%d。", cost, newBalance))

			// 生成随机数
			r := time.Now().Nanosecond() % 10
			
			// 新增逻辑：当投入积分大于1500时必输
			if cost > 1500 {
				r = -1 // 标记为必输情况
			}

			var resultMsg string
			var newCost int

			// 修改后的随机结果判断逻辑
			if r == 9 {
				// 10%概率2倍暴击
				newCost = cost * 2
				resultMsg = fmt.Sprintf("恭喜你2倍暴击！很幸运获得%d枚积分，已为你到账，当前积分余额", newCost)
			} else if r == 3 || r == 5 {
				// r=7或r=5时赢取积分
				newCost = cost
				resultMsg = fmt.Sprintf("很幸运你获得%d枚积分，已为你到账，当前积分余额", newCost)
			} else {
				// 其他情况失去积分（包括r为-1的必输情况）
				newCost = -cost
				resultMsg = fmt.Sprintf("很遗憾你失去了%d枚积分，当前积分余额", cost)
			}
			
			time.Sleep(time.Second * 2)
			// **关键：先更新数据库**
			db.Model(u).Update("coin", gorm.Expr(fmt.Sprintf("coin + %d", newCost)))
			

			// **再查询最新积分余额**
			if err := db.Where("number = ?", sender.UserID).First(u).Error; err != nil {
				sender.Reply("查询积分失败，请稍后再试")
				delete(ckList, sender.UserID)
				return
			}
			
			// 补全结果消息（包含数据库查询的真实余额）
			sender.Reply(resultMsg + fmt.Sprintf("%d。", u.Coin))

			// 清除用户记录
			delete(ckList, sender.UserID)
		}()

		return nil
	},
	},

	
	{
		Command: []string{"许愿", "愿望", "wish", "hope", "want"},
		Handle: func(sender *Sender) interface{} {
			ct := sender.JoinContens()
			if ct == "" {
				rt := []string{}
				ws := []Wish{}
				tb := db
				if !sender.IsAdmin {
					tb = tb.Where("user_number", sender.UserID)
				} else {
					tb = tb.Where("status != 1")
				}
				tb.Order("id asc").Find(&ws)
				if len(ws) == 0 {
					return "请对我说 许愿 巴拉巴拉"
				}
				for i, w := range ws {
					status := "未达成"
					if w.Status == 1 {
						status = "已撤销"
					} else if w.Status == 2 {
						status = "已达成"
					}
					id := i + 1
					if sender.IsAdmin {
						id = w.ID
					}
					rt = append(rt, fmt.Sprintf("%d. %s [%s]", id, w.Content, status))
				}
				return strings.Join(rt, "\n")
			}
			cost := 88
			if sender.IsAdmin {
				cost = 1
			}
			tx := db.Begin()
			u := &User{}
			if err := tx.Where("number = ?", sender.UserID).First(u).Error; err != nil {
				tx.Rollback()
				return "积分不足，先去打卡吧。"
			}
			w := &Wish{
				Content:    ct,
				Coin:       cost,
				UserNumber: sender.UserID,
			}
			if u.Coin < cost {
				tx.Rollback()
				return fmt.Sprintf("积分不足，需要%d个积分。", cost)
			}
			if err := tx.Create(w).Error; err != nil {
				tx.Rollback()
				return err.Error()
			}
			if tx.Model(u).Update("coin", gorm.Expr(fmt.Sprintf("coin - %d", cost))).RowsAffected == 0 {
				tx.Rollback()
				return "扣款失败"
			}
			tx.Commit()
			(&JdCookie{}).Push(fmt.Sprintf("有人许愿%s，愿望id为%d。", w.Content, w.ID))
			return fmt.Sprintf("收到愿望，已扣除%d个积分。", cost)
		},
	},
	{
		Command: []string{"愿望达成", "达成愿望"},
		Admin:   true,
		Handle: func(sender *Sender) interface{} {
			w := &Wish{}
			id := Int(sender.JoinContens())
			if id == 0 {
				return "目标未指定"
			}
			if db.First(w, id).Error != nil {
				return "目标不存在"
			}
			if w.Status == 1 {
				return "愿望已撤销"
			}
			if w.Status == 2 {
				return "愿望已达成"
			}
			if db.Model(w).Update("status", 2).RowsAffected == 0 {
				return "操作失败"
			}
			sender.Reply(fmt.Sprintf("达成了愿望 %s", w.Content))
			return nil
		},
	},
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
			// 创建一个新的通道，用于存储用户输入
			msg := make(chan string)
			ckList[sender.UserID] = msg

			// 启动一个 goroutine 来处理引导式输入
			go func() {
				defer delete(ckList, sender.UserID)

				// 步骤 1: 提示用户输入京东账号
				sender.Reply("请输入京东账号的 PtPin 或者昵称：")

				// 等待用户输入京东账号
				ptPinOrNickname := <-msg

				// 步骤 2: 检查是否存在该账号
				var found bool
				var foundCookie *JdCookie

				// 获取所有京东账号
				allCookies := GetJdCookies(func(sb *gorm.DB) *gorm.DB {
					return sb // 假设这里返回了所有数据
				})

				// 遍历所有账号进行匹配
				for _, ck := range allCookies {
					if ck.PtPin == ptPinOrNickname || ck.Nickname == ptPinOrNickname {
						foundCookie = &ck
						found = true
						break
					}
				}

				// 如果找到了该账号，提示用户输入优先级
				if found {
					sender.Reply(fmt.Sprintf("你输入的京东账号是：%s，请输入该账号的优先级（一个正整数）：", ptPinOrNickname))

					// 循环直到输入有效的优先级
					for {
						priorityStr := <-msg

						// 检查优先级输入是否是正整数
						priority, err := strconv.Atoi(priorityStr)
						if err != nil || (priority != -1 && priority != 0 && priority <= 0) {
							// 如果输入无效，提示重新输入
							sender.Reply("优先级必须是 -1、0 或正整数，请重新输入。")
						} else {
							// 输入有效，更新优先级
							foundCookie.Update(Priority, priority)
							sender.Reply(fmt.Sprintf("已设置账号%s(%s)的优先级为%d。", foundCookie.PtPin, foundCookie.Nickname, priority))
							break // 跳出循环
						}
					}
				} else {
					// 如果没有找到对应的账号，立即回复提示
					sender.Reply("系统没有找到对应的用户名，请检查输入是否正确。")
				}
			}()

			return nil
		},
	},

	{
		Command: []string{"京粉查询"},
		Handle: func(sender *Sender) interface{} {
			ck, err := GetJdCookie("jd_404a0a053ec13")
			if err != nil {
				sender.Reply("ck获取失败")
				return "ck获取失败"
			}
			ptKey := ck.PtKey
			ptPin := ck.PtPin

			QueryJingFen(sender, ptKey, ptPin)
			return nil
		},
	},

	
	{
		Command: []string{"cmd", "command"},
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
		Command: []string{"管理后台"},
		Admin:   true,
		Handle: func(sender *Sender) interface{} {
			u := uuid.New()
			s := u.String()
			SaveCache("AdminToken", s)
			return "你的临时授权码为：" + s
		},
	},
	{
		Command: []string{"环境变量", "environments", "envs"},
		Admin:   true,
		Handle: func(_ *Sender) interface{} {
			rt := []string{}
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
	{
		Command: []string{"重置推一推"},
		Admin:   true,
		Handle: func(sender *Sender) interface{} {
			cks := GetJdCookies()
			for _, ck := range cks {
				ck.Update(Tyt, True)
			}
			return nil
		},
	},
	{
		Command: []string{"重置活动"},
		Admin:   true,
		Handle: func(sender *Sender) interface{} {
			cks := GetJdCookies()
			for _, ck := range cks {
				ck.Update(Dig, True)
			}
			return nil
		},
	},
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
	{
		Command: []string{"降级"},
		Handle: func(sender *Sender) interface{} {
			return "滚"
		},
	},
	{
		Command: []string{"。。。"},
		Handle: func(sender *Sender) interface{} {
			return "你很无语吗？"
		},
	},

	{
		Command: []string{"祈祷", "祈愿", "祈福"},
		Handle: func(sender *Sender) interface{} {
			sender.Reply("🔄 机器人端祈福功能升级中\n\n📱 请前往以下渠道完成祈福：\n• APP 用户后台 → 积分任务 → 祈福\n• 网页用户后台 → 积分任务 → 祈福\n\n⏳ 升级完成后将第一时间通知大家，感谢理解！")
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
		Command: []string{"更新指定"},
		Admin:   true,
		Handle: func(sender *Sender) interface{} {
			sender.handleJdCookies(func(ck *JdCookie) {
				if len(ck.WsKey) > 0 {
					var pinky = fmt.Sprintf("pin=%s;wskey=%s;", ck.PtPin, ck.WsKey)
					rsp := getKey(pinky)
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
					}
				} else {
					sender.Reply(fmt.Sprintf("Wskey为空，%s", ck.Nickname))
				}

			})
			return nil
		},
	},

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
				sender.Reply(fmt.Sprintf("已删除账号%s", ck.Nickname))
			})
			return nil
		},
	},

	{
		Command: []string{"删除美团", "美团删除"},
		Handle: func(sender *Sender) interface{} {
			meiTuans := GetMeiTuan(sender)
			if len(meiTuans) > 0 {
				// 进入队列
				msg := make(chan string)
				meituanList[sender.UserID] = msg
				go Delete_meituan(sender, msg, meiTuans)
				msgs := []string{
					"请回复以下序列号删除指定账号，如需退出请回复'q'退出登录流程：",
				}
				for i, tuan := range meiTuans {
					msgs = append(msgs, fmt.Sprintf("%d、%s", i, tuan.Nickname))
				}
				sender.Reply(strings.Join(msgs, "\n"))
			} else {
				sender.Reply("查无美团账号，请使用 '美团登录'口令，按要求提交ck")
				return nil
			}
			return nil
		},
	},

	{
    Command: []string{"新闻"},
    Admin:   true,
    Handle: func(sender *Sender) interface{} {
        // 调用上面编写的函数
        HandleNews() 
        return nil
    },
},

	{
		Command: []string{"图片推送"},
		Admin:   true,
		Handle: func(sender *Sender) interface{} {
			WxImg_ts()
			return nil
		},
	},

	{
		Command: []string{"群推送"},
		Admin:   true,
		Handle: func(sender *Sender) interface{} {
			SendWxGroupMsg("1", "56984485809@chatroom", "测试")
			return nil
		},
	},
	{
		Command: []string{"Q群推送"},
		Admin:   true,
		Handle: func(sender *Sender) interface{} {
			SendQQGroup(955812631, 1, "测试")
			return nil
		},
	},

	{
		Command: []string{"新农场推送", "新版农场推送"},
		Admin:   true,
		Handle: func(sender *Sender) interface{} {
			sender.Reply("开始通知韭菜新版农场成熟情况")
			CX_jd_fruit_new_cx()
			sender.Reply("已完成通知")
			return nil
		},
	},

	{
		Command: []string{"话费通知", "话费兑换通知"},
		Admin:   true,
		Handle: func(sender *Sender) interface{} {
			sender.Reply("开始通知满足兑换10元话费的韭菜")
			CX_jd_dwapp_cx()
			sender.Reply("已完成通知")
			return nil
		},
	},

	{
		Command: []string{"保价推送"},
		Admin:   true,
		Handle: func(sender *Sender) interface{} {
			sender.Reply("开始通知韭菜保价情况")
			CX_jd_OnceApply_cx()
			sender.Reply("已完成通知")
			return nil
		},
	},

	{
		Command: []string{"查找内鬼"},
		Admin:   true,
		Handle: func(sender *Sender) interface{} {
			sender.Reply("开始查找内鬼。。。。")

			// 群ID列表，可以根据需要修改
			groupIDs := []string{"955812631", "916246295"} // 假设有三个群

			// 在 goroutine 中异步执行任务，避免阻塞其他指令
			go func() {
				for _, groupID := range groupIDs {
					GetGroupMembers(groupID)
				}
			}()

			return nil
		},
	},

	{
		Command: []string{"扣除500优先级"},
		Admin:   true,
		Handle: func(sender *Sender) interface{} {
			sender.Reply("开始扣除前30名优先级")
			PriorityDel()
			sender.Reply("已完成")
			return nil
		},
	},

	{
		Command: []string{"删除账号", "账号删除", "京东账号删除", "删除京东账号"},
		Handle: func(sender *Sender) interface{} {
			sender.Reply("删除账号，会导致优先级清空，且无法恢复，请谨慎操作。。。。")
			time.Sleep(2 * time.Second)
			id := sender.UserID
			var idType string
			if sender.Type == "tg" {
				idType = Telegram
			} else {
				idType = QQ
			}
			cks := GetJdCookies(func(sb *gorm.DB) *gorm.DB {
				return sb.Where(fmt.Sprintf("%s = ? and %s = ?", idType, Available), id, False)
			})

			if len(cks) > 0 {
				msg := make(chan string)
				ckList[sender.UserID] = msg
				go Delete_jdck(sender, msg, cks)
				msgs := []string{
					"请回复以下序列号删除指定失效账号，如需退出请回复'q'退出登录流程：",
				}
				for i, ck := range cks {
					msgs = append(msgs, fmt.Sprintf("%d、%s", i, ck.Nickname))
				}
				sender.Reply(strings.Join(msgs, "\n"))
			} else {
				sender.Reply("无失效账号，新增账号请对机器人发送“密码登录”")
				return nil
			}
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
		Command: []string{"清空WCK"},
		Admin:   true,
		Handle: func(sender *Sender) interface{} {
			cleanWck()
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
		Command: []string{"Available", "可用"},
		Admin:   true,
		Handle: func(sender *Sender) interface{} {
			sender.handleJdCookies(func(ck *JdCookie) {
				ck.Update(Available, True)
				sender.Reply(fmt.Sprintf("已设置可用账号%s(%s)", ck.PtPin, ck.Nickname))
			})
			return nil
		},
	},
	{
		Command: []string{"不可用", "unAvailable", "取消可用"},
		Admin:   true,
		Handle: func(sender *Sender) interface{} {
			sender.handleJdCookies(func(ck *JdCookie) {
				ck.Update(Available, False)
				sender.Reply(fmt.Sprintf("已设置取消可用账号%s(%s)", ck.PtPin, ck.Nickname))
			})
			return nil
		},
	},

	{
		Command: []string{"读取2"},
		Handle: func(sender *Sender) interface{} {
			var u User
			if db.Where("Account = ?", "18954177124").First(&u).Error != nil {
				sender.Reply("找到了")
			} else {
				sender.Reply("未找到")
			}
			return nil
		},
	},
	{
		Command: []string{"老版转账"},
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
			if sender.UserID == sender.ReplySenderUserID {
				db.Model(User{}).Where("number = ?", sender.UserID).Updates(map[string]interface{}{
					"coin": gorm.Expr(fmt.Sprintf("coin - %d", cost)),
				})
				return fmt.Sprintf("转账成功，扣除手续费%d枚积分。", cost)
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
					return fmt.Sprintf("转账失败，手续费需要%d个积分。", cost)
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
		Command: []string{"关闭查询"},
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
	{
		Command: []string{"开启查询"},
		Admin:   true,
		Handle: func(sender *Sender) interface{} {
			UnExportEnv(&Env{
				Name: "qq",
			})
			sender.Reply("操作成功")
			return nil
		},
	},

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

	{
		Command: []string{"开启微信自动收款"},
		Admin:   true,
		Handle: func(sender *Sender) interface{} {
			env := &Env{
				Name:  "Autocollection",
				Value: "1",
			}
			ExportEnv(env)
			sender.Reply("操作成功")
			return nil
		},
	},
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

	//自动加好友 口令验证
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

	{
		Command: []string{"链接转口令"},
		Admin:   true,
		Handle: func(sender *Sender) interface{} {
			klurl := sender.Contents[0]
			result := LJtoKL(klurl)
			sender.Reply(fmt.Sprintf("复制本条信息打开JDapp %s", result))
			return nil
		},
	},
	
{
    Command: []string{"用户信息", "查询id", "查询ID", "我的ID", "我的信息"},
    Admin:   false,
    Handle: func(sender *Sender) interface{} {
     
        msgs := []string{
            "信息如下：",
            "WxId: " + sender.WxId,
            "UserID: " + strconv.Itoa(sender.UserID),
            "ChatID: " + strconv.Itoa(sender.ChatID),
            "GroupId: " + strconv.Itoa(sender.GroupId),
            "WxGroupId: " + sender.WxGroupId,
            "Type: " + sender.Type,
            "Contents: " + strings.Join(sender.Contents, ", "),
            "MessageID: " + strconv.Itoa(sender.MessageID),
            "Username: " + sender.Username,
            "IsAdmin: " + strconv.FormatBool(sender.IsAdmin),
            "ReplySenderUserID: " + strconv.Itoa(sender.ReplySenderUserID),
        }
        return strings.Join(msgs, "\n")
    },
},
	//获取我的userid
	{
		Command: []string{"我的微信号"},
		Handle: func(sender *Sender) interface{} {
			return sender.WxId
		},
	},

	//获取我的wxGroupId
	{
		Command: []string{"微信群号"},
		Handle: func(sender *Sender) interface{} {
			return sender.WxGroupId
		},
	},

	//设置自动加好友验证后自动发送的消息
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
	{
		Command: []string{"导出wskey"},
		Admin:   false,
		Handle: func(sender *Sender) interface{} {
			if len(sender.Contents) != 0 {
				sender.Reply("发送指令格式错误")
			} else {
				sender.handleJdCookies(func(ck *JdCookie) {
					sender.Reply(fmt.Sprintf("pin=%s;wskey=%s;", ck.PtPin, ck.RWskey))
				})
			}
			return nil
		},
	},

	{
		Command: []string{"回填微信", "微信回填"},
		Admin:   true,
		Handle: func(sender *Sender) interface{} {
			cks := GetJdCookies()
			xx := 0
			successCount := 0
			for i := range cks {
				if cks[i].QQ != 0 {
					WeiXin := getWeiXinId(cks[i].QQ)
					if WeiXin != "找不到对应的微信ID" {
						ck := cks[i]
						ck.Updates(JdCookie{WeiXin: WeiXin})
						successCount++
					}
					xx++
				}
				time.Sleep(500 * time.Millisecond)
			}
			(&JdCookie{}).Push(fmt.Sprintf("回填微信完成，共处理%d个数据，回填成功：%d个", xx, successCount))
			return nil

		},
	},
	{
		Command: []string{"美团登录", "登录美团", "美团扫码", "美团登陆"},
		Handle: func(sender *Sender) interface{} {
			Meituan_getck(sender)
			return nil
		},
	},

	{
		Command: []string{"猜数字", "数字游戏"},
		Handle: func(sender *Sender) interface{} {
			if !Config.Game.GameOpen || !Config.Game.GuessNumberOpen {
				sender.Reply("管理员已关闭游戏")
				return nil
			}
			jbcoin := Config.Game.GuessNumberCost
			coin := GetCoin(sender.UserID)
			maxGuessTimes := Config.Game.GuessNumberTimes

			if coin < jbcoin*maxGuessTimes {
				sender.Reply("积分不够本次游戏，已退出游戏，请直接私聊微信机器人转账，1元=100积分，转账成功即可完成积分充值，或者复制网址http://180.152.5.230:8005/到其他浏览器打开购买卡密充值")
				return nil
			}

			msg := make(chan string)
			ckList[sender.UserID] = msg
			go Guess_Number(sender, msg, maxGuessTimes)

			message := fmt.Sprintf("猜数字规则：游戏次数%d次，每次猜扣%d积分，猜中奖励%d积分，游戏过程中存在5个地雷猜中直接扣%d积分，结束游戏。\n请回复0-100内数字猜测，中途如需退出游戏回复'q'退出流程：",
				maxGuessTimes, jbcoin, Config.Game.GuessNumberReward, Config.Game.GuessNumberMineCost)
			sender.Reply(message)

			// 返回nil表示函数执行完毕
			return nil
		},
	},

{
		Command: []string{"比大小"},
		Handle: func(sender *Sender) interface{} {
			if !Config.Game.GameOpen || !Config.Game.BigSmallOpen {
				sender.Reply("管理员已关闭游戏")
				return nil
			}
			gameMutex.Lock()
			gameCount := len(games)
			gameMutex.Unlock()

			if gameCount >= Config.Game.BigSmallMaxGames {
				sender.Reply(fmt.Sprintf("当前游戏房间已满（最多%d个游戏），请回复【加入】参与现有游戏", Config.Game.BigSmallMaxGames))
				return nil
			}

			u := &User{}
			if err := db.Where("number = ?", sender.UserID).First(u).Error; err != nil || u.Coin < Config.Game.BigSmallCost {
				sender.Reply(fmt.Sprintf("创建游戏需要至少%d积分，你的积分不足", Config.Game.BigSmallCost))
				return nil
			}

			// 生成游戏ID (6位随机数字)
			gameID := fmt.Sprintf("%06d", rand.Intn(1000000))

			// 创建新游戏
			gameMutex.Lock()
			newGame := &Game{
				ID:         gameID,
				Creator:    sender.UserID,
				Platform:   sender.Type,
				Players:    make(map[int]int),
				Status:     "waiting",
				Scores:     make(map[int]int),
				CreateTime: time.Now(), // 记录创建时间
			}
			newGame.Players[sender.UserID] = -1
			games[gameID] = newGame
			
			// 更新按时间排序的游戏列表
			gamesByTime = append(gamesByTime, newGame)
			sort.Slice(gamesByTime, func(i, j int) bool {
				return gamesByTime[i].CreateTime.Before(gamesByTime[j].CreateTime)
			})
			gameMutex.Unlock()

			// 构建平台类型描述
			platformDesc := "微信"
			if sender.Type == "qqg" {
				platformDesc = "QQ"
			}

			sender.Reply(fmt.Sprintf("%d人比大小游戏已创建！游戏ID: %s\n"+
				"创建人: 用户%d\n创建平台: %s\n当前人数: 1/%d，缺少%d人\n"+
				"其他玩家可以回复【加入】参与游戏\n"+
				"每人需要%d积分参与",
				Config.Game.BigSmallPlayers, gameID, sender.UserID, platformDesc,
				Config.Game.BigSmallPlayers, Config.Game.BigSmallPlayers-1,
				Config.Game.BigSmallCost))

			// 处理游戏加入和开始的逻辑
			msgChan := make(chan string)
			ckList[sender.UserID] = msgChan

			go func(gameID string) {
				defer close(msgChan)
				delete(ckList, sender.UserID)
				
				var game *Game
				var gameExists bool
				
				// 定义锁操作辅助函数
				lockAndCheckGame := func() bool {
					gameMutex.Lock()
					game, gameExists = games[gameID]
					return gameExists
				}
				
				unlockGame := func() {
					gameMutex.Unlock()
				}
				
				// 等待玩家加入
				startTime := time.Now()
				for {
					// 检查是否超时(20分钟)
					if time.Since(startTime) > 20*time.Minute {
						if lockAndCheckGame() {
							if game.Status == "waiting" {
								delete(games, gameID)
								sender.Reply(fmt.Sprintf("游戏 %s 因超时未满员已取消", gameID))
							}
						}
						unlockGame()
						return
					}

					// 检查是否人满
					if lockAndCheckGame() {
						if len(game.Players) >= Config.Game.BigSmallPlayers {
							game.Status = "playing"
							unlockGame()
							break
						}
					} else {
						unlockGame()
						return // 游戏已被删除
					}

					unlockGame()
					time.Sleep(1 * time.Second)
				}

				// 人满，开始游戏
				if lockAndCheckGame() {
					// 扣除所有玩家积分
					for playerID := range game.Players {
						db.Model(&User{}).Where("number = ?", playerID).Update("coin", gorm.Expr("coin - 60"))
					}

					// 为每个玩家生成随机数字
					for playerID := range game.Players {
						game.Players[playerID] = rand.Intn(101)
					}

					// 获取玩家数字并排序
					type PlayerScore struct {
						ID    int
						Score int
					}
					var rankings []PlayerScore
					for id, score := range game.Players {
						rankings = append(rankings, PlayerScore{ID: id, Score: score})
					}

					// 按分数降序排序
					sort.Slice(rankings, func(i, j int) bool {
						return rankings[i].Score > rankings[j].Score
					})

					// 分配奖励
					if len(rankings) > 0 {
						game.Scores[rankings[0].ID] = 120
					}
					if len(rankings) > 1 {
						game.Scores[rankings[1].ID] = 90
					}
					if len(rankings) > 2 {
						game.Scores[rankings[2].ID] = 70
					}
					if len(rankings) > 3 {
						game.Scores[rankings[3].ID] = 40
					}
					if len(rankings) > 4 {
						game.Scores[rankings[4].ID] = 20
					}
					if len(rankings) > 5 {
						game.Scores[rankings[5].ID] = 0
					}

					// 更新玩家积分
					for playerID, reward := range game.Scores {
						db.Model(&User{}).Where("number = ?", playerID).Update("coin", gorm.Expr("coin + ?", reward))
					}

					// 生成结果消息
					resultMsg := "🎮 比大小游戏排行榜 🎮\n\n"
					resultMsg += fmt.Sprintf("创建时间：%s\n\n", game.CreateTime.Format("2006-01-02 15:04:05"))
					resultMsg += "【排名信息】\n"
					resultMsg += "-------------------------\n"

					// 定义排名图标映射
					rankIcons := map[int]string{
						1: "🥇", 2: "🥈", 3: "🥉",
						4: "4️⃣", 5: "5️⃣", 6: "6️⃣",
					}

					for i, player := range rankings {
						rank := i + 1
						var reward int
						switch rank {
						case 1: reward = 120
						case 2: reward = 90
						case 3: reward = 70
						case 4: reward = 40
						case 5: reward = 20
						case 6: reward = 0
						}
						icon := rankIcons[rank]
						
						// 纯文本分行格式
						resultMsg += fmt.Sprintf("  %s 第%d名 - 用户%05d | 数字: %-2d | 奖励: +%3d\n", 
							icon, rank, player.ID, player.Score, reward)
					}
					resultMsg += "-----------------------------\n"
					resultMsg += "\n📝 游戏规则：\n"
					resultMsg += fmt.Sprintf("  • 积分变动：参与扣除%d积分，奖励实时到账\n", Config.Game.BigSmallCost)
					resultMsg += "  • 排名逻辑：数字大小决定名次（例：98 > 85 > 76）"

					// 发送结果（示例函数，需根据实际平台实现）
					SendWxGroupMsg("1", "56984485809@chatroom", resultMsg)
					SendQQGroup(955812631, 1, resultMsg)
					SendQQGroup(916246295, 1, resultMsg)
					delete(games, gameID)
				}
				unlockGame()
			}(gameID)

			return nil
		},
	},
	

{
    Command: []string{"加入"},
    Handle: func(sender *Sender) interface{} {
        if !Config.Game.GameOpen || !Config.Game.BigSmallOpen {
            sender.Reply("管理员已关闭游戏")
            return nil
        }
        gameMutex.Lock()
        var waitingGames []*Game
        now := time.Now()
        
        // 先过滤掉过期的游戏
        var validGamesByTime []*Game
        for _, game := range gamesByTime {
            if now.Sub(game.CreateTime) <= 20*time.Minute {
                validGamesByTime = append(validGamesByTime, game)
            } else {
                // 删除过期的游戏
                delete(games, game.ID)
            }
        }
        gamesByTime = validGamesByTime
        
        // 现在只收集未过期且等待中的游戏
        for _, game := range gamesByTime {
            if game.Status == "waiting" {
                waitingGames = append(waitingGames, game)
            }
        }
        gameMutex.Unlock()

        if len(waitingGames) == 0 {
            sender.Reply("当前没有等待中的游戏，你可以发送【比大小】创建游戏")
            return nil
        }

        // 发送优化后的游戏列表
        gameListMsg := "当前等待中的游戏（按创建时间排序）：\n\n"
        for i, game := range waitingGames {
            platformDesc := "微信"
            if game.Platform == "qqg" {
                platformDesc = "QQ"
            }
            
            // 计算剩余时间
            remainingTime := 20*time.Minute - now.Sub(game.CreateTime)
            remainingMinutes := int(remainingTime.Minutes())
            remainingSeconds := int(remainingTime.Seconds()) % 60
            
            // 格式化创建时间
            timeDesc := game.CreateTime.Format("15:04:05")
            
            gameInfo := fmt.Sprintf("【%d】游戏ID: %s\n", i+1, game.ID)
            gameInfo += fmt.Sprintf("  ├─ 创建时间: %s\n", timeDesc)
            gameInfo += fmt.Sprintf("  ├─ 剩余时间: %d分%d秒\n",  remainingMinutes, remainingSeconds)
            gameInfo += fmt.Sprintf("  ├─ 创建平台: %s\n", platformDesc)
            gameInfo += fmt.Sprintf("  ├─ 创建人: 用户%d\n", game.Creator)
            gameInfo += fmt.Sprintf("  └─ 当前人数: %d/6 (还差: %d人)\n", 
                len(game.Players), 6-len(game.Players))
            
            gameListMsg += gameInfo + "\n"
        }
        
        gameListMsg += "请在30秒内输入【】中的数字序号选择游戏\n" +
                       "示例：输入【1】加入第一个游戏"

        sender.Reply(gameListMsg)

        // 创建channel并处理用户输入
        msgChan := make(chan string)
        ckList[sender.UserID] = msgChan

        go func() {
            defer close(msgChan)
            defer delete(ckList, sender.UserID)
            timeout := time.NewTimer(30 * time.Second)
            defer timeout.Stop()

            // 异常处理
            defer func() {
                if r := recover(); r != nil {
                    gameMutex.Lock()
                    for _, game := range games {
                        if game.Status == "waiting" {
                            delete(game.Players, sender.UserID)
                        }
                    }
                    gameMutex.Unlock()
                }
            }()

            select {
            case msg := <-msgChan:
                // 用户输入处理
                selectedGameIndex, err := strconv.Atoi(strings.TrimSpace(msg))
                if err != nil {
                    sender.Reply("输入无效，请输入一个有效的数字序号！")
                    return
                }

                if selectedGameIndex < 1 || selectedGameIndex > len(waitingGames) {
                    sender.Reply(fmt.Sprintf("输入无效，有效序号范围是1到%d！", len(waitingGames)))
                    return
                }

                sliceIndex := selectedGameIndex - 1
                selectedGame := waitingGames[sliceIndex]
                selectedGameID := selectedGame.ID

                gameMutex.Lock()
                game, exists := games[selectedGameID]
                if !exists || game.Status != "waiting" {
                    sender.Reply("该游戏已不存在或状态已改变，无法加入")
                    gameMutex.Unlock()
                    return
                }

                // 再次检查游戏是否过期
                if now.Sub(game.CreateTime) > 20*time.Minute {
                    sender.Reply("该游戏已过期，无法加入")
                    delete(games, selectedGameID)
                    gameMutex.Unlock()
                    return
                }

                if len(game.Players) >= Config.Game.BigSmallPlayers {
                    sender.Reply("该游戏已满员，无法加入")
                    gameMutex.Unlock()
                    return
                }

                if _, alreadyJoined := game.Players[sender.UserID]; alreadyJoined {
                    sender.Reply("你已加入该游戏")
                    gameMutex.Unlock()
                    return
                }

                u := &User{}
                if err := db.Where("number = ?", sender.UserID).First(u).Error; err != nil || u.Coin < Config.Game.BigSmallCost {
                    sender.Reply(fmt.Sprintf("加入游戏需要%d积分，你的积分不足", Config.Game.BigSmallCost))
                    gameMutex.Unlock()
                    return
                }

                // 加入游戏
                game.Players[sender.UserID] = -1
                gameMutex.Unlock()
                sender.Reply(fmt.Sprintf("成功加入游戏 %s！当前玩家 %d/6，等待游戏开始...", selectedGameID, len(game.Players)))

            case <-timeout.C:
                // 超时处理
                sender.Reply("选择超时！30秒内未选择游戏，操作已取消。")
                
                // 清理可能的临时加入状态
                gameMutex.Lock()
                for _, game := range games {
                    if game.Status == "waiting" {
                        delete(game.Players, sender.UserID)
                    }
                }
                gameMutex.Unlock()
            }
        }()
        return nil
    },
},

{
	// ────────────────────────────────────────────────────────────
	//  决斗发起命令（支持自定义下注：决斗 100）
	// ────────────────────────────────────────────────────────────
	Command: []string{"决斗"},
	Handle: func(sender *Sender) interface{} {
		if !Config.Game.GameOpen || !Config.Game.DuelOpen {
			sender.Reply("管理员已关闭游戏")
			return nil
		}
		bet := Config.Game.DuelDefaultBet
		if len(sender.Contents) > 1 {
			if v, err := strconv.Atoi(strings.TrimSpace(sender.Contents[1])); err == nil && v >= Config.Game.DuelDefaultBet && v <= 500 {
				bet = v
			}
		}

		duelMutex.Lock()
		duelCount := len(duels)
		duelMutex.Unlock()
		if duelCount >= Config.Game.DuelMaxRooms {
			sender.Reply(fmt.Sprintf("当前决斗房间已满（最多%d个），请回复【迎战】参与现有决斗", Config.Game.DuelMaxRooms))
			return nil
		}

		u := &User{}
		if err := db.Where("number = ?", sender.UserID).First(u).Error; err != nil || u.Coin < bet {
			sender.Reply(fmt.Sprintf("发起决斗需要至少 %d 积分，你的积分不足", bet))
			return nil
		}

		// ──── Step 1: 让发起者选职业 ────
		sender.Reply(ClassListMsg())

		// Step 1: 仅在选职业阶段注册 ckList，选完后立即注销，不影响其他指令
		msgChan := make(chan string, 3)
		ckList[sender.UserID] = msgChan

		go func() {
			// ── 选职业阶段：用完立即注销，让用户恢复正常消息流 ──
			var chosenClass ClassType
			classTimeout := time.NewTimer(40 * time.Second)
			defer classTimeout.Stop()
			select {
			case msg := <-msgChan:
				idx, err := strconv.Atoi(strings.TrimSpace(msg))
				if err == nil && idx >= 1 && idx <= len(AllClasses) {
					chosenClass = AllClasses[idx-1].Name
				} else {
					chosenClass = AllClasses[rand.Intn(len(AllClasses))].Name
					sender.Reply(fmt.Sprintf("输入无效，已随机分配职业：%s", string(chosenClass)))
				}
			case <-classTimeout.C:
				chosenClass = AllClasses[rand.Intn(len(AllClasses))].Name
				sender.Reply(fmt.Sprintf("选择超时，随机分配职业：%s", string(chosenClass)))
			}

			// ★ 选完职业后立即从 ckList 注销，用户可以正常使用其他指令了 ★
			delete(ckList, sender.UserID)
			for len(msgChan) > 0 {
				<-msgChan
			}
			close(msgChan)

			// 生成发起者属性
			name := sender.Username
			if name == "" {
				name = fmt.Sprintf("用户%d", sender.UserID)
			}
			initiatorAttrs := GenerateBattleAttr(sender.UserID, name, chosenClass)

			// 创建决斗房间
			duelID := fmt.Sprintf("%04d", rand.Intn(10000))
			duelMutex.Lock()
			newDuel := &Duel{
				ID:             duelID,
				Initiator:      sender.UserID,
				InitiatorAttrs: initiatorAttrs,
				InitiatorClass: chosenClass,
				Challenger:     0,
				Status:         "waiting",
				CreateTime:     time.Now(),
				Platform:       sender.Type,
				Bet:            bet,
			}
			duels[duelID] = newDuel
			duelsByTime = append(duelsByTime, newDuel)
			sort.Slice(duelsByTime, func(i, j int) bool {
				return duelsByTime[i].CreateTime.Before(duelsByTime[j].CreateTime)
			})
			duelMutex.Unlock()

			def := GetClassDef(chosenClass)
			platformDesc := "微信"
			if strings.HasPrefix(sender.Type, "qq") {
				platformDesc = "QQ"
			}

			// 展示属性面板
			attrs := initiatorAttrs
			skillNames := ""
			for _, sk := range attrs.Skills {
				skillNames += fmt.Sprintf("[%s%s] ", sk.Emoji, sk.Name)
			}
			sender.Reply(fmt.Sprintf(
				"⚔️ 决斗已发起！ID: %s\n\n"+
					"🎭 职业：%s %s\n%s\n\n"+
					"━━━ 你的属性 ━━━\n"+
					"❤️ 生命：%d  💪 力量：%d\n"+
					"🧠 智力：%d  ⚡ 敏捷：%d\n"+
					"🍀 运气：%d  🛡️ 物防：%d  🔮 魔防：%d\n\n"+
					"🎯 本次技能：%s\n\n"+
					"💰 下注积分：%d，胜者获得 %d（净赚 %d）\n"+
					"平台(%s)玩家回复【迎战】即可加入，等待15分钟",
				duelID,
				def.Emoji, string(chosenClass), def.Description,
				attrs.MaxHP, attrs.Str,
				attrs.Int, attrs.Agi,
				attrs.Luck, attrs.PDef, attrs.MDef,
				skillNames,
				bet, bet*2, bet,
				platformDesc,
			))

			// ──── Step 2: 等待对手加入（后台静默，不拦截任何消息）────
			waitTimeout := time.NewTimer(15 * time.Minute)
			defer waitTimeout.Stop()
			ticker := time.NewTicker(1 * time.Second)
			defer ticker.Stop()
			var targetDuel *Duel
		waitLoop:
			for {
				select {
				case <-waitTimeout.C:
					duelMutex.Lock()
					if d, exists := duels[duelID]; exists && d.Status == "waiting" {
						cleanupDuel(duelID)
						_ = d
					}
					duelMutex.Unlock()
					sender.Reply(fmt.Sprintf("决斗 %s 因超时无人迎战已取消 😔", duelID))
					return
				case <-ticker.C:
					duelMutex.Lock()
					targetDuel, _ = duels[duelID]
					if targetDuel != nil && targetDuel.Challenger != 0 && targetDuel.ChallengerAttrs != nil {
						targetDuel.Status = "fighting"
						duelMutex.Unlock()
						break waitLoop
					}
					duelMutex.Unlock()
				}
			}

			// ──── Step 3: 开始战斗 ────
			RunDuelBattle(duelID, targetDuel, sender)
		}()

		return nil
	},
},
{
	// ────────────────────────────────────────────────────────────
	//  迎战命令：列出等待中的决斗 → 选择 → 选职业 → 加入
	// ────────────────────────────────────────────────────────────
	Command: []string{"迎战", "应战", "挑战"},
	Handle: func(sender *Sender) interface{} {
		if !Config.Game.GameOpen || !Config.Game.DuelOpen {
			sender.Reply("管理员已关闭游戏")
			return nil
		}
		duelMutex.Lock()
		var waitingDuels []*Duel
		now := time.Now()

		// 清理过期决斗
		var validByTime []*Duel
		for _, d := range duelsByTime {
			if now.Sub(d.CreateTime) <= 15*time.Minute {
				validByTime = append(validByTime, d)
			} else {
				delete(duels, d.ID)
			}
		}
		duelsByTime = validByTime

		// 筛选同平台等待中的决斗
		for _, d := range duelsByTime {
			if d.Status == "waiting" && d.Initiator != sender.UserID {
				senderIsQQ := strings.HasPrefix(sender.Type, "qq")
				duelIsQQ := strings.HasPrefix(d.Platform, "qq")
				if senderIsQQ == duelIsQQ {
					waitingDuels = append(waitingDuels, d)
				}
			}
		}
		duelMutex.Unlock()

		if len(waitingDuels) == 0 {
			sender.Reply("当前没有等待中的决斗\n可以回复【决斗】自己发起挑战！")
			return nil
		}

		// 展示决斗列表
		listMsg := "🗡️ 等待挑战者的决斗列表：\n\n"
		for i, d := range waitingDuels {
			remaining := 15*time.Minute - now.Sub(d.CreateTime)
			def := GetClassDef(d.InitiatorClass)
			listMsg += fmt.Sprintf("【%d】决斗ID: %s\n", i+1, d.ID)
			listMsg += fmt.Sprintf("  ├─ 发起者：用户%d (%s%s)\n", d.Initiator, def.Emoji, string(d.InitiatorClass))
			listMsg += fmt.Sprintf("  ├─ 下注积分：%d → 胜者获得 %d\n", d.Bet, d.Bet*2)
			listMsg += fmt.Sprintf("  └─ 剩余时间：%d分%d秒\n\n",
				int(remaining.Minutes()), int(remaining.Seconds())%60)
		}
		listMsg += "30秒内回复序号加入（例如：1）"
		sender.Reply(listMsg)

		// 使用带缓冲的单 channel 全程复用，避免二次注册 ckList 造成的竞态阻塞
		msgChan := make(chan string, 1)
		ckList[sender.UserID] = msgChan

		go func() {
			defer func() {
				delete(ckList, sender.UserID)
				// 排空 channel 后再关闭，防止 bot 主循环卡在 c2 <- msg
				for len(msgChan) > 0 {
					<-msgChan
				}
				close(msgChan)
			}()

			// Step 1: 等待选择决斗序号
			selectTimeout := time.NewTimer(30 * time.Second)
			defer selectTimeout.Stop()
			var selectedDuel *Duel
			select {
			case msg := <-msgChan:
				idx, err := strconv.Atoi(strings.TrimSpace(msg))
				if err != nil || idx < 1 || idx > len(waitingDuels) {
					sender.Reply("输入无效，请重新回复【迎战】")
					return
				}
				selectedDuel = waitingDuels[idx-1]
			case <-selectTimeout.C:
				sender.Reply("选择超时，请重新回复【迎战】")
				return
			}

			// 检查积分
			u := &User{}
			if err := db.Where("number = ?", sender.UserID).First(u).Error; err != nil || u.Coin < selectedDuel.Bet {
				sender.Reply(fmt.Sprintf("参与此决斗需要至少 %d 积分，你的积分不足", selectedDuel.Bet))
				return
			}

			// Step 2: 让挑战者选职业（复用同一 channel，无需重新注册 ckList）
			sender.Reply(ClassListMsg())

			var chosenClass ClassType
			classTimeout := time.NewTimer(40 * time.Second)
			defer classTimeout.Stop()
			select {
			case msg := <-msgChan:
				idx, err := strconv.Atoi(strings.TrimSpace(msg))
				if err == nil && idx >= 1 && idx <= len(AllClasses) {
					chosenClass = AllClasses[idx-1].Name
				} else {
					chosenClass = AllClasses[rand.Intn(len(AllClasses))].Name
					sender.Reply(fmt.Sprintf("输入无效，已随机分配职业：%s", string(chosenClass)))
				}
			case <-classTimeout.C:
				chosenClass = AllClasses[rand.Intn(len(AllClasses))].Name
				sender.Reply(fmt.Sprintf("选择超时，随机分配职业：%s", string(chosenClass)))
			}

			// 生成挑战者属性
			name := sender.Username
			if name == "" {
				name = fmt.Sprintf("用户%d", sender.UserID)
			}
			challengerAttrs := GenerateBattleAttr(sender.UserID, name, chosenClass)

			// Step 3: 加入决斗
			duelMutex.Lock()
			d, exists := duels[selectedDuel.ID]
			if !exists || d.Status != "waiting" || d.Challenger != 0 {
				duelMutex.Unlock()
				sender.Reply("该决斗已被其他人加入或已结束，请重新回复【迎战】")
				return
			}
			d.Challenger = sender.UserID
			d.ChallengerAttrs = challengerAttrs
			d.ChallengerClass = chosenClass
			duelMutex.Unlock()

			def := GetClassDef(chosenClass)
			skillNames := ""
			for _, sk := range challengerAttrs.Skills {
				skillNames += fmt.Sprintf("[%s%s] ", sk.Emoji, sk.Name)
			}
			sender.Reply(fmt.Sprintf(
				"✅ 成功加入决斗 %s！\n\n"+
					"🎭 职业：%s %s\n\n"+
					"━━━ 你的属性 ━━━\n"+
					"❤️ 生命：%d  💪 力量：%d\n"+
					"🧠 智力：%d  ⚡ 敏捷：%d\n"+
					"🍀 运气：%d  🛡️ 物防：%d  🔮 魔防：%d\n\n"+
					"🎯 本次技能：%s\n\n"+
					"战斗即将开始，请等待战报...",
				selectedDuel.ID,
				def.Emoji, string(chosenClass),
				challengerAttrs.MaxHP, challengerAttrs.Str,
				challengerAttrs.Int, challengerAttrs.Agi,
				challengerAttrs.Luck, challengerAttrs.PDef, challengerAttrs.MDef,
				skillNames,
			))
		}()

		return nil
	},
},
{
	// ────────────────────────────────────────────────────────────
	//  决斗排行榜
	// ────────────────────────────────────────────────────────────
	Command: []string{"决斗排行", "战斗排行"},
	Handle: func(sender *Sender) interface{} {
		var users []User
		if err := db.Order("coin desc").Limit(10).Find(&users).Error; err != nil {
			sender.Reply("查询排行榜失败")
			return nil
		}
		var sb strings.Builder
		sb.WriteString("🏆 积分排行榜 TOP 10\n\n")
		medals := []string{"🥇", "🥈", "🥉", "4️⃣", "5️⃣", "6️⃣", "7️⃣", "8️⃣", "9️⃣", "🔟"}
		for i, u := range users {
			medal := ""
			if i < len(medals) {
				medal = medals[i]
			}
			name := u.Nickname
			if name == "" {
				name = fmt.Sprintf("用户%d", u.Number)
			}
			sb.WriteString(fmt.Sprintf("%s %s — %d 积分\n", medal, name, u.Coin))
		}
		sender.Reply(sb.String())
		return nil
	},
},
}


func Guess_Number(sender *Sender, msg chan string, maxGuessCount int) {
	// 生成主要数字
	number := rand.Intn(101)

	// 生成唯一的地雷数字

	landmines := make([]int, 5)

	// 生成两个位于40到60之间的地雷数字
	landmines[0] = rand.Intn(21) + 40
	landmines[1] = rand.Intn(21) + 40

	// 生成其他三个地雷数字
	for i := 2; i < 5; i++ {
		// 生成地雷数字
		landmine := rand.Intn(101)

		// 检查地雷数字是否与主要数字相同
		for landmine == number {
			landmine = rand.Intn(101)
		}

		// 检查地雷数字是否与先前生成的地雷相同
		for j := 0; j < i; j++ {
			if landmine == landmines[j] {
				// 重新生成地雷数字
				landmine = rand.Intn(101)
				j = -1 // 重新检查新生成的地雷数字与之前的地雷数字是否相同
			}
		}

		landmines[i] = landmine
	}

	guessCount := 0

	for {
		n, ok := <-msg
		if !ok {
			break
		}
		if n == "q" {
			sender.Reply("已退出猜数字游戏")
			ckList[sender.UserID] = nil
			close(msg)
			return
		}

		guess, err := strconv.Atoi(n)

		coin := GetCoin(sender.UserID)
		jbcoin := Config.Game.GuessNumberCost
		if coin < jbcoin {
			sender.Reply(fmt.Sprintf("积分不足，每次猜测需要%d个积分，退出游戏！", jbcoin))
			ckList[sender.UserID] = nil
			return
		}

		if err != nil || guess < 0 || guess > 100 {
			sender.Reply(fmt.Sprintf("输入错误，请重新输入0-100内数字，退出请回复q,剩余猜测次数：%d次，已扣除%d个积分，剩余积分%d", maxGuessCount-guessCount, jbcoin, GetCoin(sender.UserID)))
			continue
		}
		guessCount++
		RemCoin(sender.UserID, jbcoin)

		hitMine := false
		for _, mine := range landmines {
			if guess == mine {
				hitMine = true
				break
			}
		}
		if hitMine {
			mineCost := 100
			currentCoin := GetCoin(sender.UserID)
			if currentCoin < mineCost {
				sender.Reply(fmt.Sprintf("很抱歉，你踩中了地雷！需要扣除%d个积分，但当前积分只有%d，游戏结束。", mineCost, currentCoin))
				ckList[sender.UserID] = nil
				close(msg)
				return
			}
			RemCoin(sender.UserID, mineCost)
			sender.Reply(fmt.Sprintf("很抱歉，你运气太差了，踩中了地雷！扣除了100个积分，游戏结束。剩余积分：%d，地雷数字是：%v", GetCoin(sender.UserID), landmines))
			ckList[sender.UserID] = nil
			close(msg)
			return
		} else if guess > number {
			sender.Reply(fmt.Sprintf("输入的数字太大了，请重新输入，剩余猜测次数：%d次，已扣除%d个积分，剩余积分%d", maxGuessCount-guessCount, jbcoin, GetCoin(sender.UserID)))
		} else if guess < number {
			sender.Reply(fmt.Sprintf("输入的数字太小了，请重新输入，剩余猜测次数：%d次，已扣除%d个积分，剩余积分%d", maxGuessCount-guessCount, jbcoin, GetCoin(sender.UserID)))
		} else {
			coin2 := 300 //奖励积分数量
			AdddCoin(sender.UserID, coin2)
			sender.Reply(fmt.Sprintf("恭喜你猜中了！很幸运的避开了地雷数字，你一共猜了%d次，奖励%d个积分，剩余积分%d，其中地雷数字是：%v", guessCount, coin2, GetCoin(sender.UserID), landmines))
			ckList[sender.UserID] = nil
			close(msg)
			return
		}

		if guessCount >= maxGuessCount {
			sender.Reply(fmt.Sprintf("猜数字游戏次数已用完，游戏结束，答案是%d，祝你下次好运！地雷数字是：%v", number, landmines))
			ckList[sender.UserID] = nil
			close(msg)
			return
		}
	}
}

var mx = map[int]time.Time{} // 存储用户上次祈福的日期
// 全局游戏状态变量

var (
    games     = make(map[string]*Game)
    gameMutex sync.Mutex
   gamesByTime []*Game  // 按创建时间升序排列的游戏指针数组
)


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
				} else if strconv.Itoa(cks[i].QQ) == x[1] {
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
	ws := []Wish{}
	if err := tx.Where("status = 0 and user_number = ?", sender.UserID).Find(&ws).Error; err != nil {
		tx.Rollback()
		sender.Reply(err.Error())
	}
	for _, w := range ws {
		if tx.Model(User{}).Where("number = ? ", sender.UserID).Update(
			"coin", gorm.Expr(fmt.Sprintf("coin + %d", w.Coin)),
		).RowsAffected == 0 {
			tx.Rollback()
			sender.Reply("愿望未达成退还积分失败。")
			return
		}
		sender.Reply(fmt.Sprintf("愿望未达成退还%d枚积分。", w.Coin))
		if tx.Model(&w).Update(
			"status", 1,
		).RowsAffected == 0 {
			tx.Rollback()
			sender.Reply("愿望未达成退还积分失败。")
			return
		}
	}
	tx.Commit()
}


//##检查ck显示函数
func GetAccountStatusText(ck *JdCookie) (string, bool) {
	// 核心检测逻辑（复用原有CookieOK函数）
	isValid := CookieOK(ck)
	
	// 带图标的状态文本
	if isValid {
		return "✅有效", true
	}
	return "❌无效", false
}



// 删除美团账号
func Delete_meituan(sender *Sender, msg chan string, meituans []MeiTuan) {
	for {
		n, ok := <-msg
		//说明发送方关闭了channel
		if !ok {
			break
		}
		if n == "q" {
			sender.Reply("退出登录流程")
			meituanList[sender.UserID] = nil
			close(msg)
			return
		}
		num, err := strconv.Atoi(n)
		if err != nil {
			sender.Reply("请输入数字，检测到非数字输入已退出流程!")
			meituanList[sender.UserID] = nil
			return
		}
		regular := `^0$|^[1-9]\d*$`
		reg := regexp.MustCompile(regular)
		if reg.MatchString(n) {

			if len(meituans) <= num {
				sender.Reply("输入序列号错误，已退出！")
				meituanList[sender.UserID] = nil
				return
			}
		} else {
			sender.Reply("输入序列号错误，已退出！！")
			meituanList[sender.UserID] = nil
			return
		}
		meituanck := meituans[num]
		db.Delete(meituanck)

		sender.Reply(fmt.Sprintf("已删除美团账号%s", meituans[num].Nickname))
		meituanList[sender.UserID] = nil
	}
}

func WxImg_ts() {
	type AutoGenerated1 struct {
		Token     string `json:"token"`
		API       string `json:"api"`
		RobotWxid string `json:"robot_wxid"`
		ToWxid    string `json:"to_wxid"`
		Path      string `json:"path"`
	}

	req := httplib.Post(Config.Wx.Url)
	reply := &AutoGenerated1{
		Token:     Config.Wx.Token,
		API:       "SendImageMsg",
		RobotWxid: Config.Wx.Robotid,
		ToWxid:    "21086228291@chatroom",                      //群id
		Path:      "http://180.152.5.230:5701/static/wxzs.jpg", //图片ulr地址
	}
	random := browser.Random()
	req.Header("User-Agent", random)
	marshal, _ := json.Marshal(reply)
	logs.Info(string(marshal))
	req.Body(string(marshal))
	s, _ := req.String()
	logs.Info(s)
}

func PriorityDel() {
	// 获取所有的 JD Cookie，并按照优先级降序排列，取前30名
	allCookies := GetJdCookies(func(sb *gorm.DB) *gorm.DB {
		return sb.Order("priority DESC").Limit(40)
	})

	// 遍历第7名到第30名账号并执行优先级扣除处理并保存
	for i, cookie := range allCookies {
		// 从第7名（下标6）开始，直到第30名（下标29）
		if i >= 6 && i <= 20 {
			// 扣除500优先级
			adjustedPriority := cookie.Priority - 160

			// 更新账号的优先级
			cookie.Priority = adjustedPriority
			// 保存更新后的账号信息到数据库
			db.Save(cookie)
		}
	}
}

func handleGrokChat(sender *Sender, msg chan string) {
	defer func() {
		delete(AiinputList, sender.UserID)
	}()
	apiURL := "https://cloud.luchentech.com/api/maas/chat/completions"
	token := GetEnv("ai_token")
	if token == "" {
		sender.Reply("请先设置ai_token")
		return
	}

	messages := []map[string]interface{}{}
	for {
		timeout := time.After(300 * time.Second)
		select {
		case input, ok := <-msg:
			if !ok || input == "q" {
				sender.Reply("您已退出 Ai 连续对话模式。")
				return
			}

			messages = append(messages, map[string]interface{}{
				"role":    "user",
				"content": input,
			})
			requestBody := map[string]interface{}{
				"model":      "deepseek_r1",
				"messages":   messages,
				"stream":     false,
				"max_tokens": 512,
			}
			body, _ := json.Marshal(requestBody)
			req, _ := http.NewRequest("POST", apiURL, strings.NewReader(string(body)))
			req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))
			req.Header.Add("Content-Type", "application/json")
			res, err := http.DefaultClient.Do(req)
			if err != nil {
				sender.Reply("请求失败，无法从新接口获取响应。")
				logs.Error("发送请求失败: %s", err)
				return
			}
			defer res.Body.Close()
			bodyBytes, err := io.ReadAll(res.Body)
			if err != nil {
				sender.Reply("解析响应失败，请稍后再试。")
				logs.Error("读取响应失败: %s", err)
				return
			}
			if res.StatusCode != 200 {
				sender.Reply(fmt.Sprintf("请求失败，状态码：%d", res.StatusCode))
				logs.Error("请求失败，状态码: %d, 响应: %s", res.StatusCode, string(bodyBytes))
				return
			}
			var response map[string]interface{}
			err = json.Unmarshal(bodyBytes, &response)
			if err != nil {
				sender.Reply("解析响应失败，请稍后再试。")
				logs.Error("解析 JSON 响应错误: %s", err)
				return
			}
			choices := response["choices"].([]interface{})
			if len(choices) == 0 {
				sender.Reply("未收到有效的响应内容。")
				return
			}
			content := choices[0].(map[string]interface{})["message"].(map[string]interface{})["content"].(string)
			messages = append(messages, map[string]interface{}{
				"role":    "assistant",
				"content": content,
			})
			sender.Reply(content)
		case <-timeout:
			sender.Reply("操作超时，退出 Ai 连续对话模式。")
			return
		}
	}
}


//##千寻拉群函数

func QxInviteGroup(uid string, gid string) {
	// 记录开始执行
	logs.Info("开始执行QxInviteGroup，uid: ", uid, " gid: ", gid)

	// 组装请求 URL
	url := Config.Wx.Url + "DaenWxHook/httpapi/?wxid=" + Config.Wx.Robotid
	logs.Info("请求的URL: ", url)

	// 组装请求体
	requestBody := map[string]interface{}{
		"type": "Q0021",
		"data": map[string]string{
			"wxid":    gid,
			"objWxid": uid,
			"type":    "2",
		},
	}
	// 序列化请求体
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		logs.Error("请求体序列化失败:", err)
		return
	}
	logs.Info("请求体序列化成功，内容: ", string(jsonData))

	// 创建 HTTP 请求
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		logs.Error("创建 HTTP 请求失败:", err)
		return
	}
	logs.Info("HTTP 请求创建成功")

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	logs.Info("设置请求头: Content-Type=application/json")

	// 发送 HTTP 请求
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		logs.Error("发送 HTTP 请求失败:", err)
		return
	}
	defer resp.Body.Close()
	logs.Info("HTTP 请求发送成功，状态码: ", resp.StatusCode)

	// 读取响应
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		logs.Error("读取响应失败:", err)
		return
	}
	logs.Info("响应内容读取成功")

	// 记录响应结果
	logs.Info("邀请请求响应:", string(respBody))
}






















//###精粉查询

func QueryJingFen(sender *Sender, ptKey, ptPin string) error {
	url := "https://api.m.jd.com/"
	timestamp := fmt.Sprintf("%d", time.Now().UnixMilli())

	body := fmt.Sprintf(`{"funName":"getIndexStatisticsInfoList","param":{"startTime":"%s","endTime":"%s"}}`,
		time.Now().Format("2006-01-02 00:00:00"),
		time.Now().Format("2006-01-02 00:00:00"))

	params := map[string]string{
		"functionId":    "union_data_bi_wholebulletinboard_api",
		"client":        "android",
		"clientVersion": "3.13.42",
		"appid":         "u_jfapp",
		"loginType":     "2",
		"body":          body,
		"t":             timestamp,
		"openudid":      "",
		"uuid":          "",
		"eid":           "",
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	q := req.URL.Query()
	for k, v := range params {
		q.Add(k, v)
	}
	req.URL.RawQuery = q.Encode()

	// 设置请求头
	cookie := fmt.Sprintf("login_mode=2; qwd_chn=99; qwd_schn=1; pt_key=%s; pt_pin=%s; jxjpin=%s; pwdt_id=%s;", ptKey, ptPin, ptPin, ptPin)
	req.Header.Set("Host", "api.m.jd.com")
	req.Header.Set("accept", "application/json, text/plain, */*")
	req.Header.Set("x-rp-client", "h5_1.0.0")
	req.Header.Set("user-agent", "Mozilla/5.0 (Linux; Android 13; M2012K11AC Build/TKQ1.220829.002; wv) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/104.0.5112.97 Mobile Safari/537.36 JDHybrid/HybridAndroid/2.4.3-target31;jdapp; JXJ/3.13.42")
	req.Header.Set("x-referer-page", "https://jingfenapp.jd.com/pages/commission")
	req.Header.Set("origin", "https://jingfenapp.jd.com")
	req.Header.Set("x-requested-with", "com.jd.jxj")
	req.Header.Set("referer", "https://jingfenapp.jd.com/pages/commission?cacheRemove=1")
	req.Header.Set("accept-encoding", "gzip, deflate")
	req.Header.Set("accept-language", "zh-CN,zh;q=0.9")
	req.Header.Set("cookie", cookie)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var reader io.ReadCloser
	if resp.Header.Get("Content-Encoding") == "gzip" {
		reader, err = gzip.NewReader(resp.Body)
		if err != nil {
			return fmt.Errorf("gzip 解压失败: %v", err)
		}
		defer reader.Close()
	} else {
		reader = resp.Body
	}

	bodyBytes, err := io.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("读取响应失败: %v", err)
	}

	var result struct {
		Code   int `json:"code"`
		Result struct {
			ClickCount        int     `json:"clickCount"`
			IntroduceUv       int     `json:"introduceUv"`
			ValidOrderCount   int     `json:"validOrderCount"`
			ValidOrderAmount  float64 `json:"validOrderAmount"`
			PredictCommission float64 `json:"predictCommission"`
		} `json:"result"`
	}

	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		return fmt.Errorf("json解析失败: %v", err)
	}

	msgs := []string{}

	if result.Code == 200 {
		msgs = append(msgs, fmt.Sprintf("今日点击：%d", result.Result.ClickCount))
		msgs = append(msgs, fmt.Sprintf("引 入 UV：%d", result.Result.IntroduceUv))
		msgs = append(msgs, fmt.Sprintf("有效订单：%d", result.Result.ValidOrderCount))
		msgs = append(msgs, fmt.Sprintf("订单金额：%.2f", result.Result.ValidOrderAmount))
		msgs = append(msgs, fmt.Sprintf("预估收入：%.2f", result.Result.PredictCommission))
	} else {
		msgs = append(msgs, "请求失败或响应格式不正确")
	}
	sender.Reply(strings.Join(msgs, "\n"))

	return nil
}
