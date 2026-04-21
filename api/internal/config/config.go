package config

import (
	"log"
	"os"
)

type Config struct {
	PostgresDSN      string
	MongoURL         string
	MongoDB          string
	ElasticsearchURL string
	Port             string
}

func Load() *Config {
	return &Config{
		PostgresDSN:      mustEnv("POSTGRES_DSN"),
		MongoURL:         mustEnv("MONGO_URL"),
		MongoDB:          mustEnv("MONGO_DB"),
		ElasticsearchURL: mustEnv("ELASTICSEARCH_URL"),
		Port:             mustEnv("PORT"),
	}
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("required env %s is not set", key)
	}
	return v
}
