"""LLM の呼び出し口。プロバイダを差し替えられるようにしてある。

本番は `agy` CLI。認証は GEMINI_API_KEY 等の環境変数に委ねる。

ローカルの Ollama も残してある。外に出したくない情報源を扱う場合や、`agy` が
使えないときのフォールバックとして使う。
"""

from __future__ import annotations

import json
import os
import shutil
import subprocess

import requests

DEFAULT_AGY_MODEL = ""
DEFAULT_OLLAMA_MODEL = "batiai/qwen3.6-27b:iq3"
# 台本1本の生成。数千字の出力を見込む。
TIMEOUT_SEC = 900


class LLMError(RuntimeError):
    pass


class LLM:
    def complete(self, prompt: str) -> str:
        raise NotImplementedError


class AgyCLI(LLM):
    """`agy -p` を叩く。認証は GEMINI_API_KEY 等の環境変数に委ねる。"""

    def __init__(self, model: str = DEFAULT_AGY_MODEL, binary: str = "agy") -> None:
        self.model = model
        self.binary = binary

    def complete(self, prompt: str) -> str:
        if shutil.which(self.binary) is None:
            raise LLMError(
                f"{self.binary!r} not found on PATH. "
                "Install the agy CLI, or use --llm ollama."
            )
        cmd = [self.binary, "-p", prompt, "--output-format", "json"]
        if self.model:
            cmd.extend(["--model", self.model])

        proc = subprocess.run(
            cmd,
            capture_output=True, text=True, timeout=TIMEOUT_SEC,
        )
        if proc.returncode != 0:
            raise LLMError(f"agy exited {proc.returncode}: {proc.stderr[-1500:]}")

        try:
            envelope = json.loads(proc.stdout)
        except json.JSONDecodeError as e:
            raise LLMError(f"agy returned non-JSON output: {e}\n{proc.stdout[:500]}") from e

        if isinstance(envelope, dict):
            if envelope.get("is_error"):
                raise LLMError(f"agy reported an error: {envelope.get('result', '')[:500]}")
            if "result" in envelope:
                return envelope["result"]
        return proc.stdout


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
    if provider in ("agy", "claude"):
        return AgyCLI(model or DEFAULT_AGY_MODEL)
    if provider == "ollama":
        url = ollama_url or os.environ.get("OLLAMA_URL", "http://lm-server:11434")
        return Ollama(url, model or DEFAULT_OLLAMA_MODEL)
    raise ValueError(f"unknown llm provider: {provider!r} (expected 'agy' or 'ollama')")


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
