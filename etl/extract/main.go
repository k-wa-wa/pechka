package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Config struct {
	Device      string
	NFSMkvDir   string
	PostgresDSN string
}

func mustGetenv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("required env var %s is not set", key)
	}
	return v
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func postgresDSN() string {
	if dsn := os.Getenv("POSTGRES_DSN"); dsn != "" {
		return dsn
	}
	host := mustGetenv("DB_HOST")
	port := getenv("DB_PORT", "5432")
	user := mustGetenv("DB_USER")
	password := mustGetenv("DB_PASSWORD")
	dbname := mustGetenv("DB_NAME")
	sslmode := getenv("SSL_MODE", "disable")
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		host, port, user, password, dbname, sslmode)
}

func configFromEnv() Config {
	return Config{
		Device:      mustGetenv("DEVICE"),
		NFSMkvDir:   mustGetenv("NFS_MKV_DIR"),
		PostgresDSN: postgresDSN(),
	}
}

func main() {
	cfg := configFromEnv()
	ctx := context.Background()

	discLabel := getDiscLabel(cfg.Device)

	outputDir := filepath.Join(cfg.NFSMkvDir, discLabel)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Fatalf("failed to create output directory: %v", err)
	}

	log.Printf("Extracting disc: %s", discLabel)
	cmd := exec.Command("makemkvcon", "mkv", "disc:0", "all", outputDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Fatalf("makemkvcon failed: %v", err)
	}

	db, err := pgxpool.New(ctx, cfg.PostgresDSN)
	if err != nil {
		log.Fatalf("failed to connect to PostgreSQL: %v", err)
	}
	defer db.Close()

	_, err = db.Exec(ctx,
		"INSERT INTO discs (label) VALUES ($1) ON CONFLICT (label) DO NOTHING",
		discLabel,
	)
	if err != nil {
		log.Fatalf("failed to register disc: %v", err)
	}

	log.Printf("Disc %s registered in database.", discLabel)
}

func getDiscLabel(device string) string {
	out, err := exec.Command("blkid", device).Output()
	if err == nil {
		for _, part := range strings.Fields(string(out)) {
			if strings.HasPrefix(part, "LABEL=") {
				return strings.Trim(strings.TrimPrefix(part, "LABEL="), "\"")
			}
		}
	}
	label := fmt.Sprintf("DISC_%s", time.Now().Format("20060102_150405"))
	log.Printf("WARNING: Could not get disc label, using generated: %s", label)
	return label
}
