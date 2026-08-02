"""Remotion を呼び出して mp4 を焼く。

映像の合成は Remotion に任せる。音声も Remotion 側で多重化するため、ここでは
manifest を props として渡すだけでよい(docs/407 §2.4)。
"""

from __future__ import annotations

import json
import shutil
import subprocess
from pathlib import Path

# remotion プロジェクトの場所。batch-tech-feed/remotion/ に置いてある。
PROJECT_DIR = Path(__file__).resolve().parent.parent / "remotion"
ENTRY = "src/index.ts"
COMPOSITION = "Digest"


class RenderError(RuntimeError):
    pass


def run(
    manifest: dict,
    out_dir: str,
    dst: str,
    concurrency: int | None = None,
    crf: int = 20,
    quiet: bool = False,
) -> str:
    if not (PROJECT_DIR / "node_modules").is_dir():
        raise RenderError(
            f"{PROJECT_DIR}/node_modules not found; run `npm install` in that directory first"
        )
    if shutil.which("npx") is None:
        raise RenderError("npx not found on PATH (Node.js is required to render)")

    # props はファイル渡しにする。manifest は画像を data URI で抱えるため、
    # コマンドライン引数に載せると容易に長さ制限を超える。
    #
    # 中身は必ず Composition の props の形 ({"manifest": ...}) にすること。manifest を
    # 直に書くと defaultProps とマージされて既定値が生き残り、尺が既定のまま
    # 静かに短い動画が出てくる(エラーにならないので気づきにくい)。
    props_path = Path(out_dir) / "props.json"
    props_path.write_text(
        json.dumps({"manifest": manifest}, ensure_ascii=False), encoding="utf-8"
    )

    cmd = [
        "npx", "remotion", "render", ENTRY, COMPOSITION, str(Path(dst).resolve()),
        f"--props={props_path.resolve()}",
        # ナレーション wav をブラウザから読ませるため、作業ディレクトリを public 扱いにする。
        # 画像は manifest 内で data URI に畳んであるので、ここに置く必要があるのは音声だけ。
        f"--public-dir={Path(out_dir).resolve()}",
        f"--crf={crf}",
    ]
    if concurrency:
        cmd.append(f"--concurrency={concurrency}")
    if quiet:
        cmd.append("--log=error")

    proc = subprocess.run(cmd, cwd=PROJECT_DIR)
    if proc.returncode != 0:
        raise RenderError(f"remotion render failed with exit code {proc.returncode}")
    return dst
