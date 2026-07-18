package models

import (
	"crypto/subtle"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/beego/beego/v2/server/web/context"
)

const SecretMaskSentinel = "********"

// MaskSecret 管理端展示用：非空密钥统一掩码，保存时配合 MergeSecretField 保留原值。
func MaskSecret(value string) string {
	if strings.TrimSpace(value) == "" {
		return ""
	}
	return SecretMaskSentinel
}

// IsMaskedSecret 判断是否为掩码占位或未修改的脱敏值。
func IsMaskedSecret(value string) bool {
	v := strings.TrimSpace(value)
	return v == "" || v == SecretMaskSentinel || strings.HasPrefix(v, "****")
}

// MergeSecretField 保存配置时：掩码/空值保留旧密钥。
func MergeSecretField(incoming, current string) string {
	if IsMaskedSecret(incoming) {
		return current
	}
	return strings.TrimSpace(incoming)
}

// VerifySharedToken 常量时间比较共享 token（query/header 均可）。
func VerifySharedToken(got, expected string) bool {
	got = strings.TrimSpace(got)
	expected = strings.TrimSpace(expected)
	if expected == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(got), []byte(expected)) == 1
}

// WebhookExpectedToken 微信 Hook / 消息推送等回调鉴权：优先 env，其次 config.wx.token。
func WebhookExpectedToken(envKey string) string {
	if t := strings.TrimSpace(GetEnv(envKey)); t != "" {
		return t
	}
	return strings.TrimSpace(Config.Wx.Token)
}

// QQWebSocketExpectedToken QQ 反向 WebSocket 鉴权 token。
func QQWebSocketExpectedToken() string {
	return strings.TrimSpace(GetEnv("qq_ws_token"))
}

// IsLoopbackIP 是否本机回环地址（go-cqhttp 同机部署时可放行）。
func IsLoopbackIP(ip string) bool {
	parsed := net.ParseIP(strings.TrimSpace(ip))
	if parsed == nil {
		return false
	}
	return parsed.IsLoopback()
}

// IsAllowedCORSOrigin 允许同源与配置白名单，替代 AllowAllOrigins。
func IsAllowedCORSOrigin(origin, requestHost string) bool {
	origin = strings.TrimSpace(origin)
	if origin == "" {
		return true
	}
	u, err := url.Parse(origin)
	if err != nil || u.Host == "" {
		return false
	}
	originHost := strings.ToLower(u.Hostname())
	if requestHost != "" {
		reqHost := strings.ToLower(strings.TrimSpace(requestHost))
		if h, _, err := net.SplitHostPort(reqHost); err == nil {
			reqHost = h
		}
		if originHost == reqHost || strings.EqualFold(u.Host, requestHost) {
			return true
		}
	}
	if originHost == "localhost" || originHost == "127.0.0.1" {
		return true
	}
	for _, allowed := range corsAllowedOrigins() {
		if strings.EqualFold(strings.TrimSpace(allowed), origin) {
			return true
		}
	}
	return false
}

func corsAllowedOrigins() []string {
	raw := strings.TrimSpace(GetEnv("cors_allowed_origins"))
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// ValidateOutboundHTTPURL 管理端「测试连通」防 SSRF：仅允许 http(s) 且禁止内网/本机。
func ValidateOutboundHTTPURL(raw string) (*url.URL, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("地址不能为空")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("地址格式无效")
	}
	scheme := strings.ToLower(u.Scheme)
	if scheme != "http" && scheme != "https" {
		return nil, fmt.Errorf("仅允许 http/https 地址")
	}
	host := strings.TrimSpace(u.Hostname())
	if host == "" {
		return nil, fmt.Errorf("地址缺少主机名")
	}
	if strings.EqualFold(host, "localhost") {
		return nil, fmt.Errorf("禁止访问本机地址")
	}
	if ip := net.ParseIP(host); ip != nil {
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified() {
			return nil, fmt.Errorf("禁止访问内网或保留地址")
		}
	}
	return u, nil
}

// RedactSensitiveLog 日志脱敏：CK、token、长 JSON。
func RedactSensitiveLog(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if len(s) > 256 {
		return fmt.Sprintf("<redacted len=%d>", len(s))
	}
	lower := strings.ToLower(s)
	for _, key := range []string{"pt_key=", "pt_pin=", "password", "token", "secret", "wskey"} {
		if strings.Contains(lower, key) {
			return "<redacted sensitive>"
		}
	}
	return s
}

// ---- 全局限流 ----

type rateLimitRule struct {
	limit   int
	window  time.Duration
	scope   string
}

type rateLimitEntry struct {
	mu      sync.Mutex
	hits    []time.Time
}

var (
	rateLimitStore sync.Map
	rateLimitRules = []struct {
		prefix string
		rule   rateLimitRule
	}{
		{"/api/login/reset/", rateLimitRule{limit: 8, window: 15 * time.Minute, scope: "reset"}},
		{"/api/getUserInfo", rateLimitRule{limit: 40, window: time.Minute, scope: "getUserInfo"}},
		{"/api/getUserPin", rateLimitRule{limit: 40, window: time.Minute, scope: "getUserPin"}},
		{"/permisson", rateLimitRule{limit: 30, window: time.Minute, scope: "permisson"}},
		{"/api/wxserver", rateLimitRule{limit: 30, window: time.Minute, scope: "wxserver"}},
		{"/api/send_wx_msg", rateLimitRule{limit: 30, window: time.Minute, scope: "wxmsg"}},
		{"/wx/receive", rateLimitRule{limit: 120, window: time.Minute, scope: "wxhook"}},
	}
	defaultRateLimit = rateLimitRule{limit: 180, window: time.Minute, scope: "global"}
)

func rateLimitKey(ip, scope string) string {
	return scope + ":" + ip
}

func allowRate(ip, path string) (bool, int) {
	ip = strings.TrimSpace(ip)
	if ip == "" {
		ip = "unknown"
	}
	rule := defaultRateLimit
	for _, item := range rateLimitRules {
		if strings.HasPrefix(path, item.prefix) {
			rule = item.rule
			break
		}
	}
	key := rateLimitKey(ip, rule.scope)
	now := time.Now()
	v, _ := rateLimitStore.LoadOrStore(key, &rateLimitEntry{})
	entry := v.(*rateLimitEntry)
	entry.mu.Lock()
	defer entry.mu.Unlock()
	cutoff := now.Add(-rule.window)
	kept := entry.hits[:0]
	for _, t := range entry.hits {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}
	if len(kept) >= rule.limit {
		retry := int(rule.window.Seconds())
		if len(kept) > 0 {
			retry = int(kept[0].Add(rule.window).Sub(now).Seconds())
			if retry < 1 {
				retry = 1
			}
		}
		entry.hits = kept
		return false, retry
	}
	entry.hits = append(kept, now)
	return true, 0
}

// BuildCORSAllowOrigins 构建 CORS 白名单（localhost + env 配置）。
func BuildCORSAllowOrigins() []string {
	origins := []string{
		"http://localhost:*",
		"http://127.0.0.1:*",
		"https://localhost:*",
		"https://127.0.0.1:*",
	}
	return append(origins, corsAllowedOrigins()...)
}

// RateLimitBeegoFilter Beego 全局限流过滤器。
func RateLimitBeegoFilter(ctx *context.Context) {
	path := ctx.Request.URL.Path
	if strings.HasPrefix(path, "/static/") || strings.HasPrefix(path, "/vweb/") || strings.HasPrefix(path, "/uploads/") {
		return
	}
	ip := ctx.Input.IP()
	ok, retry := allowRate(ip, path)
	if !ok {
		ctx.Output.Header("Retry-After", fmt.Sprintf("%d", retry))
		ctx.Abort(http.StatusTooManyRequests, `{"code":429,"msg":"请求过于频繁，请稍后再试"}`)
	}
}

// CheckOptionalApiToken 可选 ApiToken：配置了则必须匹配（query/body 由调用方传入）。
func CheckOptionalApiToken(token string) bool {
	expected := strings.TrimSpace(Config.ApiToken)
	if expected == "" {
		return true
	}
	return VerifySharedToken(token, expected)
}

// MaskSystemConfigForAdmin 系统 env 配置脱敏。
func MaskSystemConfigForAdmin(cfg SystemConfig) SystemConfig {
	cfg.ImagePassword = MaskSecret(cfg.ImagePassword)
	cfg.ImageToken = MaskSecret(cfg.ImageToken)
	cfg.RabbitApiToken = MaskSecret(cfg.RabbitApiToken)
	cfg.RabbitToken = MaskSecret(cfg.RabbitToken)
	cfg.NolanToken = MaskSecret(cfg.NolanToken)
	cfg.BBKToken = MaskSecret(cfg.BBKToken)
	return cfg
}

// MergeSystemConfigSecrets 保存系统配置时合并掩码字段。
func MergeSystemConfigSecrets(incoming, current SystemConfig) SystemConfig {
	incoming.ImagePassword = MergeSecretField(incoming.ImagePassword, current.ImagePassword)
	incoming.ImageToken = MergeSecretField(incoming.ImageToken, current.ImageToken)
	incoming.RabbitApiToken = MergeSecretField(incoming.RabbitApiToken, current.RabbitApiToken)
	incoming.RabbitToken = MergeSecretField(incoming.RabbitToken, current.RabbitToken)
	incoming.NolanToken = MergeSecretField(incoming.NolanToken, current.NolanToken)
	incoming.BBKToken = MergeSecretField(incoming.BBKToken, current.BBKToken)
	return incoming
}

// MaskJdConfigForAdmin config.yaml 敏感项脱敏（管理端 GET）。
func MaskJdConfigForAdmin(cfg map[string]interface{}) map[string]interface{} {
	maskKeys := []string{"apiToken", "master", "wxToken", "yybAPIToken", "jpushMasterSecret"}
	for _, key := range maskKeys {
		if v, ok := cfg[key].(string); ok && strings.TrimSpace(v) != "" {
			cfg[key] = MaskSecret(v)
			cfg[key+"Configured"] = true
		}
	}
	return cfg
}
