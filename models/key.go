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
    UsedAt     time.Time // 新增字段，用于记录卡密使用时间
    BatchID    string
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
            UsedAt:     time.Time{}, // 初始化为零值
        }
        if err := db.Create(&u).Error; err != nil {
            return err.Error()
        } else {
            str = append(str, ids)
        }
    }
    return strings.Join(str, "\n")
}



func useKey(id string, use int, source ...string) string {
	src := ""
	if len(source) > 0 {
		src = source[0]
	}
    var u Key
    // 查询数据库以获取具有提供的id的密钥（Token）
    err := db.Where("Token = ?", id).First(&u).Error
    if err != nil {
        return "查无此卡"
    }

    if u.Use {
        return "卡密已被使用"
    }

    var user = &User{}
    // 查询用户以获取提供的use（用户ID）
    err = db.Where("Number = ?", use).First(&user).Error
    if err != nil {
        // 如果找不到用户，则创建一个新的用户
        db.Create(&User{
            Class:   "qq",
            Number:  use,
            Coin:    u.Value,
            ActiveAt: time.Now(),
        })
    } else {
        // 如果找到用户，则更新用户的Coin值
        user.Coin += u.Value
        user.ActiveAt = time.Now()
        db.Where("Number = ?", use).Updates(user)
    }

    // 将密钥标记为已使用，并将其与用户关联
    u.UseBy = use
    u.Use = true
    u.UsedAt = time.Now() // 记录使用时间
    db.Where("Token = ?", id).Updates(u)

    // 在Limit表中创建一个新条目，指定了过期时间和其他细节
    db.Create(&Limit{
        CreateAt: time.Now().Unix() + 30,
        Typ:      1,
        Number:   use,
        Num:      1,
    })

    (&JdCookie{}).Push(fmt.Sprintf("%d通过卡密增加了%d积分，卡密：%s，使用时间：%s", use, u.Value, u.Token, u.UsedAt.Format("2006-01-02 15:04:05")))
    logDesc := fmt.Sprintf("卡密: %s", u.Token)
    if src != "" {
        logDesc = fmt.Sprintf("%s卡密: %s", src, u.Token)
    }
    RecordCoinLog(use, u.Value, "卡密兑换", logDesc)
    return fmt.Sprintf("使用成功，积分增加%d，使用时间：%s", u.Value, u.UsedAt.Format("2006-01-02 15:04:05"))
}


//##创建赠送卡密函数
//##创建赠送卡密函数
//##创建赠送卡密函数
func create_ZSKey(num int, value int) string {
	var str []string
	batchID := uuid.NewV4().String() // 生成一个批次ID，用于标识这一批次卡密
	for i := 0; i < num; i++ {
		id := uuid.NewV4()
		ids := "ZSKM" + id.String()
		var u Key
		u = Key{
			Expiration: time.Now(),
			Token:      ids,
			Use:        false,
			Value:      value,
			UsedAt:     time.Time{}, // 初始化为零值
			BatchID:    batchID,     // 记录批次ID
		}
		if err := db.Create(&u).Error; err != nil {
			return err.Error()
		} else {
			str = append(str, ids)
		}
	}
	return strings.Join(str, "\n")
}

func use_ZSKey(id string, use int, source ...string) string {
    src := ""
    if len(source) > 0 {
        src = source[0]
    }
    var u Key
    // 查询数据库以获取具有提供的id的密钥（Token）
    err := db.Where("Token = ?", id).First(&u).Error
    if err != nil {
   
        return "查无此卡"
    }



    // 检查用户是否已经使用过本次卡密批次的卡密
    var usedCount int64
    // 这里确保查询的是在同一批次下，用户是否已经使用过卡密
    err = db.Model(&Key{}).Where("use_by = ? AND batch_id = ? AND `Use` = true", use, u.BatchID).Count(&usedCount).Error
    if err != nil {
   
        return "查询已使用卡密失败"
    }


    if usedCount > 0 {
        return "每个人只能使用一张卡密"
    }

    if u.Use {

        return "卡密已被使用"
    }

    var user = &User{}
    // 查询用户以获取提供的use（用户ID）
    err = db.Where("Number = ?", use).First(&user).Error
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
        user.ActiveAt = time.Now()
        db.Where("Number = ?", use).Updates(user)

    }

    // 将密钥标记为已使用，并将其与用户关联
    u.UseBy = use
    u.Use = true
    u.UsedAt = time.Now() // 记录使用时间
    err = db.Where("Token = ?", id).Updates(u).Error
    if err != nil {

        return "更新卡密使用状态失败"
    }

    // 推送使用信息
    pushMessage := fmt.Sprintf("%d通过卡密增加了%d积分，卡密：%s，使用时间：%s", use, u.Value, u.Token, u.UsedAt.Format("2006-01-02 15:04:05"))

    (&JdCookie{}).Push(pushMessage)
    logDesc := fmt.Sprintf("赠送卡密: %s", u.Token)
    if src != "" {
        logDesc = fmt.Sprintf("%s赠送卡密: %s", src, u.Token)
    }
    RecordCoinLog(use, u.Value, "卡密兑换", logDesc)

    return fmt.Sprintf("使用成功，积分增加%d，使用时间：%s", u.Value, u.UsedAt.Format("2006-01-02 15:04:05"))
}
