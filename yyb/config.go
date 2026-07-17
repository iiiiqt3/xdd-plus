package yyb

import (
	"time"

	"gorm.io/gorm"
)

// Config 应用宝模块配置
type Config struct {
	Enabled            bool
	ResourceRoot       string
	DBFilename         string
	GormDB             *gorm.DB
	TCPProxy           string
	SessionTTL         time.Duration
	RequestTimeout     time.Duration
	AvatarTimeout      time.Duration
	ScanTimeout        time.Duration
	QRSessionTTL       time.Duration
	Proxy51Enabled           bool
	Proxy51BusinessEnabled   bool
}

func DefaultConfig() Config {
	return Config{
		Enabled:            false,
		ResourceRoot:       "yyb/resource",
		DBFilename:         "yyb.db",
		SessionTTL:         30 * time.Minute,
		RequestTimeout:     8 * time.Second,
		AvatarTimeout:      10 * time.Second,
		ScanTimeout:        180 * time.Second,
		QRSessionTTL:       5 * time.Minute,
	}
}
