package models

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
)

var jdProxyPurchaseDetailRe = regexp.MustCompile(`京东任务代理\s*(\d+)\s*个月.*?到期\s*([0-9:\-\s]+)`)

// PortalJdProxySubscription 网页端京东任务代理订阅（按用户）
type PortalJdProxySubscription struct {
	ID         int       `gorm:"primaryKey;autoIncrement"`
	UserNumber int       `gorm:"uniqueIndex;not null"`
	ExpireAt   time.Time `gorm:"index;not null"`
	UpdatedAt  time.Time
}

func (PortalJdProxySubscription) TableName() string {
	return "portal_jd_proxy_subscription"
}

// GetJdTaskProxyMonthlyCoin 每月代理订阅积分价格
func GetJdTaskProxyMonthlyCoin() int {
	if sysConfig.JdTaskProxyMonthlyCoin < 0 {
		return 0
	}
	return sysConfig.JdTaskProxyMonthlyCoin
}

// IsPortalJdProxyActive 用户是否拥有有效的京东任务代理订阅
func IsPortalJdProxyActive(userNumber int) bool {
	if userNumber <= 0 {
		return false
	}
	var sub PortalJdProxySubscription
	if err := db.Where("user_number = ?", userNumber).First(&sub).Error; err != nil {
		return false
	}
	return sub.ExpireAt.After(time.Now())
}

// PortalJdProxyStatus 门户展示用代理订阅状态
type PortalJdProxyStatus struct {
	Active      bool   `json:"active"`
	ExpireAt    string `json:"expireAt"`
	MonthlyCoin int    `json:"monthlyCoin"`
	UserCoin    int    `json:"userCoin"`
	ProxyReady  bool   `json:"proxyReady"`
}

// ResolvePortalJdTaskProxy 门户订阅用户：手动代理开关开 → 用手动配置；关 → 用系统「全局代理」
func ResolvePortalJdTaskProxy() (enabled bool, url, renum, redelay string) {
	if IsJdManualProxySwitchEnabled() {
		return resolveManualOnlyJdTaskProxy()
	}
	return ResolveSystemJdTaskProxy()
}

// ResolvePortalJdTaskProxyReady 门户是否可购买/使用代理（按当前代理来源判断配置是否齐全）
func ResolvePortalJdTaskProxyReady() bool {
	ok, url, _, _ := ResolvePortalJdTaskProxy()
	return ok && url != ""
}

// GetPortalJdProxyStatus 查询用户代理订阅状态
func GetPortalJdProxyStatus(userNumber int) PortalJdProxyStatus {
	st := PortalJdProxyStatus{
		MonthlyCoin: GetJdTaskProxyMonthlyCoin(),
		UserCoin:    GetCoin(userNumber),
		ProxyReady:  ResolvePortalJdTaskProxyReady(),
	}
	var sub PortalJdProxySubscription
	if err := db.Where("user_number = ?", userNumber).First(&sub).Error; err == nil {
		if sub.ExpireAt.After(time.Now()) {
			st.Active = true
			st.ExpireAt = sub.ExpireAt.Format("2006-01-02 15:04:05")
		}
	}
	return st
}

// ResolveManualJdTaskProxyReady 后台手动任务是否已配置可用代理（含回退系统代理）
func ResolveManualJdTaskProxyReady() bool {
	ok, url, _, _ := ResolveManualJdTaskProxy()
	return ok && url != ""
}

// PurchasePortalJdProxy 使用积分购买/续费京东任务代理
func PurchasePortalJdProxy(userNumber int, months int, clientCtx ClientContext) (PortalJdProxyStatus, error) {
	if months < 1 || months > 36 {
		return PortalJdProxyStatus{}, fmt.Errorf("购买月数需在 1-36 之间")
	}
	monthly := GetJdTaskProxyMonthlyCoin()
	if monthly <= 0 {
		return PortalJdProxyStatus{}, fmt.Errorf("代理订阅暂未开放，请联系管理员")
	}
	if !ResolvePortalJdTaskProxyReady() {
		return PortalJdProxyStatus{}, fmt.Errorf("管理员尚未配置任务代理，暂不可购买")
	}
	total := monthly * months
	if err := DeductCoinChecked(userNumber, total); err != nil {
		return PortalJdProxyStatus{}, err
	}

	now := time.Now()
	var sub PortalJdProxySubscription
	err := db.Where("user_number = ?", userNumber).First(&sub).Error
	base := now
	if err == nil && sub.ExpireAt.After(now) {
		base = sub.ExpireAt
	}
	newExpire := base.AddDate(0, months, 0)

	if err != nil {
		sub = PortalJdProxySubscription{
			UserNumber: userNumber,
			ExpireAt:   newExpire,
			UpdatedAt:  now,
		}
		if createErr := db.Create(&sub).Error; createErr != nil {
			db.Model(&User{}).Where("number = ?", userNumber).Update("coin", gorm.Expr(fmt.Sprintf("coin+%d", total)))
			return PortalJdProxyStatus{}, createErr
		}
	} else {
		if updateErr := db.Model(&sub).Updates(map[string]interface{}{
			"expire_at":  newExpire,
			"updated_at": now,
		}).Error; updateErr != nil {
			db.Model(&User{}).Where("number = ?", userNumber).Update("coin", gorm.Expr(fmt.Sprintf("coin+%d", total)))
			return PortalJdProxyStatus{}, updateErr
		}
	}

	detail := fmt.Sprintf("京东任务代理 %d 个月，到期 %s", months, newExpire.Format("2006-01-02"))
	RecordCoinLogEx(userNumber, -total, "代理订阅", detail, clientCtx)
	return GetPortalJdProxyStatus(userNumber), nil
}

// ApplyPortalJdTaskProxyEnvs 门户任务：订阅有效时按手动/系统代理来源注入
func ApplyPortalJdTaskProxyEnvs(userNumber int, envs map[string]string) {
	if envs == nil || !IsPortalJdProxyActive(userNumber) {
		return
	}
	ok, url, renum, redelay := ResolvePortalJdTaskProxy()
	if !ok || url == "" {
		return
	}
	applyJdProxyEnvsFromConfig(envs, url, renum, redelay)
}

// AdminJdProxyPurchaseStats 门户任务代理购买汇总
type AdminJdProxyPurchaseStats struct {
	TotalPurchases int   `json:"totalPurchases"`
	TotalCoin      int   `json:"totalCoin"`
	UniqueUsers    int   `json:"uniqueUsers"`
	ActiveCount    int   `json:"activeCount"`
}

// AdminJdProxyPurchaseItem 单次代理购买记录（来自 coin_log）
type AdminJdProxyPurchaseItem struct {
	ID              int    `json:"id"`
	UserNumber      int    `json:"userNumber"`
	Username        string `json:"username"`
	Nickname        string `json:"nickname"`
	QQ              string `json:"qq"`
	Coin            int    `json:"coin"`
	Months          int    `json:"months"`
	ExpireAt        string `json:"expireAt"`
	CurrentExpireAt string `json:"currentExpireAt"`
	CurrentlyActive bool   `json:"currentlyActive"`
	ClientSource    string `json:"clientSource"`
	ClientPlatform  string `json:"clientPlatform"`
	SourceLabel     string `json:"sourceLabel"`
	Detail          string `json:"detail"`
	PurchasedAt     string `json:"purchasedAt"`
}

func parseJdProxyPurchaseDetail(detail string) (months int, expireAt string) {
	m := jdProxyPurchaseDetailRe.FindStringSubmatch(strings.TrimSpace(detail))
	if len(m) < 3 {
		return 0, ""
	}
	months, _ = strconv.Atoi(m[1])
	expireAt = strings.TrimSpace(m[2])
	if len(expireAt) > 10 {
		expireAt = expireAt[:10]
	}
	return months, expireAt
}

func buildAdminJdProxyPurchaseQuery(userNumber int, days int) *gorm.DB {
	query := db.Model(&CoinLog{}).Where("type = ?", "代理订阅")
	if userNumber > 0 {
		query = query.Where("user_number = ?", userNumber)
	}
	if days > 0 {
		since := time.Now().AddDate(0, 0, -days)
		query = query.Where("created_at >= ?", since)
	}
	return query
}

// GetAdminJdProxyPurchaseStats 代理购买汇总（可按用户/天数筛选）
func GetAdminJdProxyPurchaseStats(userNumber int, days int) AdminJdProxyPurchaseStats {
	stats := AdminJdProxyPurchaseStats{}
	base := buildAdminJdProxyPurchaseQuery(userNumber, days)
	var total int64
	base.Count(&total)
	stats.TotalPurchases = int(total)

	var sumCoin int64
	buildAdminJdProxyPurchaseQuery(userNumber, days).Select("COALESCE(SUM(ABS(amount)),0)").Scan(&sumCoin)
	stats.TotalCoin = int(sumCoin)

	var unique int64
	buildAdminJdProxyPurchaseQuery(userNumber, days).Distinct("user_number").Count(&unique)
	stats.UniqueUsers = int(unique)

	now := time.Now()
	subQuery := db.Model(&PortalJdProxySubscription{}).Where("expire_at > ?", now)
	if userNumber > 0 {
		subQuery = subQuery.Where("user_number = ?", userNumber)
	}
	var active int64
	subQuery.Count(&active)
	stats.ActiveCount = int(active)
	return stats
}

// GetAdminJdProxyPurchases 代理购买记录列表
func GetAdminJdProxyPurchases(userNumber int, days int, page int, limit int) ([]AdminJdProxyPurchaseItem, int64, AdminJdProxyPurchaseStats) {
	if page < 1 {
		page = 1
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	stats := GetAdminJdProxyPurchaseStats(userNumber, days)

	var total int64
	buildAdminJdProxyPurchaseQuery(userNumber, days).Count(&total)
	if total == 0 {
		return []AdminJdProxyPurchaseItem{}, 0, stats
	}

	var logs []CoinLog
	buildAdminJdProxyPurchaseQuery(userNumber, days).
		Order("id desc").
		Offset((page - 1) * limit).
		Limit(limit).
		Find(&logs)

	userNumbers := make([]int, 0, len(logs))
	seen := map[int]bool{}
	for _, log := range logs {
		if log.UserNumber > 0 && !seen[log.UserNumber] {
			seen[log.UserNumber] = true
			userNumbers = append(userNumbers, log.UserNumber)
		}
	}

	usernameMap := map[int]string{}
	if len(userNumbers) > 0 {
		var accounts []WebUserAccount
		db.Where("user_number IN ?", userNumbers).Find(&accounts)
		for _, acc := range accounts {
			usernameMap[acc.UserNumber] = acc.Username
		}
	}

	userMap := map[int]*User{}
	if len(userNumbers) > 0 {
		var users []User
		db.Where("number IN ?", userNumbers).Find(&users)
		for i := range users {
			userMap[users[i].Number] = &users[i]
		}
	}

	subMap := map[int]PortalJdProxySubscription{}
	if len(userNumbers) > 0 {
		var subs []PortalJdProxySubscription
		db.Where("user_number IN ?", userNumbers).Find(&subs)
		for _, sub := range subs {
			subMap[sub.UserNumber] = sub
		}
	}

	now := time.Now()
	items := make([]AdminJdProxyPurchaseItem, 0, len(logs))
	for _, log := range logs {
		ctx, detail := ResolveCoinLogContext(log)
		months, expireAt := parseJdProxyPurchaseDetail(detail)
		item := AdminJdProxyPurchaseItem{
			ID:             log.ID,
			UserNumber:     log.UserNumber,
			Username:       usernameMap[log.UserNumber],
			Coin:           -log.Amount,
			Months:         months,
			ExpireAt:       expireAt,
			Detail:         detail,
			ClientSource:   ctx.Source,
			ClientPlatform: ctx.Platform,
			SourceLabel:    ctx.AdminLabel(),
			PurchasedAt:    log.CreatedAt.Format("2006-01-02 15:04:05"),
		}
		if u := userMap[log.UserNumber]; u != nil {
			item.Nickname = u.Nickname
			item.QQ = u.QQ
		}
		if sub, ok := subMap[log.UserNumber]; ok {
			item.CurrentExpireAt = sub.ExpireAt.Format("2006-01-02 15:04:05")
			item.CurrentlyActive = sub.ExpireAt.After(now)
		}
		items = append(items, item)
	}
	return items, total, stats
}

// ===================== 门户京东任务自动执行（代理订阅用户） =====================

// PortalJdAutoSetting 用户自动执行配置（按门户账号 + 任务列表）
type PortalJdAutoSetting struct {
	UserNumber     int       `gorm:"primaryKey"`
	AccountIndexes string    `gorm:"type:text"` // json []int，与手动执行 chips 一致
	TasksJSON      string    `gorm:"type:text"` // json []PortalJdAutoTaskEntry
	UpdatedAt      time.Time
}

func (PortalJdAutoSetting) TableName() string { return "portal_jd_auto_setting" }

// PortalJdAutoTaskEntry 单任务自动配置
type PortalJdAutoTaskEntry struct {
	TaskID       string `json:"taskId"`
	Enabled      bool   `json:"enabled"`
	RunHour      int    `json:"runHour"`      // 用户输入 1-24
	AdjustedHour int    `json:"adjustedHour"` // 错峰后实际整点
	LastRunDate  string `json:"lastRunDate,omitempty"`
}

// PortalJdAutoConfigView 门户展示
type PortalJdAutoConfigView struct {
	AccountIndexes []int                   `json:"accountIndexes"`
	Tasks          []PortalJdAutoTaskEntry `json:"tasks"`
	Preview        []PortalJdAutoPreview   `json:"preview,omitempty"`
}

type PortalJdAutoPreview struct {
	TaskID       string `json:"taskId"`
	TaskName     string `json:"taskName"`
	RunHour      int    `json:"runHour"`
	AdjustedHour int    `json:"adjustedHour"`
}

func jdPortalBeijingNow() time.Time {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		loc = time.FixedZone("CST", 8*3600)
	}
	return time.Now().In(loc)
}

func jdAutoHourToClock(h int) int {
	if h == 24 {
		return 0
	}
	if h < 1 {
		return 1
	}
	if h > 23 {
		return 23
	}
	return h
}

func computeJdAutoAdjustedHours(entries []PortalJdAutoTaskEntry) ([]PortalJdAutoTaskEntry, error) {
	enabled := make([]PortalJdAutoTaskEntry, 0)
	for _, e := range entries {
		if !e.Enabled {
			continue
		}
		if e.RunHour < 1 || e.RunHour > 24 {
			return nil, fmt.Errorf("任务 %s 的整点需在 1-24 之间", e.TaskID)
		}
		enabled = append(enabled, e)
	}
	sort.Slice(enabled, func(i, j int) bool {
		if enabled[i].RunHour == enabled[j].RunHour {
			return enabled[i].TaskID < enabled[j].TaskID
		}
		return enabled[i].RunHour < enabled[j].RunHour
	})
	last := 0
	for i := range enabled {
		h := enabled[i].RunHour
		if i > 0 && h <= last {
			h = last + 1
		}
		if h > 24 {
			return nil, fmt.Errorf("自动任务过多，同账号无法保证至少间隔 1 小时，请减少勾选或调整整点")
		}
		enabled[i].AdjustedHour = h
		last = h
	}
	adjMap := make(map[string]int, len(enabled))
	lastDate := make(map[string]string)
	for _, e := range entries {
		lastDate[e.TaskID] = e.LastRunDate
	}
	for _, e := range enabled {
		adjMap[e.TaskID] = e.AdjustedHour
	}
	out := make([]PortalJdAutoTaskEntry, len(entries))
	for i, e := range entries {
		out[i] = e
		if e.Enabled {
			out[i].AdjustedHour = adjMap[e.TaskID]
		}
		out[i].LastRunDate = lastDate[e.TaskID]
	}
	return out, nil
}

func mergeJdAutoTasksWithCatalog(userNumber int, stored []PortalJdAutoTaskEntry) []PortalJdAutoTaskEntry {
	catalog := GetJdManualTaskList(true)
	byID := make(map[string]PortalJdAutoTaskEntry, len(stored))
	for _, e := range stored {
		byID[e.TaskID] = e
	}
	out := make([]PortalJdAutoTaskEntry, 0, len(catalog))
	for _, t := range catalog {
		e, ok := byID[t.ID]
		if !ok {
			e = PortalJdAutoTaskEntry{TaskID: t.ID, Enabled: false, RunHour: 8}
		}
		e.TaskID = t.ID
		out = append(out, e)
	}
	return out
}

func buildJdAutoPreview(tasks []PortalJdAutoTaskEntry) []PortalJdAutoPreview {
	prev := make([]PortalJdAutoPreview, 0)
	for _, e := range tasks {
		if !e.Enabled {
			continue
		}
		name := e.TaskID
		if t, ok := getJdManualTaskByID(e.TaskID); ok {
			name = t.Name
		}
		prev = append(prev, PortalJdAutoPreview{
			TaskID:       e.TaskID,
			TaskName:     name,
			RunHour:      e.RunHour,
			AdjustedHour: e.AdjustedHour,
		})
	}
	sort.Slice(prev, func(i, j int) bool {
		if prev[i].AdjustedHour == prev[j].AdjustedHour {
			return prev[i].TaskID < prev[j].TaskID
		}
		return prev[i].AdjustedHour < prev[j].AdjustedHour
	})
	return prev
}

// GetPortalJdAutoConfig 获取自动执行配置（无代理时返回空）
func GetPortalJdAutoConfig(userNumber int) (PortalJdAutoConfigView, error) {
	view := PortalJdAutoConfigView{
		AccountIndexes: []int{0},
		Tasks:          mergeJdAutoTasksWithCatalog(userNumber, nil),
	}
	if !IsPortalJdProxyActive(userNumber) {
		return view, nil
	}
	var row PortalJdAutoSetting
	if err := db.Where("user_number = ?", userNumber).First(&row).Error; err != nil {
		if computed, err2 := computeJdAutoAdjustedHours(view.Tasks); err2 == nil {
			view.Tasks = computed
		}
		view.Preview = buildJdAutoPreview(view.Tasks)
		return view, nil
	}
	if row.AccountIndexes != "" {
		_ = json.Unmarshal([]byte(row.AccountIndexes), &view.AccountIndexes)
	}
	var stored []PortalJdAutoTaskEntry
	if row.TasksJSON != "" {
		_ = json.Unmarshal([]byte(row.TasksJSON), &stored)
	}
	view.Tasks = mergeJdAutoTasksWithCatalog(userNumber, stored)
	if computed, err := computeJdAutoAdjustedHours(view.Tasks); err == nil {
		view.Tasks = computed
	}
	view.Preview = buildJdAutoPreview(view.Tasks)
	return view, nil
}

// SavePortalJdAutoConfig 保存自动执行配置
func SavePortalJdAutoConfig(userNumber int, accountIndexes []int, tasks []PortalJdAutoTaskEntry) (PortalJdAutoConfigView, error) {
	if !IsPortalJdProxyActive(userNumber) {
		return PortalJdAutoConfigView{}, fmt.Errorf("任务代理未开通或已过期")
	}
	if len(accountIndexes) == 0 {
		return PortalJdAutoConfigView{}, fmt.Errorf("请至少选择一个执行账号")
	}
	merged := mergeJdAutoTasksWithCatalog(userNumber, tasks)
	byIncoming := make(map[string]PortalJdAutoTaskEntry, len(tasks))
	for _, t := range tasks {
		byIncoming[t.TaskID] = t
	}
	for i := range merged {
		if inc, ok := byIncoming[merged[i].TaskID]; ok {
			merged[i].Enabled = inc.Enabled
			merged[i].RunHour = inc.RunHour
			merged[i].LastRunDate = inc.LastRunDate
		}
	}
	computed, err := computeJdAutoAdjustedHours(merged)
	if err != nil {
		return PortalJdAutoConfigView{}, err
	}
	accJSON, _ := json.Marshal(accountIndexes)
	taskJSON, _ := json.Marshal(computed)
	now := time.Now()
	row := PortalJdAutoSetting{
		UserNumber:     userNumber,
		AccountIndexes: string(accJSON),
		TasksJSON:      string(taskJSON),
		UpdatedAt:      now,
	}
	if err := db.Save(&row).Error; err != nil {
		return PortalJdAutoConfigView{}, err
	}
	return PortalJdAutoConfigView{
		AccountIndexes: accountIndexes,
		Tasks:          computed,
		Preview:        buildJdAutoPreview(computed),
	}, nil
}

func markJdAutoTaskRunToday(userNumber int, taskID string) {
	var row PortalJdAutoSetting
	if err := db.Where("user_number = ?", userNumber).First(&row).Error; err != nil {
		return
	}
	var tasks []PortalJdAutoTaskEntry
	if row.TasksJSON != "" {
		_ = json.Unmarshal([]byte(row.TasksJSON), &tasks)
	}
	today := jdPortalBeijingNow().Format("2006-01-02")
	changed := false
	for i := range tasks {
		if tasks[i].TaskID == taskID {
			tasks[i].LastRunDate = today
			changed = true
		}
	}
	if !changed {
		return
	}
	data, _ := json.Marshal(tasks)
	db.Model(&row).Updates(map[string]interface{}{"tasks_json": string(data), "updated_at": time.Now()})
}

// RunJdPortalAutoTasksTick 每分钟检查并触发自动任务
func RunJdPortalAutoTasksTick() {
	now := jdPortalBeijingNow()
	if now.Minute() != 0 {
		return
	}
	clockHour := now.Hour()

	var subs []PortalJdProxySubscription
	db.Where("expire_at > ?", time.Now()).Find(&subs)
	for _, sub := range subs {
		runJdAutoForUser(sub.UserNumber, clockHour, now)
	}
}

func runJdAutoForUser(userNumber int, clockHour int, now time.Time) {
	if !IsPortalJdProxyActive(userNumber) {
		return
	}
	cfg, err := GetPortalJdAutoConfig(userNumber)
	if err != nil {
		return
	}
	today := now.Format("2006-01-02")
	for _, task := range cfg.Tasks {
		if !task.Enabled {
			continue
		}
		if jdAutoHourToClock(task.AdjustedHour) != clockHour {
			continue
		}
		if task.LastRunDate == today {
			continue
		}
		taskName := task.TaskID
		if t, ok := getJdManualTaskByID(task.TaskID); ok {
			taskName = t.Name
		}
		if err := TriggerPortalJdAutoTask(userNumber, task.TaskID, taskName, cfg.AccountIndexes); err != nil {
			JD().Warnf("[京东自动任务] user=%d task=%s err=%v", userNumber, task.TaskID, err)
			continue
		}
		markJdAutoTaskRunToday(userNumber, task.TaskID)
	}
}

// TriggerPortalJdAutoTask 触发单次自动执行
func TriggerPortalJdAutoTask(userNumber int, taskID, taskName string, accountIndexes []int) error {
	if !IsPortalJdProxyActive(userNumber) {
		return fmt.Errorf("任务代理未开通或已过期")
	}
	task, ok := getJdManualTaskByID(taskID)
	if !ok || !task.Enabled {
		return fmt.Errorf("任务不存在或未启用")
	}
	displayName := task.Name
	if strings.TrimSpace(taskName) != "" {
		displayName = taskName
	}
	taskLogID := fmt.Sprintf("auto_%s_%d_%d", taskID, userNumber, time.Now().UnixNano())
	CreateTaskLogChannel(taskLogID)
	recID, err := CreatePortalJdRunRecord(userNumber, taskID, displayName, taskLogID, "auto")
	if err != nil {
		RemoveTaskLogChannel(taskLogID)
		return err
	}
	ctx := ClientContext{Source: ClientSourceWeb, Platform: ClientPlatformWeb}
	if err := SubmitPortalJdTask(userNumber, taskID, displayName, accountIndexes, taskLogID, ctx, "auto", recID); err != nil {
		FinishPortalJdRunRecord(recID, "failed", err.Error())
		RemoveTaskLogChannel(taskLogID)
		return err
	}
	JD().Infof("[京东自动任务] 已触发 user=%d task=%s accounts=%v", userNumber, taskID, accountIndexes)
	return nil
}
