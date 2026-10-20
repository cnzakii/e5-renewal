package app

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/alexedwards/argon2id"
	"github.com/glebarez/sqlite"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func randomToken() string {
	return rand.Text()
}

var passwordParams = &argon2id.Params{
	Memory:      19 * 1024,
	Iterations:  2,
	Parallelism: 1,
	SaltLength:  16,
	KeyLength:   32,
}

// HashPassword validates and hashes an administrator password with Argon2id.
func HashPassword(password string) (string, error) {
	if len(password) < 12 || len(password) > 1024 {
		return "", newError("password_length", "The administrator password must contain 12 to 1024 bytes.")
	}

	return argon2id.CreateHash(password, passwordParams)
}

// OpenDB configures the connection pool and creates the application's tables and indexes.
func OpenDB(c Config) (*gorm.DB, error) {
	if err := os.MkdirAll(c.DataDir, 0o700); err != nil {
		return nil, err
	}

	var dialector gorm.Dialector
	switch c.Driver {
	case "sqlite":
		if err := os.MkdirAll(filepath.Dir(c.DSN), 0o700); err != nil {
			return nil, err
		}

		f, err := os.OpenFile(c.DSN, os.O_CREATE|os.O_RDWR, 0o600)
		if err != nil {
			return nil, err
		}

		if err := f.Close(); err != nil {
			return nil, err
		}

		if err := os.Chmod(c.DSN, 0o600); err != nil {
			return nil, err
		}

		filename, err := filepath.Abs(c.DSN)
		if err != nil {
			return nil, err
		}

		// Connection-local pragmas must also run when the pool replaces a connection.
		source := url.URL{
			Scheme:   "file",
			Path:     filename,
			RawQuery: "_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)",
		}
		dialector = sqlite.Open(source.String())
	case "postgres":
		dialector = postgres.Open(c.DSN)
	case "mysql":
		dialector = mysql.Open(c.DSN)
	default:
		return nil, errors.New("unsupported database driver")
	}

	db, err := gorm.Open(
		dialector,
		&gorm.Config{
			Logger:         logger.Default.LogMode(logger.Silent),
			TranslateError: true,
			NowFunc:        func() time.Time { return time.Now().UTC() },
		},
	)
	if err != nil {
		logDatabaseError(context.Background(), "database connection failed", err)
		return nil, errors.New("could not connect to the database; check the database configuration")
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	ready := false
	defer func() {
		if !ready {
			if closeError := sqlDB.Close(); closeError != nil {
				slog.Warn("database cleanup failed", "error_type", fmt.Sprintf("%T", closeError))
			}
		}
	}()
	sqlDB.SetMaxOpenConns(4)
	sqlDB.SetMaxIdleConns(2)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)
	if c.Driver == "sqlite" {
		sqlDB.SetMaxOpenConns(1)
		sqlDB.SetMaxIdleConns(1)
		if err := db.Exec("PRAGMA journal_mode=WAL").Error; err != nil {
			return nil, err
		}
	}

	if err := db.AutoMigrate(&Admin{}, &Account{}, &Run{}, &GraphCall{}, &OAuthAttempt{}); err != nil {
		return nil, fmt.Errorf("database migration failed: %w", err)
	}

	ready = true

	return db, nil
}

func (r *Run) measure() {
	r.RequestCount, r.SuccessCount = len(r.Results), 0
	for _, result := range r.Results {
		if result.Error == "" && result.HTTPStatus >= 200 && result.HTTPStatus < 300 {
			r.SuccessCount++
		}
	}
}

// Initialize never overwrites an existing password during a normal restart.
func Initialize(db *gorm.DB, password string) (string, error) {
	var admin Admin
	err := db.First(&admin, 1).Error
	if err == nil {
		return "", nil
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return "", err
	}

	generated := ""
	if password == "" {
		password = randomToken()
		generated = password
	}

	hash, err := HashPassword(password)
	if err != nil {
		return "", err
	}

	if err := db.Create(&Admin{ID: 1, PasswordHash: hash, Revision: 1}).Error; err != nil {
		return "", err
	}

	return generated, nil
}

// ResetPassword is for offline recovery; live password changes must also update the server's cached revision.
func ResetPassword(db *gorm.DB, password string) error {
	hash, err := HashPassword(password)
	if err != nil {
		return err
	}

	result := db.Model(&Admin{}).Where("id = ?", 1).Updates(map[string]any{
		"password_hash": hash, "revision": gorm.Expr("revision + 1"),
	})
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected != 1 {
		return errors.New("the administrator has not been initialized")
	}

	return nil
}
