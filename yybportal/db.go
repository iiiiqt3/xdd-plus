package yybportal

import (
	"github.com/cdle/xdd/models"
	"gorm.io/gorm"
)

func migrate() error {
	return models.GormDB().AutoMigrate(&PortalYybBinding{})
}

func db() *gorm.DB {
	return models.GormDB()
}
