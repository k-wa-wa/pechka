"""LLM の呼び出し口。プロバイダを差し替えられるようにしてある。

本番は `claude` CLI（Sonnet 5）。台本の品質がこの機能の価値を直接決めるため、
そこを未知数に賭けない判断である(docs/407 §2.2)。nuage-autopilot に `claude` CLI の
運用実績があることも効いている。

ローカルの Ollama も残してある。外に出したくない情報源を扱う場合や、`claude` が
使えないときのフォールバックとして使う。
"""

from __future__ import annotations

import json
import os
import shutil
import subprocess

import requests

DEFAULT_CLAUDE_MODEL = "sonnet"
DEFAULT_OLLAMA_MODEL = "batiai/qwen3.6-27b:iq3"
# 台本1本の生成。数千字の出力を見込む。
TIMEOUT_SEC = 900


class LLMError(RuntimeError):
    pass


class LLM:
    def complete(self, prompt: str) -> str:
        raise NotImplementedError


class ClaudeCLI(LLM):
    """`claude -p` を叩く。認証は CLI 側の設定(~/.claude 等)に委ねる。"""

    def __init__(self, model: str = DEFAULT_CLAUDE_MODEL, binary: str = "claude") -> None:
        self.model = model
        self.binary = binary

    def complete(self, prompt: str) -> str:
        if shutil.which(self.binary) is None:
            raise LLMError(
                f"{self.binary!r} not found on PATH. "
                "Install the Claude CLI, or use --llm ollama."
            )
        proc = subprocess.run(
            [self.binary, "-p", prompt, "--output-format", "json", "--model", self.model],
            capture_output=True, text=True, timeout=TIMEOUT_SEC,
        )
        if proc.returncode != 0:
            raise LLMError(f"claude exited {proc.returncode}: {proc.stderr[-1500:]}")

        try:
            envelope = json.loads(proc.stdout)
        except json.JSONDecodeError as e:
            raise LLMError(f"claude returned non-JSON output: {e}\n{proc.stdout[:500]}") from e

        if envelope.get("is_error"):
            raise LLMError(f"claude reported an error: {envelope.get('result', '')[:500]}")
        return envelope.get("result", "")


class Ollama(LLM):
    def __init__(self, base_url: str, model: str = DEFAULT_OLLAMA_MODEL) -> None:
        self.base_url = base_url.rstrip("/")
        self.model = model

    def complete(self, prompt: str) -> str:
        res = requests.post(
            f"{self.base_url}/api/generate",
            json={"model": self.model, "prompt": prompt, "stream": False, "format": "json"},
            timeout=TIMEOUT_SEC,
        )
        res.raise_for_status()
        return res.json().get("response", "")


def build(provider: str, model: str = "", ollama_url: str = "") -> LLM:
    if provider == "claude":
        return ClaudeCLI(model or DEFAULT_CLAUDE_MODEL)
    if provider == "ollama":
        url = ollama_url or os.environ.get("OLLAMA_URL", "http://lm-server:11434")
        return Ollama(url, model or DEFAULT_OLLAMA_MODEL)
    raise ValueError(f"unknown llm provider: {provider!r} (expected 'claude' or 'ollama')")


def extract_json(text: str) -> dict:
    """LLM の出力から JSON を取り出す。

    指示しても ```json のフェンスや前置きが付いてくることがあるため、素の
    json.loads に頼らず、最初の { から対応する } までを取り出す。
    """
    text = text.strip()
    if text.startswith("```"):
        # ```json ... ``` を剥がす
        body = text.split("```", 2)
        if len(body) >= 2:
            text = body[1]
            if text.startswith("json"):
                text = text[4:]
            text = text.strip()

    start = text.find("{")
    if start < 0:
        raise LLMError(f"no JSON object found in the response:\n{text[:500]}")

    depth = 0
    in_string = False
    escaped = False
    for i, ch in enumerate(text[start:], start=start):
        if in_string:
            if escaped:
                escaped = False
            elif ch == "\\":
                escaped = True
            elif ch == '"':
                in_string = False
            continue
        if ch == '"':
            in_string = True
        elif ch == "{":
            depth += 1
        elif ch == "}":
            depth -= 1
            if depth == 0:
                return json.loads(text[start : i + 1])
    raise LLMError(f"unterminated JSON object in the response:\n{text[:500]}")
