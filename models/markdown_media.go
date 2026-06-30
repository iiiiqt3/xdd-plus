package models

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

)

var markdownImageRe = regexp.MustCompile(`!\[([^\]]*)\]\(([^)]+)\)`)
var mediaExtRe = regexp.MustCompile(`\.(mp4|webm|mov|avi|mp3|wav|ogg|m4a|aac|flac)(\?|$)`)

type PushMedia struct {
	Alt     string
	URL     string
	RawPath string
	Type    string // image | video | audio
}

func GetPortalPublicBaseURL() string {
	base := strings.TrimRight(strings.TrimSpace(Config.PortalPublicURL), "/")
	if base != "" {
		return base
	}
	return "http://180.152.5.230:5701"
}

func buildPublicMediaURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://") {
		return raw
	}
	if !strings.HasPrefix(raw, "/") {
		raw = "/" + raw
	}
	parts := strings.Split(raw, "/")
	for i, part := range parts {
		if part != "" {
			parts[i] = url.PathEscape(part)
		}
	}
	return GetPortalPublicBaseURL() + strings.Join(parts, "/")
}

func localUploadFile(relativePath string) string {
	relativePath = strings.TrimSpace(relativePath)
	if !strings.HasPrefix(relativePath, "/uploads/") {
		return ""
	}
	local := filepath.Join(ExecPath, filepath.FromSlash(strings.TrimPrefix(relativePath, "/")))
	if st, err := os.Stat(local); err == nil && !st.IsDir() {
		return local
	}
	return ""
}

func classifyMediaURL(raw string) string {
	lower := strings.ToLower(raw)
	if mediaExtRe.MatchString(lower) {
		if strings.Contains(lower, ".mp4") || strings.Contains(lower, ".webm") || strings.Contains(lower, ".mov") || strings.Contains(lower, ".avi") {
			return "video"
		}
		return "audio"
	}
	return "image"
}

func ExtractMarkdownMedia(text string) []PushMedia {
	matches := markdownImageRe.FindAllStringSubmatch(text, -1)
	if len(matches) == 0 {
		return nil
	}
	out := make([]PushMedia, 0, len(matches))
	for _, m := range matches {
		if len(m) < 3 {
			continue
		}
		raw := strings.TrimSpace(m[2])
		if raw == "" {
			continue
		}
		out = append(out, PushMedia{
			Alt:     strings.TrimSpace(m[1]),
			URL:     buildPublicMediaURL(raw),
			RawPath: raw,
			Type:    classifyMediaURL(raw),
		})
	}
	return out
}

func StripMarkdownMedia(text string) string {
	text = markdownImageRe.ReplaceAllString(text, "")
	text = regexp.MustCompile(`\n{3,}`).ReplaceAllString(text, "\n\n")
	return strings.TrimSpace(text)
}

func BuildQQImageCQ(imageURL string, rawPath string) string {
	if cq := buildQQImageCQFromLocal(rawPath); cq != "" {
		return cq
	}
	if imageURL != "" {
		return fmt.Sprintf("[CQ:image,file=%s]", imageURL)
	}
	return ""
}

func buildQQImageCQFromLocal(rawPath string) string {
	local := localUploadFile(rawPath)
	if local == "" {
		return ""
	}
	data, err := ioutil.ReadFile(local)
	if err != nil || len(data) == 0 {
		return ""
	}
	// 小图优先 base64，大图用公网 URL（避免 NapCat 报体积过大）
	if len(data) <= 512*1024 {
		return fmt.Sprintf("[CQ:image,file=base64://%s,type=show]", base64.StdEncoding.EncodeToString(data))
	}
	return ""
}

func SendQQGroupImage(gid int, imageURL string, rawPath string) {
	cq := BuildQQImageCQ(imageURL, rawPath)
	if cq == "" {
		Warn("QQ群图片发送跳过，无法构建 CQ 码: gid=%d url=%s", gid, imageURL)
		return
	}
	SendQQGroup(gid, 0, cq)
}

func SendWxGroupImage(gid string, imageURL string) {
	if gid == "" || imageURL == "" {
		return
	}
	SendWxImg2(gid, imageURL)
}

func PushRichTextToQQGroup(gid int, text string, media []PushMedia) {
	text = strings.TrimSpace(text)
	if text != "" {
		SendQQGroup(gid, 0, text)
	}
	for _, item := range media {
		time.Sleep(400 * time.Millisecond)
		switch item.Type {
		case "image":
			SendQQGroupImage(gid, item.URL, item.RawPath)
		case "video", "audio":
			label := "视频"
			if item.Type == "audio" {
				label = "音频"
			}
			SendQQGroup(gid, 0, fmt.Sprintf("🎬 %s（请在APP或网页端查看）：\n%s", label, item.URL))
		}
	}
}

func PushRichTextToWxGroup(gid string, text string, media []PushMedia) {
	text = strings.TrimSpace(text)
	if text != "" {
		SendWxGroupMsg("", gid, text)
	}
	for _, item := range media {
		time.Sleep(400 * time.Millisecond)
		switch item.Type {
		case "image":
			SendWxGroupImage(gid, item.URL)
		case "video", "audio":
			label := "视频"
			if item.Type == "audio" {
				label = "音频"
			}
			SendWxGroupMsg("", gid, fmt.Sprintf("🎬 %s（请在APP或网页端查看）：\n%s", label, item.URL))
		}
	}
}

func PrepareGroupPushMessage(title, content string) (string, []PushMedia) {
	fullParts := make([]string, 0, 2)
	if strings.TrimSpace(title) != "" {
		fullParts = append(fullParts, title)
	}
	if strings.TrimSpace(content) != "" {
		fullParts = append(fullParts, content)
	}
	fullMsg := strings.Join(fullParts, "\n\n")
	media := ExtractMarkdownMedia(fullMsg)

	var textBuilder strings.Builder
	textBuilder.WriteString("【活动通知】")
	if strings.TrimSpace(title) != "" {
		textBuilder.WriteString("\n")
		textBuilder.WriteString(title)
	}
	strippedContent := StripMarkdownMedia(content)
	if strippedContent != "" {
		textBuilder.WriteString("\n\n")
		textBuilder.WriteString(strippedContent)
	}
	return strings.TrimSpace(textBuilder.String()), media
}
