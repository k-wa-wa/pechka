package shared

import (
	"testing"
)

func TestBuildVariantInfo(t *testing.T) {
	v1080 := BuildVariantInfo("1080p")
	if v1080.VariantType != "1080p" {
		t.Errorf("expected 1080p, got %s", v1080.VariantType)
	}
	if v1080.Bandwidth == nil || *v1080.Bandwidth != 6192000 {
		t.Errorf("unexpected bandwidth: %v", v1080.Bandwidth)
	}

	custom := BuildVariantInfo("custom")
	if custom.VariantType != "custom" {
		t.Errorf("expected custom, got %s", custom.VariantType)
	}
}

func TestGetPostgresDSN(t *testing.T) {
	t.Setenv("POSTGRES_DSN", "postgres://localhost/test")
	if dsn := GetPostgresDSN(); dsn != "postgres://localhost/test" {
		t.Errorf("unexpected DSN: %s", dsn)
	}
}

func TestMinioConfigFromEnv(t *testing.T) {
	t.Setenv("MINIO_BUCKET", "b1")
	t.Setenv("MINIO_URL", "http://localhost:9000")
	cfg := MinioConfigFromEnv()
	if cfg.Bucket != "b1" || cfg.URL != "http://localhost:9000" {
		t.Errorf("unexpected config: %+v", cfg)
	}
}
