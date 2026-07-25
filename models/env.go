package models

import (
	"errors"
	"strings"

	"gorm.io/gorm"
)

type Env struct {
	ID    int
	Name  string `gorm:"unique;type:varchar(128)"`
	Value string `gorm:"type:longtext"`
	Note  string `gorm:"type:varchar(255)"`
}

func InitEnv() {

}

func ExportEnv(env *Env) error {
	if db == nil {
		return errors.New("database not initialized")
	}
	if env == nil {
		return errors.New("env is nil")
	}
	name := strings.TrimSpace(env.Name)
	if name == "" {
		return errors.New("env name is empty")
	}
	value := env.Value
	var existing Env
	err := db.Where("name = ?", name).First(&existing).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return db.Create(&Env{Name: name, Value: value, Note: env.Note}).Error
		}
		return err
	}
	updates := map[string]interface{}{"value": value}
	if strings.TrimSpace(env.Note) != "" {
		updates["note"] = env.Note
	}
	return db.Model(&Env{}).Where("id = ?", existing.ID).Updates(updates).Error
}

func UnExportEnv(env *Env) {
	db.Where("name = ?", env.Name).Delete(env)
}

func GetEnvs() []Env {
	var envs []Env
	db.Find(&envs)
	return envs
}

func GetEnv(name string) string {
	env := &Env{}
	db.Where("name = ?", name).First(env)
	return env.Value
}

func IsAutoAgreeFriendVerify() bool {
	env := &Env{}
	db.Where("name = ?", "AutoAgree").First(env)
	if env.Value == "" {
		return false
	}
	return true
}

	func IsAutoAgreeAutocollection() bool {
	env := &Env{}
	db.Where("name = ?", "Autocollection").First(env)
	if env.Value == "" {
		return false
	}
	return true
}


func UseAgreeMsg() bool {
	env := &Env{}
	db.Where("name = ?", "AgreeMsg").First(env)
	if env.Value == "" {
		return false
	}
	return true
}

func isOpenWskey() bool {
	env := &Env{}
	db.Where("name = ?", "CloseWskey").First(env)
	if env.Value == "" {
		return true
	}
	return false
}

func isOpenImg() bool {
	env := &Env{}
	db.Where("name = ?", "img").First(env)
	if env.Value == "" {
		return false
	}
	return true
}
