package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// OpenGORM 使用 xdd 主库（MySQL 等）存储应用宝数据
func OpenGORM(orm *gorm.DB) (*DB, error) {
	if orm == nil {
		return nil, fmt.Errorf("gorm db is nil")
	}
	if err := orm.AutoMigrate(&GormWechatAccount{}, &GormSession{}, &GormFeature{}); err != nil {
		return nil, err
	}
	db := &DB{orm: orm}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := db.EnsureDefaultFeatures(ctx); err != nil {
		return nil, err
	}
	return db, nil
}

func (db *DB) useGORM() bool { return db != nil && db.orm != nil }

func gormAccountToModel(a *WechatAccount) (*GormWechatAccount, error) {
	userJSON, err := marshalNullable(a.UserInfo)
	if err != nil {
		return nil, err
	}
	credJSON, err := marshalNullable(a.Credentials)
	if err != nil {
		return nil, err
	}
	return &GormWechatAccount{
		ID:            a.ID,
		OpenID:        a.OpenID,
		UIN:           a.UIN,
		Alias:         a.Alias,
		Nickname:      a.Nickname,
		Avatar:        a.Avatar,
		UserInfo:      userJSON.String,
		LoginBuffer:   a.LoginBuffer,
		Credentials:   credJSON.String,
		Status:        a.Status,
		LastCheckedAt: a.LastCheckedAt,
		CreatedAt:     a.CreatedAt,
		UpdatedAt:     a.UpdatedAt,
	}, nil
}

func modelToWechatAccount(m *GormWechatAccount) *WechatAccount {
	a := &WechatAccount{
		ID:            m.ID,
		OpenID:        m.OpenID,
		UIN:           m.UIN,
		Alias:         m.Alias,
		Nickname:      m.Nickname,
		Avatar:        m.Avatar,
		LoginBuffer:   m.LoginBuffer,
		Status:        m.Status,
		LastCheckedAt: m.LastCheckedAt,
		CreatedAt:     m.CreatedAt,
		UpdatedAt:     m.UpdatedAt,
	}
	if m.UserInfo != "" {
		_ = json.Unmarshal([]byte(m.UserInfo), &a.UserInfo)
	}
	if m.Credentials != "" {
		_ = json.Unmarshal([]byte(m.Credentials), &a.Credentials)
	}
	return a
}

func (db *DB) gormUpsertAccount(ctx context.Context, openid, loginBuffer string, alias, nickname, avatar *string, userInfo map[string]any, credentials map[string]any, status *string) (*WechatAccount, error) {
	now := time.Now().Unix()
	userJSON, err := marshalNullable(userInfo)
	if err != nil {
		return nil, err
	}
	credJSON, err := marshalNullable(credentials)
	if err != nil {
		return nil, err
	}
	row := GormWechatAccount{
		OpenID:      openid,
		LoginBuffer: loginBuffer,
		Alias:       alias,
		Nickname:    nickname,
		Avatar:      avatar,
		UserInfo:    userJSON.String,
		Credentials: credJSON.String,
		Status:      status,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	err = db.orm.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "open_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"login_buffer", "alias", "nickname", "avatar", "user_info", "credentials", "status", "updated_at",
		}),
	}).Create(&row).Error
	if err != nil {
		return nil, err
	}
	return db.gormGetAccountByOpenID(ctx, openid)
}

func (db *DB) gormGetAccount(ctx context.Context, id int64) (*WechatAccount, error) {
	var m GormWechatAccount
	if err := db.orm.WithContext(ctx).First(&m, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, sql.ErrNoRows
		}
		return nil, err
	}
	return modelToWechatAccount(&m), nil
}

func (db *DB) gormGetAccountByOpenID(ctx context.Context, openid string) (*WechatAccount, error) {
	var m GormWechatAccount
	if err := db.orm.WithContext(ctx).Where(&GormWechatAccount{OpenID: openid}).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, sql.ErrNoRows
		}
		return nil, err
	}
	return modelToWechatAccount(&m), nil
}

func (db *DB) gormGetAccountByUIN(ctx context.Context, uin int64) (*WechatAccount, error) {
	var m GormWechatAccount
	if err := db.orm.WithContext(ctx).Where("uin = ?", uin).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, sql.ErrNoRows
		}
		return nil, err
	}
	return modelToWechatAccount(&m), nil
}

func (db *DB) gormListAccounts(ctx context.Context) ([]*WechatAccount, error) {
	var rows []GormWechatAccount
	if err := db.orm.WithContext(ctx).Order("id asc").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]*WechatAccount, 0, len(rows))
	for i := range rows {
		out = append(out, modelToWechatAccount(&rows[i]))
	}
	return out, nil
}

func (db *DB) gormSetAccountUIN(ctx context.Context, id, uin int64) error {
	return db.orm.WithContext(ctx).Model(&GormWechatAccount{}).Where("id = ?", id).
		Updates(map[string]any{"uin": uin, "updated_at": time.Now().Unix()}).Error
}

func (db *DB) gormSetAccountProfile(ctx context.Context, id int64, nickname, avatar *string, userInfo map[string]any) error {
	userJSON, err := marshalNullable(userInfo)
	if err != nil {
		return err
	}
	updates := map[string]any{"user_info": userJSON.String, "updated_at": time.Now().Unix()}
	if nickname != nil {
		updates["nickname"] = *nickname
	}
	if avatar != nil {
		updates["avatar"] = *avatar
	}
	return db.orm.WithContext(ctx).Model(&GormWechatAccount{}).Where("id = ?", id).Updates(updates).Error
}

func (db *DB) gormSetAccountCredential(ctx context.Context, id int64, loginBuffer string, credentials map[string]any) error {
	credJSON, err := marshalNullable(credentials)
	if err != nil {
		return err
	}
	return db.orm.WithContext(ctx).Model(&GormWechatAccount{}).Where("id = ?", id).
		Updates(map[string]any{
			"login_buffer": loginBuffer,
			"credentials":  credJSON.String,
			"updated_at":   time.Now().Unix(),
		}).Error
}

func (db *DB) gormSetAccountStatus(ctx context.Context, id int64, status string) error {
	now := time.Now().Unix()
	return db.orm.WithContext(ctx).Model(&GormWechatAccount{}).Where("id = ?", id).
		Updates(map[string]any{"status": status, "last_checked_at": now, "updated_at": now}).Error
}

func (db *DB) gormDeleteAccount(ctx context.Context, id int64) error {
	return db.orm.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("wechat_account_id = ?", id).Delete(&GormSession{}).Error; err != nil {
			return err
		}
		return tx.Delete(&GormWechatAccount{}, id).Error
	})
}

func (db *DB) gormGetSession(ctx context.Context, accountID int64, tcpProxy string) (*SessionRow, error) {
	var m GormSession
	err := db.orm.WithContext(ctx).
		Where("wechat_account_id = ? AND tcp_proxy = ? AND expires_at > ?", accountID, tcpProxy, time.Now().Unix()).
		First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, sql.ErrNoRows
		}
		return nil, err
	}
	return gormModelToSession(&m)
}

func gormModelToSession(m *GormSession) (*SessionRow, error) {
	var blob map[string]any
	if err := json.Unmarshal([]byte(m.SessionBlob), &blob); err != nil {
		return nil, fmt.Errorf("decode session_blob: %w", err)
	}
	return &SessionRow{
		ID:              m.ID,
		WechatAccountID: m.WechatAccountID,
		UIN:             m.UIN,
		TCPProxy:        m.TCPProxy,
		SessionBlob:     blob,
		ExpiresAt:       m.ExpiresAt,
		CreatedAt:       m.CreatedAt,
		UpdatedAt:       m.UpdatedAt,
	}, nil
}

func (db *DB) gormPutSession(ctx context.Context, accountID int64, uin *int64, sessionBlob map[string]any, expiresAt int64, tcpProxy string) error {
	now := time.Now().Unix()
	blob, err := json.Marshal(sessionBlob)
	if err != nil {
		return err
	}
	row := GormSession{
		WechatAccountID: accountID,
		UIN:             uin,
		TCPProxy:        tcpProxy,
		SessionBlob:     string(blob),
		ExpiresAt:       expiresAt,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	return db.orm.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "wechat_account_id"}, {Name: "tcp_proxy"}},
		DoUpdates: clause.AssignmentColumns([]string{"uin", "session_blob", "expires_at", "updated_at"}),
	}).Create(&row).Error
}

func (db *DB) gormInvalidateSession(ctx context.Context, accountID int64, tcpProxy string) error {
	return db.orm.WithContext(ctx).
		Where("wechat_account_id = ? AND tcp_proxy = ?", accountID, tcpProxy).
		Delete(&GormSession{}).Error
}

func (db *DB) gormPurgeExpiredSessions(ctx context.Context) (int64, error) {
	res := db.orm.WithContext(ctx).Where("expires_at <= ?", time.Now().Unix()).Delete(&GormSession{})
	return res.RowsAffected, res.Error
}

func (db *DB) gormEnsureDefaultFeatures(ctx context.Context) error {
	for _, f := range defaultFeatures {
		row := GormFeature{Code: f.Code, Name: f.Name, Description: f.Description, Enabled: true}
		if err := db.orm.WithContext(ctx).
			Where(GormFeature{Code: f.Code}).
			Assign(GormFeature{Name: f.Name, Description: f.Description, Enabled: true}).
			FirstOrCreate(&row).Error; err != nil {
			return err
		}
	}
	return nil
}

func (db *DB) gormListFeatures(ctx context.Context, onlyEnabled bool) ([]Feature, error) {
	q := db.orm.WithContext(ctx).Model(&GormFeature{})
	if onlyEnabled {
		q = q.Where("enabled = ?", true)
	}
	var rows []GormFeature
	if err := q.Order("code asc").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]Feature, 0, len(rows))
	for _, r := range rows {
		out = append(out, Feature{Code: r.Code, Name: r.Name, Description: r.Description, Enabled: r.Enabled})
	}
	return out, nil
}

func (db *DB) gormGetFeature(ctx context.Context, code int) (*Feature, error) {
	var m GormFeature
	if err := db.orm.WithContext(ctx).First(&m, code).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, sql.ErrNoRows
		}
		return nil, err
	}
	f := Feature{Code: m.Code, Name: m.Name, Description: m.Description, Enabled: m.Enabled}
	return &f, nil
}

func (db *DB) gormGetFeatureByName(ctx context.Context, name string) (*Feature, error) {
	var m GormFeature
	if err := db.orm.WithContext(ctx).Where("LOWER(name) = LOWER(?)", name).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, sql.ErrNoRows
		}
		return nil, err
	}
	f := Feature{Code: m.Code, Name: m.Name, Description: m.Description, Enabled: m.Enabled}
	return &f, nil
}
