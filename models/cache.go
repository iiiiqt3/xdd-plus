package models

import (
	"github.com/beego/beego/v2/core/logs"
	"time"
)

type Cache struct {
	key      string `gorm:"column:Key;primaryKey"`
	Type     string
	value    string
	ActiveAt int64
}

func Dtime() {

}

func getCache(key string) (value string) {
	u := &Cache{}
	//format := "2006-01-02 15:04:05"

	err := db.Where("key = ? and active_at > ?", key, time.Now().UnixMilli()).First(&u).Error
	if err == nil {
		return u.value
	} else {
		return ""
	}
}

func saveCache(key string, value string) (flag bool) {
	u := &Cache{}
	err := db.Where("key = ?", key).First(&u).Error
	if err == nil {
		logs.Info("为空不报错")
		if u.value != "" {
			db.Where("key = ?", u.key).Updates(&Cache{
				value:    value,
				ActiveAt: time.Now().UnixMilli() + 3600,
			})
			return true
		} else {
			u.ActiveAt = time.Now().UnixMilli() + 3600
			u.key = key
			u.value = value
			begin := db.Begin()
			begin.Create(u)
			begin.Commit()
			return true
		}
	} else {
		return false
	}

}
