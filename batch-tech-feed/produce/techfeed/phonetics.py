"""技術用語・略称・記号等の読み上げ用テキスト正規化モジュール。

VOICEVOX / AivisSpeech 等の形態素解析器(OpenJtalk)が誤読しやすい技術用語や
英単語、バージョン表記などを自然な発音のカタカナ/ひらがなに置換する。
"""

from __future__ import annotations

import re

# 正規表現による単語境界置換のための辞書。
# 長い単語や具体的な表記を優先して置換する。
TECH_TERMS: list[tuple[str, str]] = [
    # フレームワーク / ライブラリ / プログラミング言語
    (r"\bKubernetes\b", "クーバネティス"),
    (r"\bk8s\b", "クーバネティス"),
    (r"\bPostgreSQL\b", "ポスグレエスケイエル"),
    (r"\bPostgres\b", "ポスグレ"),
    (r"\bTypeScript\b", "タイプスクリプト"),
    (r"\bJavaScript\b", "ジャバスクリプト"),
    (r"\bNext\.js\b", "ネクストジェイエス"),
    (r"\bNode\.js\b", "ノードジェイエス"),
    (r"\bReact\b", "リアクト"),
    (r"\bPython\b", "パイソン"),
    (r"\bGolang\b", "ゴーラング"),
    (r"\bDocker\b", "ドッカー"),
    (r"\bRust\b", "ラスト"),
    # サービス / プラットフォーム
    (r"\bGitHub\b", "ギットハブ"),
    (r"\bGitLab\b", "ギットラボ"),
    (r"\bGit\b", "ギット"),
    (r"\bAWS\b", "エーダブリューエス"),
    (r"\bGCP\b", "ジーシーピー"),
    (r"\bAzure\b", "アジュール"),
    # 技術概念 / 略語
    (r"\bCI/CD\b", "シーアイシーディー"),
    (r"\bAPI\b", "エイピーアイ"),
    (r"\bAPIs\b", "エイピーアイズ"),
    (r"\bSDK\b", "エスディーケー"),
    (r"\bSDKs\b", "エスディーケーズ"),
    (r"\bRSS\b", "アールエスエス"),
    (r"\bPR\b", "ピーアール"),
    (r"\bPRs\b", "ピーアールズ"),
    (r"\bIssue\b", "イシュー"),
    (r"\bIssues\b", "イシューズ"),
    (r"\bGA\b", "ジーエー"),
    (r"\bUI\b", "ユーアイ"),
    (r"\bUX\b", "ユーエックス"),
    (r"\bURL\b", "ユーアールエル"),
    (r"\bURLs\b", "ユーアールエルズ"),
    (r"\bHTTP\b", "エイチティーティーピー"),
    (r"\bHTTPS\b", "エイチティーティーピーエス"),
    (r"\bLLM\b", "エルエルエム"),
    (r"\bLLMs\b", "エルエルエムズ"),
    (r"\bTTS\b", "ティーティーエス"),
    (r"\bCPU\b", "シーピーユー"),
    (r"\bGPU\b", "ジーピーユー"),
    (r"\bAI\b", "エーアイ"),
    (r"\bOSS\b", "オーエスエス"),
    (r"\bREADME\b", "リードミー"),
    (r"\bDB\b", "ディービー"),
    (r"\bJSON\b", "ジェイソン"),
    (r"\bYAML\b", "ヤムル"),
    (r"\bHTML\b", "エイチティーエムエル"),
    (r"\bCSS\b", "シーエスエス"),
    (r"\bSQL\b", "エスケイエル"),
]

# バージョン番号表記の置換パターン (例: v1.35.0 -> バージョン 1.35.0)
VERSION_PATTERN = re.compile(r"\bv(\d+(\.\d+)*)\b", re.IGNORECASE)


def normalize_for_tts(text: str) -> str:
    """テキストをVOICEVOX読み上げ用に正規化する。

    技術用語の置換、バージョン表記の展開等を行う。
    """
    if not text:
        return text

    # バージョン表記 (v1.35 -> バージョン 1.35)
    result = VERSION_PATTERN.sub(r"バージョン \1", text)

    # 技術用語置換 (大文字小文字を区別して置換)
    for pattern, replacement in TECH_TERMS:
        result = re.sub(pattern, replacement, result, flags=re.IGNORECASE)

    return result
