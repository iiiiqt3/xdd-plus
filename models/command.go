package models

import (
	"encoding/base64"
	"errors"
	"fmt"
	"math/rand"
	"net/url"
	"os"
	"math"
	"encoding/json"
 	"regexp"
	"strconv"
	"strings"
	"time"
	"bytes"
	"os/exec"
	browser "github.com/EDDYCJY/fake-useragent"
	"github.com/beego/beego/v2/client/httplib"
	"github.com/buger/jsonparser"
	"github.com/beego/beego/v2/core/logs"
	"github.com/google/uuid"
	"github.com/skip2/go-qrcode"
	"gorm.io/gorm"
)

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
	case "wx":
		SendWxImg(sender.WxId, msg)
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

	
	//拉人进微信群
	{  
	Command: []string{"拉群"},  
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
			sender.Reply("请添加微信机器人回复 拉群，加入群聊 ")  
			return nil  
		}  
		},  
	},









{
		Command: []string{"拉群"},
		Handle: func(sender *Sender) interface{} {
			if sender.Type == "wx" {
				if sender.IsAdmin {
					ExportEnv(&Env{
						Name:  "WxGroupID",
						Value: sender.WxGroupId,
					})
					return "已将此群设为拉群目标"
				} else {
					env := GetEnv("WxGroupID")
					if env != "" {
						InviteGroup(sender.WxId, Config.InviteGroupID)
						return nil
					} else {
						return "未设置拉群目标！"
					}
				}
			}
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
		  Command: []string{"GPT", "GPT4", "gpt"},
		    Handle: func(sender *Sender) interface{} {

			coin := GetCoin(sender.UserID)
			   if coin <=200 {
			   	return "为避免接口滥用，积分小于200将无法使用gpt服务"
		    }
		        // 将内容连接成一个字符串
		        content := strings.Join(sender.Contents, " ")
		
		        // 修剪内容中的空格
		        content = strings.TrimSpace(content)
		
		        // 检查是否提供了非空内容
		        if content == "" {
		            return "请输入正确的格式命令和内容中间有个空格，例如： GPT 群主帅吗"
		        }
		
		        // todo: 接入GPT-4
		        url := GetEnv("gpt")
		        token := GetEnv("gpt_token")
		        post := httplib.Post(url)
		        post.Header("Authorization", token)
		        post.Header("Content-Type", "application/json")
		        post.Body(fmt.Sprintf("{\n  \"model\": \"gpt-4-1106-preview\",\n  \"messages\": [\n    {\n      \"role\": \"user\",\n      \"content\": \"%s\"\n    }\n  ]\n}", content))
		        bytes, _ := post.Bytes()
		        logs.Info(string(bytes))
		        val, _ := jsonparser.GetString(bytes, "choices", "[0]", "message", "content")
		
		        return val
		    },
		},

		{
	    Command: []string{"记录ck", "提交ck","记录CK", "提交CK"},
	    Handle: func(sender *Sender) interface{} {
	        // 检查命令参数是否符合格式
	        if len(sender.Contents) != 3 {
	            sender.Reply("格式错误！！正确格式为：记录ck 你的ck 备注 活动代号 ，4个参数之间用空格链接，如提示错误请自查" )
	            return nil
	        }
	
	        // 获取命令参数
	        value := sender.Contents[0]
	        remarks := sender.Contents[1]
	        env_name := sender.Contents[2]
	
	        // 获取扣积分设置
	        value3 := GetEnv(env_name)
	
	        // 检查是否开启了小团币功能
	        if value3 == "" {
	            sender.Reply(fmt.Sprintf("%s未开启添加功能", env_name))
	        } else {
	            // 获取用户的积分
	            coin := GetCoin(sender.UserID)
	            jbcoin, _ := strconv.Atoi(value3)
	
	            // 检查用户积分是否足够
	            if coin < jbcoin {
	                sender.Reply(fmt.Sprintf("积分不足，%s需要%d个积分，请直接私聊微信机器人转账，1元=100积分，转账成功即可完成积分充值，或者联系群主购买", env_name, jbcoin))
	            } else {
	                // 执行记录植白说账号的脚本
	                cmd := exec.Command("python3", "scripts/record.py", value, remarks, env_name)
	                var stdout, stderr bytes.Buffer
	                cmd.Stdout = &stdout
	                cmd.Stderr = &stderr                
	
	                // 执行命令
	                err := cmd.Run()
	                // 检查脚本执行结果
	                if err == nil {
	                	// 检查标准输出是否包含'记录成功'
	                	outputStr := stdout.String()
	                	if strings.Contains(outputStr, "记录成功") {
	                    // 扣除用户积分
	                    RemCoin(sender.UserID, jbcoin)
	                    sender.Reply(fmt.Sprintf("添加%s账号，已扣除%d个积分，剩余积分%d", env_name, jbcoin, GetCoin(sender.UserID)))
	                    sender.Reply("记录成功。")
	                } else {
	                    errorMsg := fmt.Sprintf("提示信息：%s", outputStr)
	                    sender.Reply(errorMsg)
	                }
	                } else {
	                	// 输出错误信息
	                	errorMsg := fmt.Sprintf("错误信息：%s", stderr.String())
	                	sender.Reply(errorMsg)
                        }
	            }
	        }
	
	        return nil
	    },
	},




	
	{
	    Command: []string{"更新ck"},
	    Handle: func(sender *Sender) interface{} {
	        // 检查命令参数是否符合格式
	        if len(sender.Contents) != 3 {
	            sender.Reply("格式错误！！正确格式为：更新ck 你的ck 备注 活动代号")
	            return nil
	        }
	
	        // 获取命令参数
	        value := sender.Contents[0]
	        remarks := sender.Contents[1]
	        env_name := sender.Contents[2]
	
	        // 获取扣积分设置
	        value3 := GetEnv("up" + env_name)
	
	        // 检查是否开启了小团币功能
	        if value3 == "" {
	            sender.Reply(fmt.Sprintf("%s未开启更新功能", env_name))
	        } else {
	            // 获取用户的积分
	            coin := GetCoin(sender.UserID)
	            jbcoin, _ := strconv.Atoi(value3)
	
	            // 检查用户积分是否足够
	            if coin < jbcoin {
	                sender.Reply(fmt.Sprintf("积分不足，%s更新需要%d个积分，请直接私聊微信机器人转账，1元=100积分，转账成功即可完成积分充值，或者联系群主购买", env_name, jbcoin))
	            } else {
	                // 执行记录植白说账号的脚本
	                cmd := exec.Command("python3", "scripts/updata.py", value, remarks, env_name)
	                var stdout, stderr bytes.Buffer
	                cmd.Stdout = &stdout
	                cmd.Stderr = &stderr                
	
	                // 执行命令
	                err := cmd.Run()
	                // 检查脚本执行结果
	                if err == nil {
	                	// 检查标准输出是否包含'更新成功'
	                	outputStr := stdout.String()
	                	if strings.Contains(outputStr, "更新成功") {
	                    // 扣除用户积分
	                    RemCoin(sender.UserID, jbcoin)
	                    sender.Reply(fmt.Sprintf("更新%s账号，已扣除%d个积分，剩余积分%d", env_name, jbcoin, GetCoin(sender.UserID)))
	                    sender.Reply("更新成功。")
	                } else {
	                    errorMsg := fmt.Sprintf("提示信息：%s", outputStr)
	                    sender.Reply(errorMsg)
	                }
	                } else {
	                	// 输出错误信息
	                	errorMsg := fmt.Sprintf("错误信息：%s", stderr.String())
	                	sender.Reply(errorMsg)
                        }
	            }
	        }
	
	        return nil
	    },
	},


	
	
/*	{
		Command: []string{"登12录", "登34陆"},
		Handle: func(sender *Sender) interface{} {
			c2 := make(chan string)
			smsList[sender.UserID] = c2
			sender.Reply("请输入手机号")
			go SmsSelect(sender, c2, "Nolan")
			return nil
		},
	},

*/

	{
		Command: []string{"登录", "登陆"},
		Handle: func(sender *Sender) interface{} {
			c3 := make(chan string)
			smsList[sender.UserID] = c3
	//		sender.Reply("请输入手机号")
			go SmsSelect(sender, c3, "Rabbit")
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

 	Command: []string{"我的plus","我的PLUS" },
        Handle: func(sender *Sender) interface{} {
            sender.handleJdCookies(func(ck *JdCookie) {
                cc := fmt.Sprintf("pt_key=%s;pt_pin=%s;", ck.PtKey,ck.PtPin)
                rsp := cmd(fmt.Sprintf(`python3 ./jd_plus_score.py "%s"`, cc), &Sender{})
                sender.Reply(rsp)
            })

            return nil
        },
    },

/*
	    {  
			 Command: []string{ "我的排名"},   
			 Handle: func(sender *Sender) interface{} {    
			 // 从发送者中获取用户的 QQ 号码    
			 userQQ := sender.UserID    
			   
			 // 初始化变量，用于存储用户的排名和用户是否拥有 JD Cookie    
			 var userRanking int    
			 var hasCookie bool    
			   
			 // 获取所有的 JD Cookie    
			 allCookies := GetJdCookies(func(sb *gorm.DB) *gorm.DB {  
			 return sb.Where(fmt.Sprintf("%s = ? ", Available), True)  
			 })    
			   
			 // 遍历所有的 JD Cookie    
			 for i, cookie := range allCookies {  
			 // 检查当前的 JD Cookie 是否属于该用户且可用  
			 if cookie.QQ == userQQ  {  
			 // 如果用户有可用的 JD Cookie，设置排名为当前遍历的索引值加一（因为索引从零开始）  
			 userRanking = i + 1  
			 hasCookie = true  
			 break  
			 }  
			 }  
			   
			 // 检查用户是否有 JD Cookie    
				// 检查用户是否有 JD Cookie  
				if hasCookie {  
					// 如果用户有 JD Cookie，回复他们的排名信息  
					replyMessage := fmt.Sprintf("您优先级最高的账号排名是第 %d 位", userRanking)  
					sender.Reply(replyMessage)  
				} else {  
					// 如果用户没有 JD Cookie，通知他们  
					sender.Reply("您尚未绑定 JD Cookie，无法查询排名。")  
				}  
		  
		
				return nil // 返回 nil，因为没有需要返回的结果
				},
						
						
			},

*/


		{
		
		Command: []string{"我的排名"},
		Handle: func(sender *Sender) interface{} {
		    // 从发送者中获取用户的 QQ 号码
		    userQQ := sender.UserID
		
		    // 初始化变量，用于存储用户的排名、用户名和优先级，以及用户是否拥有 JD Cookie
		    var userRankings []struct {
		        Rank     int
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
		                Username string
		                Priority int
		            }{
		                Rank:     i + 1,
		                Username: cookie.Nickname,
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
		            replyMessage = fmt.Sprintf("您的用户名：%s，优先级：%d ，排名是第 %d 位",  userRankings[0].Username, userRankings[0].Priority,userRankings[0].Rank)
		        } else {
		            replyMessage = "您账号排名信息："
		            for idx, ranking := range userRankings {
		                replyMessage += fmt.Sprintf("\n%d、用户名：%s，优先级：%d ，排名：%d 位", idx+1, ranking.Username, ranking.Priority, ranking.Rank)
		            }
		        }
		        sender.Reply(replyMessage)
		    } else {
		        // 如果用户没有 JD Cookie，通知他们
		        sender.Reply("您账号可能失效，或者尚未登录，无法查询排名。")
		    }
		
		    return nil // 返回 nil，因为没有需要返回的结果
		},
		
		},




		
	
	{
		Command: []string{"微信扫码"},
		Handle: func(sender *Sender) interface{} {
			BBKGetWxQrImg(sender)
			return nil
		},
	},

	{
		Command: []string{"R京东扫码"},
		Handle: func(sender *Sender) interface{} {
			RabbitGetJdQrImg(sender)
			return nil
		},
	},


{
		Command: []string{"扫码"},
		
		Handle: func(sender *Sender) interface{} {
		if sender.IsAdmin {
		sender.Reply("开始京东扫码登录")
		  } else {
			value := GetEnv("sm")
				if value == "" {
					return "未开启扫码登录"
							} else {
						coin := GetCoin(sender.UserID)
							jbcoin, _ := strconv.Atoi(value)
								if coin < jbcoin {
								return fmt.Sprintf("扫码登录需要%d个积分,请直接私聊微信机器人转账，1元=100积分，转账成功即可完成积分充值，或者联系群主购买", jbcoin)
									}
									
									RemCoin(sender.UserID, jbcoin)
									sender.Reply(fmt.Sprintf("扫码即将开始，已扣除%d个积分,剩余%d", jbcoin,  GetCoin(sender.UserID)))
								}
							}

			NolanGetJdQrImg(sender)
			//sender.Reply("渠道升级，预计今晚修复完成")
			return nil
		},
	},


	{
		Command: []string{"农场浇水", "浇水农场"},
		Handle: func(sender *Sender) interface{} {
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

				//进入队列
				msg := make(chan string)
				ckList[sender.UserID] = msg
				go Jd_fruit_watering(sender, msg, cks)
				msgs := []string{
					"请回复以下序列号指定账号运行任务，如需退出请回复'q'退出登录流程：",
				}
				for i, ck := range cks {
					msgs = append(msgs, fmt.Sprintf("%d、%s", i, ck.Nickname))
				}
				sender.Reply(strings.Join(msgs, "\n"))
			} else {
				sender.Reply("在线账号已全部失效，请对机器人发送“登录”")
				return nil
			}
			return nil
		},
	},


	{
		Command: []string{"一键保价","一键价保"},
		Handle: func(sender *Sender) interface{} {
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

				//进入队列
				msg := make(chan string)
				ckList[sender.UserID] = msg
				go Jd_price(sender, msg, cks)
			
				msgs := []string{
					"请回复以下序列号指定账号运行任务，如需退出请回复'q'退出登录流程：",
				}
				for i, ck := range cks {
					msgs = append(msgs, fmt.Sprintf("%d、%s", i, ck.Nickname))
				}
				sender.Reply(strings.Join(msgs, "\n"))
			} else {
				sender.Reply("在线账号已全部失效，请对机器人发送“登录”")
				return nil
			}
			return nil
		},
	},


{
		Command: []string{"一键评价"},
		Handle: func(sender *Sender) interface{} {
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

				//进入队列
				msg := make(chan string)
				ckList[sender.UserID] = msg
				go Jd_AutoEval(sender, msg, cks)
			
				msgs := []string{
					"请回复以下序列号指定账号运行任务，如需退出请回复'q'退出登录流程：",
				}
				for i, ck := range cks {
					msgs = append(msgs, fmt.Sprintf("%d、%s", i, ck.Nickname))
				}
				sender.Reply(strings.Join(msgs, "\n"))
			} else {
				sender.Reply("在线账号已全部失效，请对机器人发送“登录”")
				return nil
			}
			return nil
		},
	},




		


{
		Command: []string{"sign", "打卡", "签到"},
		Handle: func(sender *Sender) interface{} {
		//	if sender.Type == "tgg" {
		//		sender.Type = "tg"
		//	}
		//	if sender.Type == "qqg" {
		//		sender.Type = "qq"
		//	}
			zero, _ := time.ParseInLocation("2006-01-02", time.Now().Local().Format("2006-01-02"), time.Local)
			var u User
			var ntime = time.Now()
			var first = false
			total := []int{}
			err := db.Where("number = ?", sender.UserID).First(&u).Error
			if err != nil {
				first = true
				u = User{
					Class:    sender.Type,
					Number:   sender.UserID,
					Coin:     1,
					ActiveAt: ntime,
				}
				if err := db.Create(&u).Error; err != nil {
					return err.Error()
				}
			} else {
				if first || zero.Unix() > u.ActiveAt.Unix() {
					first = true
			} else {
				    return fmt.Sprintf("你打过卡了，积分余额%d。", u.Coin)
				}

			}
			if first {
				db.Model(User{}).Select("count(id) as total").Where("active_at > ?", zero).Pluck("total", &total)
				coin := 5
				if total[0] == 0 {
					coin = 20
				}
				if total[0] == 1 {
					coin = 18
				}
				if total[0] == 2 {
					coin = 16
				}
				if total[0] == 3 {
					coin = 14
				}
				if total[0] == 4 {
					coin = 12
				}
				if total[0] == 5 {
					coin = 10
				}
				if total[0] == 6 {
					coin = 8
				}
				if total[0] == 7 {
					coin = 6
				}
				
				if total[0] == 50 {
					coin = 10
				}
				if total[0] == 100 {
					coin = 20
				}
				if total[0]%14 == 13 {
					coin = 6
				}
				
 				db.Model(&u).Updates(map[string]interface{}{
					"active_at": ntime,
					"coin":      gorm.Expr(fmt.Sprintf("coin+%d", coin)),
				})
				u.Coin += coin
				sender.Reply(fmt.Sprintf("你是打卡第%d人，奖励%d个积分，积分余额%d。", total[0]+1, coin, u.Coin))
				ReturnCoin(sender)
		//		return ""
			}
			return nil
		},
	},






	
/*	{
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
		Command: []string{"更新优先级", "更新车位", "ces"},
		Handle: func(sender *Sender) interface{} {
			coin := GetCoin(sender.UserID)
			var total int64
			db.Model(&JdCookie{QQ: sender.UserID}).Where(fmt.Sprintf("QQ = %d", sender.UserID)).Count(&total)
			var intNum int = int(total)
			coin = coin / intNum
			t := time.Now()
			sender.Reply("提醒：如果你有多个账号，优先级会被平分到所有账户 ，5秒后开始优先级更新")
			time.Sleep(time.Second * 5)
			if t.Weekday().String() == "Monday" && int(t.Hour()) <= 22 {
				sender.handleJdCookies(func(ck *JdCookie) {
					ck.Update(Priority, ck.Priority+coin)
				})
				sender.Reply("优先级已做累积更新，")
				ClearCoin(sender.UserID)
			} else {
				sender.Reply("你错过时间了呆瓜,下周一23点前再来吧，友情提醒：优先级更新会平分账号")
			}
			return nil
		},
	},

*/

   
   
   {  
     Command: []string{"更新优先级", "更新车位", "车位更新", "优先级更新"},
	Handle: func(sender *Sender) interface{} {
    if sender.IsAdmin {
        sender.Reply("管理员请不要使用此命令")
    } else {
        // 检查命令参数
        if len(sender.Contents) < 1 { // 至少应该有两个参数
            sender.Reply("请在命令后面带入你需要更新的数字, 例如: 更新车位 100")
            return nil
        }

        // 提取积分值
        userCoinStr := sender.Contents[0]
        userCoin, err := strconv.ParseInt(userCoinStr, 10, 64)
        if err != nil || userCoin <= 0 {
            sender.Reply("输入的积分值无效，请确保输入的是一个正整数。")
            return nil
        }

        // 获取用户当前积分
        coin := GetCoin(sender.UserID)
        if int(userCoin) > int(coin) {
            sender.Reply("你输入的数字，超过你拥有的积分")
            return nil
        }

        // 获取数据库中的ck数量
        var total int64
        // db.Model(&JdCookie{QQ: sender.UserID}).Where(fmt.Sprintf("QQ = %d", sender.UserID)).Count(&total)

        db.Model(&JdCookie{QQ: sender.UserID}).Where(fmt.Sprintf("QQ = %d AND %s = ?", sender.UserID, Available), True).Count(&total)

        var intNum int = int(total)
        logs.Info(intNum)
        if intNum == 0 {
            sender.Reply("你没有任何ck可以更新，或者你的ck全部失效。")
            return nil
        }
        var coin1 int = int(math.Round(float64(userCoin) / float64(intNum)))
        logs.Info(coin1)

        // 更新ck的优先级
        // t := time.Now()
        // if t.Weekday().String() == "Monday" && int(t.Hour()) <= 23 {
        sender.handleJdCookies(func(ck *JdCookie) {
            ck.Update(Priority, ck.Priority+coin1)
        })

        sender.Reply("正在更新未失效账号的优先级，请稍后。")

        RemCoin(sender.UserID, int(userCoin))
        time.Sleep(time.Second * 2)
        // 发送回复消息
        var tcoin int
        tcoin = GetCoin(sender.UserID)
        sender.Reply(fmt.Sprintf("未失效账号优先级已做累积更新，并扣除了相应的积分，当前剩余积分%d", tcoin))
        // return nil
        // } else {
        // sender.Reply("你错过时间了呆瓜,请每周一23点前再来吧，友情提醒：优先级更新会平分未失效的账号")
        // }
        }
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
		Command: []string{"余额", "积分", "我的积分"},
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
	Command: []string{"绑定微信","微信绑定"},
		Handle: func(sender *Sender) interface{} {
		if sender.Type == "wx" || sender.Type == "wxg"{
			sender.Reply("请对QQ机器人发送指令，得到绑定码，把绑定码发给微信机器人，打通QQ、微信 ，app客户端，三端查询和积分打卡系统")
			return nil
			} else {
			sender.Reply("请复制发送给Wx机器人完成绑定，添加后可回复 拉群 入群加入微信群聊，绑定后打通QQ、微信 ，app客户端，三端查询和积分打卡系统")
			}
			return makeWxId(sender.UserID, "DXWX"+getMd5String1(strconv.Itoa(sender.UserID)))
		},
	},



	{
		Command: []string{"授权"},
		//Admin:   true,
		Handle: func(sender *Sender) interface{} {
			value3 := GetEnv("sqelm")
			if value3 == "" {
					sender.Reply("管理员未开启授权添加功能sqelm")
				} else {
					coin := GetCoin(sender.UserID)
					jbcoin, _ := strconv.Atoi(value3)
					if coin < jbcoin {
						sender.Reply(fmt.Sprintf("积分不足，添加授权需要%d个积分,请登录京东账号获取奖励（或私聊群主积分卡）", jbcoin))
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







/*	{
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
*/
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
		Command: []string{"我的优先级"},
	
		Handle: func(sender *Sender) interface{} {
			str := ""
			sender.handleJdCookies(func(ck *JdCookie) {
				str = str + fmt.Sprintf("昵称：%s 用户名：%s QQ：%d 优先级：%d \n", ck.Nickname, ck.PtPin, ck.QQ, ck.Priority)
			})
			return str
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
				rt := fmt.Sprintf("你的账号【%s】已过期，请对机器人发（登录）重新上", ck.Nickname)
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
		Command: []string{"查22询", "query"},
		Handle: func(sender *Sender) interface{} {
			sender.Reply("正在为您查询，请耐心等待,需要查询更加全面的信息，请使用 ：我的资产 口令")
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
					list := getUserNameList(strconv.Itoa(sender.UserID))
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
						url := fmt.Sprintf("%squery#/?id=%d", query, sender.UserID)
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
					list := getUserNameList(strconv.Itoa(sender.UserID))
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
        Command: []string{"我的资产", "查询"},
        
        Handle: func(sender *Sender) interface{} {    
        sender.Reply("正在为您查询，请耐心等待,")    
                         
                sender.handleJdCookies(func(ck *JdCookie) {
                     sender.Reply(ck.Query())
                })
            
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
		Command: []string{"QQ转账"},
		Admin:   true,
		Handle: func(sender *Sender) interface{} {
			qq := Int(sender.Contents[0])
			logs.Info(qq)
			if len(sender.Contents) > 1 {
				//sender.Contents = sender.Contents[1:]
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
    
        if len(sender.Contents) < 2 {    
            sender.Reply("请输入正确的指令格式，如: 转账 [对方userid号] [积分数量]，例如：转账  2345 100，表示你给2345给12345转了100积分，转账将扣除10%手续费，关于 userid获取方法，发送指令：用户信息")    
            return nil    
        }    
    
        // 获取接收转账的QQ号    
        toQQStr := sender.Contents[0]    
        toQQ, err := strconv.Atoi(toQQStr)    
        if err != nil {    
            sender.Reply("无效的QQ号。")    
            return nil    
        }    


    
  
        // 获取转账的积分数量    
        coinStr := sender.Contents[1]    
        coin, err := strconv.Atoi(coinStr)    
        if err != nil {    
            sender.Reply("请输入正确的积分数量。")    
            return nil    
        }    
    
        // 检查积分    
        senderCoins := GetCoin(qq)    
        if senderCoins < coin {    
            sender.Reply("积分不足，无法完成转账。")    
            return nil    
        }    
		// 检查积分是否为负数  
    	   if coin < 0 {  
  	   sender.Reply("积分不能为负数。")  
  		 return nil  
		}
        // 调用函数执行转账操作    
        RemCoin(qq, coin)    
        reCoins := int(float64(coin) * 0.9) // 计算接收者实际得到的积分，取整数部分
        receivedCoins := int(float64(coin) * 0.9) // 计算接收者实际得到的积分，取整数部分
	   AdddCoin(toQQ, receivedCoins)

		senderCoinsAfter := GetCoin(qq)
		receivedCoins = GetCoin(toQQ)
        // 回复消息给发送者    
        sender.Reply(fmt.Sprintf("你已向%d转账%d枚积分，剩余积分：%d；扣除手续费后对方获得：%d积分，对方积分余额为：%d", toQQ, coin, senderCoinsAfter, reCoins,receivedCoins))    
    
        return nil    
    	  },    
	},


	
	{
		Command: []string{"踩雷", "拼了"},  
			Handle: func(sender *Sender) interface{} {  
    				u := &User{}  
  
   					 cost := Int(sender.JoinContens())  
  				
  					  if cost < 0 {  
  				      return "不允许输入负数"  
   					 }  
  
  					  if cost <= 0 || cost > 100000000000000 {  
    			  	//  cost = 1  

    			  	 return "请在命令后面带入正整数：例如为：踩雷 30"
   			 }  
				
				if err := db.Where("number = ?", sender.UserID).First(u).Error; err != nil || u.Coin < cost {
					return "哎呀积分不够了，快去搞点积分吧？=> 请直接私聊微信机器人转账，1元=100积分，转账成功即可完成积分充值，或者联系群主购买"
				} else {
					currentBalance := GetCoin(sender.UserID) - cost
					sender.Reply(fmt.Sprintf("你使用%d枚积分，使用后积分余额%d", cost,currentBalance))
				}
				baga := 0
				if u.Coin > 100000000000 {
					baga = u.Coin
					cost = u.Coin
				}
				r := time.Now().Nanosecond() % 10
				if r < 7 || baga > 0 {
					currentBalance := GetCoin(sender.UserID) - cost
					sender.Reply(fmt.Sprintf("很遗憾你失去了%d枚积分，当前积分余额%d", cost, currentBalance))
					cost = -cost
				} else {
					if r == 9 {
						cost *= 2
						
						sender.Reply(fmt.Sprintf("恭喜你2倍暴击获得%d枚积分，2秒后自动转入余额。", cost))
						time.Sleep(time.Second * 2)
					} else 
					if r == 8 {
						cost *=1 
						sender.Reply(fmt.Sprintf("恭喜你暴击获得%d枚积分，2秒后自动转入余额。", cost))
						time.Sleep(time.Second * 2)
							
					} else {
						sender.Reply(fmt.Sprintf("很幸运你获得%d枚积分，2秒后自动转入余额。", cost))
						time.Sleep(time.Second * 2)
					}
					currentBalance := GetCoin(sender.UserID) + cost
					sender.Reply(fmt.Sprintf("%d枚积分已到账，当前积分余额%d", cost, currentBalance))
				}
				db.Model(u).Update("coin", gorm.Expr(fmt.Sprintf("coin + %d", cost)))
				return nil
			},
		},

			


		

/*			{
		Command: []string{"翻翻乐","赌一把"},
		Handle: func(sender *Sender) interface{} {

			cost := Int(sender.JoinContens())
			if cost <= 0 || cost > 20 {
				cost = 20
			}
			u := &User{}
			if err := db.Where("number = ?", sender.UserID).First(u).Error; err != nil || u.Coin < cost {
				return "你个穷逼，积分不足20，努力赚取积分吧"
			}
			baga := 0
			if u.Coin > 1000000000000 {
				baga = u.Coin
				cost = u.Coin
			}
			r := time.Now().Nanosecond() % 10
			if r < 6 || baga > 0 {
				sender.Reply(fmt.Sprintf("很遗憾你失去了%d枚积分。", cost))
				cost = -cost
			} else {
				if r == 9 {
					cost *= 2
					sender.Reply(fmt.Sprintf("恭喜你幸运暴击x2获得%d枚积分，2秒后自动转入余额。", cost))
					time.Sleep(time.Second * 2)
				} else {
					sender.Reply(fmt.Sprintf("很幸运你获得%d枚积分，2秒后自动转入余额。", cost))
					time.Sleep(time.Second * 2)
				}
				sender.Reply(fmt.Sprintf("%d枚积分已到账。", cost))
			}
			db.Model(u).Update("coin", gorm.Expr(fmt.Sprintf("coin + %d", cost)))
			return nil
		},
	},

*/
	//{
	//	Command: []string{"按许愿币更新排名"},
	//	Admin:   true,
	//	Handle: func(sender *Sender) interface{} {
	//		cookies:= GetJdCookies()
	//		for i := range cookies {
	//			cookie := cookies[i]
	//			if cookie.QQ {
	//
	//			}
	//			cookie.Update(Priority,cookie.)
	//		}
	//		sender.handleJdCookies(func(ck *JdCookie) {
	//			sender.Reply(ck.Query())
	//		})
	//		return "已更新排行"
	//	},
	//},
	
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


    // https://pp.iaka.cn/api/ajax.php?act=search&name=短剧名称 通过api写出搜剧代码，识别命令搜据，并解析空格后的剧名进行api查询并返回查询结果








{
    Command: []string{"搜剧","短剧"},
    Handle: func(sender *Sender) interface{} {
        if len(sender.Contents) == 0 {
            return "请输入正确的指令比如：搜剧 我能异世界穿梭"
        }
        name := sender.Contents[0]
        searchURL := fmt.Sprintf("https://pp.iaka.cn/api/ajax.php?act=search&name=%s", url.QueryEscape(name))
        req := httplib.Get(searchURL)
        req.Header("User-Agent", browser.Random())
        bytes, err := req.Bytes()
        if err != nil {
            return err.Error()
        }
        
        // 解析JSON响应
        var response map[string]interface{}
        if err = json.Unmarshal(bytes, &response); err != nil {
            return err.Error()
        }
        
        // 检查返回的code是否为0
        code, ok := response["code"].(string)
        if !ok || code != "0" {
            return "未找到相关剧集"
        }
        
        // 提取链接
        data, ok := response["data"].([]interface{})
        if !ok || len(data) == 0 {
            return "未找到相关剧集"
        }
        firstData := data[0].(map[string]interface{})
        episodeURL, ok := firstData["url"].(string)
        if !ok {
            return "未找到相关剧集链接"
        }
        
        return episodeURL
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
        today := time.Now().Format("2006-01-02") // 获取今天的日期    
        lastPrayDate, ok := mx[sender.UserID] // 检查用户上次祈福的日期    
  
        if ok && lastPrayDate.Format("2006-01-02") == today { // 如果用户今天已经祈福过    
            return "你今天已经祈福过了，明天再来吧。"    
        }    
  
        
        if time.Now().Unix() % 2 == 0 {  
             
            mx[sender.UserID] = time.Now()   
            return "祈福诚意不足，祈福失败，不增加积分。"    
        } else {  
            mx[sender.UserID] = time.Now()  
            if db.Model(User{}).Where("number = ? ", sender.UserID).Update(    
                "coin", gorm.Expr(fmt.Sprintf("coin + %d", 3)),    
            ).RowsAffected == 0 {    
                return "先去打卡吧你。"    
            }    
            return "祈福成功，愿你事事顺心如意，积分+3，"    
        } 
           
   	 }, 
   	    
	},
			
	
/*  #注释掉原有的祈福代码
	
	{
		Command: []string{"祈祷", "祈愿", "祈福"},
		Handle: func(sender *Sender) interface{} {
			if _, ok := mx[sender.UserID]; ok {
				return "你祈祷过啦，等下次我忘记了再来吧。"
			}
			mx[sender.UserID] = true
			if db.Model(User{}).Where("number = ? ", sender.UserID).Update(
				"coin", gorm.Expr(fmt.Sprintf("coin + %d", 1)),
			).RowsAffected == 0 {
				return "先去打卡吧你。"
			}
			return "积分+1"
		},
	},





		{  
		 Command: []string{"祈祷", "祈愿", "祈福"},  
		 Handle: func(sender *Sender) interface{} {  
		 today := time.Now().Format("2006-01-02") // 获取今天的日期  
		 lastPrayDate, ok := mx[sender.UserID] // 检查用户上次祈福的日期  
		  
		 if ok && lastPrayDate.Format("2006-01-02") == today { // 如果用户今天已经祈福过  
		 return "你今天已经祈福过了，明天再来吧。"  
		 }  
		  
		 mx[sender.UserID] = time.Now() // 更新用户的上次祈福日期为今天  
		 if db.Model(User{}).Where("number = ? ", sender.UserID).Update(  
		 "coin", gorm.Expr(fmt.Sprintf("coin + %d", 1)),  
		 ).RowsAffected == 0 {  
		 return "先去打卡吧你。"  
		 }  
		 return "祈福成功，愿你事事顺心如意，积分+1，"  
		 
		 },  
	 },
		
	*/




	
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
		Command: []string{"删除账号", "账号删除"},
		Handle: func(sender *Sender) interface{} {
			id := sender.UserID
			var idType string
			if sender.Type == "tg" {
				idType = Telegram
			} else {
				idType = QQ
			}
			cks := GetJdCookies(func(sb *gorm.DB) *gorm.DB {
				//return sb.Where(fmt.Sprintf("%s = ?", idType), id)
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
				sender.Reply("无失效账号，新增账号请对机器人发送“登录”")
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
		Command: []string{"用户信息"},
		Handle: func(sender *Sender) interface{} {
			return fmt.Sprintf("用户ID：%d", sender.UserID)
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
		Admin:   true,
		Handle: func(sender *Sender) interface{} {
			sender.handleJdCookies(func(ck *JdCookie) {
				sender.Reply(fmt.Sprintf("pin=%s;wskey=%s;", ck.PtPin, ck.WsKey))
			})
			return nil
		},
	},
	{
		Command: []string{"回填微信","微信回填"},
		Admin:   true,
		Handle: func(sender *Sender) interface{} {
			cks := GetJdCookies()
			xx := 0
			successCount := 0
			for i := range cks {
				if cks[i].QQ != 0 {
					//logs.Info(cks[i].QQ)
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
        // 获取游戏所需硬币数量和用户当前硬币数量
        value := GetEnv("csz")
        jbcoin, _ := strconv.Atoi(value)
        coin := GetCoin(sender.UserID)

        // 检查用户硬币是否足够
        maxGuessTimesStr := GetEnv("cszcs")
        if maxGuessTimesStr == "" {
            sender.Reply("管理员未设置游戏次数，变量名称为cszcs")
            return nil
        }

        maxGuessTimes, err := strconv.Atoi(maxGuessTimesStr)
        if err != nil {
            sender.Reply("无法将游戏次数转换为整数")
            return nil
        }

        if coin < jbcoin*maxGuessTimes {
            // 如果硬币不足，发送消息并退出游戏
            sender.Reply("积分不够本次游戏，已退出游戏，请直接私聊微信机器人转账，1元=100积分，转账成功即可完成积分充值，或者联系群主购买")
            return nil
        }

        // 创建通道并关联到ckList映射中的用户ID
        msg := make(chan string)
        ckList[sender.UserID] = msg

        // 设置最大猜测次数并启动猜数字游戏
        go Guess_Number(sender, msg, maxGuessTimes)

        // 如果硬币足够，发送游戏规则消息
        message := fmt.Sprintf("猜数字规则：游戏次数%d次，每次猜扣%d积分，猜中奖励300积分，游戏过程中存在5个地雷猜中直接扣100积分，结束游戏。\n请回复0-100内数字猜测，中途如需退出游戏回复'q'退出流程：", maxGuessTimes, jbcoin)
        sender.Reply(message)

        // 返回nil表示函数执行完毕
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
    value := GetEnv("csz")

    if value == "" {
        sender.Reply("管理员未开启猜数字游戏")
        ckList[sender.UserID] = nil
        return
    }

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
        jbcoin, _ := strconv.Atoi(value)
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
            RemCoin(sender.UserID, 100)
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


//  var mx = map[int]bool{}  

var mx = map[int]time.Time{} // 存储用户上次祈福的日期  



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




func Delete_jdck(sender *Sender, msg chan string, cks []JdCookie) {
	for {
		n, ok := <-msg
		//说明发送方关闭了channel
		if !ok {
			break
		}
		if n == "q" {
			sender.Reply("退出流程")
			ckList[sender.UserID] = nil
			close(msg)
			return
		}
		num, err := strconv.Atoi(n)

		
		if err != nil {
			//sender.Reply(fmt.Sprintf("转换失败:%s", err))
			sender.Reply("请输入数字，检测到非数字输入已退出流程!")
			ckList[sender.UserID] = nil
			return
		}
		regular := `^0$|^[1-9]\d*$`
		reg := regexp.MustCompile(regular)
		if reg.MatchString(n) {
			//cks := GetJdCookie(sender)
			if len(cks) < num {
				sender.Reply("输入序列号错误，已退出！")
				ckList[sender.UserID] = nil
				return
			}
		}
		ck := cks[num]
        ck.Removes(ck.PtPin)
		//db.Model(cks).Where(PtPin+" = ?", ck.PtPin).Delete(cks[num].PtPin)
		sender.Reply(fmt.Sprintf("已删除账号%s", cks[num].Nickname))
		ckList[sender.UserID] = nil
	}
}
