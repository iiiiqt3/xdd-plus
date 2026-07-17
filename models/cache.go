package models

import (
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
	return SaveCacheTTL(key, value, 3600)
}

// SaveCacheTTL 写入缓存，ttlSec 秒后过期
func SaveCacheTTL(key string, value string, ttlSec int) (flag bool) {
	if ttlSec <= 0 {
		ttlSec = 3600
	}
	expireAt := time.Now().Unix() + int64(ttlSec)
	u := &Cache{}
	err := db.Where("ckey = ?", key).First(&u).Error
	if err == nil {
		Info("为空不报错")
		if u.Cvalue != "" {
			db.Where("ckey = ?", u.Ckey).Updates(&Cache{
				Cvalue:   value,
				ActiveAt: expireAt,
			})
			return true
		}
		u.ActiveAt = expireAt
		u.Ckey = key
		u.Cvalue = value
		begin := db.Begin()
		begin.Create(u)
		begin.Commit()
		return true
	}
	u.ActiveAt = expireAt
	u.Ckey = key
	u.Cvalue = value
	begin := db.Begin()
	begin.Create(u)
	begin.Commit()
	return true
}
