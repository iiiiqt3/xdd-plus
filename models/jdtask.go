package models

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v2"
	"gorm.io/gorm"
)

const (
	jdManualScriptRelDir   = "scripts/自定义执行京东脚本/6dylan6_jdpro"
	jdManualLogRelDir      = "scripts/自定义执行京东脚本/logs"
	jdManualTasksConfigRel = "conf/jd_manual_tasks.yaml"
	jdManualLogRetainDays  = 7
)

var (
	jdEnvNameRe = regexp.MustCompile(`new\s+Env\s*\(\s*['"]([^'"]+)['"]`)
	jdManualExcludeScripts = map[string]bool{
		"jdCookie.js": true,
	}
)

// JdManualTaskItem 手动京东任务配置项
type JdManualTaskItem struct {
	ID      string            `yaml:"id" json:"id"`
	Script  string            `yaml:"script" json:"script"`
	Name    string            `yaml:"name" json:"name"`
	Enabled bool              `yaml:"enabled" json:"enabled"`
	Order   int               `yaml:"order" json:"order"`
	Coin    int               `yaml:"coin" json:"coin"`
	Envs    map[string]string `yaml:"envs,omitempty" json:"envs,omitempty"`
}

// JdManualProxyConfig 手动京东任务专用代理
type JdManualProxyConfig struct {
	Enabled bool   `yaml:"enabled" json:"enabled"`
	URL     string `yaml:"url" json:"url"`
	Renum   string `yaml:"renum" json:"renum"`
	Redelay string `yaml:"redelay" json:"redelay"`
}

// JdManualScanConfig 扫描配置
type JdManualScanConfig struct {
	Dir            string   `yaml:"dir" json:"dir"`
	Pattern        string   `yaml:"pattern" json:"pattern"`
	Exclude        []string `yaml:"exclude" json:"exclude"`
	DefaultEnabled bool     `yaml:"default_enabled" json:"default_enabled"`
}

// JdManualTasksFile yaml 根结构
type JdManualTasksFile struct {
	Scan  JdManualScanConfig  `yaml:"scan" json:"scan"`
	Proxy JdManualProxyConfig `yaml:"proxy" json:"proxy"`
	Tasks []JdManualTaskItem  `yaml:"tasks" json:"tasks"`
}

type jdManualRegistry struct {
	mu    sync.RWMutex
	file  JdManualTasksFile
	byID  map[string]*JdManualTaskItem
}

var jdManualTasks = &jdManualRegistry{
	byID: make(map[string]*JdManualTaskItem),
}

func jdManualScriptDir() string {
	return filepath.Join(ExecPath, jdManualScriptRelDir)
}

func jdManualLogDir() string {
	return filepath.Join(ExecPath, jdManualLogRelDir)
}

func jdManualConfigPath() string {
	return filepath.Join(ExecPath, jdManualTasksConfigRel)
}

func defaultJdManualTasksFile() JdManualTasksFile {
	return JdManualTasksFile{
		Scan: JdManualScanConfig{
			Dir:            jdManualScriptRelDir,
			Pattern:        "jd_*.js",
			Exclude:        []string{"jdCookie.js"},
			DefaultEnabled: false,
		},
		Proxy: JdManualProxyConfig{
			Enabled: false,
			URL:     "",
			Renum:   "10",
			Redelay: "2",
		},
		Tasks: []JdManualTaskItem{},
	}
}

// InitJdManualTasks 启动时加载配置并扫描脚本目录
func InitJdManualTasks() {
	_ = os.MkdirAll(jdManualScriptDir(), 0755)
	_ = os.MkdirAll(jdManualLogDir(), 0755)
	if err := loadJdManualTasksFile(); err != nil {
		JD().Warnf("[手动京东任务] 加载配置失败，使用默认: %v", err)
		jdManualTasks.mu.Lock()
		jdManualTasks.file = defaultJdManualTasksFile()
		jdManualTasks.rebuildIndexLocked()
		jdManualTasks.mu.Unlock()
	}
	if _, err := ScanAndSyncJdManualTasks(); err != nil {
		JD().Warnf("[手动京东任务] 首次扫描失败: %v", err)
	} else {
		JD().Infof("[手动京东任务] 初始化完成，共 %d 个任务", len(GetJdManualTaskList(false)))
	}
}

func loadJdManualTasksFile() error {
	path := jdManualConfigPath()
	data, err := ioutil.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			f := defaultJdManualTasksFile()
			jdManualTasks.mu.Lock()
			jdManualTasks.file = f
			jdManualTasks.rebuildIndexLocked()
			jdManualTasks.mu.Unlock()
			return saveJdManualTasksFileUnlocked(f)
		}
		return err
	}
	var f JdManualTasksFile
	if err := yaml.Unmarshal(data, &f); err != nil {
		return err
	}
	if f.Scan.Dir == "" {
		f.Scan = defaultJdManualTasksFile().Scan
	}
	if f.Scan.Pattern == "" {
		f.Scan.Pattern = "jd_*.js"
	}
	if f.Proxy.Renum == "" {
		f.Proxy.Renum = "10"
	}
	if f.Proxy.Redelay == "" {
		f.Proxy.Redelay = "2"
	}
	if f.Tasks == nil {
		f.Tasks = []JdManualTaskItem{}
	}
	for i := range f.Tasks {
		if f.Tasks[i].Envs == nil {
			f.Tasks[i].Envs = map[string]string{}
		}
	}
	jdManualTasks.mu.Lock()
	jdManualTasks.file = f
	jdManualTasks.rebuildIndexLocked()
	jdManualTasks.mu.Unlock()
	return nil
}

func (r *jdManualRegistry) rebuildIndexLocked() {
	r.byID = make(map[string]*JdManualTaskItem, len(r.file.Tasks))
	for i := range r.file.Tasks {
		id := strings.TrimSpace(r.file.Tasks[i].ID)
		if id == "" {
			continue
		}
		r.file.Tasks[i].ID = id
		r.byID[id] = &r.file.Tasks[i]
	}
}

func saveJdManualTasksFileUnlocked(f JdManualTasksFile) error {
	_ = os.MkdirAll(filepath.Dir(jdManualConfigPath()), 0755)
	data, err := yaml.Marshal(&f)
	if err != nil {
		return err
	}
	header := []byte("# 手动京东任务配置（扫描自动维护 tasks，可在后台修改 name/enabled/order/coin/envs/proxy）\n")
	return ioutil.WriteFile(jdManualConfigPath(), append(header, data...), 0644)
}

func saveJdManualTasksLocked() error {
	return saveJdManualTasksFileUnlocked(jdManualTasks.file)
}

func jdManualTaskIDFromScript(filename string) string {
	base := strings.TrimSuffix(filepath.Base(filename), ".js")
	base = strings.TrimPrefix(base, "jd_")
	if base == "" {
		return strings.TrimSuffix(filepath.Base(filename), ".js")
	}
	return base
}

func detectJdScriptDisplayName(scriptPath string) string {
	data, err := ioutil.ReadFile(scriptPath)
	if err != nil {
		return ""
	}
	// 只扫前 8KB，足够覆盖文件头
	chunk := data
	if len(chunk) > 8192 {
		chunk = chunk[:8192]
	}
	m := jdEnvNameRe.FindSubmatch(chunk)
	if len(m) > 1 {
		return strings.TrimSpace(string(m[1]))
	}
	return ""
}

func isExcludedJdManualScript(name string, exclude []string) bool {
	lower := strings.ToLower(name)
	if jdManualExcludeScripts[name] || jdManualExcludeScripts[lower] {
		return true
	}
	for _, e := range exclude {
		e = strings.TrimSpace(e)
		if e == "" {
			continue
		}
		if strings.EqualFold(e, name) {
			return true
		}
	}
	return false
}

// ScanAndSyncJdManualTasks 扫描脚本目录并同步 yaml（新脚本默认禁用/coin=0；删文件则删配置）
func ScanAndSyncJdManualTasks() (JdManualTasksFile, error) {
	dir := jdManualScriptDir()
	_ = os.MkdirAll(dir, 0755)

	entries, err := ioutil.ReadDir(dir)
	if err != nil {
		return JdManualTasksFile{}, err
	}

	jdManualTasks.mu.Lock()
	defer jdManualTasks.mu.Unlock()

	f := jdManualTasks.file
	if f.Scan.Dir == "" {
		f.Scan = defaultJdManualTasksFile().Scan
	}
	exclude := f.Scan.Exclude
	defaultEnabled := f.Scan.DefaultEnabled

	existing := make(map[string]JdManualTaskItem, len(f.Tasks))
	for _, t := range f.Tasks {
		existing[t.ID] = t
	}

	foundIDs := make(map[string]bool)
	var next []JdManualTaskItem
	maxOrder := 0
	for _, t := range f.Tasks {
		if t.Order > maxOrder {
			maxOrder = t.Order
		}
	}

	for _, ent := range entries {
		if ent.IsDir() {
			continue
		}
		name := ent.Name()
		matched, _ := filepath.Match("jd_*.js", name)
		if !matched {
			continue
		}
		if isExcludedJdManualScript(name, exclude) {
			continue
		}
		id := jdManualTaskIDFromScript(name)
		if id == "" || foundIDs[id] {
			continue
		}
		foundIDs[id] = true
		scriptPath := filepath.Join(dir, name)
		detected := detectJdScriptDisplayName(scriptPath)

		if old, ok := existing[id]; ok {
			old.Script = name
			if strings.TrimSpace(old.Name) == "" {
				if detected != "" {
					old.Name = detected
				} else {
					old.Name = id
				}
			}
			if old.Envs == nil {
				old.Envs = map[string]string{}
			}
			next = append(next, old)
			continue
		}

		maxOrder += 10
		display := detected
		if display == "" {
			display = id
		}
		next = append(next, JdManualTaskItem{
			ID:      id,
			Script:  name,
			Name:    display,
			Enabled: defaultEnabled,
			Order:   maxOrder,
			Coin:    0,
			Envs:    map[string]string{},
		})
	}

	sort.SliceStable(next, func(i, j int) bool {
		if next[i].Order == next[j].Order {
			return next[i].ID < next[j].ID
		}
		return next[i].Order < next[j].Order
	})

	f.Tasks = next
	jdManualTasks.file = f
	jdManualTasks.rebuildIndexLocked()
	if err := saveJdManualTasksLocked(); err != nil {
		return f, err
	}
	return f, nil
}

// GetJdManualTasksAdmin 后台完整配置
func GetJdManualTasksAdmin() JdManualTasksFile {
	jdManualTasks.mu.RLock()
	defer jdManualTasks.mu.RUnlock()
	return cloneJdManualTasksFile(jdManualTasks.file)
}

func cloneJdManualTasksFile(src JdManualTasksFile) JdManualTasksFile {
	dst := src
	dst.Scan.Exclude = append([]string{}, src.Scan.Exclude...)
	dst.Tasks = make([]JdManualTaskItem, len(src.Tasks))
	for i, t := range src.Tasks {
		dst.Tasks[i] = t
		if t.Envs != nil {
			dst.Tasks[i].Envs = copyStringMap(t.Envs)
		} else {
			dst.Tasks[i].Envs = map[string]string{}
		}
	}
	return dst
}

// SaveJdManualTasksAdmin 保存后台编辑（保留扫描到的 script，按 id 合并）
func SaveJdManualTasksAdmin(proxy JdManualProxyConfig, tasks []JdManualTaskItem) error {
	jdManualTasks.mu.Lock()
	defer jdManualTasks.mu.Unlock()

	byScript := make(map[string]bool)
	uniq := make([]JdManualTaskItem, 0, len(tasks))
	seen := make(map[string]bool)
	for _, t := range tasks {
		t.ID = strings.TrimSpace(t.ID)
		t.Script = strings.TrimSpace(t.Script)
		if t.ID == "" || t.Script == "" {
			continue
		}
		if seen[t.ID] || byScript[t.Script] {
			continue
		}
		seen[t.ID] = true
		byScript[t.Script] = true
		if t.Name == "" {
			t.Name = t.ID
		}
		if t.Coin < 0 {
			t.Coin = 0
		}
		if t.Envs == nil {
			t.Envs = map[string]string{}
		}
		uniq = append(uniq, t)
	}
	sort.SliceStable(uniq, func(i, j int) bool {
		if uniq[i].Order == uniq[j].Order {
			return uniq[i].ID < uniq[j].ID
		}
		return uniq[i].Order < uniq[j].Order
	})

	if proxy.Renum == "" {
		proxy.Renum = "10"
	}
	if proxy.Redelay == "" {
		proxy.Redelay = "2"
	}

	jdManualTasks.file.Proxy = proxy
	jdManualTasks.file.Tasks = uniq
	jdManualTasks.rebuildIndexLocked()
	return saveJdManualTasksLocked()
}

// GetJdManualTaskList 任务列表；onlyEnabled=true 给门户
func GetJdManualTaskList(onlyEnabled bool) []JdManualTaskItem {
	jdManualTasks.mu.RLock()
	defer jdManualTasks.mu.RUnlock()
	out := make([]JdManualTaskItem, 0, len(jdManualTasks.file.Tasks))
	for _, t := range jdManualTasks.file.Tasks {
		if onlyEnabled && !t.Enabled {
			continue
		}
		item := t
		if item.Envs != nil {
			item.Envs = copyStringMap(item.Envs)
		}
		out = append(out, item)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Order == out[j].Order {
			return out[i].ID < out[j].ID
		}
		return out[i].Order < out[j].Order
	})
	return out
}

func getJdManualTaskByID(taskID string) (*JdManualTaskItem, bool) {
	jdManualTasks.mu.RLock()
	defer jdManualTasks.mu.RUnlock()
	t, ok := jdManualTasks.byID[taskID]
	if !ok || t == nil {
		return nil, false
	}
	cp := *t
	if t.Envs != nil {
		cp.Envs = copyStringMap(t.Envs)
	}
	return &cp, true
}

func getJdManualProxyConfig() JdManualProxyConfig {
	jdManualTasks.mu.RLock()
	defer jdManualTasks.mu.RUnlock()
	return jdManualTasks.file.Proxy
}

// ResolveManualJdTaskProxy 手动任务代理：开且填写 → 用手动；否则用系统京东代理
func ResolveManualJdTaskProxy() (enabled bool, url, renum, redelay string) {
	p := getJdManualProxyConfig()
	manualURL := strings.TrimSpace(p.URL)
	if p.Enabled && manualURL != "" {
		renum = strings.TrimSpace(p.Renum)
		redelay = strings.TrimSpace(p.Redelay)
		if renum == "" {
			renum = "10"
		}
		if redelay == "" {
			redelay = "2"
		}
		return true, manualURL, renum, redelay
	}
	if !IsJdTaskProxyEnabled() {
		return false, "", "", ""
	}
	renum = strings.TrimSpace(sysConfig.JdTaskProxyRenum)
	redelay = strings.TrimSpace(sysConfig.JdTaskProxyRedelay)
	if renum == "" {
		renum = "10"
	}
	if redelay == "" {
		redelay = "2"
	}
	return true, strings.TrimSpace(sysConfig.JdTaskProxyUrl), renum, redelay
}

// ApplyManualJdTaskProxyEnvs 注入手动/系统代理环境变量
func ApplyManualJdTaskProxyEnvs(envs map[string]string) {
	if envs == nil {
		return
	}
	ok, url, renum, redelay := ResolveManualJdTaskProxy()
	if !ok || url == "" {
		return
	}
	envs["DY_PROXY"] = url
	envs["DY_PROXY_RENUM"] = renum
	envs["DY_PROXY_REDELAY"] = redelay
	envs["PRO_API_PROXY_URL"] = url
	envs["PRO_PROXY_WHITELIST"] = "jd"
}

// CleanupJdManualTaskLogs 删除超过保留天数的任务日志
func CleanupJdManualTaskLogs() {
	dir := jdManualLogDir()
	entries, err := ioutil.ReadDir(dir)
	if err != nil {
		return
	}
	cutoff := time.Now().AddDate(0, 0, -jdManualLogRetainDays)
	removed := 0
	for _, ent := range entries {
		if ent.IsDir() {
			continue
		}
		if ent.ModTime().Before(cutoff) {
			if err := os.Remove(filepath.Join(dir, ent.Name())); err == nil {
				removed++
			}
		}
	}
	if removed > 0 {
		JD().Infof("[手动京东任务] 已清理 %d 个过期日志文件", removed)
	}
}

// 任务日志通道管理
var taskLogChannels = make(map[string]chan string)
var taskLogMutex sync.Mutex

// GetTaskLogChannel 获取任务日志通道
func GetTaskLogChannel(taskId string) chan string {
	taskLogMutex.Lock()
	defer taskLogMutex.Unlock()
	return taskLogChannels[taskId]
}

// CreateTaskLogChannel 创建任务日志通道（生命周期由任务批次结束时关闭）
func CreateTaskLogChannel(taskId string) chan string {
	taskLogMutex.Lock()
	defer taskLogMutex.Unlock()
	ch := make(chan string, 100)
	taskLogChannels[taskId] = ch
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
}

// safeLogSend 安全发送日志到通道，避免向已关闭的通道发送导致 panic
func safeLogSend(ch chan string, msg string) {
	defer func() { recover() }()
	select {
	case ch <- msg:
	default:
	}
}

// IsUserJdTaskRunning 同一用户同一任务是否已有执行/排队
func IsUserJdTaskRunning(userId int, taskId string) bool {
	return GetJdTaskScheduler().HasUserTask(userId, taskId)
}

// GetRunningTask 检查是否有同一任务的同一账号正在执行或排队
func GetRunningTask(userId int, taskId string, accountIndexes []int) string {
	if IsUserJdTaskRunning(userId, taskId) {
		return "running"
	}
	return ""
}

// StopPortalJdTask 停止正在执行或排队的 portal 任务
func StopPortalJdTask(taskId string) {
	GetJdTaskScheduler().StopTaskLog(taskId)
}

// SubmitPortalJdTask 提交网页端京东任务到调度队列（按账号数扣 coin）
func SubmitPortalJdTask(userId int, taskId string, taskName string, accountIndexes []int, taskLogId string, clientCtx ClientContext) error {
	logChan := GetTaskLogChannel(taskLogId)
	if logChan == nil {
		return fmt.Errorf("日志通道不存在")
	}

	task, ok := getJdManualTaskByID(taskId)
	if !ok || !task.Enabled {
		return fmt.Errorf("任务不存在或未启用")
	}
	displayName := task.Name
	if strings.TrimSpace(taskName) != "" {
		displayName = taskName
	}

	if IsUserJdTaskRunning(userId, taskId) {
		return fmt.Errorf("该任务正在执行中，请勿重复点击")
	}

	safeLogSend(logChan, fmt.Sprintf("开始执行任务: %s", displayName))

	specs, err := buildPortalJdJobSpecs(userId, taskId, displayName, accountIndexes, logChan)
	if err != nil {
		return err
	}
	if len(specs) == 0 {
		return fmt.Errorf("没有可执行的任务")
	}

	coinPerAccount := task.Coin
	if coinPerAccount < 0 {
		coinPerAccount = 0
	}
	totalCoin := coinPerAccount * len(specs)
	if totalCoin > 0 {
		if err := DeductCoinChecked(userId, totalCoin); err != nil {
			return err
		}
		RecordCoinLogEx(userId, -totalCoin, "任务扣费", fmt.Sprintf("执行任务: %s x%d账号", displayName, len(specs)), clientCtx)
		safeLogSend(logChan, fmt.Sprintf("已扣除积分 %d（每账号 %d × %d）", totalCoin, coinPerAccount, len(specs)))
	}

	safeLogSend(logChan, fmt.Sprintf("已选择 %d 个账号", len(specs)))
	if err := GetJdTaskScheduler().submitPortalBatch(userId, taskId, taskLogId, logChan, specs, clientCtx); err != nil {
		if totalCoin > 0 {
			db.Model(&User{}).Where("number = ?", userId).Update("coin", gorm.Expr(fmt.Sprintf("coin+%d", totalCoin)))
			RecordCoinLogEx(userId, totalCoin, "任务退费", fmt.Sprintf("入队失败: %s", displayName), clientCtx)
		}
		return err
	}
	return nil
}

func buildPortalJdJobSpecs(userId int, taskId string, taskName string, accountIndexes []int, logChan chan string) ([]jdJobSpec, error) {
	task, ok := getJdManualTaskByID(taskId)
	if !ok {
		return nil, fmt.Errorf("未知的任务类型 %s", taskId)
	}
	if !task.Enabled {
		return nil, fmt.Errorf("任务未启用")
	}

	scriptPath := filepath.Join(jdManualScriptDir(), task.Script)
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("脚本不存在: %s", task.Script)
	}

	selectedCks, err := resolveJdTaskAccountsByIndex(userId, accountIndexes, logChan)
	if err != nil {
		return nil, err
	}

	displayName := task.Name
	if strings.TrimSpace(taskName) != "" {
		displayName = taskName
	}

	specs := make([]jdJobSpec, 0, len(selectedCks))
	for _, ck := range selectedCks {
		envs := map[string]string{
			"pins": "&" + ck.PtPin,
		}
		for k, v := range task.Envs {
			envs[k] = v
		}
		specs = append(specs, jdJobSpec{
			TaskType:   task.ID,
			TaskName:   displayName,
			PtPin:      ck.PtPin,
			Nickname:   ck.Nickname,
			ScriptPath: scriptPath,
			Envs:       envs,
			Parser:     nil,
		})
	}
	return specs, nil
}

func run_fcwb_help_Task(sender *Sender, envVars map[string]string, FileName string) string {
	JD().Infof(fmt.Sprintf("开始运行%s任务", FileName)) // 使用 FileName 动态生成任务名称
	ApplyJdTaskProxyEnvs(envVars)
	jsFilePath := ExecPath + "/scripts/6dylan6_jdpro_help/" + FileName + ".js"

	if _, err := os.Stat(jsFilePath); os.IsNotExist(err) {
		Error("JavaScript 文件不存在: %v", err)
		return fmt.Sprintf("执行%s任务失败：JavaScript 文件不存在", FileName) // 使用 FileName
	}

	cmd := exec.Command("node", jsFilePath)

	// 设置环境变量
	for name, value := range envVars {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", name, value))
	}

	output, err := cmd.CombinedOutput()
	JD().Infof(fmt.Sprintf("%s任务脚本输出: %s", FileName, string(output)))

	if err != nil && strings.TrimSpace(string(output)) == "" {
		JD().Errorf("执行 JavaScript 脚本失败: %v", err)
		return fmt.Sprintf("执行%s任务失败：脚本执行错误", FileName) // 修正斜杠错误
	}

	// 传递 cookie 文件路径给 replexQuan_fcwb_help
	return replexQuan_fcwb_help(string(output), sender, FileName)
}

//##赚赚专属

func run_fcwb_help_Task_zz(sender *Sender, envVars map[string]string, FileName string) string {
	JD().Infof(fmt.Sprintf("开始运行%s任务", FileName)) // 使用 FileName 动态生成任务名称
	ApplyJdTaskProxyEnvs(envVars)
	jsFilePath := ExecPath + "/scripts/6dylan6_jdpro_help/" + FileName + ".js"

	if _, err := os.Stat(jsFilePath); os.IsNotExist(err) {
		Error("JavaScript 文件不存在: %v", err)
		return fmt.Sprintf("执行%s任务失败：JavaScript 文件不存在", FileName) // 使用 FileName
	}

	cmd := exec.Command("node", jsFilePath)

	// 设置环境变量
	for name, value := range envVars {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", name, value))
	}

	output, err := cmd.CombinedOutput()
	JD().Infof(fmt.Sprintf("%s任务脚本输出: %s", FileName, string(output)))

	if err != nil && strings.TrimSpace(string(output)) == "" {
		JD().Errorf("执行 JavaScript 脚本失败: %v", err)
		return fmt.Sprintf("执行%s任务失败：脚本执行错误", FileName) // 修正斜杠错误
	}

	// 传递 cookie 文件路径给 replexQuan_fcwb_help
	return replexQuan_fcwb_help_zz(string(output), sender, FileName)
}

//##环境的

func run_fcwb_help_Task1(sender *Sender, envVars map[string]string, FileName string) string {
	JD().Infof(fmt.Sprintf("开始运行%s任务", FileName)) // 使用 FileName 动态生成任务名称
	ApplyJdProTaskProxyEnvs(envVars)
	jsFilePath := ExecPath + "/scripts/huanjing/" + FileName + ".js"

	if _, err := os.Stat(jsFilePath); os.IsNotExist(err) {
		Error("JavaScript 文件不存在: %v", err)
		return fmt.Sprintf("执行%s任务失败：JavaScript 文件不存在", FileName) // 使用 FileName
	}

	cmd := exec.Command("node", jsFilePath)

	// 设置环境变量
	for name, value := range envVars {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", name, value))
	}

	output, err := cmd.CombinedOutput()
	JD().Infof(fmt.Sprintf("%s任务脚本输出: %s", FileName, string(output)))

	if err != nil && strings.TrimSpace(string(output)) == "" {
		JD().Errorf("执行 JavaScript 脚本失败: %v", err)
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
		RecordCoinForSender(sender, sender.UserID, -coinToDeduct, "助力扣费", fmt.Sprintf("成功助力%d个账号", successfulHelps))
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
		RecordCoinForSender(sender, sender.UserID, -coinToDeduct, "助力扣费", fmt.Sprintf("成功助力%d个账号", successfulHelps))
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
		RecordCoinForSender(sender, sender.UserID, -60, "助力扣费", "助力失败扣费")
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
		RecordCoinForSender(sender, sender.UserID, -coinToDeduct, "助力扣费", fmt.Sprintf("成功助力%d个账号", successfulHelps))
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
	JD().Infof(fmt.Sprintf("开始运行%s任务", FileName)) // 使用 FileName 动态生成任务名称
	ApplyJdTaskProxyEnvs(envVars)
	jsFilePath := ExecPath + "/scripts/6dylan6_jdpro_help/" + FileName + ".js"

	if _, err := os.Stat(jsFilePath); os.IsNotExist(err) {
		Error("JavaScript 文件不存在: %v", err)
		return fmt.Sprintf("执行%s任务失败：JavaScript 文件不存在", FileName) // 使用 FileName
	}

	cmd := exec.Command("node", jsFilePath)

	// 设置环境变量
	for name, value := range envVars {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", name, value))
	}

	output, err := cmd.CombinedOutput()
	JD().Infof(fmt.Sprintf("%s任务脚本输出: %s", FileName, string(output)))

	if err != nil && strings.TrimSpace(string(output)) == "" {
		JD().Errorf("执行 JavaScript 脚本失败: %v", err)
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
		RecordCoinForSender(sender, sender.UserID, -coinToDeduct, "助力扣费", fmt.Sprintf("成功助力%d个账号", successfulHelps))
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
	JD().Infof("导出所有账号")

	// 使用传入的 FileName 参数创建文件
	f, err := os.OpenFile(ExecPath+"/scripts/6dylan6_jdpro_help/"+FileName+".txt", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0777)
	if err != nil {
		JD().Warnf(fmt.Sprintf("创建%s.txt失败，", FileName), err)
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
	JD().Infof("导出所有账号")

	// 使用传入的 FileName 参数创建文件
	f, err := os.OpenFile(ExecPath+"/scripts/huanjing/"+FileName+".txt", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0777)
	if err != nil {
		JD().Warnf(fmt.Sprintf("创建%s.txt失败，", FileName), err)
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
