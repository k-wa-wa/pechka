"""生成した mp4 を HLS のバリアントへ変換する。

pechka のプレーヤー(hls.js)は `video_variants` に登録されたバリアントを見て
画質を選ぶため、mp4 のままでは再生経路に乗らない(docs/201 §3.5)。

Bluray 取り込みの ETL とは意図的にコードを共有していない。ジョブとして完全に
独立させるためで、その代償として ABR ラダーの定義がここと `batch/etl` の2箇所に
存在する。ラダーの中身は用途が違うので、揃える必要もない(下記)。
"""

from __future__ import annotations

import os
from dataclasses import dataclass
from pathlib import Path

from . import media

SEGMENT_SECONDS = 6


@dataclass(frozen=True)
class Variant:
    name: str
    bandwidth: int | None
    resolution: str | None
    codecs: str | None


# 画面収録に近い内容(文字・コード・図)なので、Bluray 側のラダーとは狙いが違う。
# 480p まで落とすと文字が潰れて読めず、帯域を節約する意味がないため作らない。
#   original : 再エンコードなしのストリームコピー。Remotion の出力は h264/yuv420p なのでそのまま通る
#   720p     : 帯域が細い環境向け。文字が辛うじて読める下限
VARIANTS: tuple[Variant, ...] = (
    Variant("original", None, None, None),
    Variant("720p", 3128000, "1280x720", "avc1.4d001f,mp4a.40.2"),
)


def _hls_args(out_dir: Path, name: str) -> list[str]:
    return [
        "-f", "hls",
        "-hls_time", str(SEGMENT_SECONDS),
        "-hls_list_size", "0",
        "-hls_segment_filename", str(out_dir / f"{name}_%04d.ts"),
        str(out_dir / f"{name}.m3u8"),
    ]


def transcode(mp4: str, out_dir: str) -> list[Variant]:
    """mp4 から各バリアントの m3u8 と ts を作り、作れたバリアントを返す。"""
    work = Path(out_dir)
    work.mkdir(parents=True, exist_ok=True)

    for variant in VARIANTS:
        if variant.name == "original":
            # 映像は作り直さない。音声だけ HLS が扱える AAC に揃える。
            args = ["-i", mp4, "-c:v", "copy", "-c:a", "aac", "-b:a", "192k"]
        else:
            args = [
                "-i", mp4,
                "-vf", "scale=1280:720",
                "-c:v", "libx264", "-preset", "fast",
                "-b:v", "3000k", "-maxrate", "3500k", "-bufsize", "6000k",
                "-c:a", "aac", "-b:a", "128k",
            ]
        media.run_ffmpeg(args + _hls_args(work, variant.name))
        print(f"  transcoded {variant.name}")

    return list(VARIANTS)


def master_playlist(variants: list[Variant]) -> str:
    """master.m3u8 を組み立てる。プレーヤーはこれを見て画質を選ぶ。"""
    lines = ["#EXTM3U", "#EXT-X-VERSION:3", ""]
    for v in variants:
        if v.name == "original":
            # 元品質は帯域が読めないので 0 を入れる。既存 ETL の master と同じ扱い。
            lines.append('#EXT-X-STREAM-INF:BANDWIDTH=0,CODECS="avc1.640028,mp4a.40.2"')
        else:
            lines.append(
                f"#EXT-X-STREAM-INF:BANDWIDTH={v.bandwidth},"
                f"RESOLUTION={v.resolution},CODECS=\"{v.codecs}\""
            )
        lines.append(f"{v.name}.m3u8")
        lines.append("")
    return "\n".join(lines)


def thumbnail(mp4: str, dst: str, at_seconds: float = 2.0) -> str:
    """先頭付近のフレームをサムネイルとして切り出す。

    冒頭はタイトルスライドなので、動画を解析して「良いフレーム」を選ぶ
    (`batch/generate-thumbnail` の手法)必要がない。
    """
    os.makedirs(os.path.dirname(dst) or ".", exist_ok=True)
    media.run_ffmpeg([
        "-ss", f"{at_seconds:.2f}", "-i", mp4,
        "-frames:v", "1", "-q:v", "3", dst,
    ])
    return dst
