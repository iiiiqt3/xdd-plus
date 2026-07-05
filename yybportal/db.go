package yybportal

import (
	"fmt"
	"strings"

	"github.com/cdle/xdd/models"
	"gorm.io/gorm"
)

func migrate() error {
	gdb := models.GormDB()
	if err := gdb.AutoMigrate(&PortalYybBinding{}); err != nil {
		return err
	}
	if err := dedupePortalBindings(gdb); err != nil {
		return fmt.Errorf("清理门户绑定重复数据: %w", err)
	}
	return nil
}

// dedupePortalBindings 删除同一用户+open_id 的重复绑定（保留 id 最大的一条）
func dedupePortalBindings(gdb *gorm.DB) error {
	var all []PortalYybBinding
	if err := gdb.Unscoped().Order("id asc").Find(&all).Error; err != nil {
		return err
	}
	if len(all) <= 1 {
		return nil
	}
	keep := make(map[string]int64, len(all))
	var deleteIDs []int64
	for _, b := range all {
		key := fmt.Sprintf("%d:%s", b.UserNumber, strings.ToLower(strings.TrimSpace(b.OpenID)))
		if key == "0:" || strings.HasSuffix(key, ":") {
			continue
		}
		if prev, ok := keep[key]; ok {
			if b.ID > prev {
				deleteIDs = append(deleteIDs, prev)
				keep[key] = b.ID
			} else {
				deleteIDs = append(deleteIDs, b.ID)
			}
		} else {
			keep[key] = b.ID
		}
	}
	if len(deleteIDs) == 0 {
		return nil
	}
	return gdb.Unscoped().Delete(&PortalYybBinding{}, deleteIDs).Error
}

func db() *gorm.DB {
	return models.GormDB()
}
