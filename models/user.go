package models

import (
	"context"
	"errors"
	"fmt"
	"github.com/beego/beego/v2/core/logs"
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
	var err error
	maxRetries := 3

	for i := 0; i < maxRetries; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

		defer cancel()
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

func getUserId(typ string, uid string) string {
	switch typ {
	case "qq", "qqg":

	case "wx", "wxg":

	case "tg":

	default:
		logs.Info("错误的渠道来源")
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
				"Wxid": user.Wxid,
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
