package models

import (
    "crypto/rand"
    "crypto/sha256"
    "encoding/hex"
    "fmt"
    "strings"
    "time"
)

type WebUserAccount struct {
    ID           int       `gorm:"primaryKey"`
    Username     string    `gorm:"size:64;uniqueIndex"`
    PasswordHash string    `gorm:"size:64"`
    PasswordSalt string    `gorm:"size:32"`
    UserNumber   int       `gorm:"uniqueIndex"`
    Status       string    `gorm:"size:16;default:active"`
    BoundAt      time.Time
    LastLoginAt  time.Time
    CreatedAt    time.Time
    UpdatedAt    time.Time
}

func normalizeWebUsername(username string) string {
    return strings.ToLower(strings.TrimSpace(username))
}

func validateWebUsername(username string) error {
    if len(username) < 3 || len(username) > 32 {
        return fmt.Errorf("用户名长度需为3-32位")
    }
    for _, r := range username {
        if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' || r == '-' || r == '.' {
            continue
        }
        return fmt.Errorf("用户名仅支持小写字母、数字、下划线、中划线和点")
    }
    return nil
}

func validateWebPassword(password string) error {
    password = strings.TrimSpace(password)
    if len(password) < 6 || len(password) > 64 {
        return fmt.Errorf("密码长度需为6-64位")
    }
    return nil
}

func generateWebPasswordSalt() (string, error) {
    buf := make([]byte, 16)
    if _, err := rand.Read(buf); err != nil {
        return "", err
    }
    return hex.EncodeToString(buf), nil
}

func buildWebPasswordHash(username, password, salt string) string {
    sum := sha256.Sum256([]byte(salt + ":" + normalizeWebUsername(username) + ":" + password))
    return hex.EncodeToString(sum[:])
}

func CreateWebUserAccount(username, password string, userNumber int) (*WebUserAccount, *User, error) {
    username = normalizeWebUsername(username)
    if err := validateWebUsername(username); err != nil {
        return nil, nil, err
    }
    if err := validateWebPassword(password); err != nil {
        return nil, nil, err
    }
    if userNumber <= 0 {
        return nil, nil, fmt.Errorf("请输入有效的绑定用户")
    }

    var user User
    if err := db.Where("number = ?", userNumber).First(&user).Error; err != nil {
        return nil, nil, fmt.Errorf("未找到对应绑定用户")
    }

    var usernameCount int64
    db.Model(&WebUserAccount{}).Where("username = ?", username).Count(&usernameCount)
    if usernameCount > 0 {
        return nil, nil, fmt.Errorf("该用户名已被注册")
    }

    var bindCount int64
    db.Model(&WebUserAccount{}).Where("user_number = ?", user.Number).Count(&bindCount)
    if bindCount > 0 {
        return nil, nil, fmt.Errorf("该用户编号已绑定网页账号")
    }

    salt, err := generateWebPasswordSalt()
    if err != nil {
        return nil, nil, fmt.Errorf("生成密码盐失败：%v", err)
    }

    now := time.Now()
    account := &WebUserAccount{
        Username:     username,
        PasswordHash: buildWebPasswordHash(username, password, salt),
        PasswordSalt: salt,
        UserNumber:   user.Number,
        Status:       "active",
        BoundAt:      now,
        LastLoginAt:  now,
    }

    if err := db.Create(account).Error; err != nil {
        return nil, nil, fmt.Errorf("创建网页账号失败：%v", err)
    }
    return account, &user, nil
}

func GetWebUserAccountByID(id int) (*WebUserAccount, error) {
    var account WebUserAccount
    if err := db.Where("id = ?", id).First(&account).Error; err != nil {
        return nil, err
    }
    return &account, nil
}

func GetWebUserAccountByUserNumber(userNumber int) (*WebUserAccount, error) {
    var account WebUserAccount
    if err := db.Where("user_number = ?", userNumber).First(&account).Error; err != nil {
        return nil, err
    }
    return &account, nil
}

func ResetWebUserPassword(username, password string) (*WebUserAccount, error) {
    username = normalizeWebUsername(username)
    if username == "" {
        return nil, fmt.Errorf("用户名不能为空")
    }
    if err := validateWebPassword(password); err != nil {
        return nil, err
    }

    var account WebUserAccount
    if err := db.Where("username = ?", username).First(&account).Error; err != nil {
        return nil, fmt.Errorf("网页账号不存在")
    }
    if account.Status != "active" {
        return nil, fmt.Errorf("该账号已被禁用")
    }

    salt, err := generateWebPasswordSalt()
    if err != nil {
        return nil, fmt.Errorf("生成密码盐失败：%v", err)
    }

    account.PasswordSalt = salt
    account.PasswordHash = buildWebPasswordHash(username, password, salt)
    account.UpdatedAt = time.Now()
    if err := db.Save(&account).Error; err != nil {
        return nil, fmt.Errorf("重置网页密码失败：%v", err)
    }
    return &account, nil
}

func AuthenticateWebUserAccount(username, password string) (*WebUserAccount, *User, error) {
    username = normalizeWebUsername(username)
    if username == "" || strings.TrimSpace(password) == "" {
        return nil, nil, fmt.Errorf("用户名和密码不能为空")
    }

    var account WebUserAccount
    if err := db.Where("username = ?", username).First(&account).Error; err != nil {
        return nil, nil, fmt.Errorf("用户名或密码错误")
    }
    if account.Status != "active" {
        return nil, nil, fmt.Errorf("该账号已被禁用")
    }

    expected := buildWebPasswordHash(username, password, account.PasswordSalt)
    if expected != account.PasswordHash {
        return nil, nil, fmt.Errorf("用户名或密码错误")
    }

    var user User
    if err := db.Where("number = ?", account.UserNumber).First(&user).Error; err != nil {
        return nil, nil, fmt.Errorf("账号绑定的用户不存在")
    }

    now := time.Now()
    db.Model(&account).Update("last_login_at", now)
    account.LastLoginAt = now
    return &account, &user, nil
}
