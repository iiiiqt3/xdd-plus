package models

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"regexp"
	"strconv"
	//	"bytes"
	"github.com/beego/beego/v2/core/logs"
	"gorm.io/gorm"
	"os/exec"
)

func handleUserChoice(sender *Sender, msg chan string, cks []JdCookie) {
	timeout := time.After(60 * time.Second)
	var selectedTask string // 保存用户选择的任务
	for {
		select {
		case n, ok := <-msg:
			if !ok {
				return
			}

			// 根据用户输入的数字选择任务
			switch n {

			case "1":
				selectedTask = "Jd_newfruit_watering"
			case "2":
				selectedTask = "jd_plantBean"
			case "3":
				selectedTask = "jd_dwapp"

			case "4":
				selectedTask = "Jd_price"
			case "5":
				selectedTask = "Jd_AutoEval"
			case "6":
				selectedTask = "jd_delLjq"
			case "7":
				selectedTask = "jd_insight"

			case "q":
				sender.Reply("退出流程")
				inputList[sender.UserID] = nil
				return
			default:
				sender.Reply("输入无效，请重新输入任务序号，或回复'q'退出流程。")
				continue
			}

			go handleAccountChoice(sender, msg, cks, selectedTask)
			return
		case <-timeout:
			sender.Reply("操作超时，退出流程！")
			inputList[sender.UserID] = nil
			return
		}
	}
}

func handleAccountChoice(sender *Sender, msg chan string, cks []JdCookie, selectedTask string) {

	msgs := []string{
		"请回复以下数字列号指定账号运行任务，如需退出请回复'q'退出任务流程：",
		"0、所有账号", // 将“所有账号”放在最上面
	}

	// 添加所有具体账号，从1开始
	for i, ck := range cks {
		statusText, _ := GetAccountStatusText(&ck)
		// 包装成和原格式一致的【状态】样式
		status := fmt.Sprintf("【%s】", statusText)

		// 添加状态到 msgs 中
		msgs = append(msgs, fmt.Sprintf("%d、%s %s", i+1, ck.Nickname, status)) // 从1开始
	}

	// 回复消息
	sender.Reply(strings.Join(msgs, "\n"))

	timeout := time.After(60 * time.Second)
	for {
		select {
		case n, ok := <-msg:
			if !ok {
				return
			}

			if strings.ToLower(n) == "q" {
				sender.Reply("已退出流程！")
				inputList[sender.UserID] = nil
				return
			}

			num, err := strconv.Atoi(n)
			if err != nil || num < 0 || num > len(cks) { // 修改这里以包括0到len(cks)的范围
				sender.Reply("输入错误，请重新输入序列号，或回复'q'退出流程。")
				continue
			}

			if num == 0 { // 用户选择“所有账号”
				// 日志：记录用户选择了所有账号，并输出所有ck的关键信息（包含pt_key）
				log.Printf("[用户:%d] 选择了所有账号执行任务[%s]，共%d个账号，ck信息如下：",
					sender.UserID, selectedTask, len(cks))
				for i, ck := range cks {
					log.Printf("  账号%d: Nickname=%s, PtPin=%s, PtKey=%s", i+1, ck.Nickname, ck.PtPin, ck.PtKey)
				}

				// 循环执行所有账号
				for _, ck := range cks {
					envs := map[string]string{
						"pins": "&" + ck.PtPin,
					}

					// 根据选择的任务执行不同的操作
					switch selectedTask {
					case "Jd_newfruit_watering":
						if IsJdTaskProxyEnabled() {
							envs["FRUIT_NEW_DELAY"] = "5"
						} else {
							envs["FRUIT_NEW_DELAY"] = "8"
						}
						go JdTaskHandler(
							sender,
							"新农场浇水",
							"ncjs",
							ExecPath+"/scripts/6dylan6_jdpro/jd_fruit_new.js",
							envs,
							replexQuan_newWatering,
						)
					case "jd_plantBean":
						go JdTaskHandler(
							sender,
							"种豆得豆任务",
							"zhongdoudedou",
							ExecPath+"/scripts/6dylan6_jdpro/jd_plantBean.js",
							envs,
							replexQuan_jd_plantBean,
						)
					case "jd_dwapp":
						envs["ONEVAL"] = "true" // 开启评价
						go JdTaskHandler(
							sender,
							"话费积分任务",
							"huafeijifen",
							ExecPath+"/scripts/6dylan6_jdpro/jd_dwapp.js",
							envs,
							replexQuan_jd_dwapp,
						)
					case "Jd_price":
						go JdTaskHandler(
							sender,
							"一键保价",
							"baojia",
							ExecPath+"/scripts/6dylan6_jdpro/jd_OnceApply.js",
							envs,
							replexQuan_Price,
						)
					case "Jd_AutoEval":
						envs["ONEVAL"] = "true" // 开启评价
						go JdTaskHandler(
							sender,
							"一键评价",
							"pingjia",
							ExecPath+"/scripts/6dylan6_jdpro/jd_AutoEval.js",
							envs,
							replexQuan_AutoEval,
						)
					case "jd_delLjq":
						go JdTaskHandler(
							sender,
							"删除垃圾券",
							"delljq",
							ExecPath+"/scripts/6dylan6_jdpro/jd_delLjq.js",
							envs,
							replexQuan_jd_delLjq,
						)
					case "jd_insight":
						go JdTaskHandler(
							sender,
							"问卷调查得豆",
							"wjdc",
							ExecPath+"/scripts/6dylan6_jdpro/jd_insight.js",
							envs,
							replexQuan_jd_insight,
						)
					}
				}
				// 所有账号任务已启动
				sender.Reply("所有账号的任务已开始执行，请耐心等待回执。")
			} else {
				// 执行用户选择的单个账号任务
				ck := cks[num-1] // 根据用户输入的序号获取账号，减去1以获得正确索引

				// 日志：记录用户选择的单个账号及对应的ck关键信息（包含pt_key）
				log.Printf("[用户:%d] 选择了第%d个账号执行任务[%s]，ck信息：Nickname=%s, PtPin=%s, PtKey=%s",
					sender.UserID, num, selectedTask, ck.Nickname, ck.PtPin, ck.PtKey)

				envs := map[string]string{
					"pins": "&" + ck.PtPin,
				}

				// 根据选择的任务执行不同的操作
				switch selectedTask {
				case "Jd_newfruit_watering":
					if IsJdTaskProxyEnabled() {
						envs["FRUIT_NEW_DELAY"] = "5"
					} else {
						envs["FRUIT_NEW_DELAY"] = "8"
					}
					go JdTaskHandler(
						sender,
						"新农场浇水",
						"ncjs",
						ExecPath+"/scripts/6dylan6_jdpro/jd_fruit_new.js",
						envs,
						replexQuan_newWatering,
					)
				case "Jd_price":
					go JdTaskHandler(
						sender,
						"一键保价",
						"baojia",
						ExecPath+"/scripts/6dylan6_jdpro/jd_OnceApply.js",
						envs,
						replexQuan_Price,
					)
				case "jd_plantBean":
					go JdTaskHandler(
						sender,
						"种豆得豆任务",
						"zhongdoudedou",
						ExecPath+"/scripts/6dylan6_jdpro/jd_plantBean.js",
						envs,
						replexQuan_jd_plantBean,
					)
				case "jd_dwapp":
					envs["ONEVAL"] = "true" // 开启评价
					go JdTaskHandler(
						sender,
						"话费积分任务",
						"huafeijifen",
						ExecPath+"/scripts/6dylan6_jdpro/jd_dwapp.js",
						envs,
						replexQuan_jd_dwapp,
					)
				case "Jd_AutoEval":
					envs["ONEVAL"] = "true" // 开启评价
					go JdTaskHandler(
						sender,
						"一键评价",
						"pingjia",
						ExecPath+"/scripts/6dylan6_jdpro/jd_AutoEval.js",
						envs,
						replexQuan_AutoEval,
					)
				case "jd_delLjq":
					go JdTaskHandler(
						sender,
						"删除垃圾券",
						"delljq",
						ExecPath+"/scripts/6dylan6_jdpro/jd_delLjq.js",
						envs,
						replexQuan_jd_delLjq,
					)
				case "jd_insight":
					go JdTaskHandler(
						sender,
						"问卷调查得豆",
						"wjdc",
						ExecPath+"/scripts/6dylan6_jdpro/jd_insight.js",
						envs,
						replexQuan_jd_insight,
					)
				}
				sender.Reply(fmt.Sprintf("%s 的任务已开始执行，请耐心等待回执。", ck.Nickname))
			}

			inputList[sender.UserID] = nil
			return
		case <-timeout:
			sender.Reply("操作超时，退出流程！")
			inputList[sender.UserID] = nil
			return
		}
	}
}

// 通用任务处理器
func JdTaskHandler(sender *Sender, taskName string, envVar string, scriptPath string, envs map[string]string, outputParser func(string, *Sender) string) {
	if sender.IsAdmin {
	} else {
		value := GetEnv(envVar)
		if value == "" {
			sender.Reply(fmt.Sprintf("管理员未开启%s功能", taskName))
			return
		}

		coin := GetCoin(sender.UserID)
		jbcoin, _ := strconv.Atoi(value)
		if coin < jbcoin {
			return
		}
		RemCoin(sender.UserID, jbcoin)
		RecordCoinLog(sender.UserID, -jbcoin, "任务扣费", fmt.Sprintf("执行任务: %s", taskName))
	}

	ExecuteTask(sender, taskName, scriptPath, envs, outputParser)
}

// 通用任务执行函数
func ExecuteTask(sender *Sender, taskName string, scriptPath string, envs map[string]string, outputParser func(string, *Sender) string) {
	logs.Info("开始运行%s", taskName)
	ApplyJdTaskProxyEnvs(envs)

	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		logs.Error("JavaScript 文件不存在: %v", err)
		sender.Reply(fmt.Sprintf("%s任务失败：脚本不存在", taskName))
		return
	}

	cmd := exec.Command("node", scriptPath)

	for key, value := range envs {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		logs.Error("cmd.StdoutPipe: ", err)
		sender.Reply(fmt.Sprintf("%s任务失败：获取输出管道失败", taskName))
		return
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		logs.Error("cmd.StderrPipe: ", err)
		sender.Reply(fmt.Sprintf("%s任务失败：获取错误管道失败", taskName))
		return
	}

	err = cmd.Start()
	if err != nil {
		logs.Error("cmd.Start: ", err)
		sender.Reply(fmt.Sprintf("%s任务失败：启动失败", taskName))
		return
	}

	// 异步读取 stderr（实时记录到后台日志）
	go func() {
		reader := bufio.NewReader(stderr)
		for {
			line, err2 := reader.ReadString('\n')
			if err2 != nil || io.EOF == err2 {
				break
			}
			logs.Info("[%s] stderr: %s", taskName, strings.TrimSpace(line))
		}
	}()

	// 实时读取 stdout，同时累积完整输出
	var fullOutput strings.Builder
	reader := bufio.NewReader(stdout)
	for {
		line, err2 := reader.ReadString('\n')
		if err2 != nil || io.EOF == err2 {
			break
		}
		fullOutput.WriteString(line)
		logs.Info("[%s] %s", taskName, strings.TrimSpace(line)) // 实时记录到后台日志
	}

	err = cmd.Wait()
	if err != nil && strings.TrimSpace(fullOutput.String()) == "" {
		logs.Error("执行 JavaScript 脚本失败: %v", err)
		sender.Reply(fmt.Sprintf("%s任务失败：执行脚本错误", taskName))
		return
	}

	// 脚本结束后，使用完整输出进行匹配（用户只看到这个结果）
	sender.Reply(outputParser(fullOutput.String(), sender))
}

// 任务日志通道管理
var taskLogChannels = make(map[string]chan string)
var taskLogMutex sync.Mutex
var taskLogTimers = make(map[string]*time.Timer)

// 任务命令管理（用于停止任务）
var taskCmds = make(map[string]*exec.Cmd)
var taskCmdMutex sync.Mutex

// GetTaskLogChannel 获取任务日志通道
func GetTaskLogChannel(taskId string) chan string {
	taskLogMutex.Lock()
	defer taskLogMutex.Unlock()
	return taskLogChannels[taskId]
}

// CreateTaskLogChannel 创建任务日志通道
func CreateTaskLogChannel(taskId string) chan string {
	taskLogMutex.Lock()
	defer taskLogMutex.Unlock()
	ch := make(chan string, 100)
	taskLogChannels[taskId] = ch
	
	// 设置最大存活时间 10 分钟，防止通道泄漏
	timer := time.AfterFunc(10*time.Minute, func() {
		RemoveTaskLogChannel(taskId)
	})
	taskLogTimers[taskId] = timer
	
	return ch
}

// RemoveTaskLogChannel 移除任务日志通道
func RemoveTaskLogChannel(taskId string) {
	taskLogMutex.Lock()
	defer taskLogMutex.Unlock()
	if ch, ok := taskLogChannels[taskId]; ok {
		close(ch)
		delete(taskLogChannels, taskId)
	}
	if timer, ok := taskLogTimers[taskId]; ok {
		timer.Stop()
		delete(taskLogTimers, taskId)
	}
}

// safeLogSend 安全发送日志到通道，避免向已关闭的通道发送导致 panic
func safeLogSend(ch chan string, msg string) {
	defer func() { recover() }()
	select {
	case ch <- msg:
	default:
	}
}

// 运行中的任务管理 {userId_taskId: {taskLogId: accountIndexes}}
var runningTasksMap = make(map[string]map[string][]int)
var runningTasksMutex sync.Mutex

// GetRunningTask 检查是否有同一任务的同一账号正在执行
func GetRunningTask(userId int, taskId string, accountIndexes []int) string {
	runningTasksMutex.Lock()
	defer runningTasksMutex.Unlock()
	
	key := fmt.Sprintf("%d_%s", userId, taskId)
	taskMap, exists := runningTasksMap[key]
	if !exists {
		return ""
	}
	
	// 检查是否有账号冲突
	for _, idx := range accountIndexes {
		for logId, runningIndexes := range taskMap {
			for _, runningIdx := range runningIndexes {
				if idx == 0 || runningIdx == 0 || idx == runningIdx {
					return logId
				}
			}
		}
	}
	
	return ""
}

// RegisterRunningTask 注册运行中的任务
func RegisterRunningTask(userId int, taskId string, accountIndexes []int, taskLogId string) {
	runningTasksMutex.Lock()
	defer runningTasksMutex.Unlock()
	
	key := fmt.Sprintf("%d_%s", userId, taskId)
	if runningTasksMap[key] == nil {
		runningTasksMap[key] = make(map[string][]int)
	}
	runningTasksMap[key][taskLogId] = accountIndexes
}

// UnregisterRunningTask 注销运行中的任务
func UnregisterRunningTask(userId int, taskId string, taskLogId string) {
	runningTasksMutex.Lock()
	defer runningTasksMutex.Unlock()
	
	key := fmt.Sprintf("%d_%s", userId, taskId)
	if taskMap, exists := runningTasksMap[key]; exists {
		delete(taskMap, taskLogId)
		if len(taskMap) == 0 {
			delete(runningTasksMap, key)
		}
	}
}

// StopPortalJdTask 停止正在执行的任务
func StopPortalJdTask(taskId string) {
	taskCmdMutex.Lock()
	defer taskCmdMutex.Unlock()
	if cmd, ok := taskCmds[taskId]; ok && cmd.Process != nil {
		cmd.Process.Kill()
		delete(taskCmds, taskId)
	}
}

// ExecutePortalJdTask 执行网页端京东任务
func ExecutePortalJdTask(userId int, taskId string, taskName string, accountIndexes []int, taskLogId string) {
	// 获取日志通道
	logChan := GetTaskLogChannel(taskLogId)
	if logChan == nil {
		logChan = CreateTaskLogChannel(taskLogId)
	}

	// 发送开始日志
	safeLogSend(logChan, fmt.Sprintf("开始执行任务: %s", taskName))

	// 获取用户的京东账号
	var idType string
	idType = QQ // 默认使用QQ
	cks := GetJdCookies(func(sb *gorm.DB) *gorm.DB {
		return sb.Where(fmt.Sprintf("%s = ? and %s = ?", idType, "Available"), userId, "True")
	})

	safeLogSend(logChan, fmt.Sprintf("查询到 %d 个有效账号 (userId=%d)", len(cks), userId))

	if len(cks) == 0 {
		safeLogSend(logChan, "错误: 没有找到有效的京东账号")
		return
	}

	// 筛选要执行的账号
	var selectedCks []JdCookie
	for _, idx := range accountIndexes {
		if idx == 0 {
			// 选择所有账号
			selectedCks = cks
			break
		}
		if idx > 0 && idx <= len(cks) {
			selectedCks = append(selectedCks, cks[idx-1])
		}
	}

	safeLogSend(logChan, fmt.Sprintf("筛选后 %d 个账号 (传入索引: %v)", len(selectedCks), accountIndexes))

	if len(selectedCks) == 0 {
		safeLogSend(logChan, "错误: 没有选择有效的账号")
		return
	}

	safeLogSend(logChan, fmt.Sprintf("已选择 %d 个账号", len(selectedCks)))

	// 根据任务类型执行
	for _, ck := range selectedCks {
		safeLogSend(logChan, fmt.Sprintf("执行账号: %s (%s)", ck.Nickname, ck.PtPin))

		envs := map[string]string{
			"pins": "&" + ck.PtPin,
		}

		var scriptPath string
		var parser func(string, *Sender) string

		switch taskId {
		case "newfruit_watering":
			scriptPath = ExecPath + "/scripts/6dylan6_jdpro/jd_fruit_new.js"
			parser = replexQuan_newWatering
			if IsJdTaskProxyEnabled() {
				envs["FRUIT_NEW_DELAY"] = "5"
			} else {
				envs["FRUIT_NEW_DELAY"] = "8"
			}
		case "plantBean":
			scriptPath = ExecPath + "/scripts/6dylan6_jdpro/jd_plantBean.js"
			parser = replexQuan_jd_plantBean
		case "dwapp":
			scriptPath = ExecPath + "/scripts/6dylan6_jdpro/jd_dwapp.js"
			parser = replexQuan_jd_dwapp
			envs["ONEVAL"] = "true"
		case "price":
			scriptPath = ExecPath + "/scripts/6dylan6_jdpro/jd_OnceApply.js"
			parser = replexQuan_Price
		case "autoEval":
			scriptPath = ExecPath + "/scripts/6dylan6_jdpro/jd_AutoEval.js"
			parser = replexQuan_AutoEval
			envs["ONEVAL"] = "true"
		case "delLjq":
			scriptPath = ExecPath + "/scripts/6dylan6_jdpro/jd_delLjq.js"
			parser = replexQuan_jd_delLjq
		case "insight":
			scriptPath = ExecPath + "/scripts/6dylan6_jdpro/jd_insight.js"
			parser = replexQuan_jd_insight
		default:
			safeLogSend(logChan, fmt.Sprintf("错误: 未知的任务类型 %s", taskId))
			return
		}

		// 执行任务并实时推送日志
		executeTaskWithLogs(taskLogId, taskName, scriptPath, envs, parser, logChan)
	}

	safeLogSend(logChan, "=====DONE=====所有账号任务执行完成")
}

// executeTaskWithLogs 执行任务并实时推送日志
func executeTaskWithLogs(taskId string, taskName string, scriptPath string, envs map[string]string, outputParser func(string, *Sender) string, logChan chan string) {
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		safeLogSend(logChan, fmt.Sprintf("错误: 脚本文件不存在 %s", scriptPath))
		return
	}

	ApplyJdTaskProxyEnvs(envs)

	cmd := exec.Command("node", scriptPath)
	for key, value := range envs {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		safeLogSend(logChan, fmt.Sprintf("错误: 获取输出管道失败 %v", err))
		return
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		safeLogSend(logChan, fmt.Sprintf("错误: 获取错误管道失败 %v", err))
		return
	}

	err = cmd.Start()
	if err != nil {
		safeLogSend(logChan, fmt.Sprintf("错误: 启动脚本失败 %v", err))
		return
	}

	// 注册命令到任务命令管理器
	taskCmdMutex.Lock()
	taskCmds[taskId] = cmd
	taskCmdMutex.Unlock()

	// 任务完成后注销命令
	defer func() {
		taskCmdMutex.Lock()
		delete(taskCmds, taskId)
		taskCmdMutex.Unlock()
	}()

	// 异步读取 stderr
	go func() {
		reader := bufio.NewReader(stderr)
		for {
			line, err2 := reader.ReadString('\n')
			if err2 != nil {
				if err2 != io.EOF && len(strings.TrimSpace(line)) > 0 {
					safeLogSend(logChan, fmt.Sprintf("[stderr] %s", strings.TrimSpace(line)))
				}
				break
			}
			if len(strings.TrimSpace(line)) > 0 {
				safeLogSend(logChan, fmt.Sprintf("[stderr] %s", strings.TrimSpace(line)))
			}
		}
	}()

	// 实时读取 stdout
	var fullOutput strings.Builder
	reader := bufio.NewReader(stdout)
	for {
		line, err2 := reader.ReadString('\n')
		if err2 != nil {
			if err2 != io.EOF && len(strings.TrimSpace(line)) > 0 {
				fullOutput.WriteString(line)
				safeLogSend(logChan, strings.TrimSpace(line))
			}
			break
		}
		fullOutput.WriteString(line)
		trimmed := strings.TrimSpace(line)
		if len(trimmed) > 0 {
			safeLogSend(logChan, trimmed)
		}
	}

	err = cmd.Wait()
	if err != nil {
		safeLogSend(logChan, fmt.Sprintf("脚本执行完成，退出码: %v", err))
	}

	// 输出匹配后的结果
	sender := &Sender{}
	result := outputParser(fullOutput.String(), sender)
	safeLogSend(logChan, fmt.Sprintf("===== 任务结果 ====="))
	safeLogSend(logChan, result)
}

func replexQuan_Watering(info string, sender *Sender) string {
	re1 := regexp.MustCompile(`(?m)^.*(【京东账号1🆔】.+?)$`)
	re2 := regexp.MustCompile(`(?m)^.*(【水果名称】.+?)$`)
	re3 := regexp.MustCompile(`(?m)^.*(【已兑换水果】.+?)$`)
	re4 := regexp.MustCompile(`(?m)^.*(【今日共浇水】.+?)$`)
	re5 := regexp.MustCompile(`(?m)^.*(【剩余水滴】.+?)$`)
	re6 := regexp.MustCompile(`(?m)^.*(【水果进度】.+?)$`)
	re7 := regexp.MustCompile(`(?m)^.*(【预测】.+?)$`)
	re8 := regexp.MustCompile(`(?m)^.*(【数据异常】.+?)$`)

	matches1 := re1.FindStringSubmatch(info)
	matches2 := re2.FindStringSubmatch(info)
	matches3 := re3.FindStringSubmatch(info)
	matches4 := re4.FindStringSubmatch(info)
	matches5 := re5.FindStringSubmatch(info)
	matches6 := re6.FindStringSubmatch(info)
	matches7 := re7.FindStringSubmatch(info)
	matches8 := re8.FindStringSubmatch(info)

	msgs := []string{
		fmt.Sprintf("农场浇水任务已完成："),
	}

	if len(matches1) > 1 {
		replaceText := strings.Replace(matches1[1], "【京东账号1🆔】", "【京东账号】", -1)
		msgs = append(msgs, replaceText)
	}
	if len(matches2) > 1 {
		msgs = append(msgs, matches2[1])
	}
	if len(matches3) > 1 {
		msgs = append(msgs, matches3[1])
	}
	if len(matches4) > 1 {
		msgs = append(msgs, matches4[1])
	}
	if len(matches5) > 1 {
		if sender.Type == "wx" || sender.Type == "wxg" {
			replaceText := strings.Replace(matches5[1], "💧", "💧", -1)
			msgs = append(msgs, replaceText)
		} else {
			msgs = append(msgs, matches5[1])
		}
	}
	if len(matches6) > 1 {
		msgs = append(msgs, matches6[1])
	}
	if len(matches7) > 1 {
		if (sender.Type == "wx" || sender.Type == "wxg") && strings.Contains(matches7[1], "🍉") {
			replaceText := strings.Replace(matches7[1], "🍉", "[庆祝]", -1)
			msgs = append(msgs, replaceText)
		} else {
			msgs = append(msgs, matches7[1])
		}
	}
	if len(matches8) > 1 {
		msgs = append(msgs, matches8[1])
	}
	msgs = append(msgs, "=================\n提示：农场兑红包，每月限兑4次数，次数可能变更，自测！\n=================")
	return strings.Join(msgs, "\n")
}

func replexQuan_newWatering(info string, sender *Sender) string {
	re1 := regexp.MustCompile(`(?m)^.*(【账号1】.+?)$`)
	re2 := regexp.MustCompile(`(?m)^.*(【水果名称】.+?)$`)
	re3 := regexp.MustCompile(`(?m)^.*(【已完成种植】.+?)$`)
	re4 := regexp.MustCompile(`(?m)^.*(【额外奖励】.+?)$`)
	re5 := regexp.MustCompile(`(?m)^.*(【种植进度】.+?)$`)
	re6 := regexp.MustCompile(`(?m)^.*(【剩余水滴】.+?)$`)
	//    re7 := regexp.MustCompile(`(?m)^.*(还未选择种植.+?)$`)

	re8 := regexp.MustCompile(`(?m)^.*(【数据异常】请手动登录app查看此账号农场是否正常)`)
	matches1 := re1.FindStringSubmatch(info)
	matches2 := re2.FindStringSubmatch(info)
	matches3 := re3.FindStringSubmatch(info)
	matches4 := re4.FindStringSubmatch(info)
	matches5 := re5.FindStringSubmatch(info)
	matches6 := re6.FindStringSubmatch(info)
	matches8 := re8.FindStringSubmatch(info)
	msgs := []string{
		fmt.Sprintf("新农场浇水任务已完成："),
	}

	if len(matches1) > 1 {
		replaceText := strings.Replace(matches1[1], "【京东账号1】", "【京东账号】", -1)
		msgs = append(msgs, replaceText)
	}
	if len(matches2) > 1 {
		msgs = append(msgs, matches2[1])
	}
	if len(matches3) > 1 {
		msgs = append(msgs, matches3[1])
	}
	if len(matches4) > 1 {
		msgs = append(msgs, matches4[1])
	}
	if len(matches5) > 1 {
		msgs = append(msgs, matches5[1])
	}
	if len(matches6) > 1 {
		if sender.Type == "wx" || sender.Type == "wxg" {
			replaceText := strings.Replace(matches6[1], "💧", "💧", -1)
			msgs = append(msgs, replaceText)
		} else {
			msgs = append(msgs, matches6[1])
		}
	}
	if len(matches8) > 1 {
		// 如果匹配到“黑号”，返回相应信息
		return "新农场都进不去了，浇什么水，，如果你京东APP能进入农场，说明IP黑了，请重新执行"
	}
	msgs = append(msgs, "新农场黑ip较为严重，如果失败请重新执行")
	msgs = append(msgs, "========================================\n提示：新农场兑换，请注意有效期！\n========================================")

	return strings.Join(msgs, "\n")

}

//## 种豆得豆匹配

func replexQuan_jd_plantBean(info string, sender *Sender) string {
	// 1. 匹配账号标识中的账号内容（【账号X】后的jd_xxx部分）
	reAccount := regexp.MustCompile(`【账号\d+】(jd_[^\s-]+)`)
	// 2. 匹配从“定时领取营养液”开始到结尾的所有内容
	reCoreLog := regexp.MustCompile(`定时领取营养液：[\s\S]*`)
	// 3. 匹配并移除结尾的结束提示（🔔种豆得豆任务, 结束! 及后续所有内容）
	reEnd := regexp.MustCompile(`🔔种豆得豆任务, 结束! [\s\S]*`)
	// 4. 匹配进入活动失败的关键字
	reFailed := regexp.MustCompile(`进入活动失败`)

	// 如果匹配到“进入活动失败”，直接返回提示
	if reFailed.MatchString(info) {
		return "账号黑了，无法执行种豆得豆任务"
	}

	// 提取账号信息并格式化
	accountMatches := reAccount.FindStringSubmatch(info)
	accountStr := ""
	if len(accountMatches) > 1 {
		accountStr = "====【京东账号】" + accountMatches[1] + "=====\n\n"
	}

	// 第一步：提取核心日志（定时领取营养液开始到结尾）
	coreLog := reCoreLog.FindString(info)
	// 第二步：移除结尾的结束提示内容
	coreLog = reEnd.ReplaceAllString(coreLog, "")
	// 第三步：清理核心日志首尾的空白字符，保证格式整洁
	coreLog = strings.TrimSpace(coreLog)

	// 拼接格式化账号和核心日志
	result := accountStr + coreLog

	return result
}

//##话费积分

func replexQuan_jd_dwapp(info string, sender *Sender) string {
	// 匹配账号信息（【京东账号\d+】及后续信息）
	re1 := regexp.MustCompile(`【京东账号\d+】(.+?)\*\*\*\*`) // 匹配账号信息
	// 匹配签到信息
	re2 := regexp.MustCompile(`签到成功：获得积分(\d+\.\d+)，剩余积分：(\d+\.\d+)`) // 匹配签到成功的信息
	re3 := regexp.MustCompile(`今日已签过！已连签(\d+)天，剩余积分：(\d+\.\d+)`)     // 匹配已经签过的信息
	// 匹配"火爆"字样
	re4 := regexp.MustCompile(`火爆`)

	// 查找并提取账号信息部分
	matches1 := re1.FindStringSubmatch(info)
	// 查找签到成功的积分部分
	matches2 := re2.FindStringSubmatch(info)
	// 查找已经签到过的部分
	matches3 := re3.FindStringSubmatch(info)
	// 查找"火爆"字样
	matches4 := re4.FindStringSubmatch(info)

	// 输出结果字符串
	var msgs []string

	// 如果找到"火爆"，输出"账号黑了"
	if len(matches4) > 0 {
		msgs = append(msgs, "账号黑了，无法签到")
		return strings.Join(msgs, "\n")
	}

	// 如果找到账号信息，则输出账号
	if len(matches1) > 0 {
		msgs = append(msgs, fmt.Sprintf("账号: %s", matches1[1]))
	}

	// 根据情况输出签到状态
	if len(matches2) > 0 {
		// 第一种情况：签到成功，输出积分信息
		msgs = append(msgs, fmt.Sprintf("签到成功：获得积分%s，剩余积分：%s", matches2[1], matches2[2]))
	} else if len(matches3) > 0 {
		// 第二种情况：已签过，输出签到天数和剩余积分
		msgs = append(msgs, fmt.Sprintf("今日已签过！已连签%s天，剩余积分：%s", matches3[1], matches3[2]))
	}

	// 返回连接的结果
	return strings.Join(msgs, "\n")
}

func replexQuan_Price(info string, sender *Sender) string {

	re1 := regexp.MustCompile(`保价失败：([^：]+)$`)
	re2 := regexp.MustCompile(`价保成功：([^：]+)`)
	re3 := regexp.MustCompile(`没有可保价的订单 😂`)

	matches1 := re1.FindStringSubmatch(info)
	matches2 := re2.FindStringSubmatch(info)
	matches3 := re3.FindStringSubmatch(info)

	msgs := []string{
		fmt.Sprintf("保价任务已完成："),
	}

	if len(matches3) > 0 {
		msgs = append(msgs, "没有可保价的订单 😂")
	}
	if len(matches1) > 1 {
		msgs = append(msgs, "保价失败："+matches1[1])
	}
	if len(matches2) > 1 {
		msgs = append(msgs, fmt.Sprintf("价保成功，回血%s元 🤑", matches2[1]))
	}

	return strings.Join(msgs, "\n")
}

func replexQuan_AutoEval(info string, sender *Sender) string {
	re1 := regexp.MustCompile(`(?m)^.*(开始【京东账号1】.+?)$`)
	re2 := regexp.MustCompile(`当前.*?个商品`)

	matches1 := re1.FindStringSubmatch(info)
	matches2 := re2.FindStringSubmatch(info)

	msgs := []string{
		"当前评价任务如下：",
	}

	if len(matches1) > 1 {
		replaceText := strings.Replace(matches1[1], "开始【京东账号1】", "【京东账号】", -1)
		msgs = append(msgs, replaceText)
	}

	if len(matches2) > 0 {
		msgs = append(msgs, matches2[0])
	}
	msgs = append(msgs, "======评价任务已完成======")

	return strings.Join(msgs, "\n")
}

func replexQuan_jd_delLjq(info string, sender *Sender) string {
	re1 := regexp.MustCompile(`(?m)^.*(开始【京东账号1】.+?)$`)
	re2 := regexp.MustCompile(`总计.*?个券`) //#总计2144个券
	re3 := regexp.MustCompile(`完成，本次执行删除.*?个垃圾券`)

	matches1 := re1.FindStringSubmatch(info)
	matches2 := re2.FindStringSubmatch(info)
	matches3 := re3.FindStringSubmatch(info)
	msgs := []string{
		"当前删除垃圾券任务如下：",
	}

	if len(matches1) > 1 {
		replaceText := strings.Replace(matches1[1], "开始【京东账号1】", "【京东账号】", -1)
		msgs = append(msgs, replaceText)
	}

	if len(matches2) > 0 {
		msgs = append(msgs, matches2[0])
	}

	if len(matches3) > 0 {
		msgs = append(msgs, matches3[0])
	}
	msgs = append(msgs, "======删除任务已完成，解决优惠券数量太多而不能使用新农场优惠券的问题，会误删，到已删除券恢复======")

	return strings.Join(msgs, "\n\n")
}

func replexQuan_jd_quan_day(info string, sender *Sender) string {
	// 匹配账号名称部分，忽略前后的星号
	re1 := regexp.MustCompile(`(?m)^\*{0,}开始【([^】]+)】([^*]+)\*{0,}$`)
	matches1 := re1.FindStringSubmatch(info)
	if len(matches1) > 2 {
		// 提取账号部分并添加分隔线
		accountType := matches1[1]
		account := strings.TrimSpace(matches1[2])
		replacement := "---【" + accountType + "】" + account + "---\n\n"

		// 匹配活动时间及后续所有内容
		re2 := regexp.MustCompile(`(?s)活动时间:.*$`)
		matches2 := re2.FindStringSubmatch(info)
		if len(matches2) > 0 {
			// 提取活动时间及后续内容
			activityInfo := matches2[0]
			return replacement + activityInfo
		}
	}
	return "未知错误"
}

func replexQuan_jd_insight(info string, sender *Sender) string {
	// 正则表达式，匹配京东账号信息
	re1 := regexp.MustCompile(`(?m)^.*(开始【京东账号\d+】.+?)$`)

	re2 := regexp.MustCompile(`(?m)^无任何信息$`)
	// 正则表达式，匹配从"1、"到"运行完毕"之间的任务信息
	re3 := regexp.MustCompile(`(?m)(1、[\s\S]+?运行完毕)`)

	// 构建消息数组
	msgs := []string{
		"当前问卷调查任务如下：",
	}

	// 如果日志中包含“无任何信息”，直接返回对应的处理信息
	if re2.MatchString(info) {
		msgs = append(msgs, "你的账号暂时没有问卷调查")
		return strings.Join(msgs, "\n\n")
	}

	// 处理第一个正则匹配，替换【京东账号1】为【京东账号】
	matches1 := re1.FindStringSubmatch(info)
	if len(matches1) > 1 {
		replaceText := strings.Replace(matches1[1], "开始【京东账号1】", "【京东账号】", -1)
		msgs = append(msgs, replaceText)
	}

	// 处理第三个正则匹配，匹配从"1、"到"运行完毕"之间的任务信息
	matches3 := re3.FindStringSubmatch(info)
	if len(matches3) > 0 {
		// 去除任务信息中的“运行完毕”部分
		taskInfo := strings.Replace(matches3[0], "运行完毕", "", -1)
		// 如果匹配到1、到🔔之间的内容，直接返回该内容
		msgs = append(msgs, taskInfo)
	}

	// 添加结束信息
	msgs = append(msgs, "\n======请按机器人给出的提示回答问卷，即可获得京豆======")

	// 返回拼接后的消息字符串
	return strings.Join(msgs, "\n")
}

func replexQuan_fcwb_auto(info string, sender *Sender) string {
	re1 := regexp.MustCompile(`(?m)^.*(开始【京东账号1】.+?)$`)
	reNoBlood := regexp.MustCompile(`没血了，溜了溜了~`)
	rePass := regexp.MustCompile(`当前难度关卡已通关`)

	matches1 := re1.FindStringSubmatch(info)
	noBloodMatch := reNoBlood.FindStringSubmatch(info)
	passMatches := rePass.FindAllStringSubmatch(info, -1) // 匹配所有 "当前难度关卡已通关"

	// 构建消息列表
	msgs := []string{
		"自动挖宝任务情况如下：",
	}

	if len(matches1) > 1 {
		replaceText := strings.Replace(matches1[1], "开始【京东账号1】", "【京东账号】", -1)
		msgs = append(msgs, replaceText)
	}

	if len(noBloodMatch) > 0 {
		msgs = append(msgs, "未能全部通关，请到活动界面查看。")
	}

	if len(passMatches) >= 3 {
		msgs = append(msgs, "游戏已通关。")
	} else if len(passMatches) == 2 {
		msgs = append(msgs, "已挖通2关。")
	} else if len(passMatches) > 0 {
		msgs = append(msgs, "部分关卡已通关。")
	}

	msgs = append(msgs, "===自动挖宝任务已完成===")

	return strings.Join(msgs, "\n")
}

func run_fcwb_help_Task(sender *Sender, envVars map[string]string, FileName string) string {
	logs.Info(fmt.Sprintf("开始运行%s任务", FileName)) // 使用 FileName 动态生成任务名称
	ApplyJdTaskProxyEnvs(envVars)
	jsFilePath := ExecPath + "/scripts/6dylan6_jdpro_help/" + FileName + ".js"

	if _, err := os.Stat(jsFilePath); os.IsNotExist(err) {
		logs.Error("JavaScript 文件不存在: %v", err)
		return fmt.Sprintf("执行%s任务失败：JavaScript 文件不存在", FileName) // 使用 FileName
	}

	cmd := exec.Command("node", jsFilePath)

	// 设置环境变量
	for name, value := range envVars {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", name, value))
	}

	output, err := cmd.CombinedOutput()
	logs.Info(fmt.Sprintf("%s任务脚本输出: %s", FileName, string(output)))

	if err != nil && strings.TrimSpace(string(output)) == "" {
		logs.Error("执行 JavaScript 脚本失败: %v", err)
		return fmt.Sprintf("执行%s任务失败：脚本执行错误", FileName) // 修正斜杠错误
	}

	// 传递 cookie 文件路径给 replexQuan_fcwb_help
	return replexQuan_fcwb_help(string(output), sender, FileName)
}

//##赚赚专属

func run_fcwb_help_Task_zz(sender *Sender, envVars map[string]string, FileName string) string {
	logs.Info(fmt.Sprintf("开始运行%s任务", FileName)) // 使用 FileName 动态生成任务名称
	ApplyJdTaskProxyEnvs(envVars)
	jsFilePath := ExecPath + "/scripts/6dylan6_jdpro_help/" + FileName + ".js"

	if _, err := os.Stat(jsFilePath); os.IsNotExist(err) {
		logs.Error("JavaScript 文件不存在: %v", err)
		return fmt.Sprintf("执行%s任务失败：JavaScript 文件不存在", FileName) // 使用 FileName
	}

	cmd := exec.Command("node", jsFilePath)

	// 设置环境变量
	for name, value := range envVars {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", name, value))
	}

	output, err := cmd.CombinedOutput()
	logs.Info(fmt.Sprintf("%s任务脚本输出: %s", FileName, string(output)))

	if err != nil && strings.TrimSpace(string(output)) == "" {
		logs.Error("执行 JavaScript 脚本失败: %v", err)
		return fmt.Sprintf("执行%s任务失败：脚本执行错误", FileName) // 修正斜杠错误
	}

	// 传递 cookie 文件路径给 replexQuan_fcwb_help
	return replexQuan_fcwb_help_zz(string(output), sender, FileName)
}

//##环境的

func run_fcwb_help_Task1(sender *Sender, envVars map[string]string, FileName string) string {
	logs.Info(fmt.Sprintf("开始运行%s任务", FileName)) // 使用 FileName 动态生成任务名称
	ApplyJdProTaskProxyEnvs(envVars)
	jsFilePath := ExecPath + "/scripts/huanjing/" + FileName + ".js"

	if _, err := os.Stat(jsFilePath); os.IsNotExist(err) {
		logs.Error("JavaScript 文件不存在: %v", err)
		return fmt.Sprintf("执行%s任务失败：JavaScript 文件不存在", FileName) // 使用 FileName
	}

	cmd := exec.Command("node", jsFilePath)

	// 设置环境变量
	for name, value := range envVars {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", name, value))
	}

	output, err := cmd.CombinedOutput()
	logs.Info(fmt.Sprintf("%s任务脚本输出: %s", FileName, string(output)))

	if err != nil && strings.TrimSpace(string(output)) == "" {
		logs.Error("执行 JavaScript 脚本失败: %v", err)
		return fmt.Sprintf("执行%s任务失败：脚本执行错误", FileName) // 修正斜杠错误
	}

	// 传递 cookie 文件路径给 replexQuan_fcwb_help1
	return replexQuan_fcwb_help1(string(output), sender, FileName)
}



// ## 原本农场助力正常的代码
func replexQuan_fcwb_help_zz(info string, sender *Sender, FileName string) string {
	// 定义正则表达式，匹配成功助力的条目和使用的总账号数
	reSuccess := regexp.MustCompile(`助力成功`)
	reTotalAccounts := regexp.MustCompile(`共使用(\d+)个账号`)
	//    reFASLCE := regexp.MustCompile(`连续火爆，跳过`)
	// 找到所有成功助力的条目
	successMatches := reSuccess.FindAllString(info, -1)
	successfulHelps := len(successMatches) // 成功助力的数量

	// 找到总账号数
	totalMatch := reTotalAccounts.FindStringSubmatch(info)
	totalAccounts := 0
	if len(totalMatch) > 1 {
		totalAccounts, _ = strconv.Atoi(totalMatch[1]) // 转换为整数
	}
	value := GetEnv(FileName)
	if value == "" {
		value = "20"
	}
	jbcoin, _ := strconv.Atoi(value)         // 将字符串转换为整数
	coinToDeduct := successfulHelps * jbcoin // 每个成功助力账号扣除20积分
	msgs := []string{}

	if successfulHelps > 0 {
		RemCoin(sender.UserID, coinToDeduct)                                                                               // 扣除积分
		RecordCoinLog(sender.UserID, -coinToDeduct, "助力扣费", fmt.Sprintf("成功助力%d个账号", successfulHelps))
		msgs = append(msgs, fmt.Sprintf("成功助力%d个账号，扣除%d积分，剩余%d积分", successfulHelps, coinToDeduct, GetCoin(sender.UserID))) // 构建反馈消息
	} else {
		msgs = append(msgs, "未找到成功助力的账号")
	}

	if totalAccounts > 0 {
		cookieFilePath := ExecPath + "/scripts/6dylan6_jdpro_help/" + FileName + ".txt"        // 生成 cookie 文件路径
		totalAccountsToDelete := totalAccounts - 5                                             // 删除账号数减少一个，即 totalAccounts - 1
		remainingLinesCount, err := removeLinesFromFile(cookieFilePath, totalAccountsToDelete) // 使用减少后的账号数
		if err != nil {
			msgs = append(msgs, "删除 "+cookieFilePath+" 行数时出错："+err.Error()) // 如果删除时出错，记录错误信息
		} else {
			msg := fmt.Sprintf("删除已使用的%d个号，剩余%d个号", totalAccountsToDelete, remainingLinesCount)
			if msg == "删除已使用的0个号，剩余1个号" {
				msg = "今日助力已用完，请明日早点来，每日早上9点准时开始"
			}
			msgs = append(msgs, msg) // 成功删除后的提示信息
		}
	} else {
		msgs = append(msgs, "未找到总账号数信息，未删除 "+FileName+".txt 行数") // 如果找不到总账号数信息，记录相应信息
	}

	msgs = append(msgs, fmt.Sprintf("===%s任务已完成===", FileName)) // 任务完成后的提示信息
	return strings.Join(msgs, "\n")                             // 返回拼接后的所有消息

}

// ##火爆后的代码
func replexQuan_fcwb_help(info string, sender *Sender, FileName string) string {
	// 定义正则表达式，匹配成功助力的条目和使用的总账号数
	reSuccess := regexp.MustCompile(`助力成功`)
	reTotalAccounts := regexp.MustCompile(`共使用(\d+)个账号`)
	reFASLCE := regexp.MustCompile(`连续火爆，跳过`)

	// 找到所有成功助力的条目
	successMatches := reSuccess.FindAllString(info, -1)
	successfulHelps := len(successMatches) // 成功助力的数量

	// 检查是否存在"连续火爆，跳过"
	hasFASLCE := reFASLCE.MatchString(info)

	// 找到总账号数
	totalMatch := reTotalAccounts.FindStringSubmatch(info)
	totalAccounts := 0
	if len(totalMatch) > 1 {
		totalAccounts, _ = strconv.Atoi(totalMatch[1]) // 转换为整数
	}
	value := GetEnv(FileName)
	if value == "" {
		value = "20"
	}
	jbcoin, _ := strconv.Atoi(value)         // 将字符串转换为整数
	coinToDeduct := successfulHelps * jbcoin // 每个成功助力账号扣除20积分
	msgs := []string{}

	if successfulHelps > 0 {
		RemCoin(sender.UserID, coinToDeduct)                                                                               // 扣除积分
		RecordCoinLog(sender.UserID, -coinToDeduct, "助力扣费", fmt.Sprintf("成功助力%d个账号", successfulHelps))
		msgs = append(msgs, fmt.Sprintf("成功助力%d个账号，扣除%d积分，剩余%d积分", successfulHelps, coinToDeduct, GetCoin(sender.UserID))) // 构建反馈消息

		// 如果同时存在"连续火爆，跳过"，提示重新执行助力
		if hasFASLCE {
			msgs = append(msgs, "没有完全完成助力任务，你可以选择请重新执行助力")
		}
	} else {
		// 如果没有成功助力，但存在"连续火爆，跳过"，单独提示
		if hasFASLCE {
			return "你的账号不能被助力，请更换其他账号"
		}
		msgs = append(msgs, "未找到成功助力的账号")
	}

	if totalAccounts > 0 {
		cookieFilePath := ExecPath + "/scripts/6dylan6_jdpro_help/" + FileName + ".txt"        // 生成 cookie 文件路径
		totalAccountsToDelete := totalAccounts - 4                                             // 删除账号数减少一个，即 totalAccounts - 1
		remainingLinesCount, err := removeLinesFromFile(cookieFilePath, totalAccountsToDelete) // 使用减少后的账号数
		if err != nil {
			msgs = append(msgs, "删除 "+cookieFilePath+" 行数时出错："+err.Error()) // 如果删除时出错，记录错误信息
		} else {
			msg := fmt.Sprintf("删除已使用的%d个号，剩余%d个号", totalAccountsToDelete, remainingLinesCount)
			if msg == "删除已使用的0个号，剩余4个号" {
				msg = "今日助力已用完，请明日早点来，每日早上9点准时开始"
			}
			msgs = append(msgs, msg) // 成功删除后的提示信息
		}
	} else {
		msgs = append(msgs, "未找到总账号数信息，未删除 "+FileName+".txt 行数") // 如果找不到总账号数信息，记录相应信息
	}

	msgs = append(msgs, fmt.Sprintf("===%s任务已完成===", FileName)) // 任务完成后的提示信息
	return strings.Join(msgs, "\n")                             // 返回拼接后的所有消息
}

//##环境的

func replexQuan_fcwb_help1(info string, sender *Sender, FileName string) string {
	// 定义正则表达式，匹配助力次数达到的数字
	reSuccess := regexp.MustCompile(`助力次数达到:\s*(\d+)`) // 匹配 "助力次数达到: X" 格式
	reAccount := regexp.MustCompile(`【京东账号(\d+)】`)     // 匹配 "【京东账号X】" 格式
	reFailure := regexp.MustCompile(`好友的活动已完成，助力失败~`)  // 匹配 "好友的活动已完成，助力失败~"

	// 检查是否匹配到助力失败的日志
	if reFailure.MatchString(info) {
		// 如果助力失败，直接扣除60积分
		RemCoin(sender.UserID, 60)
		RecordCoinLog(sender.UserID, -60, "助力扣费", "助力失败扣费")
		return fmt.Sprintf("助力以完成，或者重复发送任务，扣除60积分，剩余%d积分", GetCoin(sender.UserID))
	}

	// 从 info 中提取助力成功次数
	successMatch := reSuccess.FindStringSubmatch(info)
	var successfulHelps int
	if len(successMatch) > 1 {
		successfulHelps, _ = strconv.Atoi(successMatch[1]) // 获取助力成功次数并转换为整数
	}

	// 获取配置的扣除积分值
	value := GetEnv(FileName)
	if value == "" {
		value = "20"
	}
	jbcoin, _ := strconv.Atoi(value)         // 将字符串转换为整数
	coinToDeduct := successfulHelps * jbcoin // 每个成功助力账号扣除积分

	msgs := []string{}

	if successfulHelps > 0 {
		RemCoin(sender.UserID, coinToDeduct)                                                                               // 扣除积分
		RecordCoinLog(sender.UserID, -coinToDeduct, "助力扣费", fmt.Sprintf("成功助力%d个账号", successfulHelps))
		msgs = append(msgs, fmt.Sprintf("成功助力%d个账号，扣除%d积分，剩余%d积分", successfulHelps, coinToDeduct, GetCoin(sender.UserID))) // 构建反馈消息
	} else {
		msgs = append(msgs, "未找到成功助力的账号")
	}

	// 删除 jdCookie.txt 前面的行并返回剩余行数
	cookieFilePath := ExecPath + "/scripts/huanjing/" + FileName + ".txt" // 生成 cookie 文件路径

	// 查找所有【京东账号X】的匹配项，提取最后一个X
	accountMatches := reAccount.FindAllStringSubmatch(info, -1)
	if len(accountMatches) > 0 {
		// 输出调试信息查看 accountMatches 内容
		fmt.Printf("找到的京东账号：%v\n", accountMatches)

		// 获取最后一个账号的数字，假设它是已使用的账号数量
		lastAccountIndex, _ := strconv.Atoi(accountMatches[len(accountMatches)-1][1]) // 获取最后一个【京东账号X】的X值

		// 减去 1 来少删除一个账号
		if lastAccountIndex > 1 {
			lastAccountIndex-- // 少删除一个
		}

		// 使用减去 1 的账号数字来删除
		remainingLinesCount, err := removeLinesFromFile(cookieFilePath, lastAccountIndex) // 使用修改后的账号数量来删除
		if err != nil {
			msgs = append(msgs, "删除 "+cookieFilePath+" 行数时出错："+err.Error())
		} else {
			msgs = append(msgs, fmt.Sprintf("删除已使用的%d个号，剩余%d个号", lastAccountIndex, remainingLinesCount))
		}
	} else {
		msgs = append(msgs, "未找到京东账号信息，未删除 "+FileName+".txt 行数")
	}

	msgs = append(msgs, fmt.Sprintf("===%s任务已完成===", FileName))
	return strings.Join(msgs, "\n") // 返回拼接后的消息
}

func run_ncxcx_help_Task(sender *Sender, envVars map[string]string, FileName string) string {
	logs.Info(fmt.Sprintf("开始运行%s任务", FileName)) // 使用 FileName 动态生成任务名称
	ApplyJdTaskProxyEnvs(envVars)
	jsFilePath := ExecPath + "/scripts/6dylan6_jdpro_help/" + FileName + ".js"

	if _, err := os.Stat(jsFilePath); os.IsNotExist(err) {
		logs.Error("JavaScript 文件不存在: %v", err)
		return fmt.Sprintf("执行%s任务失败：JavaScript 文件不存在", FileName) // 使用 FileName
	}

	cmd := exec.Command("node", jsFilePath)

	// 设置环境变量
	for name, value := range envVars {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", name, value))
	}

	output, err := cmd.CombinedOutput()
	logs.Info(fmt.Sprintf("%s任务脚本输出: %s", FileName, string(output)))

	if err != nil && strings.TrimSpace(string(output)) == "" {
		logs.Error("执行 JavaScript 脚本失败: %v", err)
		return fmt.Sprintf("执行%s任务失败：脚本执行错误", FileName) // 修正斜杠错误
	}

	// 传递 cookie 文件路径给 replexQuan_ncxcx_help
	return replexQuan_ncxcx_help(string(output), sender, FileName)
}

func replexQuan_ncxcx_help(info string, sender *Sender, FileName string) string {
	// 定义正则表达式，匹配成功助力的条目和使用的总账号数
	reSuccess := regexp.MustCompile(`助力成功！(\d+)`)
	reTotalAccounts := regexp.MustCompile(`共使用(\d+)个账号`)
	// 找到所有成功助力的条目
	successMatches := reSuccess.FindAllString(info, -1)
	successfulHelps := len(successMatches) // 成功助力的数量
	// 找到总账号数
	totalMatch := reTotalAccounts.FindStringSubmatch(info)
	totalAccounts := 0
	if len(totalMatch) > 1 {
		totalAccounts, _ = strconv.Atoi(totalMatch[1]) // 转换为整数
	}
	coinToDeduct := successfulHelps * 2 // 每个成功助力账号扣除10积分
	msgs := []string{}

	if successfulHelps > 0 {
		RemCoin(sender.UserID, coinToDeduct)                                                                               // 扣除积分
		RecordCoinLog(sender.UserID, -coinToDeduct, "助力扣费", fmt.Sprintf("成功助力%d个账号", successfulHelps))
		msgs = append(msgs, fmt.Sprintf("成功助力%d个账号，扣除%d积分，剩余%d积分", successfulHelps, coinToDeduct, GetCoin(sender.UserID))) // 构建反馈消息
	} else {
		msgs = append(msgs, "未找到成功助力的账号")
	}
	if totalAccounts > 0 {
		cookieFilePath := ExecPath + "/scripts/6dylan6_jdpro_help/" + FileName + ".txt" // 生成 cookie 文件路径
		remainingLinesCount, err := removeLinesFromFile(cookieFilePath, totalAccounts)  // 使用生成的路径
		if err != nil {
			msgs = append(msgs, "删除 "+cookieFilePath+" 行数时出错："+err.Error())
		} else {
			msgs = append(msgs, fmt.Sprintf("删除已使用的%d个号，剩余%d个号", totalAccounts, remainingLinesCount))
		}
	} else {
		msgs = append(msgs, "未找到总账号数信息，未删除 "+FileName+".txt 行数")
	}
	msgs = append(msgs, fmt.Sprintf("===%s任务已完成,请手动到领水滴里面领取400水滴===", FileName))
	go SendTgMsg(Config.TelegramUserID, strings.Join(msgs, "\n"))
	return strings.Join(msgs, "\n")
}

func removeLinesFromFile(filePath string, linesToRemove int) (int, error) {
	// 读取文件内容
	input, err := os.ReadFile(filePath)
	if err != nil {
		return 0, err
	}

	// 将内容按行分割
	lines := strings.Split(string(input), "\n")

	// 确保要删除的行数不超过现有行数（不包括第一行）
	if linesToRemove >= len(lines)-1 {
		linesToRemove = len(lines) - 1
	}

	// 保留第一行，删除指定的行数
	remainingLines := append([]string{lines[0]}, lines[1+linesToRemove:]...)

	// 将剩余的行写回文件
	output := strings.Join(remainingLines, "\n")
	err = os.WriteFile(filePath, []byte(output), 0644) // 使用 os.WriteFile 来写文件
	if err != nil {
		return 0, err
	}

	// 返回剩余的行数
	return len(remainingLines), nil
}



func Exportck(FileName string) {
	var msgs []string
	cks := GetJdCookies(func(sb *gorm.DB) *gorm.DB {
		return sb.Where(fmt.Sprintf("%s >= ? and %s = ?", Priority, Available), 0, True)
	})
	for _, ck := range cks {
		msgs = append(msgs, fmt.Sprintf("pt_key=%s;pt_pin=%s;", ck.PtKey, ck.PtPin))
	}
	logs.Info("导出所有账号")

	// 使用传入的 FileName 参数创建文件
	f, err := os.OpenFile(ExecPath+"/scripts/6dylan6_jdpro_help/"+FileName+".txt", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0777)
	if err != nil {
		logs.Warn(fmt.Sprintf("创建%s.txt失败，", FileName), err)
		return
	}

	join := strings.Join(msgs, "\n")
	f.WriteString(join)
	f.Close()
}

func Exportck_huanjing(FileName string) {
	var msgs []string
	cks := GetJdCookies(func(sb *gorm.DB) *gorm.DB {
		return sb.Where(fmt.Sprintf("%s >= ? and %s = ?", Priority, Available), 0, True)
	})
	for _, ck := range cks {
		msgs = append(msgs, fmt.Sprintf("pt_key=%s;pt_pin=%s;", ck.PtKey, ck.PtPin))
	}
	logs.Info("导出所有账号")

	// 使用传入的 FileName 参数创建文件
	f, err := os.OpenFile(ExecPath+"/scripts/huanjing/"+FileName+".txt", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0777)
	if err != nil {
		logs.Warn(fmt.Sprintf("创建%s.txt失败，", FileName), err)
		return
	}

	join := strings.Join(msgs, "\n")
	f.WriteString(join)
	f.Close()
}

func Delete_jdck(sender *Sender, msg chan string, cks []JdCookie) {
	for {
		n, ok := <-msg
		//说明发送方关闭了channel
		if !ok {
			break
		}
		if n == "q" {
			sender.Reply("退出登录流程")
			ckList[sender.UserID] = nil
			close(msg)
			return
		}
		num, err := strconv.Atoi(n)
		if err != nil {
			sender.Reply("请输入数字，检测到非数字输入已退出流程!")
			ckList[sender.UserID] = nil
			return
		}
		regular := `^0$|^[1-9]\d*$`
		reg := regexp.MustCompile(regular)
		if reg.MatchString(n) {
			if len(cks) <= num {
				sender.Reply("输入序列号错误，已退出！")
				ckList[sender.UserID] = nil
				return
			}
		} else {
			sender.Reply("输入序列号错误，已退出！！")
			ckList[sender.UserID] = nil
			return
		}
		ck := cks[num]
		ck.Removes(ck.PtPin)
		sender.Reply(fmt.Sprintf("已删除账号%s", cks[num].Nickname))
		ckList[sender.UserID] = nil
	}
}
