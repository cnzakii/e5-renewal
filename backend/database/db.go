package database

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"e5-renewal/backend/models"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

var globalDB *gorm.DB

// Init opens the SQLite database and stores the singleton instance.
func Init(path string) error {
	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{
		Logger: NewGormSlogLogger(slog.Default(), GormSlogOptions{
			SlowThreshold: 200 * time.Millisecond,
			LogLevel:      gormlogger.Info,
		}),
	})
	if err != nil {
		return err
	}

	err = db.AutoMigrate(
		&models.Account{},
		&models.TaskLog{},
		&models.EndpointLog{},
		&models.Setting{},
		&models.Schedule{},
	)
	if err != nil {
		return err
	}

	globalDB = db
	return nil
}

// GetDB returns the singleton *gorm.DB instance with context attached.
func GetDB(ctx context.Context) *gorm.DB {
	return globalDB.WithContext(ctx)
}

// Singleton repo instances — usable after Init().
var (
	Accounts     = &AccountRepo{}
	TaskLogs     = &TaskLogRepo{}
	Schedules    = &ScheduleRepo{}
	Settings     = &SettingRepo{}
	EndpointLogs = &EndpointLogRepo{}
)
