package postgres

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/k-wa-wa/pechka/api/internal/domain"
)

type ContentRepository struct {
	pool *pgxpool.Pool
}

func NewContentRepository(pool *pgxpool.Pool) *ContentRepository {
	return &ContentRepository{pool: pool}
}

const contentColumns = `id, short_id, content_type, channel_id, disc_id, title, description, duration_seconds, is_360, tags, status, document_key, document_mime_type, published_at, archived_at, created_at, updated_at`

type CreateContentParams struct {
	ShortID          string
	ContentType      domain.ContentType
	ChannelID        string
	DiscID           *string
	Title            string
	Description      string
	DurationSeconds  *int
	Is360            bool
	Tags             []string
	Status           domain.ContentStatus
	DocumentKey      *string
	DocumentMimeType *string
	PublishedAt      *time.Time
}

func (r *ContentRepository) Create(ctx context.Context, params CreateContentParams) (*domain.Content, error) {
	channelID := strings.TrimSpace(params.ChannelID)
	if channelID == "" {
		return nil, fmt.Errorf("channel_id is required")
	}

	return WithRetryWrite(ctx, r.pool, func() (*domain.Content, error) {
		row := r.pool.QueryRow(ctx, `
			INSERT INTO contents (short_id, content_type, channel_id, disc_id, title, description, duration_seconds, is_360, tags, status, document_key, document_mime_type, published_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
			RETURNING `+contentColumns+`
		`, params.ShortID, params.ContentType, channelID, params.DiscID, params.Title, params.Description,
			params.DurationSeconds, params.Is360, params.Tags, params.Status, params.DocumentKey, params.DocumentMimeType, params.PublishedAt)

		return scanContent(row)
	})
}

type UpdateContentParams struct {
	ID          string
	ChannelID   *string
	Title       *string
	Description *string
	Tags        []string
	Status      *domain.ContentStatus
}

func buildUpdateContentQuery(params UpdateContentParams) (string, []any, bool) {
	setClauses := []string{}
	args := []any{}
	argIdx := 1

	if params.ChannelID != nil {
		setClauses = append(setClauses, fmt.Sprintf("channel_id = $%d", argIdx))
		args = append(args, *params.ChannelID)
		argIdx++
	}
	if params.Title != nil {
		setClauses = append(setClauses, fmt.Sprintf("title = $%d", argIdx))
		args = append(args, *params.Title)
		argIdx++
	}
	if params.Description != nil {
		setClauses = append(setClauses, fmt.Sprintf("description = $%d", argIdx))
		args = append(args, *params.Description)
		argIdx++
	}
	if params.Tags != nil {
		setClauses = append(setClauses, fmt.Sprintf("tags = $%d", argIdx))
		args = append(args, params.Tags)
		argIdx++
	}
	if params.Status != nil {
		setClauses = append(setClauses, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, *params.Status)
		argIdx++
	}

	if len(setClauses) == 0 {
		return "", nil, false
	}

	args = append(args, params.ID)
	query := fmt.Sprintf(`
		UPDATE contents SET %s
		WHERE id = $%d AND deleted_at IS NULL
		RETURNING `+contentColumns+`
	`, strings.Join(setClauses, ", "), argIdx)
	return query, args, true
}

func (r *ContentRepository) Update(ctx context.Context, params UpdateContentParams) (*domain.Content, error) {
	query, args, hasUpdates := buildUpdateContentQuery(params)
	if !hasUpdates {
		return r.GetByID(ctx, params.ID)
	}

	return WithRetry(ctx, r.pool, func() (*domain.Content, error) {
		row := r.pool.QueryRow(ctx, query, args...)
		return scanContent(row)
	})
}

const deleteContentQuery = "UPDATE contents SET deleted_at = CURRENT_TIMESTAMP WHERE id = $1 AND deleted_at IS NULL"

func (r *ContentRepository) Delete(ctx context.Context, id string) error {
	return WithRetryVoid(ctx, r.pool, func() error {
		_, err := r.pool.Exec(ctx, deleteContentQuery, id)
		return err
	})
}

const (
	setArchivedTrueQuery = `
		UPDATE contents SET archived_at = CURRENT_TIMESTAMP
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING ` + contentColumns
	setArchivedFalseQuery = `
		UPDATE contents SET archived_at = NULL
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING ` + contentColumns
	getContentByIDQuery = `
		SELECT ` + contentColumns + `
		FROM contents WHERE id = $1 AND deleted_at IS NULL`
)

// SetArchived はコンテンツのアーカイブ状態を切り替える。
// 実データは削除せず archived_at にタイムスタンプを立てる/クリアするだけのアーカイブ。
// deleted_at IS NULL を条件とし、論理削除済みレコードへの誤操作・誤復元を防ぐ。
func (r *ContentRepository) SetArchived(ctx context.Context, id string, archived bool) (*domain.Content, error) {
	return WithRetry(ctx, r.pool, func() (*domain.Content, error) {
		var row scanner
		if archived {
			row = r.pool.QueryRow(ctx, setArchivedTrueQuery, id)
		} else {
			row = r.pool.QueryRow(ctx, setArchivedFalseQuery, id)
		}
		return scanContent(row)
	})
}

func (r *ContentRepository) GetByID(ctx context.Context, id string) (*domain.Content, error) {
	return WithRetry(ctx, r.pool, func() (*domain.Content, error) {
		row := r.pool.QueryRow(ctx, getContentByIDQuery, id)
		return scanContent(row)
	})
}

type ListContentsParams struct {
	Status    *domain.ContentStatus
	ChannelID *string
	// IncludeArchived が false の場合、アーカイブ済み(archived_at IS NOT NULL)のコンテンツは除外される。
	// admin画面はアーカイブ済みも含めて表示・管理するため true を指定する。
	IncludeArchived bool
	Limit           int
	Offset          int
}

func (r *ContentRepository) List(ctx context.Context, params ListContentsParams) ([]*domain.Content, error) {
	whereClause, args, argIdx := buildListContentsWhere(params)
	query := `SELECT ` + contentColumns + ` FROM contents` + whereClause

	query += fmt.Sprintf(" ORDER BY updated_at DESC LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
	args = append(args, params.Limit, params.Offset)

	return WithRetry(ctx, r.pool, func() ([]*domain.Content, error) {
		rows, err := r.pool.Query(ctx, query, args...)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		var contents []*domain.Content
		for rows.Next() {
			c, err := scanContent(rows)
			if err != nil {
				return nil, err
			}
			contents = append(contents, c)
		}
		return contents, rows.Err()
	})
}

// Count は List と同じ絞り込み条件(Status/ChannelID/IncludeArchived)に一致する件数を返す。
// Limit/Offset は無視される。
func (r *ContentRepository) Count(ctx context.Context, params ListContentsParams) (int, error) {
	whereClause, args, _ := buildListContentsWhere(params)
	query := "SELECT COUNT(*) FROM contents" + whereClause

	return WithRetry(ctx, r.pool, func() (int, error) {
		var total int
		if err := r.pool.QueryRow(ctx, query, args...).Scan(&total); err != nil {
			return 0, err
		}
		return total, nil
	})
}

func buildListContentsWhere(params ListContentsParams) (string, []any, int) {
	conditions := []string{"deleted_at IS NULL"}
	args := []any{}
	argIdx := 1

	if params.Status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, *params.Status)
		argIdx++
	}
	if params.ChannelID != nil {
		conditions = append(conditions, fmt.Sprintf("channel_id = $%d", argIdx))
		args = append(args, *params.ChannelID)
		argIdx++
	}
	if !params.IncludeArchived {
		conditions = append(conditions, "archived_at IS NULL")
	}

	return " WHERE " + strings.Join(conditions, " AND "), args, argIdx
}

const (
	// LatestPlaylistLockKey は api（複数レプリカ）と batch/refresh-latest-playlist で共有する 64-bit アドバイザリロックキー。
	LatestPlaylistLockKey int64 = 0x706563686b610001
	LatestPlaylistLimit         = 50
	LatestPlaylistTimeout       = 10 * time.Second
	LatestPlaylistKey           = "resources/hls/latest.m3u8"
	LatestPlaylistQuery         = `SELECT c.short_id, c.title, c.published_at
		 FROM contents c
		 WHERE c.status = 'ready'
		   AND c.archived_at IS NULL
		   AND c.deleted_at IS NULL
		   AND EXISTS (
		     SELECT 1 FROM video_variants v
		     WHERE v.content_id = c.id AND v.variant_type = 'master'
		   )
		 ORDER BY c.published_at DESC NULLS LAST, c.updated_at DESC, c.short_id ASC
		 LIMIT $1`
)

type PlaylistItem struct {
	ShortID     string
	Title       string
	PublishedAt *time.Time
}

type playlistRows interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
	Close()
}

type playlistTx interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (playlistRows, error)
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
}

type playlistTxBeginner interface {
	BeginTx(ctx context.Context, txOptions pgx.TxOptions) (playlistTx, error)
}

type pgxPoolPlaylistBeginner struct {
	pool *pgxpool.Pool
}

func (b pgxPoolPlaylistBeginner) BeginTx(ctx context.Context, txOptions pgx.TxOptions) (playlistTx, error) {
	tx, err := b.pool.BeginTx(ctx, txOptions)
	if err != nil {
		return nil, err
	}
	return pgxPlaylistTx{tx: tx}, nil
}

type pgxPlaylistTx struct {
	tx pgx.Tx
}

func (t pgxPlaylistTx) Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
	return t.tx.Exec(ctx, sql, arguments...)
}

func (t pgxPlaylistTx) Query(ctx context.Context, sql string, args ...any) (playlistRows, error) {
	return t.tx.Query(ctx, sql, args...)
}

func (t pgxPlaylistTx) Commit(ctx context.Context) error {
	return t.tx.Commit(ctx)
}

func (t pgxPlaylistTx) Rollback(ctx context.Context) error {
	return t.tx.Rollback(ctx)
}

func scanPlaylistItem(row scanner) (PlaylistItem, error) {
	var item PlaylistItem
	if err := row.Scan(&item.ShortID, &item.Title, &item.PublishedAt); err != nil {
		return PlaylistItem{}, err
	}
	return item, nil
}

var playlistTitleReplacer = strings.NewReplacer("\r\n", " ", "\r", " ", "\n", " ")

func sanitizePlaylistTitle(title string) string {
	return playlistTitleReplacer.Replace(title)
}

// BuildLatestPlaylist は batch/refresh-latest-playlist と同一の #EXTM3U 形式でプレイリスト文字列を生成する。
// 配信先が /resources/hls/latest.m3u8 であるため、各エントリは相対 URI（<short_id>/master.m3u8）を出力する。
func BuildLatestPlaylist(items []PlaylistItem) string {
	var buf bytes.Buffer
	buf.WriteString("#EXTM3U\n")
	for _, item := range items {
		buf.WriteString(fmt.Sprintf("#EXTINF:-1,%s\n", sanitizePlaylistTitle(item.Title)))
		buf.WriteString(fmt.Sprintf("%s/master.m3u8\n", item.ShortID))
	}
	return buf.String()
}

// RefreshLatestPlaylist は READ COMMITTED トランザクション内でアドバイザリロックを取得し、
// 最新の未削除・未アーカイブかつ ready なコンテンツ一覧を取得して uploadFn（MinIO PutObject）を完了させた後にコミット（ロック解放）する。
// S3 停滞によるロック長期保持や接続プール枯渇を防ぐため、全体処理に LatestPlaylistTimeout の期限を設ける。
func (r *ContentRepository) RefreshLatestPlaylist(ctx context.Context, uploadFn func(ctx context.Context, playlist string) error) error {
	if r.pool == nil {
		return fmt.Errorf("postgres connection pool is nil")
	}
	if uploadFn == nil {
		return fmt.Errorf("uploadFn is required")
	}
	return WithRetryVoid(ctx, r.pool, func() error {
		return refreshLatestPlaylistWithTimeout(ctx, pgxPoolPlaylistBeginner{pool: r.pool}, LatestPlaylistTimeout, uploadFn)
	})
}

func refreshLatestPlaylistWithBeginner(ctx context.Context, beginner playlistTxBeginner, uploadFn func(ctx context.Context, playlist string) error) error {
	return refreshLatestPlaylistWithTimeout(ctx, beginner, LatestPlaylistTimeout, uploadFn)
}

func refreshLatestPlaylistWithTimeout(ctx context.Context, beginner playlistTxBeginner, timeout time.Duration, uploadFn func(ctx context.Context, playlist string) error) error {
	opCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	tx, err := beginner.BeginTx(opCtx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return err
	}
	defer func() {
		// opCtx が期限切れでも接続とアドバイザリロックを確実に解放するため独立したコンテキストでロールバックする
		rollbackCtx, rollbackCancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer rollbackCancel()
		_ = tx.Rollback(rollbackCtx)
	}()

	if _, err := tx.Exec(opCtx, "SELECT pg_advisory_xact_lock($1)", LatestPlaylistLockKey); err != nil {
		return fmt.Errorf("acquire playlist advisory lock: %w", err)
	}

	rows, err := tx.Query(opCtx, LatestPlaylistQuery, LatestPlaylistLimit)
	if err != nil {
		return fmt.Errorf("query latest playlist contents: %w", err)
	}

	var items []PlaylistItem
	for rows.Next() {
		item, scanErr := scanPlaylistItem(rows)
		if scanErr != nil {
			rows.Close()
			return fmt.Errorf("scan playlist row: %w", scanErr)
		}
		items = append(items, item)
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		rows.Close()
		return fmt.Errorf("iterate playlist rows: %w", rowsErr)
	}
	rows.Close()

	playlist := BuildLatestPlaylist(items)
	if err := uploadFn(opCtx, playlist); err != nil {
		return err
	}

	if err := tx.Commit(opCtx); err != nil {
		return fmt.Errorf("commit playlist refresh transaction: %w", err)
	}
	return nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanContent(row scanner) (*domain.Content, error) {
	var c domain.Content
	err := row.Scan(
		&c.ID, &c.ShortID, &c.ContentType, &c.ChannelID, &c.DiscID, &c.Title, &c.Description,
		&c.DurationSeconds, &c.Is360, &c.Tags, &c.Status, &c.DocumentKey, &c.DocumentMimeType,
		&c.PublishedAt, &c.ArchivedAt, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &c, nil
}
