package models

import (
	"fmt"
	"gorm.io/gorm"
)

type User struct {
	ID       int
	UserId   string `gorm:"unique"`
	Class    string `gorm:"column:class;"`
	ActiveAt string `gorm:"column:activeAt;"`
	Coin     int
	IsAdmin  bool `gorm:"column:isAdmin;"`
}

func ClearCoin(uid string) int {
	var u User
	if db.Where("userid = ?", uid).First(&u).Error != nil {
		return 0
	}
	db.Model(u).Updates(map[string]interface{}{
		"coin": gorm.Expr(fmt.Sprintf("%d", 1)),
	})
	u.Coin = 1
	return u.Coin
}

func AdddCoin(uid string, num int) int {
	var u User
	if db.Where("userid = ?", uid).First(&u).Error != nil {
		return 0
	}
	db.Model(u).Updates(map[string]interface{}{
		"coin": gorm.Expr(fmt.Sprintf("coin+%d", num)),
	})
	u.Coin += num
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

func RemCoin(uid string, num int) int {
	var u User
	db.Where("userid = ?", uid).First(&u)
	db.Model(u).Updates(map[string]interface{}{
		"coin": gorm.Expr(fmt.Sprintf("coin-%d", num)),
	})
	u.Coin -= num
	return u.Coin
}

func GetCoin(uid string) int {
	var u User
	db.Where("userid = ?", uid).First(&u)
	return u.Coin
}
