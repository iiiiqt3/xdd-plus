package models

import (
	"bytes"
	"compress/gzip"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const (
	kuwoAESKey     = "eXNpVmtMSkhIbnZNV0NIcQ=="
	kuwoAESIV      = "aWNoWW9vWCtNYjFnUmV0UA=="
	kuwoUserAgent  = "Mozilla/5.0 (Linux; Android 14; Pixel 8 Pro Build/AP4A.250405.002; wv) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/136.0.7103.60 Mobile Safari/537.36/ kuwopage"
	kuwoCapURL     = "http://www.kuwo.cn/api/common/captcha/getcode"
	kuwoOCRURL     = "https://ddddocr.linzixuan.work/classification"
	kuwoLoginURL   = "https://wapi.kuwo.cn/api/www/login/loginByKw"
	kuwoSmsURL     = "https://integralapi.kuwo.cn/api/v1/online/sign/v1/userBindPhone"
	kuwoWithdrawURL = "https://integralapi.kuwo.cn/api/v1/online/sign/v1/getWithdraw"
)

// KuwoSession holds login credentials for a Kuwo account.
type KuwoSession struct {
	LoginUid      string
	LoginSid      string
	Phone         string
	EncryptedPhone string
}

// KuwoTask holds a withdrawal task configuration.
type KuwoTask struct {
	Session  *KuwoSession
	QuotaID  string
	SmsCode  string
	ResultCh chan KuwoWithdrawResult
}

// KuwoWithdrawRequest holds the parameters for a withdrawal request.
type KuwoWithdrawRequest struct {
	Phone    string
	QuotaID  string
	SmsCode  string
}

// KuwoWithdrawResult is the result of a withdrawal attempt.
type KuwoWithdrawResult struct {
	UID     string
	Phone   string
	Success bool
	Message string
	Error   error
}

// KuwoQuotaMap maps withdrawal amounts to their quota IDs.
var KuwoQuotaMap = map[string]string{
	"1":  "60004",
	"2":  "30002",
	"10": "60001",
}

// ---------- helpers ----------

func kuwoRandomAppUID() string {
	const digits = "0123456789"
	b := make([]byte, 10)
	for i := range b {
		idx, _ := rand.Int(rand.Reader, big.NewInt(int64(len(digits))))
		b[i] = digits[idx.Int64()]
	}
	return string(b)
}

func kuwoMD5(s string) string {
	h := md5.Sum([]byte(s))
	return hex.EncodeToString(h[:])
}

// KuwoEncryptPhone encrypts a phone number with AES-128-CBC.
func KuwoEncryptPhone(phone string) string {
	key, _ := base64.StdEncoding.DecodeString(kuwoAESKey)
	iv, _ := base64.StdEncoding.DecodeString(kuwoAESIV)

	block, err := aes.NewCipher(key)
	if err != nil {
		return ""
	}
	// PKCS7 padding
	padLen := aes.BlockSize - len(phone)%aes.BlockSize
	padded := make([]byte, len(phone)+padLen)
	copy(padded, phone)
	for i := len(phone); i < len(padded); i++ {
		padded[i] = byte(padLen)
	}

	ciphertext := make([]byte, len(padded))
	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(ciphertext, padded)
	return base64.StdEncoding.EncodeToString(ciphertext)
}

func kuwoNewHTTPClient() *http.Client {
	return &http.Client{
		Timeout: 15 * time.Second,
	}
}

func kuwoDoRequest(req *http.Request) ([]byte, int, error) {
	client := kuwoNewHTTPClient()
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	var body []byte
	contentEncoding := resp.Header.Get("Content-Encoding")
	if strings.Contains(contentEncoding, "gzip") {
		reader, gzErr := gzip.NewReader(resp.Body)
		if gzErr != nil {
			body, err = io.ReadAll(resp.Body)
		} else {
			defer reader.Close()
			body, err = io.ReadAll(reader)
		}
	} else {
		body, err = io.ReadAll(resp.Body)
	}
	return body, resp.StatusCode, err
}

func kuwoGetCaptcha() (imgBase64 string, token string, err error) {
	// 1. 获取验证码（返回JSON，含 data.img 和 data.token）
	reqURL := kuwoCapURL + "?reqId=" + kuwoRandomAppUID() + "&httpsStatus=1"
	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return "", "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.6261.95 Safari/537.36")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Referer", "http://www.kuwo.cn/")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9")

	body, _, err := kuwoDoRequest(req)
	if err != nil {
		return "", "", err
	}

	var capResp struct {
		Data struct {
			Img   string `json:"img"`
			Token string `json:"token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &capResp); err != nil {
		return "", "", fmt.Errorf("parse captcha response: %w", err)
	}
	if capResp.Data.Img == "" || capResp.Data.Token == "" {
		return "", "", fmt.Errorf("captcha response missing img or token")
	}

	// 去掉 data:image/jpeg;base64, 前缀
	imgStr := capResp.Data.Img
	if idx := strings.Index(imgStr, ","); idx >= 0 {
		imgStr = imgStr[idx+1:]
	}

	// 2. OCR识别验证码（发送JSON {image: base64}）
	ocrPayload, _ := json.Marshal(map[string]string{"image": imgStr})
	ocrReq, err := http.NewRequest("POST", kuwoOCRURL, bytes.NewReader(ocrPayload))
	if err != nil {
		return "", "", err
	}
	ocrReq.Header.Set("Content-Type", "application/json")
	ocrReq.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	ocrBody, _, err := kuwoDoRequest(ocrReq)
	if err != nil {
		return "", "", err
	}

	var ocrResp struct {
		Result string `json:"result"`
	}
	if err := json.Unmarshal(ocrBody, &ocrResp); err != nil {
		// fallback: treat entire body as text
		return strings.TrimSpace(string(ocrBody)), capResp.Data.Token, nil
	}
	return strings.TrimSpace(ocrResp.Result), capResp.Data.Token, nil
}

// ---------- login ----------

// KuwoLogin performs the full login flow: captcha -> OCR -> login.
func KuwoLogin(phone, password string) (*KuwoSession, error) {
	encPhone := KuwoEncryptPhone(phone)

	for attempt := 0; attempt < 5; attempt++ {
		// 1. 获取验证码 + token
		captchaCode, captchaToken, err := kuwoGetCaptcha()
		if err != nil {
			return nil, fmt.Errorf("get captcha failed: %w", err)
		}
		fmt.Printf("[kuwo] captcha attempt %d: code=%s token=%s\n", attempt+1, captchaCode, captchaToken)

		// 2. 登录（JSON body）
		loginBody, _ := json.Marshal(map[string]string{
			"userIp":          "www.kuwo.cn",
			"uname":           phone,
			"password":        password,
			"verifyCode":      captchaCode,
			"img":             "", // 不需要回传图片
			"verifyCodeToken": captchaToken,
		})

		req, err := http.NewRequest("POST", kuwoLoginURL+"?httpsStatus=1", bytes.NewReader(loginBody))
		if err != nil {
			return nil, err
		}
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.6261.95 Safari/537.36")
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Origin", "http://www.kuwo.cn")
		req.Header.Set("Referer", "http://www.kuwo.cn/")
		req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9")

		body, _, err := kuwoDoRequest(req)
		if err != nil {
			return nil, err
		}

		var resp struct {
			Code int    `json:"code"`
			Msg  string `json:"msg"`
			Data struct {
				Cookies struct {
					Websid interface{} `json:"websid"`
					Userid interface{} `json:"userid"`
				} `json:"cookies"`
			} `json:"data"`
		}
		if err := json.Unmarshal(body, &resp); err != nil {
			return nil, fmt.Errorf("parse login response: %w", err)
		}

		fmt.Printf("[kuwo] login response: code=%d msg=%s\n", resp.Code, resp.Msg)

		uid := fmt.Sprintf("%.0f", resp.Data.Cookies.Userid)
		sid := fmt.Sprintf("%v", resp.Data.Cookies.Websid)
		if resp.Code == 200 && uid != "" && uid != "<nil>" && sid != "" && sid != "<nil>" {
			return &KuwoSession{
				LoginUid:       uid,
				LoginSid:       sid,
				Phone:          phone,
				EncryptedPhone: encPhone,
			}, nil
		}

		// captcha识别错误，重试
		fmt.Printf("[kuwo] login attempt %d failed: %s\n", attempt+1, resp.Msg)
	}

	return nil, fmt.Errorf("login failed after 5 attempts")
}

// ---------- SMS ----------

// KuwoSendSms sends an SMS verification code for the logged-in session.
func KuwoSendSms(session *KuwoSession) error {
	params := url.Values{}
	params.Set("loginUid", session.LoginUid)
	params.Set("loginSid", session.LoginSid)
	params.Set("mobile", session.EncryptedPhone)

	reqURL := kuwoSmsURL + "?" + params.Encode()
	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", kuwoUserAgent)
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Origin", "https://h5app.kuwo.cn")
	req.Header.Set("Referer", "https://h5app.kuwo.cn/apps/earning-sign/cash_out.html")

	body, _, err := kuwoDoRequest(req)
	if err != nil {
		return err
	}

	var resp struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("parse sms response: %w", err)
	}
	if resp.Code != 200 {
		return fmt.Errorf("send sms failed: %s", resp.Msg)
	}
	return nil
}

// ---------- withdraw ----------

// KuwoExecuteWithdraw executes a single withdrawal request.
func KuwoExecuteWithdraw(session *KuwoSession, quotaId, smsCode string) (string, error) {
	params := url.Values{}
	params.Set("encry", "")
	params.Set("type", "")
	params.Set("quotaId", quotaId)
	params.Set("loginUid", session.LoginUid)
	params.Set("loginSid", session.LoginSid)
	params.Set("appuid", kuwoRandomAppUID())
	params.Set("source", "kwplayer_ar_12.1.4.0_40.apk")
	params.Set("version", "1")
	params.Set("phone", session.EncryptedPhone)
	params.Set("code", smsCode)

	reqURL := kuwoWithdrawURL + "?" + params.Encode()
	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", kuwoUserAgent)
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Origin", "https://h5app.kuwo.cn")
	req.Header.Set("Referer", "https://h5app.kuwo.cn/apps/earning-sign/cash_out.html")

	body, _, err := kuwoDoRequest(req)
	if err != nil {
		return "", err
	}

	var resp struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			Text string `json:"text"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", fmt.Errorf("parse withdraw response: %w", err)
	}

	text := resp.Data.Text
	if text == "" {
		text = resp.Msg
	}

	if strings.Contains(text, "提现成功") || strings.Contains(text, "提现申请发起成功") {
		return text, nil
	}
	return text, fmt.Errorf("withdraw failed: %s", text)
}

// ---------- concurrent ----------

// KuwoConcurrentWithdraw performs withdrawals for multiple sessions concurrently.
func KuwoConcurrentWithdraw(sessions []*KuwoSession, quotaId, smsCode string) []KuwoWithdrawResult {
	results := make([]KuwoWithdrawResult, len(sessions))
	var wg sync.WaitGroup

	for i, sess := range sessions {
		wg.Add(1)
		go func(idx int, s *KuwoSession) {
			defer wg.Done()
			msg, err := KuwoExecuteWithdraw(s, quotaId, smsCode)
			results[idx] = KuwoWithdrawResult{
				UID:     s.LoginUid,
				Phone:   s.Phone,
				Success: err == nil,
				Message: msg,
				Error:   err,
			}
			if err != nil {
				fmt.Printf("[kuwo] withdraw uid=%s failed: %v\n", s.LoginUid, err)
			} else {
				fmt.Printf("[kuwo] withdraw uid=%s success: %s\n", s.LoginUid, msg)
			}
		}(i, sess)
	}

	wg.Wait()
	return results
}

// ---------- convenience ----------

// KuwoGetQuotaID returns the quota ID for the given amount string ("1","3","5","10","30").
func KuwoGetQuotaID(amount string) (string, bool) {
	id, ok := KuwoQuotaMap[amount]
	return id, ok
}

// KuwoFullWithdrawFlow is a convenience function that:
// 1. Logs in
// 2. Sends SMS
// 3. Executes withdrawal for the given amount
//
// The caller must supply the SMS code via callback after receiving the code.
func KuwoFullWithdrawFlow(phone, password, smsCode, amount string) ([]KuwoWithdrawResult, error) {
	quotaId, ok := KuwoGetQuotaID(amount)
	if !ok {
		return nil, fmt.Errorf("invalid amount: %s, supported: 1,3,5,10,30", amount)
	}

	session, err := KuwoLogin(phone, password)
	if err != nil {
		return nil, fmt.Errorf("login failed: %w", err)
	}

	if err := KuwoSendSms(session); err != nil {
		return nil, fmt.Errorf("send sms failed: %w", err)
	}

	// Wait for SMS code
	fmt.Printf("[kuwo] SMS sent, waiting for code...\n")
	time.Sleep(3 * time.Second)

	results := KuwoConcurrentWithdraw([]*KuwoSession{session}, quotaId, smsCode)
	return results, nil
}

// KuwoFormatQuotaDisplay returns a human-readable list of available quotas.
func KuwoFormatQuotaDisplay() string {
	var parts []string
	for amount, id := range KuwoQuotaMap {
		parts = append(parts, fmt.Sprintf("%s元(id=%s)", amount, id))
	}
	return strings.Join(parts, ", ")
}

// GetKuwoCredentials 从用户的KWYY活动项目中读取酷我账号密码
func GetKuwoCredentials(userNumber int) (phone, password string, err error) {
	projects, err := GetPortalProjects(userNumber)
	if err != nil {
		return "", "", err
	}
	for _, p := range projects {
		if p.ActivityID == "KWYY" && p.EnvValue != "" {
			parts := strings.SplitN(p.EnvValue, "#", 2)
			if len(parts) == 2 {
				return parts[0], parts[1], nil
			}
			return p.EnvValue, "", nil
		}
	}
	return "", "", fmt.Errorf("未找到酷我音乐活动配置")
}

// CheckKuwoAuth 检查当前用户是否有酷我音乐(KWYY)活动授权
func CheckKuwoAuth(userNumber int) (bool, string) {
	projects, err := GetPortalProjects(userNumber)
	if err != nil || len(projects) == 0 {
		return false, "您还没有上车任何活动，请先前往「项目中心」上车酷我音乐活动"
	}

	for _, p := range projects {
		if p.ActivityID == "KWYY" {
			if p.BizStatus == "expired" {
				return false, "您的酷我音乐活动授权已到期，请前往「项目中心」续费"
			}
			return true, ""
		}
	}
	return false, "您还没有上车酷我音乐活动，请先前往「项目中心」上车"
}

func init() {
	fmt.Printf("[kuwo] module loaded, available quotas: %s\n", KuwoFormatQuotaDisplay())
}
