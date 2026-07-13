package models

import (
    "crypto/rand"
    "crypto/sha256"
    "encoding/hex"
    "fmt"
    "strconv"
    "strings"
    "time"
)

type WebUserAccount struct {
    ID           int       `gorm:"primaryKey"`
    Username     string    `gorm:"size:64;uniqueIndex"`
    PasswordHash string    `gorm:"size:64"`
    PasswordSalt string    `gorm:"size:32"`
    PasswordPlain string   `gorm:"column:password_plain;size:64"` // 管理员可查，注册/重置时写入
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

func applyWebAccountPassword(account *WebUserAccount, username, password, salt string) {
    account.PasswordSalt = salt
    account.PasswordHash = buildWebPasswordHash(username, password, salt)
    account.PasswordPlain = password
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
        Username:   username,
        UserNumber: user.Number,
        Status:     "active",
        BoundAt:    now,
        LastLoginAt: now,
    }
    applyWebAccountPassword(account, username, password, salt)

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
    applyWebAccountPassword(&account, username, password, salt)
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

// AdminWebAccountLookupItem 管理端网页账号查询/列表结果
type AdminWebAccountLookupItem struct {
	AccountID     int    `json:"accountId"`
	UserID        int    `json:"userId"`
	UserNumber    int    `json:"userNumber"`
	Nickname      string `json:"nickname"`
	QQ            string `json:"qq"`
	Wxid          string `json:"wxid"`
	Coin          int    `json:"coin"`
	Username      string `json:"username"`
	HasWebAccount bool   `json:"hasWebAccount"`
	Password      string `json:"password"`
	PasswordNote  string `json:"passwordNote"`
	Status        string `json:"status"`
	BoundAt       string `json:"boundAt,omitempty"`
	LastLoginAt   string `json:"lastLoginAt,omitempty"`
}

func adminWebAccountListItem(account WebUserAccount, user *User) AdminWebAccountLookupItem {
	item := AdminWebAccountLookupItem{
		AccountID:     account.ID,
		Username:      account.Username,
		UserNumber:    account.UserNumber,
		HasWebAccount: true,
		Status:        account.Status,
		PasswordNote:  "",
	}
	if strings.TrimSpace(account.PasswordPlain) != "" {
		item.Password = account.PasswordPlain
	} else {
		item.PasswordNote = "历史账号无密码存档，请重置后可查看"
	}
	if !account.BoundAt.IsZero() {
		item.BoundAt = account.BoundAt.Format("2006-01-02 15:04:05")
	}
	if !account.LastLoginAt.IsZero() {
		item.LastLoginAt = account.LastLoginAt.Format("2006-01-02 15:04:05")
	}
	if user != nil {
		item.UserID = user.ID
		item.Nickname = user.Nickname
		item.QQ = user.QQ
		item.Wxid = user.Wxid
		item.Coin = user.Coin
	}
	return item
}

// ListWebUserAccountsForAdmin 分页列出全部网页注册账号
func ListWebUserAccountsForAdmin(search string, page, limit int) ([]AdminWebAccountLookupItem, int, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	search = strings.TrimSpace(search)

	var accounts []WebUserAccount
	var total int64
	tx := db.Model(&WebUserAccount{})
	if search != "" {
		like := "%" + search + "%"
		var userNumbers []int
		db.Model(&User{}).Where(
			"nickname LIKE ? OR wxid LIKE ? OR qq LIKE ? OR CAST(number AS CHAR) LIKE ?",
			like, like, like, like,
		).Pluck("number", &userNumbers)
		if len(userNumbers) > 0 {
			tx = tx.Where(
				"username LIKE ? OR CAST(user_number AS CHAR) LIKE ? OR user_number IN ?",
				like, like, userNumbers,
			)
		} else {
			tx = tx.Where("username LIKE ? OR CAST(user_number AS CHAR) LIKE ?", like, like)
		}
	}
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * limit
	if err := tx.Order("id DESC").Offset(offset).Limit(limit).Find(&accounts).Error; err != nil {
		return nil, 0, err
	}

	out := make([]AdminWebAccountLookupItem, 0, len(accounts))
	userCache := map[int]*User{}
	for _, account := range accounts {
		var user *User
		if cached, ok := userCache[account.UserNumber]; ok {
			user = cached
		} else {
			var row User
			if db.Where("number = ?", account.UserNumber).First(&row).Error == nil {
				user = &row
				userCache[account.UserNumber] = user
			}
		}
		out = append(out, adminWebAccountListItem(account, user))
	}
	return out, int(total), nil
}

func generateAdminWebPassword() (string, error) {
	const chars = "abcdefghjkmnpqrstuvwxyzABCDEFGHJKMNPQRSTUVWXYZ23456789"
	buf := make([]byte, 10)
	raw := make([]byte, 10)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	for i, b := range raw {
		buf[i] = chars[int(b)%len(chars)]
	}
	return string(buf), nil
}

// LookupWebUserAccountsForAdmin 按用户名/编号/QQ/微信/昵称等查询网页账号
func LookupWebUserAccountsForAdmin(keyword string) ([]AdminWebAccountLookupItem, error) {
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return nil, fmt.Errorf("请输入查询条件")
	}

	seen := map[int]bool{}
	out := make([]AdminWebAccountLookupItem, 0, 4)
	appendUser := func(user User, account *WebUserAccount) {
		if seen[user.Number] {
			return
		}
		seen[user.Number] = true
		item := AdminWebAccountLookupItem{
			UserID:        user.ID,
			UserNumber:    user.Number,
			Nickname:      user.Nickname,
			QQ:            user.QQ,
			Wxid:          user.Wxid,
			Coin:          user.Coin,
			HasWebAccount: account != nil,
			Password:      "",
			PasswordNote:  "",
		}
		if account != nil {
			item.Username = account.Username
			item.AccountID = account.ID
			item.Status = account.Status
			if strings.TrimSpace(account.PasswordPlain) != "" {
				item.Password = account.PasswordPlain
				item.PasswordNote = ""
			} else {
				item.PasswordNote = "历史账号无密码存档，请重置后可查看"
			}
			if !account.BoundAt.IsZero() {
				item.BoundAt = account.BoundAt.Format("2006-01-02 15:04:05")
			}
			if !account.LastLoginAt.IsZero() {
				item.LastLoginAt = account.LastLoginAt.Format("2006-01-02 15:04:05")
			}
		} else {
			item.PasswordNote = "该用户尚未注册网页登录账号"
		}
		out = append(out, item)
	}

	normalized := normalizeWebUsername(keyword)
	var directAccount WebUserAccount
	if normalized != "" {
		if err := db.Where("username = ?", normalized).First(&directAccount).Error; err == nil {
			var user User
			if db.Where("number = ?", directAccount.UserNumber).First(&user).Error == nil {
				appendUser(user, &directAccount)
			}
		}
	}

	if num, err := strconv.Atoi(keyword); err == nil && num > 0 {
		var user User
		if db.Where("number = ?", num).First(&user).Error == nil {
			account, _ := GetWebUserAccountByUserNumber(user.Number)
			appendUser(user, account)
		}
		var accountByNum WebUserAccount
		if err := db.Where("user_number = ?", num).First(&accountByNum).Error; err == nil {
			var user User
			if db.Where("number = ?", accountByNum.UserNumber).First(&user).Error == nil {
				appendUser(user, &accountByNum)
			}
		}
	}

	like := "%" + keyword + "%"
	var users []User
	db.Where("nickname LIKE ? OR wxid LIKE ? OR qq LIKE ? OR CAST(number AS CHAR) LIKE ?",
		like, like, like, like).Order("number ASC").Limit(20).Find(&users)
	for _, user := range users {
		account, _ := GetWebUserAccountByUserNumber(user.Number)
		appendUser(user, account)
	}

	if len(out) == 0 {
		return nil, fmt.Errorf("未找到匹配用户或网页账号")
	}
	return out, nil
}

// AdminResetWebUserPasswordForAdmin 管理员重置网页账号密码并返回新密码
func AdminResetWebUserPasswordForAdmin(username string, userNumber int, newPassword string, autoGenerate bool) (map[string]any, error) {
	username = normalizeWebUsername(username)
	var account *WebUserAccount
	var err error

	if username != "" {
		account, err = func() (*WebUserAccount, error) {
			var row WebUserAccount
			if e := db.Where("username = ?", username).First(&row).Error; e != nil {
				return nil, e
			}
			return &row, nil
		}()
	} else if userNumber > 0 {
		account, err = GetWebUserAccountByUserNumber(userNumber)
	} else {
		return nil, fmt.Errorf("请提供网页账号或用户编号")
	}
	if err != nil || account == nil {
		return nil, fmt.Errorf("未找到网页登录账号")
	}

	password := strings.TrimSpace(newPassword)
	if autoGenerate || password == "" {
		password, err = generateAdminWebPassword()
		if err != nil {
			return nil, fmt.Errorf("生成新密码失败：%v", err)
		}
	}
	if err := validateWebPassword(password); err != nil {
		return nil, err
	}

	updated, err := ResetWebUserPassword(account.Username, password)
	if err != nil {
		return nil, err
	}

	var user User
	_ = db.Where("number = ?", updated.UserNumber).First(&user).Error

	return map[string]any{
		"username":   updated.Username,
		"password":   password,
		"userNumber": updated.UserNumber,
		"nickname":   user.Nickname,
		"qq":         user.QQ,
		"wxid":       user.Wxid,
	}, nil
}
