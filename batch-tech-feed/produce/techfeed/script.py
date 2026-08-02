"""台本(script.json)の読み込みと検証。

台本はこのパイプラインの単一の中間表現であり、動画・記事・字幕がすべてここから
派生する(docs/407 §2.2)。したがって形式の妥当性はレンダリング前にここで弾き切り、
後段(TTS・Remotion・ffmpeg)が不正な入力を意識しなくて済むようにする。
"""

from __future__ import annotations

import base64
import json
import mimetypes
from dataclasses import dataclass, field
from pathlib import Path

LAYOUTS = ("title", "bullets", "code", "figure", "diagram")

# 画像は台本からの相対パスで書き、読み込み時に data URI へ畳む。
# file:// で開いたページからローカル画像を参照する際のパス解決とサンドボックスの
# 問題を避けるため。外部URLは取り込まない(docs/407 §2.1 の方針)。
IMAGE_SUFFIXES = (".png", ".jpg", ".jpeg", ".webp", ".gif", ".svg")


class ScriptError(ValueError):
    """台本の形式が不正であることを示す。パスを含めて呼び出し側が場所を特定できるようにする。"""


@dataclass
class Source:
    title: str
    url: str
    publisher: str = ""


@dataclass
class Line:
    """ナレーション1文。TTS・スライド遷移・字幕cueの共通単位である(docs/407 §2.3)。"""

    text: str
    # bullets レイアウトで、この文が言及している項目のindex。
    # None は「スライドの状態を進めない」を意味し、1項目について複数文で語れるようにする。
    focus: int | None = None


@dataclass
class Slide:
    layout: str
    title: str = ""
    subtitle: str = ""
    items: list[str] = field(default_factory=list)
    code: str = ""
    language: str = ""
    # data URI に畳んだ画像。figure レイアウト、または bullets の2カラム表示で使う。
    image: str = ""
    caption: str = ""
    # Mermaid のソース。ブラウザ内で描画する(docs/407 §2.4)。
    diagram: str = ""


@dataclass
class Section:
    seq: int
    slide: Slide
    narration: list[Line]
    sources: list[Source] = field(default_factory=list)


@dataclass
class Script:
    digest_date: str
    title: str
    sections: list[Section]

    @property
    def line_count(self) -> int:
        return sum(len(s.narration) for s in self.sections)


def _require(cond: bool, where: str, msg: str) -> None:
    if not cond:
        raise ScriptError(f"{where}: {msg}")


def _load_image(ref: str, base_dir: Path, where: str) -> str:
    _require(
        not ref.startswith(("http://", "https://")),
        where,
        "remote images are not fetched; place the file next to the script and reference it by path",
    )
    if ref.startswith("data:"):
        return ref

    path = (base_dir / ref).resolve()
    _require(path.is_file(), where, f"image not found: {path}")
    _require(
        path.suffix.lower() in IMAGE_SUFFIXES,
        where,
        f"unsupported image type {path.suffix!r} (expected one of {IMAGE_SUFFIXES})",
    )
    mime = mimetypes.guess_type(path.name)[0] or "application/octet-stream"
    return f"data:{mime};base64," + base64.b64encode(path.read_bytes()).decode("ascii")


def _parse_slide(raw: dict, where: str, base_dir: Path) -> Slide:
    layout = raw.get("layout")
    _require(layout in LAYOUTS, where, f"layout must be one of {LAYOUTS}, got {layout!r}")

    image = raw.get("image", "")
    slide = Slide(
        layout=layout,
        title=raw.get("title", ""),
        subtitle=raw.get("subtitle", ""),
        items=list(raw.get("items", [])),
        code=raw.get("code", ""),
        language=raw.get("language", ""),
        image=_load_image(image, base_dir, f"{where}.image") if image else "",
        caption=raw.get("caption", ""),
        diagram=raw.get("diagram", ""),
    )
    if layout == "bullets":
        _require(bool(slide.items), where, "bullets layout requires a non-empty 'items'")
    if layout == "code":
        _require(bool(slide.code), where, "code layout requires 'code'")
    if layout == "figure":
        _require(bool(slide.image), where, "figure layout requires 'image'")
    if layout == "diagram":
        _require(bool(slide.diagram), where, "diagram layout requires 'diagram' (mermaid source)")
    return slide


def _parse_narration(raw: list, slide: Slide, where: str) -> list[Line]:
    _require(isinstance(raw, list) and bool(raw), where, "narration must be a non-empty list")

    lines: list[Line] = []
    for i, item in enumerate(raw):
        at = f"{where}.narration[{i}]"
        # 文字列だけの略記も許す。focus を使わないレイアウト(title/code)では台本が書きやすい。
        if isinstance(item, str):
            _require(bool(item.strip()), at, "must not be empty")
            lines.append(Line(text=item))
            continue

        _require(isinstance(item, dict), at, "must be a string or an object")
        text = item.get("text")
        # 空文をTTSに投げると尺0の音声ができ、タイムラインに無意味なcueが増えるため弾く。
        _require(isinstance(text, str) and bool(text.strip()), at, "'text' must be a non-empty string")

        focus = item.get("focus")
        if focus is not None:
            _require(isinstance(focus, int), at, "'focus' must be an integer or null")
            # focus はスライド項目のindexなので、範囲外は台本のtypoとして早期に弾く。
            _require(
                slide.layout == "bullets",
                at,
                f"'focus' is only meaningful for the bullets layout (got {slide.layout})",
            )
            _require(
                0 <= focus < len(slide.items),
                at,
                f"'focus' out of range: {focus} not in [0, {len(slide.items)})",
            )
        lines.append(Line(text=text, focus=focus))
    return lines


def parse(raw: dict, base_dir: Path | None = None) -> Script:
    """台本を検証して読み込む。base_dir は画像の相対パスの起点(既定はカレント)。"""
    _require(isinstance(raw, dict), "$", "script must be a JSON object")
    base = base_dir or Path.cwd()

    sections_raw = raw.get("sections")
    _require(
        isinstance(sections_raw, list) and bool(sections_raw), "$", "'sections' must be a non-empty list"
    )

    sections: list[Section] = []
    for i, sec in enumerate(sections_raw):
        where = f"$.sections[{i}]"
        _require(isinstance(sec, dict), where, "must be an object")

        slide = _parse_slide(sec.get("slide") or {}, f"{where}.slide", base)
        narration = _parse_narration(sec.get("narration") or [], slide, where)
        sources = [
            Source(
                title=s.get("title", ""),
                url=s.get("url", ""),
                publisher=s.get("publisher", ""),
            )
            for s in sec.get("sources", [])
        ]
        sections.append(
            Section(seq=sec.get("seq", i + 1), slide=slide, narration=narration, sources=sources)
        )

    return Script(
        digest_date=raw.get("digest_date", ""),
        title=raw.get("title", ""),
        sections=sections,
    )


def load(path: str | Path, assets_dir: str | Path | None = None) -> Script:
    """台本を読む。assets_dir を渡すと画像の相対パスをそこから解決する。

    台本を ConfigMap で渡す場合、画像は ConfigMap に載せられないためイメージ側に
    焼き込むことになる。そのとき台本の位置と画像の位置が別になるので分離できるようにしてある。
    """
    p = Path(path)
    with open(p, encoding="utf-8") as f:
        return parse(json.load(f), base_dir=Path(assets_dir) if assets_dir else p.parent)
