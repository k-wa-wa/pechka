"""焼き上がった mp4 を pechka の配信経路に載せる。

    mp4 ──▶ HLS 変換 ──▶ MinIO ──▶ PostgreSQL ──▶ (Benthos CDC) ──▶ MongoDB / Elasticsearch
                                                                        │
                                                                        ▼
                                                                    pechka で再生

`contents.status` は最後に `ready` へ上げる。アップロードの途中で落ちても、
中途半端なコンテンツが一覧に出ないようにするためである。
"""

from __future__ import annotations

import tempfile
from pathlib import Path

from . import catalog, hls, storage


def run(
    manifest: dict,
    mp4: str,
    source_key: str,
    description: str = "",
    keep_work: str | None = None,
) -> dict:
    if not Path(mp4).is_file():
        raise SystemExit(f"{mp4} not found; run the 'build' step first")

    title = manifest.get("title") or "技術ダイジェスト"
    duration_seconds = round(manifest.get("total_ms", 0) / 1000)

    store = storage.Storage.from_env()
    conn = catalog.connect()
    try:
        content_id, short_id = catalog.upsert_content(
            conn, source_key, title, description, duration_seconds
        )
        print(f"content: id={content_id} short_id={short_id} source_key={source_key}")

        with tempfile.TemporaryDirectory(prefix="techfeed-hls-") as tmp:
            work = keep_work or tmp
            Path(work).mkdir(parents=True, exist_ok=True)

            print("transcoding to HLS...")
            variants = hls.transcode(mp4, work)

            prefix = storage.hls_prefix(short_id)
            print(f"uploading HLS to s3://{store.bucket}/{prefix}/ ...")
            uploaded = store.put_dir(work, prefix)
            store.put_text(hls.master_playlist(variants), f"{prefix}/master.m3u8")
            print(f"  {uploaded} file(s) + master.m3u8")

            thumb = Path(work) / "thumb_01.jpg"
            hls.thumbnail(mp4, str(thumb))
            thumb_key = storage.thumbnail_key(short_id)
            store.put_file(str(thumb), thumb_key)
            catalog.set_thumbnail(conn, content_id, thumb_key)
            print(f"  thumbnail -> {thumb_key}")

        # master も再生対象として登録する。プレーヤーは既定でこれを掴む。
        catalog.register_variant(conn, content_id, short_id, "master", None, None, None)
        for v in variants:
            catalog.register_variant(
                conn, content_id, short_id, v.name, v.bandwidth, v.resolution, v.codecs
            )
        print(f"registered {len(variants) + 1} variant(s)")

        catalog.mark_ready(conn, content_id)
        print(f"content is ready: /contents/{short_id}")
    finally:
        conn.close()

    return {"content_id": content_id, "short_id": short_id}
