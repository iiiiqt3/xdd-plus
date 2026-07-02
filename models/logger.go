package models

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

// Category 日志分类，用于后台筛选与检索
type Category string

const (
	CatSystem   Category = "system"   // 启动、配置、定时任务、守护进程
	CatAdmin    Category = "admin"    // 管理后台操作
	CatPortal   Category = "portal"   // 网页门户 /api/portal
	CatApp      Category = "app"      // 移动端 / 旧版 API
	CatUser     Category = "user"     // 微信/QQ 用户指令与消息
	CatJD       Category = "jd"       // 京东任务、查询、CK
	CatKuwo     Category = "kuwo"     // 酷我抢兑
	CatQinglong Category = "qinglong" // 青龙 API 调用
	CatSync     Category = "sync"     // 数据库↔青龙同步
	CatWx       Category = "wx"       // 微信协议
	CatBot      Category = "bot"      // QQ/TG 机器人
	CatAPI      Category = "api"      // 通用 HTTP 请求
	CatDB       Category = "db"       // 数据库迁移与异常
	CatTask     Category = "task"     // 通用任务调度
)

// Level 日志级别
type Level string

const (
	LevelDebug Level = "DEBUG"
	LevelInfo  Level = "INFO"
	LevelWarn  Level = "WARN"
	LevelError Level = "ERROR"
)

// CategoryMeta 供 admin 前端展示
type CategoryMeta struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Desc  string `json:"desc"`
}

// AllCategories 全部分类元数据
func AllCategories() []CategoryMeta {
	return []CategoryMeta{
		{Key: string(CatSystem), Label: "系统", Desc: "启动、配置热更新、定时任务"},
		{Key: string(CatAdmin), Label: "管理后台", Desc: "admin 后台 API 操作"},
		{Key: string(CatPortal), Label: "网页门户", Desc: "portal 用户网页操作"},
		{Key: string(CatApp), Label: "App/旧API", Desc: "移动端与 /api/login 等接口"},
		{Key: string(CatUser), Label: "用户指令", Desc: "用户通过机器人发送的指令与业务处理"},
		{Key: string(CatJD), Label: "京东", Desc: "京东任务、查询、登录"},
		{Key: string(CatKuwo), Label: "酷我", Desc: "酷我抢兑模块"},
		{Key: string(CatQinglong), Label: "青龙", Desc: "青龙面板 API"},
		{Key: string(CatSync), Label: "同步", Desc: "数据库与青龙同步服务"},
		{Key: string(CatWx), Label: "微信协议", Desc: "微信协议设备与扫码"},
		{Key: string(CatBot), Label: "机器人", Desc: "QQ/TG/微信Hook 连接、消息通道与推送"},
		{Key: string(CatAPI), Label: "HTTP", Desc: "通用 HTTP 请求追踪"},
		{Key: string(CatDB), Label: "数据库", Desc: "迁移与数据库异常"},
		{Key: string(CatTask), Label: "任务", Desc: "通用任务执行"},
	}
}

// CategoryFromPath 根据请求路径推断分类（HTTP 中间件用）
func CategoryFromPath(path string) Category {
	switch {
	case hasPrefix(path, "/api/admin/"):
		return CatAdmin
	case hasPrefix(path, "/api/portal/"):
		return CatPortal
	case hasPrefix(path, "/api/login/"), hasPrefix(path, "/api/account"), hasPrefix(path, "/api/getUser"):
		return CatApp
	case hasPrefix(path, "/api/wx"), hasPrefix(path, "/wx"):
		return CatWx
	default:
		return CatAPI
	}
}

func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

// Entry 单条结构化日志
type Entry struct {
	ID       int64     `json:"id"`
	Time     time.Time `json:"time"`
	Level    Level     `json:"level"`
	Category Category  `json:"category"`
	Message  string    `json:"message"`
}

// FormatLine 写入文件的统一格式（TAB 分隔，便于解析与 grep）
// 2026-06-30T14:32:01.123+08:00\tINFO\tsystem\t配置已热更新
func (e Entry) FormatLine() string {
	ts := e.Time.Format("2006-01-02T15:04:05.000-07:00")
	msg := strings.ReplaceAll(e.Message, "\t", " ")
	msg = strings.ReplaceAll(msg, "\n", "\\n")
	return fmt.Sprintf("%s\t%s\t%s\t%s", ts, e.Level, e.Category, msg)
}

// DisplayLine 终端/前端展示
func (e Entry) DisplayLine() string {
	ts := e.Time.Format("2006-01-02 15:04:05.000")
	return fmt.Sprintf("%s [%s] [%s] %s", ts, e.Level, e.Category, e.Message)
}

// ParseLine 解析文件行
func ParseLine(line string) (Entry, error) {
	line = strings.TrimSpace(line)
	if line == "" {
		return Entry{}, fmt.Errorf("empty line")
	}
	parts := strings.SplitN(line, "\t", 4)
	if len(parts) < 4 {
		return Entry{}, fmt.Errorf("invalid format")
	}
	t, err := time.Parse("2006-01-02T15:04:05.000-07:00", parts[0])
	if err != nil {
		t, err = time.Parse("2006-01-02T15:04:05.000Z07:00", parts[0])
		if err != nil {
			return Entry{}, err
		}
	}
	msg := strings.ReplaceAll(parts[3], "\\n", "\n")
	return Entry{
		Time:     t,
		Level:    Level(parts[1]),
		Category: Category(parts[2]),
		Message:  msg,
	}, nil
}

// MarshalJSON 供 API 返回
func (e Entry) MarshalJSON() ([]byte, error) {
	type alias struct {
		ID       int64  `json:"id"`
		Time     string `json:"time"`
		Level    string `json:"level"`
		Category string `json:"category"`
		Message  string `json:"message"`
	}
	return json.Marshal(alias{
		ID:       e.ID,
		Time:     e.Time.Format("2006-01-02 15:04:05.000"),
		Level:    string(e.Level),
		Category: string(e.Category),
		Message:  e.Message,
	})
}

const (
	defaultRingSize     = 20000
	defaultMaxDays      = 7
	maxQueryScanTotal   = 10000 // 历史查询最多扫描匹配条数，防止 OOM
	defaultMaxSizeMB    = 200   // 单文件超过则切分
)

var (
	mu           sync.RWMutex
	ring         []Entry
	ringStart    int
	ringCount    int
	nextID       int64
	logDir       string
	currentFile  *os.File
	currentDate  string
	consoleOn    = true
	minLevel     = LevelDebug
	subscribers  []chan Entry
	subMu        sync.Mutex
	maxDays      = defaultMaxDays
	initialized  bool
)

// Config 日志配置
type LogConfig struct {
	LogDir    string
	Console   bool
	MinLevel  Level
	MaxDays   int
	RingSize  int
}

// Init 初始化日志系统（在 init 中自动调用，也可手动传入配置）
func InitLogger(cfg ...LogConfig) {
	mu.Lock()
	defer mu.Unlock()
	if initialized {
		return
	}

	c := LogConfig{
		LogDir:   resolveLogDir(),
		Console:  !isDaemonMode(),
		MinLevel: LevelDebug,
		MaxDays:  defaultMaxDays,
		RingSize: defaultRingSize,
	}
	if len(cfg) > 0 {
		if cfg[0].LogDir != "" {
			c.LogDir = cfg[0].LogDir
		}
		if cfg[0].MaxDays > 0 {
			c.MaxDays = cfg[0].MaxDays
		}
		if cfg[0].RingSize > 0 {
			c.RingSize = cfg[0].RingSize
		}
		if cfg[0].MinLevel != "" {
			c.MinLevel = cfg[0].MinLevel
		}
		// Console 显式传入时尊重；否则按 daemon 判断
		if len(cfg) > 0 {
			c.Console = cfg[0].Console
		}
	}

	logDir = c.LogDir
	consoleOn = c.Console
	minLevel = c.MinLevel
	maxDays = c.MaxDays
	ring = make([]Entry, c.RingSize)

	if err := os.MkdirAll(logDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "[logger] 创建日志目录失败: %v\n", err)
	}
	openDailyFile(time.Now())
	initialized = true

	go startCleanupLoop()
}


func resolveLogDir() string {
	if dir := os.Getenv("XDD_LOG_DIR"); dir != "" {
		return dir
	}
	if exec, err := os.Executable(); err == nil {
		return filepath.Join(filepath.Dir(exec), "logs")
	}
	return "logs"
}

func isDaemonMode() bool {
	for _, arg := range os.Args[1:] {
		if arg == "-d" {
			return true
		}
	}
	return false
}

func levelRank(l Level) int {
	switch l {
	case LevelDebug:
		return 0
	case LevelInfo:
		return 1
	case LevelWarn:
		return 2
	case LevelError:
		return 3
	default:
		return 1
	}
}

func shouldLog(l Level) bool {
	return levelRank(l) >= levelRank(minLevel)
}

// write 核心写入（format + args）
func write(cat Category, level Level, format string, args ...interface{}) {
	writeMessage(cat, level, safeFormat(format, args...))
}

// writeMessage 写入已格式化的消息
func writeMessage(cat Category, level Level, msg string) {
	if !shouldLog(level) {
		return
	}
	msg = sanitize(msg)

	mu.Lock()
	nextID++
	e := Entry{
		ID:       nextID,
		Time:     time.Now(),
		Level:    level,
		Category: cat,
		Message:  msg,
	}
	pushRingLocked(e)
	writeFileLocked(e)
	mu.Unlock()

	if consoleOn {
		fmt.Println(e.DisplayLine())
	}
	broadcast(e)
}

// safeFormat 安全格式化，非法 % 占位符时回退为 Sprint
func safeFormat(format string, args ...interface{}) string {
	if len(args) == 0 {
		return format
	}
	var result string
	ok := func() (ok bool) {
		defer func() {
			if recover() != nil {
				ok = false
			}
		}()
		result = fmt.Sprintf(format, args...)
		return true
	}()
	if ok {
		return result
	}
	return fmt.Sprint(append([]interface{}{format}, args...)...)
}

// formatArgs 兼容 beego logs：支持 Info(err)、Info("a", b) 与 Info("fmt %s", v)
func formatArgs(v ...interface{}) string {
	if len(v) == 0 {
		return ""
	}
	if format, ok := v[0].(string); ok && len(v) > 1 && strings.Contains(format, "%") {
		return safeFormat(format, v[1:]...)
	}
	return fmt.Sprint(v...)
}

func pushRingLocked(e Entry) {
	if len(ring) == 0 {
		return
	}
	idx := (ringStart + ringCount) % len(ring)
	if ringCount < len(ring) {
		ringCount++
	} else {
		ringStart = (ringStart + 1) % len(ring)
	}
	ring[idx] = e
}

func openDailyFile(t time.Time) {
	date := t.Format("2006-01-02")
	if currentFile != nil && currentDate == date {
		return
	}
	if currentFile != nil {
		currentFile.Close()
		currentFile = nil
	}
	currentDate = date
	path := filepath.Join(logDir, "xdd-"+date+".log")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[logger] 打开日志文件失败: %v\n", err)
		return
	}
	currentFile = f
}

func writeFileLocked(e Entry) {
	openDailyFile(e.Time)
	if currentFile == nil {
		return
	}
	line := e.FormatLine() + "\n"
	_, _ = currentFile.WriteString(line)
}

func broadcast(e Entry) {
	subMu.Lock()
	defer subMu.Unlock()
	for _, ch := range subscribers {
		select {
		case ch <- e:
		default:
		}
	}
}

// Subscribe 订阅实时日志（admin SSE）
func Subscribe() chan Entry {
	ch := make(chan Entry, 256)
	subMu.Lock()
	subscribers = append(subscribers, ch)
	subMu.Unlock()
	return ch
}

// Unsubscribe 取消订阅
func Unsubscribe(ch chan Entry) {
	subMu.Lock()
	defer subMu.Unlock()
	for i, s := range subscribers {
		if s == ch {
			subscribers = append(subscribers[:i], subscribers[i+1:]...)
			close(ch)
			break
		}
	}
}

// Recent 获取内存中最近的日志
func Recent(limit int, afterID int64, category Category, level Level, keyword string) []Entry {
	mu.RLock()
	defer mu.RUnlock()
	if limit <= 0 || limit > 5000 {
		limit = 500
	}
	out := make([]Entry, 0, limit)
	kw := strings.ToLower(strings.TrimSpace(keyword))

	for i := ringCount - 1; i >= 0 && len(out) < limit; i-- {
		idx := (ringStart + i) % len(ring)
		e := ring[idx]
		if afterID > 0 && e.ID <= afterID {
			continue
		}
		if category != "" && e.Category != category {
			continue
		}
		if level != "" && e.Level != level {
			continue
		}
		if kw != "" && !strings.Contains(strings.ToLower(e.Message), kw) && !strings.Contains(strings.ToLower(string(e.Category)), kw) {
			continue
		}
		out = append(out, e)
	}
	// 反转为时间正序
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out
}

// QueryFiles 查询历史日志文件中的记录（流式扫描，限制最大条数）
func QueryFiles(dateFrom, dateTo string, category Category, level Level, keyword string, page, limit int) ([]Entry, int, bool, error) {
	if page < 1 {
		page = 1
	}
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	// 未指定日期时默认只查保留期内
	if dateFrom == "" && dateTo == "" {
		dateFrom = time.Now().AddDate(0, 0, -maxDays).Format("2006-01-02")
	}
	files, err := listLogFiles()
	if err != nil {
		return nil, 0, false, err
	}
	var all []Entry
	truncated := false
	kw := strings.ToLower(strings.TrimSpace(keyword))

	for i := len(files) - 1; i >= 0; i-- {
		if len(all) >= maxQueryScanTotal {
			truncated = true
			break
		}
		f := files[i]
		base := filepath.Base(f)
		date := strings.TrimPrefix(strings.TrimSuffix(base, ".log"), "xdd-")
		if dateFrom != "" && date < dateFrom {
			continue
		}
		if dateTo != "" && date > dateTo {
			continue
		}
		remain := maxQueryScanTotal - len(all)
		chunk := scanFileMatches(f, category, level, kw, remain)
		for j := len(chunk) - 1; j >= 0; j-- {
			all = append(all, chunk[j])
			if len(all) >= maxQueryScanTotal {
				truncated = true
				break
			}
		}
	}
	total := len(all)
	start := (page - 1) * limit
	if start >= total {
		return []Entry{}, total, truncated, nil
	}
	end := start + limit
	if end > total {
		end = total
	}
	slice := all[start:end]
	for i, j := 0, len(slice)-1; i < j; i, j = i+1, j-1 {
		slice[i], slice[j] = slice[j], slice[i]
	}
	return slice, total, truncated, nil
}

// scanFileMatches 按行扫描文件，保留最后 max 条匹配（内存友好）
func scanFileMatches(path string, category Category, level Level, kw string, max int) []Entry {
	if max <= 0 {
		return nil
	}
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	matched := make([]Entry, 0, 64)
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		e, err := ParseLine(sc.Text())
		if err != nil {
			continue
		}
		if category != "" && e.Category != category {
			continue
		}
		if level != "" && e.Level != level {
			continue
		}
		if kw != "" && !strings.Contains(strings.ToLower(e.Message), kw) {
			continue
		}
		if len(matched) >= max {
			copy(matched, matched[1:])
			matched[len(matched)-1] = e
		} else {
			matched = append(matched, e)
		}
	}
	return matched
}

// LogFileInfo 日志文件信息
type LogFileInfo struct {
	Name string `json:"name"`
	Date string `json:"date"`
	Size int64  `json:"size"`
}

// ListFiles 列出日志文件
func ListFiles() ([]LogFileInfo, error) {
	paths, err := listLogFiles()
	if err != nil {
		return nil, err
	}
	out := make([]LogFileInfo, 0, len(paths))
	for _, p := range paths {
		st, err := os.Stat(p)
		if err != nil {
			continue
		}
		base := filepath.Base(p)
		date := strings.TrimPrefix(strings.TrimSuffix(base, ".log"), "xdd-")
		out = append(out, LogFileInfo{Name: base, Date: date, Size: st.Size()})
	}
	return out, nil
}

func listLogFiles() ([]string, error) {
	entries, err := os.ReadDir(logDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var files []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasPrefix(name, "xdd-") && strings.HasSuffix(name, ".log") {
			files = append(files, filepath.Join(logDir, name))
		}
		// 兼容旧版 xdd.log
		if name == "xdd.log" {
			files = append(files, filepath.Join(logDir, name))
		}
	}
	sort.Strings(files)
	return files, nil
}

// ReadFileContent 读取指定日志文件内容（原始文本）
func ReadFileContent(filename string, tailLines int) (string, error) {
	if filename == "" || strings.Contains(filename, "..") || strings.Contains(filename, "/") {
		return "", fmt.Errorf("invalid filename")
	}
	path := filepath.Join(logDir, filename)
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	if tailLines <= 0 {
		return string(data), nil
	}
	lines := strings.Split(string(data), "\n")
	if len(lines) <= tailLines {
		return string(data), nil
	}
	return strings.Join(lines[len(lines)-tailLines:], "\n"), nil
}

// Stats 日志统计
type Stats struct {
	RingCount   int            `json:"ringCount"`
	RingCap     int            `json:"ringCap"`
	LogDir      string         `json:"logDir"`
	FileCount   int            `json:"fileCount"`
	TotalSize   int64          `json:"totalSize"`
	MaxDays     int            `json:"maxDays"`
	ByCategory  map[string]int `json:"byCategory"`
}

// GetStats 统计信息
func GetStats() Stats {
	mu.RLock()
	defer mu.RUnlock()
	st := Stats{
		RingCount:  ringCount,
		RingCap:    len(ring),
		LogDir:     logDir,
		MaxDays:    maxDays,
		ByCategory: map[string]int{},
	}
	for i := 0; i < ringCount; i++ {
		idx := (ringStart + i) % len(ring)
		st.ByCategory[string(ring[idx].Category)]++
	}
	files, _ := listLogFiles()
	st.FileCount = len(files)
	for _, f := range files {
		if info, err := os.Stat(f); err == nil {
			st.TotalSize += info.Size()
		}
	}
	return st
}

// Cleanup 清理过期日志
func Cleanup() (int, error) {
	cutoff := time.Now().AddDate(0, 0, -maxDays)
	files, err := listLogFiles()
	if err != nil {
		return 0, err
	}
	removed := 0
	for _, f := range files {
		base := filepath.Base(f)
		if base == "xdd.log" {
			continue // 保留兼容旧文件，由管理员手动处理
		}
		dateStr := strings.TrimPrefix(strings.TrimSuffix(base, ".log"), "xdd-")
		t, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			continue
		}
		if t.Before(cutoff) {
			if err := os.Remove(f); err == nil {
				removed++
			}
		}
	}
	return removed, nil
}

func startCleanupLoop() {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()
	// 启动时也执行一次
	if n, err := Cleanup(); err == nil && n > 0 {
		write(CatSystem, LevelInfo, "自动清理过期日志 %d 个文件", n)
	}
	for range ticker.C {
		if n, err := Cleanup(); err == nil && n > 0 {
			write(CatSystem, LevelInfo, "自动清理过期日志 %d 个文件", n)
		}
	}
}

// SetLogDir 允许 models 初始化后更新日志目录（与 ExecPath 对齐）
func SetLogDir(dir string) {
	if dir == "" {
		return
	}
	mu.Lock()
	defer mu.Unlock()
	logDir = dir
	_ = os.MkdirAll(logDir, 0755)
	openDailyFile(time.Now())
}

// LogDir 当前日志目录
func LogDir() string {
	mu.RLock()
	defer mu.RUnlock()
	return logDir
}

var (
	rePtKey    = regexp.MustCompile(`(?i)(pt_key=)[^;,\s"&]+`)
	rePtPin    = regexp.MustCompile(`(?i)(pt_pin=)[^;,\s"&]+`)
	reWskey    = regexp.MustCompile(`(?i)(wskey=)[^\s"',;]+`)
	reBearer   = regexp.MustCompile(`(?i)(Bearer\s+)[A-Za-z0-9._\-]+`)
	rePhone    = regexp.MustCompile(`(\d{3})\d{4}(\d{4})`)
	rePwd      = regexp.MustCompile(`(?i)(password["']?\s*[:=]\s*["']?)[^"',\s}]+`)
	rePwdPlain = regexp.MustCompile(`(?i)((?:密码|口令|pwd)[：:\s]+)[^\s\\n,]+`)
	reSocksPwd = regexp.MustCompile(`(?i)(Socks5[^\\n]*密码[：:\s]+)[^\s\\n,]+`)
	reLoginPwd = regexp.MustCompile(`(?i)(当前登录密码[：:\s]+)[^\s\\n,]+`)
)

func sanitize(msg string) string {
	if msg == "" {
		return msg
	}
	msg = rePtKey.ReplaceAllString(msg, "${1}***")
	msg = rePtPin.ReplaceAllString(msg, "${1}***")
	msg = reWskey.ReplaceAllString(msg, "${1}***")
	msg = reBearer.ReplaceAllString(msg, "${1}***")
	msg = rePwd.ReplaceAllString(msg, "${1}***")
	msg = rePwdPlain.ReplaceAllString(msg, "${1}***")
	msg = reSocksPwd.ReplaceAllString(msg, "${1}***")
	msg = reLoginPwd.ReplaceAllString(msg, "${1}***")
	msg = rePhone.ReplaceAllString(msg, "${1}****${2}")
	if len(msg) > 8000 {
		msg = msg[:8000] + "...(truncated)"
	}
	return msg
}

// Module 带分类的日志模块
type Module struct {
	category Category
}

func For(cat Category) *Module {
	return &Module{category: cat}
}

func (m *Module) Debugf(format string, args ...interface{}) { write(m.category, LevelDebug, format, args...) }
func (m *Module) Infof(format string, args ...interface{})  { write(m.category, LevelInfo, format, args...) }
func (m *Module) Warnf(format string, args ...interface{})  { write(m.category, LevelWarn, format, args...) }
func (m *Module) Errorf(format string, args ...interface{}) { write(m.category, LevelError, format, args...) }

// 便捷模块
func System() *Module   { return For(CatSystem) }
func Admin() *Module    { return For(CatAdmin) }
func Portal() *Module   { return For(CatPortal) }
func App() *Module      { return For(CatApp) }
func UserLog() *Module  { return For(CatUser) }
func JD() *Module       { return For(CatJD) }
func Kuwo() *Module     { return For(CatKuwo) }
func Qinglong() *Module { return For(CatQinglong) }
func Sync() *Module     { return For(CatSync) }
func Wx() *Module       { return For(CatWx) }
func Bot() *Module      { return For(CatBot) }
func API() *Module      { return For(CatAPI) }
func DB() *Module       { return For(CatDB) }
func TaskLog() *Module  { return For(CatTask) }

// ========== beego logs 兼容层（默认 system 分类）==========

func Info(v ...interface{})                 { writeMessage(CatSystem, LevelInfo, formatArgs(v...)) }
func Error(v ...interface{})                { writeMessage(CatSystem, LevelError, formatArgs(v...)) }
func Warn(v ...interface{})                 { writeMessage(CatSystem, LevelWarn, formatArgs(v...)) }
func Debug(v ...interface{})                { writeMessage(CatSystem, LevelDebug, formatArgs(v...)) }
func Critical(v ...interface{})             { writeMessage(CatSystem, LevelError, formatArgs(v...)) }
func Notice(v ...interface{})              { writeMessage(CatSystem, LevelInfo, formatArgs(v...)) }
func Alert(v ...interface{})               { writeMessage(CatSystem, LevelWarn, formatArgs(v...)) }
func Emergency(v ...interface{})            { writeMessage(CatSystem, LevelError, formatArgs(v...)) }

// Trace 兼容 beego（映射为 Debug）
func Trace(v ...interface{}) { writeMessage(CatSystem, LevelDebug, formatArgs(v...)) }

// Infof 等格式化别名
func Infof(format string, v ...interface{})  { write(CatSystem, LevelInfo, format, v...) }
func Errorf(format string, v ...interface{}) { write(CatSystem, LevelError, format, v...) }
func Warnf(format string, v ...interface{})  { write(CatSystem, LevelWarn, format, v...) }
func Debugf(format string, v ...interface{}) { write(CatSystem, LevelDebug, format, v...) }

// Log 显式分类写入
func Log(cat Category, level Level, format string, args ...interface{}) {
	write(cat, level, format, args...)
}

// Logf 带分类
func Logf(cat Category, level Level, format string, args ...interface{}) {
	write(cat, level, format, args...)
}

// CategoryLabel 中文标签
func CategoryLabel(cat string) string {
	for _, c := range AllCategories() {
		if c.Key == cat {
			return c.Label
		}
	}
	return cat
}

// LevelColor 前端颜色 hint
func LevelColor(level string) string {
	switch strings.ToUpper(level) {
	case "ERROR":
		return "#f56c6c"
	case "WARN":
		return "#e6a23c"
	case "DEBUG":
		return "#909399"
	default:
		return "#67c23a"
	}
}

// stdLogWriter 将标准库 log 输出重定向到统一日志
type stdLogWriter struct {
	category Category
}

func (w *stdLogWriter) Write(p []byte) (n int, err error) {
	msg := string(p)
	if len(msg) > 0 && msg[len(msg)-1] == '\n' {
		msg = msg[:len(msg)-1]
	}
	if msg != "" {
		write(w.category, LevelInfo, "%s", msg)
	}
	return len(p), nil
}

// RedirectStdLog 劫持标准库 log 包
func RedirectStdLog() {
	log.SetOutput(&stdLogWriter{category: CatSystem})
	log.SetFlags(0)
}


// Printf 兼容旧代码 log.Printf
func Printf(format string, args ...interface{}) {
	write(CatSystem, LevelInfo, format, args...)
}

// Println 兼容 fmt.Println 用于日志的场景
func Println(args ...interface{}) {
	writeMessage(CatSystem, LevelInfo, fmt.Sprint(args...))
}

// PrintfCat 带分类的 Printf 替代
func PrintfCat(cat Category, format string, args ...interface{}) {
	write(cat, LevelInfo, format, args...)
}

