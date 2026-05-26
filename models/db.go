package models

import (
	"log"
	"strconv"
	"strings"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var db *gorm.DB

var keys map[string]bool
var pins map[string]bool

func initDB() {
	var err error
	var c = &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	}

	if strings.Contains(Config.Database, "@tcp(") {
		db, err = gorm.Open(mysql.Open(Config.Database), c)
	} else if strings.Contains(Config.Database, "dbname=") {
		db, err = gorm.Open(postgres.Open(Config.Database), c)
	} else {
		db, err = gorm.Open(sqlite.Open(Config.Database), c)
	}
	if err != nil {
		panic(err)
	}
	if err := db.AutoMigrate(
		&JdCookie{},
		&User{},
		&WebUserAccount{},
		&RegisterBindCode{},
		&PasswordResetCode{},
		&Env{},
		&Token{},
		&UserAdmin{},
		&Cache{},
		&Key{},
		&Auth{},
		&Limit{},
		&Wish{},
		&WebNotification{},
		&WebNotificationRead{},
		&PortalPrayRecord{},
		&AppFeedback{},
		&PortalWxDevice{},
		//&LoginSelectType{},
	); err != nil {
		log.Printf("[数据库迁移] AutoMigrate 失败: %v", err)
	}

	createActivityProjectTable()

	keys = make(map[string]bool)
	pins = make(map[string]bool)
	var jps []JdCookie
	db.Find(&jps)
	for _, jp := range jps {
		keys[jp.PtKey] = true
		pins[jp.PtPin] = true
	}
}

func HasPin(pin string) bool {
	if _, ok := pins[pin]; ok {
		return ok
	}
	pins[pin] = true
	return false
}

func HasKey(key string) bool {
	if _, ok := keys[key]; ok {
		return ok
	}
	keys[key] = true
	return false
}

func HasWsKey(key string) bool {
	if _, ok := keys[key]; ok {
		return ok
	}
	keys[key] = true
	return false
}

type Wish struct {
	ID         int
	CreatedAt  time.Time
	UserNumber int
	Content    string
	Coin       int
	Status     int // 1 2
}

type JdCookie struct {
	ID              int    `gorm:"column:ID;primaryKey"`
	Priority        int    `gorm:"column:Priority;default:1"`
	CreateAt        string `gorm:"column:CreateAt"`
	LoseAt          string `gorm:"column:LoseAt"`
	UpdateAt        string `gorm:"column:UpdateAt"`
	PtKey           string `gorm:"column:PtKey"`
	PtPin           string `gorm:"column:PtPin;unique"`
	WsKey           string `gorm:"column:WsKey"`
	RWskey          string `gorm:"column:RWsKey"`
	Note            string `gorm:"column:Note"`
	Available       string `gorm:"column:Available;default:true" validate:"oneof=true false"`
	Nickname        string `gorm:"column:Nickname"`
	BeanNum         string `gorm:"column:BeanNum"`
	QQ              int    `gorm:"column:QQ"`
	WeiXin          string `gorm:"column:WeiXin"`
	Hack            string `gorm:"column:Hack"`
	Appoint         string `gorm:"column:Appoint"` //指定
	PushPlus        string `gorm:"column:PushPlus"`
	WxPush          string `gorm:"column:WxPush"`
	Telegram        int    `gorm:"column:Telegram"`
	Pool            string `gorm:"-"`
	UserID          int    `gorm:"column:UserId"`
	UserLevel       string `gorm:"column:UserLevel"`
	LevelName       string `gorm:"column:LevelName"`
	Account         string `gorm:"column:Account"`
	Password        string `gorm:"column:Password"`
	Smsverify       string `gorm:"column:Smsverify;default:false" validate:"oneof=true false"`
	IsApp           string `gorm:"column:IsApp"`
	NoticeNum       int    `gorm:"column:NoticeNum;default:4"`
	Socks5_Ip       string `gorm:"column:Socks5_Ip"`
	Socks5_Port     string `gorm:"column:Socks5_Port"`
	Socks5_Account  string `gorm:"column:Socks5_Account"`
	Socks5_Password string `gorm:"column:Socks5_Password"`
}

var UserLevel = "UserLevel"
var LevelName = "LevelName"
var ScanedAt = "ScanedAt"
var LoseAt = "LoseAt"
var CreateAt = "CreateAt"
var Note = "Note"
var Available = "Available"
var UnAvailable = "UnAvailable"
var PtKey = "PtKey"
var PtPin = "PtPin"
var Content = "Content"
var WsKey = "WsKey"
var Address = "Address"
var Priority = "Priority"
var Nickname = "Nickname"
var BeanNum = "BeanNum"
var Pool = "Pool"
var True = "true"
var False = "false"
var QQ = "QQ"
var RWSKEY = "RWsKey"
var PushPlus = "PushPlus"
var Save chan *JdCookie
var ExecPath string
var Telegram = "Telegram"
var Tyt = "Tyt"
var Dig = "Dig"
var Account = "Account"
var Password = "Password"
var Smsverify = "Smsverify"
var IsApp = "IsApp"
var Hack = "Hack"
var NoticeNum = "NoticeNum"
var Appoint = "Appoint"
var Socks5_Ip = "Socks5_Ip"
var Socks5_Port = "Socks5_Port"
var Socks5_Account = "Socks5_Account"
var Socks5_Password = "Socks5_Password"

func Date() string {
	return time.Now().Local().Format("2006-01-02")
}

func GetJdCookies(sbs ...func(sb *gorm.DB) *gorm.DB) []JdCookie {
	var cks []JdCookie
	tb := db
	for _, sb := range sbs {
		tb = sb(tb)
	}
	tb.Order("priority desc,ID asc").Find(&cks)
	return cks
}

func GetJdCookie(pin string) (*JdCookie, error) {
	ck := &JdCookie{}
	return ck, db.Where(PtPin+" = ?", pin).First(ck).Error
}

func (ck *JdCookie) Updates(values interface{}) {
	if ck.ID != 0 {
		db.Model(ck).Updates(values)
		return
	}
	if ck.PtPin != "" {
		db.Model(ck).Where(PtPin+" = ?", ck.PtPin).Updates(values)
		return
	}
}

func (ck *JdCookie) Update(column string, value interface{}) {
	if ck.ID != 0 {
		db.Model(ck).Update(column, value)
		return
	}
	if ck.PtPin != "" {
		db.Model(JdCookie{}).Where(PtPin+" = ?", ck.PtPin).Update(column, value)
		return
	}
}

func (ck *JdCookie) Removes(values interface{}) {
	if ck.ID != 0 {
		db.Model(ck).Delete(values)
	}
	if ck.PtPin != "" {
		db.Model(ck).Where(PtPin+" = ?", ck.PtPin).Delete(values)
	}
}

func NewJdCookie(ck *JdCookie) error {

	ck.Priority = Config.DefaultPriority
	date := Date()
	ck.CreateAt = date
	ck.UpdateAt = date
	tx := db.Begin()
	if err := tx.Create(ck).Error; err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit().Error
}

func UpdateCookie(ck *JdCookie) error {

	ck.Priority = Config.DefaultPriority
	date := Date()
	ck.CreateAt = date
	ck.UpdateAt = date
	tx := db.Begin()
	if err := tx.Where(PtPin+" = ?", ck.PtPin).Updates(ck).Error; err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit().Error
}

func CheckIn(pin, key string) int {
	if !strings.Contains(key, "app_open") {
		if !HasPin(pin) {
			NewJdCookie(&JdCookie{
				PtKey: key,
				PtPin: pin,
			})
			return 0
		} else if !HasKey(key) {
			ck, _ := GetJdCookie(pin)
			ck.PtKey = key
			ck.Updates(JdCookie{PtKey: key})
			return 1
		}
	}
	return 2
}

func IsUser(qq int64) bool {

	var u []JdCookie
	return db.Where("QQ = ?", qq).First(&u).Error == nil
}

func createActivityProjectTable() {
	var count int64
	if err := db.Raw("SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = 'activity_project'").Scan(&count).Error; err != nil {
		log.Printf("[数据库迁移] 检查表失败: %v", err)
		return
	}
	if count > 0 {
		log.Println("[数据库迁移] activity_project 表已存在，检查字段类型...")
		var colType string
		db.Raw("SELECT DATA_TYPE FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = 'activity_project' AND column_name = 'user_number'").Scan(&colType)
		if colType == "int" {
			if err := db.Exec("ALTER TABLE activity_project MODIFY COLUMN user_number BIGINT NOT NULL DEFAULT 0").Error; err != nil {
				log.Printf("[数据库迁移] 修改 user_number 字段为 BIGINT 失败: %v", err)
			} else {
				log.Println("[数据库迁移] user_number 字段已改为 BIGINT")
			}
		} else {
			log.Printf("[数据库迁移] user_number 字段已是 %s，无需修改", colType)
		}

		// 自动迁移：将 activity_id 从旧的连续编号（"1","2","3"...）更新为 EnvKey
		// 仅当 activity_id 是纯数字且与 env_key 不同时才执行，确保幂等（多次启动不会重复执行）
		var oldStyleProjects []ActivityProject
		if err := db.Where("activity_id != env_key AND deleted_at IS NULL").Find(&oldStyleProjects).Error; err == nil {
			var needMigrate []ActivityProject
			for _, p := range oldStyleProjects {
				if isNumeric(p.ActivityID) && p.ActivityID != p.EnvKey {
					needMigrate = append(needMigrate, p)
				}
			}
			if len(needMigrate) > 0 {
				tx := db.Begin()
				migrated := 0
				for _, p := range needMigrate {
					if err := tx.Model(&ActivityProject{}).Where("id = ?", p.ID).Update("activity_id", p.EnvKey).Error; err != nil {
						log.Printf("[数据库迁移] 更新记录 ID=%d 失败: %v", p.ID, err)
						continue
					}
					migrated++
				}
				if err := tx.Commit().Error; err != nil {
					tx.Rollback()
					log.Printf("[数据库迁移] activity_id 迁移事务提交失败: %v", err)
				} else {
					log.Printf("[数据库迁移] activity_id 已从旧编号迁移为 EnvKey，共更新 %d 条记录", migrated)
				}
			} else {
				log.Println("[数据库迁移] activity_id 无需迁移（已是 EnvKey 或无数据）")
			}
		}
		return
	}

	sql := `
CREATE TABLE activity_project (
	id INT AUTO_INCREMENT PRIMARY KEY,
	activity_id VARCHAR(64) NOT NULL DEFAULT '',
	activity_name VARCHAR(128) NOT NULL DEFAULT '',
	env_key VARCHAR(128) NOT NULL DEFAULT '',
	env_value TEXT,
	remarks TEXT,
	remark_alias VARCHAR(128) NOT NULL DEFAULT '',
	user_number BIGINT NOT NULL DEFAULT 0,
	qinglong_config_name VARCHAR(64) NOT NULL DEFAULT '',
	qinglong_env_id INT NOT NULL DEFAULT 0,
	status INT NOT NULL DEFAULT 0,
	expire_date VARCHAR(10) NOT NULL DEFAULT '',
	is_monthly_deduct TINYINT(1) NOT NULL DEFAULT 0,
	monthly_coin INT NOT NULL DEFAULT 0,
	need_coin INT NOT NULL DEFAULT 0,
	created_at DATETIME,
	updated_at DATETIME,
	deleted_at DATETIME,
	sync_status VARCHAR(16) NOT NULL DEFAULT 'pending',
	sync_error TEXT,
	sync_at DATETIME,
	INDEX idx_activity_id (activity_id),
	INDEX idx_env_key (env_key),
	INDEX idx_remarks (remarks(255)),
	INDEX idx_user_number (user_number),
	INDEX idx_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`
	if err := db.Exec(sql).Error; err != nil {
		log.Printf("[数据库迁移] 创建 activity_project 表失败: %v", err)
		return
	}
	log.Println("[数据库迁移] activity_project 表创建成功")
}

// isNumeric 判断字符串是否为纯数字（用于识别旧版连续编号格式的 activity_id）
func isNumeric(s string) bool {
	if s == "" {
		return false
	}
	_, err := strconv.Atoi(s)
	return err == nil
}
