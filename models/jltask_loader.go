package models

import (
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

//	"github.com/beego/beego/v2/adapter/logs"
	"github.com/fsnotify/fsnotify"
	"gopkg.in/yaml.v2"
)

// ===================== YAML 配置结构定义 =====================

// YAMLActivityConfig 对应 YAML 文件中的活动配置
type YAMLActivityConfig struct {
	Name               string              `yaml:"名称"`
	EnvKey             string              `yaml:"环境变量名"`
	NeedCoin           int                 `yaml:"所需积分"`
	QingLongConfigName string              `yaml:"青龙配置名"`
	IsMonthlyDeduct    bool                `yaml:"是否按月扣费"`
	MonthlyCoin        int                 `yaml:"每月积分"`
	DisplayOrder       int                 `yaml:"排序编号"`
	Enabled            bool                `yaml:"启用状态"`
	Guide              string              `yaml:"玩法和说明"`
	InputFields        []YAMLInputField    `yaml:"输入字段"`
	CKBuilderTemplate  string              `yaml:"CK构建模板"`
	RemarksTemplate    string              `yaml:"备注构建模板"`
	ScriptPaths        YAMLScriptPaths     `yaml:"脚本路径"`
}

// YAMLInputField 对应 YAML 中的输入字段配置
type YAMLInputField struct {
	Key        string           `yaml:"字段标识"`
	Prompt     string           `yaml:"提示文本"`
	Validators []YAMLValidator  `yaml:"验证规则"`  // 支持多个验证规则
	Required   bool             `yaml:"是否必填"`
	TrimSpace  bool             `yaml:"是否去空格"`
	TimeoutSec int              `yaml:"超时秒数"`
	ErrorMsg   string           `yaml:"错误提示"`
}

// YAMLValidator 验证规则配置
type YAMLValidator struct {
	Type      string   `yaml:"类型"`       // regex/forbidden_chars/required_chars/no_chinese/not_empty/min_length/max_length/prefix/suffix/equals/phone/not_all_same
	Pattern   string   `yaml:"正则表达式"`   // 用于 regex 类型
	Chars     []string `yaml:"字符列表"`     // 用于 forbidden_chars/required_chars
	Value     string   `yaml:"值"`         // 用于 prefix/suffix/equals
	Length    int      `yaml:"长度"`        // 用于 min_length/max_length
	ErrorMsg  string   `yaml:"错误提示"`     // 验证失败时的提示
}


// YAMLScriptPaths 对应 YAML 中的脚本路径配置
type YAMLScriptPaths struct {
	Query    string `yaml:"查询"`
	Record   string `yaml:"记录"`
	Update   string `yaml:"更新"`
	QueryEnv string `yaml:"查询环境变量"`
	Delete   string `yaml:"删除"`
}

// YAMLActivitiesConfig 根配置结构
type YAMLActivitiesConfig struct {
	QingLongConfigs []YAMLQingLongConfig `yaml:"qinglong_configs"` // 青龙容器配置
	Activities      []YAMLActivityConfig  `yaml:"activities"`
}

// YAMLQingLongConfig 对应 YAML 中的青龙容器配置
type YAMLQingLongConfig struct {
	Name         string `yaml:"名称"`
	Host         string `yaml:"地址"`
	ClientID     string `yaml:"客户端ID"`
	ClientSecret string `yaml:"客户端密钥"`
	Timeout      int    `yaml:"超时秒数"`
}

// ===================== 热加载管理器 =====================

// ActivityLoader 活动配置热加载管理器
type ActivityLoader struct {
	configPath string
	watcher    *fsnotify.Watcher
	mu         sync.RWMutex
	stopChan   chan bool
}

var (
	// activityLoader 全局热加载器实例
	activityLoader *ActivityLoader
	// activityLoaderOnce 确保单例
	activityLoaderOnce sync.Once
)

// GetActivityLoader 获取热加载器单例
func GetActivityLoader() *ActivityLoader {
	activityLoaderOnce.Do(func() {
		activityLoader = &ActivityLoader{
			configPath: filepath.Join("conf", "activities.yaml"),
			stopChan:   make(chan bool),
		}
	})
	return activityLoader
}

// StartHotReload 启动热加载监控
func (al *ActivityLoader) StartHotReload() error {
	// 首次加载配置
	if err := al.LoadConfig(); err != nil {
		return fmt.Errorf("初始加载配置失败: %v", err)
	}

	// 创建文件监控器
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("创建文件监控器失败: %v", err)
	}
	al.watcher = watcher

	// 监控配置文件
	if err := watcher.Add(al.configPath); err != nil {
		return fmt.Errorf("添加文件监控失败: %v", err)
	}

	// 启动监控协程
	go al.watchLoop()

	log.Printf("[热加载] 已启动对 %s 的监控", al.configPath)
	return nil
}

// StopHotReload 停止热加载监控
func (al *ActivityLoader) StopHotReload() {
	close(al.stopChan)
	if al.watcher != nil {
		al.watcher.Close()
	}
}

// watchLoop 文件监控循环
func (al *ActivityLoader) watchLoop() {
	// 防抖计时器，避免频繁修改导致多次加载
	var reloadTimer *time.Timer
	const debounceDuration = 500 * time.Millisecond

	for {
		select {
		case event, ok := <-al.watcher.Events:
			if !ok {
				return
			}
			// 只关注写入和创建事件
			if event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create {
				log.Printf("[热加载] 检测到文件变化: %s", event.Name)
				
				// 防抖处理
				if reloadTimer != nil {
					reloadTimer.Stop()
				}
				reloadTimer = time.AfterFunc(debounceDuration, func() {
					if err := al.LoadConfig(); err != nil {
						log.Printf("[热加载] 配置加载失败: %v", err)
						al.notifyAdmin(fmt.Sprintf("⚠️ 活动配置热加载失败\n错误: %v\n时间: %s", err, time.Now().Format("2006-01-02 15:04:05")))
					} else {
						log.Printf("[热加载] 配置加载成功")
						al.notifyAdmin(fmt.Sprintf("✅ 活动配置热加载成功\n时间: %s", time.Now().Format("2006-01-02 15:04:05")))
					}
				})
			}

		case err, ok := <-al.watcher.Errors:
			if !ok {
				return
			}
			log.Printf("[热加载] 监控错误: %v", err)

		case <-al.stopChan:
			return
		}
	}
}

// LoadConfig 加载配置文件
func (al *ActivityLoader) LoadConfig() error {
	// 读取文件
	data, err := ioutil.ReadFile(al.configPath)
	if err != nil {
		return fmt.Errorf("读取配置文件失败: %v", err)
	}

	// 解析 YAML
	var yamlConfig YAMLActivitiesConfig
	if err := yaml.Unmarshal(data, &yamlConfig); err != nil {
		return fmt.Errorf("解析 YAML 失败: %v", err)
	}

	// 初始化青龙配置（优先从 YAML 读取，如果 YAML 中没有配置则使用内置默认值）
	initQingLongConfigsFromYAML(yamlConfig.QingLongConfigs)

	// 转换为 ActivityConfig 并校验
	newConfigs := make([]*ActivityConfig, 0, len(yamlConfig.Activities))
	envKeySet := make(map[string]bool) // 用于校验 EnvKey 唯一性
	var errors []string

	for i, yamlAct := range yamlConfig.Activities {
		// 校验必填字段
		if yamlAct.Name == "" {
			errors = append(errors, fmt.Sprintf("第%d个活动: 名称不能为空", i+1))
			continue
		}
		if yamlAct.EnvKey == "" {
			errors = append(errors, fmt.Sprintf("活动[%s]: 环境变量名不能为空", yamlAct.Name))
			continue
		}

		// 校验 EnvKey 唯一性
		if envKeySet[yamlAct.EnvKey] {
			errors = append(errors, fmt.Sprintf("活动[%s]: 环境变量名[%s]重复", yamlAct.Name, yamlAct.EnvKey))
			continue
		}
		envKeySet[yamlAct.EnvKey] = true

		// 校验青龙配置是否存在
		qlManager.mu.RLock()
		_, qlExists := qlManager.Configs[yamlAct.QingLongConfigName]
		qlManager.mu.RUnlock()
		
		if yamlAct.QingLongConfigName != "" && !qlExists {
			errors = append(errors, fmt.Sprintf("活动[%s]: 青龙配置[%s]不存在", yamlAct.Name, yamlAct.QingLongConfigName))
			continue
		}

		// 构建 ActivityConfig（禁用的活动也加载，只是标记为不可用）
		act := al.convertYAMLToActivity(yamlAct)
		newConfigs = append(newConfigs, act)
	}

	// 如果有错误，记录但不阻止加载（部分成功）
	if len(errors) > 0 {
		errorMsg := strings.Join(errors, "\n")
		log.Printf("[热加载] 配置校验警告:\n%s", errorMsg)
		al.notifyAdmin(fmt.Sprintf("⚠️ 活动配置加载警告\n%s", errorMsg))
	}

	// 排序
	sort.Slice(newConfigs, func(i, j int) bool {
		if newConfigs[i].DisplayOrder != newConfigs[j].DisplayOrder {
			return newConfigs[i].DisplayOrder < newConfigs[j].DisplayOrder
		}
		return newConfigs[i].Name < newConfigs[j].Name
	})

	// ID 使用 EnvKey 作为稳定标识，MenuIndex 仅用于菜单展示
	menuIdx := 1
	for _, cfg := range newConfigs {
		cfg.ID = cfg.EnvKey
		if cfg.Enabled {
			cfg.MenuIndex = strconv.Itoa(menuIdx)
			menuIdx++
		} else {
			cfg.MenuIndex = "" // 禁用活动不在菜单中显示
		}
	}

	// 加锁更新全局配置
	activityConfigsMu.Lock()
	ActivityConfigs = newConfigs
	activityConfigsMu.Unlock()

	log.Printf("[热加载] 成功加载 %d 个活动", len(newConfigs))
	return nil
}

// convertYAMLToActivity 将 YAML 配置转换为 ActivityConfig
func (al *ActivityLoader) convertYAMLToActivity(yamlAct YAMLActivityConfig) *ActivityConfig {
	act := &ActivityConfig{
		Name:               yamlAct.Name,
		EnvKey:             yamlAct.EnvKey,
		NeedCoin:           yamlAct.NeedCoin,
		QingLongConfigName: yamlAct.QingLongConfigName,
		Guide:              strings.TrimSpace(yamlAct.Guide),
		IsMonthlyDeduct:    yamlAct.IsMonthlyDeduct,
		MonthlyCoin:        yamlAct.MonthlyCoin,
		DisplayOrder:       yamlAct.DisplayOrder,
		Enabled:            yamlAct.Enabled,
		InputFields:        make([]InputField, 0, len(yamlAct.InputFields)),
		CKTemplate:         yamlAct.CKBuilderTemplate,
		ScriptPaths: struct {
			Query    string
			Record   string
			Update   string
			QueryEnv string
			Delete   string
		}{
			Query:    yamlAct.ScriptPaths.Query,
			Record:   yamlAct.ScriptPaths.Record,
			Update:   yamlAct.ScriptPaths.Update,
			QueryEnv: yamlAct.ScriptPaths.QueryEnv,
			Delete:   yamlAct.ScriptPaths.Delete,
		},
	}

	// 转换输入字段
	for _, yamlField := range yamlAct.InputFields {
		prompt := yamlField.Prompt
		if strings.TrimSpace(prompt) == "" {
			prompt = yamlField.Key
		}
		field := InputField{
			Key:        yamlField.Key,
			Prompt:     prompt,
			Required:   yamlField.Required,
			TrimSpace:  yamlField.TrimSpace,
			TimeoutSec: yamlField.TimeoutSec,
			ErrorMsg:   yamlField.ErrorMsg,
			Validator:  al.buildValidator(yamlField.Validators),
		}
		if field.TimeoutSec == 0 {
			field.TimeoutSec = 60
		}
		act.InputFields = append(act.InputFields, field)
	}

	// 设置构建器
	act.CKBuilder = al.getCKBuilder(yamlAct.CKBuilderTemplate, yamlAct.InputFields)
	act.RemarksBuilder = al.getRemarksBuilder(yamlAct.RemarksTemplate)

	return act
}

// buildValidator 根据 YAML 验证规则构建验证器函数
// 支持多个验证规则组合，全部通过才算验证成功
func (al *ActivityLoader) buildValidator(validators []YAMLValidator) func(input string) (bool, string) {
	// 如果没有配置验证规则，返回默认验证器（允许任何输入）
	if len(validators) == 0 {
		return func(input string) (bool, string) {
			return true, ""
		}
	}

	return func(input string) (bool, string) {
		for _, v := range validators {
			valid, msg := al.executeValidator(v, input)
			if !valid {
				return false, msg
			}
		}
		return true, ""
	}
}

// executeValidator 执行单个验证规则
func (al *ActivityLoader) executeValidator(v YAMLValidator, input string) (bool, string) {
	switch v.Type {
	case "regex":
		// 正则表达式验证
		if v.Pattern == "" {
			return true, ""
		}
		matched, err := regexp.MatchString(v.Pattern, input)
		if err != nil || !matched {
			if v.ErrorMsg != "" {
				return false, v.ErrorMsg
			}
			return false, "输入格式不正确，请重新输入！"
		}
		return true, ""

	case "forbidden_chars":
		// 禁止包含某些字符
		for _, char := range v.Chars {
			if strings.Contains(input, char) {
				if v.ErrorMsg != "" {
					return false, v.ErrorMsg
				}
				return false, fmt.Sprintf("不能包含字符 '%s'，请重新输入！", char)
			}
		}
		return true, ""

	case "required_chars":
		// 必须包含某些字符
		for _, char := range v.Chars {
			if !strings.Contains(input, char) {
				if v.ErrorMsg != "" {
					return false, v.ErrorMsg
				}
				return false, fmt.Sprintf("必须包含字符 '%s'，请重新输入！", char)
			}
		}
		return true, ""

	case "not_empty":
		// 不能为空
		if len(input) == 0 {
			if v.ErrorMsg != "" {
				return false, v.ErrorMsg
			}
			return false, "该字段不能为空，请重新输入！"
		}
		return true, ""

	case "phone":
		valid, msg := ValidatePhone(input)
		if !valid {
			if v.ErrorMsg != "" {
				return false, v.ErrorMsg
			}
			return false, msg
		}
		return true, ""

	case "min_length":
		if utf8.RuneCountInString(input) < v.Length {
			if v.ErrorMsg != "" {
				return false, v.ErrorMsg
			}
			return false, fmt.Sprintf("长度不能少于%d个字符，请重新输入！", v.Length)
		}
		return true, ""

	case "max_length":
		if utf8.RuneCountInString(input) > v.Length {
			if v.ErrorMsg != "" {
				return false, v.ErrorMsg
			}
			return false, fmt.Sprintf("长度不能超过%d个字符，请重新输入！", v.Length)
		}
		return true, ""

	case "prefix":
		// 必须以某字符串开头
		if !strings.HasPrefix(input, v.Value) {
			if v.ErrorMsg != "" {
				return false, v.ErrorMsg
			}
			return false, fmt.Sprintf("必须以 '%s' 开头，请重新输入！", v.Value)
		}
		return true, ""

	case "suffix":
		// 必须以某字符串结尾
		if !strings.HasSuffix(input, v.Value) {
			if v.ErrorMsg != "" {
				return false, v.ErrorMsg
			}
			return false, fmt.Sprintf("必须以 '%s' 结尾，请重新输入！", v.Value)
		}
		return true, ""

	case "not_all_same":
		if utf8.RuneCountInString(input) == 0 {
			return true, ""
		}
		allSame := true
		runes := []rune(input)
		firstRune := runes[0]
		for i := 1; i < len(runes); i++ {
			if runes[i] != firstRune {
				allSame = false
				break
			}
		}
		if allSame {
			if v.ErrorMsg != "" {
				return false, v.ErrorMsg
			}
			return false, "不能全是相同字符，请重新输入！"
		}
		return true, ""

	case "equals":
		// 必须等于指定值
		if v.Value == "" {
			return true, ""
		}
		if input != v.Value {
			if v.ErrorMsg != "" {
				return false, v.ErrorMsg
			}
			return false, fmt.Sprintf("输入必须等于 '%s'，请重新输入！", v.Value)
		}
		return true, ""

	default:
		// 未知的验证类型，记录警告但允许通过
		log.Printf("[热加载] 未知的验证类型: %s", v.Type)
		return true, ""
	}
}


// getCKBuilder 根据模板生成 CK 构建函数
func (al *ActivityLoader) getCKBuilder(template string, fields []YAMLInputField) func(inputs map[string]string) string {
	return func(inputs map[string]string) string {
		result := template
		emptyPlaceholders := make([]string, 0, len(fields))
		for _, field := range fields {
			placeholder := "{{." + field.Key + "}}"
			value := strings.TrimSpace(inputs[field.Key])
			result = strings.ReplaceAll(result, placeholder, value)
			if value == "" {
				emptyPlaceholders = append(emptyPlaceholders, regexp.QuoteMeta(placeholder))
			}
		}
		if len(emptyPlaceholders) > 0 {
			pattern := `(?:(?:#|&|\||,|;|:|@|/|_|-)+)?(?:` + strings.Join(emptyPlaceholders, "|") + `)(?:(?:#|&|\||,|;|:|@|/|_|-)+)?`
			re := regexp.MustCompile(pattern)
			result = re.ReplaceAllString(result, "")
		}
		result = strings.Trim(result, "#&|,;:@/_-")
		if strings.Contains(result, "@@@BING@@@") {
			result = strings.ReplaceAll(result, "&", "%26")
		}
		return result
	}
}

// getRemarksBuilder 根据模板生成备注构建函数
func (al *ActivityLoader) getRemarksBuilder(template string) func(qq int, userRemarks string, inputs map[string]string) string {
	return func(qq int, userRemarks string, inputs map[string]string) string {
		result := template
		result = strings.ReplaceAll(result, "{{.qq}}", strconv.Itoa(qq))
		result = strings.ReplaceAll(result, "{{.user_remarks}}", userRemarks)
		for key, value := range inputs {
			placeholder := "{{." + key + "}}"
			result = strings.ReplaceAll(result, placeholder, value)
		}
		return result
	}
}

// notifyAdmin 通知管理员
func (al *ActivityLoader) notifyAdmin(msg string) {
	// 使用项目原有的推送方式
	(&JdCookie{}).Push(msg)
}

// ===================== 初始化入口 =====================

// InitActivityListWithHotReload 初始化活动列表并启动热加载
func InitActivityListWithHotReload() error {
	loader := GetActivityLoader()
	return loader.StartHotReload()
}

// ReloadActivities 手动重新加载活动配置（供管理员命令使用）
func ReloadActivities() string {
	loader := GetActivityLoader()
	if err := loader.LoadConfig(); err != nil {
		return fmt.Sprintf("重新加载失败: %v", err)
	}
	return fmt.Sprintf("重新加载成功，当前共 %d 个活动", len(ActivityConfigs))
}

// GetActivityCount 获取当前活动数量（用于调试）
func GetActivityCount() int {
	activityConfigsMu.RLock()
	defer activityConfigsMu.RUnlock()
	return len(ActivityConfigs)
}
