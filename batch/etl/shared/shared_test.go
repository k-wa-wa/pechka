package shared

import (
	"testing"
)

func TestBuildVariantInfo(t *testing.T) {
	tests := []struct {
		variantType string
		wantBW      int
		wantRes     string
		wantCodecs  string
	}{
		{
			variantType: "1080p",
			wantBW:      6192000,
			wantRes:     "1920x1080",
			wantCodecs:  "avc1.640028,mp4a.40.2",
		},
		{
			variantType: "720p",
			wantBW:      3128000,
			wantRes:     "1280x720",
			wantCodecs:  "avc1.4d001f,mp4a.40.2",
		},
		{
			variantType: "480p",
			wantBW:      1628000,
			wantRes:     "854x480",
			wantCodecs:  "avc1.42e01f,mp4a.40.2",
		},
		{
			variantType: "audio",
			wantBW:      192000,
			wantRes:     "",
			wantCodecs:  "mp4a.40.2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.variantType, func(t *testing.T) {
			vi := BuildVariantInfo(tt.variantType)
			if vi.VariantType != tt.variantType {
				t.Errorf("expected %s, got %s", tt.variantType, vi.VariantType)
			}
			if vi.Bandwidth == nil || *vi.Bandwidth != tt.wantBW {
				t.Errorf("bandwidth mismatch: got %v, want %d", vi.Bandwidth, tt.wantBW)
			}
			if tt.wantRes != "" {
				if vi.Resolution == nil || *vi.Resolution != tt.wantRes {
					t.Errorf("resolution mismatch: got %v, want %s", vi.Resolution, tt.wantRes)
				}
			} else if vi.Resolution != nil {
				t.Errorf("expected nil resolution, got %v", *vi.Resolution)
			}
			if vi.Codecs == nil || *vi.Codecs != tt.wantCodecs {
				t.Errorf("codecs mismatch: got %v, want %s", vi.Codecs, tt.wantCodecs)
			}
		})
	}

	t.Run("original", func(t *testing.T) {
		vi := BuildVariantInfo("original")
		if vi.VariantType != "original" {
			t.Errorf("expected original, got %s", vi.VariantType)
		}
		if vi.Bandwidth != nil || vi.Resolution != nil || vi.Codecs != nil {
			t.Errorf("expected nil fields for original, got %+v", vi)
		}
	})

	t.Run("custom", func(t *testing.T) {
		custom := BuildVariantInfo("custom")
		if custom.VariantType != "custom" {
			t.Errorf("expected custom, got %s", custom.VariantType)
		}
		if custom.Bandwidth != nil || custom.Resolution != nil {
			t.Errorf("expected nil fields for custom, got %+v", custom)
		}
	})
}

func TestGetPostgresDSN(t *testing.T) {
	t.Run("POSTGRES_DSN direct override", func(t *testing.T) {
		t.Setenv("POSTGRES_DSN", "postgres://localhost/test")
		if dsn := GetPostgresDSN(); dsn != "postgres://localhost/test" {
			t.Errorf("unexpected DSN: %s", dsn)
		}
	})

	t.Run("Empty DB_HOST returns empty string", func(t *testing.T) {
		t.Setenv("POSTGRES_DSN", "")
		t.Setenv("DB_HOST", "")
		if dsn := GetPostgresDSN(); dsn != "" {
			t.Errorf("expected empty string, got %s", dsn)
		}
	})

	t.Run("DB_* env vars with custom port and sslmode", func(t *testing.T) {
		t.Setenv("POSTGRES_DSN", "")
		t.Setenv("DB_HOST", "pg.internal")
		t.Setenv("DB_PORT", "5433")
		t.Setenv("DB_USER", "pechka")
		t.Setenv("DB_PASSWORD", "secret")
		t.Setenv("DB_NAME", "pechkadb")
		t.Setenv("SSL_MODE", "verify-full")

		expected := "host=pg.internal port=5433 user=pechka password=secret dbname=pechkadb sslmode=verify-full"
		if dsn := GetPostgresDSN(); dsn != expected {
			t.Errorf("expected %q, got %q", expected, dsn)
		}
	})

	t.Run("DB_* env vars with defaults", func(t *testing.T) {
		t.Setenv("POSTGRES_DSN", "")
		t.Setenv("DB_HOST", "127.0.0.1")
		t.Setenv("DB_PORT", "")
		t.Setenv("DB_USER", "postgres")
		t.Setenv("DB_PASSWORD", "pass")
		t.Setenv("DB_NAME", "test")
		t.Setenv("SSL_MODE", "")

		expected := "host=127.0.0.1 port=5432 user=postgres password=pass dbname=test sslmode=disable"
		if dsn := GetPostgresDSN(); dsn != expected {
			t.Errorf("expected %q, got %q", expected, dsn)
		}
	})
}

func TestMinioConfigFromEnv(t *testing.T) {
	t.Setenv("MINIO_BUCKET", "b1")
	t.Setenv("MINIO_URL", "http://localhost:9000")
	t.Setenv("MINIO_ACCESS_KEY", "access")
	t.Setenv("MINIO_SECRET_KEY", "secret")
	t.Setenv("MINIO_USE_SSL", "true")

	cfg := MinioConfigFromEnv()
	if cfg.Bucket != "b1" || cfg.URL != "http://localhost:9000" || cfg.AccessKey != "access" || cfg.SecretKey != "secret" || !cfg.UseSSL {
		t.Errorf("unexpected config: %+v", cfg)
	}
}

func TestHelpers(t *testing.T) {
	i := IntPtr(42)
	if i == nil || *i != 42 {
		t.Errorf("IntPtr failed: %v", i)
	}

	s := StrPtr("hello")
	if s == nil || *s != "hello" {
		t.Errorf("StrPtr failed: %v", s)
	}
}
