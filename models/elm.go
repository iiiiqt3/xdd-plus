package models

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	elmActivityEnvKey       = "elmck" // 饿了么(协议) 活动环境变量
	elmMiniAppID            = "wxece3a9a4c82f58c9"
	elmAppKey               = "12574478"
	elmHost                 = "https://waimai-guide.ele.me"
	elmOrigin               = "https://tb.ele.me"
	elmReferer              = "https://tb.ele.me/app/TBTakeout/engage-hub/home"
	elmVersion              = "1.2.30"
	elmHomepageAPI          = "mtop.alsc.interact.et.interact.center.homepage"
	elmTokenAPI             = "mtop.alsc.user.session.ele.check"
	elmExchangeAPI          = "mtop.alsc.interact.playapp.reward.right.exchange"
	elmExchangeAsac         = "alsc5KvbdX5mHl3sdv4guV"
	elmExchangeSource       = "INTERACT_CENTER_EXCHANGE_MALL"
	elmPrepareBeforeSec     = 30
	elmAttemptsPerAccount   = 15
	elmAttemptIntervalMs    = 300
	elmExchangeRounds       = 3
	elmTaskRetainDuration   = 3 * time.Hour
	elmDefaultLat           = "30.27415"
	elmDefaultLng           = "120.15507"
)

var (
	elmUserAgent = "Mozilla/5.0 (iPhone; CPU iPhone OS 18_5 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.5 Mobile/15E148 Safari/604.1"
	elmCPNCodes  = `["PLAY_NOTICE_CPN","INTERACT_RESOURCE_CPN","MORE_MENU_CPN","STAR_MSG_CONTENT_CPN","PLAY_RESOURCE_CPN","INTERACT_CENTER_SKIN_COMPONENT","INTERACT_CENTER_BUBBLE","INTERACT_CENTER_LOTTERY"]`
	elmRoundLagMs = []int{0, 80, 250}
)

var elmHTTPClient = &http.Client{
	Timeout: 25 * time.Second,
	Transport: &http.Transport{
		MaxIdleConns:        64,
		MaxIdleConnsPerHost: 32,
		IdleConnTimeout:     90 * time.Second,
		ForceAttemptHTTP2:   true,
	},
}

// ElmAccountInfo 饿了么抢兑账号（来自已上车项目）
type ElmAccountInfo struct {
	Ref      string `json:"ref"`
	Remark   string `json:"remark"`
	ProjectID int   `json:"projectId"`
}

// ElmProductBrief 目标商品摘要
type ElmProductBrief struct {
	Title  string `json:"title"`
	ID     string `json:"id"`
	Cost   int    `json:"cost"`
	Status string `json:"status"`
}

// ElmWindowInfo 当前抢兑窗口
type ElmWindowInfo struct {
	IsFriday    bool   `json:"isFriday"`
	InWindow    bool   `json:"inWindow"`
	CanStart    bool   `json:"canStart"`
	WindowLabel string `json:"windowLabel"`
	ExecuteAt   string `json:"executeAt"`
	PrepareAt   string `json:"prepareAt"`
	Message     string `json:"message"`
	Keyword     string `json:"keyword"`
}

// ElmTodaySlotProduct 今日某场次商品预览
type ElmTodaySlotProduct struct {
	TargetHour int              `json:"targetHour"`
	SlotLabel  string           `json:"slotLabel"`
	ExecuteAt  string           `json:"executeAt"`
	Product    *ElmProductBrief `json:"product,omitempty"`
}

// ElmTodayProductsInfo 今日抢兑商品预览
type ElmTodayProductsInfo struct {
	IsFriday      bool                   `json:"isFriday"`
	StarBalance   int                    `json:"starBalance"`
	AccountRemark string                 `json:"accountRemark"`
	Slots         []ElmTodaySlotProduct  `json:"slots"`
	Message       string                 `json:"message"`
}

// ElmTaskLog 抢兑日志
type ElmTaskLog struct {
	Time    string `json:"time"`
	Level   string `json:"level"`
	Message string `json:"message"`
}

// ElmExchangeResult 单账号结果
type ElmExchangeResult struct {
	Ref     string `json:"ref"`
	Remark  string `json:"remark"`
	Success bool   `json:"success"`
	Message string `json:"message"`
	Product string `json:"product,omitempty"`
}

// ElmScheduledTask 饿了么抢兑任务
type ElmScheduledTask struct {
	ID          string              `json:"id"`
	UserNumber  int                 `json:"userNumber"`
	Keyword     string              `json:"keyword"`
	TargetHour  int                 `json:"targetHour"`
	ExecuteAt   time.Time           `json:"executeAt"`
	PrepareAt   time.Time           `json:"prepareAt"`
	CreatedAt   time.Time           `json:"createdAt"`
	Status      string              `json:"status"`
	Product     *ElmProductBrief    `json:"product,omitempty"`
	Accounts    []ElmAccountInfo    `json:"accounts"`
	Results     []ElmExchangeResult `json:"results,omitempty"`
	Logs        []ElmTaskLog        `json:"logs,omitempty"`
	logMu       sync.Mutex          `json:"-"`
	ready       []*elmReadyAccount  `json:"-"`
	cancelled   atomic.Bool         `json:"-"`
}

func (t *ElmScheduledTask) isCancelled() bool {
	return t != nil && t.cancelled.Load()
}

type elmReadyAccount struct {
	info     ElmAccountInfo
	client   *elmMtopClient
	product  map[string]interface{}
	star     int
}

type elmMtopClient struct {
	cookie  string
	session *http.Client
}

type elmCacheItem struct {
	Cookie     string `json:"cookie"`
	Remark     string `json:"remark"`
	UpdateTime int64  `json:"updateTime"`
}

var elmScheduledTasks = struct {
	sync.RWMutex
	tasks map[string]*ElmScheduledTask
}{tasks: make(map[string]*ElmScheduledTask)}

func elmBJLocation() *time.Location {
	return kuwoBeijingLocation()
}

func elmLogDir() string {
	return filepath.Join(ExecPath, "logs", "elm")
}

func elmCacheFile(taskID string) string {
	return filepath.Join(elmLogDir(), "cache", taskID+".json")
}

func elmTaskLogFile(taskID string) string {
	return filepath.Join(elmLogDir(), taskID+".log")
}

func elmDailyLogFile() string {
	return filepath.Join(elmLogDir(), fmt.Sprintf("elm_%s.log", time.Now().In(elmBJLocation()).Format("2006-01-02")))
}

func elmWriteDailyLog(logTime, level, msg string) {
	_ = os.MkdirAll(elmLogDir(), 0755)
	line := fmt.Sprintf("[%s] [%s] %s\n", logTime, level, msg)
	f, err := os.OpenFile(elmDailyLogFile(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = f.WriteString(line)
}

// elmWriteAdminLog 写入管理员后台「饿了么」分类日志，并追加每日汇总文件 logs/elm/elm_YYYY-MM-DD.log
func elmWriteAdminLog(level, format string, args ...interface{}) {
	msg := format
	if len(args) > 0 {
		msg = fmt.Sprintf(format, args...)
	}
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "error":
		Elm().Errorf("%s", msg)
	case "warn", "warning":
		Elm().Warnf("%s", msg)
	default:
		Elm().Infof("%s", msg)
	}
	logTime := time.Now().In(elmBJLocation()).Format("15:04:05.000")
	go elmWriteDailyLog(logTime, level, msg)
}

func elmMd5Hex(text string) string {
	sum := md5.Sum([]byte(text))
	return hex.EncodeToString(sum[:])
}

func elmCookieParse(cookie string) map[string]string {
	jar := map[string]string{}
	for _, part := range strings.Split(cookie, ";") {
		part = strings.TrimSpace(part)
		if part == "" || !strings.Contains(part, "=") {
			continue
		}
		k, v, _ := strings.Cut(part, "=")
		jar[strings.TrimSpace(k)] = strings.TrimSpace(v)
	}
	return jar
}

func elmCookieDump(jar map[string]string) string {
	parts := make([]string, 0, len(jar))
	for k, v := range jar {
		parts = append(parts, k+"="+v)
	}
	sort.Strings(parts)
	return strings.Join(parts, "; ")
}

func elmFirstRet(data map[string]interface{}) string {
	if data == nil {
		return ""
	}
	if ret, ok := data["ret"].([]interface{}); ok && len(ret) > 0 {
		return fmt.Sprint(ret[0])
	}
	return ""
}

func elmIsSuccess(data map[string]interface{}) bool {
	return strings.HasPrefix(elmFirstRet(data), "SUCCESS")
}

func elmMaskRef(ref string) string {
	ref = strings.TrimSpace(ref)
	if len(ref) <= 10 {
		return ref
	}
	return ref[:6] + "…" + ref[len(ref)-4:]
}

func (c *elmMtopClient) token() string {
	v := elmCookieParse(c.cookie)["_m_h5_tk"]
	if v == "" {
		return ""
	}
	if i := strings.Index(v, "_"); i > 0 {
		return v[:i]
	}
	return v
}

func (c *elmMtopClient) mergeCookies(resp *http.Response) {
	jar := elmCookieParse(c.cookie)
	for _, ck := range resp.Cookies() {
		jar[ck.Name] = ck.Value
	}
	c.cookie = elmCookieDump(jar)
}

func (c *elmMtopClient) refreshToken() error {
	jar := elmCookieParse(c.cookie)
	delete(jar, "_m_h5_tk")
	delete(jar, "_m_h5_tk_enc")
	bare := elmCookieDump(jar)
	nowMs := fmt.Sprintf("%d", time.Now().UnixMilli())
	dataJSON := "{}"
	query := url.Values{
		"jsv":            {"2.7.5"},
		"appKey":         {elmAppKey},
		"t":              {nowMs},
		"sign":           {elmMd5Hex("&" + nowMs + "&" + elmAppKey + "&" + dataJSON)},
		"api":            {elmTokenAPI},
		"v":              {"1.0"},
		"type":           {"originaljson"},
		"dataType":       {"json"},
		"H5Request":      {"true"},
		"mainDomain":     {"ele.me"},
		"subDomain":      {"waimai-guide"},
		"pageDomain":     {"ele.me"},
		"syncCookieMode": {"true"},
	}
	req, err := http.NewRequest(http.MethodPost, elmHost+"/h5/"+elmTokenAPI+"/1.0/?"+query.Encode(), strings.NewReader("data="+url.QueryEscape(dataJSON)))
	if err != nil {
		return err
	}
	req.Header.Set("content-type", "application/x-www-form-urlencoded;charset=UTF-8")
	req.Header.Set("accept", "application/json")
	req.Header.Set("cookie", bare)
	req.Header.Set("origin", elmOrigin)
	req.Header.Set("referer", elmReferer)
	req.Header.Set("user-agent", elmUserAgent)
	resp, err := c.session.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	for _, ck := range resp.Cookies() {
		jar[ck.Name] = ck.Value
	}
	c.cookie = elmCookieDump(jar)
	if c.token() == "" {
		snip := strings.TrimSpace(string(body))
		if len(snip) > 200 {
			snip = snip[:200]
		}
		return fmt.Errorf("mtop token 续期失败: %s", snip)
	}
	return nil
}

func (c *elmMtopClient) request(api string, data map[string]interface{}, method string, extraHeaders map[string]string, retryToken bool) (map[string]interface{}, error) {
	if c.token() == "" {
		if err := c.refreshToken(); err != nil {
			return nil, err
		}
	}
	dataJSON, _ := json.Marshal(data)
	nowMs := fmt.Sprintf("%d", time.Now().UnixMilli())
	sign := elmMd5Hex(c.token() + "&" + nowMs + "&" + elmAppKey + "&" + string(dataJSON))
	query := url.Values{
		"jsv":            {"2.7.2"},
		"appKey":         {elmAppKey},
		"t":              {nowMs},
		"sign":           {sign},
		"api":            {api},
		"v":              {"1.0"},
		"dataType":       {"json"},
		"timeout":        {"20000"},
		"ttid":           {"H5@Web_android_" + elmVersion},
		"mainDomain":     {"ele.me"},
		"subDomain":      {"waimai-guide"},
		"pageDomain":     {"ele.me"},
		"H5Request":      {"true"},
		"type":           {"originaljson"},
		"SV":             {"5.0"},
		"syncCookieMode": {"true"},
	}
	headers := map[string]string{
		"accept":             "application/json",
		"content-type":       "application/x-www-form-urlencoded",
		"cookie":             c.cookie,
		"origin":             elmOrigin,
		"referer":            elmReferer,
		"user-agent":         elmUserAgent,
		"x-ele-check-client": "ele",
	}
	for k, v := range extraHeaders {
		headers[k] = v
	}
	var req *http.Request
	var err error
	if strings.EqualFold(method, "GET") {
		query.Set("data", string(dataJSON))
		req, err = http.NewRequest(http.MethodGet, elmHost+"/h5/"+api+"/1.0/5.0/?"+query.Encode(), nil)
	} else {
		req, err = http.NewRequest(http.MethodPost, elmHost+"/h5/"+api+"/1.0/5.0/?"+query.Encode(), strings.NewReader("data="+url.QueryEscape(string(dataJSON))))
	}
	if err != nil {
		return nil, err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := c.session.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	c.mergeCookies(resp)
	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		return map[string]interface{}{"ret": []interface{}{"CLIENT_BAD_JSON::" + fmt.Sprint(resp.StatusCode)}, "raw": string(body[:min(len(body), 300)])}, nil
	}
	ret := strings.ToUpper(elmFirstRet(payload))
	if retryToken && (strings.Contains(ret, "TOKEN_EXPIRED") || strings.Contains(ret, "TOKEN_EXOIRED") || strings.Contains(ret, "SESSION_EXPIRED")) {
		if err := c.refreshToken(); err != nil {
			return nil, err
		}
		return c.request(api, data, method, extraHeaders, false)
	}
	return payload, nil
}

func (c *elmMtopClient) homepage() (map[string]interface{}, error) {
	body := map[string]interface{}{
		"bizScene":  "interact_center",
		"version":   elmVersion,
		"cpnCodes":  elmCPNCodes,
		"latitude":  "0.0",
		"longitude": "0.0",
	}
	return c.request(elmHomepageAPI, body, "GET", nil, true)
}

func (c *elmMtopClient) exchange(product map[string]interface{}) (map[string]interface{}, error) {
	body := map[string]interface{}{
		"bizScene":             "interact_center",
		"version":              elmVersion,
		"exchangeId":           fmt.Sprint(product["exchangeId"]),
		"actId":                fmt.Sprint(product["exchangeActId"]),
		"exchangeCollectionId": fmt.Sprint(product["exchangeCollectionId"]),
		"source":               elmExchangeSource,
		"latitude":             elmDefaultLat,
		"longitude":            elmDefaultLng,
	}
	return c.request(elmExchangeAPI, body, "POST", map[string]string{"asac": elmExchangeAsac}, true)
}

func elmProducts(home map[string]interface{}) []map[string]interface{} {
	root, _ := home["data"].(map[string]interface{})
	if root == nil {
		return nil
	}
	inner, _ := root["data"].(map[string]interface{})
	if inner == nil {
		return nil
	}
	exchange, _ := inner["exchange"].(map[string]interface{})
	if exchange == nil {
		return nil
	}
	raw, _ := exchange["data"].([]interface{})
	out := make([]map[string]interface{}, 0, len(raw))
	for _, item := range raw {
		if m, ok := item.(map[string]interface{}); ok {
			out = append(out, m)
		}
	}
	return out
}

func elmStarBalance(home map[string]interface{}) int {
	root, _ := home["data"].(map[string]interface{})
	if root == nil {
		return -1
	}
	inner, _ := root["data"].(map[string]interface{})
	if inner == nil {
		return -1
	}
	prop, _ := inner["property"].(map[string]interface{})
	if prop == nil {
		return -1
	}
	data, _ := prop["data"].(map[string]interface{})
	if data == nil {
		return -1
	}
	star, _ := data["STAR"].(map[string]interface{})
	if star == nil {
		return -1
	}
	switch v := star["amount"].(type) {
	case float64:
		return int(v)
	case string:
		var f float64
		fmt.Sscan(v, &f)
		return int(f)
	default:
		return -1
	}
}

func elmProductBrief(item map[string]interface{}) ElmProductBrief {
	info, _ := item["exchangeInfo"].(map[string]interface{})
	material, _ := item["materialInfo"].(map[string]interface{})
	title := fmt.Sprint(material["title"])
	if title == "" {
		title = fmt.Sprint(item["rightName"])
	}
	cost := 0
	if info != nil {
		switch v := info["consumeAmount"].(type) {
		case float64:
			cost = int(v)
		case string:
			fmt.Sscan(v, &cost)
		}
	}
	status := "UNKNOWN"
	if info != nil {
		status = fmt.Sprint(info["exchangeStatus"])
	}
	return ElmProductBrief{
		Title:  title,
		ID:     fmt.Sprint(item["exchangeId"]),
		Cost:   cost,
		Status: status,
	}
}

func elmProductTitle(item map[string]interface{}) string {
	material, _ := item["materialInfo"].(map[string]interface{})
	title := strings.TrimSpace(fmt.Sprint(material["title"]))
	if title == "" {
		title = strings.TrimSpace(fmt.Sprint(item["rightName"]))
	}
	return title
}

func elmProductStatus(item map[string]interface{}) string {
	info, _ := item["exchangeInfo"].(map[string]interface{})
	if info == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(info["exchangeStatus"]))
}

func elmIsSeckillProduct(status string) bool {
	status = strings.ToUpper(strings.TrimSpace(status))
	if status == "" {
		return false
	}
	return status == "SECKILL_NOT_STARTED" || status == "AVAILABLE" || status == "CAN_EXCHANGE" || strings.Contains(status, "SECKILL")
}

func elmMatchSlotHour(title string, targetHour int) bool {
	title = strings.TrimSpace(title)
	if title == "" || targetHour <= 0 {
		return false
	}
	h := fmt.Sprintf("%d", targetHour)
	for _, p := range []string{h + "点", h + ":00", h + "时", fmt.Sprintf("%02d:00", targetHour)} {
		if strings.Contains(title, p) {
			return true
		}
	}
	return false
}

func elmSelectProductByKeyword(products []map[string]interface{}, keyword string) map[string]interface{} {
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return nil
	}
	words := []string{}
	for _, w := range regexp.MustCompile(`[|,，]`).Split(keyword, -1) {
		w = strings.TrimSpace(strings.ToLower(w))
		if w != "" {
			words = append(words, w)
		}
	}
	if len(words) == 0 {
		return nil
	}
	for _, item := range products {
		title := strings.ToLower(elmProductTitle(item))
		ok := true
		for _, w := range words {
			if !strings.Contains(title, w) {
				ok = false
				break
			}
		}
		if ok {
			return item
		}
	}
	return nil
}

// elmSelectProductForSlot 按当前场次自动识别商品：优先标题含 10点/15点，其次唯一秒杀商品；keyword 可选覆盖
func elmSelectProductForSlot(products []map[string]interface{}, targetHour int, keyword string) map[string]interface{} {
	if len(products) == 0 {
		return nil
	}
	if p := elmSelectProductByKeyword(products, keyword); p != nil {
		return p
	}
	if targetHour > 0 {
		var hourMatches []map[string]interface{}
		for _, item := range products {
			if elmMatchSlotHour(elmProductTitle(item), targetHour) {
				hourMatches = append(hourMatches, item)
			}
		}
		if len(hourMatches) == 1 {
			return hourMatches[0]
		}
		if len(hourMatches) > 1 {
			for _, item := range hourMatches {
				if elmIsSeckillProduct(elmProductStatus(item)) {
					return item
				}
			}
			return hourMatches[0]
		}
	}
	seckill := make([]map[string]interface{}, 0, len(products))
	for _, item := range products {
		if elmIsSeckillProduct(elmProductStatus(item)) {
			seckill = append(seckill, item)
		}
	}
	if len(seckill) == 1 {
		return seckill[0]
	}
	if len(seckill) > 1 && targetHour > 0 {
		for _, item := range seckill {
			if elmMatchSlotHour(elmProductTitle(item), targetHour) {
				return item
			}
		}
	}
	return nil
}

func elmNormalizeRef(ref string) string {
	ref = strings.TrimSpace(ref)
	if strings.HasPrefix(ref, "wx:") {
		return ref[3:]
	}
	return ref
}

func elmIsProtocolRef(ref string) bool {
	ref = strings.TrimSpace(ref)
	return strings.HasPrefix(ref, "wxid_") || strings.HasPrefix(ref, "yyb:") || strings.HasPrefix(ref, "owNAX") || (!strings.Contains(ref, "=") && !strings.Contains(ref, ";"))
}

func elmProtocolLogin(ref string) (string, error) {
	code, err := ProtocolGetWxAppCode(ref, elmMiniAppID)
	if err != nil {
		return "", err
	}
	authCode, _ := json.Marshal(map[string]string{"authorizationCode": code})
	form := url.Values{
		"type":                 {"weixin_mini_program"},
		"appId":                {elmMiniAppID},
		"appName":              {"eleme"},
		"appEntrance":          {"weixin"},
		"lang":                 {"zh_CN"},
		"isMobile":             {"true"},
		"returnUrl":            {""},
		"needPassWebViewCookie": {"false"},
		"authorizationCode":  {string(authCode)},
	}
	var lastErr string
	for attempt := 0; attempt < 2; attempt++ {
		req, err := http.NewRequest(http.MethodPost, "https://ipassport.ele.me/mini_program/login.do?_bx-m=0.0.15", strings.NewReader(form.Encode()))
		if err != nil {
			return "", err
		}
		req.Header.Set("content-type", "application/x-www-form-urlencoded;charset=UTF-8")
		req.Header.Set("user-agent", elmUserAgent)
		req.Header.Set("referer", "https://servicewechat.com/"+elmMiniAppID+"/")
		resp, err := elmHTTPClient.Do(req)
		if err != nil {
			return "", err
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		jar := map[string]string{}
		for _, ck := range resp.Cookies() {
			jar[ck.Name] = ck.Value
		}
		cookie := elmCookieDump(jar)
		if regexp.MustCompile(`(?i)(?:^|;\s*)(?:USERID|SID)=`).MatchString(cookie) {
			return cookie, nil
		}
		lastErr = string(body[:min(len(body), 300)])
		if attempt == 0 && regexp.MustCompile(`RGV587_ERROR::SM|被挤爆|哎哟喂`).MatchString(lastErr) {
			time.Sleep(10 * time.Second)
			code, err = ProtocolGetWxAppCode(ref, elmMiniAppID)
			if err != nil {
				return "", err
			}
			authCode, _ = json.Marshal(map[string]string{"authorizationCode": code})
			form.Set("authorizationCode", string(authCode))
			continue
		}
		break
	}
	return "", fmt.Errorf("饿了么登录失败: %s", lastErr)
}

func elmLoadCacheMap(taskID string) map[string]elmCacheItem {
	path := elmCacheFile(taskID)
	raw, err := os.ReadFile(path)
	if err != nil {
		return map[string]elmCacheItem{}
	}
	out := map[string]elmCacheItem{}
	_ = json.Unmarshal(raw, &out)
	return out
}

func elmSaveCacheItem(taskID, key string, item elmCacheItem) {
	_ = os.MkdirAll(filepath.Dir(elmCacheFile(taskID)), 0755)
	cache := elmLoadCacheMap(taskID)
	item.UpdateTime = time.Now().UnixMilli()
	cache[key] = item
	raw, _ := json.MarshalIndent(cache, "", "  ")
	_ = os.WriteFile(elmCacheFile(taskID), raw, 0644)
}

func elmDeleteCacheItem(taskID, key string) {
	if strings.TrimSpace(taskID) == "" || strings.TrimSpace(key) == "" {
		return
	}
	cache := elmLoadCacheMap(taskID)
	if _, ok := cache[key]; !ok {
		return
	}
	delete(cache, key)
	raw, _ := json.MarshalIndent(cache, "", "  ")
	_ = os.WriteFile(elmCacheFile(taskID), raw, 0644)
}

func elmPrepareAccount(taskID string, acc ElmAccountInfo) (*elmReadyAccount, error) {
	ref := strings.TrimSpace(acc.Ref)
	if ref == "" {
		return nil, fmt.Errorf("账号为空")
	}
	cacheKey := elmNormalizeRef(ref)
	cache := elmLoadCacheMap(taskID)
	cookie := ""
	if elmIsProtocolRef(ref) {
		if item, ok := cache[cacheKey]; ok {
			cookie = item.Cookie
		}
	} else {
		cookie = ref
	}
	client := &elmMtopClient{cookie: cookie, session: elmHTTPClient}
	if cookie != "" {
		if err := client.refreshToken(); err == nil {
			home, err := client.homepage()
			if err == nil && elmIsSuccess(home) {
				elmSaveCacheItem(taskID, cacheKey, elmCacheItem{Cookie: client.cookie, Remark: acc.Remark})
				elmWriteAdminLog("info", "[CK] [%s] 使用缓存成功 ref=%s", acc.Remark, elmMaskRef(ref))
				return &elmReadyAccount{info: acc, client: client}, nil
			}
		}
		if elmIsProtocolRef(ref) {
			elmWriteAdminLog("warn", "[CK] [%s] 缓存 CK 失效，将重新协议登录 ref=%s", acc.Remark, elmMaskRef(ref))
			elmDeleteCacheItem(taskID, cacheKey)
		}
	}
	if !elmIsProtocolRef(ref) {
		elmWriteAdminLog("error", "[CK] [%s] 纯 CK 已失效 ref=%s", acc.Remark, elmMaskRef(ref))
		return nil, fmt.Errorf("纯 CK 已失效")
	}
	elmWriteAdminLog("info", "[CK] [%s] 缓存失效，走协议登录 ref=%s", acc.Remark, elmMaskRef(ref))
	cookie, err := elmProtocolLogin(ref)
	if err != nil {
		elmWriteAdminLog("error", "[CK] [%s] 协议登录失败: %v", acc.Remark, err)
		return nil, err
	}
	client.cookie = cookie
	if err := client.refreshToken(); err != nil {
		elmWriteAdminLog("error", "[CK] [%s] token 刷新失败: %v", acc.Remark, err)
		return nil, err
	}
	home, err := client.homepage()
	if err != nil || !elmIsSuccess(home) {
		elmWriteAdminLog("error", "[CK] [%s] 登录后商城查询失败: %s", acc.Remark, elmFirstRet(home))
		return nil, fmt.Errorf("登录后商城查询失败: %s", elmFirstRet(home))
	}
	elmSaveCacheItem(taskID, cacheKey, elmCacheItem{Cookie: client.cookie, Remark: acc.Remark})
	elmWriteAdminLog("info", "[CK] [%s] 协议登录成功并已缓存", acc.Remark)
	return &elmReadyAccount{info: acc, client: client}, nil
}

func elmRetryableExchangeError(code string) bool {
	code = strings.ToUpper(strings.TrimSpace(code))
	for _, key := range []string{"STOCK_NOT_ENOUGH", "INVENTORY_NOT_ENOUGH", "CAMP_CONSULT_RULE_NOT_PASS", "SECKILL_NOT_STARTED", "SYSTEM_ERROR", "BUSY"} {
		if strings.Contains(code, key) {
			return true
		}
	}
	return false
}

func (t *ElmScheduledTask) AddLog(level, format string, args ...interface{}) {
	if t == nil {
		return
	}
	msg := format
	if len(args) > 0 {
		msg = fmt.Sprintf(format, args...)
	}
	logTime := time.Now().In(elmBJLocation()).Format("15:04:05.000")
	t.logMu.Lock()
	t.Logs = append(t.Logs, ElmTaskLog{Time: logTime, Level: level, Message: msg})
	t.logMu.Unlock()
	adminMsg := fmt.Sprintf("[task=%s user=%d] %s", t.ID, t.UserNumber, msg)
	go func() {
		elmWriteAdminLog(level, "%s", adminMsg)
		_ = os.MkdirAll(elmLogDir(), 0755)
		line := fmt.Sprintf("[%s] [%s] %s\n", logTime, level, msg)
		f, err := os.OpenFile(elmTaskLogFile(t.ID), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return
		}
		defer f.Close()
		_, _ = f.WriteString(line)
	}()
}

func (t *ElmScheduledTask) LogsSnapshot() []ElmTaskLog {
	if t == nil {
		return nil
	}
	t.logMu.Lock()
	defer t.logMu.Unlock()
	out := make([]ElmTaskLog, len(t.Logs))
	copy(out, t.Logs)
	return out
}

func elmCurrentWindow(now time.Time) (inWindow bool, targetHour int, executeAt, prepareAt time.Time, label string) {
	bj := now.In(elmBJLocation())
	if bj.Weekday() != time.Friday {
		return false, 0, time.Time{}, time.Time{}, ""
	}
	minutes := bj.Hour()*60 + bj.Minute()
	if minutes >= 9*60+30 && minutes < 9*60+56 {
		executeAt = time.Date(bj.Year(), bj.Month(), bj.Day(), 10, 0, 0, 0, elmBJLocation())
		prepareAt = executeAt.Add(-elmPrepareBeforeSec * time.Second)
		return true, 10, executeAt, prepareAt, "周五 10:00 场"
	}
	if minutes >= 14*60+30 && minutes < 14*60+56 {
		executeAt = time.Date(bj.Year(), bj.Month(), bj.Day(), 15, 0, 0, 0, elmBJLocation())
		prepareAt = executeAt.Add(-elmPrepareBeforeSec * time.Second)
		return true, 15, executeAt, prepareAt, "周五 15:00 场"
	}
	return false, 0, time.Time{}, time.Time{}, ""
}

// ElmGetWindowInfo 返回当前抢兑窗口信息
func ElmGetWindowInfo() ElmWindowInfo {
	now := time.Now().In(elmBJLocation())
	info := ElmWindowInfo{
		IsFriday: now.Weekday() == time.Friday,
		Keyword:  "自动识别当前场次商品",
	}
	inWindow, _, executeAt, prepareAt, label := elmCurrentWindow(now)
	info.InWindow = inWindow
	info.WindowLabel = label
	if inWindow {
		info.CanStart = true
		info.ExecuteAt = executeAt.Format("2006-01-02 15:04:05")
		info.PrepareAt = prepareAt.Format("2006-01-02 15:04:05")
		info.Message = "当前可参与抢兑，系统将倒计时到 " + executeAt.Format("15:04:05") + " 执行"
	} else if !info.IsFriday {
		info.Message = "仅每周五开放抢兑（09:30-09:55 / 14:30-14:55 可报名）"
	} else {
		info.Message = "当前不在抢兑报名时段（09:30-09:55 或 14:30-14:55）"
	}
	return info
}

// GetAllElmAccounts 读取用户已上车的饿了么(elmck)账号
func GetAllElmAccounts(userNumber int) ([]ElmAccountInfo, error) {
	projects, err := GetActivityProjectsByUserAndEnv(userNumber, elmActivityEnvKey, elmActivityEnvKey)
	if err != nil {
		return nil, err
	}
	out := make([]ElmAccountInfo, 0, len(projects))
	for _, p := range projects {
		if strings.TrimSpace(p.EnvValue) == "" {
			continue
		}
		ref := strings.TrimSpace(p.EnvValue)
		remark := projectRemarkAlias(p)
		if i := strings.Index(ref, "#"); i >= 0 {
			if remark == "" {
				remark = strings.TrimSpace(ref[i+1:])
			}
			ref = strings.TrimSpace(ref[:i])
		}
		out = append(out, ElmAccountInfo{Ref: ref, Remark: remark, ProjectID: p.ID})
	}
	return out, nil
}

// CheckElmAuth 检查用户是否有饿了么(elmck)活动授权
func CheckElmAuth(userNumber int) (bool, string) {
	projects, err := GetActivityProjectsByUserAndEnv(userNumber, elmActivityEnvKey, elmActivityEnvKey)
	if err != nil || len(projects) == 0 {
		return false, "您还没有上车饿了么(协议)活动，请先前往「项目中心」上车"
	}
	now := time.Now()
	for _, p := range projects {
		if p.ExpireDate == "" {
			continue
		}
		expireTime, err := time.Parse("2006-01-02", p.ExpireDate)
		if err == nil {
			expireThreshold := time.Date(expireTime.Year(), expireTime.Month(), expireTime.Day(), 0, 0, 0, 0, time.Local).AddDate(0, 0, 1)
			if now.After(expireThreshold) {
				return false, "您的饿了么活动授权已到期，请前往「项目中心」续费"
			}
		}
	}
	return true, ""
}

func elmFindActiveTaskByUser(userNumber int) *ElmScheduledTask {
	elmScheduledTasks.RLock()
	defer elmScheduledTasks.RUnlock()
	for _, t := range elmScheduledTasks.tasks {
		if t != nil && t.UserNumber == userNumber && (t.Status == "pending" || t.Status == "running") {
			return t
		}
	}
	return nil
}

func elmPruneOldTasks() {
	cutoff := time.Now().Add(-elmTaskRetainDuration)
	elmScheduledTasks.Lock()
	defer elmScheduledTasks.Unlock()
	for id, t := range elmScheduledTasks.tasks {
		if t == nil {
			delete(elmScheduledTasks.tasks, id)
			continue
		}
		if (t.Status == "completed" || t.Status == "failed") && t.CreatedAt.Before(cutoff) {
			delete(elmScheduledTasks.tasks, id)
		}
	}
}

func elmTaskJitter(userNumber int) time.Duration {
	if userNumber <= 0 {
		return 0
	}
	return time.Duration(userNumber%120) * time.Millisecond
}

func elmSleepUntil(target time.Time) {
	elmSleepUntilWithCancel(target, nil)
}

func elmSleepUntilWithCancel(target time.Time, task *ElmScheduledTask) bool {
	for {
		if task != nil && task.isCancelled() {
			return false
		}
		wait := time.Until(target)
		if wait <= 0 {
			return true
		}
		step := 500 * time.Millisecond
		if wait < 2*time.Second {
			step = 10 * time.Millisecond
		}
		if wait < step {
			time.Sleep(wait)
			return task == nil || !task.isCancelled()
		}
		time.Sleep(step)
	}
}

func elmRefreshTarget(acc *elmReadyAccount, targetHour int, keyword string) (*elmReadyAccount, error) {
	home, err := acc.client.homepage()
	if err != nil || !elmIsSuccess(home) {
		prepared, err2 := elmPrepareAccount("", acc.info)
		if err2 != nil {
			return nil, err2
		}
		acc = prepared
		home, err = acc.client.homepage()
	}
	if err != nil || !elmIsSuccess(home) {
		return nil, fmt.Errorf("商城查询失败: %s", elmFirstRet(home))
	}
	products := elmProducts(home)
	product := elmSelectProductForSlot(products, targetHour, keyword)
	if product == nil {
		var names []string
		for _, item := range products {
			b := elmProductBrief(item)
			names = append(names, fmt.Sprintf("[%s] %s", b.Status, b.Title))
		}
		if len(names) == 0 {
			return nil, fmt.Errorf("未找到目标商品（商城暂无兑换商品）")
		}
		return nil, fmt.Errorf("未找到 %d 点场目标商品，当前: %s", targetHour, strings.Join(names, " | "))
	}
	star := elmStarBalance(home)
	brief := elmProductBrief(product)
	if star >= 0 && star < brief.Cost {
		return nil, fmt.Errorf("幸运星不足 %d/%d", star, brief.Cost)
	}
	acc.product = product
	acc.star = star
	return acc, nil
}

func elmRunExchange(acc *elmReadyAccount, task *ElmScheduledTask) ElmExchangeResult {
	res := ElmExchangeResult{Ref: acc.info.Ref, Remark: acc.info.Remark}
	product := acc.product
	if product == nil {
		refreshed, err := elmRefreshTarget(acc, task.TargetHour, task.Keyword)
		if err != nil || refreshed == nil || refreshed.product == nil {
			res.Message = "未找到目标商品"
			if err != nil {
				res.Message = err.Error()
			}
			return res
		}
		acc = refreshed
		product = acc.product
	}
	brief := elmProductBrief(product)
	for attempt := 1; attempt <= elmAttemptsPerAccount; attempt++ {
		resp, err := acc.client.exchange(product)
		if err != nil {
			res.Message = err.Error()
			task.AddLog("warn", "[%s] 第%d次请求失败: %v", acc.info.Remark, attempt, err)
			time.Sleep(time.Duration(elmAttemptIntervalMs) * time.Millisecond)
			continue
		}
		data, _ := resp["data"].(map[string]interface{})
		errorCode := fmt.Sprint(data["errorCode"])
		errorMsg := fmt.Sprint(data["errorMsg"])
		if errorMsg == "" {
			errorMsg = elmFirstRet(resp)
		}
		if elmIsSuccess(resp) && errorCode == "" {
			res.Success = true
			res.Message = "兑换成功"
			res.Product = brief.Title
			task.AddLog("success", "🎉 [%s] 兑换成功: %s", acc.info.Remark, brief.Title)
			return res
		}
		if errorCode == "UPP_SEND_PRIZE_SENDING" {
			res.Success = true
			res.Message = "兑换已提交，奖励发放中"
			res.Product = brief.Title
			task.AddLog("success", "🎉 [%s] 兑换已提交: %s", acc.info.Remark, brief.Title)
			return res
		}
		res.Message = strings.TrimSpace(errorCode + " " + errorMsg)
		task.AddLog("warn", "[%s] 第%d/%d次失败: %s", acc.info.Remark, attempt, elmAttemptsPerAccount, res.Message)
		if !elmRetryableExchangeError(errorCode) {
			return res
		}
		time.Sleep(time.Duration(elmAttemptIntervalMs) * time.Millisecond)
	}
	return res
}

func elmRunScheduledTask(taskID string) {
	elmScheduledTasks.RLock()
	task, ok := elmScheduledTasks.tasks[taskID]
	elmScheduledTasks.RUnlock()
	if !ok || task == nil {
		return
	}

	jitter := elmTaskJitter(task.UserNumber)
	firstFire := task.ExecuteAt.Add(time.Duration(elmRoundLagMs[0]) * time.Millisecond).Add(jitter)
	task.AddLog("info", "后台调度已启动，目标 %s，用户错峰 %s", task.ExecuteAt.Format("15:04:05"), jitter.Round(time.Millisecond))

	if !elmSleepUntilWithCancel(task.PrepareAt, task) {
		task.Status = "cancelled"
		task.AddLog("warn", "任务已停止")
		return
	}
	if task.isCancelled() {
		task.Status = "cancelled"
		task.AddLog("warn", "任务已停止")
		return
	}
	task.AddLog("info", "开始预检：刷新 CK / 余额 / 商品 ID")
	ready := make([]*elmReadyAccount, 0, len(task.Accounts))
	var mu sync.Mutex
	var wg sync.WaitGroup
	for _, acc := range task.Accounts {
		wg.Add(1)
		go func(a ElmAccountInfo) {
			defer wg.Done()
			prepared, err := elmPrepareAccount(task.ID, a)
			if err != nil {
				task.AddLog("error", "[%s] CK 预热失败: %v", a.Remark, err)
				return
			}
			refreshed, err := elmRefreshTarget(prepared, task.TargetHour, task.Keyword)
			if err != nil {
				task.AddLog("error", "[%s] 预检失败: %v", a.Remark, err)
				return
			}
			brief := elmProductBrief(refreshed.product)
			task.AddLog("success", "✅ [%s] 预检完成: %s | %d星 | 状态=%s", a.Remark, brief.Title, brief.Cost, brief.Status)
			mu.Lock()
			ready = append(ready, refreshed)
			mu.Unlock()
		}(acc)
	}
	wg.Wait()
	if task.isCancelled() {
		task.Status = "cancelled"
		task.AddLog("warn", "任务已停止")
		return
	}
	task.ready = ready
	if len(ready) == 0 {
		task.Status = "failed"
		task.AddLog("error", "没有通过预检的账号，任务结束")
		return
	}
	if task.Product == nil && ready[0].product != nil {
		b := elmProductBrief(ready[0].product)
		task.Product = &b
	}

	if !elmSleepUntilWithCancel(firstFire, task) || task.isCancelled() {
		task.Status = "cancelled"
		task.AddLog("warn", "任务已停止")
		return
	}
	task.Status = "running"

	successRefs := map[string]bool{}
	for roundIdx, lag := range elmRoundLagMs {
		if task.isCancelled() {
			break
		}
		fireAt := task.ExecuteAt.Add(time.Duration(lag)*time.Millisecond).Add(jitter)
		if time.Now().Before(fireAt) {
			if !elmSleepUntilWithCancel(fireAt, task) {
				break
			}
		}
		if task.isCancelled() {
			break
		}
		task.AddLog("info", "第 %d/%d 轮抢兑开始（+%dms）", roundIdx+1, len(elmRoundLagMs), lag)
		var roundWg sync.WaitGroup
		resultsMu := sync.Mutex{}
		for _, acc := range ready {
			key := elmNormalizeRef(acc.info.Ref)
			if successRefs[key] {
				continue
			}
			roundWg.Add(1)
			go func(a *elmReadyAccount) {
				defer roundWg.Done()
				result := elmRunExchange(a, task)
				resultsMu.Lock()
				defer resultsMu.Unlock()
				if result.Success {
					successRefs[key] = true
				}
				replaced := false
				for i := range task.Results {
					if elmNormalizeRef(task.Results[i].Ref) == key {
						task.Results[i] = result
						replaced = true
						break
					}
				}
				if !replaced {
					task.Results = append(task.Results, result)
				}
			}(acc)
		}
		roundWg.Wait()
	}

	if task.isCancelled() {
		task.Status = "cancelled"
		task.AddLog("warn", "任务已停止")
		return
	}
	task.Status = "completed"
	successN := 0
	for _, r := range task.Results {
		if r.Success {
			successN++
		}
	}
	task.AddLog("info", "任务结束：成功 %d / 账号 %d", successN, len(task.Accounts))
}

// ElmScheduleExchange 创建饿了么抢兑任务
func ElmScheduleExchange(userNumber int, refs []string, keyword string) (*ElmScheduledTask, bool, error) {
	elmPruneOldTasks()
	if ok, msg := CheckElmAuth(userNumber); !ok {
		return nil, false, fmt.Errorf("%s", msg)
	}
	now := time.Now().In(elmBJLocation())
	inWindow, targetHour, executeAt, prepareAt, label := elmCurrentWindow(now)
	if !inWindow {
		return nil, false, fmt.Errorf("当前不在抢兑报名时段（周五 09:30-09:55 或 14:30-14:55）")
	}
	if existing := elmFindActiveTaskByUser(userNumber); existing != nil {
		elmWriteAdminLog("info", "[task=%s user=%d] 复用进行中的抢兑任务", existing.ID, userNumber)
		return existing, true, nil
	}
	all, err := GetAllElmAccounts(userNumber)
	if err != nil {
		return nil, false, err
	}
	selected := map[string]bool{}
	for _, r := range refs {
		selected[strings.TrimSpace(r)] = true
	}
	accounts := make([]ElmAccountInfo, 0)
	for _, a := range all {
		if len(selected) == 0 || selected[a.Ref] {
			accounts = append(accounts, a)
		}
	}
	if len(accounts) == 0 {
		return nil, false, fmt.Errorf("请选择至少一个已上车的账号")
	}

	taskID := fmt.Sprintf("elm_%d_%d", userNumber, time.Now().UnixMilli())
	task := &ElmScheduledTask{
		ID:         taskID,
		UserNumber: userNumber,
		Keyword:    keyword,
		TargetHour: targetHour,
		ExecuteAt:  executeAt,
		PrepareAt:  prepareAt,
		CreatedAt:  time.Now(),
		Status:     "pending",
		Accounts:   accounts,
	}
	task.AddLog("info", "抢兑任务已创建：%s", label)
	if strings.TrimSpace(keyword) != "" {
		task.AddLog("info", "商品匹配：关键词 %s", strings.TrimSpace(keyword))
	} else {
		task.AddLog("info", "商品匹配：自动识别 %d 点场", targetHour)
	}
	task.AddLog("info", "执行时间：%s | 预检时间：%s", executeAt.Format("15:04:05"), prepareAt.Format("15:04:05"))
	task.AddLog("info", "账号数：%d", len(accounts))

	// 点击抢兑时先缓存 CK，并拉取商品信息展示
	var previewWg sync.WaitGroup
	previewCh := make(chan *elmReadyAccount, len(accounts))
	for _, acc := range accounts {
		previewWg.Add(1)
		go func(a ElmAccountInfo) {
			defer previewWg.Done()
			prepared, err := elmPrepareAccount(taskID, a)
			if err != nil {
				task.AddLog("warn", "[%s] CK 缓存失败: %v", a.Remark, err)
				return
			}
			refreshed, err := elmRefreshTarget(prepared, targetHour, keyword)
			if err != nil {
				task.AddLog("warn", "[%s] 商品预览失败: %v", a.Remark, err)
				return
			}
			previewCh <- refreshed
		}(acc)
	}
	go func() {
		previewWg.Wait()
		close(previewCh)
	}()
	for item := range previewCh {
		if task.Product == nil && item.product != nil {
			b := elmProductBrief(item.product)
			task.Product = &b
			task.AddLog("info", "目标商品：%s | 需要 %d 幸运星 | 状态 %s", b.Title, b.Cost, b.Status)
			break
		}
	}
	if task.Product == nil {
		elmFillTaskProductPreview(task, userNumber, targetHour)
	}

	elmScheduledTasks.Lock()
	elmScheduledTasks.tasks[taskID] = task
	elmScheduledTasks.Unlock()

	go elmRunScheduledTask(taskID)
	return task, false, nil
}

// ElmGetScheduledTask 查询任务
func ElmGetScheduledTask(taskID string) *ElmScheduledTask {
	elmScheduledTasks.RLock()
	defer elmScheduledTasks.RUnlock()
	return elmScheduledTasks.tasks[taskID]
}

// ElmGetActiveTaskByUser 用户进行中的任务
func ElmGetActiveTaskByUser(userNumber int) *ElmScheduledTask {
	return elmFindActiveTaskByUser(userNumber)
}

// ElmCancelExchange 停止进行中的抢兑任务
func ElmCancelExchange(userNumber int, taskID string) (*ElmScheduledTask, error) {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		if t := elmFindActiveTaskByUser(userNumber); t != nil {
			taskID = t.ID
		}
	}
	if taskID == "" {
		return nil, fmt.Errorf("没有进行中的抢兑任务")
	}
	elmScheduledTasks.RLock()
	task := elmScheduledTasks.tasks[taskID]
	elmScheduledTasks.RUnlock()
	if task == nil || task.UserNumber != userNumber {
		return nil, fmt.Errorf("任务不存在")
	}
	switch task.Status {
	case "completed", "failed", "cancelled":
		return task, fmt.Errorf("任务已结束")
	}
	task.cancelled.Store(true)
	task.Status = "cancelled"
	task.AddLog("warn", "用户手动停止任务")
	elmWriteAdminLog("warn", "[task=%s user=%d] 用户手动停止任务", task.ID, userNumber)
	return task, nil
}

func elmFillTaskProductPreview(task *ElmScheduledTask, userNumber, targetHour int) {
	if task == nil || task.Product != nil {
		return
	}
	info, err := ElmGetTodayProducts(userNumber)
	if err != nil || info == nil {
		return
	}
	for _, slot := range info.Slots {
		if slot.TargetHour == targetHour && slot.Product != nil {
			b := *slot.Product
			task.Product = &b
			task.AddLog("info", "目标商品(商城)：%s | 需要 %d 幸运星 | 状态 %s", b.Title, b.Cost, b.Status)
			return
		}
	}
}

func init() {
	_ = os.MkdirAll(filepath.Join(elmLogDir(), "cache"), 0755)
	elmWriteAdminLog("info", "饿了么抢兑模块已加载 slotAutoMatch=true rounds=%v", elmRoundLagMs)
}

// ElmGetTodayProducts 查询今日两场抢兑商品（进入页面时展示）
func ElmGetTodayProducts(userNumber int) (*ElmTodayProductsInfo, error) {
	bj := time.Now().In(elmBJLocation())
	info := &ElmTodayProductsInfo{
		IsFriday: bj.Weekday() == time.Friday,
		Slots: []ElmTodaySlotProduct{
			{TargetHour: 10, SlotLabel: "周五 10:00 场"},
			{TargetHour: 15, SlotLabel: "周五 15:00 场"},
		},
	}
	if info.IsFriday {
		info.Slots[0].ExecuteAt = time.Date(bj.Year(), bj.Month(), bj.Day(), 10, 0, 0, 0, elmBJLocation()).Format("2006-01-02 15:04:05")
		info.Slots[1].ExecuteAt = time.Date(bj.Year(), bj.Month(), bj.Day(), 15, 0, 0, 0, elmBJLocation()).Format("2006-01-02 15:04:05")
	} else {
		info.Slots[0].ExecuteAt = "每周五 10:00"
		info.Slots[1].ExecuteAt = "每周五 15:00"
	}
	if ok, msg := CheckElmAuth(userNumber); !ok {
		info.Message = msg
		return info, nil
	}
	accounts, err := GetAllElmAccounts(userNumber)
	if err != nil {
		return nil, err
	}
	if len(accounts) == 0 {
		info.Message = "暂无已上车账号"
		return info, nil
	}
	var lastErr error
	for _, acc := range accounts {
		prepared, err := elmPrepareAccount("today_preview_"+fmt.Sprint(userNumber), acc)
		if err != nil {
			lastErr = err
			continue
		}
		home, err := prepared.client.homepage()
		if err != nil || !elmIsSuccess(home) {
			if err != nil {
				lastErr = err
			} else {
				lastErr = fmt.Errorf("商城查询失败: %s", elmFirstRet(home))
			}
			continue
		}
		products := elmProducts(home)
		info.StarBalance = elmStarBalance(home)
		info.AccountRemark = acc.Remark
		for i := range info.Slots {
			if p := elmSelectProductForSlot(products, info.Slots[i].TargetHour, ""); p != nil {
				b := elmProductBrief(p)
				info.Slots[i].Product = &b
			}
		}
		found := 0
		for _, s := range info.Slots {
			if s.Product != nil {
				found++
			}
		}
		if found > 0 {
			info.Message = "已加载今日商城商品"
			var titles []string
			for _, s := range info.Slots {
				if s.Product != nil {
					titles = append(titles, fmt.Sprintf("%d点:%s", s.TargetHour, s.Product.Title))
				}
			}
			elmWriteAdminLog("info", "[preview user=%d] 今日商品 account=%s star=%d %s",
				userNumber, acc.Remark, info.StarBalance, strings.Join(titles, " | "))
			return info, nil
		}
		lastErr = fmt.Errorf("商城暂无匹配场次商品")
	}
	if lastErr != nil {
		info.Message = lastErr.Error()
		elmWriteAdminLog("warn", "[preview user=%d] 商品加载失败: %v", userNumber, lastErr)
	} else {
		info.Message = "未能加载商品信息"
		elmWriteAdminLog("warn", "[preview user=%d] 未能加载商品信息", userNumber)
	}
	return info, nil
}

// ElmPreviewProduct 预览当前商品（供前端展示）
func ElmPreviewProduct(userNumber int, refs []string, keyword string) (*ElmProductBrief, int, error) {
	if ok, msg := CheckElmAuth(userNumber); !ok {
		return nil, 0, fmt.Errorf("%s", msg)
	}
	all, err := GetAllElmAccounts(userNumber)
	if err != nil {
		return nil, 0, err
	}
	selected := map[string]bool{}
	for _, r := range refs {
		selected[strings.TrimSpace(r)] = true
	}
	for _, acc := range all {
		if len(selected) > 0 && !selected[acc.Ref] {
			continue
		}
		prepared, err := elmPrepareAccount("preview_"+fmt.Sprint(userNumber), acc)
		if err != nil {
			continue
		}
		home, err := prepared.client.homepage()
		if err != nil || !elmIsSuccess(home) {
			continue
		}
		product := elmSelectProductForSlot(elmProducts(home), 0, keyword)
		if product == nil {
			continue
		}
		brief := elmProductBrief(product)
		return &brief, elmStarBalance(home), nil
	}
	return nil, 0, fmt.Errorf("未能预览到目标商品，请确认账号 CK 有效")
}

// ElmTaskStatusPayload 供 API 返回
func ElmTaskStatusPayload(task *ElmScheduledTask) map[string]interface{} {
	if task == nil {
		return nil
	}
	product := task.Product
	return map[string]interface{}{
		"id":         task.ID,
		"status":     task.Status,
		"targetHour": task.TargetHour,
		"executeAt":  task.ExecuteAt.Format("2006-01-02 15:04:05"),
		"prepareAt":  task.PrepareAt.Format("2006-01-02 15:04:05"),
		"keyword":    task.Keyword,
		"product":    product,
		"results":    task.Results,
		"logs":       task.LogsSnapshot(),
	}
}