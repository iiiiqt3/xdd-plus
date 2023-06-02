package models

import (
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
	db.AutoMigrate(
		&JdCookie{},
		&User{},
		&Env{},
		&Token{},
		&UserAdmin{},
		&Cache{},
		&Key{},
		&BakJdCookie{},
		&SystemConfig{},
		//&LoginSelectType{},
	)

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
	ID        int    `gorm:"column:ID;primaryKey"`
	Priority  int    `gorm:"column:Priority;default:1"`
	CreateAt  string `gorm:"column:CreateAt"`
	LoseAt    string `gorm:"column:LoseAt"`
	UpdateAt  string `gorm:"column:UpdateAt"`
	PtKey     string `gorm:"column:PtKey"`
	PtPin     string `gorm:"column:PtPin;unique"`
	WsKey     string `gorm:"column:WsKey"`
	RWskey    string `gorm:"column:RWsKey"`
	Note      string `gorm:"column:Note"`
	Available string `gorm:"column:Available;default:true" validate:"oneof=true false"`
	Nickname  string `gorm:"column:Nickname"`
	BeanNum   string `gorm:"column:BeanNum"`
	QQ        int    `gorm:"column:QQ"`
	WeiXin    string `gorm:"column:WeiXin"`
	PushPlus  string `gorm:"column:PushPlus"`
	WxPush    string `gorm:"column:WxPush"`
	Telegram  int    `gorm:"column:Telegram"`
	Tyt       string `gorm:"column:Tyt;default:true" validate:"oneof=true false"`
	Dig       string `gorm:"column:Dig;default:true" validate:"oneof=true false"`
	Help      string `gorm:"column:Help;default:false" validate:"oneof=true false"`
	Pool      string `gorm:"-"`
	Hack      string `gorm:"column:Hack"  validate:"oneof=true false"`
	UserLevel string `gorm:"column:UserLevel"`
	LevelName string `gorm:"column:LevelName"`
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
var Hack = "Hack"
var Tyt = "Tyt"
var Dig = "Dig"

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
	if ck.Hack == "" {
		ck.Hack = False
	}
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
	if ck.Hack == "" {
		ck.Hack = False
	}
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
				Hack:  False,
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
