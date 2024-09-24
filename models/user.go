package models

import (
	"context"
	"errors"
	"fmt"
	"github.com/beego/beego/v2/core/logs"
	"gorm.io/gorm"
	"math/rand"
	"time"
)

type User struct {
	ID       int
	Number   int    `gorm:"unique"`
	Class    string `gorm:"column:class;"`
	ActiveAt time.Time
	Coin     int
	Wxid     string `gorm:"column:wxid;"`
	Nickname string `gorm:"column:nickname;"`
	UosId    string `gorm:"column:uosid;"`
	QQ       string `gorm:"column:qq;"`
	Telegram string `gorm:"column:telegram;"`
	IsAdmin  bool   `gorm:"column:isAdmin;"`
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
	var err error
	maxRetries := 3

	for i := 0; i < maxRetries; i++ {
		ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)

		err = db.WithContext(ctx).Where("wxid = ?", wxid).First(&u).Error
		if err == nil {
			break
		}

		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			// Handle timeout here, for example log an error and retry
			logs.Error("Database query timed out. Retrying...")
			continue
		}
	}

	if err != nil {
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
