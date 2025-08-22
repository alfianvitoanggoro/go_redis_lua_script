package db

import (
	"database/sql"
	"fmt"
	"time"

	"grls/internal/config"
	"grls/pkg/logger"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var sqlDBWrite, sqlDBRead *sql.DB

func ConnectDBWrite(dbConfig *config.DBConfig) (*gorm.DB, error) {
	dsn := buildWriteDSN(dbConfig.DBWrite)
	db, sqlDB, err := connect(dsn, dbConfig.DBPool)
	if err != nil {
		return nil, err
	}

	sqlDBWrite = sqlDB
	return db, nil
}

func ConnectDBRead(dbConfig *config.DBConfig) (*gorm.DB, error) {
	dsn := buildReadDSN(dbConfig.DBRead)
	db, sqlDB, err := connect(dsn, dbConfig.DBPool)
	if err != nil {
		return nil, err
	}
	sqlDBRead = sqlDB
	return db, nil
}

func CloseDBWrite() {
	if sqlDBWrite != nil {
		if err := sqlDBWrite.Close(); err != nil {
			logger.Warn(fmt.Sprintf("⚠️ Error closing WRITE DB: %v", err))
		} else {
			logger.Info("🔌 WRITE DB connection closed.")
		}
	}
}

func CloseDBRead() {
	if sqlDBRead != nil {
		if err := sqlDBRead.Close(); err != nil {
			logger.Warn(fmt.Sprintf("⚠️ Error closing READ DB: %v", err))
		} else {
			logger.Info("🔌 READ DB connection closed.")
		}
	}
}

func buildWriteDSN(dbWriteConfig *config.DBWriteConfig) string {
	return fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
		dbWriteConfig.Host,
		dbWriteConfig.User,
		dbWriteConfig.Password,
		dbWriteConfig.Name,
		dbWriteConfig.Port,
		dbWriteConfig.SSLMode,
	)
}

func buildReadDSN(dbReadConfig *config.DBReadConfig) string {
	return fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
		dbReadConfig.Host,
		dbReadConfig.User,
		dbReadConfig.Password,
		dbReadConfig.Name,
		dbReadConfig.Port,
		dbReadConfig.SSLMode,
	)
}

func connect(dsn string, dbPoolingConfig *config.DBPooling) (*gorm.DB, *sql.DB, error) {
	db, err := gorm.Open(postgres.New(postgres.Config{
		DSN:                  dsn,
		PreferSimpleProtocol: true,
	}), &gorm.Config{
		PrepareStmt: true,
	})
	if err != nil {
		return nil, nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, nil, err
	}

	// Connection pool config
	sqlDB.SetMaxOpenConns(dbPoolingConfig.MaxOpenConns)
	sqlDB.SetMaxIdleConns(dbPoolingConfig.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(time.Duration(dbPoolingConfig.ConnMaxLifetime) * time.Minute)
	sqlDB.SetConnMaxIdleTime(time.Duration(dbPoolingConfig.ConnMaxIdleTime) * time.Minute * time.Minute)

	return db, sqlDB, nil
}
