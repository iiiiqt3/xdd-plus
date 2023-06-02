package models

var Sys SystemConfig

type SystemConfig struct {
	NolanUrl       string
	NolanToken     string
	RabbitUrl      string
	RabbitApiToken string
	RabbitToken    string
	BBKWxUrl       string
	BBKJdUrl       string
	BBKToken       string
}

func ListConfig() SystemConfig {
	var config SystemConfig
	db.Find(&config)
	Sys = config

	return config
}
