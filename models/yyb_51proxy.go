package models

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/cdle/xdd/yyb"
)

const (
	Yyb51FailClosedProxy           = "socks5://127.0.0.1:9"
	yyb51ShortLivedProxyExtractTime = 3
	yyb51ShortLivedProxyTTL        = 5 * time.Minute
	yyb51DefaultAPIBase            = "http://bapi.51daili.com/getapi2"
)

// YybProxyAreaRequest 门户/后台加载 51 代理省市区
type YybProxyAreaRequest struct {
	PackID       string `json:"packid"`
	ParentCode   string `json:"parent_code"`
	LinePoolType string `json:"line_pool_type"`
}

// YybProxyLoginOption 扫码登录与账号级代理续提
type YybProxyLoginOption struct {
	Enabled          bool   `json:"enabled"`
	PackID           string `json:"packid"`
	RegionCode       string `json:"region_code"`
	RegionName       string `json:"region_name"`
	ExistingProxyURL string `json:"-"`
	ExistingIPPort   string `json:"-"`
}

type yyb51ProxyItem struct {
	IPPort          string `json:"ipport"`
	RegionCode      any    `json:"regioncode"`
	IPAddressCode   any    `json:"ipaddress"`
	ExpireTime      string `json:"expiretime"`
	ExpireTimeCamel string `json:"expireTime"`
	IPLower         string `json:"ip"`
	IP              string `json:"IP"`
	Port            any    `json:"Port"`
	IpAddress       string `json:"IpAddress"`
	IPAddressName   string `json:"IpAddressName"`
}

type yyb51ProxyResponse struct {
	Code    int              `json:"code"`
	Success any              `json:"success"`
	Msg     string           `json:"msg"`
	Data    []yyb51ProxyItem `json:"data"`
}

func yyb51ConfigValues() (enabled bool, apiBase, accessName, accessPassword, uid, defaultPack, linePool, isp, bypassCode, bypassName string) {
	c := Config.Yyb
	enabled = c.Proxy51Enabled
	apiBase = strings.TrimSpace(c.Proxy51APIBase)
	if apiBase == "" {
		apiBase = yyb51DefaultAPIBase
	}
	accessName = strings.TrimSpace(c.Proxy51AccessName)
	accessPassword = strings.TrimSpace(c.Proxy51AccessPassword)
	uid = strings.TrimSpace(c.Proxy51UID)
	defaultPack = strings.TrimSpace(c.Proxy51DefaultPackID)
	if defaultPack == "" {
		if strings.EqualFold(strings.TrimSpace(c.Proxy51Plan), "traffic") {
			defaultPack = "12"
		} else {
			defaultPack = "2"
		}
	}
	linePool = firstNonEmptyYyb(c.Proxy51LinePoolIndex, "-1")
	isp = strings.TrimSpace(c.Proxy51ISP)
	if isp == "" {
		isp = "1"
	}
	bypassCode = strings.TrimSpace(c.Proxy51BypassRegionCode)
	if bypassCode == "" {
		bypassCode = "310100"
	}
	bypassName = strings.TrimSpace(c.Proxy51BypassRegionName)
	if bypassName == "" {
		bypassName = "上海"
	}
	return
}

func YybProxyShouldBypass(regionCode, regionName string) bool {
	_, _, _, _, _, _, _, _, bypassCode, bypassName := yyb51ConfigValues()
	code := strings.TrimSpace(regionCode)
	name := strings.TrimSpace(regionName)
	if bypassCode != "" && code == bypassCode {
		return true
	}
	return bypassName != "" && name != "" && strings.Contains(name, bypassName)
}

// YybProxyAreaList 代理 51 地区 API，失败时回退内置列表
func YybProxyAreaList(req YybProxyAreaRequest) (map[string]interface{}, error) {
	_, _, _, _, _, defaultPack, linePool, _, _, _ := yyb51ConfigValues()
	packID := strings.TrimSpace(req.PackID)
	if packID == "" {
		packID = defaultPack
	}
	if packID == "" {
		return nil, fmt.Errorf("请先在后台配置 51 代理默认 packid")
	}
	form := url.Values{}
	form.Set("packid", packID)
	form.Set("type", firstNonEmptyYyb(req.LinePoolType, linePool, "-1"))
	parent := strings.TrimSpace(req.ParentCode)
	if parent != "" {
		form.Set("pRegionCode", parent)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	httpReq, _ := http.NewRequestWithContext(ctx, http.MethodPost, "https://www.51daili.com/index/api/area.html", strings.NewReader(form.Encode()))
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil || resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return yybNormalizeAreaListResponse(parent, yybFallbackAreaList(parent)), nil
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var out map[string]interface{}
	if json.Unmarshal(body, &out) != nil || yybAreaListEmpty(out, parent) {
		return yybNormalizeAreaListResponse(parent, yybFallbackAreaList(parent)), nil
	}
	return yybNormalizeAreaListResponse(parent, out), nil
}

// 兼容旧签名
func YybProxyAreaListLegacy(parentCode, packID string) (map[string]interface{}, error) {
	return YybProxyAreaList(YybProxyAreaRequest{ParentCode: parentCode, PackID: packID})
}

func yybNormalizeAreaListResponse(parent string, out map[string]interface{}) map[string]interface{} {
	if out == nil {
		out = map[string]interface{}{}
	}
	parent = strings.TrimSpace(parent)
	keys := []string{"provinceList", "province", "list"}
	if parent != "" {
		keys = []string{"city", "cityList", "list"}
	}
	for _, k := range keys {
		if arr, ok := out[k].([]interface{}); ok && len(arr) > 0 {
			out["list"] = arr
			return out
		}
	}
	fallback := yybFallbackAreaList(parent)
	for k, v := range fallback {
		out[k] = v
	}
	if parent != "" {
		if arr, ok := fallback["city"].([]interface{}); ok {
			out["list"] = arr
		}
	} else if arr, ok := fallback["provinceList"].([]interface{}); ok {
		out["list"] = arr
	}
	return out
}

func yybAreaListEmpty(out map[string]interface{}, parent string) bool {
	if out == nil {
		return true
	}
	keys := []string{"provinceList", "province", "list"}
	if strings.TrimSpace(parent) != "" {
		keys = []string{"city", "cityList", "list"}
	}
	for _, k := range keys {
		if arr, ok := out[k].([]interface{}); ok && len(arr) > 0 {
			return false
		}
	}
	return true
}

func yybFallbackAreaList(parent string) map[string]interface{} {
	parent = strings.TrimSpace(parent)
	if parent == "" {
		return map[string]interface{}{"provinceList": []interface{}{
			map[string]interface{}{"parentCode": 0, "regionCode": 320000, "regionName": "江苏"},
			map[string]interface{}{"parentCode": 0, "regionCode": 440000, "regionName": "广东"},
			map[string]interface{}{"parentCode": 0, "regionCode": 110000, "regionName": "北京"},
			map[string]interface{}{"parentCode": 0, "regionCode": 310000, "regionName": "上海"},
			map[string]interface{}{"parentCode": 0, "regionCode": 330000, "regionName": "浙江"},
		}}
	}
	cities := map[string][]interface{}{
		"320000": {
			map[string]interface{}{"parentCode": 320000, "regionCode": 320100, "regionName": "南京"},
			map[string]interface{}{"parentCode": 320000, "regionCode": 320500, "regionName": "苏州"},
			map[string]interface{}{"parentCode": 320000, "regionCode": 320200, "regionName": "无锡"},
		},
		"440000": {
			map[string]interface{}{"parentCode": 440000, "regionCode": 440100, "regionName": "广州"},
			map[string]interface{}{"parentCode": 440000, "regionCode": 440300, "regionName": "深圳"},
		},
		"110000": {map[string]interface{}{"parentCode": 110000, "regionCode": 110100, "regionName": "北京"}},
		"310000": {map[string]interface{}{"parentCode": 310000, "regionCode": 310100, "regionName": "上海"}},
		"330000": {
			map[string]interface{}{"parentCode": 330000, "regionCode": 330100, "regionName": "杭州"},
			map[string]interface{}{"parentCode": 330000, "regionCode": 330200, "regionName": "宁波"},
		},
	}
	return map[string]interface{}{"provinceList": []interface{}{}, "city": cities[parent]}
}

// YybBuildProxyForLogin 为应用宝扫码提取/复用 51 SOCKS5
func YybBuildProxyForLogin(opt YybProxyLoginOption) (string, map[string]interface{}, error) {
	enabled, _, _, _, _, defaultPack, _, _, bypassCode, bypassName := yyb51ConfigValues()
	if !enabled || !opt.Enabled {
		return "", nil, nil
	}
	packID := firstNonEmptyYyb(opt.PackID, defaultPack)
	regionCode := strings.TrimSpace(opt.RegionCode)
	regionName := strings.TrimSpace(opt.RegionName)
	if regionCode == "" && regionName == "" {
		regionCode = bypassCode
		regionName = bypassName
	}
	if YybProxyShouldBypass(regionCode, regionName) {
		return "", map[string]interface{}{
			"yyb_proxy_enabled":     false,
			"yyb_proxy_bypass":      true,
			"yyb_proxy_packid":      packID,
			"yyb_proxy_region_code": regionCode,
			"yyb_proxy_region_name": firstNonEmptyYyb(regionName, bypassName),
			"yyb_proxy_last_at":     time.Now().Unix(),
		}, nil
	}
	if strings.TrimSpace(opt.ExistingProxyURL) != "" && yybProxyStillUsable(opt.ExistingProxyURL) {
		now := time.Now().Unix()
		return strings.TrimSpace(opt.ExistingProxyURL), map[string]interface{}{
			"yyb_proxy_enabled":       true,
			"yyb_proxy_packid":        packID,
			"yyb_proxy_region_code":   regionCode,
			"yyb_proxy_region_name":   regionName,
			"yyb_proxy_url":           strings.TrimSpace(opt.ExistingProxyURL),
			"yyb_proxy_ipport":        strings.TrimSpace(opt.ExistingIPPort),
			"yyb_proxy_expire_at":     now + int64(yyb51ShortLivedProxyTTL/time.Second),
			"yyb_proxy_last_at":       now,
			"yyb_proxy_last_alive_at": now,
			"yyb_proxy_reused":        true,
		}, nil
	}
	return yybExtract51Proxy(packID, regionCode, regionName)
}

// YybProxyTCPForCredentials 按账号已保存的代理地区续提 SOCKS5（getCode 等调用）
func YybProxyTCPForCredentials(credentials map[string]any) (proxyURL string, updated map[string]any, changed bool) {
	enabled, _, _, _, _, defaultPack, _, _, bypassCode, bypassName := yyb51ConfigValues()
	if !enabled || credentials == nil {
		return "", nil, false
	}
	cred := copyMapAny(credentials)
	savedRegionCode := stringFromYybAny(cred["yyb_proxy_region_code"])
	savedRegionName := stringFromYybAny(cred["yyb_proxy_region_name"])
	if boolFromYybAny(cred["yyb_proxy_bypass"]) || YybProxyShouldBypass(savedRegionCode, savedRegionName) {
		cred["yyb_proxy_enabled"] = false
		cred["yyb_proxy_bypass"] = true
		cred["yyb_proxy_region_code"] = firstNonEmptyYyb(savedRegionCode, bypassCode)
		cred["yyb_proxy_region_name"] = firstNonEmptyYyb(savedRegionName, bypassName)
		return "", cred, true
	}
	if !boolFromYybAny(cred["yyb_proxy_enabled"]) {
		return "", nil, false
	}
	proxyURL = stringFromYybAny(cred["yyb_proxy_url"])
	if proxyURL != "" && yybProxyStillUsable(proxyURL) {
		now := time.Now().Unix()
		cred["yyb_proxy_last_alive_at"] = now
		return proxyURL, cred, true
	}
	packID := firstNonEmptyYyb(stringFromYybAny(cred["yyb_proxy_packid"]), defaultPack)
	freshProxy, meta, err := yybExtract51Proxy(packID, savedRegionCode, savedRegionName)
	if err != nil || freshProxy == "" {
		if proxyURL != "" {
			return proxyURL, cred, false
		}
		return Yyb51FailClosedProxy, cred, false
	}
	yybApplyProxyMeta(cred, meta)
	return freshProxy, cred, true
}

func yybProxyStillUsable(proxyURL string) bool {
	proxyURL = strings.TrimSpace(proxyURL)
	if proxyURL == "" || proxyURL == Yyb51FailClosedProxy {
		return false
	}
	return yyb.ProbeTCPProxy(proxyURL)
}

func yybExtract51Proxy(packID, regionCode, regionName string) (string, map[string]interface{}, error) {
	_, apiBase, accessName, accessPassword, uid, _, linePool, isp, _, _ := yyb51ConfigValues()
	packID = strings.TrimSpace(packID)
	if packID == "" {
		return "", nil, fmt.Errorf("51 代理 packid 为空")
	}
	if accessName == "" || accessPassword == "" {
		return "", nil, fmt.Errorf("请先在后台配置 51 代理账号")
	}
	u, err := url.Parse(firstNonEmptyYyb(apiBase, yyb51DefaultAPIBase))
	if err != nil {
		return "", nil, err
	}
	q := u.Query()
	q.Set("linePoolIndex", linePool)
	q.Set("packid", packID)
	q.Set("time", fmt.Sprintf("%d", yyb51ShortLivedProxyExtractTime))
	q.Set("qty", "1")
	q.Set("port", "2")
	q.Set("format", "json")
	q.Set("field", "ipport,regioncode")
	q.Set("ct", "1")
	if isp != "" {
		q.Set("isp", isp)
	}
	q.Set("accessName", accessName)
	q.Set("accessPassword", accessPassword)
	q.Set("rid", yybRandID())
	if uid != "" {
		q.Set("uid", uid)
	}
	if regionCode = strings.TrimSpace(regionCode); regionCode != "" {
		q.Set("regionCode", regionCode)
	}
	u.RawQuery = q.Encode()
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", nil, fmt.Errorf("51 代理提取 HTTP %d", resp.StatusCode)
	}
	var parsed yyb51ProxyResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		ipport, expireText, regionText, parseErr := yybParseLoose51Proxy(body)
		if parseErr != nil {
			return "", nil, fmt.Errorf("51 代理返回非 JSON: %s", truncateStrYyb(string(body), 200))
		}
		if err := yybValidateExtractedRegion(regionCode, regionText); err != nil {
			return "", nil, err
		}
		return yybBuildProxyMetaFromIPPort(accessName, accessPassword, packID, regionCode, firstNonEmptyYyb(regionName, regionText), ipport, expireText)
	}
	if parsed.Code != 0 || !yyb51Success(parsed.Success) || len(parsed.Data) == 0 {
		msg := strings.TrimSpace(parsed.Msg)
		if msg == "" {
			msg = truncateStrYyb(string(body), 200)
		}
		return "", nil, fmt.Errorf("51 代理提取失败: %s", msg)
	}
	item := parsed.Data[0]
	ipport := strings.TrimSpace(item.IPPort)
	if ipport == "" {
		ipport = strings.TrimSpace(item.IPLower)
	}
	if ipport == "" && strings.TrimSpace(item.IP) != "" {
		ipport = fmt.Sprintf("%s:%s", strings.TrimSpace(item.IP), stringFromYybAny(item.Port))
	}
	if ipport == "" {
		return "", nil, fmt.Errorf("51 代理未返回 ipport")
	}
	returnedRegion := firstNonEmptyYyb(stringFromYybAny(item.RegionCode), stringFromYybAny(item.IPAddressCode), stringFromYybAny(item.IpAddress))
	if err := yybValidateExtractedRegion(regionCode, returnedRegion); err != nil {
		return "", nil, err
	}
	return yybBuildProxyMetaFromIPPort(accessName, accessPassword, packID, regionCode, firstNonEmptyYyb(regionName, item.IPAddressName, item.IpAddress), ipport, firstNonEmptyYyb(item.ExpireTime, item.ExpireTimeCamel))
}

func yybBuildProxyMetaFromIPPort(accessName, accessPassword, packID, regionCode, regionName, ipport, expireText string) (string, map[string]interface{}, error) {
	ipport = strings.TrimSpace(ipport)
	if ipport == "" {
		return "", nil, fmt.Errorf("51 代理未返回 ipport")
	}
	proxyURL := "socks5://" + url.UserPassword(accessName, accessPassword).String() + "@" + ipport
	expireAt := time.Now().Add(yyb51ShortLivedProxyTTL).Unix()
	if expireText != "" {
		if t, err := time.ParseInLocation("2006-01-02 15:04:05", expireText, time.Local); err == nil {
			expireAt = t.Unix()
		}
	}
	meta := map[string]interface{}{
		"yyb_proxy_enabled":     true,
		"yyb_proxy_packid":      packID,
		"yyb_proxy_region_code": regionCode,
		"yyb_proxy_region_name": regionName,
		"yyb_proxy_url":         proxyURL,
		"yyb_proxy_ipport":      ipport,
		"yyb_proxy_expire_at":   expireAt,
		"yyb_proxy_last_at":     time.Now().Unix(),
	}
	return proxyURL, meta, nil
}

func yybValidateExtractedRegion(expected, actual string) error {
	expected = strings.TrimSpace(expected)
	actual = strings.TrimSpace(actual)
	if expected == "" || actual == "" {
		return nil
	}
	if expected != actual {
		return fmt.Errorf("51 代理地区不匹配: 期望 %s, 实际 %s", expected, actual)
	}
	return nil
}

func yyb51Success(v any) bool {
	switch x := v.(type) {
	case bool:
		return x
	case string:
		return strings.EqualFold(strings.TrimSpace(x), "true") || strings.TrimSpace(x) == "1"
	case float64:
		return x != 0
	case int:
		return x != 0
	default:
		return false
	}
}

func yybParseLoose51Proxy(body []byte) (ipport, expireText, regionText string, err error) {
	var raw map[string]interface{}
	if e := json.Unmarshal(body, &raw); e != nil {
		return "", "", "", e
	}
	data, _ := raw["data"].([]interface{})
	if len(data) == 0 {
		return "", "", "", fmt.Errorf("empty data")
	}
	switch v := data[0].(type) {
	case string:
		return strings.TrimSpace(v), "", "", nil
	case map[string]interface{}:
		ipport = firstNonEmptyYyb(stringFromYybAny(v["ipport"]), stringFromYybAny(v["IPPort"]))
		if ipport == "" {
			ipport = stringFromYybAny(v["ip"])
		}
		if ipport == "" && stringFromYybAny(v["IP"]) != "" {
			ipport = stringFromYybAny(v["IP"]) + ":" + stringFromYybAny(v["Port"])
		}
		expireText = firstNonEmptyYyb(stringFromYybAny(v["expiretime"]), stringFromYybAny(v["expireTime"]), stringFromYybAny(v["ExpireTime"]))
		regionText = firstNonEmptyYyb(stringFromYybAny(v["regioncode"]), stringFromYybAny(v["ipaddress"]), stringFromYybAny(v["IpAddressName"]), stringFromYybAny(v["IpAddress"]))
		return ipport, expireText, regionText, nil
	default:
		return "", "", "", fmt.Errorf("unsupported data item")
	}
}

func yybApplyProxyMeta(dst map[string]any, meta map[string]interface{}) {
	if dst == nil || meta == nil {
		return
	}
	for k, v := range meta {
		dst[k] = v
	}
}

func yybRandID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err == nil {
		return fmt.Sprintf("%d%s", time.Now().UnixMilli(), hex.EncodeToString(b[:]))
	}
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

func firstNonEmptyYyb(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func truncateStrYyb(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

func stringFromYybAny(v any) string {
	switch x := v.(type) {
	case string:
		return strings.TrimSpace(x)
	case float64:
		if x == float64(int64(x)) {
			return fmt.Sprintf("%d", int64(x))
		}
		return fmt.Sprintf("%v", x)
	case int:
		return fmt.Sprintf("%d", x)
	case int64:
		return fmt.Sprintf("%d", x)
	default:
		return ""
	}
}

func boolFromYybAny(v any) bool {
	switch x := v.(type) {
	case bool:
		return x
	case string:
		return strings.EqualFold(strings.TrimSpace(x), "true") || strings.TrimSpace(x) == "1"
	case float64:
		return x != 0
	case int:
		return x != 0
	default:
		return false
	}
}

func copyMapAny(in map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range in {
		out[k] = v
	}
	return out
}

// PortalYybProxyConfig 门户展示 51 代理配置
func PortalYybProxyConfig() map[string]interface{} {
	enabled, _, _, _, _, defaultPack, linePool, _, bypassCode, bypassName := yyb51ConfigValues()
	return map[string]interface{}{
		"proxyEnabled":          enabled,
		"proxyDefaultPackid":    defaultPack,
		"proxyLinePoolIndex":    linePool,
		"proxyBypassRegionCode": bypassCode,
		"proxyBypassRegionName": bypassName,
	}
}
