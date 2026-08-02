"""HLS とサムネイルを S3 互換ストレージ(MinIO)へ置く。

キーの構造は既存の配信経路に合わせる(docs/201 §5.2)。Nginx が
`/resources/hls/*` をこのバケットへ流すため、ここを外すと再生できない。
"""

from __future__ import annotations

import os
from dataclasses import dataclass
from pathlib import Path

import boto3

CONTENT_TYPES = {
    ".m3u8": "application/x-mpegURL",
    ".ts": "video/MP2T",
    ".jpg": "image/jpeg",
    ".jpeg": "image/jpeg",
    ".png": "image/png",
}


def hls_prefix(short_id: str) -> str:
    return f"resources/hls/{short_id}"


def thumbnail_key(short_id: str) -> str:
    return f"thumbnails/{short_id}/thumb_01.jpg"


@dataclass
class Storage:
    bucket: str
    client: object

    @classmethod
    def from_env(cls) -> "Storage":
        url = _require_env("MINIO_URL").rstrip("/")
        scheme = "https" if os.environ.get("MINIO_USE_SSL", "false").lower() == "true" else "http"
        client = boto3.client(
            "s3",
            endpoint_url=f"{scheme}://{url}",
            aws_access_key_id=_require_env("MINIO_ACCESS_KEY"),
            aws_secret_access_key=_require_env("MINIO_SECRET_KEY"),
        )
        return cls(bucket=_require_env("MINIO_BUCKET"), client=client)

    def put_file(self, path: str, key: str) -> None:
        ctype = CONTENT_TYPES.get(Path(path).suffix.lower(), "application/octet-stream")
        self.client.upload_file(path, self.bucket, key, ExtraArgs={"ContentType": ctype})

    def put_text(self, body: str, key: str) -> None:
        ctype = CONTENT_TYPES.get(Path(key).suffix.lower(), "text/plain")
        self.client.put_object(
            Bucket=self.bucket, Key=key, Body=body.encode("utf-8"), ContentType=ctype
        )

    def put_dir(self, local_dir: str, prefix: str) -> int:
        """ディレクトリ直下のファイルを prefix 配下へ置き、置いた数を返す。"""
        count = 0
        for entry in sorted(Path(local_dir).iterdir()):
            if entry.is_file():
                self.put_file(str(entry), f"{prefix}/{entry.name}")
                count += 1
        return count


def _require_env(name: str) -> str:
    value = os.environ.get(name)
    if not value:
        raise SystemExit(f"{name} env var is required")
    return value
