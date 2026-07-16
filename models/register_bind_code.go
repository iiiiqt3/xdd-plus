package models

import (
    "fmt"
    "strings"
    "time"

    "github.com/google/uuid"
)

type RegisterBindCode struct {
    ID            int       `gorm:"primaryKey"`
    Code          string    `gorm:"size:64;uniqueIndex"`
    UserNumber    int       `gorm:"index"`
    Used          bool      `gorm:"default:false"`
    UsedAt        time.Time
    ExpiresAt     time.Time `gorm:"index"`
    UsedByAccount string    `gorm:"size:64"`
    CreatedAt     time.Time
    UpdatedAt     time.Time
}

const RegisterBindCodePrefix = "REG"
const RegisterBindCodeTTL = 30 * time.Minute

func CreateRegisterBindCode(userNumber int) (*RegisterBindCode, error) {
    if userNumber <= 0 {
        return nil, fmt.Errorf("无效的用户编号")
    }
    code := RegisterBindCodePrefix + strings.ReplaceAll(uuid.New().String(), "-", "")[:12]
    entity := &RegisterBindCode{
        Code:       code,
        UserNumber: userNumber,
        Used:       false,
        ExpiresAt:  time.Now().Add(RegisterBindCodeTTL),
    }
    if err := db.Create(entity).Error; err != nil {
        return nil, fmt.Errorf("创建注册绑定ID失败：%v", err)
    }
    return entity, nil
}

func ConsumeRegisterBindCode(code string, username string) (int, error) {
    code = strings.TrimSpace(strings.ToUpper(code))
    if code == "" {
        return 0, fmt.Errorf("注册绑定ID不能为空")
    }

    var bind RegisterBindCode
    if err := db.Where("code = ?", code).First(&bind).Error; err != nil {
        return 0, fmt.Errorf("注册绑定ID无效，请重新向机器人获取")
    }
    if bind.Used {
        return 0, fmt.Errorf("注册绑定ID已被使用，请重新向机器人获取")
    }
    if time.Now().After(bind.ExpiresAt) {
        return 0, fmt.Errorf("注册绑定ID已过期，请重新向机器人获取")
    }

    bind.Used = true
    bind.UsedAt = time.Now()
    bind.UsedByAccount = username
    if err := db.Save(&bind).Error; err != nil {
        return 0, fmt.Errorf("锁定注册绑定ID失败：%v", err)
    }
    return bind.UserNumber, nil
}
