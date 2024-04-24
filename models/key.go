package models

import (
	"fmt"
	uuid "github.com/satori/go.uuid"
	"strings"
	"time"
)

type Key struct {
	Expiration time.Time
	Token      string
	Use        bool
	UseBy      int
	Value      int
}

func createKey(num int, value int) string {
	var str []string
	for i := 0; i < num; i++ {
		id := uuid.NewV4()
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

func useKey(id string, use int) string {
    // 如果用户编号为694738267，则跳过时间限制检查
    if use == 694738267 || getLimitByTime(use, 1, 30000) { // # 1是类型 600是秒 其他限制可以使用234继续下去
        var u Key
        // 查询数据库以获取具有提供的id的密钥（Token）
        err := db.Where("Token = ?", id).First(&u).Error
        if err == nil {
            if u.Use != true {
                var user = &User{}
                // 查询用户以获取提供的use（用户ID）
                err := db.Where("Number = ?", use).First(&user).Error
                if err != nil {
                    // 如果找不到用户，则创建一个新的用户
                    db.Create(&User{
                        Class:    "qq",
                        Number:   use,
                        Coin:     u.Value,
                        ActiveAt: time.Now(),
                    })
                } else {
                    // 如果找到用户，则更新用户的Coin值
                    user.Coin += u.Value
                    db.Where("Number = ?", use).Updates(user)
                }
                
                // 将密钥标记为已使用，并将其与用户关联
                u.UseBy = use
                u.Use = true
                db.Where("Token = ?", id).Updates(u)
                
                // 在Limit表中创建一个新条目，指定了过期时间和其他细节
                db.Create(&Limit{
                    CreateAt: time.Now().Unix() + 30000,
                    Typ:      1,
                    Number:   use,
                    Num:      1,
                })
                return fmt.Sprintf("使用成功，积分增加%d", u.Value)
            } else {
                return "卡密已被使用"
            }
        }
        return "查无此卡"        
    } else {
        return "每个人固定时间内只能使用一张卡密，请等待冷却，如果你急用，可联系群主。微信请对机器人说：拉群 进群后可联系群主"
    }
}




