"""ffmpeg / ffprobe の薄いラッパ。

映像は Remotion が焼くので、ここが受け持つのは音声と HLS 変換だけである。

音声の尺は「計算」せず必ず ffprobe で実測する(docs/407 §2.3)。TTS エンジンが返す
理論値とファイルの実尺は一致しないことがあり、その差が動画後半での絵・字幕の
ずれとして蓄積するため。
"""

from __future__ import annotations

import json
import subprocess

# 合成した各wavはここに揃える。エンジンやモックで出力形式が違っても、
# 後段の concat が「全ファイル同一フォーマット」を前提にできるようにするため。
SAMPLE_RATE = 48000
CHANNELS = 1


class MediaError(RuntimeError):
    pass


def _run(cmd: list[str]) -> subprocess.CompletedProcess:
    proc = subprocess.run(cmd, capture_output=True, text=True)
    if proc.returncode != 0:
        raise MediaError(f"{cmd[0]} failed ({proc.returncode}): {' '.join(cmd)}\n{proc.stderr[-2000:]}")
    return proc


def run_ffmpeg(args: list[str]) -> None:
    """任意の ffmpeg 呼び出し。共通のフラグだけこちらで付ける。"""
    _run(["ffmpeg", "-y", "-loglevel", "error", *args])


def probe_duration_ms(path: str) -> int:
    proc = _run(
        [
            "ffprobe", "-v", "error",
            "-show_entries", "format=duration",
            "-of", "json",
            path,
        ]
    )
    duration = json.loads(proc.stdout)["format"]["duration"]
    return int(round(float(duration) * 1000))


def normalize_audio(src: str, dst: str, pad_ms: int) -> None:
    """wavを共通フォーマットへ揃え、末尾に無音を足す。

    無音の付与はレンダリング後ではなくここで行う。文と文の「間」も尺の一部として
    実測されるべきで、後から足すとタイムラインとファイルが食い違うため。
    """
    filters = [f"aresample={SAMPLE_RATE}", f"aformat=channel_layouts=mono"]
    if pad_ms > 0:
        filters.append(f"apad=pad_dur={pad_ms / 1000:.3f}")

    _run(
        [
            "ffmpeg", "-y", "-loglevel", "error",
            "-i", src,
            "-af", ",".join(filters),
            "-ar", str(SAMPLE_RATE), "-ac", str(CHANNELS), "-c:a", "pcm_s16le",
            dst,
        ]
    )


def silence(dst: str, duration_ms: int) -> None:
    """指定尺の無音wavを作る。TTSエンジンなしでパイプラインを通すため(--engine mock)。"""
    _run(
        [
            "ffmpeg", "-y", "-loglevel", "error",
            "-f", "lavfi",
            "-i", f"anullsrc=r={SAMPLE_RATE}:cl=mono",
            "-t", f"{duration_ms / 1000:.3f}",
            "-c:a", "pcm_s16le",
            dst,
        ]
    )


def concat_audio(list_file: str, dst: str) -> None:
    _run(
        [
            "ffmpeg", "-y", "-loglevel", "error",
            "-f", "concat", "-safe", "0",
            "-i", list_file,
            "-c:a", "pcm_s16le",
            dst,
        ]
    )
