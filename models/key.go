package models

import (
	"fmt"
	"github.com/google/uuid"
	"strings"
	"time"
)

type Key struct {
	Expiration time.Time
	Token      string
	Use        bool
	UseBy      string
	Value      int
}

func createKey(num int, value int) string {
	var str []string
	for i := 0; i < num; i++ {
		id := uuid.New()
		ids := "XDD" + id.String()
		var u Key
		u = Key{
			Expiration: time.Now(),
			Token:      ids,
			Use:        false,
			Value:      value,
		}
		if err := db.Create(&u).Error; err != nil {
			return err.Error()
		} else {
			str = append(str, ids)
		}
	}
	return strings.Join(str, "\n")
}

func useKey(id string, use string) string {
	var u Key
	err := db.Where("Token = ?", id).First(&u).Error
	if err == nil {
		if u.Use != true {
			var user = &User{}
			err := db.Where("Number = ?", use).First(&user).Error
			if err != nil {
				db.Create(&User{
					Class:    "qq",
					UserId:   use,
					Coin:     u.Value,
					ActiveAt: time.Now(),
				})
			} else {
				user.Coin += u.Value
				db.Where("Number = ?", use).Updates(user)
			}
			u.UseBy = use
			u.Use = true
			db.Where("Token = ?", id).Updates(u)
			return fmt.Sprintf("使用成功,积分增加%d", u.Value)
		} else {
			return "卡密已被使用"
		}
	}
	return "查无此卡"
}
