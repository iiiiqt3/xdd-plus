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

func kuwoGetCaptcha() (string, error) {
	req, err := http.NewRequest("GET", kuwoCapURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", kuwoUserAgent)
	req.Header.Set("Referer", "http://www.kuwo.cn/")

	body, _, err := kuwoDoRequest(req)
	if err != nil {
		return "", err
	}

	// body is the captcha image; send to OCR service
	ocrReq, err := http.NewRequest("POST", kuwoOCRURL, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	ocrReq.Header.Set("Content-Type", "image/png")
	ocrReq.Header.Set("User-Agent", kuwoUserAgent)

	ocrBody, _, err := kuwoDoRequest(ocrReq)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(ocrBody)), nil
}

// ---------- login ----------

// KuwoLogin performs the full login flow: captcha -> OCR -> login.
// phone is the plain phone number; password is the Kuwo password.
func KuwoLogin(phone, password string) (*KuwoSession, error) {
	encPhone := KuwoEncryptPhone(phone)

	for attempt := 0; attempt < 5; attempt++ {
		captcha, err := kuwoGetCaptcha()
		if err != nil {
			return nil, fmt.Errorf("get captcha failed: %w", err)
		}

		data := url.Values{}
		data.Set("phone", encPhone)
		data.Set("password", kuwoMD5(password))
		data.Set("code", captcha)

		req, err := http.NewRequest("POST", kuwoLoginURL, strings.NewReader(data.Encode()))
		if err != nil {
			return nil, err
		}
		req.Header.Set("User-Agent", kuwoUserAgent)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Referer", "http://www.kuwo.cn/")

		body, _, err := kuwoDoRequest(req)
		if err != nil {
			return nil, err
		}

		var resp struct {
			Status int    `json:"status"`
			Msg    string `json:"msg"`
			Data   struct {
				LoginUID string `json:"loginUid"`
				LoginSID string `json:"loginSid"`
			} `json:"data"`
		}
		if err := json.Unmarshal(body, &resp); err != nil {
			return nil, fmt.Errorf("parse login response: %w", err)
		}

		if resp.Status == 200 && resp.Data.LoginUID != "" {
			return &KuwoSession{
				LoginUid:       resp.Data.LoginUID,
				LoginSid:       resp.Data.LoginSID,
				Phone:          phone,
				EncryptedPhone: encPhone,
			}, nil
		}

		// captcha likely wrong, retry
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

	req, err := http.NewRequest("POST", kuwoSmsURL, strings.NewReader(params.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", kuwoUserAgent)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Referer", "https://www.kuwo.cn/")

	body, _, err := kuwoDoRequest(req)
	if err != nil {
		return err
	}

	var resp struct {
		Status int    `json:"status"`
		Msg    string `json:"msg"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("parse sms response: %w", err)
	}
	if resp.Status != 200 {
		return fmt.Errorf("send sms failed: %s", resp.Msg)
	}
	return nil
}

// ---------- withdraw ----------

// KuwoExecuteWithdraw executes a single withdrawal request.
func KuwoExecuteWithdraw(session *KuwoSession, quotaId, smsCode string) (string, error) {
	params := url.Values{}
	params.Set("loginUid", session.LoginUid)
	params.Set("loginSid", session.LoginSid)
	params.Set("phone", session.EncryptedPhone)
	params.Set("quotaId", quotaId)
	params.Set("code", smsCode)
	params.Set("appuid", kuwoRandomAppUID())

	req, err := http.NewRequest("POST", kuwoWithdrawURL, strings.NewReader(params.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", kuwoUserAgent)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Referer", "https://www.kuwo.cn/")

	body, _, err := kuwoDoRequest(req)
	if err != nil {
		return "", err
	}

	var resp struct {
		Status int    `json:"status"`
		Msg    string `json:"msg"`
		Data   struct {
			Msg string `json:"msg"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", fmt.Errorf("parse withdraw response: %w", err)
	}

	msg := resp.Msg
	if resp.Data.Msg != "" {
		msg = resp.Data.Msg
	}

	if resp.Status == 200 {
		return msg, nil
	}
	return msg, fmt.Errorf("withdraw failed: %s", msg)
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

func init() {
	fmt.Printf("[kuwo] module loaded, available quotas: %s\n", KuwoFormatQuotaDisplay())
}
