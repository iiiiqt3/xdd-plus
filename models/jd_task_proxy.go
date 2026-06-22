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
	if u := GetJDProxyURL(); u != nil {
		return func(*http.Request) (*url.URL, error) {
			return u, nil
		}
	}
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
		jdProxyCacheMu.Unlock()
		return u
	}
	jdProxyCacheMu.Unlock()
	return refreshJDProxyURL()
}

// NewJDProxyHTTPClient 每次 Go 资产查询强制取新 IP（IP 仅 30 秒有效，整次查询共用）
func NewJDProxyHTTPClient() *http.Client {
	transport := &http.Transport{}
	if !IsJdTaskProxyEnabled() {
		return &http.Client{Timeout: 30 * time.Second, Transport: transport}
	}
	if proxyURL := refreshJDProxyURL(); proxyURL != nil {
		transport.Proxy = http.ProxyURL(proxyURL)
	}
	return &http.Client{
		Timeout:   30 * time.Second,
		Transport: transport,
	}
}

func refreshJDProxyURL() *url.URL {
	if !IsJdTaskProxyEnabled() {
		return nil
	}
	u, err := fetchDynamicProxyURL()
	jdProxyCacheMu.Lock()
	defer jdProxyCacheMu.Unlock()
	if err != nil {
		logs.Warn("[京东代理] 获取动态 IP 失败: %v", err)
		return jdProxyCache
	}
	jdProxyCache = u
	jdProxyCacheTime = time.Now()
	logs.Info("[京东代理] 获取动态 IP: %s", u.String())
	return u
}

func fetchDynamicProxyURL() (*url.URL, error) {
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
	client := &http.Client{Timeout: 15 * time.Second}
	var lastErr error
	for i := 0; i < renum; i++ {
		if i > 0 {
			time.Sleep(time.Duration(redelay) * time.Second)
		}
		req, err := http.NewRequest(http.MethodGet, apiURL, nil)
		if err != nil {
			lastErr = err
			continue
		}
		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = err
			continue
		}
		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
			continue
		}
		proxyURL, err := parseDynamicProxyResponse(body)
		if err != nil {
			lastErr = err
			continue
		}
		return proxyURL, nil
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("未知错误")
	}
	return nil, fmt.Errorf("动态代理 API 获取失败(重试%d次): %w", renum, lastErr)
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
