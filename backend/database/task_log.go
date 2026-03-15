package database

import (
	"context"
	"time"

	"e5-renewal/backend/models"

	"gorm.io/gorm"
)

type TaskLogRepo struct{}

// TaskLogFilter holds query parameters for listing task logs.
type TaskLogFilter struct {
	ID          uint
	AccountID   uint
	TriggerType string // "scheduled" or "manual"
	Status      string // "success", "partial", "failed"
	DateFrom    *time.Time
	DateTo      *time.Time
	Page        int
	PageSize    int
}

// ListWithTotal returns paginated task logs and total count matching the filter.
func (r *TaskLogRepo) ListWithTotal(ctx context.Context, f TaskLogFilter) ([]models.TaskLog, int64, error) {
	query := GetDB(ctx).Model(&models.TaskLog{})

	if f.ID > 0 {
		query = query.Where("id = ?", f.ID)
	}
	if f.AccountID > 0 {
		query = query.Where("account_id = ?", f.AccountID)
	}
	if f.TriggerType == "scheduled" || f.TriggerType == "manual" {
		query = query.Where("trigger_type = ?", f.TriggerType)
	}
	switch f.Status {
	case "success":
		query = query.Where("fail_count = 0")
	case "partial":
		query = query.Where("fail_count > 0 AND success_count > 0")
	case "failed":
		query = query.Where("success_count = 0 AND fail_count > 0")
	}
	if f.DateFrom != nil {
		query = query.Where("started_at >= ?", *f.DateFrom)
	}
	if f.DateTo != nil {
		query = query.Where("started_at < ?", *f.DateTo)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var logs []models.TaskLog
	offset := (f.Page - 1) * f.PageSize
	err := query.Order("started_at desc").Limit(f.PageSize).Offset(offset).Find(&logs).Error
	return logs, total, err
}

func (r *TaskLogRepo) CountByAccount(ctx context.Context, accountID uint) (int64, error) {
	var count int64
	err := GetDB(ctx).Model(&models.TaskLog{}).Where("account_id = ?", accountID).Count(&count).Error
	return count, err
}

func (r *TaskLogRepo) CountSuccessByAccount(ctx context.Context, accountID uint) (int64, error) {
	var count int64
	err := GetDB(ctx).Model(&models.TaskLog{}).Where("account_id = ? AND fail_count = 0", accountID).Count(&count).Error
	return count, err
}

func (r *TaskLogRepo) LastByAccount(ctx context.Context, accountID uint) (*models.TaskLog, error) {
	var log models.TaskLog
	err := GetDB(ctx).Where("account_id = ?", accountID).Order("started_at desc").First(&log).Error
	if err != nil {
		return nil, err
	}
	return &log, nil
}

func (r *TaskLogRepo) RecentN(ctx context.Context, n int) ([]models.TaskLog, error) {
	var logs []models.TaskLog
	err := GetDB(ctx).Order("started_at desc").Limit(n).Find(&logs).Error
	return logs, err
}

// Last20ByAccount returns the last 20 task logs for health calculation.
func (r *TaskLogRepo) Last20ByAccount(ctx context.Context, accountID uint) ([]models.TaskLog, error) {
	var logs []models.TaskLog
	err := GetDB(ctx).Where("account_id = ?", accountID).Order("started_at desc").Limit(20).Find(&logs).Error
	return logs, err
}

// CountInPeriod counts total task logs in a time range.
func (r *TaskLogRepo) CountInPeriod(ctx context.Context, since time.Time) (int64, error) {
	q := GetDB(ctx).Model(&models.TaskLog{})
	if !since.IsZero() {
		q = q.Where("started_at >= ?", since)
	}
	var count int64
	err := q.Count(&count).Error
	return count, err
}

// CountErrorsInPeriod counts task logs with fail_count > 0 in a time range.
func (r *TaskLogRepo) CountErrorsInPeriod(ctx context.Context, since time.Time) (int64, error) {
	q := GetDB(ctx).Model(&models.TaskLog{}).Where("fail_count > 0")
	if !since.IsZero() {
		q = q.Where("started_at >= ?", since)
	}
	var count int64
	err := q.Count(&count).Error
	return count, err
}

// FindInTimeRange returns all task logs in a time range.
func (r *TaskLogRepo) FindInTimeRange(ctx context.Context, start, end time.Time) ([]models.TaskLog, error) {
	var logs []models.TaskLog
	err := GetDB(ctx).Where("started_at >= ? AND started_at < ?", start, end).Find(&logs).Error
	return logs, err
}

// EndpointCountsByAccount returns total and successful endpoint counts across all task logs for an account.
func (r *TaskLogRepo) EndpointCountsByAccount(ctx context.Context, accountID uint) (totalEp int64, successEp int64, err error) {
	type result struct {
		TotalEp   int64
		SuccessEp int64
	}
	var res result
	err = GetDB(ctx).Model(&models.TaskLog{}).
		Select("COALESCE(SUM(total_endpoints), 0) as total_ep, COALESCE(SUM(success_count), 0) as success_ep").
		Where("account_id = ?", accountID).
		Scan(&res).Error
	return res.TotalEp, res.SuccessEp, err
}

// EndpointCountsInPeriod returns total and successful endpoint counts across all task logs
// that started at or after since. A zero since value counts across all time.
func (r *TaskLogRepo) EndpointCountsInPeriod(ctx context.Context, since time.Time) (totalEp int64, successEp int64, err error) {
	type result struct {
		TotalEp   int64
		SuccessEp int64
	}
	var res result
	q := GetDB(ctx).Model(&models.TaskLog{}).
		Select("COALESCE(SUM(total_endpoints), 0) as total_ep, COALESCE(SUM(success_count), 0) as success_ep")
	if !since.IsZero() {
		q = q.Where("started_at >= ?", since)
	}
	err = q.Scan(&res).Error
	return res.TotalEp, res.SuccessEp, err
}

// CreateWithEndpoints creates a TaskLog and its EndpointLogs in a single transaction.
func (r *TaskLogRepo) CreateWithEndpoints(ctx context.Context, taskLog *models.TaskLog, endpoints []models.EndpointLog) error {
	return GetDB(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(taskLog).Error; err != nil {
			return err
		}
		for i := range endpoints {
			endpoints[i].TaskLogID = taskLog.ID
			if err := tx.Create(&endpoints[i]).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
