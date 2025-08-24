package config

import (
	"grls/pkg/logger"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	DB     *DBConfig
	App    *AppConfig
	Redis  *RedisConfig
	Worker *WorkerConfig
}

type AppConfig struct {
	Name        string
	Env         string
	Port        string
	LogFilePath string
	BinFilePath string
}

type DBConfig struct {
	DBWrite *DBWriteConfig
	DBRead  *DBReadConfig
	DBPool  *DBPooling
}

type DBWriteConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string
}

type DBReadConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string
}

type DBPooling struct {
	MaxIdleConns    int
	MaxOpenConns    int
	ConnMaxLifetime int
	ConnMaxIdleTime int
}

type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       string
}

type WorkerConfig struct {
	WorkerCount int
}

func Load() *Config {
	if err := godotenv.Load(); err != nil {
		logger.Error("failed, No .env file found")
	}

	return &Config{
		DB:     LoadDBConfig(),
		App:    LoadAppConfig(),
		Redis:  LoadRedisConfig(),
		Worker: LoadWorkerConfig(),
	}
}

func LoadAppConfig() *AppConfig {
	return &AppConfig{
		Name:        getEnv("APP_NAME", "app_name"),
		Env:         getEnv("ENV", "development"),
		Port:        getEnv("APP_PORT", "50051"),
		LogFilePath: getEnv("APP_LOG_FILE", "logs/app.log"),
		BinFilePath: getEnv("APP_BIN_FILE", "./bin/grls"),
	}
}

func LoadDBConfig() *DBConfig {
	dbWrite := &DBWriteConfig{
		Host:     getEnv("DB_WRITE_HOST", "localhost"),
		Port:     getEnv("DB_WRITE_PORT", "5432"),
		User:     getEnv("DB_WRITE_USER", "postgres"),
		Password: getEnv("DB_WRITE_PASSWORD", "password"),
		Name:     getEnv("DB_WRITE_NAME", "point_system"),
		SSLMode:  getEnv("DB_WRITE_SSL_MODE", "disable"),
	}

	dbRead := &DBReadConfig{
		Host:     getEnv("DB_READ_HOST", "localhost"),
		Port:     getEnv("DB_READ_PORT", "5432"),
		User:     getEnv("DB_READ_USER", "postgres"),
		Password: getEnv("DB_READ_PASSWORD", "password"),
		Name:     getEnv("DB_READ_NAME", "point_system"),
		SSLMode:  getEnv("DB_READ_SSL_MODE", "disable"),
	}

	dbPool := &DBPooling{
		MaxIdleConns:    getEnvAsInt("DB_MAX_IDLE_CONNS", 10),
		MaxOpenConns:    getEnvAsInt("DB_MAX_OPEN_CONNS", 100),
		ConnMaxLifetime: getEnvAsInt("DB_CONN_MAX_LIFETIME", 3600),
		ConnMaxIdleTime: getEnvAsInt("DB_CONN_MAX_IDLE_TIME", 300),
	}

	return &DBConfig{
		DBWrite: dbWrite,
		DBRead:  dbRead,
		DBPool:  dbPool,
	}
}

func LoadRedisConfig() *RedisConfig {
	return &RedisConfig{
		Host:     getEnv("REDIS_HOST", "localhost"),
		Port:     getEnv("REDIS_PORT", "6379"),
		Password: getEnv("REDIS_PASSWORD", "password"),
		DB:       getEnv("REDIS_DB", "0"),
	}
}

func LoadWorkerConfig() *WorkerConfig {
	return &WorkerConfig{
		WorkerCount: getEnvAsInt("WORKER_COUNT", 5),
	}
}

// =========================================================

func GetAppPort() string {
	return getEnv("APP_PORT", "50051")
}

func GetAppEnv() string {
	return getEnv("APP_ENV", "development")
}

func GetAppBinFile() string {
	return getEnv("APP_BIN_FILE", "./bin/grls")
}

//============================================================

// getEnv returns the value of the environment variable or a default value if not set
func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

// getEnvAsInt returns the value of the environment variable as an integer or a default value if not set
func getEnvAsInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if intVal, err := strconv.Atoi(val); err == nil {
			return intVal
		}
	}
	return defaultVal
}
