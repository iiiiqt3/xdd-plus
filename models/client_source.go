package models

import (
	"strings"
	"time"
)

// =============================================================================
// 客户端来源规范（全端统一，新增功能必须遵守）
// =============================================================================
//
// HTTP Header:
//   X-Request-Source:  web | app | bot          （Portal/App 必传；旧客户端可省略，后端 UA 兜底）
//   X-Client-Platform: web | ios | android     （App 必传；网页传 web）
//
// Portal 控制器: NextPrepare 赋值 c.ClientCtx；业务用 c.RecordPortalCoin / c.RecordPortalEvent
// 积分写入: RecordCoinLogEx(..., ctx) 或 RecordCoinLog(..., ctx)
// 机器人: sender.ClientContext() / BotContext() / WxBotContext() / AdminContext()
// =============================================================================

const (
	ClientSourceWeb   = "web"
	ClientSourceApp   = "app"
	ClientSourceBot   = "bot"
	ClientSourceWx    = "wx"
	ClientSourceAdmin = "admin"
)

const (
	ClientPlatformWeb     = "web"
	ClientPlatformIOS     = "ios"
	ClientPlatformAndroid = "android"
)

// 通用来源事件类型（Admin 统计 / 审计）
const (
	SourceEventFeedback      = "feedback"
	SourceEventKuwoWithdraw  = "kuwo_withdraw"
	SourceEventKuwoSchedule  = "kuwo_schedule"
	SourceEventKuwoLogin     = "kuwo_login"
	SourceEventKuwoSms       = "kuwo_sms"
	SourceEventWxScanLogin   = "wx_scan_login"
)

// ClientContext 请求端上下文
type ClientContext struct {
	Source   string `json:"source"`
	Platform string `json:"platform"`
}

// ClientSourceEvent 带来源的非积分操作记录（反馈、酷我等）
type ClientSourceEvent struct {
	ID             int       `gorm:"primaryKey" json:"id"`
	UserNumber     int       `gorm:"index;not null" json:"userNumber"`
	EventType      string    `gorm:"size:32;index;not null" json:"eventType"`
	ClientSource   string    `gorm:"size:16;index" json:"clientSource"`
	ClientPlatform string    `gorm:"size:16" json:"clientPlatform"`
	CreatedAt      time.Time `gorm:"index" json:"createdAt"`
}

func (ClientSourceEvent) TableName() string { return "client_source_event" }

// ResolveClientContext 从 HTTP 头解析客户端来源（Portal API 通用入口）
func ResolveClientContext(requestSource, userAgent, signDeviceID, clientPlatform string) ClientContext {
	src := strings.ToLower(strings.TrimSpace(requestSource))
	switch src {
	case ClientSourceWeb, ClientSourceApp, ClientSourceBot:
	default:
		ua := strings.ToLower(userAgent)
		if strings.Contains(ua, "okhttp") || strings.TrimSpace(signDeviceID) != "" {
			src = ClientSourceApp
		} else {
			src = ClientSourceWeb
		}
	}

	platform := strings.ToLower(strings.TrimSpace(clientPlatform))
	switch platform {
	case ClientPlatformIOS, ClientPlatformAndroid, ClientPlatformWeb:
	default:
		switch src {
		case ClientSourceWeb:
			platform = ClientPlatformWeb
		case ClientSourceBot, ClientSourceWx, ClientSourceAdmin:
			platform = ""
		case ClientSourceApp:
			// 未声明平台时留空，展示为「App」
		}
	}

	return ClientContext{Source: src, Platform: platform}
}

func (c ClientContext) IsZero() bool {
	return strings.TrimSpace(c.Source) == ""
}

func (c ClientContext) normalized() ClientContext {
	if c.IsZero() {
		return ClientContext{Source: ClientSourceAdmin}
	}
	out := c
	out.Source = strings.ToLower(strings.TrimSpace(out.Source))
	out.Platform = strings.ToLower(strings.TrimSpace(out.Platform))
	return out
}

// WithDefault Portal 业务默认 web（解析失败时）
func (c ClientContext) WithDefault() ClientContext {
	if c.IsZero() {
		return WebContext()
	}
	return c.normalized()
}

// LegacyLabel 历史积分详情前缀：Web端 / App端 / 机器人 / 微信 / 后台
func (c ClientContext) LegacyLabel() string {
	switch c.normalized().Source {
	case ClientSourceApp:
		return "App端"
	case ClientSourceBot:
		return "机器人"
	case ClientSourceWx:
		return "微信"
	case ClientSourceAdmin:
		return "后台"
	default:
		return "Web端"
	}
}

// FilterKey portal/admin 筛选用的标准 key
func (c ClientContext) FilterKey() string {
	switch c.normalized().Source {
	case ClientSourceApp, ClientSourceBot, ClientSourceWx, ClientSourceAdmin:
		return c.normalized().Source
	default:
		return ClientSourceWeb
	}
}

// AdminLabel admin 列表展示
func (c ClientContext) AdminLabel() string {
	switch c.normalized().Source {
	case ClientSourceApp:
		switch c.Platform {
		case ClientPlatformIOS:
			return "App(iOS)"
		case ClientPlatformAndroid:
			return "App(安卓)"
		default:
			return "App"
		}
	case ClientSourceBot:
		return "机器人"
	case ClientSourceWx:
		return "微信"
	case ClientSourceAdmin:
		return "后台"
	default:
		return "网页"
	}
}

// AdminTagClass admin 页面 tag 样式
func (c ClientContext) AdminTagClass() string {
	switch c.normalized().Source {
	case ClientSourceApp:
		return "tag-info"
	case ClientSourceBot:
		return "tag-warning"
	case ClientSourceWx:
		return "tag-success"
	case ClientSourceAdmin:
		return ""
	default:
		return "tag-success"
	}
}

// LogCategory 统一日志分类（与 logger Category 对齐）
func (c ClientContext) LogCategory() Category {
	switch c.normalized().Source {
	case ClientSourceApp:
		return CatApp
	case ClientSourceBot:
		return CatBot
	case ClientSourceWx:
		return CatWx
	default:
		return CatPortal
	}
}

func BotContext() ClientContext {
	return ClientContext{Source: ClientSourceBot}
}

func WxBotContext() ClientContext {
	return ClientContext{Source: ClientSourceWx}
}

func AdminContext() ClientContext {
	return ClientContext{Source: ClientSourceAdmin}
}

func WebContext() ClientContext {
	return ClientContext{Source: ClientSourceWeb, Platform: ClientPlatformWeb}
}

// NormalizeSourceFilter 兼容旧版筛选值（Web端/App端）与新 key（web/app）
func NormalizeSourceFilter(filter string) string {
	switch strings.TrimSpace(filter) {
	case "Web端", "web":
		return ClientSourceWeb
	case "App端", "app":
		return ClientSourceApp
	case "微信", "wx":
		return ClientSourceWx
	case "机器人", "bot":
		return ClientSourceBot
	case "后台及其他", "后台", "admin":
		return ClientSourceAdmin
	default:
		return ""
	}
}

// NormalizeStoredSource 读取 DB 中历史 source 字段
func NormalizeStoredSource(source string) ClientContext {
	src := NormalizeSourceFilter(source)
	if src == "" {
		src = strings.ToLower(strings.TrimSpace(source))
	}
	switch src {
	case ClientSourceApp, ClientSourceBot, ClientSourceWx, ClientSourceAdmin, ClientSourceWeb:
		return ClientContext{Source: src}
	default:
		if source == "app" {
			return ClientContext{Source: ClientSourceApp}
		}
		return WebContext()
	}
}

// CoinLogContextFromLegacyDetail 从旧版 detail 前缀推断来源（只读兼容）
func CoinLogContextFromLegacyDetail(detail string) (ClientContext, string) {
	d := detail
	switch {
	case strings.HasPrefix(d, "Web端"):
		return WebContext(), strings.TrimPrefix(d, "Web端")
	case strings.HasPrefix(d, "App端"):
		return ClientContext{Source: ClientSourceApp}, strings.TrimPrefix(d, "App端")
	case strings.HasPrefix(d, "机器人"):
		return BotContext(), strings.TrimPrefix(d, "机器人")
	case strings.HasPrefix(d, "微信"):
		return WxBotContext(), strings.TrimPrefix(d, "微信")
	case strings.HasPrefix(d, "后台"):
		return AdminContext(), strings.TrimPrefix(d, "后台")
	default:
		return ClientContext{}, d
	}
}

// ResolveCoinLogContext 读取 coin_log 时解析来源（结构化字段优先）
func ResolveCoinLogContext(log CoinLog) (ctx ClientContext, cleanDetail string) {
	if strings.TrimSpace(log.ClientSource) != "" {
		return ClientContext{Source: log.ClientSource, Platform: log.ClientPlatform}, log.Detail
	}
	return CoinLogContextFromLegacyDetail(log.Detail)
}

// CoinLogFilterLabel 供 portal 筛选 Tab 展示
func CoinLogFilterLabel(key string) string {
	switch NormalizeSourceFilter(key) {
	case ClientSourceWeb:
		return "网页"
	case ClientSourceApp:
		return "App"
	case ClientSourceWx:
		return "微信"
	case ClientSourceBot:
		return "机器人"
	case ClientSourceAdmin:
		return "后台及其他"
	default:
		return "全部"
	}
}

// RecordClientSourceEvent 记录带来源的非积分操作（酷我、反馈等）
func RecordClientSourceEvent(userNumber int, eventType string, ctx ClientContext) {
	if userNumber <= 0 || strings.TrimSpace(eventType) == "" {
		return
	}
	ctx = ctx.WithDefault()
	if err := db.Create(&ClientSourceEvent{
		UserNumber:     userNumber,
		EventType:      eventType,
		ClientSource:   ctx.Source,
		ClientPlatform: ctx.Platform,
		CreatedAt:      time.Now(),
	}).Error; err != nil {
		Warn("[来源事件] 记录失败 user=%d type=%s: %v", userNumber, eventType, err)
	}
}

// RecordCoinForSender 机器人侧积分变动（自动识别 QQ/微信/TG）
func RecordCoinForSender(sender *Sender, userNumber, amount int, typ, detail string) {
	ctx := BotContext()
	if sender != nil {
		ctx = sender.ClientContext()
	}
	RecordCoinLogEx(userNumber, amount, typ, detail, ctx)
}

// =============================================================================
// Admin 来源统计
// =============================================================================

// SourceCountItem 按来源统计项
type SourceCountItem struct {
	Source   string `json:"source"`
	Platform string `json:"platform"`
	Label    string `json:"label"`
	Count    int    `json:"count"`
}

// ClientSourceStatsReport admin 来源统计
type ClientSourceStatsReport struct {
	Days           int               `json:"days"`
	TodayCheckIn   []SourceCountItem `json:"todayCheckIn"`
	CheckIn        []SourceCountItem `json:"checkIn"`
	Pray           []SourceCountItem `json:"pray"`
	Feedback       []SourceCountItem `json:"feedback"`
	Kuwo           []SourceCountItem `json:"kuwo"`
	CoinOperations []SourceCountItem `json:"coinOperations"`
}

func buildSourceCountItems(rows map[string]map[string]int) []SourceCountItem {
	items := make([]SourceCountItem, 0, len(rows))
	for source, platforms := range rows {
		for platform, count := range platforms {
			ctx := ClientContext{Source: source, Platform: platform}
			items = append(items, SourceCountItem{
				Source:   source,
				Platform: platform,
				Label:    ctx.AdminLabel(),
				Count:    count,
			})
		}
	}
	return items
}

func addSourceCount(bucket map[string]map[string]int, source, platform string, n int) {
	if bucket[source] == nil {
		bucket[source] = make(map[string]int)
	}
	bucket[source][platform] += n
}

func addSourceCountFromCtx(bucket map[string]map[string]int, ctx ClientContext, n int) {
	ctx = ctx.WithDefault()
	addSourceCount(bucket, ctx.Source, ctx.Platform, n)
}

// GetClientSourceStats 获取打卡/祈福/反馈/酷我/积分操作来源统计
func GetClientSourceStats(days int) ClientSourceStatsReport {
	if days <= 0 {
		days = 7
	}
	if days > 90 {
		days = 90
	}
	since := time.Now().AddDate(0, 0, -days)
	today := time.Now().Format("2006-01-02")

	report := ClientSourceStatsReport{Days: days}

	// 今日打卡
	var todayRows []PortalCheckInRecord
	db.Where("check_in_date = ?", today).Find(&todayRows)
	todayBucket := make(map[string]map[string]int)
	for _, row := range todayRows {
		addSourceCount(todayBucket, row.ClientSource, row.ClientPlatform, 1)
	}
	report.TodayCheckIn = buildSourceCountItems(todayBucket)

	// 近 N 日打卡
	var checkInRows []PortalCheckInRecord
	db.Where("created_at >= ?", since).Find(&checkInRows)
	checkInBucket := make(map[string]map[string]int)
	for _, row := range checkInRows {
		addSourceCount(checkInBucket, row.ClientSource, row.ClientPlatform, 1)
	}
	report.CheckIn = buildSourceCountItems(checkInBucket)

	// 近 N 日祈福
	var prayRows []PortalPrayRecord
	db.Where("created_at >= ?", since).Find(&prayRows)
	prayBucket := make(map[string]map[string]int)
	for _, row := range prayRows {
		src := row.ClientSource
		if src == "" {
			src = ClientSourceWeb
		}
		addSourceCount(prayBucket, src, row.ClientPlatform, 1)
	}
	report.Pray = buildSourceCountItems(prayBucket)

	// 近 N 日反馈
	var feedbackRows []AppFeedback
	db.Where("created_at >= ?", since).Find(&feedbackRows)
	feedbackBucket := make(map[string]map[string]int)
	for _, row := range feedbackRows {
		ctx := ClientContext{Source: row.Source, Platform: row.ClientPlatform}
		if row.Source == "" {
			ctx = ClientContext{Source: ClientSourceApp}
		} else if row.ClientPlatform == "" {
			ctx = NormalizeStoredSource(row.Source)
		}
		addSourceCountFromCtx(feedbackBucket, ctx, 1)
	}
	report.Feedback = buildSourceCountItems(feedbackBucket)

	// 近 N 日酷我操作
	var kuwoRows []ClientSourceEvent
	db.Where("created_at >= ? AND event_type IN ?", since, []string{
		SourceEventKuwoWithdraw, SourceEventKuwoSchedule, SourceEventKuwoLogin, SourceEventKuwoSms,
	}).Find(&kuwoRows)
	kuwoBucket := make(map[string]map[string]int)
	for _, row := range kuwoRows {
		addSourceCount(kuwoBucket, row.ClientSource, row.ClientPlatform, 1)
	}
	report.Kuwo = buildSourceCountItems(kuwoBucket)

	// 近 N 日积分变动
	var coinLogs []CoinLog
	db.Where("created_at >= ?", since).Order("id desc").Limit(50000).Find(&coinLogs)
	coinBucket := make(map[string]map[string]int)
	for _, log := range coinLogs {
		ctx, _ := ResolveCoinLogContext(log)
		if ctx.IsZero() {
			ctx = AdminContext()
		}
		addSourceCountFromCtx(coinBucket, ctx, 1)
	}
	report.CoinOperations = buildSourceCountItems(coinBucket)

	return report
}
