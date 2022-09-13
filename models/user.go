package models

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

type User struct {
	ID       int
	Number   int `gorm:"unique"`
	Class    string
	ActiveAt time.Time
	Coin     int
	Womail   string
	Wxid     string
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

func getWxId(uid string) int {
	var u User
	db.Where("wxid = ?", uid).First(&u)
	return u.Number
}

func setWxId(uid string, wxid string) string {

	var u User
	if db.Where("wxid = ?", uid).First(&u).Error != nil {
		return "绑定失败"
	} else {
		db.Model(u).Updates(map[string]interface{}{
			"wxid": wxid,
		})
		return "绑定成功"
	}

}

func makeWxId(uid int, wxid string) {

	var u User
	if db.Where("number = ?", uid).First(&u).Error != nil {

	} else {
		db.Model(u).Updates(map[string]interface{}{
			"wxid": wxid,
		})
	}

}
