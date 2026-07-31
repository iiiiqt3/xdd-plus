package models

import "strconv"

const PortalMinCoinForAccess = 1000

// PortalAccessInfo 门户受限内容访问状态（供网页/App 可选读取）
type PortalAccessInfo struct {
	Allowed      bool   `json:"allowed"`
	Coin         int    `json:"coin"`
	RequiredCoin int    `json:"requiredCoin"`
	GapCoin      int    `json:"gapCoin"`
	Message      string `json:"message"`
}

func BuildPortalAccessInfo(userNumber int, coin int) PortalAccessInfo {
	info := PortalAccessInfo{
		Coin:         coin,
		RequiredCoin: PortalMinCoinForAccess,
	}
	if coin >= PortalMinCoinForAccess {
		info.GapCoin = 0
	} else {
		info.GapCoin = PortalMinCoinForAccess - coin
	}
	if CanAccessPortalContent(userNumber, coin) {
		info.Allowed = true
		return info
	}
	info.Allowed = false
	info.Message = "活动中心和通知中心需积分达到 1000，或拥有有效的按月付费项目后开放"
	if info.GapCoin > 0 {
		info.Message += "（当前 " + strconv.Itoa(info.Coin) + "，还差 " + strconv.Itoa(info.GapCoin) + "）"
	}
	return info
}

func CanAccessPortalContent(userNumber int, coin int) bool {
	if coin >= PortalMinCoinForAccess {
		return true
	}
	return HasValidMonthlyProject(userNumber)
}

// HasValidMonthlyProject 是否存在未过期的按月付费项目
func HasValidMonthlyProject(userNumber int) bool {
	projects, err := GetPortalProjects(userNumber)
	if err != nil {
		return false
	}
	for _, project := range projects {
		if project.IsMonthlyDeduct && project.BizStatus != "expired" {
			return true
		}
	}
	return false
}

// ListPortalAccessUserNumbers 返回可查看受限门户内容的用户编号（用于定向极光推送）
func ListPortalAccessUserNumbers() []int {
	seen := map[int]bool{}
	out := make([]int, 0, 64)

	var richUsers []User
	db.Where("coin >= ?", PortalMinCoinForAccess).Select("number").Find(&richUsers)
	for _, u := range richUsers {
		if u.Number > 0 && !seen[u.Number] {
			seen[u.Number] = true
			out = append(out, u.Number)
		}
	}

	var accounts []WebUserAccount
	db.Select("user_number").Find(&accounts)
	for _, acc := range accounts {
		if acc.UserNumber <= 0 || seen[acc.UserNumber] {
			continue
		}
		if HasValidMonthlyProject(acc.UserNumber) {
			seen[acc.UserNumber] = true
			out = append(out, acc.UserNumber)
		}
	}
	return out
}
