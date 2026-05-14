package models

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
//	"time"
)

// 定义接收 API 返回数据的结构体
type NewsResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		Date       string   `json:"date"`
		DayOfWeek  string   `json:"day_of_week"`
		LunarDate  string   `json:"lunar_date"`
		NewsList   []string `json:"news"`
		Tip        string   `json:"tip"`
		ImageURL   string   `json:"image"`
		LinkURL    string   `json:"link"`
		ApiUpdated string   `json:"api_updated"`
	} `json:"data"`
}



func HandleNews() {
	apiURL := "https://60s.viki.moe/v2/60s"
	
	resp, err := http.Get(apiURL)
	if err != nil {
		fmt.Println("获取新闻失败:", err)
		return
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("读取响应失败:", err)
		return
	}

	var result NewsResponse
	if err := json.Unmarshal(body, &result); err != nil {
		fmt.Println("JSON 解析失败:", err)
		return
	}

	if result.Code != 200 {
		fmt.Println("API 返回错误码:", result.Code, result.Message)
		return
	}

	var sb strings.Builder
	
	// 1. 头部信息 (日期 + 农历)
	sb.WriteString(fmt.Sprintf("📅 %s %s\n", result.Data.Date, result.Data.DayOfWeek))
	sb.WriteString(fmt.Sprintf("🏮 农历：%s\n\n", result.Data.LunarDate))
	
	// 2. 分隔线
	sb.WriteString("━━━━━━━━━━━━━━\n")

	// 3. 新闻列表 (每条新闻后加一个空行，增加呼吸感)
	for i, news := range result.Data.NewsList {
		sb.WriteString(fmt.Sprintf("%d. %s\n\n", i+1, news))
	}

	// 4. 底部金句 (再次加分隔线 + 金句)
	sb.WriteString("━━━━━━━━━━━━━━\n\n")
	sb.WriteString("💡 每日一句：\n")
	sb.WriteString(result.Data.Tip)

	finalContent := sb.String()

	// 推送消息
	targetGroupID := "56711195905@chatroom"
	SendWxGroupMsg("1", targetGroupID, finalContent)

	fmt.Println("新闻推送成功！")
}