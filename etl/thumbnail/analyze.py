#!/usr/bin/env python3
"""Thumbnail Analyzer: samples frames from an MKV/HLS source and selects
the brightest, most representative frames as thumbnails, then uploads to MinIO."""

import argparse
import os
import subprocess
import tempfile
from pathlib import Path

import boto3
import psycopg2

SAMPLE_INTERVAL_SEC = 120
MAX_THUMBNAILS = 5


def extract_frames(input_path: str, output_dir: str, interval: int) -> list[str]:
    """Extract frames every `interval` seconds using ffmpeg."""
    output_pattern = str(Path(output_dir) / "frame_%04d.jpg")
    subprocess.run(
        [
            "ffmpeg", "-i", input_path,
            "-vf", f"fps=1/{interval},scale=320:-1",
            "-q:v", "2",
            output_pattern,
        ],
        check=True,
        capture_output=True,
    )
    return sorted(Path(output_dir).glob("frame_*.jpg"))


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
        key = f"thumbnails/{short_id}/thumb_{i:02d}.jpg"
        client.upload_file(path, bucket, key, ExtraArgs={"ContentType": "image/jpeg"})
        keys.append(key)
    return keys


def register_assets(conn, content_id: str, s3_keys: list[str]) -> None:
    with conn.cursor() as cur:
        for key in s3_keys:
            cur.execute(
                "INSERT INTO assets (content_id, asset_role, s3_key) VALUES (%s, 'thumbnail', %s)",
                (content_id, key),
            )
    conn.commit()


def main() -> None:
    parser = argparse.ArgumentParser(description="Thumbnail Analyzer")
    parser.add_argument("--input", required=True, help="Path to source MKV file")
    parser.add_argument("--short-id", required=True, help="Content short_id")
    parser.add_argument("--content-id", required=True, help="Content UUID")
    args = parser.parse_args()

    bucket = os.environ["MINIO_BUCKET"]
    minio_url = os.environ["MINIO_URL"]
    access_key = os.environ["MINIO_ACCESS_KEY"]
    secret_key = os.environ["MINIO_SECRET_KEY"]
    use_ssl = os.environ.get("MINIO_USE_SSL", "false").lower() == "true"
    postgres_dsn = os.environ["POSTGRES_DSN"]

    with tempfile.TemporaryDirectory() as tmpdir:
        print(f"Extracting frames from {args.input}...")
        frames = extract_frames(args.input, tmpdir, SAMPLE_INTERVAL_SEC)
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

    conn = psycopg2.connect(postgres_dsn)
    try:
        register_assets(conn, args.content_id, s3_keys)
        print("Assets registered in PostgreSQL.")
    finally:
        conn.close()


if __name__ == "__main__":
    main()
