package config

import (
	"testing"
)

func TestConfigLoad(t *testing.T) {
	// Setup env
	t.Setenv("POSTGRES_DSN", "postgres://localhost/test")
	t.Setenv("MONGO_URL", "mongodb://localhost:27017")
	t.Setenv("MONGO_DB", "pechka_test")
	t.Setenv("ELASTICSEARCH_URL", "http://localhost:9200")
	t.Setenv("PORT", "8080")
	t.Setenv("ALLOWED_IP_RANGE", "192.168.1.0/24")
	t.Setenv("MINIO_URL", "http://localhost:9000")
	t.Setenv("MINIO_BUCKET", "media")
	t.Setenv("MINIO_ACCESS_KEY", "minio")
	t.Setenv("MINIO_SECRET_KEY", "miniopass")
	t.Setenv("MINIO_USE_SSL", "true")

	cfg := Load()
	if cfg.PostgresDSN != "postgres://localhost/test" {
		t.Errorf("unexpected PostgresDSN: %s", cfg.PostgresDSN)
	}
	if cfg.Port != "8080" {
		t.Errorf("unexpected Port: %s", cfg.Port)
	}
	if !cfg.MinioUseSSL {
		t.Errorf("expected MinioUseSSL true")
	}
	if cfg.AllowedIPRange != "192.168.1.0/24" {
		t.Errorf("unexpected AllowedIPRange: %s", cfg.AllowedIPRange)
	}
}

func TestMustEnv(t *testing.T) {
	t.Setenv("TEST_KEY", "val")
	if v := mustEnv("TEST_KEY"); v != "val" {
		t.Errorf("expected val, got %s", v)
	}
}
