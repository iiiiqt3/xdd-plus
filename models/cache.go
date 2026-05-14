package models

import (
	"github.com/beego/beego/v2/core/logs"
	"time"
)

type Cache struct {
	Ckey     string `gorm:"column:CKey;primaryKey"`
	Type     string
	Cvalue   string
	ActiveAt int64
}

func GetCache(key string) (value string) {
	u := &Cache{}

	err := db.Where("ckey = ? and active_at > ?", key, time.Now().Unix()).First(&u).Error
	if err == nil {
		return u.Cvalue
	} else {
		return ""
	}
}

func SaveCache(key string, value string) (flag bool) {
	u := &Cache{}
	err := db.Where("ckey = ?", key).First(&u).Error
	if err == nil {
		logs.Info("为空不报错")
		if u.Cvalue != "" {
			db.Where("ckey = ?", u.Ckey).Updates(&Cache{
				Cvalue:   value,
				ActiveAt: time.Now().Unix() + 3600,
			})
			return true
		} else {
			u.ActiveAt = time.Now().Unix() + 3600
			u.Ckey = key
			u.Cvalue = value
			begin := db.Begin()
			begin.Create(u)
			begin.Commit()
			return true
		}
	} else {
		u.ActiveAt = time.Now().Unix() + 3600
		u.Ckey = key
		u.Cvalue = value
		begin := db.Begin()
		begin.Create(u)
		begin.Commit()
		return true
	}

}
