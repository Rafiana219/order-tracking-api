package config

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

type Config struct {
	AppPort           string
	DBHost            string
	DBPort            string
	DBUser            string
	DBPassword        string
	DBName            string
	DBSSLMode         string
	WorkerInterval    time.Duration
	WorkerMaxAttempts int
}

func Load() (Config, error) {
	// ใช้ค่าจาก environment ได้ แม้ไม่มีไฟล์ .env
	_ = godotenv.Load()

	cfg := Config{
		AppPort:           getEnv("APP_PORT", "8080"),
		DBHost:            getEnv("DB_HOST", "127.0.0.1"),
		DBPort:            getEnv("DB_PORT", "5433"),
		DBUser:            getEnv("DB_USER", "order_user"),
		DBPassword:        getEnv("DB_PASSWORD", ""),
		DBName:            getEnv("DB_NAME", "order_tracking"),
		DBSSLMode:         getEnv("DB_SSLMODE", "disable"),
		WorkerMaxAttempts: 3,
	}

	interval, err := time.ParseDuration(
		getEnv("WORKER_INTERVAL", "3s"),
	)
	if err != nil || interval <= 0 {
		return Config{}, fmt.Errorf(
			"invalid WORKER_INTERVAL: must be a positive duration",
		)
	}
	cfg.WorkerInterval = interval

	maxAttempts, err := strconv.Atoi(
		getEnv("WORKER_MAX_ATTEMPTS", "3"),
	)
	if err != nil || maxAttempts < 1 {
		return Config{}, fmt.Errorf(
			"invalid WORKER_MAX_ATTEMPTS: must be at least 1",
		)
	}
	cfg.WorkerMaxAttempts = maxAttempts

	if cfg.DBPassword == "" {
		return Config{}, fmt.Errorf("DB_PASSWORD must not be empty")
	}

	return cfg, nil
}

func (c Config) DatabaseURL() string {
	u := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(c.DBUser, c.DBPassword),
		Host:   net.JoinHostPort(c.DBHost, c.DBPort),
		Path:   c.DBName,
	}

	query := u.Query()
	query.Set("sslmode", c.DBSSLMode)
	u.RawQuery = query.Encode()

	return u.String()
}

func (c Config) NewDBPool(
	ctx context.Context,
) (*pgxpool.Pool, error) {
	poolConfig, err := pgxpool.ParseConfig(c.DatabaseURL())
	if err != nil {
		return nil, fmt.Errorf("parse database config: %w", err)
	}

	poolConfig.MaxConns = 10
	poolConfig.MinConns = 1

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("create database pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("connect to database: %w", err)
	}

	return pool, nil
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	return value
}
