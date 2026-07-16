package models

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/beego/beego/v2/client/httplib"
	"gopkg.in/yaml.v2"
)

// GetReplyContent 根据消息内容匹配回复
func GetReplyContent(msg string) (interface{}, bool) {
	filePath := ExecPath + "/conf/replies.yaml"

	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return getDefaultReply(msg)
	}

	rules := make(map[string]string)
	if err := yaml.Unmarshal(data, &rules); err != nil {
		Error("解析 replies.yaml 失败:", err)
		return "", false
	}

	for pattern, reply := range rules {
		if pattern == "" {
			continue
		}

		// 精准匹配逻辑
		finalPattern := pattern
		isRegex := strings.ContainsAny(pattern, "^$.*+?[]()|\\")
		if !isRegex {
			finalPattern = "^" + regexp.QuoteMeta(pattern) + "$"
		}

		if matched, _ := regexp.MatchString(finalPattern, msg); matched {
			// 特殊逻辑：如果消息包含"妹"，强制覆盖回复内容
			if strings.Contains(msg, "妹") {
				reply = "https://pics4.baidu.com/feed/d833c895d143ad4bfee5f874cfdcbfa9a60f069b.jpeg?token=8a8a0e1e20d4626cd31c0b838d9e4c1a"
			}

			// ========== 核心修复：增加中文检测逻辑 ==========
			// 定义中文正则（匹配任意中文字符）
			chineseRegex := regexp.MustCompile(`[\p{Han}]`)
			// 如果回复内容包含中文，直接原样返回
			if chineseRegex.MatchString(reply) {
				return reply, true
			}

			// 1. 无中文时，判断是否是纯URL（作为图片API处理）
			urlRegex := regexp.MustCompile(`^https{0,1}://[^\x{4e00}-\x{9fa5}\n\r\s]{3,}$`)
			if urlRegex.MatchString(reply) {
				realUrl := fetchRealImageUrl(reply)
				if realUrl != "" {
					Info("API 返回真实图片地址:", realUrl)
					return fmt.Sprintf("[CQ:image,file=%s]", realUrl), true
				} else {
					Warn("API 未返回有效图片地址，尝试作为文本处理")
					return fetchUrlContent(reply)
				}
			}

			// 2. 无中文但包含URL（仅替换URL为图片格式）
			if strings.Contains(reply, "http://") || strings.Contains(reply, "https://") {
				re := regexp.MustCompile(`https?://[^\s"<>]+`)
				match := re.FindString(reply)
				if match != "" {
					realUrl := fetchRealImageUrl(match)
					if realUrl != "" {
						newReply := strings.Replace(reply, match, fmt.Sprintf("[CQ:image,file=%s]", realUrl), 1)
						return newReply, true
					}
				}
			}

			// 3. 普通文本回复（无中文无URL）
			return reply, true
		}
	}

	return getDefaultReply(msg)
}

func fetchRealImageUrl(apiUrl string) string {
	client := &http.Client{
		Timeout: time.Second * 8,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("stopped after 10 redirects")
			}
			return nil
		},
	}

	req, err := http.NewRequest("GET", apiUrl, nil)
	if err != nil {
		return ""
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8")
	req.Header.Set("Referer", apiUrl)

	resp, err := client.Do(req)
	if err != nil {
		Debug("请求图片 API 失败:", err)
		return ""
	}
	defer resp.Body.Close()

	contentType := resp.Header.Get("Content-Type")
	finalUrl := resp.Request.URL.String()

	// 情况 1: 直接返回图片二进制
	if strings.HasPrefix(contentType, "image/") {
		Debug("[API] 直接返回图片流:", finalUrl)
		return finalUrl
	}

	// 读取响应体
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return ""
	}
	content := strings.TrimSpace(string(body))

	// 情况 2: 返回的是 JSON
	if strings.Contains(contentType, "application/json") || strings.HasPrefix(content, "{") {
		re := regexp.MustCompile(`"(?:imgurl|url|pic|image|data|img)"\s*:\s*"([^"]+)"`)
		matches := re.FindStringSubmatch(content)
		if len(matches) > 1 {
			Debug("[API] JSON 模式，提取到 URL:", matches[1])
			return matches[1]
		}
	}

	// 情况 3: 返回的是 HTML
	if strings.Contains(contentType, "text/html") || strings.Contains(content, "<img") {
		imgRegex := regexp.MustCompile(`<img[^>]+src=["']([^"'>]+)["'][^>]*>`)
		matches := imgRegex.FindAllStringSubmatch(content, -1)

		for _, match := range matches {
			if len(match) > 1 {
				imgSrc := match[1]
				if strings.HasPrefix(imgSrc, "http") &&
					(strings.Contains(imgSrc, ".jpg") || strings.Contains(imgSrc, ".png") ||
						strings.Contains(imgSrc, ".jpeg") || strings.Contains(imgSrc, ".gif") ||
						strings.Contains(imgSrc, "inews.gtimg.com") ||
						strings.Contains(imgSrc, "/newsapp_")) {

					Debug("[API] HTML 模式，从 <img> 标签提取到 URL:", imgSrc)
					return imgSrc
				}
			}
		}

		// 宽松匹配
		for _, match := range matches {
			if len(match) > 1 {
				imgSrc := match[1]
				if strings.HasPrefix(imgSrc, "http") && !strings.Contains(imgSrc, "icon") && !strings.Contains(imgSrc, "logo") {
					Debug("[API] HTML 模式 (宽松)，提取到 URL:", imgSrc)
					return imgSrc
				}
			}
		}

		Warn("[API] HTML 模式，但未找到合适的 <img> 标签")
	}

	// 情况 4: 返回的是纯文本 (URL)
	if strings.HasPrefix(content, "http://") || strings.HasPrefix(content, "https://") {
		if !strings.Contains(content, "<") && !strings.Contains(content, ">") {
			Debug("[API] 纯文本模式，返回 URL:", content)
			return content
		}
	}

	// 情况 5: 重定向且最终 URL 像图片
	if finalUrl != apiUrl && hasImageExtension(finalUrl) {
		Debug("[API] 重定向模式，使用最终 URL:", finalUrl)
		return finalUrl
	}

	return ""
}

// hasImageExtension 检查后缀
func hasImageExtension(urlStr string) bool {
	lower := strings.ToLower(urlStr)
	exts := []string{".jpg", ".jpeg", ".png", ".gif", ".bmp", ".webp", ".php", ".asp", ".jsp"}
	for _, ext := range exts {
		if strings.Contains(lower, ext) {
			return true
		}
	}
	return false
}

// fetchUrlContent 抓取非图片 URL 的内容 (作为兜底)
func fetchUrlContent(urlStr string) (string, bool) {
	rsp, err := httplib.Get(urlStr).Response()
	if err != nil {
		Error("请求回复 URL 失败:", urlStr, err)
		return "", false
	}
	defer rsp.Body.Close()

	ctp := rsp.Header.Get("content-type")
	if ctp == "" {
		ctp = rsp.Header.Get("Content-Type")
	}

	if strings.Contains(ctp, "text") || strings.Contains(ctp, "json") {
		data, err := ioutil.ReadAll(rsp.Body)
		if err != nil {
			return "", false
		}
		return string(data), true
	}

	// 不知道类型时返回 URL 本身
	return urlStr, true
}

// getDefaultReply 默认兜底
func getDefaultReply(msg string) (interface{}, bool) {
	if strings.Contains(msg, "壁纸") {
		url := "https://acg.toubiec.cn/random.php"
		realUrl := fetchRealImageUrl(url)
		if realUrl != "" {
			return fmt.Sprintf("[CQ:image,file=%s]", realUrl), true
		}
		return url, true
	}
	return "", false
}
