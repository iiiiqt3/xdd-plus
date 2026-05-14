package models

import (
    "fmt"
    "math/rand"
    "strings"
    "time"
)

type PasswordResetCode struct {
    ID            int       `gorm:"primaryKey"`
    Code          string    `gorm:"size:16;uniqueIndex"`
    UserNumber    int       `gorm:"index"`
    Used          bool      `gorm:"default:false"`
    UsedAt        time.Time
    ExpiresAt     time.Time `gorm:"index"`
    UsedByAccount string    `gorm:"size:64"`
    CreatedAt     time.Time
    UpdatedAt     time.Time
}

const PasswordResetCodeTTL = 30 * time.Minute
const PasswordResetCodeLength = 6

func init() {
    rand.Seed(time.Now().UnixNano())
}

func generatePasswordResetCode() string {
    return fmt.Sprintf("%06d", rand.Intn(1000000))
}

func CreatePasswordResetCode(userNumber int) (*PasswordResetCode, error) {
    if userNumber <= 0 {
        return nil, fmt.Errorf("无效的用户编号")
    }

    account, err := GetWebUserAccountByUserNumber(userNumber)
    if err != nil || account == nil {
        return nil, fmt.Errorf("你还没有注册网页账号，请先进入网页完成注册")
    }

    for i := 0; i < 10; i++ {
        entity := &PasswordResetCode{
            Code:       generatePasswordResetCode(),
            UserNumber: userNumber,
            Used:       false,
            ExpiresAt:  time.Now().Add(PasswordResetCodeTTL),
        }
        if err := db.Create(entity).Error; err == nil {
            return entity, nil
        }
    }

    return nil, fmt.Errorf("生成重置验证码失败，请稍后重试")
}

func getValidPasswordResetCode(code string) (*PasswordResetCode, error) {
    code = strings.TrimSpace(code)
    if code == "" {
        return nil, fmt.Errorf("重置验证码不能为空")
    }

    var resetCode PasswordResetCode
    if err := db.Where("code = ?", code).First(&resetCode).Error; err != nil {
        return nil, fmt.Errorf("重置验证码无效，请重新向机器人获取")
    }
    if resetCode.Used {
        return nil, fmt.Errorf("重置验证码已被使用，请重新向机器人获取")
    }
    if time.Now().After(resetCode.ExpiresAt) {
        return nil, fmt.Errorf("重置验证码已过期，请重新向机器人获取")
    }
    return &resetCode, nil
}

func GetPasswordResetAccountByCode(code string) (*WebUserAccount, error) {
    resetCode, err := getValidPasswordResetCode(code)
    if err != nil {
        return nil, err
    }

    account, err := GetWebUserAccountByUserNumber(resetCode.UserNumber)
    if err != nil {
        return nil, fmt.Errorf("该验证码对应用户尚未注册网页账号")
    }
    if account.Status != "active" {
        return nil, fmt.Errorf("该网页账号已被禁用")
    }
    return account, nil
}

func ConsumePasswordResetCode(code string, username string) (*WebUserAccount, error) {
    resetCode, err := getValidPasswordResetCode(code)
    if err != nil {
        return nil, err
    }

    account, err := GetWebUserAccountByUserNumber(resetCode.UserNumber)
    if err != nil {
        return nil, fmt.Errorf("该验证码对应用户尚未注册网页账号")
    }

    normalized := normalizeWebUsername(username)
    if normalized == "" {
        return nil, fmt.Errorf("用户名不能为空")
    }
    if account.Username != normalized {
        return nil, fmt.Errorf("验证码与用户名不匹配")
    }
    if account.Status != "active" {
        return nil, fmt.Errorf("该网页账号已被禁用")
    }

    resetCode.Used = true
    resetCode.UsedAt = time.Now()
    resetCode.UsedByAccount = account.Username
    if err := db.Save(resetCode).Error; err != nil {
        return nil, fmt.Errorf("锁定重置验证码失败：%v", err)
    }

    return account, nil
}
