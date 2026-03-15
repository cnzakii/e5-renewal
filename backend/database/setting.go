package database

import (
	"context"
	"errors"

	"e5-renewal/backend/models"

	"gorm.io/gorm"
)

type SettingRepo struct{}

// Get returns the value for a setting key. Returns empty string if not found.
func (r *SettingRepo) Get(ctx context.Context, key string) (string, error) {
	var setting models.Setting
	err := GetDB(ctx).Where("key = ?", key).First(&setting).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return setting.Value, nil
}

// Upsert creates or updates a setting.
func (r *SettingRepo) Upsert(ctx context.Context, key, value string) error {
	var setting models.Setting
	err := GetDB(ctx).Where("key = ?", key).First(&setting).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return GetDB(ctx).Create(&models.Setting{Key: key, Value: value}).Error
	}
	if err != nil {
		return err
	}
	setting.Value = value
	return GetDB(ctx).Save(&setting).Error
}
