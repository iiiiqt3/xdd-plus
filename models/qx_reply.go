package models

import (
	"encoding/json"
	"fmt"
	"strings"

	browser "github.com/EDDYCJY/fake-useragent"
	"github.com/beego/beego/v2/client/httplib"
)

// qxBaseURL 千寻 HTTP 根地址（去掉尾部斜杠）
func qxBaseURL() string {
	return strings.TrimRight(strings.TrimSpace(Config.Wx.Url), "/")
}

// qxHttpAPIURL 千寻 DaenWxHook 发消息/拉群等接口
func qxHttpAPIURL() string {
	return fmt.Sprintf("%s/DaenWxHook/httpapi/?wxid=%s", qxBaseURL(), Config.Wx.Robotid)
}

// QxHttpAPIURL 供其他包调用
func QxHttpAPIURL() string {
	return qxHttpAPIURL()
}

func QxPostHook(body interface{}) (string, error) {
	return qxPostHook(body)
}

func qxPostHook(body interface{}) (string, error) {
	url := qxHttpAPIURL()
	raw, err := json.Marshal(body)
	if err != nil {
		return "", err
	}
	req := httplib.Post(url)
	req.Header("Content-Type", "application/json")
	req.Header("User-Agent", browser.Random())
	req.Body(string(raw))
	Bot().Infof("[qx] POST %s body=%s", url, string(raw))
	resp, err := req.String()
	if err != nil {
		Bot().Errorf("[qx] 请求失败: %v", err)
		return "", err
	}
	Bot().Infof("[qx] resp=%s", resp)
	return resp, nil
}
