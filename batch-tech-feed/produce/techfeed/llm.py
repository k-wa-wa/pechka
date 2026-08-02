"""LLM の呼び出し口。google-antigravity SDK を使用する。

認証は GEMINI_API_KEY 環境変数に委ねる。
"""

from __future__ import annotations

import asyncio
import json
import os

from google.antigravity import Agent, LocalAgentConfig

DEFAULT_AGY_MODEL = ""
# 台本1本の生成。数千字の出力を見込む。
TIMEOUT_SEC = 900


class LLMError(RuntimeError):
    pass


class LLM:
    def complete(self, prompt: str) -> str:
        raise NotImplementedError


class AgySDK(LLM):
    """google-antigravity SDK の Agent を使用してプロンプトを実行する。"""

    def __init__(self, model: str = DEFAULT_AGY_MODEL) -> None:
        self.model = model

    def complete(self, prompt: str) -> str:
        if not os.environ.get("GEMINI_API_KEY"):
            raise LLMError("GEMINI_API_KEY environment variable is not set")

        async def _chat() -> str:
            config = LocalAgentConfig()
            async with Agent(config) as agent:
                response = await agent.chat(prompt)
                return await response.text()

        try:
            return asyncio.run(_chat())
        except Exception as e:
            raise LLMError(f"google-antigravity agent chat failed: {e}") from e


def build(model: str = "") -> LLM:
    return AgySDK(model or DEFAULT_AGY_MODEL)


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
