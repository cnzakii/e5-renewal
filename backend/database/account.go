package database

import (
	"context"
	"time"

	"e5-renewal/backend/models"

	"gorm.io/gorm"
)

type AccountRepo struct{}

func (r *AccountRepo) List(ctx context.Context) ([]models.Account, error) {
	var accounts []models.Account
	if err := GetDB(ctx).Order("id asc").Find(&accounts).Error; err != nil {
		return nil, err
	}
	for i := range accounts {
		plain, err := decryptAuthInfo(accounts[i].AuthInfo)
		if err != nil {
			return nil, err
		}
		accounts[i].AuthInfo = plain
	}
	return accounts, nil
}

func (r *AccountRepo) GetByID(ctx context.Context, id uint) (*models.Account, error) {
	var acc models.Account
	if err := GetDB(ctx).First(&acc, id).Error; err != nil {
		return nil, err
	}
	plain, err := decryptAuthInfo(acc.AuthInfo)
	if err != nil {
		return nil, err
	}
	acc.AuthInfo = plain
	return &acc, nil
}

func (r *AccountRepo) Create(ctx context.Context, acc *models.Account) error {
	encrypted, err := encryptAuthInfo(acc.AuthInfo)
	if err != nil {
		return err
	}
	original := acc.AuthInfo
	acc.AuthInfo = encrypted
	err = GetDB(ctx).Create(acc).Error
	acc.AuthInfo = original // restore plaintext so caller is unaffected
	return err
}

func (r *AccountRepo) Save(ctx context.Context, acc *models.Account) error {
	encrypted, err := encryptAuthInfo(acc.AuthInfo)
	if err != nil {
		return err
	}
	original := acc.AuthInfo
	acc.AuthInfo = encrypted
	err = GetDB(ctx).Save(acc).Error
	acc.AuthInfo = original // restore plaintext so caller is unaffected
	return err
}

// DeleteCascade deletes an account and all its task logs and endpoint logs.
func (r *AccountRepo) DeleteCascade(ctx context.Context, id uint) error {
	return GetDB(ctx).Transaction(func(tx *gorm.DB) error {
		var logIDs []uint
		if err := tx.Model(&models.TaskLog{}).Where("account_id = ?", id).Pluck("id", &logIDs).Error; err != nil {
			return err
		}
		if len(logIDs) > 0 {
			if err := tx.Where("task_log_id IN ?", logIDs).Delete(&models.EndpointLog{}).Error; err != nil {
				return err
			}
		}
		if err := tx.Where("account_id = ?", id).Delete(&models.TaskLog{}).Error; err != nil {
			return err
		}
		if err := tx.Where("account_id = ?", id).Delete(&models.Schedule{}).Error; err != nil {
			return err
		}
		return tx.Delete(&models.Account{}, id).Error
	})
}

func (r *AccountRepo) CountAll(ctx context.Context) (int64, error) {
	var count int64
	err := GetDB(ctx).Model(&models.Account{}).Count(&count).Error
	return count, err
}

func (r *AccountRepo) CountByAuthType(ctx context.Context, authType string) (int64, error) {
	var count int64
	err := GetDB(ctx).Model(&models.Account{}).Where("auth_type = ?", authType).Count(&count).Error
	return count, err
}

// UpdateAuthInfo updates auth_info JSON and auth_expires_at for an account.
func (r *AccountRepo) UpdateAuthInfo(ctx context.Context, id uint, authInfo string, expiresAt *time.Time) error {
	encrypted, err := encryptAuthInfo(authInfo)
	if err != nil {
		return err
	}
	return GetDB(ctx).Model(&models.Account{}).Where("id = ?", id).Updates(map[string]any{
		"auth_info":       encrypted,
		"auth_expires_at": expiresAt,
	}).Error
}

// FindExpiringBefore returns accounts whose auth expires at or before the given time.
func (r *AccountRepo) FindExpiringBefore(ctx context.Context, threshold time.Time) ([]models.Account, error) {
	var accounts []models.Account
	if err := GetDB(ctx).Where("auth_expires_at IS NOT NULL AND auth_expires_at <= ?", threshold).Find(&accounts).Error; err != nil {
		return nil, err
	}
	for i := range accounts {
		plain, err := decryptAuthInfo(accounts[i].AuthInfo)
		if err != nil {
			return nil, err
		}
		accounts[i].AuthInfo = plain
	}
	return accounts, nil
}
