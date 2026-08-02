"""LLM の呼び出し口。agy CLI を使用する。

認証は GEMINI_API_KEY 等の環境変数に委ねる。
"""

from __future__ import annotations

import json
import os
import shutil
import subprocess

DEFAULT_AGY_MODEL = ""
# 台本1本の生成。数千字の出力を見込む。
TIMEOUT_SEC = 900


class LLMError(RuntimeError):
    pass


class LLM:
    def complete(self, prompt: str) -> str:
        raise NotImplementedError


def _find_agy_binary() -> str:
    path = shutil.which("agy")
    if path:
        return path
    for candidate in [
        os.path.expanduser("~/.local/bin/agy"),
        "/root/.local/bin/agy",
        os.path.expanduser("~/.antigravity/bin/agy"),
        "/root/.antigravity/bin/agy",
    ]:
        if os.path.isfile(candidate) and os.access(candidate, os.X_OK):
            return candidate
    return ""


class AgyCLI(LLM):
    """`agy -p` を叩く。認証は GEMINI_API_KEY 等の環境変数に委ねる。"""

    def __init__(self, model: str = DEFAULT_AGY_MODEL) -> None:
        self.model = model

    def complete(self, prompt: str) -> str:
        binary = _find_agy_binary()
        if not binary:
            raise LLMError(
                "agy CLI not found on PATH or ~/.local/bin/agy."
            )
        cmd = [binary, "-p", prompt, "--output-format", "json"]
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


def build(model: str = "") -> LLM:
    return AgyCLI(model or DEFAULT_AGY_MODEL)


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
