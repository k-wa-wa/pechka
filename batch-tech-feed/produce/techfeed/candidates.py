"""候補記事の受け取り口。

収集そのものは Go の別 Job（`cmd/collect.go`）が担う。ここは受け取って読むだけである。
両者の契約は candidates.json のスキーマで、Go 側の Candidate 構造体と対になる。
片方のフィールド名を変えたら両方を直すこと。
"""

from __future__ import annotations

import json
from dataclasses import dataclass
from pathlib import Path


@dataclass
class Candidate:
    title: str
    url: str
    publisher: str
    published_at: str
    summary: str = ""
    # Hacker News のスコア等。選別のヒントに使う。
    score: int | None = None


def _parse(raw: list) -> list[Candidate]:
    known = Candidate.__dataclass_fields__.keys()
    # Go 側が将来フィールドを足しても落ちないよう、知らないキーは捨てる。
    return [Candidate(**{k: v for k, v in c.items() if k in known}) for c in raw]


def load(path: str) -> list[Candidate]:
    return _parse(json.loads(Path(path).read_text(encoding="utf-8")))
