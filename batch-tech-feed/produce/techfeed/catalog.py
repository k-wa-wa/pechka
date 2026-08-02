"""PostgreSQL のカタログへ登録する。

PostgreSQL が Single Source of Truth で、そこから Benthos CDC が MongoDB と
Elasticsearch へ流す(docs/201 §2)。よってここに入れるだけで一覧・検索・再生の
すべてに載る。MongoDB へ直接書いてはいけない。
"""

from __future__ import annotations

import hashlib
import os
import re

import psycopg2

CONTENT_TYPE = "video"
DEFAULT_TAGS = ["tech-feed"]
# contents.short_id は VARCHAR(50)。
SHORT_ID_MAX = 50


def short_id_for(source_key: str) -> str:
    """取り込み元のキーから short_id を決定的に作る。

    Bluray 取り込みは disc_id のユニーク制約で「再取り込みしても増えない」を担保して
    いるが、こちらはディスクを持たない。かといって**共有スキーマに列を足したくない**ので、
    既にある `contents.short_id` の UNIQUE 制約をそのまま同一性の担保に使う。
    source_key から決定的に導けば、再実行は必ず同じ行に当たる。

    short_id はコード上どこでも文字列として扱われている(数値化している箇所はない)ため、
    Snowflake 形式である必要はない。むしろ日次フィードでは
    `/contents/tech-feed-2026-08-01` の方が URL として分かりやすい。
    """
    slug = re.sub(r"[^a-zA-Z0-9]+", "-", source_key).strip("-").lower()
    if not slug:
        slug = "tech-feed"
    if len(slug) <= SHORT_ID_MAX:
        return slug
    # 切り詰めると別のキー同士が衝突しうるので、末尾にハッシュを付けて区別する。
    digest = hashlib.sha1(source_key.encode("utf-8")).hexdigest()[:8]
    return slug[: SHORT_ID_MAX - 9].rstrip("-") + "-" + digest


def dsn_from_env() -> str:
    if dsn := os.environ.get("POSTGRES_DSN"):
        return dsn
    host = os.environ.get("DB_HOST")
    if not host:
        raise SystemExit("POSTGRES_DSN or DB_HOST env var is required")
    return (
        f"host={host} port={os.environ.get('DB_PORT', '5432')} "
        f"user={os.environ.get('DB_USER', '')} password={os.environ.get('DB_PASSWORD', '')} "
        f"dbname={os.environ.get('DB_NAME', '')} sslmode={os.environ.get('SSL_MODE', 'disable')}"
    )


def connect():
    return psycopg2.connect(dsn_from_env())


def upsert_content(
    conn,
    source_key: str,
    title: str,
    description: str,
    duration_seconds: int,
    tags: list[str] | None = None,
) -> tuple[str, str]:
    """(content_id, short_id) を返す。同じ source_key なら既存行を更新する。

    冪等性は `contents.short_id` の UNIQUE 制約に乗せている(short_id_for を参照)。
    スキーマに手を入れずに済ませるための選択で、Bluray 取り込みの disc_id とは
    独立した仕組みになっている。
    """
    with conn.cursor() as cur:
        cur.execute(
            """
            INSERT INTO contents
                (short_id, content_type, title, description, status, duration_seconds, tags)
            VALUES (%s, %s, %s, %s, 'processing', %s, %s)
            ON CONFLICT (short_id)
            DO UPDATE SET
                title = EXCLUDED.title,
                description = EXCLUDED.description,
                duration_seconds = EXCLUDED.duration_seconds,
                tags = EXCLUDED.tags,
                status = 'processing',
                updated_at = NOW()
            RETURNING id, short_id
            """,
            (
                short_id_for(source_key),
                CONTENT_TYPE,
                title,
                description,
                duration_seconds,
                tags if tags is not None else DEFAULT_TAGS,
            ),
        )
        content_id, short_id = cur.fetchone()
    conn.commit()
    return content_id, short_id


def register_variant(
    conn,
    content_id: str,
    short_id: str,
    variant_type: str,
    bandwidth: int | None,
    resolution: str | None,
    codecs: str | None,
) -> None:
    hls_key = f"resources/hls/{short_id}/{variant_type}.m3u8"
    with conn.cursor() as cur:
        cur.execute(
            """
            INSERT INTO video_variants (content_id, variant_type, hls_key, bandwidth, resolution, codecs)
            VALUES (%s, %s, %s, %s, %s, %s)
            ON CONFLICT (content_id, variant_type)
            DO UPDATE SET hls_key = EXCLUDED.hls_key, bandwidth = EXCLUDED.bandwidth,
                          resolution = EXCLUDED.resolution, codecs = EXCLUDED.codecs
            """,
            (content_id, variant_type, hls_key, bandwidth, resolution, codecs),
        )
    conn.commit()


def set_thumbnail(conn, content_id: str, key: str) -> None:
    with conn.cursor() as cur:
        cur.execute("UPDATE contents SET thumbnail_key = %s WHERE id = %s", (key, content_id))
    conn.commit()


def mark_ready(conn, content_id: str) -> None:
    """視聴可能にする。published_at は初回だけ入れ、再実行で日付が動かないようにする。"""
    with conn.cursor() as cur:
        cur.execute(
            "UPDATE contents SET status = 'ready', published_at = COALESCE(published_at, NOW()) WHERE id = %s",
            (content_id,),
        )
    conn.commit()
