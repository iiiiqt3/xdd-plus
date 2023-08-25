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
