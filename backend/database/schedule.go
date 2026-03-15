package database

import (
	"context"

	"e5-renewal/backend/models"
)

type ScheduleRepo struct{}

func (r *ScheduleRepo) GetByAccountID(ctx context.Context, accountID uint) (*models.Schedule, error) {
	var sched models.Schedule
	err := GetDB(ctx).Where("account_id = ?", accountID).First(&sched).Error
	if err != nil {
		return nil, err
	}
	return &sched, nil
}

func (r *ScheduleRepo) Create(ctx context.Context, sched *models.Schedule) error {
	return GetDB(ctx).Create(sched).Error
}

func (r *ScheduleRepo) Save(ctx context.Context, sched *models.Schedule) error {
	return GetDB(ctx).Save(sched).Error
}

func (r *ScheduleRepo) DeleteByAccountID(ctx context.Context, accountID uint) error {
	return GetDB(ctx).Where("account_id = ?", accountID).Delete(&models.Schedule{}).Error
}

func (r *ScheduleRepo) ListActive(ctx context.Context) ([]models.Schedule, error) {
	var schedules []models.Schedule
	err := GetDB(ctx).Where("enabled = ? AND paused = ?", true, false).Find(&schedules).Error
	return schedules, err
}

// UpdateFields updates specific fields on a schedule by account ID.
func (r *ScheduleRepo) UpdateFields(ctx context.Context, accountID uint, updates map[string]any) error {
	return GetDB(ctx).Model(&models.Schedule{}).Where("account_id = ?", accountID).Updates(updates).Error
}
