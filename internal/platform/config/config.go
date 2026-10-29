// Package config читает конфигурацию из переменных окружения.
//
// Конфигурация — только через env (см. .env.example). Значения по умолчанию
// пригодны для локального dev; в prod часть полей обязательна (см. Load).
package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Env значения окружения.
const (
	EnvDev  = "dev"
	EnvProd = "prod"
)

// Config — полная конфигурация процесса.
type Config struct {
	Env  string
	HTTP HTTPConfig
	Log  LogConfig
	JWT  JWTConfig
	YDB  YDBConfig
	S3   S3Config
	YMQ  YMQConfig
}

// HTTPConfig — адреса и префикс HTTP-сервера.
type HTTPConfig struct {
	Addr   string // listen-адрес, например :8080
	Prefix string // префикс путей API, например /v1
}

// LogConfig — настройки логирования.
type LogConfig struct {
	Level string // debug | info | warn | error
}

// JWTConfig — подпись и время жизни токенов.
type JWTConfig struct {
	Secret     string
	AccessTTL  time.Duration
	RefreshTTL time.Duration
}

// YDBConfig — подключение к YDB Serverless.
type YDBConfig struct {
	Endpoint string
	Database string
}

// S3Config — Object Storage (S3-совместимый) для загрузок.
type S3Config struct {
	Endpoint        string
	Bucket          string
	AccessKeyID     string
	SecretAccessKey string
}

// YMQConfig — очередь сообщений.
type YMQConfig struct {
	QueueURL string
}

// Load собирает конфигурацию из env.
func Load() (Config, error) {
	accessTTL, err := getenvDuration("JWT_ACCESS_TTL_SECONDS", 15*time.Minute)
	if err != nil {
		return Config{}, err
	}

	refreshTTL, err := getenvDuration("JWT_REFRESH_TTL_SECONDS", 30*24*time.Hour)
	if err != nil {
		return Config{}, err
	}

	cfg := Config{
		Env: getenv("APP_ENV", EnvDev),
		HTTP: HTTPConfig{
			Addr:   getenv("HTTP_ADDR", ":8080"),
			Prefix: getenv("HTTP_PREFIX", "/v1"),
		},
		Log: LogConfig{
			Level: getenv("LOG_LEVEL", "debug"),
		},
		JWT: JWTConfig{
			Secret:     getenv("JWT_SECRET", ""),
			AccessTTL:  accessTTL,
			RefreshTTL: refreshTTL,
		},
		YDB: YDBConfig{
			Endpoint: getenv("YDB_ENDPOINT", ""),
			Database: getenv("YDB_DATABASE", ""),
		},
		S3: S3Config{
			Endpoint:        getenv("S3_ENDPOINT", "https://storage.yandexcloud.net"),
			Bucket:          getenv("S3_BUCKET", ""),
			AccessKeyID:     getenv("S3_ACCESS_KEY_ID", ""),
			SecretAccessKey: getenv("S3_SECRET_ACCESS_KEY", ""),
		},
		YMQ: YMQConfig{
			QueueURL: getenv("YMQ_QUEUE_URL", ""),
		},
	}

	if err := cfg.validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func (c Config) validate() error {
	if c.Env != EnvDev && c.Env != EnvProd {
		return fmt.Errorf("config: APP_ENV=%q, ожидается %q или %q", c.Env, EnvDev, EnvProd)
	}
	if c.Env != EnvProd {
		return nil // в dev всё необязательно
	}

	var missing []string
	if c.JWT.Secret == "" {
		missing = append(missing, "JWT_SECRET")
	}
	if c.YDB.Endpoint == "" {
		missing = append(missing, "YDB_ENDPOINT")
	}
	if c.YDB.Database == "" {
		missing = append(missing, "YDB_DATABASE")
	}
	if c.S3.Bucket == "" {
		missing = append(missing, "S3_BUCKET")
	}
	if len(missing) > 0 {
		return errors.New("config: обязательные переменные не заданы: " + strings.Join(missing, ", "))
	}

	return nil
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}

	return fallback
}

// getenvDuration читает целое число секунд; пустое значение — fallback,
// мусор или неположительное число — ошибка (молча подменять опечатку в TTL опасно).
func getenvDuration(key string, fallback time.Duration) (time.Duration, error) {
	v := os.Getenv(key)
	if v == "" {
		return fallback, nil
	}

	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil || n <= 0 {
		return 0, fmt.Errorf("config: %s=%q, ожидается положительное число секунд", key, v)
	}

	return time.Duration(n) * time.Second, nil
}
