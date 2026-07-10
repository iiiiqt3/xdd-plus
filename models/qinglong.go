package models

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

)

// ===================== 青龙配置结构 =====================
type QingLongConfig struct {
	Name         string `json:"name"`
	Host         string `json:"host"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	Timeout      int    `json:"timeout"`
}

// ===================== 青龙API响应结构 =====================
type QLTokenResponse struct {
	Code int `json:"code"`
	Data struct {
		Token      string `json:"token"`
		TokenType  string `json:"token_type"` // 新增：兼容青龙返回的token_type字段
		Expiration int64  `json:"expiration"`
	} `json:"data"`
	Message string `json:"message"`
}

type QLEnvItem struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Value     string `json:"value"`
	Remarks   string `json:"remarks"`
	Status    int    `json:"status"` // 0-禁用，1-启用
	Timestamp string `json:"timestamp,omitempty"`
	Position  int64  `json:"position,omitempty"`
	UpdatedAt string `json:"updatedAt,omitempty"`
	CreatedAt string `json:"createdAt,omitempty"`
}

// 修复：兼容青龙API的两种响应格式（code / statusCode）
type QLCommonResponse struct {
	Code       int         `json:"code"`                 // 青龙主要返回的字段
	StatusCode int         `json:"statusCode,omitempty"` // 兼容部分接口的返回
	Error      string      `json:"error"`
	Message    string      `json:"message"`
	Data       interface{} `json:"data"`
}

type QLEnvsResponse struct {
	Code    int         `json:"code"`
	Data    []QLEnvItem `json:"data"`
	Message string      `json:"message"`
}

// ===================== 青龙API客户端 =====================
type QingLongClient struct {
	config      *QingLongConfig
	token       string
	tokenExpire int64 // Token过期时间（秒级）
	client      *http.Client
	mu          sync.RWMutex
}

// NewQingLongClient 创建青龙客户端实例
func NewQingLongClient(config *QingLongConfig) *QingLongClient {
	return NewQingLongClientWithTimeout(config, 0)
}

func NewQingLongClientWithTimeout(config *QingLongConfig, timeoutSeconds int) *QingLongClient {
	if config == nil {
		panic("QingLongClient: config is nil")
	}

	timeout := timeoutSeconds
	if timeout <= 0 {
		timeout = config.Timeout
	}
	if timeout <= 0 {
		timeout = 60
	}

	return &QingLongClient{
		config: config,
		client: &http.Client{
			Timeout: time.Duration(timeout) * time.Second,
			Transport: &http.Transport{
				DisableKeepAlives: false,
				MaxIdleConns:      30,
				MaxConnsPerHost:   10,
				IdleConnTimeout:   30 * time.Second,
			},
		},
	}
}

// ===================== 工具函数 =====================
func ExtractUserIDFromRemarks(remarks string) string {
	if remarks == "" {
		return ""
	}

	parts := strings.Split(remarks, "/")
	if len(parts) == 0 {
		return ""
	}

	// 兼容带日期的备注（如 大师/694738267/2026-01-01）
	idIndex := len(parts) - 1
	if len(parts) >= 3 {
		idIndex = len(parts) - 2 // 日期在最后一位，ID在倒数第二位
	}
	idPart := strings.TrimSpace(parts[idIndex])

	// 校验是否为纯数字
	for _, r := range idPart {
		if r < '0' || r > '9' {
			return ""
		}
	}

	return idPart
}

// ===================== API方法 =====================
// GetToken 获取/刷新青龙Token
func (q *QingLongClient) GetToken() (string, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	// 检查Token是否有效（未过期，提前60秒刷新）
	now := time.Now().Unix()
	if q.token != "" && q.tokenExpire > now+60 {
		return q.token, nil
	}

	url := strings.TrimSuffix(q.config.Host, "/") + "/open/auth/token"

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %v", err)
	}

	qsp := req.URL.Query()
	qsp.Add("client_id", q.config.ClientID)
	qsp.Add("client_secret", q.config.ClientSecret)
	req.URL.RawQuery = qsp.Encode()
	req.Header.Set("Content-Type", "application/json")

	Qinglong().Infof("获取青龙Token请求URL: %s", req.URL.String())

	resp, err := q.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应失败: %v", err)
	}

	Qinglong().Infof("青龙Token响应: %s", string(body))

	var tokenResp QLTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", fmt.Errorf("解析Token响应失败: %v, 响应内容: %s", err, string(body))
	}

	if tokenResp.Code != 200 {
		return "", fmt.Errorf("获取Token失败: %s (code: %d)", tokenResp.Message, tokenResp.Code)
	}

	// 修复：正确处理Token过期时间（兼容毫秒/秒级时间戳）
	q.token = tokenResp.Data.Token
	if tokenResp.Data.Expiration > 1e10 { // 毫秒级时间戳（大于10位）
		q.tokenExpire = tokenResp.Data.Expiration / 1000
	} else { // 秒级时间戳
		q.tokenExpire = tokenResp.Data.Expiration
	}

	Qinglong().Infof("Token获取成功，过期时间: %d (当前时间: %d)", q.tokenExpire, now)
	return q.token, nil
}

// QueryEnvs 查询环境变量列表
// QueryEnvs 查询环境变量列表
func (q *QingLongClient) QueryEnvs(searchValue string) ([]QLEnvItem, error) {
	token, err := q.GetToken()
	if err != nil {
		return nil, fmt.Errorf("获取Token失败: %v", err)
	}

	apiUrl := strings.TrimSuffix(q.config.Host, "/") + "/open/envs"
	if searchValue != "" {
		apiUrl += "?searchValue=" + url.QueryEscape(searchValue)
	}

	req, err := http.NewRequest("GET", apiUrl, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := q.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	// 空响应直接返回空数组
	bodyStr := strings.TrimSpace(string(body))
	if bodyStr == "" || bodyStr == "null" {
		return []QLEnvItem{}, nil
	}

	// 新增：打印青龙返回的原始环境变量数据
	Qinglong().Infof("青龙QueryEnvs原始响应: %s", bodyStr)

	var envsResp QLEnvsResponse
	if err := json.Unmarshal(body, &envsResp); err != nil {
		return nil, fmt.Errorf("解析环境变量响应失败: %v, 响应内容: %s", err, bodyStr)
	}

	if envsResp.Code != 200 {
		return nil, fmt.Errorf("查询环境变量失败: %s (code: %d)", envsResp.Message, envsResp.Code)
	}

	// 新增：打印解析后的QLEnvItem列表（包含Status）
	for i, env := range envsResp.Data {
		Qinglong().Infof("解析后的EnvItem[%d]: ID=%d, Name=%s, Remarks=%s, Status=%d",
			i, env.ID, env.Name, env.Remarks, env.Status)
	}

	if envsResp.Data == nil {
		return []QLEnvItem{}, nil
	}
	return envsResp.Data, nil
}

// SubmitEnv 提交环境变量（修复响应解析问题）
func (q *QingLongClient) SubmitEnv(envName, value, remarks string) error {
	token, err := q.GetToken()
	if err != nil {
		return fmt.Errorf("获取Token失败: %v", err)
	}

	url := strings.TrimSuffix(q.config.Host, "/") + "/open/envs"

	// 构造请求体（移除status字段，适配青龙API）
	payload := []map[string]interface{}{
		{
			"name":    envName,
			"value":   value,
			"remarks": remarks,
		},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("序列化请求数据失败: %v", err)
	}

	Qinglong().Infof("提交环境变量请求体: %s", string(jsonData))

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("创建请求失败: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := q.client.Do(req)
	if err != nil {
		return fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取响应失败: %v", err)
	}

	Qinglong().Infof("提交环境变量响应: %s", string(body))

	// 修复：正确解析响应（兼容code和statusCode字段）
	var submitResp QLCommonResponse
	if err := json.Unmarshal(body, &submitResp); err != nil {
		return fmt.Errorf("解析响应失败: %v, 响应内容: %s", err, string(body))
	}

	// 判断响应是否成功（优先用code，兼容statusCode）
	actualCode := submitResp.Code
	if actualCode == 0 {
		actualCode = submitResp.StatusCode
	}

	if actualCode != 200 {
		errorMsg := submitResp.Message
		if errorMsg == "" {
			errorMsg = submitResp.Error
		}
		if errorMsg == "" {
			errorMsg = "未知错误"
		}
		return fmt.Errorf("提交失败: %s (code: %d)", errorMsg, actualCode)
	}

	return nil
}

// UpdateEnv 默认启用状态更新环境变量
func (q *QingLongClient) UpdateEnv(envID int, envName, value, remarks string) error {
	return q.UpdateEnvWithStatus(envID, envName, value, remarks, 1)
}

// UpdateEnvContent 仅更新变量内容，不自动启用或禁用
func (q *QingLongClient) UpdateEnvContent(envID int, envName, value, remarks string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("更新失败: CK值为空 (code: 400)")
	}
	token, err := q.GetToken()
	if err != nil {
		return fmt.Errorf("获取Token失败: %v", err)
	}

	url := strings.TrimSuffix(q.config.Host, "/") + "/open/envs"
	payload := map[string]interface{}{
		"value":   value,
		"name":    envName,
		"remarks": remarks,
		"id":      envID,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("序列化请求数据失败: %v", err)
	}

	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("创建请求失败: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := q.client.Do(req)
	if err != nil {
		return fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取响应失败: %v", err)
	}

	var updateResp QLCommonResponse
	if err := json.Unmarshal(body, &updateResp); err != nil {
		return fmt.Errorf("解析响应失败: %v, 响应内容: %s", err, string(body))
	}

	actualCode := updateResp.Code
	if actualCode == 0 {
		actualCode = updateResp.StatusCode
	}
	if actualCode != 200 {
		errorMsg := updateResp.Message
		if errorMsg == "" {
			errorMsg = updateResp.Error
		}
		return fmt.Errorf("更新失败: %s (code: %d)", errorMsg, actualCode)
	}
	return nil
}

// remarksBelongsToProject 判断青龙备注是否属于该 DB 项目
// 规则：完整备注相同，或「备注别名 + 用户编号」相同（忽略到期日）
// 不单独用用户号/别名匹配，避免「大师」与「大师的机器人」互相命中
func remarksBelongsToProject(qlRemarks string, project *ActivityProject) bool {
	if project == nil {
		return false
	}
	expected := strings.TrimSpace(project.Remarks)
	qlRemarks = strings.TrimSpace(qlRemarks)
	if expected == "" || qlRemarks == "" {
		return false
	}
	if qlRemarks == expected {
		return true
	}

	dbAlias := strings.TrimSpace(project.RemarkAlias)
	if dbAlias == "" {
		dbAlias = GetFirstRemarkParam(expected)
	}
	qlAlias := GetFirstRemarkParam(qlRemarks)
	dbUID := ExtractUserIDFromRemarks(expected)
	qlUID := ExtractUserIDFromRemarks(qlRemarks)

	return dbAlias != "" && qlAlias != "" && dbAlias == qlAlias &&
		dbUID != "" && qlUID != "" && dbUID == qlUID
}

func isQLUniqueConstraintError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "uniqueconstraint") ||
		strings.Contains(msg, "unique constraint") ||
		(strings.Contains(msg, "validation error") && strings.Contains(msg, "unique"))
}

// TranslateQLError 将青龙/ORM 英文错误转为可读中文
func TranslateQLError(msg string) string {
	lower := strings.ToLower(msg)
	switch {
	case strings.Contains(lower, "uniqueconstraint") || strings.Contains(lower, "unique constraint"):
		return "青龙中已存在相同备注或相同变量值，不能重复创建"
	case strings.Contains(msg, "未找到") || strings.Contains(lower, "not found"):
		return "青龙中未找到对应环境变量"
	case strings.Contains(msg, "CK值为空"):
		return "CK值为空，无法更新"
	case strings.Contains(lower, "token"):
		return "青龙 Token 获取失败，请检查容器 ClientID/Secret"
	case strings.Contains(lower, "timeout") || strings.Contains(msg, "超时"):
		return "请求青龙超时"
	case strings.Contains(lower, "connection refused") || strings.Contains(msg, "连接"):
		return "无法连接青龙容器"
	default:
		// 去掉 Sequelize 等技术前缀，保留后半段
		if idx := strings.Index(msg, "Validation error"); idx >= 0 {
			return "青龙数据校验失败（备注或变量值可能与已有记录冲突）"
		}
		return msg
	}
}

// FormatQLSyncError 生成带业务上下文的中文同步错误
func FormatQLSyncError(action string, err error, project *ActivityProject) error {
	if err == nil {
		return nil
	}
	zh := TranslateQLError(SanitizeError(err).Error())
	ctx := action
	if project != nil {
		alias := project.RemarkAlias
		if alias == "" {
			alias = GetFirstRemarkParam(project.Remarks)
		}
		ctx = fmt.Sprintf("%s（活动=%s 账号=%s 变量=%s）", action, project.ActivityName, alias, project.EnvKey)
	}
	return fmt.Errorf("%s: %s", ctx, zh)
}

// FindEnvForProject 按「备注别名 + 用户编号」查找青龙变量（不使用 EnvID / CK）
// 备注格式：备注/用户编号/到期日；续费只改日期时仍能命中同一条
func (q *QingLongClient) FindEnvForProject(project *ActivityProject) (*QLEnvItem, string, error) {
	if project == nil {
		return nil, "", fmt.Errorf("项目为空")
	}

	if strings.TrimSpace(project.Remarks) == "" || strings.TrimSpace(project.EnvKey) == "" {
		return nil, "", fmt.Errorf("备注或变量名为空")
	}

	// 完整备注优先（含到期日完全一致）
	if env, err := q.FindEnvByRemarksExact(project.Remarks, project.EnvKey); err == nil {
		return env, "备注完全匹配", nil
	}

	alias := strings.TrimSpace(project.RemarkAlias)
	if alias == "" {
		alias = GetFirstRemarkParam(project.Remarks)
	}
	uid := ExtractUserIDFromRemarks(project.Remarks)
	if alias == "" || uid == "" {
		return nil, "", fmt.Errorf("未找到备注为 %s 的环境变量", project.Remarks)
	}

	searchKeys := []string{project.EnvKey, alias, uid}
	seen := make(map[int]bool)
	var candidates []*QLEnvItem
	for _, key := range searchKeys {
		envs, err := q.QueryEnvs(key)
		if err != nil {
			continue
		}
		for i := range envs {
			env := &envs[i]
			if env.Name != project.EnvKey || seen[env.ID] {
				continue
			}
			seen[env.ID] = true
			if remarksBelongsToProject(env.Remarks, project) {
				candidates = append(candidates, env)
			}
		}
	}

	if len(candidates) == 1 {
		return candidates[0], "备注/用户号匹配", nil
	}
	if len(candidates) > 1 {
		return nil, "", fmt.Errorf("青龙中存在多个匹配变量，备注=%s/%s", alias, uid)
	}
	return nil, "", fmt.Errorf("未找到备注为 %s 的环境变量", project.Remarks)
}

// ResolveEnvByProject 按备注别名 + 用户编号解析青龙变量
func (q *QingLongClient) ResolveEnvByProject(project *ActivityProject) (*QLEnvItem, error) {
	env, _, err := q.FindEnvForProject(project)
	return env, err
}

// IsQLEnvNotFoundError 判断青龙 API 是否返回变量不存在
func IsQLEnvNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "not found") ||
		strings.Contains(msg, "不存在") ||
		strings.Contains(msg, "未找到")
}

// UpdateEnvWithStatus 带状态更新环境变量
func (q *QingLongClient) UpdateEnvWithStatus(envID int, envName, value, remarks string, status int) error {
	token, err := q.GetToken()
	if err != nil {
		return fmt.Errorf("获取Token失败: %v", err)
	}

	url := strings.TrimSuffix(q.config.Host, "/") + "/open/envs"

	// 构造请求体（移除status字段）
	payload := map[string]interface{}{
		"value":   value,
		"name":    envName,
		"remarks": remarks,
		"id":      envID,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("序列化请求数据失败: %v", err)
	}

	Qinglong().Infof("更新环境变量请求体: %s", string(jsonData))

	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("创建请求失败: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := q.client.Do(req)
	if err != nil {
		return fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取响应失败: %v", err)
	}

	Qinglong().Infof("更新环境变量响应: %s", string(body))

	// 修复：正确解析响应
	var updateResp QLCommonResponse
	if err := json.Unmarshal(body, &updateResp); err != nil {
		return fmt.Errorf("解析响应失败: %v, 响应内容: %s", err, string(body))
	}

	actualCode := updateResp.Code
	if actualCode == 0 {
		actualCode = updateResp.StatusCode
	}

	if actualCode != 200 {
		errorMsg := updateResp.Message
		if errorMsg == "" {
			errorMsg = updateResp.Error
		}
		return fmt.Errorf("更新失败: %s (code: %d)", errorMsg, actualCode)
	}

	// 单独处理状态设置
	if status == 1 {
		if err := q.EnableEnv(envID); err != nil {
			Qinglong().Infof("启用环境变量ID%d失败: %v", envID, err)
		}
	} else {
		if err := q.DisableEnvs([]int{envID}); err != nil {
			Qinglong().Infof("禁用环境变量ID%d失败: %v", envID, err)
		}
	}

	return nil
}

// DeleteEnv 删除指定ID的环境变量
func (q *QingLongClient) DeleteEnv(envID int) error {
	token, err := q.GetToken()
	if err != nil {
		return fmt.Errorf("获取Token失败: %v", err)
	}

	url := strings.TrimSuffix(q.config.Host, "/") + "/open/envs"
	payload := []int{envID}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("序列化请求数据失败: %v", err)
	}

	Qinglong().Infof("删除环境变量请求体: %s", string(jsonData))

	req, err := http.NewRequest("DELETE", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("创建请求失败: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := q.client.Do(req)
	if err != nil {
		return fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取响应失败: %v", err)
	}

	Qinglong().Infof("删除环境变量响应: %s", string(body))

	var deleteResp QLCommonResponse
	if err := json.Unmarshal(body, &deleteResp); err != nil {
		return fmt.Errorf("解析响应失败: %v, 响应内容: %s", err, string(body))
	}

	actualCode := deleteResp.Code
	if actualCode == 0 {
		actualCode = deleteResp.StatusCode
	}

	if actualCode != 200 {
		errorMsg := deleteResp.Message
		if errorMsg == "" {
			errorMsg = deleteResp.Error
		}
		return fmt.Errorf("删除失败: %s (code: %d)", errorMsg, actualCode)
	}

	return nil
}

// DisableEnvs 批量禁用环境变量
func (q *QingLongClient) DisableEnvs(envIDs []int) error {
	token, err := q.GetToken()
	if err != nil {
		return fmt.Errorf("获取Token失败: %v", err)
	}

	url := strings.TrimSuffix(q.config.Host, "/") + "/open/envs/disable"
	jsonData, err := json.Marshal(envIDs)
	if err != nil {
		return fmt.Errorf("序列化请求数据失败: %v", err)
	}

	Qinglong().Infof("禁用环境变量请求体: %s", string(jsonData))

	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("创建请求失败: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := q.client.Do(req)
	if err != nil {
		return fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取响应失败: %v", err)
	}

	Qinglong().Infof("禁用环境变量响应: %s", string(body))

	var disableResp QLCommonResponse
	if err := json.Unmarshal(body, &disableResp); err != nil {
		return fmt.Errorf("解析响应失败: %v, 响应内容: %s", err, string(body))
	}

	actualCode := disableResp.Code
	if actualCode == 0 {
		actualCode = disableResp.StatusCode
	}

	if actualCode != 200 {
		errorMsg := disableResp.Message
		if errorMsg == "" {
			errorMsg = disableResp.Error
		}
		return fmt.Errorf("禁用失败: %s (code: %d)", errorMsg, actualCode)
	}

	return nil
}

// EnableEnv 启用指定ID的环境变量
func (q *QingLongClient) EnableEnv(envID int) error {
	token, err := q.GetToken()
	if err != nil {
		return fmt.Errorf("获取Token失败: %v", err)
	}

	url := strings.TrimSuffix(q.config.Host, "/") + "/open/envs/enable"
	payload := []int{envID}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("序列化请求数据失败: %v", err)
	}

	Qinglong().Infof("启用环境变量请求体: %s", string(jsonData))

	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("创建请求失败: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := q.client.Do(req)
	if err != nil {
		return fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取响应失败: %v", err)
	}

	Qinglong().Infof("启用环境变量响应: %s", string(body))

	var enableResp QLCommonResponse
	if err := json.Unmarshal(body, &enableResp); err != nil {
		return fmt.Errorf("解析响应失败: %v, 响应内容: %s", err, string(body))
	}

	actualCode := enableResp.Code
	if actualCode == 0 {
		actualCode = enableResp.StatusCode
	}

	if actualCode != 200 {
		errorMsg := enableResp.Message
		if errorMsg == "" {
			errorMsg = enableResp.Error
		}
		return fmt.Errorf("启用失败: %s (code: %d)", errorMsg, actualCode)
	}

	return nil
}

// GetRemarksByEnvName 根据环境变量名获取所有备注
func (q *QingLongClient) GetRemarksByEnvName(envName string) ([]string, error) {
	envs, err := q.QueryEnvs(envName)
	if err != nil {
		return nil, err
	}

	var remarks []string
	for _, env := range envs {
		if env.Name == envName {
			remarks = append(remarks, env.Remarks)
		}
	}

	return remarks, nil
}

// FindEnvByRemarksExact 按完整备注 + 变量名精确查找
func (q *QingLongClient) FindEnvByRemarksExact(remark, envName string) (*QLEnvItem, error) {
	remark = strings.TrimSpace(remark)
	if remark == "" {
		return nil, fmt.Errorf("备注为空")
	}

	searchKeys := []string{remark}
	if uid := ExtractUserIDFromRemarks(remark); uid != "" {
		searchKeys = append(searchKeys, uid)
	}
	if alias := GetFirstRemarkParam(remark); alias != "" && alias != remark {
		searchKeys = append(searchKeys, alias)
	}

	seenID := make(map[int]bool)
	for _, key := range searchKeys {
		envs, err := q.QueryEnvs(key)
		if err != nil {
			continue
		}
		for i := range envs {
			env := &envs[i]
			if seenID[env.ID] {
				continue
			}
			seenID[env.ID] = true
			if env.Name == envName && strings.TrimSpace(env.Remarks) == remark {
				return env, nil
			}
		}
	}
	return nil, fmt.Errorf("未找到备注为 %s 的环境变量", remark)
}

// FindEnvByRemarks 根据备注和环境变量名查找环境变量（按备注别名+用户号匹配，忽略到期日）
func (q *QingLongClient) FindEnvByRemarks(remark, envName string) (*QLEnvItem, error) {
	remark = strings.TrimSpace(remark)
	if remark == "" {
		return nil, fmt.Errorf("备注为空")
	}

	project := &ActivityProject{
		Remarks:     remark,
		RemarkAlias: GetFirstRemarkParam(remark),
		EnvKey:      envName,
	}

	searchKeys := []string{remark, envName}
	if uid := ExtractUserIDFromRemarks(remark); uid != "" {
		searchKeys = append(searchKeys, uid)
	}
	if alias := GetFirstRemarkParam(remark); alias != "" && alias != remark {
		searchKeys = append(searchKeys, alias)
	}

	seenID := make(map[int]bool)
	var candidates []*QLEnvItem
	for _, key := range searchKeys {
		envs, err := q.QueryEnvs(key)
		if err != nil {
			continue
		}
		for i := range envs {
			env := &envs[i]
			if seenID[env.ID] {
				continue
			}
			seenID[env.ID] = true
			if env.Name == envName && remarksBelongsToProject(env.Remarks, project) {
				candidates = append(candidates, env)
			}
		}
	}

	if len(candidates) == 1 {
		return candidates[0], nil
	}
	if len(candidates) > 1 {
		return nil, fmt.Errorf("青龙中存在多个匹配变量，备注=%s", remark)
	}
	return nil, fmt.Errorf("未找到备注为 %s 的环境变量", remark)
}

// ===================== 青龙 Cron 任务管理 API =====================

// QLCronTask 青龙任务结构
type QLCronTask struct {
	ID         int             `json:"id"`
	Name       string          `json:"name"`
	Command    string          `json:"command"`
	Extra      string          `json:"extra,omitempty"`
	Cron       string          `json:"cron"`                      // 部分青龙版本使用 cron
	Schedule   string          `json:"schedule"`                  // 部分青龙版本使用 schedule（与 cron 相同含义）
	Status     int             `json:"status"`                    // 0-运行中，1-空闲，4-禁用（部分版本）
	IsDisabled int             `json:"isDisabled"`                // 0-启用，1-禁用（青龙主要用此字段标记禁用状态）
	LastRun    json.RawMessage `json:"lastRunningTime,omitempty"` // 兼容字符串和数字时间戳
	LastRunMS  int64           `json:"last_running_time,omitempty"`
	// 兼容不同青龙版本的时间字段命名
	LastRunAlt    json.RawMessage `json:"last_execution,omitempty"`          // 部分版本用 last_execution
	LastRunAlt2   json.RawMessage `json:"lastRunTime,omitempty"`             // 部分版本用 lastRunTime
	LastRunAlt3   json.RawMessage `json:"lastRun,omitempty"`                 // 部分版本直接用 lastRun
	LastRunAlt4   json.RawMessage `json:"last_execution_time,omitempty"`     // 部分版本直接用 last_execution_time
	NextRunTime   json.RawMessage `json:"nextRunTime,omitempty"`             // 部分版本直接返回下次运行时间
	NextRunAlt    json.RawMessage `json:"nextRun,omitempty"`                 // 部分版本直接用 nextRun
	LastDuration  int             `json:"lastDuration,omitempty"`            // 青龙面板常用字段（毫秒）
	LastDuration2 int             `json:"lastRunningTimeDuration,omitempty"` // 兼容其他版本字段名（毫秒）
}

// GetFormattedLastRun 获取格式化的最后运行时间字符串
// 青龙API返回的时间可能是字符串 "2024-01-15 10:30:45" 或数字时间戳（毫秒/秒）
func (t *QLCronTask) GetFormattedLastRun() string {
	val := t.resolveTimeField(t.LastRun)
	if val != "" {
		return val
	}
	val = t.resolveTimeField(t.LastRunAlt)
	if val != "" {
		return val
	}
	val = t.resolveTimeField(t.LastRunAlt2)
	if val != "" {
		return val
	}
	return "-"
}

// GetFormattedNextRun 获取格式化的下次运行时间字符串
func (t *QLCronTask) GetFormattedNextRun() string {
	val := t.resolveTimeField(t.NextRunTime)
	if val != "" {
		return val
	}
	return ""
}

// resolveTimeField 解析时间字段（兼容字符串和数字时间戳）
func (t *QLCronTask) resolveTimeField(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	// 尝试解析为字符串
	var str string
	if json.Unmarshal(raw, &str) == nil && str != "" {
		if len(str) > 19 {
			str = str[:19]
		}
		return str
	}
	// 尝试解析为数字时间戳
	var ts float64
	if json.Unmarshal(raw, &ts) == nil && ts > 0 {
		var sec int64
		if ts > 1e12 { // 毫秒时间戳
			sec = int64(ts) / 1000
		} else { // 秒时间戳
			sec = int64(ts)
		}
		return time.Unix(sec, 0).Format("2006-01-02 15:04:05")
	}
	return ""
}

// QLCronTaskLog 青龙任务日志结构
type QLCronTaskLog struct {
	ID        int    `json:"id"`
	TaskID    int    `json:"taskId"`   // 青龙 API 驼峰命名
	TaskName  string `json:"taskName"` // 青龙 API 驼峰命名
	Command   string `json:"command"`
	Status    int    `json:"status"`    // 0-成功，1-部分成功，2-失败，3-超时，4-取消
	StartTime string `json:"startTime"` // 青龙 API 驼峰命名
	EndTime   string `json:"endTime"`   // 青龙 API 驼峰命名
	Duration  int    `json:"duration"`
	LogFile   string `json:"logFile,omitempty"` // 通过日志文件系统补充的精确日志文件名
	LogPath   string `json:"logPath,omitempty"` // 通过日志文件系统补充的日志目录
	// 兼容下划线命名（部分青龙版本）
	TaskIdAlt    int    `json:"task_id"`
	TaskNameAlt  string `json:"task_name"`
	StartTimeAlt string `json:"start_time"`
	EndTimeAlt   string `json:"end_time"`
	StatusText   string `json:"statusText"` // 部分版本直接返回状态文本
	// 兼容 createdAt/updatedAt 命名（新版青龙日志使用这些字段名）
	CreatedAt string `json:"createdAt,omitempty"`
	UpdatedAt string `json:"updatedAt,omitempty"`
}

// QLScriptPath 脚本路径拆解结果
type QLScriptPath struct {
	FullPath string
	Path     string
	Filename string
}

// QLCronTaskLogFile 青龙任务日志文件内容
type QLCronTaskLogFile struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Title string `json:"title"`
}

// QLCronTaskDetail 青龙任务详情（包含运行中的任务信息）
type QLCronTaskDetail struct {
	ID              int    `json:"id"`
	Name            string `json:"name"`
	Command         string `json:"command"`
	Cron            string `json:"cron"`
	Status          int    `json:"status"`
	LastRunningTime string `json:"lastRunningTime,omitempty"`
}

// QueryCronTasks 查询青龙定时任务列表
func (q *QingLongClient) QueryCronTasks(searchValue string) ([]QLCronTask, error) {
	token, err := q.GetToken()
	if err != nil {
		return nil, fmt.Errorf("获取Token失败: %v", err)
	}

	apiUrl := strings.TrimSuffix(q.config.Host, "/") + "/open/crons"
	if searchValue != "" {
		apiUrl += "?searchValue=" + url.QueryEscape(searchValue)
	}

	req, err := http.NewRequest("GET", apiUrl, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := q.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	// 空响应直接返回空数组
	bodyStr := strings.TrimSpace(string(body))
	if bodyStr == "" || bodyStr == "null" {
		return []QLCronTask{}, nil
	}

	var result struct {
		Code int `json:"code"`
		Data struct {
			Data  []QLCronTask `json:"data"`
			Total int          `json:"total"`
		} `json:"data"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		// 兼容：部分青龙版本 data 直接是数组
		var result2 struct {
			Code    int          `json:"code"`
			Data    []QLCronTask `json:"data"`
			Message string       `json:"message"`
		}
		if err2 := json.Unmarshal(body, &result2); err2 != nil {
			return nil, fmt.Errorf("解析任务列表失败: %v, 响应内容: %s", err, bodyStr)
		}
		if result2.Code != 200 {
			return nil, fmt.Errorf("查询任务失败: %s (code: %d)", result2.Message, result2.Code)
		}
		if result2.Data == nil {
			return []QLCronTask{}, nil
		}
		return result2.Data, nil
	}
	if result.Code != 200 {
		return nil, fmt.Errorf("查询任务失败: %s (code: %d)", result.Message, result.Code)
	}
	if result.Data.Data == nil {
		return []QLCronTask{}, nil
	}
	return result.Data.Data, nil
}

// GetRunningCronTasks 获取正在运行的任务
func (q *QingLongClient) GetRunningCronTasks() ([]QLCronTaskDetail, error) {
	token, err := q.GetToken()
	if err != nil {
		return nil, fmt.Errorf("获取Token失败: %v", err)
	}

	url := strings.TrimSuffix(q.config.Host, "/") + "/open/crons/detail"

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := q.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	var result struct {
		Code    int                `json:"code"`
		Data    []QLCronTaskDetail `json:"data"`
		Message string             `json:"message"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("解析运行中任务失败: %v", err)
	}
	if result.Code != 200 {
		return nil, fmt.Errorf("查询运行中任务失败: %s (code: %d)", result.Message, result.Code)
	}
	return result.Data, nil
}

// GetCronTaskLogs 获取任务日志列表
func (q *QingLongClient) GetCronTaskLogs(taskID int, page, limit int) ([]QLCronTaskLog, int, error) {
	token, err := q.GetToken()
	if err != nil {
		return nil, 0, fmt.Errorf("获取Token失败: %v", err)
	}

	url := fmt.Sprintf("%s/open/crons/%d/logs", strings.TrimSuffix(q.config.Host, "/"), taskID)
	if page > 0 || limit > 0 {
		url += fmt.Sprintf("?page=%d&limit=%d", page, limit)
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("创建请求失败: %v", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := q.client.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, fmt.Errorf("读取响应失败: %v", err)
	}

	var result struct {
		Code int `json:"code"`
		Data struct {
			Data  []QLCronTaskLog `json:"data"`
			Total int             `json:"total"`
		} `json:"data"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		// 兼容：部分青龙版本 data 直接是数组
		var result2 struct {
			Code int             `json:"code"`
			Data []QLCronTaskLog `json:"data"`
		}
		if err2 := json.Unmarshal(body, &result2); err2 != nil {
			bodyPreview := string(body)
			if len(bodyPreview) > 500 {
				bodyPreview = bodyPreview[:500]
			}
			return nil, 0, fmt.Errorf("解析日志列表失败: %v, 响应内容: %s", err, bodyPreview)
		}
		if result2.Code != 200 {
			return nil, 0, fmt.Errorf("查询日志失败: (code: %d)", result2.Code)
		}
		return result2.Data, len(result2.Data), nil
	}
	if result.Code != 200 {
		return nil, 0, fmt.Errorf("查询日志失败: %s (code: %d)", result.Message, result.Code)
	}
	return result.Data.Data, result.Data.Total, nil
}

// GetCronTaskLogContent 获取任务运行日志详细内容
// 青龙接口使用任务ID而不是日志ID：GET /open/crons/:id/log
// logFile 参数可选，用于尝试获取指定历史日志（部分青龙版本可能不支持）
func (q *QingLongClient) GetCronTaskLogContent(taskID int, logFile ...string) (string, error) {
	token, err := q.GetToken()
	if err != nil {
		return "", fmt.Errorf("获取Token失败: %v", err)
	}

	baseURL := strings.TrimSuffix(q.config.Host, "/")

	// 如果指定了日志文件名，先尝试通过文件名获取特定日志
	if len(logFile) > 0 && logFile[0] != "" {
		url := fmt.Sprintf("%s/open/crons/%d/log?file=%s", baseURL, taskID, logFile[0])
		content, err := q.doGetLogContent(token, url)
		if err == nil && content != "" {
			return content, nil
		}
		// 如果通过文件名获取失败，继续走默认方式获取最新日志
	}

	url := fmt.Sprintf("%s/open/crons/%d/log", baseURL, taskID)
	return q.doGetLogContent(token, url)
}

func (q *QingLongClient) doGetLogContent(token, url string) (string, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %v", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := q.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应失败: %v", err)
	}

	// 青龙日志接口可能直接返回文本或JSON
	contentType := resp.Header.Get("Content-Type")
	if strings.Contains(contentType, "text/plain") || strings.Contains(contentType, "text/html") {
		return string(body), nil
	}

	// 尝试解析JSON格式
	var result struct {
		Code    int    `json:"code"`
		Data    string `json:"data"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &result); err == nil {
		if result.Code == 200 || result.Code == 0 {
			return result.Data, nil
		}
		if result.Message != "" {
			return "", fmt.Errorf("获取日志失败: %s (code: %d)", result.Message, result.Code)
		}
	}

	// 如果JSON解析失败，可能直接就是日志文本
	return string(body), nil
}

// GetCronTaskScript 获取任务脚本内容
func (q *QingLongClient) GetCronTaskScript(taskID int) (string, error) {
	token, err := q.GetToken()
	if err != nil {
		return "", fmt.Errorf("获取Token失败: %v", err)
	}

	url := fmt.Sprintf("%s/open/crons/%d", strings.TrimSuffix(q.config.Host, "/"), taskID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %v", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := q.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应失败: %v", err)
	}

	var result struct {
		Code int `json:"code"`
		Data struct {
			Command string `json:"command"`
			Extra   string `json:"extra"`
		} `json:"data"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("解析任务详情失败: %v", err)
	}
	if result.Code != 200 {
		return "", fmt.Errorf("查询任务失败: %s (code: %d)", result.Message, result.Code)
	}
	return result.Data.Command, nil
}

// GetScriptFileContent 获取脚本文件内容（通过青龙文件系统 API）
// path 参数是子目录路径（如 "scripts"），filename 是文件名（如 "xmyx.js"）
// 青龙 API v2.19.2+ 要求 path 参数必须存在于 URL 中（即使为空字符串）
func (q *QingLongClient) GetScriptFileContent(path, filename string) (string, error) {
	if filename == "" {
		return "", fmt.Errorf("脚本文件名为空")
	}

	token, err := q.GetToken()
	if err != nil {
		return "", fmt.Errorf("获取Token失败: %v", err)
	}

	baseURL := strings.TrimSuffix(q.config.Host, "/")
	var emptyHit bool
	var lastErr error

	tryFetch := func(url, desc string) (string, bool) {
		content, fetchErr := q.doGetScriptFileContent(token, url)
		if fetchErr != nil {
			lastErr = fetchErr
			return "", false
		}
		if strings.TrimSpace(content) == "" {
			emptyHit = true
			Qinglong().Infof("GetScriptFileContent 命中空内容: file=%s desc=%s url=%s", filename, desc, url)
			return "", false
		}
		Qinglong().Infof("GetScriptFileContent 成功: file=%s desc=%s url=%s", filename, desc, url)
		return content, true
	}

	// 构建 path 变体列表
	// v2.19.2+ 要求 path 参数必须存在，空字符串 "" 表示青龙根目录（实际对应 /scripts/）
	var pathVariants []string
	if path != "" {
		// 有子目录路径：优先原样，再尝试带 / 前缀
		pathVariants = append(pathVariants, path)
		if !strings.HasPrefix(path, "/") {
			pathVariants = append(pathVariants, "/"+path)
		}
	} else {
		// 根目录脚本：优先只尝试真实可命中的根目录形式，避免误落到 2.20 的 Invalid path format
		pathVariants = append(pathVariants, "", ".")
	}
	// 追加所有可能的物理路径变体，兼容不同青龙版本的存储结构
	pathVariants = append(pathVariants, "scripts", "/scripts", "ql/scripts", "/ql/scripts", "data/scripts", "/data/scripts")

	// 去重（避免重复尝试相同路径）
	seen := make(map[string]bool)
	uniqueVariants := make([]string, 0, len(pathVariants))
	for _, p := range pathVariants {
		if !seen[p] {
			seen[p] = true
			uniqueVariants = append(uniqueVariants, p)
		}
	}

	// 方式1：GET /open/scripts/detail?file=xxx&path=xxx
	for _, tryPath := range uniqueVariants {
		values := url.Values{}
		values.Set("file", filename)
		values.Set("path", tryPath)
		url := fmt.Sprintf("%s/open/scripts/detail?%s", baseURL, values.Encode())
		if content, ok := tryFetch(url, "detail path="+tryPath); ok {
			return content, nil
		}
	}

	// 中文文件名在青龙 2.20 上通过 /open/scripts/<文件名>?path= 读取会直接 400，跳过这类 URI 形式
	isASCIIFileName := true
	for _, r := range filename {
		if r > 127 {
			isASCIIFileName = false
			break
		}
	}
	if !isASCIIFileName {
		if emptyHit {
			return "", fmt.Errorf("脚本文件接口返回空内容，未命中真实脚本路径: file=%s path=%s", filename, path)
		}
		if lastErr != nil {
			Qinglong().Infof("GetScriptFileContent 失败: file=%s path=%s err=%v", filename, path, lastErr)
			return "", fmt.Errorf("获取脚本文件失败: file=%s path=%s err=%v", filename, path, lastErr)
		}
		return "", fmt.Errorf("获取脚本文件失败: file=%s path=%s", filename, path)
	}

	// 方式2：GET /open/scripts/xxx?path=xxx（仅 ASCII 文件名尝试 URI 形式）
	encodedFileName := url.PathEscape(filename)
	urlRoot := fmt.Sprintf("%s/open/scripts/%s", baseURL, encodedFileName)
	// v2.19.2+ 必须带 path 参数
	urlRootWithPath := fmt.Sprintf("%s/open/scripts/%s?path=", baseURL, encodedFileName)
	if content, ok := tryFetch(urlRootWithPath, "root-uri path=空"); ok {
		return content, nil
	}
	if content, ok := tryFetch(urlRoot, "root-uri 无path"); ok {
		return content, nil
	}

	// 方式3：直接 GET /open/scripts/xxx?path=xxx
	for _, tryPath := range uniqueVariants {
		url := fmt.Sprintf("%s/open/scripts/%s?path=%s", baseURL, encodedFileName, url.QueryEscape(tryPath))
		if content, ok := tryFetch(url, "uri path="+tryPath); ok {
			return content, nil
		}
	}

	if emptyHit {
		return "", fmt.Errorf("脚本文件接口返回空内容，未命中真实脚本路径: file=%s path=%s", filename, path)
	}
	if lastErr != nil {
		Qinglong().Infof("GetScriptFileContent 失败: file=%s path=%s err=%v", filename, path, lastErr)
		return "", fmt.Errorf("获取脚本文件失败: file=%s path=%s err=%v", filename, path, lastErr)
	}
	return "", fmt.Errorf("获取脚本文件失败: file=%s path=%s", filename, path)
}

func (q *QingLongClient) doGetScriptFileContent(token, url string) (string, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %v", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := q.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应失败: %v", err)
	}

	// 非成功状态码
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body)[:minInt(len(body), 300)])
	}

	contentType := resp.Header.Get("Content-Type")
	if strings.Contains(contentType, "text/plain") || strings.Contains(contentType, "text/html") {
		return string(body), nil
	}

	// 尝试解析 JSON 响应（青龙返回 code:0 或 code:200 均有可能）
	var result struct {
		Code    int    `json:"code"`
		Data    string `json:"data"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return string(body), nil
	}
	if result.Code == 200 || result.Code == 0 {
		// data 允许为空字符串，表示脚本内容为空，不应把整个 JSON 原文返回给前端
		return result.Data, nil
	}
	return "", fmt.Errorf("青龙返回错误: %s (code: %d)", result.Message, result.Code)
}

// UpdateScriptFileContent 更新脚本文件内容（通过青龙文件系统 API）
// path 参数是子目录路径（如 "scripts"），filename 是文件名，content 是新内容
// 青龙 API：POST /open/scripts/，Content-Type: multipart/form-data
func (q *QingLongClient) UpdateScriptFileContent(path, filename, content string) error {
	if filename == "" {
		return fmt.Errorf("脚本文件名为空")
	}

	token, err := q.GetToken()
	if err != nil {
		return fmt.Errorf("获取Token失败: %v", err)
	}

	baseURL := strings.TrimSuffix(q.config.Host, "/")

	// 青龙官方 API：POST /open/scripts/，使用 multipart/form-data
	// 字段：filename（必填）、content（文件内容）、path（目录路径）
	err = q.doPostScriptFile(baseURL+"/open/scripts/", token, filename, content, path)
	if err == nil {
		return nil
	}

	return fmt.Errorf("更新脚本文件失败: %v", err)
}

// doPostScriptFile 通过 POST multipart/form-data 上传/更新脚本文件
func (q *QingLongClient) doPostScriptFile(url, token, filename, content, path string) error {
	// 构建 multipart/form-data 请求体
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// 必填字段：filename
	if err := writer.WriteField("filename", filename); err != nil {
		return fmt.Errorf("写入 filename 字段失败: %v", err)
	}

	// 内容字段：content
	if content != "" {
		if err := writer.WriteField("content", content); err != nil {
			return fmt.Errorf("写入 content 字段失败: %v", err)
		}
	}

	// 目录路径字段：path（可选）
	if path != "" {
		if err := writer.WriteField("path", path); err != nil {
			return fmt.Errorf("写入 path 字段失败: %v", err)
		}
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("关闭 multipart writer 失败: %v", err)
	}

	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return fmt.Errorf("创建请求失败: %v", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := q.client.Do(req)
	if err != nil {
		return fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取响应失败: %v", err)
	}

	var result QLCommonResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		bodyPreview := string(respBody)
		if len(bodyPreview) > 300 {
			bodyPreview = bodyPreview[:300]
		}
		return fmt.Errorf("解析响应失败: %v, 响应: %s", err, bodyPreview)
	}
	if result.Code != 200 {
		bodyPreview := string(respBody)
		if len(bodyPreview) > 300 {
			bodyPreview = bodyPreview[:300]
		}
		return fmt.Errorf("更新脚本失败: %s (code: %d, 响应: %s)", result.Message, result.Code, bodyPreview)
	}
	return nil
}

// minInt 返回两个整数中较小的那个（Go 1.20 兼容）
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// UpdateCronTaskScript 更新任务脚本
// UpdateCronTaskScript 更新任务脚本内容（command 字段）
// 青龙 PUT /open/crons 要求同时提交 id、command、schedule 三个字段，缺 schedule 会导致 400 验证失败
func (q *QingLongClient) UpdateCronTaskScript(taskID int, command string) error {
	token, err := q.GetToken()
	if err != nil {
		return fmt.Errorf("获取Token失败: %v", err)
	}

	// 先获取任务详情，拿到 schedule 字段（青龙更新接口要求必须传 schedule）
	cronExpr, err := q.getCronTaskScheduleField(taskID)
	if err != nil {
		return fmt.Errorf("获取任务信息失败: %v", err)
	}

	url := strings.TrimSuffix(q.config.Host, "/") + "/open/crons"
	payload := map[string]interface{}{
		"id":       taskID,
		"command":  command,
		"schedule": cronExpr,
	}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("序列化数据失败: %v", err)
	}

	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("创建请求失败: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := q.client.Do(req)
	if err != nil {
		return fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取响应失败: %v", err)
	}

	var result QLCommonResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("解析响应失败: %v", err)
	}
	if result.Code != 200 {
		bodyPreview := string(body)
		if len(bodyPreview) > 300 {
			bodyPreview = bodyPreview[:300]
		}
		return fmt.Errorf("更新脚本失败: %s (code: %d, 响应: %s)", result.Message, result.Code, bodyPreview)
	}
	return nil
}

// UpdateCronTaskSchedule 更新任务定时规则
// 青龙 PUT /open/crons 要求同时提交 id、command、schedule 三个字段，缺 command 会导致 400 验证失败
func (q *QingLongClient) UpdateCronTaskSchedule(taskID int, cronExpr string) error {
	token, err := q.GetToken()
	if err != nil {
		return fmt.Errorf("获取Token失败: %v", err)
	}

	// 先获取任务详情，拿到 command 字段（青龙更新接口要求必须传 command）
	command, err := q.GetCronTaskScript(taskID)
	if err != nil {
		return fmt.Errorf("获取任务信息失败: %v", err)
	}

	url := strings.TrimSuffix(q.config.Host, "/") + "/open/crons"

	// 先尝试用 schedule 字段名（新版青龙标准字段名）
	err = q.doUpdateCronSchedule(token, url, taskID, command, cronExpr, "schedule")
	if err == nil {
		return nil
	}

	// 如果失败，尝试用 cron 字段名（老版青龙）
	return q.doUpdateCronSchedule(token, url, taskID, command, cronExpr, "cron")
}

func (q *QingLongClient) doUpdateCronSchedule(token, url string, taskID int, command, cronExpr, fieldName string) error {
	payload := map[string]interface{}{
		"id":      taskID,
		"command": command,
	}
	payload[fieldName] = cronExpr
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("序列化数据失败: %v", err)
	}

	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("创建请求失败: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := q.client.Do(req)
	if err != nil {
		return fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取响应失败: %v", err)
	}

	var result QLCommonResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("解析响应失败: %v", err)
	}
	if result.Code != 200 {
		bodyPreview := string(body)
		if len(bodyPreview) > 300 {
			bodyPreview = bodyPreview[:300]
		}
		return fmt.Errorf("更新定时失败(%s): %s (code: %d, 响应: %s)", fieldName, result.Message, result.Code, bodyPreview)
	}
	return nil
}

// getCronTaskScheduleField 获取任务详情中的定时规则字段
// 青龙 v2.19.2+ 使用 schedule 字段，旧版使用 cron 字段
func (q *QingLongClient) getCronTaskScheduleField(taskID int) (string, error) {
	token, err := q.GetToken()
	if err != nil {
		return "", fmt.Errorf("获取Token失败: %v", err)
	}

	url := fmt.Sprintf("%s/open/crons/%d", strings.TrimSuffix(q.config.Host, "/"), taskID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %v", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := q.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应失败: %v", err)
	}

	var result struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("解析响应失败: %v", err)
	}
	if result.Code != 200 {
		return "", fmt.Errorf("查询任务失败: %s (code: %d)", result.Message, result.Code)
	}

	// 尝试解析完整的 data 字段（支持 schedule 和 cron 两种字段名）
	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return "", fmt.Errorf("解析任务详情失败: %v", err)
	}

	if d, ok := data["data"].(map[string]interface{}); ok {
		// 优先使用 schedule（青龙 v2.19.2+ 标准字段名）
		if s, ok := d["schedule"].(string); ok && s != "" {
			return s, nil
		}
		// 回退使用 cron（旧版青龙字段名）
		if c, ok := d["cron"].(string); ok && c != "" {
			return c, nil
		}
	}
	return "", fmt.Errorf("任务详情中未找到 schedule 字段 (taskID: %d)", taskID)
}

// EnableCronTask 启用任务
func (q *QingLongClient) EnableCronTask(taskIDs []int) error {
	token, err := q.GetToken()
	if err != nil {
		return fmt.Errorf("获取Token失败: %v", err)
	}

	url := strings.TrimSuffix(q.config.Host, "/") + "/open/crons/enable"
	jsonData, err := json.Marshal(taskIDs)
	if err != nil {
		return fmt.Errorf("序列化数据失败: %v", err)
	}

	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("创建请求失败: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := q.client.Do(req)
	if err != nil {
		return fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取响应失败: %v", err)
	}

	var result QLCommonResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("解析响应失败: %v", err)
	}
	if result.Code != 200 {
		return fmt.Errorf("启用任务失败: %s (code: %d)", result.Message, result.Code)
	}
	return nil
}

// DisableCronTask 禁用任务
func (q *QingLongClient) DisableCronTask(taskIDs []int) error {
	token, err := q.GetToken()
	if err != nil {
		return fmt.Errorf("获取Token失败: %v", err)
	}

	url := strings.TrimSuffix(q.config.Host, "/") + "/open/crons/disable"
	jsonData, err := json.Marshal(taskIDs)
	if err != nil {
		return fmt.Errorf("序列化数据失败: %v", err)
	}

	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("创建请求失败: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := q.client.Do(req)
	if err != nil {
		return fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取响应失败: %v", err)
	}

	var result QLCommonResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("解析响应失败: %v", err)
	}
	if result.Code != 200 {
		return fmt.Errorf("禁用任务失败: %s (code: %d)", result.Message, result.Code)
	}
	return nil
}

// DeleteCronTask 删除任务
func (q *QingLongClient) DeleteCronTask(taskIDs []int) error {
	token, err := q.GetToken()
	if err != nil {
		return fmt.Errorf("获取Token失败: %v", err)
	}

	url := strings.TrimSuffix(q.config.Host, "/") + "/open/crons"
	jsonData, err := json.Marshal(taskIDs)
	if err != nil {
		return fmt.Errorf("序列化数据失败: %v", err)
	}

	req, err := http.NewRequest("DELETE", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("创建请求失败: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := q.client.Do(req)
	if err != nil {
		return fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取响应失败: %v", err)
	}

	var result QLCommonResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("解析响应失败: %v", err)
	}
	if result.Code != 200 {
		return fmt.Errorf("删除任务失败: %s (code: %d)", result.Message, result.Code)
	}
	return nil
}

// RunCronTask 手动运行任务
func (q *QingLongClient) RunCronTask(taskIDs []int) error {
	token, err := q.GetToken()
	if err != nil {
		return fmt.Errorf("获取Token失败: %v", err)
	}

	url := strings.TrimSuffix(q.config.Host, "/") + "/open/crons/run"
	jsonData, err := json.Marshal(taskIDs)
	if err != nil {
		return fmt.Errorf("序列化数据失败: %v", err)
	}

	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("创建请求失败: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := q.client.Do(req)
	if err != nil {
		return fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取响应失败: %v", err)
	}

	var result QLCommonResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("解析响应失败: %v", err)
	}
	if result.Code != 200 {
		return fmt.Errorf("运行任务失败: %s (code: %d)", result.Message, result.Code)
	}
	return nil
}

// StopCronTask 停止正在运行的任务
func (q *QingLongClient) StopCronTask(taskIDs []int) error {
	token, err := q.GetToken()
	if err != nil {
		return fmt.Errorf("获取Token失败: %v", err)
	}

	url := strings.TrimSuffix(q.config.Host, "/") + "/open/crons/stop"
	jsonData, err := json.Marshal(taskIDs)
	if err != nil {
		return fmt.Errorf("序列化数据失败: %v", err)
	}

	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("创建请求失败: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := q.client.Do(req)
	if err != nil {
		return fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取响应失败: %v", err)
	}

	var result QLCommonResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("解析响应失败: %v", err)
	}
	if result.Code != 200 {
		return fmt.Errorf("停止任务失败: %s (code: %d)", result.Message, result.Code)
	}
	return nil
}

// ===================== 青龙日志文件系统 API =====================

// QLLogFileEntry 青龙日志文件条目
type QLLogFileEntry struct {
	Title      string           `json:"title"`
	Key        string           `json:"key"`
	Type       string           `json:"type"` // "directory" 或 "file"
	Parent     string           `json:"parent"`
	CreateTime int64            `json:"createTime"`
	Size       int64            `json:"size,omitempty"`
	Children   []QLLogFileEntry `json:"children,omitempty"`
}

// GetLogFiles 获取青龙日志文件列表（通过文件系统 API）
// 返回树形结构的日志目录
func (q *QingLongClient) GetLogFiles() ([]QLLogFileEntry, error) {
	token, err := q.GetToken()
	if err != nil {
		return nil, fmt.Errorf("获取Token失败: %v", err)
	}

	url := strings.TrimSuffix(q.config.Host, "/") + "/open/logs"

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := q.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	var result struct {
		Code    int              `json:"code"`
		Data    []QLLogFileEntry `json:"data"`
		Message string           `json:"message"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("解析日志列表失败: %v, 响应: %s", err, string(body)[:minInt(len(body), 500)])
	}
	if result.Code != 200 {
		return nil, fmt.Errorf("获取日志列表失败: %s (code: %d)", result.Message, result.Code)
	}
	return result.Data, nil
}

// GetLogFileContent 通过文件系统 API 获取指定日志文件内容
// logPath 是日志路径（如 "/log/123"），fileName 是日志文件名（如 "2024-01-15-10-30-45-000.log"）
func (q *QingLongClient) GetLogFileContent(logPath, fileName string) (string, error) {
	if fileName == "" {
		return "", fmt.Errorf("日志文件名为空")
	}

	token, err := q.GetToken()
	if err != nil {
		return "", fmt.Errorf("获取Token失败: %v", err)
	}

	baseURL := strings.TrimSuffix(q.config.Host, "/")

	// 方式1：GET /open/logs/detail?path=xxx&file=xxx
	url1 := fmt.Sprintf("%s/open/logs/detail?path=%s&file=%s", baseURL, logPath, fileName)
	content, err := q.doGetLogContent(token, url1)
	if err == nil && content != "" {
		return content, nil
	}

	// 方式2：GET /open/logs/{file}?path=xxx
	url2 := fmt.Sprintf("%s/open/logs/%s?path=%s", baseURL, fileName, logPath)
	content2, err2 := q.doGetLogContent(token, url2)
	if err2 == nil && content2 != "" {
		return content2, nil
	}

	return "", fmt.Errorf("获取日志文件内容失败: path=%s file=%s, err: %v | %v", logPath, fileName, err, err2)
}

// GetTaskLogFiles 获取指定任务的历史日志文件列表
// 通过青龙文件系统 API，查找 taskId 对应目录下的所有日志文件
func (q *QingLongClient) GetTaskLogFiles(taskID int) ([]QLLogFileEntry, error) {
	entries, err := q.GetLogFiles()
	if err != nil {
		return nil, err
	}

	taskIDStr := fmt.Sprintf("%d", taskID)

	// 遍历日志目录，找到 taskId 对应的目录
	for _, entry := range entries {
		if entry.Type == "directory" {
			// 检查目录名是否匹配 taskId
			dirName := entry.Key
			if idx := strings.LastIndex(dirName, "/"); idx >= 0 {
				dirName = dirName[idx+1:]
			}
			if dirName == taskIDStr {
				return entry.Children, nil
			}
		}
	}

	return nil, fmt.Errorf("未找到任务 %d 的日志目录", taskID)
}

var (
	sanitizeIPv4Re     = regexp.MustCompile(`\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}(:\d+)?`)
	sanitizeIPv6Re     = regexp.MustCompile(`\[?([0-9a-fA-F]{0,4}:){2,7}[0-9a-fA-F]{0,4}\]?(:\d+)?`)
	sanitizeURLRe      = regexp.MustCompile(`https?://[^\s"]+`)
	sanitizeTokenRe    = regexp.MustCompile(`Bearer\s+[^\s"]+`)
	sanitizeSecretRe   = regexp.MustCompile(`client_secret[=:\s]+[^\s&,"']+`)
	sanitizeClientIDRe = regexp.MustCompile(`client_id[=:\s]+[^\s&,"']+`)
)

func SanitizeError(err error) error {
	if err == nil {
		return nil
	}
	msg := err.Error()
	msg = sanitizeIPv4Re.ReplaceAllString(msg, "[主机地址]")
	msg = sanitizeIPv6Re.ReplaceAllString(msg, "[主机地址]")
	msg = sanitizeURLRe.ReplaceAllString(msg, "[青龙地址]")
	msg = sanitizeTokenRe.ReplaceAllString(msg, "Bearer [已隐藏]")
	msg = sanitizeSecretRe.ReplaceAllString(msg, "client_secret=[已隐藏]")
	msg = sanitizeClientIDRe.ReplaceAllString(msg, "client_id=[已隐藏]")
	return fmt.Errorf("%s", msg)
}
