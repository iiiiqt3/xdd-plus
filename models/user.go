package models

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"gorm.io/gorm"
)

type User struct {
	ID       int
	Number   int `gorm:"unique"`
	Class    string
	ActiveAt time.Time
	Coin     int
	Wxid     string `gorm:"column:wxid;"`
	Nickname string `gorm:"column:nickname;"`
	QQ       string `gorm:"column:qq;"`
	Telegram string `gorm:"column:telegram;"`
	IsAdmin  bool   `gorm:"column:isAdmin;"`
	LastSignIn        time.Time  // 最后一次打卡时间
   	ContinuousSignIns int        // 连续打卡天数
     SignInDate        time.Time  // 当前日期的打卡时间

}

type WxUser struct {
	WxNum    string `json:"wx_num"`
	Avatar   string `json:"avatar"`
	City     string `json:"city"`
	Country  string `json:"country"`
	Nickname string `json:"nickname"`
	Province string `json:"province"`
	Note     string `json:"note"`
	Sex      int    `json:"sex"`
	Wxid     string `json:"wxid"`
}


func ClearAllContinuousSignIns() {
    // 更新所有用户的 ContinuousSignIns 为 0
    result := db.Model(&User{}).Update("ContinuousSignIns", 0)

    // 这里可以选择记录日志，表示清零操作已经执行
    fmt.Printf("清零操作完成，影响了 %d 条记录。\n", result.RowsAffected)
}

func ClearCoin(uid int) int {
	var u User
	if db.Where("number = ?", uid).First(&u).Error != nil {
		return 0
	}
	db.Model(u).Updates(map[string]interface{}{
		"coin": gorm.Expr(fmt.Sprintf("%d", 1)),
	})
	u.Coin = 1
	return u.Coin
}

// UpdateUserActiveAt 更新用户最后活跃时间
func UpdateUserActiveAt(uid int) {
	if uid <= 0 {
		return
	}
	db.Model(&User{}).Where("number = ?", uid).Update("active_at", time.Now())
}

func AdddCoin(uid int, num int) int {
	var u User
	if db.Where("number = ?", uid).First(&u).Error != nil {
		return 0
	}
	db.Model(u).Updates(map[string]interface{}{
		"coin":      gorm.Expr(fmt.Sprintf("coin+%d", num)),
		"active_at": time.Now(),
	})
	u.Coin += num
	return u.Coin
}

func AddCoin(uid int) int {
	var u User
	if db.Where("number = ?", uid).First(&u).Error != nil {
		return 0
	}
	db.Model(u).Updates(map[string]interface{}{
		"coin":      gorm.Expr("coin+1"),
		"active_at": time.Now(),
	})
	u.Coin++
	return u.Coin
}

func AddMoney(wx string, money int) bool {
	var u User
	if db.Where("wxid = ?", wx).First(&u).Error != nil {
		SendWxMsg(wx, "充值失败")
		return false
	}
	db.Model(u).Updates(map[string]interface{}{
		"coin":      gorm.Expr(fmt.Sprintf("coin+%d", money)),
		"active_at": time.Now(),
	})
	SendWxMsg(wx, fmt.Sprintf("积分剩余%d", u.Coin+money))
	return true
}

func RemCoin(uid int, num int) int {
	var u User
	if err := db.Where("number = ?", uid).First(&u).Error; err != nil {
		return 0
	}
	if u.Coin < num {
		return u.Coin
	}
	db.Model(u).Updates(map[string]interface{}{
		"coin":      gorm.Expr(fmt.Sprintf("coin-%d", num)),
		"active_at": time.Now(),
	})
	u.Coin -= num
	return u.Coin
}

// DeductCoinChecked 原子扣减积分，不足时返回错误（防止并发白嫖）
func DeductCoinChecked(uid int, num int) error {
	if num <= 0 {
		return nil
	}
	result := db.Model(&User{}).
		Where("number = ? AND coin >= ?", uid, num).
		Updates(map[string]interface{}{
			"coin":      gorm.Expr(fmt.Sprintf("coin-%d", num)),
			"active_at": time.Now(),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		var u User
		if err := db.Where("number = ?", uid).First(&u).Error; err != nil {
			return fmt.Errorf("用户不存在")
		}
		return fmt.Errorf("积分不足，当前%d，需要%d", u.Coin, num)
	}
	return nil
}

func GetCoin(uid int) int {
	var u User
	db.Where("number = ?", uid).First(&u)
	return u.Coin
}

type CoinLog struct {
	ID             int       `gorm:"primaryKey;autoIncrement"`
	UserNumber     int       `gorm:"index;not null"`
	Amount         int       `gorm:"not null"`
	BalanceAfter   int       `gorm:"not null"`
	Type           string    `gorm:"size:32;not null"`
	Detail         string    `gorm:"size:255"`
	ClientSource   string    `gorm:"size:16;index"`
	ClientPlatform string    `gorm:"size:16"`
	CreatedAt      time.Time `gorm:"index"`
}

func (CoinLog) TableName() string {
	return "coin_log"
}

func GetCoinLogs(userNumber int, limit int) []CoinLog {
	var logs []CoinLog
	db.Where("user_number = ?", userNumber).Order("id desc").Limit(limit).Find(&logs)
	return logs
}

func GetCoinLogsFiltered(userNumber int, days int, page int, limit int) ([]CoinLog, int64) {
	return GetCoinLogsFilteredBySource(userNumber, days, page, limit, "")
}

func GetCoinLogsFilteredBySource(userNumber int, days int, page int, limit int, sourceFilter string) ([]CoinLog, int64) {
	var logs []CoinLog
	var total int64
	query := db.Where("user_number = ?", userNumber)
	if days > 0 {
		since := time.Now().AddDate(0, 0, -days)
		query = query.Where("created_at >= ?", since)
	}
	if src := NormalizeSourceFilter(sourceFilter); src != "" {
		query = query.Where("client_source = ?", src)
	}
	query.Model(&CoinLog{}).Count(&total)
	query.Order("id desc").Offset((page - 1) * limit).Limit(limit).Find(&logs)
	return logs, total
}

// CoinLogView API 输出结构
type CoinLogView struct {
	ID             int    `json:"id"`
	Amount         int    `json:"amount"`
	BalanceAfter   int    `json:"balanceAfter"`
	Type           string `json:"type"`
	Detail         string `json:"detail"`
	Source         string `json:"source"`
	SourceLabel    string `json:"sourceLabel"`
	SourceTagCls   string `json:"sourceTagCls"`
	ClientSource   string `json:"clientSource"`
	ClientPlatform string `json:"clientPlatform"`
	CreatedAt      string `json:"createdAt"`
}

func ToCoinLogView(log CoinLog) CoinLogView {
	ctx, detail := ResolveCoinLogContext(log)
	if ctx.IsZero() {
		ctx = AdminContext()
	}
	return CoinLogView{
		ID:             log.ID,
		Amount:         log.Amount,
		BalanceAfter:   log.BalanceAfter,
		Type:           log.Type,
		Detail:         detail,
		Source:         ctx.FilterKey(),
		SourceLabel:    ctx.AdminLabel(),
		SourceTagCls:   ctx.AdminTagClass(),
		ClientSource:   ctx.Source,
		ClientPlatform: ctx.Platform,
		CreatedAt:      log.CreatedAt.Format("2006-01-02 15:04"),
	}
}

func RecordCoinLog(userNumber int, amount int, typ string, detail string, ctx ...ClientContext) {
	RecordCoinLogEx(userNumber, amount, typ, detail, pickClientContext(ctx))
}

func RecordCoinLogEx(userNumber int, amount int, typ string, detail string, ctx ClientContext) {
	if userNumber <= 0 {
		return
	}
	ctx = ctx.normalized()
	if ctx.IsZero() {
		ctx = AdminContext()
	}
	balanceAfter := GetCoin(userNumber)
	log := CoinLog{
		UserNumber:     userNumber,
		Amount:         amount,
		BalanceAfter:   balanceAfter,
		Type:           typ,
		Detail:         detail,
		ClientSource:   ctx.Source,
		ClientPlatform: ctx.Platform,
		CreatedAt:      time.Now(),
	}
	if err := db.Create(&log).Error; err != nil {
		Warn("[积分日志] 记录失败 user=%d amount=%d type=%s: %v", userNumber, amount, typ, err)
	}
}

func pickClientContext(ctx []ClientContext) ClientContext {
	if len(ctx) > 0 {
		return ctx[0]
	}
	return ClientContext{}
}

func GetWxid(wxid string) int {
	var u User
	var err error
	maxRetries := 3

	for i := 0; i < maxRetries; i++ {
		ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)

		err = db.WithContext(ctx).Where("wxid = ?", wxid).First(&u).Error
		if err == nil {
			break
		}

		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			Error("Database query timed out. Retrying...")
			continue
		}
	}

	if err != nil {
		tt := rand.Int()
		newUser := User{
			Class:    "wx",
			Number:   tt,
			Coin:     0,
			ActiveAt: time.Now(),
			Wxid:     wxid,
		}
		numStr := fmt.Sprintf("%d", tt)
		if len(numStr) < 16 {
			newUser.QQ = numStr
		}
		db.Create(&newUser)
		go fillWxNickname(wxid)
		return tt
	} else {
		updates := map[string]interface{}{}
		numStr := fmt.Sprintf("%d", u.Number)
		if len(numStr) < 16 && u.QQ == "" {
			updates["qq"] = numStr
		}
		if u.Nickname == "" {
			go fillWxNickname(wxid)
		}
		if len(updates) > 0 {
			db.Model(&User{}).Where("number = ?", u.Number).Updates(updates)
		}
		return u.Number
	}
}

func fillWxNickname(wxid string) {
	nickname := GetWxNickname(wxid)
	if nickname == "" {
		return
	}
	result := db.Model(&User{}).Where("wxid = ? AND (nickname = '' OR nickname IS NULL)", wxid).Update("nickname", nickname)
	if result.RowsAffected > 0 {
		Info("微信昵称补全成功:", wxid, "->", nickname)
	}
}

func UpdateUserNicknameIfEmpty(number int, nickname string) {
	if nickname == "" {
		return
	}
	result := db.Model(&User{}).Where("number = ? AND (nickname = '' OR nickname IS NULL)", number).Update("nickname", nickname)
	if result.RowsAffected > 0 {
		Info("QQ昵称补全成功:", number, "->", nickname)
	}
}

func getUserId(typ string, uid string) string {
	switch typ {
	case "qq", "qqg":

	case "wx", "wxg":

	case "tg":

	default:
		Info("错误的渠道来源")
	}

	return ""
}

func setWxId(uid string, wxid string) string {
	var u User
	db.Where("wxid = ? and class = ?", wxid, "wx").Delete(&u)
	if db.Where("wxid = ?", uid).First(&u).Error != nil {
		return "绑定失败"
	} else {
		db.Model(u).Updates(map[string]interface{}{
			"wxid": wxid,
		})
		return "绑定成功"
	}
}

func makeWxId(uid int, wxid string) string {
	var u User
	if db.Where("number = ?", uid).First(&u).Error != nil {
		db.Create(&User{
			Class:    "qq",
			Number:   uid,
			Coin:     0,
			ActiveAt: time.Now(),
			Wxid:     wxid,
		})
	} else {
		db.Model(u).Updates(map[string]interface{}{
			"wxid": wxid,
		})
	}
	return wxid
}

func getWeiXinId(QQid int) string {
	var u User
	if db.Where("number = ?", QQid).First(&u).Error != nil {
		return "找不到对应的微信ID"
	}
	return u.Wxid
}

func deleteDuplicateWxidUsers() {
	var users []User
	db.Find(&users)

	seen := make(map[string]bool)
	for _, user := range users {
		// 判断wxid是否为空
		if user.Wxid == "" {
			continue
		}

		//查询jd_cookie是否存在对应的number有的话写入wxid
		var jdCookie JdCookie
		if db.Where("qq = ?", user.Number).First(&jdCookie).Error == nil {
			db.Model(jdCookie).Updates(map[string]interface{}{
				"WeiXin": user.Wxid,
			})
		}

		if _, ok := seen[user.Wxid]; ok {
			// wxid已经存在，删除这个用户
			//判断用户coin是否为0是的话删除否则保留
			if user.Coin == 0 {
				db.Delete(&user)
			} else {
				JdCookie{}.Push(fmt.Sprintf("用户 %d 的微信ID %s 重复，但是积分不为0，已保留", user.Number, user.Wxid))
			}
		} else {
			// wxid不存在，将其添加到seen map中
			seen[user.Wxid] = true
		}
	}
}