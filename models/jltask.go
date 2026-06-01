package models

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"path/filepath"
//	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"math"
	"unicode/utf8"
//	"encoding/base64" 
	"github.com/beego/beego/v2/adapter/logs"
)

// ===================== 配置结构体定义 =====================

// InputField 定义单个输入字段的配置
type InputField struct {
	Key        string                              // 字段唯一标识
	Prompt     string                              // 输入提示文本
	Validator  func(input string) (bool, string)   // 输入验证函数
	Required   bool                                // 是否必填
	TrimSpace  bool                                // 是否自动去首尾空格
	TimeoutSec int                                 // 输入超时时间（秒）
	ErrorMsg   string                              // 默认错误提示
}

// QingLongManager 青龙配置管理器
type QingLongManager struct {
	Configs map[string]*QingLongConfig // 青龙配置集合
	Default string                     // 默认配置名称
	mu      sync.RWMutex               // 读写锁
}

var qlManager = &QingLongManager{
	Configs: make(map[string]*QingLongConfig),
	Default: "ql1",
}

var inputMu sync.Mutex


// ActivityConfig 活动配置结构体
type ActivityConfig struct {
	ID                 string        // 稳定标识，使用 EnvKey 作为唯一ID，永不变化
	MenuIndex          string        // [运行时生成] 菜单展示用的连续编号 (1, 2, 3...)
	Name               string        // 活动名称
	EnvKey             string        // 青龙环境变量名
	NeedCoin           int           // 一次性扣积分值
	QingLongConfigName string        // 关联的青龙配置名称
	Guide              string        // 玩法和说明
	InputFields        []InputField  // 输入字段列表
	CKTemplate         string        // CK构建模板
	CKBuilder          func(inputs map[string]string) string
	RemarksBuilder     func(qq int, userRemarks string, inputs map[string]string) string
	ScriptPaths        struct {
		Query    string // 查询脚本路径
		Record   string // 记录CK脚本路径
		Update   string // 更新CK脚本路径
		QueryEnv string // 查询环境变量脚本路径
		Delete   string // 删除CK脚本路径
	}
	// 新增按月扣费配置
	IsMonthlyDeduct bool   // 是否按月扣积分（默认false）
	MonthlyCoin     int    // 每月扣积分值（默认200）

	// 新增按天扣费配置
	IsDailyDeduct bool   // 是否按天扣积分（默认false）
	DailyCoin     int    // 每天扣积分值
	MinDays       int    // 最小天数（按天计费时生效，默认1）

	// 👇 核心字段：控制菜单显示顺序，数值越小越靠前
	DisplayOrder int
	Enabled      bool
}

// ===================== 全局变量变更 =====================

// ActivityConfigs 活动配置切片（热加载时会更新）
var ActivityConfigs []*ActivityConfig

// activityConfigsMu 保护 ActivityConfigs 的读写锁
var activityConfigsMu sync.RWMutex

// ===================== 初始化逻辑 =====================

// InitQingLongConfigs 使用内置默认值初始化青龙配置（兼容旧调用）
func InitQingLongConfigs() {
	initQingLongConfigsFromYAML(nil)
}

// initQingLongConfigsFromYAML 从 YAML 配置初始化青龙容器
// 如果 yamlConfigs 为空或解析失败，使用内置默认值作为兜底
func initQingLongConfigsFromYAML(yamlConfigs []YAMLQingLongConfig) {
	qlManager.mu.Lock()
	defer qlManager.mu.Unlock()

	// 清空旧配置，重新加载
	qlManager.Configs = make(map[string]*QingLongConfig)

	if len(yamlConfigs) > 0 {
		// 从 YAML 配置加载
		for _, yc := range yamlConfigs {
			if yc.Name == "" || yc.Host == "" {
				continue
			}
			timeout := yc.Timeout
			if timeout <= 0 {
				timeout = 80 // 默认超时
			}
			qlManager.Configs[yc.Name] = &QingLongConfig{
				Name:         yc.Name,
				Host:         yc.Host,
				ClientID:     yc.ClientID,
				ClientSecret: yc.ClientSecret,
				Timeout:      timeout,
			}
		}
		log.Printf("青龙配置已从 YAML 加载 %d 个容器", len(qlManager.Configs))
	}

	// 如果 YAML 没有提供任何配置，使用内置默认值作为兜底
	if len(qlManager.Configs) == 0 {
		qlManager.Configs["ql1"] = &QingLongConfig{
			Name:         "ql1",
			Host:         "http://180.152.5.230:1048/",
			ClientID:     "_tGnK-FBg6y7",
			ClientSecret: "1zAj-wR-XeG2yWzXpKDotBB4",
			Timeout:      80,
		}
		qlManager.Configs["ql2"] = &QingLongConfig{
			Name:         "ql2",
			Host:         "http://180.152.5.230:1050/",
			ClientID:     "CuhZ8o_hSJy_",
			ClientSecret: "96jHSM2bqq3V8efoJWHf_Sny",
			Timeout:      80,
		}
		qlManager.Configs["ql3"] = &QingLongConfig{
			Name:         "ql3",
			Host:         "http://180.152.5.230:2041/",
			ClientID:     "5JR-3bTGevmM",
			ClientSecret: "T-549NXG8wSpgoKQHwIXxs_W",
			Timeout:      80,
		}
		log.Println("青龙配置已使用内置默认值初始化")
	}
}


// InitActivityList 已迁移至 activities.yaml 配置，通过热加载管理
// 保留空函数防止其他位置调用时编译错误
func InitActivityList() {
	// 已迁移至 activities.yaml，由 InitActivityListWithHotReload() 替代
}

// ===================== 辅助查找函数 =====================

// getActivityByID 根据活动ID或菜单编号查找活动配置
// 优先按 ID(EnvKey) 精确匹配，其次按 MenuIndex 匹配（兼容用户菜单输入）
func getActivityByID(id string) *ActivityConfig {
	activityConfigsMu.RLock()
	defer activityConfigsMu.RUnlock()
	
	// 优先按 ID(EnvKey) 精确匹配
	for _, cfg := range ActivityConfigs {
		if cfg.ID == id {
			return cfg
		}
	}
	// 其次按 MenuIndex 匹配（用户菜单输入的连续编号）
	for _, cfg := range ActivityConfigs {
		if cfg.MenuIndex == id {
			return cfg
		}
	}
	return nil
}

// getQingLongConfigForActivity 根据活动ID获取对应的青龙配置
func getQingLongConfigForActivity(activityID string) *QingLongConfig {
	qlManager.mu.RLock()
	defer qlManager.mu.RUnlock()

	if len(qlManager.Configs) == 0 {
		log.Println("警告：青龙配置未初始化，正在初始化...")
		// 注意：实际生产中建议在main中初始化，这里做简单处理
		qlManager.mu.RUnlock()
		InitQingLongConfigs()
		qlManager.mu.RLock()
	}

	config := getActivityByID(activityID)
	if config == nil || config.QingLongConfigName == "" {
		qlConfig := qlManager.Configs[qlManager.Default]
		if qlConfig == nil {
			log.Printf("错误：默认青龙配置 %s 不存在", qlManager.Default)
			for _, cfg := range qlManager.Configs {
				return cfg
			}
			log.Printf("致命错误：没有可用的青龙配置")
			return nil
		}
		return qlConfig
	}

	qlConfig, exists := qlManager.Configs[config.QingLongConfigName]
	if !exists {
		log.Printf("警告：青龙配置 %s 不存在，使用默认配置 %s", config.QingLongConfigName, qlManager.Default)
		qlConfig = qlManager.Configs[qlManager.Default]
		if qlConfig == nil {
			log.Printf("致命错误：默认青龙配置不存在")
			return nil
		}
		return qlConfig
	}

	return qlConfig
}

// ===================== 通用验证函数 =====================
func ValidatePhone(input string) (bool, string) {
	input = strings.TrimSpace(input)
	runeCount := utf8.RuneCountInString(input)
	if runeCount != 11 {
		return false, fmt.Sprintf("手机号必须是11位数字，当前输入了%d位，请重新输入！", runeCount)
	}
	for i, r := range input {
		if r >= '\uff10' && r <= '\uff19' {
			return false, fmt.Sprintf("手机号第%d位是全角数字，请切换到半角输入！", i+1)
		}
		if r < '0' || r > '9' {
			return false, fmt.Sprintf("手机号第%d位不是数字，请重新输入！", i+1)
		}
	}
	if input[0] != '1' {
		return false, "手机号必须以1开头，请重新输入！"
	}
	return true, ""
}

// ===================== 工具函数 =====================
func checkExit(input string) bool {
	return input == "q" || input == "Q"
}

// GetSortedActivityIDs 返回已启用的活动 ID 列表（稳定标识，即 EnvKey）
func GetSortedActivityIDs() []string {
	activityConfigsMu.RLock()
	defer activityConfigsMu.RUnlock()
	
	ids := make([]string, 0, len(ActivityConfigs))
	for _, cfg := range ActivityConfigs {
		if cfg.Enabled {
			ids = append(ids, cfg.ID)
		}
	}
	return ids
}

func BuildActivityMenu(promptPrefix string) string {
	activityConfigsMu.RLock()
	defer activityConfigsMu.RUnlock()
	
	menu := promptPrefix + "\n"
	// 遍历已排序的全局切片，仅展示启用的活动
	for _, cfg := range ActivityConfigs {
		if cfg.Enabled {
			menu += fmt.Sprintf("%s. %s\n", cfg.MenuIndex, cfg.Name)
		}
	}
	return menu
}

func isChineseChar(str string) bool {
	for _, r := range str {
		if r >= '\u4e00' && r <= '\u9fff' {
			return true
		}
	}
	return false
}

// getUserInputByField 收集单个字段的用户输入
func getUserInputByField(sender *Sender, msgChannel chan string, field InputField) (string, bool) {
	timeout := time.Duration(field.TimeoutSec) * time.Second
	if field.TimeoutSec == 0 {
		timeout = 60 * time.Second
	}

	errorCount := 0
	maxErrorTimes := 3

	for {
		sender.Reply(field.Prompt)
		select {
		case input := <-msgChannel:
			if field.TrimSpace {
				input = strings.TrimSpace(input)
			}

			if checkExit(input) {
				sender.Reply("程序已退出。")
				return "", true
			}

			if isChineseChar(input) {
				sender.Reply("输入内容不能包含中文字符，程序已退出。")
				return "", true
			}

			if field.Required && input == "" {
				errorCount++
				if errorCount >= maxErrorTimes {
					sender.Reply(fmt.Sprintf("连续%d次输入为空，程序已结束。", maxErrorTimes))
					return "", true
				}
				sender.Reply(fmt.Sprintf("该参数不能为空！请重新输入（剩余重试次数：%d）：", maxErrorTimes - errorCount))
				continue
			}

			valid, errMsg := field.Validator(input)
			if !valid {
				errorCount++
				if errorCount >= maxErrorTimes {
					sender.Reply(fmt.Sprintf("连续%d次输入格式错误，程序已结束。", maxErrorTimes))
					return "", true
				}
				if errMsg == "" {
					errMsg = field.ErrorMsg
				}
				sender.Reply(fmt.Sprintf("%s 请重新输入（剩余重试次数：%d）：", errMsg, maxErrorTimes - errorCount))
				continue
			}

			return input, false

		case <-time.After(timeout):
			sender.Reply("输入超时，程序已结束。")
			return "", true
		}
	}
}

func collectInputs(sender *Sender, msgChannel chan string, config *ActivityConfig) (map[string]string, bool) {
	inputs := make(map[string]string)
	for _, field := range config.InputFields {
		input, exit := getUserInputByField(sender, msgChannel, field)
		if exit {
			return nil, true
		}
		inputs[field.Key] = input
	}
	return inputs, false
}

func confirmActivityGuide(sender *Sender, msgChannel chan string, config *ActivityConfig) bool {
	guide := strings.TrimSpace(config.Guide)
	if guide == "" {
		return true
	}

	confirmPrompt := fmt.Sprintf("【%s】玩法和说明：\n%s\n\n输入 y 确认继续，输入 q 退出。", config.Name, guide)
	_, exit := getUserInputByField(sender, msgChannel, InputField{
		Key:    "guide_confirm",
		Prompt: confirmPrompt,
		Validator: func(s string) (bool, string) {
			s = strings.TrimSpace(strings.ToLower(s))
			if s == "y" {
				return true, ""
			}
			return false, "请输入 y 确认继续，或输入 q 退出！"
		},
		Required:   true,
		TrimSpace:  true,
		TimeoutSec: 120,
		ErrorMsg:   "请输入 y 确认继续，或输入 q 退出！",
	})
	return !exit
}

// ===================== 青龙查询核心函数 =====================
func queryQinglongRemarks(qq int, config *ActivityConfig) (bool, string, []string) {
	if config.ScriptPaths.QueryEnv != "" {
		return queryQinglongRemarksByScript(qq, config)
	}
	return queryQinglongRemarksByGo(qq, config)
}

func queryQinglongRemarksByScript(qq int, config *ActivityConfig) (bool, string, []string) {
	queryEnvPath := config.ScriptPaths.QueryEnv
	if queryEnvPath == "" {
		queryEnvPath = "scripts/query/query_qinglong_env.py"
	}

	uidStr := strconv.Itoa(qq)
	cmd := exec.Command("python3", queryEnvPath, uidStr, config.EnvKey)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		errMsg := strings.TrimSpace(stderr.String())
		if errMsg == "" {
			errMsg = err.Error()
		}
		sanitizedErr := SanitizeError(fmt.Errorf("%s", errMsg))
		log.Printf("[用户%d][查询青龙] 脚本执行失败：%v", qq, sanitizedErr)
		return false, fmt.Sprintf("查询服务异常：%v", sanitizedErr), nil
	}

	output := strings.TrimSpace(stdout.String())
	log.Printf("[用户%d][查询青龙] 脚本原始返回：%s", qq, output)

	if output == "not_found" {
		return false, "青龙中未找到该用户ID对应的CK，请发送【记录账号】上车", nil
	} else if strings.HasPrefix(output, "查询异常") || strings.HasPrefix(output, "参数错误") {
		sanitizedOutput := SanitizeError(fmt.Errorf("%s", output))
		log.Printf("[用户%d][查询青龙] 脚本返回异常：%v", qq, sanitizedOutput)
		return false, fmt.Sprintf("查询服务异常：%v", sanitizedOutput), nil
	}

	return true, fmt.Sprintf("共找到%s账号", config.EnvKey), []string{output}
}

func queryQinglongRemarksByGo(qq int, config *ActivityConfig) (bool, string, []string) {
	qlConfig := getQingLongConfigForActivity(config.ID)
	if qlConfig == nil {
		log.Printf("[用户%d][查询CK] 错误：无法获取青龙配置，活动ID：%s", qq, config.ID)
		return false, "获取青龙配置失败，请联系管理员", nil
	}

	client := NewQingLongClient(qlConfig)

	uidStr := strconv.Itoa(qq)
	envs, err := client.QueryEnvByRemarks(uidStr, config.EnvKey)
	if err != nil {
		sanitizedErr := SanitizeError(err)
		var errMsg string
		if strings.Contains(err.Error(), "Token") {
			errMsg = "青龙Token获取失败，请联系管理员"
		} else if strings.Contains(err.Error(), "请求失败") || strings.Contains(err.Error(), "connection refused") {
			errMsg = "青龙服务器连接失败，请稍后重试"
		} else if strings.Contains(err.Error(), "解析") {
			errMsg = "青龙数据解析失败，请联系管理员"
		} else {
			errMsg = fmt.Sprintf("查询青龙账号失败：%v", sanitizedErr)
		}
		log.Printf("[用户%d][查询CK] 错误：%v，活动ID：%s", qq, sanitizedErr, config.ID)
		return false, errMsg, nil
	}

	if len(envs) == 0 {
		log.Printf("[用户%d][查询CK] 未找到匹配的CK，活动ID：%s", qq, config.ID)
		return false, "青龙中未找到该用户ID对应的CK，请发送【记录账号】上车", nil
	}

	jsonData, err := json.Marshal(envs)
	if err != nil {
		log.Printf("[用户%d][查询CK] JSON序列化失败：%v，原始数据：%+v", qq, err, envs)
		return false, "青龙数据处理失败，请联系管理员", nil
	}

	log.Printf("[用户%d][查询CK] 成功找到%d个账号，活动ID：%s", qq, len(envs), config.ID)
	return true, fmt.Sprintf("共找到%d个%s账号", len(envs), config.EnvKey), []string{string(jsonData)}
}

// ===================== 脚本执行通用函数 =====================
func executeScript(sender *Sender, cmdPath string, args ...string) (string, error) {
	cmd := exec.Command(cmdPath, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		stderrStr := strings.TrimSpace(stderr.String())
		stdoutStr := strings.TrimSpace(stdout.String())
		logs.Warn("脚本执行失败 | 命令: %s %v | 错误: %v | stderr: %s | stdout: %s", cmdPath, args, err, stderrStr, stdoutStr)
		userMsg := stderrStr
		if userMsg == "" {
			userMsg = stdoutStr
		}
		if userMsg == "" {
			userMsg = err.Error()
		}
		sanitizedErr := SanitizeError(fmt.Errorf("%s", userMsg))
		return "", fmt.Errorf("脚本执行异常：%v", sanitizedErr)
	}
	return stdout.String(), nil
}

// ===================== 记录CK =====================
func handleRecordCKByGo(qq int, ckValue, finalRemarks, envKey string, config *ActivityConfig) (string, error) {
	duplicate, err := CheckDuplicateRemarksDB(finalRemarks, envKey)
	if err != nil {
		return "", fmt.Errorf("检查重复备注失败：%v", err)
	}

	if duplicate {
		return "记录失败，已有相同备注，需换一个备注.", nil
	}

	expireDate := ""
	if config.IsMonthlyDeduct || config.IsDailyDeduct {
		d, ok := ParseRemarksDate(finalRemarks)
		if ok {
			expireDate = d.Format(DateLayout)
		}
	}

	project := &ActivityProject{
		ActivityID:         config.ID,
		ActivityName:       config.Name,
		EnvKey:             envKey,
		EnvValue:           ckValue,
		Remarks:            finalRemarks,
		UserNumber:         qq,
		QingLongConfigName: config.QingLongConfigName,
		Status:             0,
		ExpireDate:         expireDate,
		IsMonthlyDeduct:    config.IsMonthlyDeduct,
		MonthlyCoin:        config.MonthlyCoin,
		IsDailyDeduct:      config.IsDailyDeduct,
		DailyCoin:          config.DailyCoin,
		SyncStatus:         "pending",
	}
	if config.IsDailyDeduct && config.MinDays > 0 {
		minDays := config.MinDays
		project.MinDays = &minDays
	}
	if !config.IsMonthlyDeduct && !config.IsDailyDeduct {
		project.NeedCoin = config.NeedCoin
	}

	if err := CreateActivityProject(project); err != nil {
		return "", fmt.Errorf("保存到数据库失败：%v", err)
	}

	go TriggerSync(project.ID)

	return "记录成功", nil
}

func HandleRecordCK(sender *Sender) interface{} {
	msgChannel := make(chan string)
	qq := sender.UserID
	inputMu.Lock()
	inputList[qq] = msgChannel
	inputMu.Unlock()

	go func() {
		defer delete(inputList, qq)

		menu := BuildActivityMenu("请选择活动代号（输入数字）任意地方输入q可退出程序：")
		envID, exit := getUserInputByField(sender, msgChannel, InputField{
			Key:        "env_id",
			Prompt:     menu,
			Validator: func(s string) (bool, string) {
				if getActivityByID(s) == nil {
					return false, "活动代号不存在，请输入菜单中的有效数字！"
				}
				return true, ""
			},
			Required:   true,
			TrimSpace:  false,
			TimeoutSec: 60,
			ErrorMsg:   "活动代号输入错误，请重新输入！",
		})
		if exit {
			return
		}

		config := getActivityByID(envID)
		if config == nil {
			sender.Reply("活动配置不存在")
			return
		}

		if !confirmActivityGuide(sender, msgChannel, config) {
			return
		}

		inputs, exit := collectInputs(sender, msgChannel, config)
		if exit {
			return
		}

		userRemarks, exit := getUserInputByField(sender, msgChannel, InputField{
			Key:        "remarks",
			Prompt:     "请输入唯一备注名（无需加ID，系统自动拼接）:",
			Validator: func(s string) (bool, string) {
				if len(s) == 0 {
					return false, "备注名不能为空"
				}
				if strings.Contains(s, "/") {
					return false, "备注名不允许包含/符号"
				}
				return true, ""
			},
			Required:   true,
			TrimSpace:  true,
			TimeoutSec: 60,
			ErrorMsg:   "备注名输入错误",
		})
		if exit {
			return
		}

		var months int
		var expireDate string
		if config.IsDailyDeduct {
			daysStr, exit := getUserInputByField(sender, msgChannel, InputField{
				Key:        "days",
				Prompt:     fmt.Sprintf("该活动按天扣费每天【%d】积分，你当前积分还剩【%d】，请输入授权天数（如7/15/30）：", config.DailyCoin, GetCoin(sender.UserID)),
				Validator: func(s string) (bool, string) {
					num, err := strconv.Atoi(s)
					if err != nil {
						return false, "请输入有效的数字！"
					}
					if num < 1 || num > 365 {
						return false, "天数需在1-365之间！"
					}
					return true, ""
				},
				Required:   true,
				TrimSpace:  false,
				TimeoutSec: 60,
				ErrorMsg:   "天数输入错误，请重新输入！",
			})
			if exit {
				return
			}
			months, _ = strconv.Atoi(daysStr)
			expireDate = GenerateExpireDateFromDays(months)
		} else if config.IsMonthlyDeduct {
			monthStr, exit := getUserInputByField(sender, msgChannel, InputField{
				Key:        "months",
				Prompt:     fmt.Sprintf("该活动按月扣费每月【%d】积分，你当前积分还剩【%d】，请输入授权时长（月数，如1/3/6）：", config.MonthlyCoin, GetCoin(sender.UserID)),
				Validator: func(s string) (bool, string) {
					num, err := strconv.Atoi(s)
					if err != nil {
						return false, "请输入有效的数字！"
					}
					if num < 1 || num > 12 {
						return false, "时长需在1-12个月之间！"
					}
					return true, ""
				},
				Required:   true,
				TrimSpace:  false,
				TimeoutSec: 60,
				ErrorMsg:   "时长输入错误，请重新输入！",
			})
			if exit {
				return
			}
			months, _ = strconv.Atoi(monthStr)
			expireDate = GenerateExpireDate(months)
		}

		ckValue := config.CKBuilder(inputs)
		finalRemarks := config.RemarksBuilder(qq, userRemarks, inputs)
		if config.IsMonthlyDeduct || config.IsDailyDeduct {
			finalRemarks = fmt.Sprintf("%s/%s", finalRemarks, expireDate)
		}

		totalCoin := 0
		if config.IsDailyDeduct {
			totalCoin = config.DailyCoin * months
		} else if config.IsMonthlyDeduct {
			totalCoin = config.MonthlyCoin * months
		} else {
			totalCoin = config.NeedCoin
		}

		userCoin := GetCoin(sender.UserID)
		if userCoin < totalCoin {
			sender.Reply(fmt.Sprintf("积分不足！%s需要%d积分，你当前有%d积分。", config.Name, totalCoin, userCoin))
			return
		}

		var output string
		var err error

		if config.ScriptPaths.Record != "" {
			output, err = executeScript(sender, "python3", config.ScriptPaths.Record, ckValue, finalRemarks, config.EnvKey)
		} else {
			output, err = handleRecordCKByGo(qq, ckValue, finalRemarks, config.EnvKey, config)
		}

		if err != nil {
			sender.Reply(fmt.Sprintf("操作失败：%v", err))
			return
		}

		if strings.Contains(output, "记录成功") {
			RemCoin(sender.UserID, totalCoin)
			RecordCoinLog(sender.UserID, -totalCoin, "上车扣费", fmt.Sprintf("%s上车", config.Name))
			if config.IsDailyDeduct {
				sender.Reply(fmt.Sprintf("添加%s账号成功！已扣除%d积分（%d天×%d积分/天），剩余%d积分。授权有效期至：%s",
					config.Name, totalCoin, months, config.DailyCoin, userCoin-totalCoin, expireDate))
			} else if config.IsMonthlyDeduct {
				sender.Reply(fmt.Sprintf("添加%s账号成功！已扣除%d积分（%d个月×%d积分/月），剩余%d积分。授权有效期至：%s",
					config.Name, totalCoin, months, config.MonthlyCoin, userCoin-totalCoin, expireDate))
			} else {
				sender.Reply(fmt.Sprintf("添加%s账号成功！已扣除%d积分，剩余%d积分。",
					config.Name, totalCoin, userCoin-totalCoin))
			}
		} else {
			sender.Reply(fmt.Sprintf("记录失败：%s", output))
		}
	}()

	return nil
}

// ===================== 更新CK =====================
func handleUpdateCKByGo(qq int, ckValue, remarks, envKey string, config *ActivityConfig) (string, error) {
	project, err := GetActivityProjectByRemarks(config.ID, remarks, envKey)
	if err != nil {
		log.Printf("[用户%d][更新CK] 查询账号失败，备注：%s，错误：%v", qq, remarks, err)
		if strings.Contains(err.Error(), "record not found") {
			return "", fmt.Errorf("未找到该账号信息，无法更新")
		}
		return "", fmt.Errorf("查询账号信息失败：%v", err)
	}

	if project.Status != 0 {
		log.Printf("[用户%d][更新CK] 账号处于禁用状态（Status=%d），拒绝更新，备注：%s，ID=%d",
			qq, project.Status, remarks, project.ID)
		return "", fmt.Errorf("该账号已禁用（状态码：%d），不允许更新！请先发送【记录授权】", project.Status)
	}

	if ckValue == "" {
		log.Printf("[用户%d][更新CK] 提交的CK值为空，备注：%s", qq, remarks)
		return "", fmt.Errorf("提交的CK值不能为空")
	}
	if ckValue == project.EnvValue {
		log.Printf("[用户%d][更新CK] CK值未变化，备注：%s", qq, remarks)
		return "更新失败：提交的CK与原有CK相同", nil
	}

	project.EnvValue = ckValue
	project.SyncStatus = "pending_update"
	project.SyncError = ""
	if err := UpdateActivityProject(project); err != nil {
		log.Printf("[用户%d][更新CK] 更新数据库失败，备注：%s，错误：%v", qq, remarks, err)
		return "", fmt.Errorf("更新CK失败：%v", err)
	}

	go TriggerSync(project.ID)

	log.Printf("[用户%d][更新CK] 更新成功，备注：%s", qq, remarks)
	return "环境变量值更新成功", nil
}

func HandleUpdateCK(sender *Sender) interface{} {
	msgChannel := make(chan string)
	qq := sender.UserID
	inputMu.Lock()
	inputList[qq] = msgChannel
	inputMu.Unlock()

	go func() {
		defer delete(inputList, qq)

		menu := BuildActivityMenu("请选择要更新的活动代号（输入数字）任意地方输入q可退出程序：")
		envID, exit := getUserInputByField(sender, msgChannel, InputField{
			Key:        "env_id",
			Prompt:     menu,
			Validator: func(s string) (bool, string) {
				if getActivityByID(s) == nil {
					return false, "活动代号不存在，请输入菜单中的有效数字！"
				}
				return true, ""
			},
			Required:   true,
			TrimSpace:  false,
			TimeoutSec: 60,
			ErrorMsg:   "活动代号输入错误，请重新输入！",
		})
		if exit {
			log.Printf("[用户%d][更新CK] 选择活动阶段主动退出", qq)
			return
		}

		config := getActivityByID(envID)
		if config == nil {
			sender.Reply("活动配置不存在，请联系管理员")
			return
		}

		sender.Reply(fmt.Sprintf("正在从数据库中查询你的%s CK，请稍候...", config.EnvKey))

		projects, dbErr := GetActivityProjectsByUserAndEnv(qq, config.ID, config.EnvKey)
		if dbErr != nil {
			sender.Reply("数据库查询失败，请联系管理员")
			log.Printf("[用户%d][更新CK] 数据库查询失败：%v", qq, dbErr)
			return
		}

		var remarksList []string
		var displayList []string
		ckStatusMap := make(map[string]int)

		if len(projects) == 0 {
			sender.Reply("未找到可更新的账号")
			log.Printf("[用户%d][更新CK] CK数据为空", qq)
			return
		}

		log.Printf("[用户%d][更新CK] 从数据库找到%d条记录", qq, len(projects))
		for _, project := range projects {
			if project.Remarks == "" {
				continue
			}
			statusInt := project.Status
			if statusInt != 0 {
				statusInt = 1
			}
			remarksList = append(remarksList, project.Remarks)
			ckStatusMap[project.Remarks] = statusInt

			displayName := GetFirstRemarkParam(project.Remarks)
			if statusInt == 1 {
				displayName += " 【禁用】"
				log.Printf("[用户%d][更新CK] 发现禁用账号：%s（DB ID=%d，Status=%d）", qq, displayName, project.ID, project.Status)
			}
			displayList = append(displayList, displayName)
		}

		if len(remarksList) == 0 {
			sender.Reply("未找到可更新的账号")
			log.Printf("[用户%d][更新CK] 过滤后无可用账号", qq)
			return
		}

		var selectedRemarks string
		if len(remarksList) > 1 {
			var accountList string
			for i, displayName := range displayList {
				accountList += fmt.Sprintf("%d. %s\n", i+1, displayName)
			}
			selectIndexStr, exit := getUserInputByField(sender, msgChannel, InputField{
				Key:        "select_index",
				Prompt:     fmt.Sprintf("共找到%d个%s账号，请选择要更新的序号：\n%s请输入要更新的账号序号（输入数字）：", len(remarksList), config.EnvKey, accountList),
				Validator: func(s string) (bool, string) {
					idx, err := strconv.Atoi(s)
					if err != nil {
						return false, "序号必须是数字，请重新输入！"
					}
					if idx < 1 || idx > len(remarksList) {
						return false, fmt.Sprintf("序号必须在1-%d之间，请重新输入！", len(remarksList))
					}
					selectedRemark := remarksList[idx-1]
					status := ckStatusMap[selectedRemark]
					if status == 1 {
						return false, fmt.Sprintf("该账号【%s】已禁用，不允许更新！请先发送【记录授权】重新启用", GetFirstRemarkParam(selectedRemark))
					}
					return true, ""
				},
				Required:   true,
				TrimSpace:  false,
				TimeoutSec: 60,
				ErrorMsg:   "序号输入错误，请重新输入！",
			})
			if exit {
				log.Printf("[用户%d][更新CK] 选择账号阶段主动退出", qq)
				return
			}

			selectIndex, _ := strconv.Atoi(selectIndexStr)
			selectedRemarks = remarksList[selectIndex-1]
			sender.Reply(fmt.Sprintf("你选择更新的账号是：%s", displayList[selectIndex-1]))
		} else {
			selectedRemarks = remarksList[0]
			status := ckStatusMap[selectedRemarks]
			if status == 1 {
				sender.Reply(fmt.Sprintf("该账号【%s】已禁用，不允许更新！请先发送【记录授权】重新启用", displayList[0]))
				log.Printf("[用户%d][更新CK] 唯一账号处于禁用状态，禁止更新，备注：%s", qq, selectedRemarks)
				return
			}
			sender.Reply(fmt.Sprintf("共找到1个%s账号，自动选中：%s", config.EnvKey, displayList[0]))
		}

		inputs, exit := collectInputs(sender, msgChannel, config)
		if exit {
			log.Printf("[用户%d][更新CK] 收集CK参数阶段主动退出", qq)
			return
		}

		ckValue := config.CKBuilder(inputs)

		finalStatus := ckStatusMap[selectedRemarks]
		if finalStatus == 1 {
			sender.Reply(fmt.Sprintf("更新失败：该账号【%s】已禁用，不允许更新！", GetFirstRemarkParam(selectedRemarks)))
			log.Printf("[用户%d][更新CK] 最终校验发现禁用账号，拒绝更新，备注：%s", qq, selectedRemarks)
			return
		}

		sender.Reply(fmt.Sprintf("正在更新【%s】的CK，请稍候...", GetFirstRemarkParam(selectedRemarks)))

		var output string
		var err2 error

		if config.ScriptPaths.Update != "" {
			output, err2 = executeScript(sender, "python3", config.ScriptPaths.Update, ckValue, selectedRemarks, config.EnvKey)
		} else {
			output, err2 = handleUpdateCKByGo(qq, ckValue, selectedRemarks, config.EnvKey, config)
		}

		if err2 != nil {
			sender.Reply(err2.Error())
			log.Printf("[用户%d][更新CK] 最终失败：%v，备注：%s", qq, err2, selectedRemarks)
			return
		}

		if strings.Contains(output, "更新成功") {
			sender.Reply(fmt.Sprintf("【%s】CK更新成功！", GetFirstRemarkParam(selectedRemarks)))
		} else {
			sender.Reply(fmt.Sprintf("更新提示：%s", output))
		}
	}()

	return nil
}

// ===================== 删除CK =====================
func handleDeleteCKByGo(qq int, remarks, envKey string, config *ActivityConfig) (string, error) {
	project, err := GetActivityProjectByRemarks(config.ID, remarks, envKey)
	if err != nil {
		if strings.Contains(err.Error(), "record not found") {
			return "", fmt.Errorf("未找到该备注对应的CK记录")
		}
		log.Printf("[用户%d][删除CK] 查询账号失败，备注：%s，错误：%v", qq, remarks, err)
		return "", fmt.Errorf("查询账号信息失败：%v", err)
	}

	if err := SoftDeleteActivityProject(project.ID); err != nil {
		log.Printf("[用户%d][删除CK] 删除失败，备注：%s，错误：%v", qq, remarks, err)
		return "", fmt.Errorf("删除账号失败：%v", err)
	}

	go TriggerSync(project.ID)

	return "环境变量删除成功", nil
}






//##新增月付费用户删除退换积分
func HandleDeleteCK(sender *Sender) interface{} {
	msgChannel := make(chan string)
	qq := sender.UserID
	inputMu.Lock()
	inputList[qq] = msgChannel
	inputMu.Unlock()

	go func() {
		defer delete(inputList, qq)

		menu := BuildActivityMenu("请选择要删除的活动代号（输入数字）任意地方输入q可退出程序：")
		envID, exit := getUserInputByField(sender, msgChannel, InputField{
			Key:        "env_id",
			Prompt:     menu,
			Validator: func(s string) (bool, string) {
				if getActivityByID(s) == nil {
					return false, "活动代号不存在，请输入菜单中的有效数字！"
				}
				return true, ""
			},
			Required:   true,
			TrimSpace:  false,
			TimeoutSec: 60,
			ErrorMsg:   "活动代号输入错误，请重新输入！",
		})
		if exit {
			return
		}

		config := getActivityByID(envID)
		if config == nil {
			sender.Reply("活动配置不存在")
			return
		}

		sender.Reply(fmt.Sprintf("正在从数据库中查询你的%s CK，请稍候...", config.EnvKey))

		projects, dbErr := GetActivityProjectsByUserAndEnv(qq, config.ID, config.EnvKey)
		if dbErr != nil {
			sender.Reply("数据库查询失败，请联系管理员")
			log.Printf("[用户%d][删除CK] 数据库查询失败：%v", qq, dbErr)
			return
		}

		var remarksList []string
		var displayList []string

		if len(projects) == 0 {
			sender.Reply("未找到可删除的账号")
			return
		}

		for _, project := range projects {
			if project.Remarks != "" {
				remarksList = append(remarksList, project.Remarks)
				displayList = append(displayList, GetFirstRemarkParam(project.Remarks))
			}
		}

		var selectedRemarks string
		if len(remarksList) > 1 {
			var accountList string
			for i, displayName := range displayList {
				accountList += fmt.Sprintf("%d. %s\n", i+1, displayName)
			}
			selectIndexStr, exit := getUserInputByField(sender, msgChannel, InputField{
				Key:        "select_index",
				Prompt:     fmt.Sprintf("共找到%d个%s账号，请选择要删除的序号：\n%s请输入要删除的账号序号（输入数字）：", len(remarksList), config.EnvKey, accountList),
				Validator: func(s string) (bool, string) {
					idx, err := strconv.Atoi(s)
					if err != nil {
						return false, "序号必须是数字，请重新输入！"
					}
					if idx < 1 || idx > len(remarksList) {
						return false, fmt.Sprintf("序号必须在1-%d之间，请重新输入！", len(remarksList))
					}
					return true, ""
				},
				Required:   true,
				TrimSpace:  false,
				TimeoutSec: 60,
				ErrorMsg:   "序号输入错误，请重新输入！",
			})
			if exit {
				return
			}

			selectIndex, _ := strconv.Atoi(selectIndexStr)
			selectedRemarks = remarksList[selectIndex-1]
			sender.Reply(fmt.Sprintf("你选择删除的账号是：%s", displayList[selectIndex-1]))
		} else {
			selectedRemarks = remarksList[0]
			sender.Reply(fmt.Sprintf("共找到1个%s账号，自动选中：%s", config.EnvKey, displayList[0]))
		}

		// ===================== 月付费/按天计费退还积分逻辑 =====================
		var returnCoin int = 0
		confirmPrompt := fmt.Sprintf("确认要删除【%s】这个账号吗？（输入y确认，其他字符取消）", GetFirstRemarkParam(selectedRemarks))

		selectedProjectMonthlyCoin := config.MonthlyCoin
		selectedProjectDailyCoin := config.DailyCoin
		selectedProjectNeedCoin := config.NeedCoin
		var selectedProject *ActivityProject
		for i, p := range projects {
			if p.Remarks == selectedRemarks {
				if p.MonthlyCoin > 0 {
					selectedProjectMonthlyCoin = p.MonthlyCoin
				}
				if p.DailyCoin > 0 {
					selectedProjectDailyCoin = p.DailyCoin
				}
				selectedProjectNeedCoin = p.NeedCoin
				selectedProject = &projects[i]
				break
			}
		}

		if (config.IsMonthlyDeduct || config.IsDailyDeduct) && selectedProjectNeedCoin == 0 && selectedProject != nil {
			paidDays := CalcPaidRemainingDays(selectedProject)

			if config.IsDailyDeduct && selectedProjectDailyCoin > 0 {
				returnCoin = selectedProjectDailyCoin * paidDays
				confirmPrompt = fmt.Sprintf("【温馨提示】当前为按天计费活动\n删除后将退还积分：%d分\n确认删除【%s】？（输入y确认，其他字符取消）",
					returnCoin, GetFirstRemarkParam(selectedRemarks))
			} else if selectedProjectMonthlyCoin > 0 {
				returnCoin = int(math.Round(float64(selectedProjectMonthlyCoin) * float64(paidDays) / 30))
				confirmPrompt = fmt.Sprintf("【温馨提示】当前为月付费活动\n删除后将退还积分：%d分\n确认删除【%s】？（输入y确认，其他字符取消）",
					returnCoin, GetFirstRemarkParam(selectedRemarks))
			}
		}

		// 确认删除
		confirm, exit := getUserInputByField(sender, msgChannel, InputField{
			Key:        "confirm_delete",
			Prompt:     confirmPrompt,
			Validator: func(s string) (bool, string) {
				if checkExit(s) {
					return false, "程序已退出"
				}
				return true, ""
			},
			Required:   true,
			TrimSpace:  true,
			TimeoutSec: 30,
			ErrorMsg:   "输入错误，请重新输入！",
		})
		if exit {
			return
		}
		if strings.ToLower(confirm) != "y" {
			sender.Reply("已取消删除操作")
			return
		}

		sender.Reply(fmt.Sprintf("正在删除【%s】的CK，请稍候...", GetFirstRemarkParam(selectedRemarks)))

		var output string
		var err error

		if config.ScriptPaths.Delete != "" {
			output, err = executeScript(sender, "python3", config.ScriptPaths.Delete, selectedRemarks, config.EnvKey)
		} else {
			output, err = handleDeleteCKByGo(qq, selectedRemarks, config.EnvKey, config)
		}

		if err != nil {
			sender.Reply(err.Error())
			return
		}

		// 删除成功后执行积分退还
		if strings.Contains(output, "删除成功") {
			if (config.IsMonthlyDeduct || config.IsDailyDeduct) && returnCoin > 0 {
				AdddCoin(qq, returnCoin)
				RecordCoinLog(qq, returnCoin, "退还", fmt.Sprintf("删除%s退还", config.Name))
				sender.Reply(fmt.Sprintf("【%s】删除成功！积分已退还：+%d分", GetFirstRemarkParam(selectedRemarks), returnCoin))
			} else {
				sender.Reply(fmt.Sprintf("【%s】删除成功！", GetFirstRemarkParam(selectedRemarks)))
			}
		} else {
			sender.Reply(fmt.Sprintf("删除提示：%s", output))
		}
	}()

	return nil
}
// ===================== 查询CK记录 =====================
func HandleQueryRecord(sender *Sender) interface{} {
	qq := sender.UserID
	msgChannel := make(chan string)
	inputMu.Lock()
	inputList[qq] = msgChannel
	inputMu.Unlock()

	go func() {
		defer func() {
			close(msgChannel)
			delete(inputList, qq)
		}()

		logPrefix := fmt.Sprintf("[用户%d][查询记录] ", qq)

		getInput := func(prompt string) string {
			sender.Reply(prompt)
			select {
			case input := <-msgChannel:
				input = strings.TrimSpace(input)
				if input == "q" {
					sender.Reply("你已选择退出，操作终止")
					log.Printf("%s用户主动退出查询", logPrefix)
					return ""
				}
				return input
			case <-time.After(30 * time.Second):
				sender.Reply("输入超时，操作终止")
				log.Printf("%s用户输入超时，终止操作", logPrefix)
				return ""
			}
		}

		menu := BuildActivityMenu("请输入数字选择查询项目（输入q可退出，30秒未输入超时）：")
		optSelect := getInput(menu)
		if optSelect == "" {
			return
		}

		config := getActivityByID(optSelect)
		if config == nil {
			sender.Reply("选择错误！请输入菜单中的有效数字")
			log.Printf("%s用户选择错误：输入了非有效数字", logPrefix)
			return
		}

		sender.Reply(fmt.Sprintf("已选择【%s】，正在从数据库获取账号信息...", config.Name))

		projects, err := GetActivityProjectsByUserAndEnv(qq, config.ID, config.EnvKey)
		if err != nil {
			sender.Reply(fmt.Sprintf("【%s】查询失败：数据库错误，请联系管理员", config.Name))
			log.Printf("%s数据库查询失败：%v", logPrefix, err)
			return
		}

		if len(projects) == 0 {
			sender.Reply(fmt.Sprintf("查询失败：未找到【%s】绑定账号，请先发送【记录授权】上车", config.Name))
			return
		}

		accountCount := len(projects)

		sender.Reply(fmt.Sprintf("检测到你有【%d】个【%s】账号，即将为你依次执行查询...", accountCount, config.Name))

		scriptPath := config.ScriptPaths.Query
		if scriptPath == "" {
			sender.Reply(fmt.Sprintf("【%s】暂无查询脚本，无法执行查询", config.Name))
			return
		}
		
		scriptExt := filepath.Ext(scriptPath)
		var execCmd string
		switch scriptExt {
		case ".js":
			execCmd = "node"
		case ".py":
			execCmd = "python3"
		default:
			sender.Reply("不支持的脚本类型，请联系管理员")
			return
		}


for index, project := range projects {
    accountNo := index + 1

    mainRemark := GetFirstRemarkParam(project.Remarks)
    if mainRemark == "" {
        mainRemark = "未知备注"
    }

    expireTime := project.ExpireDate

    if project.Status != 0 {
        sender.Reply(fmt.Sprintf("⚠️ 第%d个账号【%s】已被禁用（可能授权已过期），无法查询，请重新发送【记录授权】续费！", accountNo, mainRemark))
        log.Printf("%s第%d个账号【%s】状态为禁用（Status=%d），跳过查询", logPrefix, accountNo, mainRemark, project.Status)
        continue
    }

    if (config.IsMonthlyDeduct || config.IsDailyDeduct) && expireTime != "" {
        expireTimeObj, parseErr := time.ParseInLocation("2006-01-02", expireTime, time.Local)
        if parseErr == nil {
            expireThreshold := time.Date(expireTimeObj.Year(), expireTimeObj.Month(), expireTimeObj.Day(), 0, 0, 0, 0, time.Local).AddDate(0, 0, 1)
            if time.Now().After(expireThreshold) || time.Now().Equal(expireThreshold) {
                sender.Reply(fmt.Sprintf("⚠️ 第%d个账号【%s】授权已过期（过期时间：%s），无法查询，请发送【记录授权】续费！", accountNo, mainRemark, expireTime))
                log.Printf("%s第%d个账号【%s】已过期（ExpireDate=%s），跳过查询", logPrefix, accountNo, mainRemark, expireTime)
                continue
            }
        }
    }

    ckValue := project.EnvValue
    if ckValue == "" {
        sender.Reply(fmt.Sprintf("第%d个账号查询失败\n错误：CK数据为空（可能同步异常），请联系管理员", accountNo))
        log.Printf("%s第%d个账号CK值为空（DB ID=%d）", logPrefix, accountNo, project.ID)
        continue
    }

    log.Printf("%s执行第%d个账号脚本：%s %s", logPrefix, accountNo, execCmd, scriptPath)

    output, err := executeScript(sender, execCmd, scriptPath, ckValue)

    if err != nil {
        sender.Reply(fmt.Sprintf("第%d个账号查询失败：%v", accountNo, err))
        log.Printf("%s第%d个账号查询失败 - 错误：%v", logPrefix, accountNo, err)
    } else if output == "" {
        noResultMsg := fmt.Sprintf("第%d个账号备注：【%s】\n查询完成，暂无查询结果", accountNo, mainRemark)
        if expireTime != "" {
            noResultMsg = fmt.Sprintf("第%d个账号备注：【%s】\n授权过期时间：【%s】\n查询完成，暂无查询结果", accountNo, mainRemark, expireTime)
        }
        sender.Reply(noResultMsg)
    } else {
        prefixMsg := fmt.Sprintf("第%d个账号备注：【%s】\n", accountNo, mainRemark)
        if expireTime != "" {
            prefixMsg += fmt.Sprintf("授权过期时间：【%s】\n", expireTime)
        }
        finalOutput := prefixMsg + output
        sender.Reply(fmt.Sprintf("\n%s", finalOutput))
    }

    if accountNo < accountCount {
        time.Sleep(150 * time.Millisecond)
    }
}

		sender.Reply(fmt.Sprintf("✅ 【%s】共%d个账号查询全部完成！", config.Name, accountCount))
		log.Printf("%s%s共%d个账号查询完成", logPrefix, config.Name, accountCount)
	}()

	return nil
}


func HandleAuthorizeCK(sender *Sender) interface{} {
	msgChannel := make(chan string)
	qq := sender.UserID

	// 将当前QQ和信道存入全局映射，以便 getUserInputByField 获取输入
	inputList[qq] = msgChannel

	go func() {
		// 确保函数退出时清理全局映射，防止内存泄漏
		defer delete(inputList, qq)

		var menu string
		menu = "请选择要授权的活动代号（输入数字）任意地方输入q可退出程序：\n"
		hasRenewable := false

		activityConfigsMu.RLock()
		for _, cfg := range ActivityConfigs {
			if cfg.IsDailyDeduct && cfg.Enabled {
				menu += fmt.Sprintf("【%s】、 %s（每天%d积分）\n", cfg.MenuIndex, cfg.Name, cfg.DailyCoin)
				hasRenewable = true
			} else if cfg.IsMonthlyDeduct && cfg.Enabled {
				menu += fmt.Sprintf("【%s】、 %s（每月%d积分）\n", cfg.MenuIndex, cfg.Name, cfg.MonthlyCoin)
				hasRenewable = true
			}
		}
		activityConfigsMu.RUnlock()

		if !hasRenewable {
			sender.Reply("暂无支持授权续费的活动！")
			return
		}

		envID, exit := getUserInputByField(sender, msgChannel, InputField{
			Key:        "env_id",
			Prompt:     menu,
			Validator: func(s string) (bool, string) {
				cfg := getActivityByID(s)
				if cfg == nil || (!cfg.IsMonthlyDeduct && !cfg.IsDailyDeduct) {
					return false, "活动代号不存在或不支持授权续费，请输入菜单中的有效数字！"
				}
				return true, ""
			},
			Required:   true,
			TrimSpace:  false,
			TimeoutSec: 60,
			ErrorMsg:   "活动代号输入错误，请重新输入！",
		})
		if exit {
			return
		}

		config := getActivityByID(envID)
		if config == nil {
			sender.Reply("活动配置不存在")
			return
		}

		sender.Reply(fmt.Sprintf("正在从数据库中查询你的%s CK，请稍候...", config.EnvKey))

		projects, dbErr := GetActivityProjectsByUserAndEnv(qq, config.ID, config.EnvKey)
		if dbErr != nil {
			sender.Reply("数据库查询失败，请联系管理员")
			log.Printf("[用户%d][续费授权] 数据库查询失败：%v", qq, dbErr)
			return
		}

		var remarksList []string
		var displayList []string

		if len(projects) == 0 {
			sender.Reply("未找到可授权的账号")
			return
		}

		for _, project := range projects {
			if project.Remarks != "" {
				remarksList = append(remarksList, project.Remarks)
				displayList = append(displayList, GetFirstRemarkParam(project.Remarks))
			}
		}

		// 3. 选择账号（如果有多个）
		var selectedRemarks string
		if len(remarksList) > 1 {
			var accountList string
			for i, displayName := range displayList {
				accountList += fmt.Sprintf("%d. %s\n", i+1, displayName)
			}
			selectIndexStr, exit := getUserInputByField(sender, msgChannel, InputField{
				Key:        "select_index",
				Prompt:     fmt.Sprintf("共找到%d个%s账号，请选择要授权的序号：\n%s请输入要授权的账号序号（输入数字）：", len(remarksList), config.EnvKey, accountList),
				Validator: func(s string) (bool, string) {
					idx, err := strconv.Atoi(s)
					if err != nil {
						return false, "序号必须是数字，请重新输入！"
					}
					if idx < 1 || idx > len(remarksList) {
						return false, fmt.Sprintf("序号必须在1-%d之间，请重新输入！", len(remarksList))
					}
					return true, ""
				},
				Required:   true,
				TrimSpace:  false,
				TimeoutSec: 60,
				ErrorMsg:   "序号输入错误，请重新输入！",
			})
			if exit {
				return
			}

			selectIndex, _ := strconv.Atoi(selectIndexStr)
			selectedRemarks = remarksList[selectIndex-1]
			sender.Reply(fmt.Sprintf("你选择授权的账号是：%s", displayList[selectIndex-1]))
		} else {
			selectedRemarks = remarksList[0]
			sender.Reply(fmt.Sprintf("共找到1个%s账号，自动选中：%s", config.EnvKey, displayList[0]))
		}

		// 4. 获取授权时长
		var months int
		var totalCoin int
		if config.IsDailyDeduct {
			minDays := config.MinDays
			if minDays < 1 {
				minDays = 1
			}
			daysStr, exit := getUserInputByField(sender, msgChannel, InputField{
				Key:        "days",
				Prompt:     fmt.Sprintf("该活动每天扣费【%d】积分，您当前的积分还剩【%d】，请输入授权天数（最少%d天，如%d/7/30）：", config.DailyCoin, GetCoin(sender.UserID), minDays, minDays),
				Validator: func(s string) (bool, string) {
					num, err := strconv.Atoi(s)
					if err != nil {
						return false, "请输入有效的数字！"
					}
					if num < minDays || num > 365 {
						return false, fmt.Sprintf("天数需在%d-365之间！", minDays)
					}
					return true, ""
				},
				Required:   true,
				TrimSpace:  false,
				TimeoutSec: 60,
				ErrorMsg:   "天数输入错误，请重新输入！",
			})
			if exit {
				return
			}
			months, _ = strconv.Atoi(daysStr)
			totalCoin = config.DailyCoin * months
		} else {
			monthStr, exit := getUserInputByField(sender, msgChannel, InputField{
				Key:        "months",
				Prompt:     fmt.Sprintf("该活动每月扣费【%d】积分，您当前的积分还剩【%d】，请输入授权时长（月数，如1/3/6）：", config.MonthlyCoin, GetCoin(sender.UserID)),
				Validator: func(s string) (bool, string) {
					num, err := strconv.Atoi(s)
					if err != nil {
						return false, "请输入有效的数字！"
					}
					if num < 1 || num > 12 {
						return false, "时长需在1-12个月之间！"
					}
					return true, ""
				},
				Required:   true,
				TrimSpace:  false,
				TimeoutSec: 60,
				ErrorMsg:   "时长输入错误，请重新输入！",
			})
			if exit {
				return
			}
			months, _ = strconv.Atoi(monthStr)
			totalCoin = config.MonthlyCoin * months
		}

		// 5. 扣除积分校验
		userCoin := GetCoin(sender.UserID)
		if userCoin < totalCoin {
			sender.Reply(fmt.Sprintf("积分不足！授权需要%d积分，你当前有%d积分。",
				totalCoin, userCoin))
			return
		}

		// ================= 【核心修复部分：日期计算逻辑】 =================
		
		var newExpireDate string
		var baseTime time.Time
		var hasOldDate bool

		baseTime, hasOldDate = ParseRemarksDate(selectedRemarks)

		if config.IsDailyDeduct {
			if hasOldDate {
				now := time.Now()
				expireThreshold := time.Date(baseTime.Year(), baseTime.Month(), baseTime.Day(), 0, 0, 0, 0, time.Local).AddDate(0, 0, 1)
				if now.Before(expireThreshold) {
					newExpireDate = baseTime.AddDate(0, 0, months).Format(DateLayout)
					logs.Info("按天续费（未过期）：基准日期[%s] + [%d]天 = [%s]", baseTime.Format(DateLayout), months, newExpireDate)
				} else {
					newExpireDate = GenerateExpireDateFromDays(months)
					logs.Info("按天续费（已过期，从今天起算）：当前时间 + [%d]天 = [%s]", months, newExpireDate)
				}
			} else {
				newExpireDate = GenerateExpireDateFromDays(months)
				logs.Info("按天新开通：当前时间 + [%d]天 = [%s]", months, newExpireDate)
			}
		} else {
			if hasOldDate {
				newExpireDate = GenerateExpireDateFromBase(baseTime, months)
				logs.Info("按月续费：基准日期[%s] + [%d]个月 = [%s]", 
					baseTime.Format(DateLayout), months, newExpireDate)
			} else {
				newExpireDate = GenerateExpireDate(months)
				logs.Info("按月新开通：当前时间 + [%d]个月 = [%s]", 
					months, newExpireDate)
			}
		}

		newRemarks := BuildMonthDeductRemarks(selectedRemarks, newExpireDate)
		
		// =============================================================

		project, err := GetActivityProjectByRemarks(config.ID, selectedRemarks, config.EnvKey)
		if err != nil {
			sender.Reply(fmt.Sprintf("查询账号信息失败：%v", err))
			log.Printf("查询账号信息失败：%v", err)
			return
		}

		project.Remarks = newRemarks
		project.ExpireDate = newExpireDate
		project.NeedCoin = 0
		project.Status = 0
		if config.IsDailyDeduct {
			project.IsDailyDeduct = true
			project.DailyCoin = config.DailyCoin
		}
		project.SyncStatus = "pending_update"
		project.SyncError = ""
		if err := UpdateActivityProject(project); err != nil {
			sender.Reply(fmt.Sprintf("更新数据库失败：%v", err))
			log.Printf("更新数据库失败：%v", err)
			return
		}

		go TriggerSync(project.ID)

		// 9. 扣除积分
		RemCoin(sender.UserID, totalCoin)
		RecordCoinLog(sender.UserID, -totalCoin, "续费扣费", fmt.Sprintf("%s续费", config.Name))

		if config.IsDailyDeduct {
			sender.Reply(fmt.Sprintf("✅ 授权成功！\n已扣除%d积分（%d天×%d积分/天）\n剩余积分：%d\n授权有效期至：%s",
				totalCoin, months, config.DailyCoin, userCoin-totalCoin, newExpireDate))
		} else {
			sender.Reply(fmt.Sprintf("✅ 授权成功！\n已扣除%d积分（%d个月×%d积分/月）\n剩余积分：%d\n授权有效期至：%s",
				totalCoin, months, config.MonthlyCoin, userCoin-totalCoin, newExpireDate))
		}
		
		logs.Info("用户[%d] 授权活动[%s] 成功，新有效期[%s], 扣除积分[%d]", 
			sender.UserID, config.Name, newExpireDate, totalCoin)
	}()

	return nil
}