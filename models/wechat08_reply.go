package models

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	browser "github.com/EDDYCJY/fake-useragent"
	"github.com/beego/beego/v2/client/httplib"
	"github.com/buger/jsonparser"
)

// wechat08BotBaseURL 机器人专用 API 根路径，例如 http://host:8061/api/v1/bot/
func wechat08BotBaseURL() string {
	u := strings.TrimSpace(Config.Wx.Url)
	if u == "" {
		return ""
	}
	if !strings.HasSuffix(u, "/") {
		u += "/"
	}
	return u
}

func wechat08HttpAPIURL() string {
	base := wechat08BotBaseURL()
	return fmt.Sprintf("%shttpapi/?wxid=%s&token=%s", base, Config.Wx.Robotid, Config.Wx.Token)
}

func wechat08PostJSON(path string, body interface{}) (string, error) {
	base := wechat08BotBaseURL()
	url := base + strings.TrimPrefix(path, "/")
	if strings.Contains(url, "?") {
		url += "&token=" + Config.Wx.Token
	} else {
		url += "?token=" + Config.Wx.Token
	}
	req := httplib.Post(url)
	req.Header("Content-Type", "application/json")
	req.Header("X-Bot-Token", Config.Wx.Token)
	req.Header("User-Agent", browser.Random())
	req.SetTimeout(30*time.Second, 30*time.Second)
	raw, err := json.Marshal(body)
	if err != nil {
		return "", err
	}
	req.Body(raw)
	return req.String()
}

func wechat08CallType(typeCode string, data map[string]string) (string, error) {
	payload := map[string]interface{}{
		"type": typeCode,
		"data": data,
	}
	req := httplib.Post(wechat08HttpAPIURL())
	req.Header("Content-Type", "application/json")
	req.Header("X-Bot-Token", Config.Wx.Token)
	req.Header("User-Agent", browser.Random())
	req.SetTimeout(30*time.Second, 30*time.Second)
	raw, _ := json.Marshal(payload)
	Bot().Infof("[wechat08] type=%s body=%s", typeCode, string(raw))
	req.Body(raw)
	resp, err := req.String()
	if err != nil {
		return "", err
	}
	Bot().Infof("[wechat08] resp=%s", resp)
	return resp, nil
}

func wechat08RespOK(resp string) bool {
	if resp == "" {
		return false
	}
	code, err := jsonparser.GetInt([]byte(resp), "Code")
	if err == nil {
		return code == 0
	}
	ok, err := jsonparser.GetBoolean([]byte(resp), "Success")
	return err == nil && ok
}

// Wechat08SendMsg 发文字
func Wechat08SendMsg(uid, msg string) {
	msg = strings.ReplaceAll(msg, "\r", "\n")
	_, err := wechat08CallType("Q0001", map[string]string{
		"wxid": uid,
		"msg":  msg,
	})
	if err != nil {
		Error("[wechat08] 发文字失败:", err)
	}
}

// Wechat08SendImgBytes 发图片（本地字节 → 图床 URL → Q0010）
func Wechat08SendImgBytes(uid string, file []byte) {
	unix := time.Now().Unix()
	filename := ExecPath + fmt.Sprintf("/%d.jpg", unix)
	f, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0777)
	if err != nil {
		Bot().Warnf("wechat08 写临时图失败: %v", err)
		return
	}
	_, _ = f.Write(file)
	_ = f.Close()
	img := uploadImg(filename)
	_ = os.Remove(filename)
	Wechat08SendImgURL(uid, img)
}

// Wechat08SendImgURL 发图片 URL
func Wechat08SendImgURL(uid, imgURL string) {
	_, err := wechat08CallType("Q0010", map[string]string{
		"wxid": uid,
		"path": imgURL,
	})
	if err != nil {
		Error("[wechat08] 发图片失败:", err)
	}
}

// Wechat08SendVideoURL 发视频（URL 下载后转 base64）
func Wechat08SendVideoURL(uid, videoURL, thumbURL string, playLength int) {
	if playLength <= 0 {
		playLength = 10
	}
	data := map[string]string{
		"wxid":       uid,
		"base64":     videoURL,
		"playlength": fmt.Sprintf("%d", playLength),
	}
	if thumbURL != "" {
		data["imagebase64"] = thumbURL
	}
	_, err := wechat08CallType("Q_VIDEO", data)
	if err != nil {
		Error("[wechat08] 发视频失败:", err)
	}
}

// Wechat08SendGroupAt 群 @ 消息
func Wechat08SendGroupAt(uid, gid, msg string) {
	msg = strings.ReplaceAll(msg, "\r", "\n")
	body := map[string]interface{}{
		"Wxid":    Config.Wx.Robotid,
		"ToWxid":  gid,
		"Content": msg,
		"At":      uid,
		"token":   Config.Wx.Token,
	}
	resp, err := wechat08PostJSON("send/txt", body)
	if err != nil {
		Error("[wechat08] 群@失败:", err)
		return
	}
	Bot().Infof("[wechat08] 群@响应: %s", resp)
}

// Wechat08TransferRequest 确认收款
func Wechat08TransferRequest(toWxid, transferID, transactionID, invalidTime, money string) error {
	data := map[string]string{
		"wxid":          toWxid,
		"transferid":    transferID,
		"transactionid": transactionID,
		"invalidtime":   invalidTime,
	}
	resp, err := wechat08CallType("Q0016", data)
	if err != nil {
		return err
	}
	if !wechat08RespOK(resp) {
		msg, _ := jsonparser.GetString([]byte(resp), "Message")
		if msg == "" {
			msg = resp
		}
		return fmt.Errorf("转账确认失败: %s", msg)
	}
	_ = money
	return nil
}

// Wechat08AgreeFriend 同意好友
func Wechat08AgreeFriend(v3, v4, scene, uid string) bool {
	if scene == "" {
		scene = "3"
	}
	resp, err := wechat08CallType("Q0017", map[string]string{
		"v3":    v3,
		"v4":    v4,
		"scene": scene,
	})
	if err != nil {
		Error("[wechat08] 同意好友失败:", err)
		return false
	}
	ok := wechat08RespOK(resp)
	if ok {
		welcome := GetEnv("Welcome")
		if welcome != "" && uid != "" {
			Wechat08SendMsg(uid, welcome)
		}
	}
	return ok
}

// Wechat08InviteGroup 拉人进群
func Wechat08InviteGroup(uid, gid string) {
	UserLog().Infof("Wechat08InviteGroup uid=%s gid=%s", uid, gid)
	_, err := wechat08CallType("Q0021", map[string]string{
		"wxid":    gid,
		"objWxid": uid,
		"type":    "2",
	})
	if err != nil {
		Error("[wechat08] 拉群失败:", err)
	}
}

// Wechat08DownloadAsBase64 辅助：下载 URL 为 base64（调试用）
func Wechat08DownloadAsBase64(url string) (string, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(b), nil
}
