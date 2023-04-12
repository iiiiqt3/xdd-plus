package models

type Env struct {
	ID    int
	Name  string `gorm:"unique"`
	Value string
}

func InitEnv() {

}

func ExportEnv(env *Env) {
	value := env.Value
	if err := db.Where("name = ?", env.Name).First(env).Error; err != nil {
		db.Create(env)
	} else {
		db.Model(env).Update("value", value)
	}
}

func UnExportEnv(env *Env) {
	db.Where("name = ?", env.Name).Delete(env)
}

func GetEnvs() []Env {
	envs := []Env{}
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
