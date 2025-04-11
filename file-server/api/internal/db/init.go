package db

import (
	"context"
	"fmt"
	"time"

	"github.com/caarlos0/env"
	"github.com/jackc/pgx/v5/pgxpool"
)

var Pool *pgxpool.Pool

type dbConfig struct {
	Host     string `env:"DB_HOST"`
	Port     string `env:"DB_PORT"`
	User     string `env:"DB_USER"`
	Password string `env:"DB_PASSWORD"`
	DBName   string `env:"DB_NAME"`
	SslMode  string `env:"SSL_MODE" envDefault:"require"`
}

func newDBConfig() *dbConfig {
	DBCfg := &dbConfig{}
	env.Parse(DBCfg)
	return DBCfg
}

var (
	maxRetries    = 5
	retryInterval = 2 * time.Second
)

func InitDB() (*pgxpool.Pool, error) {
	dbCfg := newDBConfig()
	connStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		dbCfg.Host, dbCfg.Port, dbCfg.User, dbCfg.Password, dbCfg.DBName, dbCfg.SslMode,
	)

	var err error
	for range maxRetries {
		Pool, err = pgxpool.New(context.Background(), connStr)
		if err == nil {
			break
		}
		time.Sleep(retryInterval)
	}

	if err != nil {
		return nil, err
	}

	err = Pool.Ping(context.Background())
	if err != nil {
		return nil, err
	}

	return Pool, nil
}
