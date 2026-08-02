"""ナレーション音声の合成。

VOICEVOX互換のHTTP API(AivisSpeech Engine / VOICEVOX ENGINE)を叩く。両者はAPI互換の
ため、エンジンの差し替えは --engine-url の変更だけで済む(docs/407 §2.3)。

AivisSpeech Engine は公式に「単一ユーザー利用想定で、多数のリクエストを高速に捌く
API サーバ用途には最適化されていない」と明言している。よってここでは並列化せず、
文単位で逐次に叩く。
"""

from __future__ import annotations

import os
from pathlib import Path

import requests

from . import media, phonetics
from .timeline import Entry

# 文と文のあいだに入れる無音。これがないと読み上げが詰まって聞き取りづらい。
DEFAULT_PAD_MS = 250
# TTSエンジン側の合成は数秒かかりうるが、無応答で無限に待たされるのは避ける。
REQUEST_TIMEOUT_SEC = 120

# デフォルト設定値
DEFAULT_SPEED_SCALE = 1.25
DEFAULT_INTONATION_SCALE = 1.15

# --engine mock 用。日本語の読み上げ速度をおおよそ 7 文字/秒とみなした概算で、
# 尺の妥当性を目視確認できる程度の精度があれば十分である。
MOCK_MS_PER_CHAR = 145
MOCK_BASE_MS = 400


class Synthesizer:
    def synthesize(self, text: str, dst: str) -> None:
        raise NotImplementedError


class VoicevoxSynthesizer(Synthesizer):
    """VOICEVOX互換エンジン(AivisSpeech Engine を含む)のクライアント。"""

    def __init__(
        self,
        base_url: str,
        speaker: int,
        speed: float = DEFAULT_SPEED_SCALE,
        intonation: float = DEFAULT_INTONATION_SCALE,
    ) -> None:
        self.base_url = base_url.rstrip("/")
        self.speaker = speaker
        self.speed = speed
        self.intonation = intonation
        self._session = requests.Session()

    def synthesize(self, text: str, dst: str) -> None:
        # 技術用語や英略称をVOICEVOX読み上げ用に正規化する。
        tts_text = phonetics.normalize_for_tts(text)

        query = self._session.post(
            f"{self.base_url}/audio_query",
            params={"text": tts_text, "speaker": self.speaker},
            timeout=REQUEST_TIMEOUT_SEC,
        )
        query.raise_for_status()
        params = query.json()
        params["speedScale"] = self.speed
        params["intonationScale"] = self.intonation
        # 間はこちら側で一律に付けるため(media.normalize_audio)、エンジン側の前後無音は落とす。
        params["prePhonemeLength"] = 0.0
        params["postPhonemeLength"] = 0.0

        audio = self._session.post(
            f"{self.base_url}/synthesis",
            params={"speaker": self.speaker},
            json=params,
            timeout=REQUEST_TIMEOUT_SEC,
        )
        audio.raise_for_status()
        Path(dst).write_bytes(audio.content)


class MockSynthesizer(Synthesizer):
    """文字数から尺を概算した無音を返す。

    TTSエンジンを立てずに収集〜合成〜レンダリングの配線を検証するためのもの。
    実尺の測定経路は本番と同じ(ffprobe)なので、タイミングの組み立てはここでも検証できる。
    """

    def synthesize(self, text: str, dst: str) -> None:
        media.silence(dst, MOCK_BASE_MS + len(text) * MOCK_MS_PER_CHAR)


def build_synthesizer(
    engine: str,
    engine_url: str,
    speaker: int,
    speed: float = DEFAULT_SPEED_SCALE,
    intonation: float = DEFAULT_INTONATION_SCALE,
) -> Synthesizer:
    if engine == "mock":
        return MockSynthesizer()
    if engine == "voicevox":
        return VoicevoxSynthesizer(engine_url, speaker, speed, intonation)
    raise ValueError(f"unknown engine: {engine!r} (expected 'voicevox' or 'mock')")


def run(
    entries: list[Entry],
    out_dir: str,
    synthesizer: Synthesizer,
    pad_ms: int = DEFAULT_PAD_MS,
) -> list[int]:
    """各文を合成し、共通フォーマットに正規化して実尺(ms)のリストを返す。"""
    audio_dir = Path(out_dir) / "audio"
    audio_dir.mkdir(parents=True, exist_ok=True)

    durations: list[int] = []
    for entry in entries:
        raw = audio_dir / f"{entry.seq:04d}.raw.wav"
        final = audio_dir / f"{entry.seq:04d}.wav"

        synthesizer.synthesize(entry.text, str(raw))
        media.normalize_audio(str(raw), str(final), pad_ms)
        os.remove(raw)

        duration = media.probe_duration_ms(str(final))
        entry.audio_path = str(final)
        durations.append(duration)
        print(f"  [{entry.seq:04d}] {duration / 1000:6.2f}s  {entry.text[:40]}")

    return durations
