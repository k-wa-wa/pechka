package cmd

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/k-wa-wa/pechka/batch/etl/shared"
)

type LoadConfig struct {
	DiscLabel    string
	ContentTitle string
	ContentType  string
	Is360        bool
}

func loadConfigFromEnv() LoadConfig {
	discLabel := os.Getenv("DISC_LABEL")
	if discLabel == "" {
		log.Fatalf("DISC_LABEL env var is required")
	}
	contentTitle := os.Getenv("CONTENT_TITLE")
	if contentTitle == "" {
		contentTitle = discLabel
	}
	return LoadConfig{
		DiscLabel:    discLabel,
		ContentTitle: contentTitle,
		ContentType:  os.Getenv("CONTENT_TYPE"),
		Is360:          os.Getenv("IS_360") == "true",
	}
}

func ensureDisc(ctx context.Context, db *pgxpool.Pool, label string) (string, error) {
	var id string
	err := db.QueryRow(ctx,
		"INSERT INTO discs (label) VALUES ($1) ON CONFLICT (label) DO UPDATE SET label = EXCLUDED.label RETURNING id",
		label,
	).Scan(&id)
	return id, err
}

func upsertContent(ctx context.Context, db *pgxpool.Pool, proposedShortID, discID string, cfg LoadConfig) (contentID string, finalShortID string, err error) {
	err = db.QueryRow(ctx,
		`INSERT INTO contents (short_id, content_type, disc_id, title, status, is_360)
		 VALUES ($1, $2, $3, $4, 'processing', $5)
		 ON CONFLICT (disc_id) WHERE disc_id IS NOT NULL
		 DO UPDATE SET status = 'processing', updated_at = NOW()
		 RETURNING id, short_id`,
		proposedShortID, cfg.ContentType, discID, cfg.ContentTitle, cfg.Is360,
	).Scan(&contentID, &finalShortID)
	if err != nil {
		return "", "", fmt.Errorf("failed to upsert content: %w", err)
	}
	if finalShortID != proposedShortID {
		log.Printf("Existing content found for disc_id=%s title=%q. Reusing existing short_id=%s id=%s", discID, cfg.ContentTitle, finalShortID, contentID)
	} else {
		log.Printf("New content created for disc_id=%s title=%q: short_id=%s id=%s", discID, cfg.ContentTitle, finalShortID, contentID)
	}
	return contentID, finalShortID, nil
}

func markContentReady(ctx context.Context, db *pgxpool.Pool, contentID string) error {
	_, err := db.Exec(ctx,
		"UPDATE contents SET status = 'ready', published_at = NOW() WHERE id = $1",
		contentID,
	)
	return err
}

func getContentIDByShortID(ctx context.Context, db *pgxpool.Pool, shortID string) (string, error) {
	var id string
	err := db.QueryRow(ctx, "SELECT id FROM contents WHERE short_id = $1", shortID).Scan(&id)
	return id, err
}

func RunLoad(ctx context.Context, osArgs []string) error {
	fs := flag.NewFlagSet("load", flag.ContinueOnError)
	phase := fs.String("phase", "all", "Execution phase: init, finalize, or all")
	if err := fs.Parse(osArgs); err != nil {
		return err
	}

	cfg := loadConfigFromEnv()
	if cfg.ContentType == "" {
		cfg.ContentType = "video"
	}

	shortID := os.Getenv("SHORT_ID")
	if shortID == "" {
		return fmt.Errorf("SHORT_ID env var is required")
	}

	dsn := shared.GetPostgresDSN()
	if dsn == "" {
		return fmt.Errorf("failed to get database DSN from environment")
	}

	db, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return fmt.Errorf("failed to connect to PostgreSQL: %w", err)
	}
	defer db.Close()

	var contentID string

	if *phase == "init" || *phase == "all" {
		discID, err := ensureDisc(ctx, db, cfg.DiscLabel)
		if err != nil {
			return fmt.Errorf("failed to ensure disc: %w", err)
		}

		var finalShortID string
		contentID, finalShortID, err = upsertContent(ctx, db, shortID, discID, cfg)
		if err != nil {
			return fmt.Errorf("failed to upsert content: %w", err)
		}
		shortID = finalShortID
		log.Printf("Content initialized: id=%s short_id=%s", contentID, shortID)

		if err := os.WriteFile("/tmp/content-id", []byte(contentID), 0644); err != nil {
			log.Printf("WARNING: failed to write /tmp/content-id: %v", err)
		}
	}

	if *phase == "finalize" || *phase == "all" {
		contentID = os.Getenv("CONTENT_ID")
		if contentID == "" {
			var err error
			contentID, err = getContentIDByShortID(ctx, db, shortID)
			if err != nil {
				return fmt.Errorf("failed to find content ID by short ID %s: %w", shortID, err)
			}
		}

		if err := markContentReady(ctx, db, contentID); err != nil {
			return fmt.Errorf("failed to mark content ready: %w", err)
		}
		log.Printf("Content marked ready: id=%s short_id=%s", contentID, shortID)
	}

	if err := os.WriteFile("/tmp/short-id", []byte(shortID), 0644); err != nil {
		log.Printf("WARNING: failed to write /tmp/short-id: %v", err)
	}

	log.Printf("Load complete: short_id=%s phase=%s", shortID, *phase)
	return nil
}
