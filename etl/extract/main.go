package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Config struct {
	Device      string
	NFSMkvDir   string
	PostgresDSN string
}

type MkvFile struct {
	MkvPath string `json:"mkv-path"`
	Label   string `json:"label"`
	Title   string `json:"title"`
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
		Device:      os.Getenv("DEVICE"),
		NFSMkvDir:   mustGetenv("NFS_MKV_DIR"),
		PostgresDSN: postgresDSN(),
	}
}

func writeOutputs(label string, mkvFiles []MkvFile) {
	if err := os.WriteFile("/tmp/bluray-label", []byte(label), 0644); err != nil {
		log.Printf("WARNING: failed to write /tmp/bluray-label: %v", err)
	}
	data, err := json.Marshal(mkvFiles)
	if err != nil {
		log.Printf("WARNING: failed to marshal mkv files: %v", err)
		data = []byte("[]")
	}
	if err := os.WriteFile("/tmp/mkv-files.json", data, 0644); err != nil {
		log.Printf("WARNING: failed to write /tmp/mkv-files.json: %v", err)
	}
}

func scanMkvFiles(dir, label string) ([]MkvFile, error) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var res []MkvFile
	for _, f := range files {
		if !f.IsDir() && strings.HasSuffix(f.Name(), ".mkv") {
			name := strings.TrimSuffix(f.Name(), ".mkv")
			res = append(res, MkvFile{
				MkvPath: filepath.Join(dir, f.Name()),
				Label:   label,
				Title:   name,
			})
		}
	}
	return res, nil
}

func main() {
	cfg := configFromEnv()
	ctx := context.Background()

	var discLabel string
	manualLabel := os.Getenv("DISC_LABEL")

	if manualLabel != "" {
		log.Printf("Manual mode. Using label: %s", manualLabel)
		discLabel = manualLabel
	} else {
		log.Printf("Auto mode. Detecting disc on device: %s", cfg.Device)
		if cfg.Device == "" {
			log.Fatal("DEVICE env var is required in auto mode")
		}
		var err error
		discLabel, err = getDiscLabel(cfg.Device)
		if err != nil {
			log.Printf("No disc detected: %v. Skipping extraction.", err)
			writeOutputs("", []MkvFile{})
			os.Exit(0)
		}
	}

	outputDir := filepath.Join(cfg.NFSMkvDir, discLabel)

	if manualLabel == "" {
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
	} else {
		log.Printf("Skipping extraction. Scanning existing files in: %s", outputDir)
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

	mkvFiles, err := scanMkvFiles(outputDir, discLabel)
	if err != nil {
		log.Fatalf("failed to scan MKV files: %v", err)
	}
	log.Printf("Found %d MKV files", len(mkvFiles))

	writeOutputs(discLabel, mkvFiles)
}

func getDiscLabel(device string) (string, error) {
	out, err := exec.Command("blkid", device).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to run blkid: %w, output: %s", err, string(out))
	}
	for _, part := range strings.Fields(string(out)) {
		if strings.HasPrefix(part, "LABEL=") {
			return strings.Trim(strings.TrimPrefix(part, "LABEL="), "\""), nil
		}
	}
	return "", fmt.Errorf("LABEL not found in blkid output")
}
