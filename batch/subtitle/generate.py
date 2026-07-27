#!/usr/bin/env python3
"""字幕生成バッチ: MinIO上のMKVから音声を抽出し、faster-whisperで文字起こしして
subtitle_tracks / subtitle_cues に登録する（status='draft'のまま。曲の特定・
歌詞の正誤判定は未実装のため、admin画面での人手レビューを経て published にする）。"""

import argparse
import os
import subprocess
import sys
import tempfile
from pathlib import Path

import boto3
import psycopg2
import psycopg2.extras

WHISPER_MODEL = "large-v3"
CPU_THREADS = int(os.environ.get("CPU_THREADS", "4"))
# PoCで判明した通り、歌唱を含むため vad_filter は常に False で固定する
# (VAD有効だと歌唱区間がほぼ丸ごと「非発話」として落ちるため)。
VAD_FILTER = False
# 連続して同一テキストがこの回数以上続いたらハルシネーション疑いとしてフラグする
# (PoCで観測した「ご視聴ありがとうございました」×3連続のようなループ対策)。
# 歌唱パートでは「Yeah Yeah」等の正当な2連続反復も頻出するため、閾値は3以上とし
# 誤検知(flagged乱発)を抑える。
HALLUCINATION_REPEAT_THRESHOLD = 3


def extract_audio(source_url: str, output_wav: str) -> None:
    """ffmpegにMinIOの署名付きURLを直接渡し、フルダウンロードせず音声のみ抽出する。"""
    subprocess.run(
        [
            "ffmpeg", "-y",
            "-i", source_url,
            "-vn", "-ac", "1", "-ar", "16000", "-acodec", "pcm_s16le",
            output_wav,
        ],
        check=True,
    )


def transcribe(wav_path: str) -> list[dict]:
    from faster_whisper import WhisperModel

    print(f"Loading faster-whisper {WHISPER_MODEL} (int8, cpu, {CPU_THREADS} threads)...")
    model = WhisperModel(WHISPER_MODEL, device="cpu", compute_type="int8", cpu_threads=CPU_THREADS)

    print(f"Starting transcription (vad_filter={VAD_FILTER})...")
    segments, info = model.transcribe(
        wav_path,
        language="ja",
        vad_filter=VAD_FILTER,
        beam_size=1,
        condition_on_previous_text=False,
    )
    print(f"Detected language={info.language} prob={info.language_probability:.2f}")

    results = []
    for seg in segments:
        text = seg.text.strip()
        results.append({"start_ms": int(seg.start * 1000), "end_ms": int(seg.end * 1000), "text": text})
        print(f"  [{seg.start:7.2f} -> {seg.end:7.2f}] {text}")
    return results


def flag_hallucinations(segments: list[dict]) -> list[dict]:
    """連続する同一テキストをハルシネーション疑いとしてフラグする。
    歌詞のサビ反復のような正当な繰り返しは非連続（間に他の歌詞が挟まる）ため誤検知しにくい。"""
    run_start = 0
    for i in range(1, len(segments) + 1):
        same_as_prev = i < len(segments) and segments[i]["text"] == segments[run_start]["text"]
        if not same_as_prev:
            run_len = i - run_start
            if run_len >= HALLUCINATION_REPEAT_THRESHOLD:
                for j in range(run_start, i):
                    segments[j]["flagged"] = True
            run_start = i
    for seg in segments:
        seg.setdefault("flagged", False)
    return segments


def find_unsafe_overwrite_reason(conn, content_id: str, language: str) -> str | None:
    """既存トラックが人手レビュー済み/編集済みの場合、上書きしてよい理由がないので
    None以外を返す（呼び出し側は処理を中断する）。バッチは常にトラックを丸ごと
    作り直す設計のため、ここで弾かないとadmin画面での修正が再実行一発で消えてしまう。"""
    with conn.cursor() as cur:
        cur.execute(
            "SELECT id, status FROM subtitle_tracks WHERE content_id = %s AND language = %s",
            (content_id, language),
        )
        row = cur.fetchone()
        if row is None:
            return None
        track_id, status = row

        if status == "published":
            return f"existing track {track_id} is published"

        cur.execute(
            "SELECT COUNT(*) FROM subtitle_cues WHERE track_id = %s AND text != original_text",
            (track_id,),
        )
        edited_count = cur.fetchone()[0]
        if edited_count > 0:
            return f"existing track {track_id} has {edited_count} manually edited cue(s)"

    return None


def register_track(conn, content_id: str, language: str, segments: list[dict]) -> None:
    with conn.cursor() as cur:
        # 再実行時は既存トラックを丸ごと入れ替える（cueはON DELETE CASCADEで消える）。
        # find_unsafe_overwrite_reason() で published/編集済みでないことを事前に確認済み。
        cur.execute(
            "DELETE FROM subtitle_tracks WHERE content_id = %s AND language = %s",
            (content_id, language),
        )
        cur.execute(
            """
            INSERT INTO subtitle_tracks (content_id, language, status, model)
            VALUES (%s, %s, 'draft', %s)
            RETURNING id
            """,
            (content_id, language, WHISPER_MODEL),
        )
        track_id = cur.fetchone()[0]

        rows = [
            (track_id, seq, seg["start_ms"], seg["end_ms"], seg["text"], seg["text"], seg["flagged"])
            for seq, seg in enumerate(segments)
        ]
        psycopg2.extras.execute_values(
            cur,
            """
            INSERT INTO subtitle_cues (track_id, seq, start_ms, end_ms, text, original_text, flagged)
            VALUES %s
            """,
            rows,
        )
    conn.commit()
    flagged_count = sum(1 for s in segments if s["flagged"])
    print(f"Registered track {track_id}: {len(segments)} cues ({flagged_count} flagged) [status=draft]")


def main() -> None:
    parser = argparse.ArgumentParser(description="Subtitle Generator")
    parser.add_argument("--input", required=True, help="Object key to source MKV file on MinIO")
    parser.add_argument("--content-id", required=True, help="Content UUID")
    parser.add_argument("--language", default="ja", help="Subtitle language code (default: ja)")
    args = parser.parse_args()

    bucket = os.environ["MINIO_BUCKET"]
    minio_url = os.environ["MINIO_URL"]
    access_key = os.environ["MINIO_ACCESS_KEY"]
    secret_key = os.environ["MINIO_SECRET_KEY"]
    use_ssl = os.environ.get("MINIO_USE_SSL", "false").lower() == "true"
    postgres_dsn = os.environ["POSTGRES_DSN"]

    # ダウンロード・文字起こし(数十分かかりうる)を始める前に、上書きしてよいか確認する。
    guard_conn = psycopg2.connect(postgres_dsn)
    try:
        reason = find_unsafe_overwrite_reason(guard_conn, args.content_id, args.language)
    finally:
        guard_conn.close()
    if reason is not None:
        print(f"Refusing to overwrite: {reason}. Aborting without re-transcribing.", file=sys.stderr)
        sys.exit(1)

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
        ExpiresIn=21600,  # フル尺の文字起こしは数十分かかるため長めに確保する
    )

    with tempfile.TemporaryDirectory() as tmpdir:
        wav_path = str(Path(tmpdir) / "audio.wav")
        print("Extracting audio (16kHz mono) via ffmpeg streaming from presigned URL...")
        extract_audio(presigned_url, wav_path)

        segments = transcribe(wav_path)
        if not segments:
            print("No segments transcribed. Exiting without registering a track.")
            return

        segments = flag_hallucinations(segments)

    conn = psycopg2.connect(postgres_dsn)
    try:
        register_track(conn, args.content_id, args.language, segments)
    finally:
        conn.close()


if __name__ == "__main__":
    main()
