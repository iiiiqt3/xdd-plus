package models

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"fmt"
	"bytes"
	"sync"

	"github.com/beego/beego/v2/client/httplib"
	"github.com/beego/beego/v2/core/logs"
	"github.com/buger/jsonparser"
	"gorm.io/gorm"
)

type UserSession struct {
	apiBackend       string
	account          string
	password         string
	uid              string
	appck            string
	smsCode          string
	smsRetry         int
	isAuto           bool
	isUser           bool
	Socks5_Password  string // 原有字段
	Socks5_Ip        string // 原有字段
	Socks5_Port      string // 原有字段
	Socks5_Account   string // 原有字段
}

type LoginResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    struct {
		Status       int         `json:"status"`
		Mode         interface{} `json:"mode"`
		Ck           string      `json:"ck"`
		Rwskey       string      `json:"rwskey"`
		AccessToken  string      `json:"accessToken"`
		RefreshToken string      `json:"refreshToken"`
		Roles        []interface{} `json:"roles"`
		Img          interface{} `json:"img"`
		Username     string      `json:"username"`
		Expires      string      `json:"expires"`
		JmpUrl       string      `json:"jmp_url"`
	} `json:"data"`
}


// 全局信号量：控制接口请求频率（3秒/次）
var (
	rateLimitSemaphore = make(chan struct{}, 1) // 缓冲1个，确保同一时间只有1个请求
)

// 初始化限流：启动一个goroutine，每3秒释放一次信号量
func initRateLimit() {
	go func() {
		for {
			select {
			case rateLimitSemaphore <- struct{}{}: // 释放信号（允许下一个请求）
			default:
			}
			time.Sleep(4 * time.Second) // 每3秒释放一次，确保间隔≥3秒
		}
	}()
}

// 初始化函数：程序启动时执行（确保限流机制生效）
func init() {
	initRateLimit()
}



// 解析Socks5代理列表
func parseSocks5List() []map[string]string {
	var proxyList []map[string]string
	socks5Str := GetEnv("SOCKS5_PROXY_LIST")
	if socks5Str == "" {
		return proxyList
	}

	proxies := strings.Split(socks5Str, ",")
	for _, proxy := range proxies {
		proxy = strings.TrimSpace(proxy)
		if proxy == "" {
			continue
		}
		parts := strings.Split(proxy, "|")
		if len(parts) != 4 {
			logs.Warn("无效的Socks5配置格式：%s，正确格式：ip|port|account|password", proxy)
			continue
		}
		proxyList = append(proxyList, map[string]string{
			"ip":       parts[0],
			"port":     parts[1],
			"account":  parts[2],
			"password": parts[3],
		})
	}
	return proxyList
}

// 统计数据库中各Socks5 IP的使用次数
func countSocks5Usage() map[string]int {
	usageMap := make(map[string]int)
	
	// 查询所有已配置Socks5的有效账号
	var cks []JdCookie
	err := db.Where("Socks5_Ip != '' AND Socks5_Port != '' AND Available = ?", True).Find(&cks).Error
	if err != nil {
		logs.Warn("统计Socks5使用次数失败：%v", err)
		return usageMap
	}
	
	// 统计每个IP的使用次数
	for _, ck := range cks {
		usageMap[ck.Socks5_Ip]++
	}
	
	return usageMap
}

// 智能选择Socks5代理（优先选择数据库中使用次数最少的IP）
func smartSelectSocks5() (map[string]string, error) {
	// 1. 获取环境变量中的所有代理列表
	proxyList := parseSocks5List()
	if len(proxyList) == 0 {
		return nil, fmt.Errorf("未配置Socks5代理列表")
	}
	
	// 2. 统计数据库中各IP的使用次数
	usageMap := countSocks5Usage()
	logs.Info("Socks5 IP使用统计：%v", usageMap)
	
	// 3. 对代理列表按使用次数排序（最少使用的在前）
	sort.Slice(proxyList, func(i, j int) bool {
		countI := usageMap[proxyList[i]["ip"]]
		countJ := usageMap[proxyList[j]["ip"]]
		// 先按使用次数升序排列
		if countI != countJ {
			return countI < countJ
		}
		// 次数相同则按IP字典序排列（保证排序稳定）
		return proxyList[i]["ip"] < proxyList[j]["ip"]
	})
	
	// 4. 提取使用次数最少的所有代理
	minCount := usageMap[proxyList[0]["ip"]]
	var candidateProxies []map[string]string
	for _, proxy := range proxyList {
		if usageMap[proxy["ip"]] == minCount {
			candidateProxies = append(candidateProxies, proxy)
		} else {
			break // 排序后后面的次数都大于等于当前，可提前退出
		}
	}
	
	// 5. 在候选代理中随机选择一个（保证同次数IP的负载均衡）
	rand.Seed(time.Now().UnixNano())
	selected := candidateProxies[rand.Intn(len(candidateProxies))]
	
	logs.Info("智能选择Socks5代理 - IP：%s，当前使用次数：%d，候选列表长度：%d",
		selected["ip"], minCount, len(candidateProxies))
	
	return selected, nil
}

// 根据PtPin获取对应的Socks5配置
func getSocks5ByPtPin(ptPin string) (map[string]string, error) {
	var ck JdCookie
	err := db.Where("PtPin = ?", ptPin).First(&ck).Error
	if err != nil {
		return nil, err
	}

	if ck.Socks5_Ip == "" || ck.Socks5_Port == "" {
		return nil, fmt.Errorf("该账号未配置Socks5代理")
	}

	return map[string]string{
		"ip":       ck.Socks5_Ip,
		"port":     ck.Socks5_Port,
		"account":  ck.Socks5_Account,
		"password": ck.Socks5_Password,
	}, nil
}

// Autojdck 开始流程，提示用户输入手机号
func Autojdck(sender *Sender, Auto *UserSession) {
	Auto.apiBackend = GetEnv("apiBackend2")
	if Auto.apiBackend == "" {
		sender.Reply("密码登录维护中。。。。请使用短信登录，指令：【短信登录】")
		return
	}
	Auto.smsRetry = 0
	Auto.isAuto = false
	Auto.isUser = true
	sender.Reply("\n1、上车后请到京东-我的-支付设置，关闭小额免密，同时开启虚拟资产验密\n2、回复【QQ群】查看Q群和QQ机器人具体信息")

	// 获取用户名下的账号列表
	cks := GetJdCookies(func(sb *gorm.DB) *gorm.DB {
		return sb.Where("QQ = ? AND Password != ?", sender.UserID, "")
	})

	// 如果用户名下有账号
	if len(cks) > 0 {
		msgs := []string{
			"请回复下面【】里面序号进行密码快捷登录，0为新增账号：\n--------------------------------",
			"【0】新增账号",
		}

		// 检测并标记账号状态
		var validCount int
		for i, ck := range cks {
			status := "❌无效"
			if CookieOK(&ck) {
				status = "✅有效"
				validCount++
			}
			msgs = append(msgs, fmt.Sprintf("【%d】%s %s", i+1, ck.Nickname, status))
		}

		// 显示有效账号统计
		msgs = append(msgs, fmt.Sprintf("--------------------------------\n有效账号: %d/%d", validCount, len(cks)))

		sender.Reply(strings.Join(msgs, "\n"))

		c2 := make(chan string)
		smsList[sender.UserID] = c2

		// 创建超时计时器
		timeout := time.NewTimer(30 * time.Second)

		// 处理账号选择
		go func() {
			defer timeout.Stop() // 确保计时器被停止

			for {
				select {
				case n, ok := <-c2:
					if !ok {
						return
					}

					// 重置计时器
					if !timeout.Stop() {
						<-timeout.C
					}
					timeout.Reset(30 * time.Second)

					if n == "q" {
						sender.Reply("退出登录流程")
						smsList[sender.UserID] = nil
						return
					}

					// 解析用户选择的序号
					idx, err := strconv.Atoi(n)
					if err != nil || idx < 0 || idx > len(cks) {
						sender.Reply("输入无效，请重新输入或回复'q'退出")
						continue
					}

					// 0序号为新增账号
					if idx == 0 {
						// 如果是群聊且选择新增账号，则提示加好友
						if sender.Type == "qqg" || sender.Type == "wxg" {
							sender.Reply("新增账号涉及到输入密码，请加好友后私聊机器人进行操作。")
							smsList[sender.UserID] = nil
							return
						}
						
						// 新增：智能选择Socks5代理（优先选择数据库中使用最少的IP）
						socks5, err := smartSelectSocks5()
						if err != nil {
							logs.Warn("选择Socks5代理失败：%v", err)
							sender.Reply("未配置Socks5代理，将使用默认网络环境登录")
						} else {
							// 保存选中的Socks5配置到Auto对象
							Auto.Socks5_Ip = socks5["ip"]
							Auto.Socks5_Port = socks5["port"]
							Auto.Socks5_Account = socks5["account"]
							Auto.Socks5_Password = socks5["password"]
							logs.Info("为新增账号智能选择Socks5代理：%s:%s（当前该IP已使用%d次）", 
								Auto.Socks5_Ip, Auto.Socks5_Port, countSocks5Usage()[Auto.Socks5_Ip])
						}
						
						sender.Reply("请输入京东账号绑定的手机号或者京东用户名：")
						accountInput(sender, c2, Auto)
						return
					}

					// 用户选择了已有账号
					selectedCk := cks[idx-1]
					Auto.account = selectedCk.Account
					Auto.password = selectedCk.Password
					// 读取选中账号的Socks5信息
					Auto.Socks5_Ip = selectedCk.Socks5_Ip
					Auto.Socks5_Port = selectedCk.Socks5_Port
					Auto.Socks5_Account = selectedCk.Socks5_Account
					Auto.Socks5_Password = selectedCk.Socks5_Password

					// ========== 新增：检查Socks5配置，为空则自动随机选择 ==========
					if Auto.Socks5_Ip == "" || Auto.Socks5_Port == "" {
						logs.Info("选中的账号[%s（%s）]未配置Socks5，开始自动随机选择", selectedCk.Nickname, selectedCk.Account)
						socks5, err := smartSelectSocks5()
						if err != nil {
							logs.Warn("为账号[%s]自动选择Socks5失败：%v，将使用默认网络环境", selectedCk.Nickname, err)
						} else {
							// 赋值随机选中的Socks5配置
							Auto.Socks5_Ip = socks5["ip"]
							Auto.Socks5_Port = socks5["port"]
							Auto.Socks5_Account = socks5["account"]
							Auto.Socks5_Password = socks5["password"]
							logs.Info("为账号[%s]自动选择Socks5代理：%s:%s（当前该IP已使用%d次）", 
								selectedCk.Nickname, Auto.Socks5_Ip, Auto.Socks5_Port, countSocks5Usage()[Auto.Socks5_Ip])
							
							// 可选：将选中的Socks5更新到数据库，下次登录直接复用（推荐开启）
							err = db.Model(&selectedCk).Updates(map[string]interface{}{
								"Socks5_Ip":        socks5["ip"],
								"Socks5_Port":      socks5["port"],
								"Socks5_Account":   socks5["account"],
								"Socks5_Password":  socks5["password"],
							}).Error
							if err != nil {
								logs.Warn("更新账号[%s]的Socks5配置到数据库失败：%v", selectedCk.Nickname, err)
							} else {
								logs.Info("账号[%s]的Socks5配置已更新到数据库，下次登录可直接复用", selectedCk.Nickname)
							}
						}
					}
					// ========== 新增结束 ==========

					sender.Reply("登录中，请稍等，请勿重复操作...")

					// 检查 Cookie 是否有效
					if !CookieOK(&selectedCk) {
					loginAPI(sender, Auto)
					} else {
						sender.Reply("当前账号已登录且有效")
					}
					smsList[sender.UserID] = nil
					return

				case <-timeout.C:
					sender.Reply("操作超时，已自动退出登录流程")
					smsList[sender.UserID] = nil
					return
				}
			}
		}()
		return
	}

	// 用户名下没有账号，按原有逻辑执行
	// 如果是群聊且没有账号，提示加好友
	if sender.Type == "qqg" || sender.Type == "wxg" {
		sender.Reply("新增账号请加好友后再进行【密码登录】操作，防止信息泄露。")
		return
	}

	// 新增：智能选择Socks5代理（优先选择数据库中使用最少的IP）
	socks5, err := smartSelectSocks5()
	if err != nil {
		logs.Warn("选择Socks5代理失败：%v", err)
		sender.Reply("未配置Socks5代理，将使用默认网络环境登录")
	} else {
		// 保存选中的Socks5配置到Auto对象
		Auto.Socks5_Ip = socks5["ip"]
		Auto.Socks5_Port = socks5["port"]
		Auto.Socks5_Account = socks5["account"]
		Auto.Socks5_Password = socks5["password"]
		logs.Info("为新增账号智能选择Socks5代理：%s:%s（当前该IP已使用%d次）", 
			Auto.Socks5_Ip, Auto.Socks5_Port, countSocks5Usage()[Auto.Socks5_Ip])
	}

	sender.Reply("请输入京东账号绑定的手机号或者京东用户名：")
	c2 := make(chan string)
	smsList[sender.UserID] = c2
	accountInput(sender, c2, Auto)
}

// 账号输入流程
func accountInput(sender *Sender, msg chan string, Auto *UserSession) {
	for {
		// 设置定时器
		timeout := time.After(60 * time.Second)
		select {
		case n, ok := <-msg:
			if !ok {
				break
			}

			// 提前检查退出指令 'q'
			if n == "q" {
				sender.Reply("退出登录流程")
				smsList[sender.UserID] = nil
				return
			}

			deal := false

			// 手机号正则表达式
			phoneRegex := `^(13[0-9]|14[01456879]|15[0-35-9]|16[2567]|17[0-8]|18[0-9]|19[0-35-9])\d{8}$`
			phoneReg := regexp.MustCompile(phoneRegex)

			// 用户名正则表达式 (支持字母、数字和下划线)
			usernameRegex := `^[a-zA-Z0-9_]+$`
			usernameReg := regexp.MustCompile(usernameRegex)

			// 判断输入是手机号还是用户名
			if phoneReg.MatchString(n) {
				logs.Info("输入账号阶段 - 手机号")
				Auto.account = n
				deal = true
				// 获取与该账号相关的cookies
				cks := GetJdCookies(func(sb *gorm.DB) *gorm.DB {
					return sb.Where("Account = ?", Auto.account)
				})

				// 检查是否有已存储的密码
				if len(cks) > 0 {
					// 如果密码已存储，检查当前用户ID与数据库中的QQ号是否一致
					if cks[0].QQ != sender.UserID {
						// 如果用户ID和数据库QQ不一致，则必须输入密码
						sender.Reply("检测到您ID与系统中存在QQ号不一致，请输入密码：")
						// 输入密码的处理函数
						passwordInput(sender, msg, Auto)
						return
					} else {
						// 如果一致，直接使用数据库存储的密码和ip
						Auto.password = cks[0].Password
						Auto.Socks5_Ip = cks[0].Socks5_Ip
						Auto.Socks5_Port = cks[0].Socks5_Port
						Auto.Socks5_Account = cks[0].Socks5_Account
						Auto.Socks5_Password = cks[0].Socks5_Password

						// 注释掉检查 Cookie 是否有效的逻辑
						// 直接使用数据库密码进行登录（原逻辑中 Cookie 无效时的处理）
						if Auto.password != "" {
							sender.Reply("登录中，请稍等，请勿重复操作...")
							loginAPI(sender, Auto)
							smsList[sender.UserID] = nil
							return
						

						// 原 Cookie 有效时的逻辑已注释
						 } else {
						 	// 如果 Cookie 有效，跳过登录步骤
						 	sender.Reply("当前输入的账号有效，无需重新登录。")
						 	smsList[sender.UserID] = nil
						 }
					}
				} else {
					// 如果没有找到对应的 cookies 记录，则要求输入密码
					sender.Reply("请输入密码：")
					passwordInput(sender, msg, Auto)
					return
				}
			} else if usernameReg.MatchString(n) {
				logs.Info("输入账号阶段 - 用户名")
				Auto.account = n
				deal = true
				// 获取与该账号相关的cookies
				cks := GetJdCookies(func(sb *gorm.DB) *gorm.DB {
					return sb.Where("Account = ?", Auto.account)
				})

				// 检查是否有已存储的密码
				if len(cks) > 0 {
					// 如果密码已存储，检查当前用户ID与数据库中的QQ号是否一致
					if cks[0].QQ != sender.UserID {
						// 如果用户ID和数据库QQ不一致，则必须输入密码
						sender.Reply("检测到您ID与系统中绑定的QQ号不一致，请输入密码：")
						// 输入密码的处理函数
						passwordInput(sender, msg, Auto)
						return
					} else {
						// 如果一致，直接使用数据库存储的密码
						Auto.password = cks[0].Password
						Auto.Socks5_Ip = cks[0].Socks5_Ip
						Auto.Socks5_Port = cks[0].Socks5_Port
						Auto.Socks5_Account = cks[0].Socks5_Account
						Auto.Socks5_Password = cks[0].Socks5_Password

						// 注释掉检查 Cookie 是否有效的逻辑
						// 直接使用数据库密码进行登录（原逻辑中 Cookie 无效时的处理）
						if Auto.password != "" {
							sender.Reply("登录中，请稍等，请勿重复操作...")
							loginAPI(sender, Auto)
							smsList[sender.UserID] = nil
							return
						}

						// 原 Cookie 有效时的逻辑已注释
						return
					}
				} else {
					// 如果没有找到对应的 cookies 记录，则要求输入密码
					sender.Reply("请输入密码：")
					passwordInput(sender, msg, Auto)
					return
				}
			}

			// 如果输入既不是手机号也不是用户名，提示输入格式错误
			if !deal {
				sender.Reply("账号输入格式错误，请重新输入，或回复‘q’退出流程")
			}
		case <-timeout:
			// 超时处理
			sender.Reply("操作超时，退出登录流程！")
			smsList[sender.UserID] = nil
			return
		}
	}
}

// 密码输入流程
func passwordInput(sender *Sender, msg chan string, Auto *UserSession) {
	for {
		n, ok := <-msg
		if !ok {
			break
		}

		if n != "" && n != "q" {
			logs.Info("输入密码阶段")
			Auto.password = n
			sender.Reply("登录中，请稍等，请勿重复操作...")
			flag := loginAPI(sender, Auto)
			if flag {
				smsList[sender.UserID] = nil
				return
			}
		}
		if n == "q" {
			sender.Reply("退出登录流程")
			smsList[sender.UserID] = nil
			return
		}
	}
}

func loginAPI(sender *Sender, Auto *UserSession) bool {
	value := GetEnv("password_login")
	if value == "大师" {
		Ds_loginAPI(sender, Auto)
		return false
	} else if value == "兔子" {
		// 调用兔子登录，如果返回需要切换，则使用pro登录
		if Rabbit_loginAPI(sender, Auto) {
			// 兔子登录需要切换到pro
			sender.Reply("检测到当前登录IP被限制，自动切换到线程2登录...")
			// 直接使用pro登录逻辑
			goto ProLogin
		}
		return false
	} else if value == "BBK" {
		// BBK登录方式
		if BBK_loginAPI(sender, Auto) {
			return true
		}
		return false
	}

ProLogin:
	logs.Info("pro登录阶段")
	requesturl := GetEnv("apiBackend2_fzjh")

	// 构建请求参数
	params := map[string]interface{}{
		"username":   Auto.account,
		"password":   Auto.password,
		"isAuto":     Auto.isAuto,
		"BotApitoken": GetEnv("NolanToken"),
	}

	// 将参数转换为 JSON 字节切片
	jsonData, err := json.Marshal(params)
	if err != nil {
		log.Printf("JSON 编码出错: %v\n", err)
		return false
	}

	// 创建 HTTP 请求
	req, err := http.NewRequest("POST", requesturl, bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("创建请求出错: %v\n", err)
		return false
	}
	req.Header.Set("Content-Type", "application/json")

	// 发送 HTTP 请求
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("发送请求出错: %v\n", err)
		return false
	}
	defer resp.Body.Close()

	// 读取响应体
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("读取响应体出错: %v\n", err)
		return false
	}

	// 解析响应 JSON 数据
	var loginResp LoginResponse
	err = json.Unmarshal(body, &loginResp)
	if err != nil {
		log.Printf("解析响应 JSON 出错: %v\n", err)
		return false
	}

	// 新增的判断，如果 message 是特定的风险提示
	if loginResp.Data.Status == 555 && !loginResp.Success {
		message := loginResp.Message

		// 否则按照原有流程处理
		jumpURL := loginResp.Data.JmpUrl
		log.Printf("Status: %d\n", loginResp.Data.Status)
		log.Printf("认证提示: %s\n", message)
		log.Printf("跳转链接: %s\n", jumpURL)
		logs.Info("Satatus: %d\n", loginResp.Data.Status)
		logs.Info("认证提示: %s\n", message)
		logs.Info("跳转链接: %s\n", jumpURL)

		sender.Reply(jumpURL)
		time.Sleep(1 * time.Second)
		sender.Reply("请在200秒内点击上面的地址打开进行安全认证，认证后直接回复【y】即可，退出请输入【q】")

		cks := GetJdCookies(func(sb *gorm.DB) *gorm.DB { return sb.Where("Account = ?", Auto.account) })
		if len(cks) > 0 {
			cks[0].Update(Smsverify, "true")
			cks[0].Update(Available, "false")
		}

		// 启动监听用户输入的过程
		go func() {
			if smsList[sender.UserID] == nil {
				smsList[sender.UserID] = make(chan string)
			}
			defer delete(smsList, sender.UserID) // 删除消息通道
			for {
				select {
				case msg := <-smsList[sender.UserID]:
					if msg == "q" || msg == "Q" {
						sender.Reply("退出手动认证流程")
						smsList[sender.UserID] = nil
						return
					}

					if msg == "y" || msg == "Y" {
						sender.Reply("开始登录...")
						loginAPI(sender, Auto)
						return
					}

					sender.Reply("无效输入，请回复'y'继续登录，或回复'q'退出。")
				case <-time.After(200 * time.Second):
					sender.Reply("操作超时，退出手动认证流程")
					smsList[sender.UserID] = nil
					return
				}
			}
		}()

		return true
	}

	// 继续按照原有逻辑处理其他情况
	if loginResp.Data.Status == 0 && !loginResp.Success {
		message := loginResp.Message
		log.Printf("认证提示: %s\n", message)
		if message == "您的账号存在风险，为了您的账号安全请到京东商城App登录" || message == "登录失败: 您的账号存在风险，为了您的账号安全，打开京东商城APP重新登录，风险解除后即可正常使用" {
			// 特定风险提示处理
			sender.Reply("请到京东官方app登录一下，过下人脸识别，然后再次使用机器人密码登录即可")

			cks := GetJdCookies(func(sb *gorm.DB) *gorm.DB { return sb.Where("Account = ?", Auto.account) })
			if len(cks) > 0 {
				cks[0].Update(Smsverify, "true")
				cks[0].Update(Available, "false")
				// 跳过更新操作，不执行清空 Password 和 Account
			}
			return true
		} else if strings.Contains(message, "账号或密码不正确") {
			sender.Reply("请确认自己的账号密码是否正确，或修改后在登陆")
			cks := GetJdCookies(func(sb *gorm.DB) *gorm.DB { return sb.Where("Account = ?", Auto.account) })
			if len(cks) > 0 {
				cks[0].Update("Password", "")
				cks[0].Update("Account", "")
			}
			return true
		} else {
			// 其他 Status=0 且 Success=false 的情况
			sender.Reply(message)
			return true
		}
	} else if loginResp.Data.Status == 0 && loginResp.Success {
		// 如果 Status 是 0 且 Success 是 true
		ck := loginResp.Data.Ck
		log.Printf("提取到的 ck 值为: %s\n", ck)
		Auto.appck = ck
		// 修正：添加对 Autockup 的调用
		Autockup(Auto.appck, sender, Auto)
		return true
	}
	return true
}













// BBK登录API实现
func BBK_loginAPI(sender *Sender, Auto *UserSession) bool {
	logs.Info("=== BBK登录开始 ===\n当前登录账号：%s\n当前登录密码：%s\nSocks5配置 - IP：%s，端口：%s，账号：%s，密码：%s",
		Auto.account, Auto.password, Auto.Socks5_Ip, Auto.Socks5_Port, Auto.Socks5_Account, Auto.Socks5_Password)
	logs.Info("BBK登录接口地址：%s", "http://111.229.133.91:10066/wangjing/dsLogin")

	retryCount := 0
	maxRetry := 3
	// 新增：速度过快的独立重试计数器
	fastRetryCount := 0
	maxFastRetry := 3
	// 新增：代理过期重试计数器
	proxyExpiredRetryCount := 0
	maxProxyExpiredRetry := 2 // 代理过期最多重试2次

retryLogin:
	retryCount++
	if retryCount > maxRetry {
		sender.Reply("多次尝试切换代理仍登录失败，请稍后再试（可能当前接口整体不稳定）")
		// 修复：字符串换行+函数参数括号缺失问题，单引号包裹字符串，参数正确传入
		logs.Info("=== BBK登录结束（重试次数耗尽，最多重试%d次）===", maxRetry)
		return true
	}

	socks5Str := ""
	if Auto.Socks5_Ip != "" && Auto.Socks5_Port != "" {
		socks5Str = fmt.Sprintf("%s|%s|%s|%s",
			Auto.Socks5_Ip,
			Auto.Socks5_Port,
			Auto.Socks5_Account,
			Auto.Socks5_Password)
		logs.Info("Socks5拼接后字符串：%s", socks5Str)
	} else {
		logs.Info("未配置Socks5代理，socks5参数为空字符串")
		socks5Str = ""
	}

	params := struct {
		Account  string `json:"account"`
		Password string `json:"password"`
		Socks5   string `json:"socks5"`
	}{
		Account:  Auto.account,
		Password: Auto.password,
		Socks5:   socks5Str,
	}
	logs.Info("BBK登录请求参数：%+v", params)

	jsonData, err := json.Marshal(params)
	if err != nil {
		log.Printf("【步骤1：JSON编码失败】BBK登录JSON编码出错: %v", err)
		sender.Reply("登录请求处理失败（参数格式错误）")
		logs.Info("=== BBK登录结束（JSON编码失败）===")
		return true
	}
	logs.Info("BBK登录请求JSON数据：%s", string(jsonData))

	req, err := http.NewRequest("POST", "http://111.229.133.91:10066/wangjing/dsLogin", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("【步骤2：创建请求失败】BBK登录创建请求出错: %v", err)
		sender.Reply("登录请求发送失败")
		logs.Info("=== BBK登录结束（创建请求失败）===")
		return true
	}

	req.Header.Set("Content-Type", "application/json;charset=utf-8")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	logs.Info("BBK登录请求头：Content-Type=%s，User-Agent=%s",
		req.Header.Get("Content-Type"), req.Header.Get("User-Agent"))

	logs.Info("开始发送BBK登录请求（第%d次尝试），超时时间：60秒", retryCount)
	client := &http.Client{
		Timeout: 60 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("【步骤3：发送请求失败】BBK登录接口调用失败: %v", err)
		sender.Reply(fmt.Sprintf("登录连接超时（第%d次尝试失败）", retryCount))
		logs.Info("=== BBK登录结束（发送请求失败）===")
		return true
	}
	defer resp.Body.Close()

	logs.Info("【步骤4：接收响应】BBK登录接口响应状态码：%d，状态：%s", resp.StatusCode, resp.Status)
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("【步骤5：读取响应失败】BBK登录读取响应出错: %v", err)
		sender.Reply("登录响应处理失败")
		logs.Info("=== BBK登录结束（读取响应失败）===")
		return true
	}
	logs.Info("BBK登录接口响应原始内容：%s（长度：%d字节）", string(body), len(body))

	var BBKResp struct {
		Code  int    `json:"code"`
		Msg   string `json:"msg"`
		Data  string `json:"data"`
		Ck    string `json:"ck"`
		Wskey string `json:"wskey"` // 接收接口返回的wskey
	}
	err = json.Unmarshal(body, &BBKResp)
	if err != nil {
		log.Printf("【步骤6：JSON解析失败】BBK登录响应解析失败: %v，原始响应：%s", err, string(body))
		sender.Reply(fmt.Sprintf("登录失败：%s", string(body)))
		logs.Info("=== BBK登录结束（JSON解析失败）===")
		return true
	}

	logs.Info("【步骤7：处理响应】BBK登录响应 - Code：%d，Msg：%s，Data：%s，CK：%s，Wskey：%s",
		BBKResp.Code, BBKResp.Msg, BBKResp.Data, BBKResp.Ck, BBKResp.Wskey)
	switch {
	case BBKResp.Code == 0 && BBKResp.Wskey != "":
		log.Printf("【步骤8：登录成功，获取到Wskey】Wskey: %s", BBKResp.Wskey)

		// --- 核心：保存Wskey到数据库字段 ---
		cks := GetJdCookies(func(sb *gorm.DB) *gorm.DB { return sb.Where("Account = ?", Auto.account) })
		if len(cks) > 0 {
			// 更新数据库中该账号的Wskey字段（直接赋值原始wskey，无需编码）
			cks[0].Update("Wskey", BBKResp.Wskey)
			logs.Info("【步骤8-0：Wskey数据库保存成功】账号：%s，Wskey：%s", Auto.account, BBKResp.Wskey)
		} else {
			logs.Warn("【步骤8-0：Wskey数据库保存失败】未找到账号%s对应的数据库记录", Auto.account)
		}

		// --- Wskey 转换为 CK 逻辑 ---
		// 1. 对 wskey 进行 URL 编码
		encodedWskey := url.QueryEscape(BBKResp.Wskey)
		convertURL := fmt.Sprintf("http://m.jing521.cn:10066/wangjing/newWskey?key=%s", encodedWskey)
		logs.Info("【步骤8-1：调用转换接口】转换接口地址：%s", convertURL)

		// 2. 调用转换接口
		convertResp, err := client.Get(convertURL)
		if err != nil {
			log.Printf("【步骤8-2：转换接口调用失败】%v", err)
			sender.Reply("登录成功，但Wskey转换失败（已保存Wskey到数据库）")
			logs.Info("=== BBK登录结束（Wskey转换失败，Wskey已保存）===")
			return true
		}
		defer convertResp.Body.Close()

		// 3. 读取转换接口响应
		convertBody, err := ioutil.ReadAll(convertResp.Body)
		if err != nil {
			log.Printf("【步骤8-3：读取转换响应失败】%v", err)
			sender.Reply("登录成功，但转换结果读取失败（已保存Wskey到数据库）")
			logs.Info("=== BBK登录结束（转换结果读取失败，Wskey已保存）===")
			return true
		}
		logs.Info("【步骤8-4：转换接口返回内容】%s", string(convertBody))

		// 4. 解析转换后的CK（响应是 `pt_pin=xxx; pt_key=xxx;` 格式）
		convertedCk := string(convertBody)
		if convertedCk == "" {
			log.Printf("【步骤8-5：转换结果为空】")
			sender.Reply("登录成功，但未获取到有效的CK（已保存Wskey到数据库）")
			logs.Info("=== BBK登录结束（转换结果为空，Wskey已保存）===")
			return true
		}

		// 5. 更新 Auto 的 CK 并调用原有函数上传
		Auto.appck = convertedCk
		Autockup(Auto.appck, sender, Auto)
		// --- 转换逻辑结束 ---

		accountName := strings.TrimPrefix(strings.TrimSuffix(BBKResp.Msg, "]登录成功"), "[")
		if accountName == BBKResp.Msg {
			accountName = Auto.account
		}
		sender.Reply(fmt.Sprintf("登录成功！账号：%s", accountName))
		logs.Info("=== BBK登录结束（登录成功、Wskey保存成功、CK转换完成）===")
		return true

	case BBKResp.Code == 0 && BBKResp.Wskey == "":
		log.Printf("【步骤8：登录成功但无Wskey】BBK登录成功但未返回Wskey")
		sender.Reply("登录成功但未获取到Wskey（无数据可保存）")
		logs.Info("=== BBK登录结束（登录成功无Wskey）===")
		return true

	case BBKResp.Code == 128:
		log.Printf("【步骤8：需要验证】BBK登录需要验证，原始链接：%s", BBKResp.Data)
		// 原始验证链接使用：无需编码、无需拼接前缀，直接返回
		jumpURL := BBKResp.Data
		// 【备用代码】需要编码+拼接前缀时启用（注释掉上方，解开下方）
		
		sender.Reply(jumpURL)
		time.Sleep(1 * time.Second)
		sender.Reply("200秒内打开地址验证，如果出现客户端不支持或反复需要认证，建议复制链接到京东APP客服服务聊天界面认证，后直接回复【y】即可，退出请输入【q】")

		cks := GetJdCookies(func(sb *gorm.DB) *gorm.DB { return sb.Where("Account = ?", Auto.account) })
		if len(cks) > 0 {
			cks[0].Update(Smsverify, "true")
			cks[0].Update(Available, "false")
		}

		// 启动监听用户输入的过程
		go func() {
			if smsList[sender.UserID] == nil {
				smsList[sender.UserID] = make(chan string)
			}
			defer delete(smsList, sender.UserID) // 删除消息通道
			for {
				select {
				case msg := <-smsList[sender.UserID]:
					if msg == "q" || msg == "Q" {
						sender.Reply("退出手动认证流程")
						smsList[sender.UserID] = nil
						return
					}

					if msg == "y" || msg == "Y" {
						sender.Reply("开始登录...")
						loginAPI(sender, Auto)
						return
					}

					sender.Reply("无效输入，请回复'y'继续登录，或回复'q'退出。")
				case <-time.After(200 * time.Second):
					sender.Reply("操作超时，退出手动认证流程")
					smsList[sender.UserID] = nil
					return
				}
			}
		}()

		return true

	// 500错误码-账号存在风险场景 (原有逻辑)
	case BBKResp.Code == 500 && strings.Contains(BBKResp.Msg, "您的账号存在风险，为了您的账号安全，打开京东商城APP重新登录"):
		log.Printf("【步骤8：登录失败-账号存在风险】%s", BBKResp.Msg)
		sender.Reply("请使用京东官方APP登录解除人脸后再次重试，如果仍然不行，应该是狗东搞事，可回复【app下载】使用app提交")
		logs.Info("=== BBK登录结束（账号存在风险）===")
		return true

	// 【新增】500错误码-强风控账号场景
	case BBKResp.Code == 500 && strings.Contains(BBKResp.Msg, "强风控账号"):
		log.Printf("【步骤8：登录失败-强风控账号】检测到强风控账号：%s, 消息：%s", Auto.account, BBKResp.Msg)
		
		// 1. 查询数据库记录
		cks := GetJdCookies(func(sb *gorm.DB) *gorm.DB { return sb.Where("Account = ?", Auto.account) })
		if len(cks) > 0 {
			// 2. 清空敏感信息：账号、密码、Socks5代理信息
			err := db.Model(&cks[0]).Updates(map[string]interface{}{
			//	"Socks5_Ip":       "",
			//	"Socks5_Port":     "",
			//	"Socks5_Account":  "",
			//	"Socks5_Password": "",
				// 可选：同时也清空Wskey和CK，防止保留无效凭证
				"WsKey":           "",
				"PtKey":              "", 
			}).Error
			
			if err != nil {
				logs.Error("【严重】清空强风控账号[%s]的数据库信息失败：%v", Auto.account, err)
				// 即使更新失败，也继续执行下面的回复逻辑，避免程序卡死
			} else {
				logs.Info("【成功】已清空强风控账号[%s]的后台登录信息及代理配置", Auto.account)
			}
		} else {
			logs.Warn("未找到账号[%s]的数据库记录，无法执行清空操作", Auto.account)
		}

		// 3. 告知用户
		sender.Reply("⚠️ 检测到当前账号为【强风控账号】，登录失败。\n系统已自动清空后台代理(Socks5)信息。\n请先使用【短信登录】。")
		
		logs.Info("=== BBK登录结束（强风控账号，已清空数据）===")
		return true

	case BBKResp.Code == 500 && BBKResp.Msg == "登录失败:undefined":
		logs.Info("【步骤8：触发重试条件】BBK登录返回：code=500 + msg=登录失败:undefined（第%d次重试）", retryCount)
		sender.Reply(fmt.Sprintf("登录临时受限，正在切换代理（第%d次重试）...", retryCount))

		newSocks5, err := smartSelectSocks5()
		if err != nil {
			logs.Warn("重新选择Socks5代理失败：%v", err)
			sender.Reply("切换代理失败，无法继续重试")
			logs.Info("=== BBK登录结束（代理重选失败）===")
			return true
		}

		Auto.Socks5_Ip = newSocks5["ip"]
		Auto.Socks5_Port = newSocks5["port"]
		Auto.Socks5_Account = newSocks5["account"]
		Auto.Socks5_Password = newSocks5["password"]
		logs.Info("重新选择Socks5代理成功：%s:%s（当前该IP已使用%d次）",
			Auto.Socks5_Ip, Auto.Socks5_Port, countSocks5Usage()[Auto.Socks5_Ip])

		goto retryLogin

	// 新增：处理code=500且msg为空的情况（代理过期）
	case BBKResp.Code == 500 && BBKResp.Msg == "":
		proxyExpiredRetryCount++
		if proxyExpiredRetryCount > maxProxyExpiredRetry {
			logs.Info("【步骤8：代理过期重试耗尽】BBK登录返回code=500且msg为空（代理过期），已重试%d次，终止重试", maxProxyExpiredRetry)
			sender.Reply(fmt.Sprintf("当前使用的Socks5代理已过期，更换%d次代理后仍登录失败，请联系管理员更新代理列表", maxProxyExpiredRetry))
			logs.Info("=== BBK登录结束（代理过期重试耗尽）===")
			return true
		}

		logs.Info("【步骤8：代理过期重试】BBK登录返回code=500且msg为空（判定为代理过期），第%d次重试（最多%d次）", proxyExpiredRetryCount, maxProxyExpiredRetry)
		sender.Reply(fmt.Sprintf("检测到当前Socks5代理已过期，正在自动更换代理（第%d次重试）...", proxyExpiredRetryCount))

		// 1. 重新选择新的Socks5代理
		newSocks5, err := smartSelectSocks5()
		if err != nil {
			logs.Warn("重新选择Socks5代理失败：%v", err)
			sender.Reply("更换代理失败，无法继续重试")
			logs.Info("=== BBK登录结束（代理重选失败）===")
			return true
		}

		// 2. 更新Auto对象的代理信息
		oldProxyIP := Auto.Socks5_Ip
		Auto.Socks5_Ip = newSocks5["ip"]
		Auto.Socks5_Port = newSocks5["port"]
		Auto.Socks5_Account = newSocks5["account"]
		Auto.Socks5_Password = newSocks5["password"]
		logs.Info("更换过期代理成功：原IP=%s，新IP=%s:%s（当前该IP已使用%d次）",
			oldProxyIP, Auto.Socks5_Ip, Auto.Socks5_Port, countSocks5Usage()[Auto.Socks5_Ip])

		// 3. 更新数据库中该账号的代理信息
		cks := GetJdCookies(func(sb *gorm.DB) *gorm.DB { return sb.Where("Account = ?", Auto.account) })
		if len(cks) > 0 {
			err = db.Model(&cks[0]).Updates(map[string]interface{}{
				"Socks5_Ip":        newSocks5["ip"],
				"Socks5_Port":      newSocks5["port"],
				"Socks5_Account":   newSocks5["account"],
				"Socks5_Password":  newSocks5["password"],
			}).Error
			if err != nil {
				logs.Warn("更新账号[%s]的过期代理信息失败：%v", Auto.account, err)
			} else {
				logs.Info("已更新账号[%s]的过期代理信息：%s:%s -> %s:%s",
					Auto.account, oldProxyIP, cks[0].Socks5_Port, newSocks5["ip"], newSocks5["port"])
			}
		} else {
			logs.Warn("未找到账号[%s]对应的数据库记录，无法更新代理信息", Auto.account)
		}

		// 4. 等待2秒后重试登录
		time.Sleep(2 * time.Second)
		goto retryLogin

	// Code=-1 速度过快场景
	case BBKResp.Code == -1 && strings.Contains(BBKResp.Msg, "速度过快"):
		fastRetryCount++
		if fastRetryCount > maxFastRetry {
			logs.Info("【步骤8：速度过快重试耗尽】BBK登录返回包含“速度过快”的提示，已重试%d次，终止重试", maxFastRetry)
			sender.Reply(fmt.Sprintf("操作速度过快，已重试%d次仍失败，请稍后重新发起登录", maxFastRetry))
			logs.Info("=== BBK登录结束（速度过快重试耗尽）===")
			return true
		}

		logs.Info("【步骤8：速度过快重试】BBK登录返回：code=-1，msg=%s，第%d次重试（最多%d次）", BBKResp.Msg, fastRetryCount, maxFastRetry)
		sender.Reply(fmt.Sprintf("%s，等待2秒后进行第%d次自动重试（最多%d次）...", BBKResp.Msg, fastRetryCount, maxFastRetry))
		time.Sleep(2 * time.Second)
		goto retryLogin

	default:
		log.Printf("【步骤8：登录失败】BBK登录失败，错误码: %d, 错误信息: %s", BBKResp.Code, BBKResp.Msg)
		sender.Reply(fmt.Sprintf("登录失败: %s（错误码：%d）", BBKResp.Msg, BBKResp.Code))

		if strings.Contains(BBKResp.Msg, "账号或密码不正确") {
			cks := GetJdCookies(func(sb *gorm.DB) *gorm.DB { return sb.Where("Account = ?", Auto.account) })
			if len(cks) > 0 {
				cks[0].Update("Password", "")
				cks[0].Update("Account", "")
				cks[0].Update("Wskey", "") // 清空数据库中的Wskey字段
				logs.Info("账号密码错误，已清空数据库中存储的账号、密码及Wskey")
			}
		}
		logs.Info("=== BBK登录结束（登录失败）===")
		return true
	}
}





































//##以下为兔子密码登录

func RabbitInit(account string) bool {
	logs.Info("账密初始化")
	req := httplib.Post(fmt.Sprintf("%s/bot/pwd/init?BotApiToken=%s", sysConfig.RabbitUrl, sysConfig.RabbitApiToken))
	req.Header("content-type", "application/json; charset=utf-8")
	data, _ := req.Body(`{"account":"` + account + `"}`).Bytes()

	success, _ := jsonparser.GetBoolean(data, "success")
	code, _ := jsonparser.GetInt(data, "code")
	if success {
		return true
	}
	if code == 505 {
		return false
	}

	retry666 := 0
	totalAttempts := 1

	for {
		totalAttempts++
		req := httplib.Post(fmt.Sprintf("%s/bot/pwd/auto_captcha?BotApiToken=%s", sysConfig.RabbitUrl, sysConfig.RabbitApiToken))
		req.Header("content-type", "application/json; charset=utf-8")
		data, _ := req.Body(`{"account":"` + account + `"}`).Bytes()

		success, _ := jsonparser.GetBoolean(data, "success")
		code, _ := jsonparser.GetInt(data, "code")

		if success {
			return true
		}
		if code == 666 {
			retry666++
			if retry666 > 3 {
				return false
			}
			continue
		}
		if code == 505 || totalAttempts >= 6 {
			return false
		}
		return false
	}
}

func encryptPwdAesGcm(pwd, account string) (string, error) {
	md5Bytes := sha512.Sum512([]byte("#(*():dfgjn^%&89$%#" + account + "#(*():dfgjn^%&89$%#"))
	key := md5Bytes[:32]
	nonce := getRandomBytes(12)
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("创建AES加密块失败: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("创建GCM失败: %w", err)
	}
	paddedPwd := append(append(getRandomBytes(16), []byte(pwd)...), getRandomBytes(16)...)
	ciphertext := aead.Seal(nil, nonce, paddedPwd, nil)
	ciplen := len(ciphertext) - 16
	cipherpwd := ciphertext[:ciplen]
	tag := ciphertext[ciplen:]
	encryptPwd := base64.StdEncoding.EncodeToString(append(append(tag, cipherpwd...), nonce...))
	return encryptPwd, nil
}

func getRandomBytes(n int) []byte {
	b := make([]byte, n)
	rand.Read(b)
	return b
}

func riskSend(account string) bool {
	logs.Info("发送风控验证码...")

	// 第一次请求 risk/risk_send
	var req *httplib.BeegoHTTPRequest
	req = httplib.Post(fmt.Sprintf("%s/bot/risk/risk_send?BotApiToken=%s", sysConfig.RabbitUrl, sysConfig.RabbitApiToken))
	req.Header("content-type", "application/json; charset=utf-8")
	data, err := req.Body(`{"account":"` + account + `"}`).Bytes()

	if err != nil {
		logs.Error("请求 risk/risk_send 出错: %v", err)
		return false
	}

	// 打印响应体
	logs.Info("接品请求 risk/risk_send 响应数据: %s", string(data))

	// 解析响应数据
	success, _ := jsonparser.GetBoolean(data, "success")
	code, _ := jsonparser.GetInt(data, "code")
	if success {
		return true
	}
	if code == 505 {
		return false
	}

	// 循环请求 risk/risk_auto_captcha
	i := 1
	for {
		i++

		var req *httplib.BeegoHTTPRequest
		req = httplib.Post(fmt.Sprintf("%s/bot/risk/risk_auto_captcha?BotApiToken=%s", sysConfig.RabbitUrl, sysConfig.RabbitApiToken))
		req.Header("content-type", "application/json; charset=utf-8")
		data, err := req.Body(`{"account":"` + account + `"}`).Bytes()

		if err != nil {
			logs.Error("请求 risk/risk_auto_captcha 出错: %v", err)
			return false
		}

		// 打印响应体
		logs.Info("接品请求 risk/risk_auto_captcha 响应数据: %s", string(data))

		// 解析响应数据
		success, _ := jsonparser.GetBoolean(data, "success")
		code, _ := jsonparser.GetInt(data, "code")
		if success {
			return true
		}
		if code == 666 {
			time.Sleep(2 * time.Second)
			continue
		}
		if code == 505 || i == 6 {
			time.Sleep(2 * time.Second)
			return false
		}

		return false
	}
}

// 风控验证码验证
func riskVerifyCode(sender *Sender, Auto *UserSession, code string) bool {
	logs.Info("验证风控验证码")

	// 构造请求体
	requestBody := fmt.Sprintf(`{"account":"%s","code":"%s"}`, Auto.account, code)
	logs.Info("请求体: %s", requestBody)

	// 构造 HTTP 请求
	var req *httplib.BeegoHTTPRequest
	req = httplib.Post(fmt.Sprintf("%s/bot/risk/risk_verify_code?BotApiToken=%s", sysConfig.RabbitUrl, sysConfig.RabbitApiToken))
	req.Header("content-type", "application/json; charset=utf-8")

	// 发送请求并获取响应
	data, err := req.Body(requestBody).Bytes()
	if err != nil {
		logs.Error("请求 risk_verify_code 出错: %v", err)
		return false
	}

	// 打印响应体
	logs.Info("响应体: %s", string(data))

	// 解析响应数据
	success, _ := jsonparser.GetBoolean(data, "success")
	message, _ := jsonparser.GetString(data, "message")
	var statusCode int64
	statusCode, _ = jsonparser.GetInt(data, "code")

	if success {
		ck, _ := jsonparser.GetString(data, "ck")
		//pin, _ := jsonparser.GetString(data, "pin")
		if ck != "" {
			Auto.appck = ck
		}
		Autockup(Auto.appck, sender, Auto)
		return true
	}

	// 回复错误消息
	sender.Reply(fmt.Sprintf("错误消息: %s, 错误代码: %d", message, statusCode))
	return success
}

// 调用兔子api登录，返回是否需要切换到pro登录
func Rabbit_loginAPI(sender *Sender, Auto *UserSession) bool {
	logs.Info("调用兔子账密登录阶段")
	init := RabbitInit(Auto.account)
	if !init {
		sender.Reply("初始化失败，请稍后重试")
		smsList[sender.UserID] = nil
		return false
	}

	var req *httplib.BeegoHTTPRequest
	req = httplib.Post(fmt.Sprintf("%s/bot/pwd/login?BotApiToken=%s", sysConfig.RabbitUrl, sysConfig.RabbitApiToken))
	req.Header("content-type", "application/json; charset=utf-8")

	gcm, err := encryptPwdAesGcm(Auto.password, Auto.account)
	if err != nil {
		sender.Reply(fmt.Sprintf("加密密码时发生错误: %v", err))
		smsList[sender.UserID] = nil
		return false
	}

	data, _ := req.Body(fmt.Sprintf(`{"account":"%s","pwd":"%s"}`, Auto.account, gcm)).Bytes()
	success, _ := jsonparser.GetBoolean(data, "success")
	message, _ := jsonparser.GetString(data, "message")

	if success {
		// 提取 ck 和 pin
		ck, _ := jsonparser.GetString(data, "ck")
		//pin, _ := jsonparser.GetString(data, "pin")
		if ck != "" {
			Auto.appck = ck
		}
		Autockup(Auto.appck, sender, Auto)
		smsList[sender.UserID] = nil
		return false
	}

	code, _ := jsonparser.GetInt(data, "code")
	RiskUrl, _ := jsonparser.GetString(data, "RiskUrl")

	// 新增：处理操作过于频繁的情况
	if strings.Contains(message, "操作过于频繁") && strings.Contains(message, "24小时后再试") {
		sender.Reply("群主服务器IP已黑，请等待恢复，安卓用户回复【app下载】复制链接到浏览器打开，使用app提交和查询，或者直接使用【短信登录】临时过渡")
		smsList[sender.UserID] = nil
		// 返回true表示需要切换到pro登录
		return true
	}

	if strings.Contains(message, "账号或密码不正确") {
		cks := GetJdCookies(func(sb *gorm.DB) *gorm.DB {
			return sb.Where("Account = ?", Auto.account)
		})
		if len(cks) > 0 {
			cks[0].Update("Password", "")
			cks[0].Update("Account", "")
		}
		sender.Reply("登录失败，请检测账号或密码是否正确，请重新登录！")
		smsList[sender.UserID] = nil
		return false
	}

	if code == 555 {
		sender.Reply(message)
		sender.Reply(RiskUrl)
		sender.Reply("验证后请重新登录")
		cks := GetJdCookies(func(sb *gorm.DB) *gorm.DB { return sb.Where("Account = ?", Auto.account) })
		if len(cks) > 0 {
			cks[0].Update(Smsverify, "true")
			cks[0].Update(Available, "false")
		}
		smsList[sender.UserID] = nil
		return false
	}

	if code == 601 || code == 602 {
		cks := GetJdCookies(func(sb *gorm.DB) *gorm.DB { return sb.Where("Account = ?", Auto.account) })
		if len(cks) > 0 {
			cks[0].Update(Smsverify, "true")
			cks[0].Update(Available, "false")
		}
		if !Auto.isAuto {
			// 调用 riskSend 函数
			RabbiriskSend := riskSend(Auto.account)
			if !RabbiriskSend {
				sender.Reply("验证码发送失败，尝试自动验证")
				smsList[sender.UserID] = nil
				return false
			}
			sender.Reply("您的账号需要验证，机器人将自动请求下发验证码，请查看短信/接听语音获取验证码，然后输入验证码：\n\n如果没有收到验证码，请回复's'打印手动验证地址，点击链接验证。")
			for {
				select {
				case code := <-smsList[sender.UserID]:
					if code == "q" {
						sender.Reply("退出风控验证流程")
						smsList[sender.UserID] = nil
						return false
					}

					if code == "s" || code == "S" {
						sender.Reply(fmt.Sprintf("手动验证地址: %s", RiskUrl))
						time.Sleep(3 * time.Second)
						sender.Reply("手动验证完成后回复'y'。")

						continue
					}
					if code == "y" || code == "Y" {
						sender.Reply("请耐心等待，长时间无响应请重新发送【密码登录】即可。")
						loginAPI(sender, Auto)
						return false
					}
					if len(code) == 6 && regexp.MustCompile(`^\d{6}$`).MatchString(code) {
						smsList[sender.UserID] = nil
						if riskVerifyCode(sender, Auto, code) {
							return false
						} else {
							sender.Reply("验证码验证失败，退出流程")
							smsList[sender.UserID] = nil
							return false
						}
					} else {
						sender.Reply("无效的验证码，请重新输入6位数字，或回复'q'退出流程。")
						continue
					}
				case <-time.After(200 * time.Second):
					sender.Reply("操作超时，退出流程")
					smsList[sender.UserID] = nil
					return false
				}
			}
		}
		// 如果 Auto.isAuto 为 true，流程走到这里，需要返回一个值
		return false
	} else {
		//#   if message == "您的账号存在风险，为了您的账号安全请到京东商城App登录" ||  message == "登录失败: 您的账号存在风险，为了您的账号安全，打开京东商城APP重新登录，风险解除后即可正常使用" {
		if strings.Contains(message, "您的账号存在风险") {
			// 更新状态
			cks := GetJdCookies(func(sb *gorm.DB) *gorm.DB {
				return sb.Where("Account = ?", Auto.account)
			})
			if len(cks) > 0 {
				cks[0].Update(Smsverify, "true")
				cks[0].Update(Available, "false")
			}

			sender.Reply("请到京东官方app登录一下，过下人脸识别，然后再次使用机器人密码登录即可")
		} else {
			// 不更新状态，只回复登录失败信息
			sender.Reply(fmt.Sprintf("登录失败: %s", message))
		}
		smsList[sender.UserID] = nil
		return false
	}
}

// 调用大师api登录
func Ds_loginAPI(sender *Sender, Auto *UserSession) bool {
	logs.Info("登录阶段")
	requesturl := GetEnv("apiBackend1") + "/api/encrypt"

	// 构建请求参数
	params := map[string]interface{}{
		"phone":  Auto.account,
		"pwd":    Auto.password, // 使用编码后的密码
		"isAuto": Auto.isAuto,
		"token":  "194482018e854ee1a6cfc9ecf6beb76d",
	}

	jsonData, err := json.Marshal(params)
	if err != nil {
		fmt.Printf("JSON序列化失败: %s\n", err)
		return true
	}

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("POST", requesturl, bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Printf("创建请求失败: %s\n", err)
		return true
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		sender.Reply("服务器失联啦，过会再试")
		return true
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("请求失败，状态码: %d\n", resp.StatusCode)
		return true
	}
	var result map[string]interface{}
	if err = json.NewDecoder(resp.Body).Decode(&result); err != nil {
		fmt.Printf("解析响应失败: %s\n", err)
		return true
	}
	logs.Info(fmt.Sprintf("响应: %v\n", result))

	errCode, ok := result["err_code"].(float64)
	if !ok {
		sender.Reply("响应格式错误")
		return true
	}

	if errCode == 133 || errCode == 138 || errCode == 137 || errCode == 142 || errCode == 143 || errCode == 128 || errCode == 141 {
		errMsg, _ := result["err_msg"].(string)
		jmpUrl, _ := result["jmp_url"].(string)
		sender.Reply(errMsg)
		sender.Reply(jmpUrl)
		sender.Reply("请点击链接，或者复制到浏览器打开，验证后请回复【密码登录】再次登录即可,")
		cks := GetJdCookies(func(sb *gorm.DB) *gorm.DB { return sb.Where("Account = ?", Auto.account) })
		if len(cks) > 0 {
			cks[0].Update(Smsverify, "true")
			cks[0].Update(Available, "false")
		}
		return true
	}

	if errCode == 118 {
		sender.Reply("账号触发特殊验证，请先使用【短信登录】后，再次尝试密码登录，如果再次失败请稍后再试")
		return true
	}

	if errCode == 257 {
		sender.Reply("过滑块失败，请重试")
		return true
	}

	if errCode == 6 || errCode == 7 {
		errMsg, _ := result["err_msg"].(string)
		sender.Reply(errMsg)
		cks := GetJdCookies(func(sb *gorm.DB) *gorm.DB { return sb.Where("Account = ?", Auto.account) })
		if len(cks) > 0 {
			cks[0].Update(Password, "")
			cks[0].Update(Account, "")
		}
		return true
	}

	if errCode == 1 {
		errMsg, _ := result["err_msg"].(string)
		sender.Reply(errMsg)
		sender.Reply("接口故障，请先使用【短信登录】提交ck")
		return true
	}

	if errCode != 0 {
		errMsg, _ := result["err_msg"].(string)
		sender.Reply(errMsg)
		return true
	}

	pin, _ := result["pt_pin"].(string)
	logs.Info(pin)

	if ptKey, ok := result["pt_key"].(string); ok && ptKey != "" {
		escapedPin := url.QueryEscape(pin)
		logs.Info(escapedPin)
		cookie := fmt.Sprintf("pt_key=%s;pin=%s;", ptKey, escapedPin)
		logs.Info(cookie)
		Auto.appck = cookie
	}

	fmt.Printf("登录成功: %s\n", pin)
	Autockup(Auto.appck, sender, Auto)
	return true
}
// 上传ck
func Autockup(cookie string, sender *Sender, Auto *UserSession) {
	logs.Info("上传CK阶段")
	pin := FetchJdCookieValue("pin", Auto.appck)
	ptkey := FetchJdCookieValue("pt_key", Auto.appck)
	ck := JdCookie{
		PtPin:           pin,
		PtKey:           ptkey,
		Available:       True, // 保留原有大写True
		QQ:              sender.UserID,
		Account:         Auto.account,
		Password:        Auto.password,
		// 新增：保存Socks5信息（仅当使用BBK登录时）
		Socks5_Ip:       Auto.Socks5_Ip,
		Socks5_Port:     Auto.Socks5_Port,
		Socks5_Account:  Auto.Socks5_Account,
		Socks5_Password: Auto.Socks5_Password,
	}

	if nck, err := GetJdCookie(ck.PtPin); err == nil {
		if Auto.isUser {
			// ========== 注释积分奖励逻辑 - 开始 ==========
			// ========== 注释积分奖励逻辑 - 结束 ==========
			cookieUpdate := JdCookie{
				RWskey:    "null",
				QQ:        sender.UserID,
				PtKey:     ptkey,
				Available: True, // 保留原有大写True
				Account:   Auto.account,
				Password:  Auto.password,
				Smsverify: "false",
				NoticeNum: 2,
			}
			// 仅当使用BBK登录时才更新Socks5信息
			if GetEnv("password_login") == "BBK" {
				cookieUpdate.Socks5_Ip = Auto.Socks5_Ip
				cookieUpdate.Socks5_Port = Auto.Socks5_Port
				cookieUpdate.Socks5_Account = Auto.Socks5_Account
				cookieUpdate.Socks5_Password = Auto.Socks5_Password
			}
			// ========== 注释积分奖励相关的UpdateAt更新 - 开始 ==========
			// ========== 注释积分奖励相关的UpdateAt更新 - 结束 ==========
			switch sender.Type {
			case "wx", "wxg":
				cookieUpdate.WeiXin = sender.WxId
			case "tg", "tgg":
				cookieUpdate.Telegram = sender.UserID
			}
			nck.Updates(cookieUpdate)
			sender.Reply(fmt.Sprintf("登录成功:%s", pin))
			(&JdCookie{}).Push(fmt.Sprintf("来自密码登录成功:%s", pin))
		} else {
			// ========== 注释积分奖励逻辑 - 开始 ==========
			// ========== 注释积分奖励逻辑 - 结束 ==========
			cookieUpdate := JdCookie{
				RWskey:    "null",
				PtKey:     ptkey,
				Available: True, // 保留原有大写True
				Smsverify: "false",
				NoticeNum: 2,
			}
			// 仅当使用BBK登录时才更新Socks5信息
			if GetEnv("password_login") == "BBK" {
				cookieUpdate.Socks5_Ip = Auto.Socks5_Ip
				cookieUpdate.Socks5_Port = Auto.Socks5_Port
				cookieUpdate.Socks5_Account = Auto.Socks5_Account
				cookieUpdate.Socks5_Password = Auto.Socks5_Password
			}
			// ========== 注释积分奖励相关的逻辑 - 开始 ==========
			// 	(&JdCookie{}).Push(fmt.Sprintf("来自密码自动登录成功:%s，成功发放积分奖励", pin))
			// 	(&JdCookie{}).Push(fmt.Sprintf("来自密码自动登录成功:%s，三日内积分已发放", pin))
			// ========== 注释积分奖励相关的逻辑 - 结束 ==========
			// 新增：保留自动登录成功的推送（无积分相关）
			(&JdCookie{}).Push(fmt.Sprintf("来自密码自动登录成功:%s", pin))
			nck.Updates(cookieUpdate)
		}

	} else {
		// 新增账号，保存Socks5信息（仅BBK登录时）
		if GetEnv("password_login") != "BBK" {
			// 非BBK登录，清空Socks5信息
			ck.Socks5_Ip = ""
			ck.Socks5_Port = ""
			ck.Socks5_Account = ""
			ck.Socks5_Password = ""
		}
		NewJdCookie(&ck)
		msg := fmt.Sprintf("来自密码登录的添加账号，账号名:%s", ck.PtPin)
		switch sender.Type {
		case "wx", "wxg":
			ck.Update("WeiXin", sender.WxId)
		case "tg", "tgg":
			ck.Update("Telegram", sender.UserID)
		}
		sender.Reply(fmt.Sprintf(msg))
		// ========== 注释积分奖励逻辑 - 开始 ==========
		// ========== 注释积分奖励逻辑 - 结束 ==========
		sender.Reply(ck.Query())
		(&JdCookie{}).Push(msg)
	}
	go func() {
		Save <- &JdCookie{}
	}()
	return
}

// ========== 注释积分奖励核心函数 - 开始 ==========

// 		coin := 20 //奖励积分数量
// 			"coin": gorm.Expr(fmt.Sprintf("coin+%d", coin)),
// ========== 注释积分奖励核心函数 - 结束 ==========

func UpAutoCookie() {
	logs.Info("开始密码自动登录检测（8IP多线程模式）")

	// 步骤1：筛选待自动登录的账号（不变）
	cks := GetJdCookies(func(sb *gorm.DB) *gorm.DB {
		return sb.Where("Account IS NOT NULL AND Account != '' AND Password IS NOT NULL AND Password != '' AND Smsverify = ? AND Socks5_Ip IS NOT NULL AND Socks5_Ip != ''", False).
			Order("Priority DESC")
	})
	logs.Info(fmt.Sprintf("需要自动登录账号总数：%d个", len(cks)))
	if len(cks) == 0 {
		logs.Info("无符合条件的自动登录账号，检测结束")
		return
	}

	// 步骤2：按Socks5_IP自动分组（不变）
	ipGroups := make(map[string][]JdCookie)
	for _, ck := range cks {
		ipGroups[ck.Socks5_Ip] = append(ipGroups[ck.Socks5_Ip], ck)
	}
	logs.Info(fmt.Sprintf("实际有效IP组数：%d个（目标8个）", len(ipGroups)))
	for ip, group := range ipGroups {
		logs.Info(fmt.Sprintf("IP:%s 分配账号数：%d个", ip, len(group)))
	}

	// 步骤3：启动并发线程（核心优化：循环内延迟逻辑）
	var wg sync.WaitGroup
	for ip, group := range ipGroups {
		wg.Add(1)
		go func(ip string, accountGroup []JdCookie) {
			defer wg.Done()
			logs.Info(fmt.Sprintf("IP:%s 线程启动，开始处理%d个账号", ip, len(accountGroup)))

			for i, ck := range accountGroup {
				logs.Info(fmt.Sprintf("IP:%s 正在处理第%d个账号（账号：%s）", ip, i+1, ck.Account))

				// 跨线程限流：等待3秒信号量（不变）
				<-rateLimitSemaphore

				// 关键标记：记录当前账号是否执行了登录操作（Cookie无效才会登录）
				didLogin := false

				// 检查Cookie有效性，无效则登录（逻辑不变，新增标记）
				if !CookieOK(&ck) {
					didLogin = true // 标记：当前账号执行了登录
					time.Sleep(1 * time.Second) // 原逻辑保留的1秒延迟
					Auto := &UserSession{}
					sender2 := &Sender{
						UserID: 1,
						Type:   "tg",
					}
					Auto.isAuto = true
					Auto.isUser = false
					Auto.account = ck.Account
					Auto.password = ck.Password
					Auto.Socks5_Ip = ck.Socks5_Ip
					Auto.Socks5_Port = ck.Socks5_Port
					Auto.Socks5_Account = ck.Socks5_Account
					Auto.Socks5_Password = ck.Socks5_Password
					logs.Info("IP:%s 自动登录账号 %s 使用Socks5代理：%s:%s", ip, ck.Account, Auto.Socks5_Ip, Auto.Socks5_Port)
					loginAPI(sender2, Auto)
				} else {
					logs.Info(fmt.Sprintf("IP:%s 账号 %s Cookie仍有效，跳过登录", ip, ck.Account))
				}

				// 核心优化：组内延迟仅在「执行了登录」且「不是最后一个账号」时触发
				if i < len(accountGroup)-1 { // 不是最后一个账号
					if didLogin {
						// 无效账号登录后：延迟150秒
						logs.Info(fmt.Sprintf("IP:%s 账号%s登录成功，组内延迟%d秒（Config.Later配置）", ip, ck.Account, Config.Later))
						time.Sleep(time.Second * time.Duration(Config.Later))
					} else {
						// 有效账号跳过登录：无延迟（或可选1秒短延迟防高频查询）
						logs.Info(fmt.Sprintf("IP:%s 账号%sCookie有效，跳过组内延迟", ip, ck.Account))
						// 可选：添加1秒短延迟（避免连续查询Cookie有效性）
					}
				} else {
					// 最后一个账号：无论是否登录，都无延迟
					logs.Info(fmt.Sprintf("IP:%s 账号%s是当前组最后一个账号，无后续延迟", ip, ck.Account))
				}
			}

			logs.Info(fmt.Sprintf("IP:%s 线程处理完成，共处理%d个账号", ip, len(accountGroup)))
		}(ip, group)
	}

	// 等待所有线程执行完成（不变）
	wg.Wait()
	logs.Info("所有IP线程自动登录检测完成")

	// 保存数据库变更（不变）
	go func() {
		Save <- &JdCookie{}
	}()
}




func all_UpAutoCookie() {
	logs.Info("开始密码自动登录检测（8IP多线程模式）")

	// 步骤1：筛选待自动登录的账号（不变）
	cks := GetJdCookies(func(sb *gorm.DB) *gorm.DB {
		return sb.Where("Account IS NOT NULL AND Account != '' AND Password IS NOT NULL AND Password != ''  AND Socks5_Ip IS NOT NULL AND Socks5_Ip != ''").
			Order("Priority DESC")
	})
	logs.Info(fmt.Sprintf("需要自动登录账号总数：%d个", len(cks)))
	if len(cks) == 0 {
		logs.Info("无符合条件的自动登录账号，检测结束")
		return
	}

	// 步骤2：按Socks5_IP自动分组（不变）
	ipGroups := make(map[string][]JdCookie)
	for _, ck := range cks {
		ipGroups[ck.Socks5_Ip] = append(ipGroups[ck.Socks5_Ip], ck)
	}
	logs.Info(fmt.Sprintf("实际有效IP组数：%d个（目标8个）", len(ipGroups)))
	for ip, group := range ipGroups {
		logs.Info(fmt.Sprintf("IP:%s 分配账号数：%d个", ip, len(group)))
	}

	// 步骤3：启动并发线程（核心优化：循环内延迟逻辑）
	var wg sync.WaitGroup
	for ip, group := range ipGroups {
		wg.Add(1)
		go func(ip string, accountGroup []JdCookie) {
			defer wg.Done()
			logs.Info(fmt.Sprintf("IP:%s 线程启动，开始处理%d个账号", ip, len(accountGroup)))

			for i, ck := range accountGroup {
				logs.Info(fmt.Sprintf("IP:%s 正在处理第%d个账号（账号：%s）", ip, i+1, ck.Account))

				// 跨线程限流：等待3秒信号量（不变）
				<-rateLimitSemaphore

				// 关键标记：记录当前账号是否执行了登录操作（Cookie无效才会登录）
				didLogin := false

				// 检查Cookie有效性，无效则登录（逻辑不变，新增标记）
				if !CookieOK(&ck) {
					didLogin = true // 标记：当前账号执行了登录
					time.Sleep(1 * time.Second) // 原逻辑保留的1秒延迟
					Auto := &UserSession{}
					sender2 := &Sender{
						UserID: 1,
						Type:   "tg",
					}
					Auto.isAuto = true
					Auto.isUser = false
					Auto.account = ck.Account
					Auto.password = ck.Password
					Auto.Socks5_Ip = ck.Socks5_Ip
					Auto.Socks5_Port = ck.Socks5_Port
					Auto.Socks5_Account = ck.Socks5_Account
					Auto.Socks5_Password = ck.Socks5_Password
					logs.Info("IP:%s 自动登录账号 %s 使用Socks5代理：%s:%s", ip, ck.Account, Auto.Socks5_Ip, Auto.Socks5_Port)
					loginAPI(sender2, Auto)
				} else {
					logs.Info(fmt.Sprintf("IP:%s 账号 %s Cookie仍有效，跳过登录", ip, ck.Account))
				}

				// 核心优化：组内延迟仅在「执行了登录」且「不是最后一个账号」时触发
				if i < len(accountGroup)-1 { // 不是最后一个账号
					if didLogin {
						// 无效账号登录后：延迟150秒
						logs.Info(fmt.Sprintf("IP:%s 账号%s登录成功，组内延迟%d秒（Config.Later配置）", ip, ck.Account, Config.Later))
						time.Sleep(time.Second * time.Duration(Config.Later))
					} else {
						// 有效账号跳过登录：无延迟（或可选1秒短延迟防高频查询）
						logs.Info(fmt.Sprintf("IP:%s 账号%sCookie有效，跳过组内延迟", ip, ck.Account))
						// 可选：添加1秒短延迟（避免连续查询Cookie有效性）
					}
				} else {
					// 最后一个账号：无论是否登录，都无延迟
					logs.Info(fmt.Sprintf("IP:%s 账号%s是当前组最后一个账号，无后续延迟", ip, ck.Account))
				}
			}

			logs.Info(fmt.Sprintf("IP:%s 线程处理完成，共处理%d个账号", ip, len(accountGroup)))
		}(ip, group)
	}

	// 等待所有线程执行完成（不变）
	wg.Wait()
	logs.Info("所有IP线程自动登录检测完成")

	// 保存数据库变更（不变）
	go func() {
		Save <- &JdCookie{}
	}()
}



//##所有失效都推送
func all_initAutoCookie() {
	(&JdCookie{}).Push("开始密码登录检测")
	cks := GetJdCookies(func(sb *gorm.DB) *gorm.DB {
		return sb.Where("Smsverify = ? AND Account IS NOT NULL AND Account != ''", True)
	})
	xj := 0
	for _, ck := range cks {
		time.Sleep(time.Second * time.Duration(Config.Later))
		time.Sleep(time.Duration(rand.Intn(4000)+1000) * time.Millisecond)

		if !CookieOK(&ck) {
			//todo 通知账号失效
			time.Sleep(time.Second)   //#时间改成1秒
			ck.Push(fmt.Sprintf("====尊贵的密码登录用户====\n1、您的账号：%s，已失效，回复【密码登录】体验全新密码登录，快到你不敢相信\n2、回复【删除账号】删除失效账号", ck.PtPin))
			(&JdCookie{}).Push(fmt.Sprintf("需要验证账号：%s", ck.PtPin))
			xj++
		}
	}
	(&JdCookie{}).Push(fmt.Sprintf("密码登录检测结束，检测账号数量%d个，需要验证账号%d个", len(cks), xj))
	go func() {
		Save <- &JdCookie{}
	}()
}

//##num次数
func initAutoCookie() {
    (&JdCookie{}).Push("开始密码登录检测")
    cks := GetJdCookies(func(sb *gorm.DB) *gorm.DB {
        return sb.Where("Smsverify = ? AND Account IS NOT NULL AND Account != ''", True)
    })
    xj := 0

    for _, ck := range cks {
        time.Sleep(time.Second * time.Duration(Config.Later))
        time.Sleep(time.Duration(rand.Intn(4000)+1000) * time.Millisecond)

        // 检查账号是否失效
        if !CookieOK(&ck) {
            // 当NoticeNum大于0时进行通知推送
            if ck.NoticeNum > 0 {
                // 更新NoticeNum，减去1
                ck.NoticeNum--

                // 更新数据库中账号的NoticeNum字段
                err := db.Model(&ck).Update("NoticeNum", ck.NoticeNum).Error
                if err != nil {
                    (&JdCookie{}).Push(fmt.Sprintf("更新NoticeNum失败，账号：%s，错误：%v", ck.PtPin, err))
                    continue
                }

                // 推送通知
                time.Sleep(time.Second) // 时间改成1秒
                ck.Push(fmt.Sprintf("====尊贵的密码登录用户====\n1、您的账号：%s，已失效，回复【密码登录】重新登录\n2、回复【删除账号】删除失效的账号\n3、回复【QQ群】查看Q群和QQ机器人具体信息", ck.PtPin))
                (&JdCookie{}).Push(fmt.Sprintf("需要验证账号：%s", ck.PtPin))

                // 更新通知计数
                xj++
            } else {
                // NoticeNum为0时，跳过此账号
                continue
            }
        }
    }
    (&JdCookie{}).Push(fmt.Sprintf("密码登录检测结束，检测账号数量%d个，需要推送账号%d个", len(cks), xj))
    go func() {
        Save <- &JdCookie{}
    }()
}