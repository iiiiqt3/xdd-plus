package models

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	SignatureSecret     = "G0uD0ng@2024#S1gn@ture!K3y" // 签名密钥
	SignatureExpireSec  = 30                             // 签名有效期（秒）
	SignatureVersion    = "v2"                           // 签名版本
	SignatureSalt       = "x9D$kL2mN#pQ7rT5"            // 盐值
	SignatureAppID      = "goudong"                      // 应用ID
	SignatureNonceLen   = 16                             // 随机数长度
)

// SignatureHeaders 签名请求头
type SignatureHeaders struct {
	Timestamp  string `header:"X-Sign-Timestamp"`
	Nonce      string `header:"X-Sign-Nonce"`
	DeviceID   string `header:"X-Sign-DeviceID"`
	Signature  string `header:"X-Sign-Value"`
	Version    string `header:"X-Sign-Version"`
	AppVersion string `header:"X-App-Version"`
}

// VerifyResult 验证结果
type VerifyResult struct {
	Valid   bool
	Message string
}

// GenerateSignature 生成签名
// 签名算法：
// 1. 收集参数：timestamp + nonce + deviceId + appVersion + 请求路径
// 2. 参数排序拼接
// 3. 第一轮：SHA-256(拼接字符串 + salt)
// 4. 第二轮：HMAC-SHA256(第一轮结果, secret)
// 5. 第三轮：SHA-256(第二轮结果 + timestamp反转 + nonce反转)
func GenerateSignature(timestamp, nonce, deviceID, appVersion, path string) string {
	// 第一轮：SHA-256(拼接字符串 + salt)
	round1Input := fmt.Sprintf("%s|%s|%s|%s|%s|%s",
		timestamp, nonce, deviceID, appVersion, path, SignatureSalt)
	round1 := sha256Hex(round1Input)

	// 第二轮：HMAC-SHA256(第一轮结果, secret)
	round2 := hmacSha256Hex(round1, SignatureSecret)

	// 第三轮：SHA-256(第二轮结果 + timestamp反转 + nonce反转)
	reversedTimestamp := reverseString(timestamp)
	reversedNonce := reverseString(nonce)
	round3Input := fmt.Sprintf("%s%s%s", round2, reversedTimestamp, reversedNonce)
	round3 := sha256Hex(round3Input)

	return round3
}

// VerifySignature 验证签名
func VerifySignature(timestamp, nonce, deviceID, appVersion, path, signature string) VerifyResult {
	// 检查参数是否为空
	if timestamp == "" || nonce == "" || signature == "" {
		return VerifyResult{Valid: false, Message: "签名参数不完整，请升级最新版APP"}
	}

	// 检查时间戳格式
	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return VerifyResult{Valid: false, Message: "签名参数错误，请升级最新版APP"}
	}

	// 检查签名有效期
	now := time.Now().Unix()
	if abs(now-ts) > SignatureExpireSec {
		return VerifyResult{Valid: false, Message: "签名已过期，请升级最新版APP"}
	}

	// 检查随机数长度
	if len(nonce) < 8 {
		return VerifyResult{Valid: false, Message: "签名参数错误，请升级最新版APP"}
	}

	// 生成期望的签名
	expectedSignature := GenerateSignature(timestamp, nonce, deviceID, appVersion, path)

	// 使用常量时间比较，防止时序攻击
	if !hmac.Equal([]byte(signature), []byte(expectedSignature)) {
		return VerifyResult{Valid: false, Message: "签名验证失败，请升级最新版APP执行签到和打卡"}
	}

	return VerifyResult{Valid: true, Message: "验证通过"}
}

// GenerateRequestParams 生成请求参数（供客户端使用）
func GenerateRequestParams(deviceID, appVersion, path string) map[string]string {
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	nonce := generateNonce(SignatureNonceLen)
	signature := GenerateSignature(timestamp, nonce, deviceID, appVersion, path)

	return map[string]string{
		"timestamp":  timestamp,
		"nonce":      nonce,
		"device_id":  deviceID,
		"app_version": appVersion,
		"signature":  signature,
		"version":    SignatureVersion,
	}
}

// 辅助函数

func sha256Hex(input string) string {
	h := sha256.New()
	h.Write([]byte(input))
	return hex.EncodeToString(h.Sum(nil))
}

func hmacSha256Hex(message, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(message))
	return hex.EncodeToString(h.Sum(nil))
}

func reverseString(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

func generateNonce(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[time.Now().UnixNano()%int64(len(charset))]
		time.Sleep(1) // 确保每次随机数不同
	}
	return string(result)
}

func abs(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}

// ExtractSignatureFromHeaders 从请求头中提取签名信息
func ExtractSignatureFromHeaders(headers map[string]string) SignatureHeaders {
	return SignatureHeaders{
		Timestamp:  headers["X-Sign-Timestamp"],
		Nonce:      headers["X-Sign-Nonce"],
		DeviceID:   headers["X-Sign-DeviceID"],
		Signature:  headers["X-Sign-Value"],
		Version:    headers["X-Sign-Version"],
		AppVersion: headers["X-App-Version"],
	}
}

// VerifyRequest 验证请求签名
func VerifyRequest(headers map[string]string, path string) VerifyResult {
	sig := ExtractSignatureFromHeaders(headers)

	// 检查签名版本
	if sig.Version != SignatureVersion {
		return VerifyResult{Valid: false, Message: "签名版本不匹配，请升级最新版APP执行签到和打卡"}
	}

	return VerifySignature(
		sig.Timestamp,
		sig.Nonce,
		sig.DeviceID,
		sig.AppVersion,
		path,
		sig.Signature,
	)
}

// GetSignatureInfo 获取签名信息（用于调试）
func GetSignatureInfo() map[string]interface{} {
	return map[string]interface{}{
		"version":      SignatureVersion,
		"expire_sec":   SignatureExpireSec,
		"app_id":       SignatureAppID,
		"nonce_length": SignatureNonceLen,
	}
}

// GenerateClientCode 生成客户端代码示例
func GenerateClientCode(platform string) string {
	var code string

	switch strings.ToLower(platform) {
	case "android":
		code = generateAndroidCode()
	case "ios":
		code = generateIOSCode()
	default:
		code = "Unsupported platform"
	}

	return code
}

func generateAndroidCode() string {
	return `// Android 签名生成代码
import java.security.MessageDigest
import javax.crypto.Mac
import javax.crypto.spec.SecretKeySpec

object SignatureHelper {
    private const val SECRET = "${SignatureSecret}"
    private const val SALT = "${SignatureSalt}"
    private const val VERSION = "${SignatureVersion}"
    
    fun generateSignature(
        timestamp: String,
        nonce: String,
        deviceId: String,
        appVersion: String,
        path: String
    ): String {
        // 第一轮：SHA-256
        val round1Input = "$timestamp|$nonce|$deviceId|$appVersion|$path|$SALT"
        val round1 = sha256(round1Input)
        
        // 第二轮：HMAC-SHA256
        val round2 = hmacSha256(round1, SECRET)
        
        // 第三轮：SHA-256
        val reversedTimestamp = timestamp.reversed()
        val reversedNonce = nonce.reversed()
        val round3Input = "$round2$reversedTimestamp$reversedNonce"
        return sha256(round3Input)
    }
    
    private fun sha256(input: String): String {
        val digest = MessageDigest.getInstance("SHA-256")
        val hash = digest.digest(input.toByteArray())
        return hash.joinToString("") { "%02x".format(it) }
    }
    
    private fun hmacSha256(message: String, secret: String): String {
        val mac = Mac.getInstance("HmacSHA256")
        val secretKey = SecretKeySpec(secret.toByteArray(), "HmacSHA256")
        mac.init(secretKey)
        val hash = mac.doFinal(message.toByteArray())
        return hash.joinToString("") { "%02x".format(it) }
    }
}`
}

func generateIOSCode() string {
	return `// iOS 签名生成代码
import CryptoKit
import Foundation

struct SignatureHelper {
    static let secret = "${SignatureSecret}"
    static let salt = "${SignatureSalt}"
    static let version = "${SignatureVersion}"
    
    static func generateSignature(
        timestamp: String,
        nonce: String,
        deviceId: String,
        appVersion: String,
        path: String
    ) -> String {
        // 第一轮：SHA-256
        let round1Input = "\\(timestamp)|\\(nonce)|\\(deviceId)|\\(appVersion)|\\(path)|\\(salt)"
        let round1 = sha256(round1Input)
        
        // 第二轮：HMAC-SHA256
        let round2 = hmacSha256(message: round1, secret: secret)
        
        // 第三轮：SHA-256
        let reversedTimestamp = String(timestamp.reversed())
        let reversedNonce = String(nonce.reversed())
        let round3Input = "\\(round2)\\(reversedTimestamp)\\(reversedNonce)"
        return sha256(round3Input)
    }
    
    private static func sha256(_ input: String) -> String {
        let data = Data(input.utf8)
        let hash = SHA256.hash(data: data)
        return hash.map { String(format: "%02x", $0) }.joined()
    }
    
    private static func hmacSha256(message: String, secret: String) -> String {
        let key = SymmetricKey(data: Data(secret.utf8))
        let data = Data(message.utf8)
        let signature = HMAC<SHA256>.authenticationCode(for: data, using: key)
        return signature.map { String(format: "%02x", $0) }.joined()
    }
}`
}
