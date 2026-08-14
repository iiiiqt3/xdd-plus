package models

import "strconv"

const defaultPortalMinCoinForAccess = 1000

// GetPortalMinCoinForAccess 门户活动中心/通知中心可见所需最低积分。
// 未配置时默认 1000；明确填 0 表示不限制，所有登录用户可见。
func GetPortalMinCoinForAccess() int {
	if sysConfig.PortalMinCoinForAccess == nil {
		return defaultPortalMinCoinForAccess
	}
	v := *sysConfig.PortalMinCoinForAccess
	if v < 0 {
		return defaultPortalMinCoinForAccess
	}
	return v
}

// PortalAccessInfo 门户受限内容访问状态（供网页/App 可选读取）
type PortalAccessInfo struct {
	Allowed      bool   `json:"allowed"`
	Coin         int    `json:"coin"`
	RequiredCoin int    `json:"requiredCoin"`
	GapCoin      int    `json:"gapCoin"`
	Message      string `json:"message"`
}

func BuildPortalAccessInfo(userNumber int, coin int) PortalAccessInfo {
	required := GetPortalMinCoinForAccess()
	info := PortalAccessInfo{
		Coin:         coin,
		RequiredCoin: required,
	}
	if coin >= required {
		info.GapCoin = 0
	} else {
		info.GapCoin = required - coin
	}
	if CanAccessPortalContent(userNumber, coin) {
		info.Allowed = true
		return info
	}
	info.Allowed = false
	info.Message = "活动中心和通知中心需积分达到 " + strconv.Itoa(required) + "，或拥有有效的按月付费项目后开放"
	if info.GapCoin > 0 {
		info.Message += "（当前 " + strconv.Itoa(info.Coin) + "，还差 " + strconv.Itoa(info.GapCoin) + "）"
	}
	return info
}

func CanAccessPortalContent(userNumber int, coin int) bool {
	if coin >= GetPortalMinCoinForAccess() {
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
	required := GetPortalMinCoinForAccess()
	seen := map[int]bool{}
	out := make([]int, 0, 64)

	var richUsers []User
	db.Where("coin >= ?", required).Select("number").Find(&richUsers)
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
