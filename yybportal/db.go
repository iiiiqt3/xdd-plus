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

// dedupePortalBindings 删除同一用户+open_id 的重复绑定（优先保留未删除、id 最大的一条）
func dedupePortalBindings(gdb *gorm.DB) error {
	var all []PortalYybBinding
	if err := gdb.Unscoped().Order("id asc").Find(&all).Error; err != nil {
		return err
	}
	if len(all) <= 1 {
		return nil
	}
	type slot struct {
		keepID int64
		active bool
	}
	keep := make(map[string]slot, len(all))
	var deleteIDs []int64
	for _, b := range all {
		openid := strings.ToLower(strings.TrimSpace(b.OpenID))
		if openid == "" {
			continue
		}
		key := fmt.Sprintf("%d:%s", b.UserNumber, openid)
		active := !b.DeletedAt.Valid
		prev, ok := keep[key]
		if !ok {
			keep[key] = slot{keepID: b.ID, active: active}
			continue
		}
		replace := false
		switch {
		case active && !prev.active:
			replace = true
		case active == prev.active && b.ID > prev.keepID:
			replace = true
		}
		if replace {
			deleteIDs = append(deleteIDs, prev.keepID)
			keep[key] = slot{keepID: b.ID, active: active}
		} else {
			deleteIDs = append(deleteIDs, b.ID)
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
