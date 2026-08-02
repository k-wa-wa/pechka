"""候補記事から台本(script.json)を書かせる。

LLM の出力をそのままレンダリングに渡さない。必ず `script.py` の検証を通し、
落ちたらエラー内容を添えて書き直させる。**バッチが受け取る境界は常に検証済みの
台本である**(docs/407 §2.2)。
"""

from __future__ import annotations

import json
from datetime import date
from pathlib import Path

from . import llm as llm_mod
from . import script as script_mod
from .candidates import Candidate

MAX_ATTEMPTS = 3
# 候補が多すぎるとプロンプトが膨らむだけで選別の質は上がらない。
MAX_CANDIDATES = 60

PROMPT = """\
あなたは技術ダイジェスト動画の構成作家である。以下の候補記事から今日取り上げる\
{topics}件を選び、解説動画の台本を JSON で書け。

# 選別の基準
- 実務に影響するもの、一次情報（公式ブログ・リリースノート）を優先する
- 単なる宣伝、内容の薄いもの、同じ話題の重複は落とす
- 読者はインフラ・バックエンドを触る個人開発者である

# 出力する JSON の形式

{{
  "digest_date": "{digest_date}",
  "title": "今日の技術トピック {topics}選",
  "sections": [ ...セクション... ]
}}

セクションは次の形。1つ目は必ず layout="title" の表紙にし、以降が各トピック。

{{
  "seq": 2,
  "slide": {{
    "layout": "bullets",          // title | bullets | code | diagram のいずれか
    "title": "見出し（20字程度）",
    "subtitle": "補足（30字程度、任意）",
    "items": ["要点1", "要点2", "要点3"]   // bullets のとき必須。3件前後
    // code のとき: "code": "...", "language": "yaml"
    // diagram のとき: "diagram": "flowchart LR\\n  A[\\"箱\\"] --> B[\\"箱\\"]"
  }},
  "narration": [
    {{ "text": "読み上げる1文。", "focus": null }},
    {{ "text": "この文は項目1の話。", "focus": 0 }}
  ],
  "sources": [
    {{ "title": "記事タイトル", "url": "https://...", "publisher": "掲載元" }}
  ]
}}

# 厳守すること
- **narration は1文ずつに分ける。** 音声合成・字幕・スライド遷移の単位がこれである
- `focus` は bullets のときだけ使う。items の index（0始まり）を指し、`null` は\
「スライドを進めず前の状態のまま」を意味する。1つの項目を2文で語るなら同じ focus を2回書く
- `focus` に items の範囲外を書かない
- **sources には候補記事の url をそのまま入れる。URL を創作しない**
- 記事本文を丸ごと引き写さない。見出しと自分の言葉による要約に留める
- 図が説明に効く場合は diagram（Mermaid の flowchart）を使う。ノードのラベルは\
必ず二重引用符で囲む
- title レイアウトの表紙には narration を2文程度、sources は不要
- 全体で90秒〜3分程度の分量にする
- **JSON のみを出力する。前置き・説明・コードフェンスを付けない**

# 候補記事
{candidates}
"""


def _candidates_block(candidates: list[Candidate]) -> str:
    lines = []
    for i, c in enumerate(candidates[:MAX_CANDIDATES], start=1):
        score = f" [{c.score}pt]" if c.score else ""
        summary = c.summary.replace("\n", " ")[:200]
        lines.append(f"{i}. {c.title}{score}\n   url: {c.url}\n   from: {c.publisher}\n   {summary}")
    return "\n".join(lines)


def build_prompt(candidates: list[Candidate], digest_date: str, topics: int) -> str:
    return PROMPT.format(
        topics=topics,
        digest_date=digest_date,
        candidates=_candidates_block(candidates),
    )


def run(
    candidates: list[Candidate],
    provider: str,
    digest_date: str = "",
    topics: int = 3,
    model: str = "",
    ollama_url: str = "",
) -> dict:
    if not candidates:
        raise SystemExit("no candidates to compose from; run the 'collect' step first")

    digest_date = digest_date or date.today().isoformat()
    client = llm_mod.build(provider, model, ollama_url)
    prompt = build_prompt(candidates, digest_date, topics)

    last_error = ""
    for attempt in range(1, MAX_ATTEMPTS + 1):
        ask = prompt if attempt == 1 else (
            f"{prompt}\n\n# 直前の出力は次の理由で不正だった。修正して JSON を出し直せ\n{last_error}"
        )
        print(f"  attempt {attempt}/{MAX_ATTEMPTS} (provider={provider})...")
        raw = client.complete(ask)

        try:
            data = llm_mod.extract_json(raw)
            # ここで弾いておかないと、不正な台本が後段の描画まで流れてしまう。
            script_mod.parse(data)
            return data
        except (llm_mod.LLMError, script_mod.ScriptError, json.JSONDecodeError, ValueError) as e:
            last_error = f"{type(e).__name__}: {e}"
            print(f"    rejected: {last_error}")

    raise SystemExit(f"compose failed after {MAX_ATTEMPTS} attempts. last error: {last_error}")


def save(script: dict, path: str) -> None:
    Path(path).write_text(json.dumps(script, ensure_ascii=False, indent=2), encoding="utf-8")
