package models

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync/atomic"
	"sync"
	"time"

	"github.com/beego/beego/v2/core/logs"
)

var ipPortPattern = regexp.MustCompile(`(\d{1,3}(?:\.\d{1,3}){3}:\d+)`)

var (
	jdProxyCache     *url.URL
	jdProxyCacheTime time.Time
	jdProxyCacheMu   sync.Mutex
	// 动态 IP 有效期约 30 秒，缓存略短以免用到过期 IP
	jdProxyCacheTTL = 20 * time.Second
)

// IsJdTaskProxyEnabled 京东动态代理是否启用（查询 Go + 任务脚本共用，需开关打开且 API 非空）
func IsJdTaskProxyEnabled() bool {
	if strings.TrimSpace(sysConfig.JdTaskProxyUrl) == "" {
		return false
	}
	return sysConfig.JdTaskProxyEnabled
}

// ApplyJdTaskProxyEnvs 向 Node 脚本环境变量注入 DY_PROXY
func ApplyJdTaskProxyEnvs(envs map[string]string) {
	if envs == nil || !IsJdTaskProxyEnabled() {
		return
	}
	envs["DY_PROXY"] = strings.TrimSpace(sysConfig.JdTaskProxyUrl)
	renum := strings.TrimSpace(sysConfig.JdTaskProxyRenum)
	if renum == "" {
		renum = "10"
	}
	redelay := strings.TrimSpace(sysConfig.JdTaskProxyRedelay)
	if redelay == "" {
		redelay = "2"
	}
	envs["DY_PROXY_RENUM"] = renum
	envs["DY_PROXY_REDELAY"] = redelay
}

// ApplyJdProTaskProxyEnvs 部分脚本使用 PRO_API_PROXY_URL，与 DY_PROXY 共用同一 API 地址
func ApplyJdProTaskProxyEnvs(envs map[string]string) {
	ApplyJdTaskProxyEnvs(envs)
	if envs == nil || !IsJdTaskProxyEnabled() {
		return
	}
	envs["PRO_API_PROXY_URL"] = strings.TrimSpace(sysConfig.JdTaskProxyUrl)
	envs["PRO_PROXY_WHITELIST"] = "jd"
}

// JDProxyFunc 返回 httplib / http 可用的代理函数；未启用代理时返回 nil
func JDProxyFunc() func(*http.Request) (*url.URL, error) {
	if !IsJdTaskProxyEnabled() {
		return nil
	}
	if u := GetJDProxyURL(); u != nil {
		logs.Info("[京东代理] asset 请求走代理: %s", u.Host)
		return func(*http.Request) (*url.URL, error) {
			return u, nil
		}
	}
	logs.Warn("[京东代理] asset 请求未获取到代理，将直连")
	return nil
}

// GetJDProxyURL 获取动态代理；20 秒内复用缓存（供 asset 等同一次查询内多次请求）
func GetJDProxyURL() *url.URL {
	if !IsJdTaskProxyEnabled() {
		return nil
	}
	jdProxyCacheMu.Lock()
	if jdProxyCache != nil && time.Since(jdProxyCacheTime) < jdProxyCacheTTL {
		u := jdProxyCache
		age := time.Since(jdProxyCacheTime)
		jdProxyCacheMu.Unlock()
		logs.Info("[京东代理] 使用缓存 IP: %s (已缓存 %.0fs / %ds)", u.Host, age.Seconds(), int(jdProxyCacheTTL.Seconds()))
		return u
	}
	jdProxyCacheMu.Unlock()
	logs.Info("[京东代理] 缓存过期或为空，准备请求 API 取 IP")
	return refreshJDProxyURL("asset")
}

// jdProxyLogTransport 包装 Transport，记录每次 HTTP 是否经代理发出
type jdProxyLogTransport struct {
	base      *http.Transport
	proxyHost string
	count     int64
}

func (t *jdProxyLogTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	start := time.Now()
	resp, err := t.base.RoundTrip(req)
	elapsed := time.Since(start)
	atomic.AddInt64(&t.count, 1)
	status := 0
	if resp != nil {
		status = resp.StatusCode
	}
	// 仅记录慢请求、失败请求，避免逐条刷屏
	if err != nil || status != http.StatusOK || elapsed >= 2*time.Second {
		target := req.URL.Host + req.URL.Path
		if err != nil {
			if t.proxyHost != "" {
				logs.Warn("[京东代理] [jd_query] → %s %s 经代理 %s 失败(%v): %v", req.Method, target, t.proxyHost, elapsed, err)
			} else {
				logs.Warn("[京东代理] [jd_query] → %s %s 直连失败(%v): %v", req.Method, target, elapsed, err)
			}
		} else if t.proxyHost != "" {
			logs.Info("[京东代理] [jd_query] → %s %s 经代理 %s | HTTP %d | %v", req.Method, target, t.proxyHost, status, elapsed)
		} else {
			logs.Info("[京东代理] [jd_query] → %s %s 直连 | HTTP %d | %v", req.Method, target, status, elapsed)
		}
	}
	return resp, err
}

func (t *jdProxyLogTransport) RequestCount() int64 {
	if t == nil {
		return 0
	}
	return atomic.LoadInt64(&t.count)
}

func (t *jdProxyLogTransport) ProxyHost() string {
	if t == nil {
		return ""
	}
	return t.proxyHost
}

// NewJDProxyHTTPClient 每次 Go 资产查询强制取新 IP（IP 仅 30 秒有效，整次查询共用）
func NewJDProxyHTTPClient() (*http.Client, *jdProxyLogTransport) {
	base := &http.Transport{
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   30,
		MaxConnsPerHost:       30,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 15 * time.Second,
	}
	logTr := &jdProxyLogTransport{base: base}
	if !IsJdTaskProxyEnabled() {
		if strings.TrimSpace(sysConfig.JdTaskProxyUrl) == "" {
			logs.Info("[京东代理] Go查询直连: 未配置动态 IP API")
		} else {
			logs.Info("[京东代理] Go查询直连: 代理开关未启用")
		}
		return &http.Client{Timeout: 20 * time.Second, Transport: logTr}, logTr
	}
	logs.Info("[京东代理] 京豆/农场查询，请求 API 取新 IP")
	if proxyURL := refreshJDProxyURL("jd_query"); proxyURL != nil {
		base.Proxy = http.ProxyURL(proxyURL)
		logTr.proxyHost = proxyURL.Host
		logs.Info("[京东代理] 京豆/农场已绑定代理: %s", proxyURL.Host)
	} else {
		logs.Warn("[京东代理] 京豆/农场取 IP 失败，将直连")
	}
	return &http.Client{
		Timeout:   20 * time.Second,
		Transport: logTr,
	}, logTr
}

func refreshJDProxyURL(caller string) *url.URL {
	if !IsJdTaskProxyEnabled() {
		return nil
	}
	u, err := fetchDynamicProxyURL(caller)
	jdProxyCacheMu.Lock()
	defer jdProxyCacheMu.Unlock()
	if err != nil {
		logs.Warn("[京东代理] [%s] 获取动态 IP 失败: %v", caller, err)
		if jdProxyCache != nil {
			logs.Warn("[京东代理] [%s] 回退使用缓存 IP: %s", caller, jdProxyCache.Host)
		}
		return jdProxyCache
	}
	jdProxyCache = u
	jdProxyCacheTime = time.Now()
	logs.Info("[京东代理] [%s] API 取 IP 成功: %s", caller, u.String())
	return u
}

func fetchDynamicProxyURL(caller string) (*url.URL, error) {
	apiURL := strings.TrimSpace(sysConfig.JdTaskProxyUrl)
	if apiURL == "" {
		return nil, fmt.Errorf("动态 IP API 地址为空")
	}
	renum := 10
	if s := strings.TrimSpace(sysConfig.JdTaskProxyRenum); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			renum = n
		}
	}
	redelay := 2
	if s := strings.TrimSpace(sysConfig.JdTaskProxyRedelay); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			redelay = n
		}
	}
	logs.Info("[京东代理] [%s] 调用 API: %s (最多重试%d次)", caller, maskProxyAPIURL(apiURL), renum)
	client := &http.Client{Timeout: 15 * time.Second}
	var lastErr error
	for i := 0; i < renum; i++ {
		if i > 0 {
			logs.Info("[京东代理] [%s] 第%d次重试，等待%ds...", caller, i+1, redelay)
			time.Sleep(time.Duration(redelay) * time.Second)
		}
		req, err := http.NewRequest(http.MethodGet, apiURL, nil)
		if err != nil {
			lastErr = err
			logs.Warn("[京东代理] [%s] 第%d次请求构建失败: %v", caller, i+1, err)
			continue
		}
		start := time.Now()
		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			logs.Warn("[京东代理] [%s] 第%d次请求失败(%v): %v", caller, i+1, time.Since(start), err)
			continue
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = err
			logs.Warn("[京东代理] [%s] 第%d次读取响应失败: %v", caller, i+1, err)
			continue
		}
		bodyText := strings.TrimSpace(string(body))
		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("HTTP %d: %s", resp.StatusCode, bodyText)
			logs.Warn("[京东代理] [%s] 第%d次 HTTP %d，响应: %s", caller, i+1, resp.StatusCode, truncateLog(bodyText, 200))
			continue
		}
		logs.Info("[京东代理] [%s] 第%d次 API 响应(%v): %s", caller, i+1, time.Since(start), truncateLog(bodyText, 200))
		proxyURL, err := parseDynamicProxyResponse(body)
		if err != nil {
			lastErr = err
			logs.Warn("[京东代理] [%s] 第%d次解析 IP 失败: %v", caller, i+1, err)
			continue
		}
		return proxyURL, nil
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("未知错误")
	}
	return nil, fmt.Errorf("动态代理 API 获取失败(重试%d次): %w", renum, lastErr)
}

func maskProxyAPIURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	q := u.Query()
	for _, key := range []string{"vkey", "apikey", "key", "secret", "token"} {
		if q.Get(key) != "" {
			q.Set(key, "***")
		}
	}
	u.RawQuery = q.Encode()
	return u.String()
}

func truncateLog(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

// parseDynamicProxyResponse 解析代理 API 响应，兼容携趣等「标准文本 ip:port」及 JSON 格式
func parseDynamicProxyResponse(data []byte) (*url.URL, error) {
	text := strings.TrimSpace(string(data))
	if text == "" {
		return nil, fmt.Errorf("代理 API 返回空内容")
	}

	var raw interface{}
	if err := json.Unmarshal(data, &raw); err == nil {
		if u, ok := proxyURLFromJSON(raw); ok {
			return u, nil
		}
	}

	lines := strings.FieldsFunc(text, func(r rune) bool { return r == '\n' || r == '\r' })
	for _, line := range lines {
		line = strings.TrimSpace(strings.Trim(line, `"`))
		if line == "" {
			continue
		}
		if strings.Contains(line, "://") {
			u, err := url.Parse(line)
			if err == nil {
				return u, nil
			}
		}
		if match := ipPortPattern.FindString(line); match != "" {
			return url.Parse("http://" + match)
		}
	}
	return nil, fmt.Errorf("无法解析代理 API 响应: %s", text)
}

func proxyURLFromJSON(raw interface{}) (*url.URL, bool) {
	switch v := raw.(type) {
	case map[string]interface{}:
		for _, key := range []string{"data", "obj", "result", "proxy"} {
			if nested, ok := v[key]; ok {
				if u, ok := proxyURLFromJSON(nested); ok {
					return u, true
				}
			}
		}
		if ip := stringify(v["ip"]); ip != "" {
			port := stringify(v["port"])
			if port != "" {
				u, err := url.Parse("http://" + ip + ":" + port)
				return u, err == nil
			}
		}
		if proxy := stringify(v["proxy"]); proxy != "" {
			if !strings.Contains(proxy, "://") {
				proxy = "http://" + proxy
			}
			u, err := url.Parse(proxy)
			return u, err == nil
		}
	case []interface{}:
		if len(v) > 0 {
			return proxyURLFromJSON(v[0])
		}
	case string:
		if u, err := parseDynamicProxyResponse([]byte(v)); err == nil {
			return u, true
		}
	}
	return nil, false
}
