package yybportal

import (
	"fmt"

	"github.com/cdle/xdd/models"
	"gorm.io/gorm"
)

func migrate() error {
	gdb := models.GormDB()
	if err := dedupePortalBindings(gdb); err != nil {
		return fmt.Errorf("清理门户绑定重复数据: %w", err)
	}
	return gdb.AutoMigrate(&PortalYybBinding{})
}

// dedupePortalBindings 迁移前删除同一用户+openid 的重复绑定
func dedupePortalBindings(gdb *gorm.DB) error {
	sql := `
DELETE t1 FROM portal_yyb_bindings t1
INNER JOIN portal_yyb_bindings t2
  ON t1.user_number = t2.user_number
 AND LOWER(t1.openid) = LOWER(t2.openid)
 AND t1.id < t2.id
WHERE t1.deleted_at IS NULL AND t2.deleted_at IS NULL`
	return gdb.Exec(sql).Error
}

func db() *gorm.DB {
	return models.GormDB()
}
