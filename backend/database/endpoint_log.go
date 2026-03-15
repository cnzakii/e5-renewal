package database

import (
	"context"

	"e5-renewal/backend/models"
)

type EndpointLogRepo struct{}

func (r *EndpointLogRepo) ListByTaskLogID(ctx context.Context, taskLogID uint) ([]models.EndpointLog, error) {
	var logs []models.EndpointLog
	err := GetDB(ctx).Where("task_log_id = ?", taskLogID).Order("executed_at asc").Find(&logs).Error
	return logs, err
}
