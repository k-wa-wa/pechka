package config

import (
	"log/slog"
	"os"
)

type Config struct {
	PostgresDSN      string
	MongoURL         string
	MongoDB          string
	ElasticsearchURL string
	Port             string
	AllowedIPRange   string
	MinioURL         string
	MinioBucket      string
	MinioAccessKey   string
	MinioSecretKey   string
	MinioUseSSL      bool
}

func Load() *Config {
	return &Config{
		PostgresDSN:      mustEnv("POSTGRES_DSN"),
		MongoURL:         mustEnv("MONGO_URL"),
		MongoDB:          mustEnv("MONGO_DB"),
		ElasticsearchURL: mustEnv("ELASTICSEARCH_URL"),
		Port:             mustEnv("PORT"),
		AllowedIPRange:   os.Getenv("ALLOWED_IP_RANGE"),
		MinioURL:         mustEnv("MINIO_URL"),
		MinioBucket:      mustEnv("MINIO_BUCKET"),
		MinioAccessKey:   mustEnv("MINIO_ACCESS_KEY"),
		MinioSecretKey:   mustEnv("MINIO_SECRET_KEY"),
		MinioUseSSL:      os.Getenv("MINIO_USE_SSL") == "true",
	}
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		slog.Error("required env var is not set", "key", key)
		os.Exit(1)
	}
	return v
}
