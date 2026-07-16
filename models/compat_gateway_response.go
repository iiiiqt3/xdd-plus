package models

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
)

// BuildCompatUnsupportedResponse 应用宝协议层不支持该路径时的明确错误
func BuildCompatUnsupportedResponse(path, hint string) []byte {
	msg := "应用宝协议不支持该接口：" + strings.TrimSpace(path)
	if strings.TrimSpace(hint) != "" {
		msg += "（" + strings.TrimSpace(hint) + "）"
	}
	return BuildCompatErrorResponse(msg)
}

// WrapCompatWxappResponse 将应用宝 operate/getCode 等原始结果包装为旧脚本信封
func WrapCompatWxappResponse(apiName string, raw map[string]interface{}) []byte {
	if raw == nil {
		return BuildCompatErrorResponse("应用宝未返回数据")
	}
	if msg := compatBackendResponseError(raw); msg != "" {
		return BuildCompatErrorResponse(msg)
	}
	result := raw["result"]
	if result == nil {
		if data, ok := raw["data"].(map[string]interface{}); ok {
			result = data["result"]
		}
	}
	if result == nil {
		result = raw
	}
	if m, ok := result.(map[string]interface{}); ok {
		if code, ok := m["code"]; ok && apiName == "" {
			// getCode 结果常是 {"code":"xxx"}
			result = map[string]interface{}{"code": code}
		}
	}
	result = NormalizeCompatWxappResult(apiName, result)
	return BuildCompatSuccessResponse(result)
}

func compatBackendResponseError(raw map[string]interface{}) string {
	if raw == nil {
		return ""
	}
	if compatNumericNonZero(raw["code"]) || compatNumericNonZero(raw["Code"]) {
		msg := compatFirstNonEmpty(raw, "message", "Message", "msg", "Msg")
		if msg == "" {
			b, _ := json.Marshal(raw)
			msg = string(b)
		}
		return msg
	}
	for _, key := range []string{"success", "Success", "status"} {
		if v, ok := raw[key].(bool); ok && !v {
			msg := compatFirstNonEmpty(raw, "message", "Message", "msg", "Msg")
			if msg == "" {
				b, _ := json.Marshal(raw)
				msg = string(b)
			}
			return msg
		}
	}
	return ""
}

// NormalizeCompatWxappResult 旧脚本兼容：webapi_getuserencryptkey 需 base64 包装
func NormalizeCompatWxappResult(apiName string, result interface{}) interface{} {
	if apiName != "webapi_getuserencryptkey" {
		return result
	}
	m, ok := result.(map[string]interface{})
	if !ok {
		return result
	}
	rawData, ok := m["data"].(string)
	if !ok || strings.TrimSpace(rawData) == "" || !strings.HasPrefix(strings.TrimSpace(rawData), "{") {
		return result
	}
	var keyObj map[string]interface{}
	if json.Unmarshal([]byte(rawData), &keyObj) != nil {
		return result
	}
	if compatFirstNonEmpty(keyObj, "encrypt_key", "encryptKey") == "" {
		return result
	}
	oldOuter := map[string]interface{}{
		"data":   rawData,
		"err_no": m["err_no"],
	}
	oldOuterBytes, _ := json.Marshal(oldOuter)
	m["data"] = base64.StdEncoding.EncodeToString(oldOuterBytes)
	m["jsapiBaseresponse"] = map[string]interface{}{"errcode": 0, "errmsg": "ok"}
	return m
}

// BuildCompatEncryptKeyResponse 从 operate 结果提取 encryptKey 字段（getLatestUserKey）
func BuildCompatEncryptKeyResponse(raw map[string]interface{}) []byte {
	if msg := compatBackendResponseError(raw); msg != "" {
		return BuildCompatErrorResponse(msg)
	}
	material := extractCompatEncryptKeyMaterial(raw)
	if compatFirstNonEmpty(material, "encryptKey", "encrypt_key", "key") == "" {
		return BuildCompatErrorResponse("应用宝未返回 encryptKey")
	}
	return BuildCompatSuccessResponse(material)
}

func extractCompatEncryptKeyMaterial(v interface{}) map[string]interface{} {
	out := map[string]interface{}{}
	var walk func(interface{}) bool
	walk = func(node interface{}) bool {
		switch x := node.(type) {
		case map[string]interface{}:
			if raw := compatFirstNonEmpty(x, "data", "Data"); raw != "" {
				trimmed := strings.TrimSpace(raw)
				if strings.HasPrefix(trimmed, "{") {
					var inner map[string]interface{}
					if json.Unmarshal([]byte(trimmed), &inner) == nil && walk(inner) {
						return true
					}
				} else if decoded, err := base64.StdEncoding.DecodeString(trimmed); err == nil {
					var inner map[string]interface{}
					if json.Unmarshal(decoded, &inner) == nil && walk(inner) {
						return true
					}
				}
			}
			key := compatFirstNonEmpty(x, "encryptKey", "encrypt_key", "key", "EncryptKey")
			if key != "" {
				out["encryptKey"] = key
				out["encrypt_key"] = key
				out["key"] = key
				out["version"] = fmt.Sprint(compatFirstNonNil(x, "version", "Version", "encryptVer"))
				out["iv"] = compatFirstNonEmpty(x, "iv", "IV")
				out["expireTime"] = compatFirstNonNil(x, "expireTime", "expire_time", "expire_in", "expireIn")
				out["raw"] = x
				return true
			}
			for _, child := range x {
				if walk(child) {
					return true
				}
			}
		case []interface{}:
			for _, child := range x {
				if walk(child) {
					return true
				}
			}
		}
		return false
	}
	walk(v)
	return out
}

// ExtractCompatAPINameFromPayload 从请求体提取 api_name
func ExtractCompatAPINameFromPayload(payload map[string]interface{}) string {
	if payload == nil {
		return ""
	}
	if p, ok := payload["payload"].(map[string]interface{}); ok {
		return compatFirstNonEmpty(p, "api_name")
	}
	if p, ok := payload["Payload"].(map[string]interface{}); ok {
		return compatFirstNonEmpty(p, "api_name")
	}
	for _, key := range []string{"data", "Data"} {
		if raw, ok := payload[key].(string); ok && strings.TrimSpace(raw) != "" {
			var inner map[string]interface{}
			if json.Unmarshal([]byte(raw), &inner) == nil {
				return compatFirstNonEmpty(inner, "api_name")
			}
		}
		if p, ok := payload[key].(map[string]interface{}); ok {
			return compatFirstNonEmpty(p, "api_name")
		}
	}
	return compatFirstNonEmpty(payload, "api_name")
}

func compatFirstNonEmpty(m map[string]interface{}, keys ...string) string {
	for _, k := range keys {
		if v, ok := m[k]; ok {
			if s := strings.TrimSpace(fmt.Sprint(v)); s != "" && s != "<nil>" {
				return s
			}
		}
	}
	return ""
}

func compatFirstNonNil(m map[string]interface{}, keys ...string) interface{} {
	for _, k := range keys {
		if v, ok := m[k]; ok && v != nil {
			if s, ok := v.(string); !ok || strings.TrimSpace(s) != "" {
				return v
			}
		}
	}
	return ""
}

func compatNumericNonZero(v interface{}) bool {
	switch x := v.(type) {
	case int:
		return x != 0
	case int64:
		return x != 0
	case float64:
		return x != 0
	case json.Number:
		n, _ := x.Int64()
		return n != 0
	case string:
		return strings.TrimSpace(x) != "" && strings.TrimSpace(x) != "0"
	default:
		return false
	}
}
