package yybportal

import (
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/cdle/xdd/models"
)

// HandleBotJdLogin 机器人端应用宝京东 CK 刷新
func HandleBotJdLogin(sender *models.Sender, msg chan string) {
	accounts, err := findUserYybAccounts(sender.UserID)
	if err != nil || len(accounts) == 0 {
		sender.Reply("❌ 未检测到可用的应用宝账号\n\n请先在用户中心「应用宝协议」页面扫码绑定，或发送「应用宝扫码」绑定账号。\n已退出登录流程。")
		return
	}

	var menu strings.Builder
	menu.WriteString("📲 请选择应用宝账号：\n\n")
	for i, acc := range accounts {
		nick := acc.Nickname
		if nick == "" {
			nick = acc.OpenID
		}
		st := strings.ToLower(strings.TrimSpace(acc.Status))
		statusText := "🟢 可用"
		if st != "" && st != "alive" && st != "online" {
			statusText = "🔴 失效"
		}
		menu.WriteString(fmt.Sprintf("%d、%s (%s) [%s]\n", i+1, nick, acc.OpenID, statusText))
	}
	menu.WriteString("\n输入序号刷新对应账号，输入 0 刷新全部，输入 q 退出：")
	sender.Reply(menu.String())

	devInput, ok := models.WaitJdBotInput(sender, msg, 60)
	if !ok {
		return
	}
	if devInput == "q" || devInput == "Q" {
		sender.Reply("已退出登录流程")
		return
	}
	devIdx, err := strconv.Atoi(devInput)
	if err != nil || devIdx < 0 || devIdx > len(accounts) {
		sender.Reply("输入无效，已退出登录流程")
		return
	}

	if devIdx == 0 {
		sender.Reply(fmt.Sprintf("⏳ 正在刷新全部 %d 个账号，请稍候...", len(accounts)))
		success := 0
		fail := 0
		for i, acc := range accounts {
			if i > 0 {
				time.Sleep(time.Duration(rand.Intn(2000)+3000) * time.Millisecond)
			}
			if botYybJdRefreshOne(sender, acc.OpenID) {
				success++
			} else {
				fail++
			}
		}
		sender.Reply(fmt.Sprintf("🔄 批量刷新完成\n✅ 成功: %d\n❌ 失败: %d", success, fail))
	} else {
		sender.Reply("⏳ 正在刷新，请稍候...")
		botYybJdRefreshOne(sender, accounts[devIdx-1].OpenID)
	}
}

func botYybJdRefreshOne(sender *models.Sender, openid string) bool {
	ptKey, ptPin, err := yybJdRefreshCK(openid)
	if err != nil {
		var riskErr *models.RiskVerifyError
		if errors.As(err, &riskErr) {
			if models.WaitJdRiskVerify(sender, riskErr.JmpURL) {
				return botYybJdRefreshOne(sender, openid)
			}
			return false
		}
		sender.Reply(fmt.Sprintf("❌ 刷新失败: %v", err))
		return false
	}
	nick, err := saveYybJdCookie(sender.UserID, openid, ptKey, ptPin)
	if err != nil {
		sender.Reply(fmt.Sprintf("❌ %v", err))
		return false
	}
	sender.Reply(fmt.Sprintf("✅ [%s] CK刷新成功！(应用宝)", nick))
	return true
}
