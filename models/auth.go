package models

type Auth struct {
	Tel string `json:"tel"`
}

func GetAuth(tel string) bool {
	ck := &Auth{}
	tx := db.Where("tel = ?", tel).First(ck).Error
	if tx == nil {
		return true
	}
	return false
}

func AddAuth(tel string) bool {
	ck := &Auth{
		Tel: tel,
	}
	tx := db.Begin()
	if err := tx.Create(ck).Error; err != nil {
		tx.Rollback()
		return true
	} else {
		return false
	}
}

func Remove(tel string) bool {
	ck := &Auth{
		Tel: tel,
	}
	tx := db.Where("tel = ?", tel).Delete(ck).Error
	if tx == nil {
		return true
	}
	return false
}
