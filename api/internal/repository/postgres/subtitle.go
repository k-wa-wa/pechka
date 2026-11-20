package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
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

const (
	listTracksQuery = "SELECT st.id, st.content_id, st.language, st.status, st.model, st.created_at, st.updated_at " +
		"FROM subtitle_tracks st JOIN contents c ON c.id = st.content_id " +
		"WHERE st.content_id = $1 AND c.deleted_at IS NULL ORDER BY st.language"

	getTrackQuery = "SELECT st.id, st.content_id, st.language, st.status, st.model, st.created_at, st.updated_at " +
		"FROM subtitle_tracks st JOIN contents c ON c.id = st.content_id " +
		"WHERE st.id = $1 AND c.deleted_at IS NULL"

	setTrackStatusQuery = "UPDATE subtitle_tracks st SET status = $1 FROM contents c " +
		"WHERE st.id = $2 AND st.content_id = c.id AND c.deleted_at IS NULL " +
		"RETURNING st.id, st.content_id, st.language, st.status, st.model, st.created_at, st.updated_at"

	listCuesQuery = "SELECT cu.id, cu.track_id, cu.seq, cu.start_ms, cu.end_ms, cu.text, cu.original_text, cu.flagged, cu.updated_at " +
		"FROM subtitle_cues cu " +
		"JOIN subtitle_tracks st ON st.id = cu.track_id " +
		"JOIN contents c ON c.id = st.content_id " +
		"WHERE cu.track_id = $1 AND c.deleted_at IS NULL ORDER BY cu.seq, cu.start_ms, cu.id"

	insertCueCheckParentQuery = "SELECT 1 FROM subtitle_tracks st JOIN contents c ON c.id = st.content_id " +
		"WHERE st.id = $1 AND c.deleted_at IS NULL FOR SHARE"

	deleteCueQuery = "DELETE FROM subtitle_cues cu USING subtitle_tracks st, contents c " +
		"WHERE cu.id = $1 AND cu.track_id = st.id AND st.content_id = c.id AND c.deleted_at IS NULL"
)

func (r *SubtitleRepository) ListTracksByContentID(ctx context.Context, contentID string) ([]*domain.SubtitleTrack, error) {
	return WithRetry(ctx, r.pool, func() ([]*domain.SubtitleTrack, error) {
		rows, err := r.pool.Query(ctx, listTracksQuery, contentID)
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
	})
}

func (r *SubtitleRepository) GetTrack(ctx context.Context, trackID string) (*domain.SubtitleTrack, error) {
	return WithRetry(ctx, r.pool, func() (*domain.SubtitleTrack, error) {
		row := r.pool.QueryRow(ctx, getTrackQuery, trackID)
		return scanTrack(row)
	})
}

func (r *SubtitleRepository) SetTrackStatus(ctx context.Context, trackID string, status domain.SubtitleTrackStatus) (*domain.SubtitleTrack, error) {
	return WithRetry(ctx, r.pool, func() (*domain.SubtitleTrack, error) {
		row := r.pool.QueryRow(ctx, setTrackStatusQuery, status, trackID)
		return scanTrack(row)
	})
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
	return WithRetry(ctx, r.pool, func() ([]*domain.SubtitleCue, error) {
		// seq は (track_id, seq) の一意制約があるため通常重複しないが、
		// 制約が DEFERRABLE で移行期間中の重複がありうるため念のためタイブレークしておく。
		rows, err := r.pool.Query(ctx, listCuesQuery, trackID)
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
	})
}

type UpdateCueParams struct {
	ID      string
	Text    *string
	StartMs *int
	EndMs   *int
}

func buildUpdateCueQuery(params UpdateCueParams) (string, []any) {
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
		"UPDATE subtitle_cues cu SET %s FROM subtitle_tracks st, contents c WHERE cu.id = $%d AND cu.track_id = st.id AND st.content_id = c.id AND c.deleted_at IS NULL RETURNING cu.id, cu.track_id, cu.seq, cu.start_ms, cu.end_ms, cu.text, cu.original_text, cu.flagged, cu.updated_at",
		strings.Join(setClauses, ", "), argIdx,
	)
	return query, args
}

func (r *SubtitleRepository) UpdateCue(ctx context.Context, params UpdateCueParams) (*domain.SubtitleCue, error) {
	return WithRetry(ctx, r.pool, func() (*domain.SubtitleCue, error) {
		query, args := buildUpdateCueQuery(params)
		row := r.pool.QueryRow(ctx, query, args...)
		return scanCue(row)
	})
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
	return WithRetryWrite(ctx, r.pool, func() (*domain.SubtitleCue, error) {
		tx, err := r.pool.Begin(ctx)
		if err != nil {
			return nil, err
		}
		defer tx.Rollback(ctx)

		var ok int
		if err := tx.QueryRow(ctx, insertCueCheckParentQuery, params.TrackID).Scan(&ok); err != nil {
			return nil, err
		}

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
	})
}

func (r *SubtitleRepository) DeleteCue(ctx context.Context, id string) error {
	return WithRetryVoid(ctx, r.pool, func() error {
		tag, err := r.pool.Exec(ctx, deleteCueQuery, id)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return pgx.ErrNoRows
		}
		return nil
	})
}
