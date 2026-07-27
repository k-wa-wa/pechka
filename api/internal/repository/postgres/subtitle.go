package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/k-wa-wa/pechka/api/internal/domain"
)

type SubtitleRepository struct {
	pool *pgxpool.Pool
}

func NewSubtitleRepository(pool *pgxpool.Pool) *SubtitleRepository {
	return &SubtitleRepository{pool: pool}
}

func scanTrack(row scanner) (*domain.SubtitleTrack, error) {
	var t domain.SubtitleTrack
	err := row.Scan(&t.ID, &t.ContentID, &t.Language, &t.Status, &t.Model, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

const trackColumns = "id, content_id, language, status, model, created_at, updated_at"

func (r *SubtitleRepository) ListTracksByContentID(ctx context.Context, contentID string) ([]*domain.SubtitleTrack, error) {
	rows, err := r.pool.Query(ctx,
		"SELECT "+trackColumns+" FROM subtitle_tracks WHERE content_id = $1 ORDER BY language",
		contentID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tracks []*domain.SubtitleTrack
	for rows.Next() {
		t, err := scanTrack(rows)
		if err != nil {
			return nil, err
		}
		tracks = append(tracks, t)
	}
	return tracks, rows.Err()
}

func (r *SubtitleRepository) GetTrack(ctx context.Context, trackID string) (*domain.SubtitleTrack, error) {
	row := r.pool.QueryRow(ctx, "SELECT "+trackColumns+" FROM subtitle_tracks WHERE id = $1", trackID)
	return scanTrack(row)
}

func (r *SubtitleRepository) SetTrackStatus(ctx context.Context, trackID string, status domain.SubtitleTrackStatus) (*domain.SubtitleTrack, error) {
	row := r.pool.QueryRow(ctx,
		"UPDATE subtitle_tracks SET status = $1 WHERE id = $2 RETURNING "+trackColumns,
		status, trackID,
	)
	return scanTrack(row)
}

func scanCue(row scanner) (*domain.SubtitleCue, error) {
	var c domain.SubtitleCue
	err := row.Scan(&c.ID, &c.TrackID, &c.Seq, &c.StartMs, &c.EndMs, &c.Text, &c.OriginalText, &c.Flagged, &c.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

const cueColumns = "id, track_id, seq, start_ms, end_ms, text, original_text, flagged, updated_at"

func (r *SubtitleRepository) ListCuesByTrackID(ctx context.Context, trackID string) ([]*domain.SubtitleCue, error) {
	// seq は (track_id, seq) の一意制約があるため通常重複しないが、
	// 制約が DEFERRABLE で移行期間中の重複がありうるため念のためタイブレークしておく。
	rows, err := r.pool.Query(ctx,
		"SELECT "+cueColumns+" FROM subtitle_cues WHERE track_id = $1 ORDER BY seq, start_ms, id",
		trackID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cues []*domain.SubtitleCue
	for rows.Next() {
		c, err := scanCue(rows)
		if err != nil {
			return nil, err
		}
		cues = append(cues, c)
	}
	return cues, rows.Err()
}

type UpdateCueParams struct {
	ID      string
	Text    *string
	StartMs *int
	EndMs   *int
}

func (r *SubtitleRepository) UpdateCue(ctx context.Context, params UpdateCueParams) (*domain.SubtitleCue, error) {
	setClauses := []string{}
	args := []any{}
	argIdx := 1

	if params.Text != nil {
		setClauses = append(setClauses, fmt.Sprintf("text = $%d", argIdx))
		args = append(args, *params.Text)
		argIdx++
	}
	if params.StartMs != nil {
		setClauses = append(setClauses, fmt.Sprintf("start_ms = $%d", argIdx))
		args = append(args, *params.StartMs)
		argIdx++
	}
	if params.EndMs != nil {
		setClauses = append(setClauses, fmt.Sprintf("end_ms = $%d", argIdx))
		args = append(args, *params.EndMs)
		argIdx++
	}
	// 人手で内容を確定させた行なので、以後はハルシネーション疑いのフラグを外す
	setClauses = append(setClauses, "flagged = false")

	args = append(args, params.ID)
	query := fmt.Sprintf(
		"UPDATE subtitle_cues SET %s WHERE id = $%d RETURNING "+cueColumns,
		strings.Join(setClauses, ", "), argIdx,
	)

	row := r.pool.QueryRow(ctx, query, args...)
	return scanCue(row)
}

type InsertCueParams struct {
	TrackID string
	Seq     int
	StartMs int
	EndMs   int
	Text    string
}

// InsertCue は params.Seq の位置に割り込ませる形でcueを追加する。
// 挿入位置以降のcueのseqを1つずつ後ろにシフトしてから挿入することで、
// (track_id, seq) の一意性を保ったまま順序を維持する。シフトとINSERTの間は
// 一時的にseqが重複しうるため、DEFERRABLE UNIQUE制約でコミット時にまとめて検査させる。
func (r *SubtitleRepository) InsertCue(ctx context.Context, params InsertCueParams) (*domain.SubtitleCue, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx,
		"UPDATE subtitle_cues SET seq = seq + 1 WHERE track_id = $1 AND seq >= $2",
		params.TrackID, params.Seq,
	); err != nil {
		return nil, err
	}

	row := tx.QueryRow(ctx, `
		INSERT INTO subtitle_cues (track_id, seq, start_ms, end_ms, text, original_text, flagged)
		VALUES ($1, $2, $3, $4, $5, $5, false)
		RETURNING `+cueColumns,
		params.TrackID, params.Seq, params.StartMs, params.EndMs, params.Text,
	)
	cue, err := scanCue(row)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return cue, nil
}

func (r *SubtitleRepository) DeleteCue(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, "DELETE FROM subtitle_cues WHERE id = $1", id)
	return err
}
