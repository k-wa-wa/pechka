package db

import (
	"context"
	"fmt"
	"time"

	"github.com/caarlos0/env"
	"github.com/jackc/pgx/v5"
)

var DB *pgx.Conn

type dBConfig struct {
	Host     string `env:"DB_HOST"`
	Port     string `env:"DB_PORT"`
	User     string `env:"DB_USER"`
	Password string `env:"DB_PASSWORD"`
	DBName   string `env:"DB_NAME"`
}

func newDBConfig() *dBConfig {
	DBCfg := &dBConfig{}
	env.Parse(DBCfg)
	return DBCfg
}

var (
	maxRetries    = 5
	retryInterval = 2 * time.Second
)

func InitDB() (*pgx.Conn, error) {
	dbCfg := newDBConfig()
	connStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=require",
		dbCfg.Host, dbCfg.Port, dbCfg.User, dbCfg.Password, dbCfg.DBName,
	)

	var err error
	for i := 0; i < maxRetries; i++ {
		DB, err = pgx.Connect(context.Background(), connStr)
		if err == nil {
			break
		}
		time.Sleep(retryInterval)
	}

	if err != nil {
		return nil, err
	}

	err = DB.Ping(context.Background())
	if err != nil {
		return nil, err
	}

	return DB, nil
}
