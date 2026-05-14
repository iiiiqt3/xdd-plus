package models

import "github.com/beego/beego/v2/core/logs"

type BakJdCookie struct {
	ID     int    `gorm:"column:ID;primaryKey"`
	PtKey  string `gorm:"column:PtKey"`
	PtPin  string `gorm:"column:PtPin;unique"`
	WsKey  string `gorm:"column:WsKey"`
	RWskey string `gorm:"column:RWsKey"`
}

func AutoBak() {
	cks := GetJdCookies()
	for _, ck := range cks {
		if nck, err := GetBakJdCookie(ck.PtPin); err == nil {
			nck.Updates(BakJdCookie{
				PtKey:  ck.PtKey,
				WsKey:  ck.WsKey,
				RWskey: ck.RWskey,
			})
		} else {
			NewBakJdCookie(&BakJdCookie{
				PtKey:  ck.PtKey,
				PtPin:  ck.PtPin,
				WsKey:  ck.WsKey,
				RWskey: ck.RWskey,
			})
		}
	}
}

func Reduction() {
	var cks []BakJdCookie
	db.Find(&cks)
	for _, ck := range cks {
		if nck, err := GetJdCookie(ck.PtPin); err == nil {
			nck.Updates(JdCookie{
				PtKey:  ck.PtKey,
				WsKey:  ck.WsKey,
				RWskey: ck.RWskey,
			})
		} else {
			logs.Info("已删除该pin")
		}
	}

}

func GetBakJdCookie(pin string) (*BakJdCookie, error) {
	ck := &BakJdCookie{}
	return ck, db.Where(PtPin+" = ?", pin).First(ck).Error
}

func (ck *BakJdCookie) Updates(values interface{}) {
	if ck.ID != 0 {
		db.Model(ck).Updates(values)
		return
	}
	if ck.PtPin != "" {
		db.Model(ck).Where(PtPin+" = ?", ck.PtPin).Updates(values)
		return
	}
}

func NewBakJdCookie(ck *BakJdCookie) error {
	tx := db.Begin()
	if err := tx.Create(ck).Error; err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit().Error
}
