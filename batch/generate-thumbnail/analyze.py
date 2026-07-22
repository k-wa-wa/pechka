#!/usr/bin/env python3
"""Thumbnail Analyzer: samples frames from an MKV/HLS source and selects
the brightest, most representative frames as thumbnails, then uploads to MinIO."""

import argparse
import os
import subprocess
import sys
import tempfile
from pathlib import Path

import boto3
import psycopg2

SAMPLE_INTERVAL_SEC = 120
MAX_THUMBNAILS = 5
THUMBNAIL_PREFIX = "thumbnails/"


def get_duration_seconds(input_path: str) -> float:
    """Probe media duration via ffprobe (reads container metadata only, no decode)."""
    result = subprocess.run(
        [
            "ffprobe", "-v", "error",
            "-show_entries", "format=duration",
            "-of", "default=noprint_wrappers=1:nokey=1",
            input_path,
        ],
        capture_output=True, text=True,
    )
    try:
        return float(result.stdout.strip())
    except ValueError:
        return 0.0


def extract_frame_at(input_path: str, timestamp: float, output_path: Path) -> bool:
    """Seek to `timestamp` (fast input seek) and grab a single frame."""
    try:
        subprocess.run(
            [
                "ffmpeg", "-ss", str(timestamp), "-i", input_path,
                "-frames:v", "1",
                "-vf", "scale=320:-1",
                "-pix_fmt", "yuvj420p",
                "-q:v", "2",
                str(output_path),
            ],
            check=True,
            capture_output=True,
        )
    except subprocess.CalledProcessError as e:
        print(f"ffmpeg failed to extract frame at {timestamp}s:", file=sys.stderr)
        print("stdout:", e.stdout.decode(), file=sys.stderr)
        print("stderr:", e.stderr.decode(), file=sys.stderr)
        return False
    return output_path.exists()


def extract_frames(input_path: str, output_dir: str, interval: int) -> list[Path]:
    """Seek to evenly spaced timestamps and extract one frame at each, instead of
    sequentially decoding the entire source just to sample a handful of frames."""
    duration = get_duration_seconds(input_path)
    timestamps = list(range(0, int(duration), interval)) if duration > 0 else [0]

    frames = []
    for i, ts in enumerate(timestamps, 1):
        out_path = Path(output_dir) / f"frame_{i:04d}.jpg"
        if extract_frame_at(input_path, ts, out_path):
            frames.append(out_path)

    return sorted(frames)


def brightness_score(path: str) -> float:
    """Compute mean brightness using ffprobe/ffmpeg."""
    result = subprocess.run(
        [
            "ffprobe", "-v", "quiet",
            "-show_entries", "frame_tags=lavfi.signalstats.YAVG",
            "-f", "lavfi",
            f"movie={path},signalstats",
            "-of", "default=noprint_wrappers=1:nokey=1",
        ],
        capture_output=True, text=True,
    )
    try:
        return float(result.stdout.strip())
    except ValueError:
        return 0.0


def select_thumbnails(frames: list[str], max_count: int) -> list[str]:
    """Select up to max_count frames with highest brightness, avoiding black frames."""
    scored = [(f, brightness_score(str(f))) for f in frames]
    # Filter out near-black frames (brightness < 10)
    scored = [(f, s) for f, s in scored if s >= 10.0]
    scored.sort(key=lambda x: -x[1])
    # Pick evenly distributed from top half of brightness
    selected = [str(f) for f, _ in scored[:max_count]]
    return selected


def upload_thumbnails(
    thumbnails: list[str],
    short_id: str,
    bucket: str,
    minio_url: str,
    access_key: str,
    secret_key: str,
    use_ssl: bool,
) -> list[str]:
    """Upload thumbnails to MinIO and return their S3 keys."""
    endpoint = minio_url.rstrip("/")
    client = boto3.client(
        "s3",
        endpoint_url=("https://" if use_ssl else "http://") + endpoint,
        aws_access_key_id=access_key,
        aws_secret_access_key=secret_key,
    )
    keys = []
    for i, path in enumerate(thumbnails, 1):
        key = f"{THUMBNAIL_PREFIX}{short_id}/thumb_{i:02d}.jpg"
        client.upload_file(path, bucket, key, ExtraArgs={"ContentType": "image/jpeg"})
        keys.append(key)
    return keys


def register_assets(conn, content_id: str, s3_keys: list[str], thumbnail_key: str) -> None:
    with conn.cursor() as cur:
        for key in s3_keys:
            cur.execute(
                """
                INSERT INTO assets (content_id, asset_role, s3_key)
                VALUES (%s, 'thumbnail', %s)
                ON CONFLICT (content_id, asset_role, s3_key) DO NOTHING
                """,
                (content_id, key),
            )
        # contents.thumbnail_key に代表サムネイルを設定する。
        # CDC (Benthos) が contents テーブルの変更を購読しているため、
        # ここに書けば自動的に MongoDB の表示用ドキュメントへ反映される。
        # フロントエンドは `/thumbnails/${thumbnail_key}` として参照する
        # (nginx 側の rewrite が bucket 直下の "thumbnails/" プレフィックスを
        # 付与するため、ここには含めない)。
        cur.execute(
            "UPDATE contents SET thumbnail_key = %s WHERE id = %s",
            (thumbnail_key, content_id),
        )
    conn.commit()


def main() -> None:
    parser = argparse.ArgumentParser(description="Thumbnail Analyzer")
    parser.add_argument("--input", required=True, help="Object key to source MKV file on MinIO")
    parser.add_argument("--short-id", required=True, help="Content short_id")
    parser.add_argument("--content-id", required=True, help="Content UUID")
    args = parser.parse_args()

    bucket = os.environ["MINIO_BUCKET"]
    minio_url = os.environ["MINIO_URL"]
    access_key = os.environ["MINIO_ACCESS_KEY"]
    secret_key = os.environ["MINIO_SECRET_KEY"]
    use_ssl = os.environ.get("MINIO_USE_SSL", "false").lower() == "true"
    postgres_dsn = os.environ["POSTGRES_DSN"]

    # Generate presigned URL for ffmpeg input
    endpoint = minio_url.rstrip("/")
    s3_client = boto3.client(
        "s3",
        endpoint_url=("https://" if use_ssl else "http://") + endpoint,
        aws_access_key_id=access_key,
        aws_secret_access_key=secret_key,
    )
    
    print(f"Generating presigned URL for MinIO object: {args.input}...")
    presigned_url = s3_client.generate_presigned_url(
        "get_object",
        Params={"Bucket": bucket, "Key": args.input},
        ExpiresIn=3600,
    )

    with tempfile.TemporaryDirectory() as tmpdir:
        print("Seeking to sample points and extracting frames...")
        frames = extract_frames(presigned_url, tmpdir, SAMPLE_INTERVAL_SEC)
        if not frames:
            print("No frames extracted. Exiting.")
            return

        print(f"Extracted {len(frames)} frames. Selecting thumbnails...")
        selected = select_thumbnails(frames, MAX_THUMBNAILS)
        if not selected:
            print("No suitable frames found (all frames too dark). Exiting.")
            return
        print(f"Selected {len(selected)} thumbnails.")

        print("Uploading thumbnails to MinIO...")
        s3_keys = upload_thumbnails(
            selected, args.short_id, bucket, minio_url, access_key, secret_key, use_ssl
        )
        print(f"Uploaded: {s3_keys}")

    # frontend が `/thumbnails/${thumbnail_key}` として参照する値
    # (nginx が bucket 直下の "thumbnails/" プレフィックスを付与するため除去する)
    thumbnail_key = s3_keys[0].removeprefix(THUMBNAIL_PREFIX)

    conn = psycopg2.connect(postgres_dsn)
    try:
        register_assets(conn, args.content_id, s3_keys, thumbnail_key)
        print("Assets registered in PostgreSQL.")
    finally:
        conn.close()


if __name__ == "__main__":
    main()
