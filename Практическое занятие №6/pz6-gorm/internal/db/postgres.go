package db

import (
	"fmt"
	"os"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Connect открывает пул соединений GORM по DSN.
//
// GORM_LOG=info включает вывод всех SQL-запросов, которые генерирует GORM
// (по умолчанию — только предупреждения, например медленные запросы).
func Connect(dsn string) (*gorm.DB, error) {
	if dsn == "" {
		return nil, fmt.Errorf("DB_DSN is empty")
	}

	level := logger.Warn
	if os.Getenv("GORM_LOG") == "info" {
		level = logger.Info
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(level),
		// превращает ошибки драйвера в gorm.ErrDuplicatedKey, gorm.ErrForeignKeyViolated и т.д.
		TranslateError: true,
	})
	if err != nil {
		return nil, fmt.Errorf("connect db: %w", err)
	}

	// Настроим пул соединений
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(20)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetConnMaxLifetime(time.Hour)

	return db, nil
}
