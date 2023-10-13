package models

import (
	"fmt"
	"math/rand"
	"time"

	"gorm.io/gorm"
)

type User struct {
	ID         int
	Number     int `gorm:"unique"`
	Class      string
	ActiveAt   time.Time
	Coin       int
	Wxid       string
	QQ         string
	TelegramId string
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
func AdddCoin(uid int, num int) int {
	var u User
	if db.Where("number = ?", uid).First(&u).Error != nil {
		return 0
	}
	db.Model(u).Updates(map[string]interface{}{
		"coin": gorm.Expr(fmt.Sprintf("coin+%d", num)),
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
		"coin": gorm.Expr("coin+1"),
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
		"coin": gorm.Expr(fmt.Sprintf("coin+%d", money)),
	})
	SendWxMsg(wx, fmt.Sprintf("积分剩余%d", u.Coin+money))
	return true
}

func RemCoin(uid int, num int) int {
	var u User
	db.Where("number = ?", uid).First(&u)
	db.Model(u).Updates(map[string]interface{}{
		"coin": gorm.Expr(fmt.Sprintf("coin-%d", num)),
	})
	u.Coin -= num
	return u.Coin
}

func GetCoin(uid int) int {
	var u User
	db.Where("number = ?", uid).First(&u)
	return u.Coin
}

func getWxId(wxid string) int {
	var u User
	if db.Where("wxid = ?", wxid).First(&u).Error != nil {
		tt := rand.Int()
		db.Create(&User{
			Class:    "wx",
			Number:   tt,
			Coin:     0,
			ActiveAt: time.Now(),
			Wxid:     wxid,
		})
		return tt
	} else {
		return u.Number
	}
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
