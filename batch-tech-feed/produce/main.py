#!/usr/bin/env python3
"""技術ダイジェストの動画生成バッチ。

手書きの台本(script.json)から解説動画の mp4 を焼くところまでを担う。情報収集と
LLMによる台本生成(collect / compose)は後続フェーズで追加する(docs/407 §5)。

    python main.py build --script examples/script_sample.json --out /tmp/digest

工程は synthesize(音声) と render(映像) に分かれ、両者の間の契約は manifest.json
である。工程単位の再実行を可能にするため(docs/102 US-5.3)。
"""

from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path

from techfeed import (
    candidates as candidates_mod, compose, narration, publish, renderer,
    script as script_mod, synthesize, timeline,
)

MANIFEST_NAME = "manifest.json"
CANDIDATES_NAME = "candidates.json"
SCRIPT_NAME = "script.json"


def _load_entries(script_path: str, assets_dir: str = "") -> tuple[script_mod.Script, list[timeline.Entry]]:
    scr = script_mod.load(script_path, assets_dir)
    entries = timeline.build(scr)
    print(f"script: {len(scr.sections)} section(s), {scr.line_count} line(s)")
    return scr, entries


def _manifest_path(out_dir: str) -> Path:
    return Path(out_dir) / MANIFEST_NAME


def _save_manifest(
    scr: script_mod.Script,
    entries: list[timeline.Entry],
    out_dir: str,
    fps: int,
    narration_path: str,
) -> dict:
    manifest = timeline.to_manifest(scr, entries, fps, narration_path)
    path = _manifest_path(out_dir)
    path.write_text(json.dumps(manifest, ensure_ascii=False, indent=2), encoding="utf-8")
    print(f"manifest: {path} (total {manifest['total_ms'] / 1000:.1f}s)")
    return manifest


def _load_manifest(out_dir: str) -> dict:
    path = _manifest_path(out_dir)
    if not path.exists():
        raise SystemExit(f"{path} not found; run the 'synthesize' step first")
    return json.loads(path.read_text(encoding="utf-8"))


def cmd_compose(args: argparse.Namespace) -> str:
    # 収集を別 Job に出したので、作業ディレクトリはここで確保する。
    Path(args.out).mkdir(parents=True, exist_ok=True)
    path = args.candidates or str(Path(args.out) / CANDIDATES_NAME)
    print(f"reading candidates from {path}")
    candidates = candidates_mod.load(path)
    print(f"composing a script from {len(candidates)} candidate(s)...")
    data = compose.run(
        candidates, args.llm, digest_date=args.digest_date,
        topics=args.topics, model=args.llm_model, ollama_url=args.ollama_url,
    )
    dst = str(Path(args.out) / SCRIPT_NAME)
    compose.save(data, dst)
    print(f"script: {len(data.get('sections', []))} section(s) -> {dst}")
    return dst


def cmd_synthesize(args: argparse.Namespace) -> dict:
    scr, entries = _load_entries(args.script, args.assets_dir)
    synth = synthesize.build_synthesizer(args.engine, args.engine_url, args.speaker, args.speed)

    print(f"synthesizing with engine={args.engine}...")
    durations = synthesize.run(entries, args.out, synth, pad_ms=args.pad_ms)
    total = timeline.apply_durations(entries, durations)
    print(f"total narration: {total / 1000:.1f}s")

    narration_path = narration.build(entries, args.out)
    return _save_manifest(scr, entries, args.out, args.fps, narration_path)


def cmd_render(args: argparse.Namespace) -> None:
    manifest = _load_manifest(args.out)
    dst = args.output or str(Path(args.out) / "digest.mp4")
    print(f"rendering with remotion ({len(manifest['entries'])} line(s))...")
    renderer.run(manifest, args.out, dst, concurrency=args.concurrency,
                 crf=args.crf, quiet=args.quiet)
    print(f"done: {dst}")


def _publish(args: argparse.Namespace, manifest: dict) -> None:
    mp4 = args.output or str(Path(args.out) / "digest.mp4")
    source_key = args.source_key or f"tech-feed:{manifest.get('digest_date') or 'unknown'}"
    publish.run(manifest, mp4, source_key, description=args.description)


def cmd_publish(args: argparse.Namespace) -> None:
    _publish(args, _load_manifest(args.out))


def cmd_build(args: argparse.Namespace) -> None:
    """synthesize → render を一度に回す。主経路。"""
    manifest = cmd_synthesize(args)
    dst = args.output or str(Path(args.out) / "digest.mp4")
    print("rendering with remotion...")
    renderer.run(manifest, args.out, dst, concurrency=args.concurrency,
                 crf=args.crf, quiet=args.quiet)
    print(f"done: {dst}")


def cmd_all(args: argparse.Namespace) -> None:
    """build + publish。中間成果物が同じディスク上にある必要があるため、
    ワークフローでもこれを1つのコンテナで回す(工程ごとにPodを分けると /work が共有されない)。"""
    cmd_build(args)
    _publish(args, _load_manifest(args.out))


def cmd_produce(args: argparse.Namespace) -> None:
    """compose → build → publish。収集(Go の別 Job)の結果を受け取って回す。"""
    args.script = cmd_compose(args)
    cmd_all(args)


def _add_compose(parser: argparse.ArgumentParser) -> None:
    parser.add_argument(
        "--candidates", default="",
        help="candidates.json written by the collect job (default: <out>/candidates.json)",
    )
    parser.add_argument("--llm", default="claude", choices=["claude", "ollama"])
    parser.add_argument("--llm-model", default="", help="model name for the chosen provider")
    parser.add_argument("--ollama-url", default="", help="base URL when --llm ollama")
    parser.add_argument("--topics", type=int, default=3, help="how many topics to cover")
    parser.add_argument("--digest-date", default="", help="YYYY-MM-DD (default: today)")


def _add_common(parser: argparse.ArgumentParser) -> None:
    parser.add_argument("--script", required=True, help="path to script.json")
    parser.add_argument("--out", required=True, help="working directory for intermediate artifacts")
    parser.add_argument("--fps", type=int, default=30)
    parser.add_argument(
        "--assets-dir", default="",
        help="base directory for image paths in the script (default: the script's own directory)",
    )


def _add_tts(parser: argparse.ArgumentParser) -> None:
    parser.add_argument(
        "--engine",
        default="voicevox",
        choices=["voicevox", "mock"],
        help="'mock' generates silence sized from the text length (no TTS engine needed)",
    )
    parser.add_argument("--engine-url", default="http://aivisspeech:10101")
    parser.add_argument("--speaker", type=int, default=0)
    parser.add_argument("--speed", type=float, default=1.0)
    parser.add_argument("--pad-ms", type=int, default=synthesize.DEFAULT_PAD_MS)


def _add_render(parser: argparse.ArgumentParser) -> None:
    parser.add_argument("--output", default="", help="output mp4 path (default: <out>/digest.mp4)")
    parser.add_argument("--crf", type=int, default=20)
    parser.add_argument(
        "--concurrency", type=int, default=0,
        help="parallel browser tabs for rendering (0 = let remotion decide)",
    )
    parser.add_argument("--quiet", action="store_true", help="suppress remotion progress output")


def _add_publish(parser: argparse.ArgumentParser) -> None:
    # build と共有するため、--output が二重に定義されないようにする。
    if not any(a.dest == "output" for a in parser._actions):
        parser.add_argument("--output", default="", help="mp4 to publish (default: <out>/digest.mp4)")
    parser.add_argument(
        "--source-key", default="",
        help="idempotency key for re-runs (default: tech-feed:<digest_date>)",
    )
    parser.add_argument("--description", default="", help="content description")


def main() -> None:
    parser = argparse.ArgumentParser(description="Tech digest video generator")
    sub = parser.add_subparsers(dest="command", required=True)

    p = sub.add_parser("compose", help="write a script from the collected candidates (LLM)")
    p.add_argument("--out", required=True, help="working directory")
    _add_compose(p)
    p.set_defaults(func=cmd_compose)

    p = sub.add_parser("synthesize", help="synthesize narration audio and fix the timeline")
    _add_common(p)
    _add_tts(p)
    p.set_defaults(func=cmd_synthesize)

    p = sub.add_parser("render", help="render the video from manifest.json (remotion)")
    _add_common(p)
    _add_render(p)
    p.set_defaults(func=cmd_render)

    p = sub.add_parser("publish", help="upload HLS to S3 and register the content in PostgreSQL")
    _add_common(p)
    _add_publish(p)
    p.set_defaults(func=cmd_publish)

    p = sub.add_parser("build", help="synthesize + render")
    _add_common(p)
    _add_tts(p)
    _add_render(p)
    p.set_defaults(func=cmd_build)

    p = sub.add_parser("all", help="synthesize + render + publish")
    _add_common(p)
    _add_tts(p)
    _add_render(p)
    _add_publish(p)
    p.set_defaults(func=cmd_all)

    p = sub.add_parser("produce", help="compose + build + publish (consumes the collect job's output)")
    p.add_argument("--out", required=True, help="working directory")
    p.add_argument("--fps", type=int, default=30)
    p.add_argument("--assets-dir", default="")
    p.add_argument("--script", default="", help=argparse.SUPPRESS)
    _add_compose(p)
    _add_tts(p)
    _add_render(p)
    _add_publish(p)
    p.set_defaults(func=cmd_produce)

    args = parser.parse_args()
    try:
        args.func(args)
    except script_mod.ScriptError as e:
        print(f"invalid script: {e}", file=sys.stderr)
        raise SystemExit(1)
    except renderer.RenderError as e:
        print(f"render failed: {e}", file=sys.stderr)
        raise SystemExit(1)


if __name__ == "__main__":
    main()
