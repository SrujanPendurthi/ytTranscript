package db

import (
	"fmt"
	"log"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// ConnectWithRetry calls Connect repeatedly until it succeeds or attempts run
// out, sleeping delay between tries. On a Docker Desktop restart the restart
// policy bypasses depends_on, so this process can boot before Postgres is
// accepting connections — retrying avoids a crash-loop until it's ready.
func ConnectWithRetry(dsn string, attempts int, delay time.Duration) (*gorm.DB, error) {
	var lastErr error
	for i := 1; i <= attempts; i++ {
		database, err := Connect(dsn)
		if err == nil {
			return database, nil
		}
		lastErr = err
		log.Printf("db: connect attempt %d/%d failed: %v", i, attempts, err)
		time.Sleep(delay)
	}
	return nil, fmt.Errorf("db: giving up after %d attempts: %w", attempts, lastErr)
}

func Connect(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)

	return db, nil
}
