"""台本 + 音声の実尺からタイムラインを組み立てる。

このパイプラインには「手で打つタイムライン」が存在しない(docs/407 §2.3)。
スライドの切り替え時刻も字幕cueの時刻も、すべて合成された音声の実測尺から機械的に
導出する。ゆえに同じ台本から作り直せば常に同じ尺の動画になる(冪等)。
"""

from __future__ import annotations

from dataclasses import asdict, dataclass, field

from .script import Script, Section


@dataclass
class SlideState:
    """スライドの「ある瞬間の見た目」。1文につき1状態進む(docs/407 §2.4)。"""

    layout: str
    header: str
    title: str
    subtitle: str = ""
    items: list[str] = field(default_factory=list)
    # items のうち何件目までを表示するか。箇条書きを1項目ずつ出すために使う。
    revealed: int = 0
    # 現在読み上げている項目のindex。ハイライト表示に使う。
    highlight: int | None = None
    code: str = ""
    language: str = ""
    image: str = ""
    caption: str = ""
    diagram: str = ""
    sources: list[str] = field(default_factory=list)
    # セクション番号/総数。進捗インジケータに使う。
    section_seq: int = 1
    section_total: int = 1



@dataclass
class Entry:
    """ナレーション1文に対応する、音声・スライド・字幕の一体となった単位。"""

    seq: int
    section_seq: int
    text: str
    state: SlideState
    # 以下は音声合成が終わって初めて確定する。
    audio_path: str = ""
    start_ms: int = 0
    end_ms: int = 0

    @property
    def duration_ms(self) -> int:
        return self.end_ms - self.start_ms


def _states_for_section(section: Section, header: str, total: int) -> list[SlideState]:
    """セクション内の各文に対応するスライド状態を作る。"""
    slide = section.slide
    sources = [s.publisher or s.url for s in section.sources if (s.publisher or s.url)]

    base = SlideState(
        layout=slide.layout,
        header=header,
        title=slide.title,
        subtitle=slide.subtitle,
        items=list(slide.items),
        code=slide.code,
        language=slide.language,
        image=slide.image,
        caption=slide.caption,
        diagram=slide.diagram,
        sources=sources,
        section_seq=section.seq,
        section_total=total,
    )

    states: list[SlideState] = []
    revealed = 0
    highlight: int | None = None
    for line in section.narration:
        if slide.layout == "bullets" and line.focus is not None:
            # 既に出した項目は引っ込めない。focus は「ここまで出して、ここを強調」を意味する。
            revealed = max(revealed, line.focus + 1)
            highlight = line.focus
        # focus が None のときは状態を据え置く。1項目を複数文で語れるようにするため。
        states.append(
            SlideState(
                **{**asdict(base), "revealed": revealed, "highlight": highlight}
            )
        )
    return states


def build(script: Script) -> list[Entry]:
    """台本からタイムラインの骨格を作る。この時点では尺は未確定(すべて0)。"""
    total = len(script.sections)
    header = script.title

    entries: list[Entry] = []
    seq = 0
    for section in script.sections:
        states = _states_for_section(section, header, total)
        for line, state in zip(section.narration, states):
            entries.append(
                Entry(seq=seq, section_seq=section.seq, text=line.text, state=state)
            )
            seq += 1
    return entries


def apply_durations(entries: list[Entry], durations_ms: list[int]) -> int:
    """実測した各文の尺を積み上げて start/end を確定させ、総尺(ms)を返す。

    尺は「計算」ではなく ffprobe による「実測」を渡すこと。想定尺と実ファイルの尺が
    ずれると、動画が進むにつれて字幕とスライドが少しずつ後ろへずれていくため。
    """
    if len(entries) != len(durations_ms):
        raise ValueError(
            f"entry/duration count mismatch: {len(entries)} entries, {len(durations_ms)} durations"
        )

    cursor = 0
    for entry, dur in zip(entries, durations_ms):
        entry.start_ms = cursor
        cursor += dur
        entry.end_ms = cursor
    return cursor


def to_manifest(
    script: Script, entries: list[Entry], fps: int, narration_path: str = ""
) -> dict:
    """レンダラに依存しない、動画1本の完全な記述。

    このパイプラインの核は「音声の実尺からタイムラインを導く」ことであり、それは
    どのレンダラを使うかとは独立している(docs/407 §2.4)。よって manifest を
    Python 側と描画側の唯一の契約とし、スライドの状態までここに書き出す。
    描画側が focus の解釈をやり直さずに済み、字幕の投入(MVP-2)も同じ物を読める。
    """
    return {
        "digest_date": script.digest_date,
        "title": script.title,
        "fps": fps,
        "total_ms": entries[-1].end_ms if entries else 0,
        "narration": narration_path,
        "entries": [
            {
                "seq": e.seq,
                "section_seq": e.section_seq,
                "text": e.text,
                "start_ms": e.start_ms,
                "end_ms": e.end_ms,
                "audio": e.audio_path,
                "state": asdict(e.state),
            }
            for e in entries
        ],
    }
