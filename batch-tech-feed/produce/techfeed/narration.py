"""文ごとの wav を1本のナレーション音声にまとめる。

映像との合成は Remotion 側が行う(docs/407 §2.4)。Python 側の責務は
「実尺の確定した音声」と「その実尺から導いた manifest」を渡すところまで。
"""

from __future__ import annotations

from pathlib import Path

from . import media
from .timeline import Entry

NARRATION_NAME = "narration.wav"


def _concat_line(path: str) -> str:
    # concat デマクサのフォーマット上、パス中のシングルクォートはエスケープが必要。
    escaped = str(path).replace("'", "'\\''")
    return f"file '{escaped}'\n"


def build(entries: list[Entry], out_dir: str) -> str:
    """連結したナレーション wav を out_dir 直下に作り、そのファイル名を返す。

    絶対パスではなくファイル名を返すのは、Remotion 側が out_dir を public ディレクトリ
    として受け取り、staticFile() で参照するためである。
    """
    work = Path(out_dir)
    work.mkdir(parents=True, exist_ok=True)

    audio_list = work / "audio.txt"
    audio_list.write_text(
        "".join(_concat_line(e.audio_path) for e in entries), encoding="utf-8"
    )

    dst = work / NARRATION_NAME
    media.concat_audio(str(audio_list), str(dst))
    return NARRATION_NAME
