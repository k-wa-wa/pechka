#!/usr/bin/env python3
"""
k8s/base/infra/benthos/streams/{contents,subtitles}.yaml に定義された実際の
Bloblang 式（mapping / unarchive / document_map / filter_map / switch 分岐）を
そのまま抽出して本物の Benthos バイナリで実行検証する。
"""

from __future__ import annotations

import json
import shutil
import subprocess
import tempfile
from pathlib import Path
import yaml

ROOT = Path(__file__).resolve().parent.parent
STREAMS_DIR = ROOT / "k8s" / "base" / "infra" / "benthos" / "streams"


def run_benthos_pipeline(pipeline_cfg: dict, inputs: list[str]) -> list[dict]:
    with tempfile.TemporaryDirectory() as tmpdir:
        cfg_path = Path(tmpdir) / "pipeline.yaml"
        cfg_path.write_text(yaml.safe_dump(pipeline_cfg, sort_keys=False), encoding="utf-8")
        proc = subprocess.run(
            ["benthos", "-c", str(cfg_path)],
            input="\n".join(inputs) + "\n",
            text=True,
            capture_output=True,
            check=False,
        )
        if proc.returncode != 0:
            raise AssertionError(
                f"benthos exited with {proc.returncode}:\nSTDOUT:\n{proc.stdout}\nSTDERR:\n{proc.stderr}"
            )
        outputs = []
        for line in proc.stdout.splitlines():
            line = line.strip()
            if not line:
                continue
            outputs.append(json.loads(line))
        return outputs


def test_contents_stream() -> None:
    cfg = yaml.safe_load((STREAMS_DIR / "contents.yaml").read_text(encoding="utf-8"))
    processors = cfg["pipeline"]["processors"]
    # processors[0] は sql_raw、processors[1:] が mapping + unarchive
    stream_processors = processors[1:]
    mongo_out = cfg["output"]["fallback"][0]["retry"]["output"]["broker"]["outputs"][0]["mongodb"]
    document_map = mongo_out["document_map"]
    filter_map = mongo_out["filter_map"]

    # 1. Elasticsearch 側に入る前段（mapping + unarchive）の検証（配列・単一オブジェクト・空配列）
    es_pipeline = {
        "logger": {"level": "ERROR"},
        "input": {"stdin": {"scanner": {"lines": {}}}},
        "pipeline": {"threads": 1, "processors": stream_processors},
        "output": {"stdout": {}},
    }

    active_row = {
        "id": "11111111-1111-1111-1111-111111111111",
        "short_id": "vid-active",
        "content_type": "video",
        "channel_id": "bluray",
        "channel_name": "Blu-ray",
        "disc_id": "22222222-2222-2222-2222-222222222222",
        "title": "Active Video",
        "description": "Desc",
        "duration_seconds": 120,
        "is_360": False,
        "tags": '["a","b"]',
        "status": "ready",
        "thumbnail_key": "thumbnails/vid-active.jpg",
        "document_key": None,
        "document_mime_type": None,
        "published_at": "2025-01-02T03:04:05.123Z",
        "archived_at": None,
        "created_at": "2025-01-01T00:00:00.456Z",
        "updated_at": "2025-01-02T03:04:05.789Z",
        "deleted_at": None,
        "disc_label": "DISC-1",
        "variants": '[{"variant_type":"master","hls_key":"resources/hls/vid-active/master.m3u8"}]',
        "has_subtitles": True,
    }
    deleted_row = {
        "id": "33333333-3333-3333-3333-333333333333",
        "short_id": "vid-deleted",
        "content_type": "video",
        "channel_id": "tech-feed",
        "channel_name": "Tech Feed",
        "disc_id": None,
        "title": "Deleted Video",
        "description": "",
        "duration_seconds": None,
        "is_360": False,
        "tags": "[]",
        "status": "ready",
        "thumbnail_key": None,
        "document_key": None,
        "document_mime_type": None,
        "published_at": None,
        "archived_at": "2025-02-01T10:00:00.000Z",
        "created_at": "2025-01-10T00:00:00.000Z",
        "updated_at": "2025-02-02T12:34:56.789Z",
        "deleted_at": "2025-02-03T01:02:03.456Z",
        "disc_label": None,
        "variants": "[]",
        "has_subtitles": False,
    }

    es_outputs = run_benthos_pipeline(
        es_pipeline,
        [json.dumps([active_row, deleted_row])],
    )
    assert len(es_outputs) == 2, f"expected 2 unarchived messages, got {len(es_outputs)}"
    assert es_outputs[0]["published_at"] == "2025-01-02T03:04:05.123Z", "ES path must keep ISO-8601 string"
    assert es_outputs[0]["archived_at"] is None
    assert es_outputs[0]["deleted_at"] is None
    assert es_outputs[0]["tags"] == ["a", "b"]
    assert es_outputs[0]["variants"][0]["variant_type"] == "master"
    assert es_outputs[1]["deleted_at"] == "2025-02-03T01:02:03.456Z"

    single_outputs = run_benthos_pipeline(es_pipeline, [json.dumps(active_row)])
    assert len(single_outputs) == 1 and single_outputs[0]["id"] == active_row["id"]

    empty_outputs = run_benthos_pipeline(es_pipeline, [json.dumps([])])
    assert empty_outputs == []

    # 2. MongoDB document_map / filter_map の実実行検証（YAML 内の式をそのまま使用）
    mongo_pipeline = {
        "logger": {"level": "ERROR"},
        "input": {"stdin": {"scanner": {"lines": {}}}},
        "pipeline": {
            "threads": 1,
            "processors": [
                *stream_processors,
                {
                    "branch": {
                        "request_map": "root = this",
                        "processors": [{"mapping": filter_map}],
                        "result_map": "meta filter_json = content().string()",
                    }
                },
                {"mapping": document_map},
                {"mapping": 'root = {"filter": meta("filter_json").parse_json(), "doc": this}'},
            ]
        },
        "output": {"stdout": {}},
    }

    mongo_outputs = run_benthos_pipeline(mongo_pipeline, [json.dumps([active_row, deleted_row])])
    assert len(mongo_outputs) == 2

    m0_filter, m0_set = mongo_outputs[0]["filter"], mongo_outputs[0]["doc"]["$set"]
    assert m0_filter == {"_id": "vid-active"}
    assert m0_set["published_at"] == {"$date": {"$numberLong": "1735787045123"}}
    assert m0_set["created_at"] == {"$date": {"$numberLong": "1735689600456"}}
    assert m0_set["updated_at"] == {"$date": {"$numberLong": "1735787045789"}}
    assert m0_set["archived_at"] is None, "null archived_at must stay null"
    assert m0_set["deleted_at"] is None, "null deleted_at must stay null"

    m1_filter, m1_set = mongo_outputs[1]["filter"], mongo_outputs[1]["doc"]["$set"]
    assert m1_filter == {"_id": "vid-deleted"}
    assert m1_set["published_at"] is None, "null published_at must stay null"
    assert m1_set["archived_at"] == {"$date": {"$numberLong": "1738404000000"}}
    assert m1_set["created_at"] == {"$date": {"$numberLong": "1736467200000"}}
    assert m1_set["updated_at"] == {"$date": {"$numberLong": "1738499696789"}}
    assert m1_set["deleted_at"] == {"$date": {"$numberLong": "1738544523456"}}


def test_subtitles_stream() -> None:
    cfg = yaml.safe_load((STREAMS_DIR / "subtitles.yaml").read_text(encoding="utf-8"))
    sql_query = cfg["pipeline"]["processors"][0]["sql_raw"]["query"]
    assert "c.deleted_at IS NOT NULL" in sql_query, "subtitles sql_raw must always include deleted parents"

    stream_processors = cfg["pipeline"]["processors"][1:]
    cases = cfg["output"]["fallback"][0]["retry"]["output"]["switch"]["cases"]
    upsert_check = cases[0]["check"]
    upsert_doc_map = cases[0]["output"]["mongodb"]["document_map"]
    delete_check = cases[1]["check"]
    delete_filter_map = cases[1]["output"]["mongodb"]["filter_map"]

    sub_pipeline = {
        "logger": {"level": "ERROR"},
        "input": {"stdin": {"scanner": {"lines": {}}}},
        "pipeline": {
            "threads": 1,
            "processors": [
                *stream_processors,
                {
                    "switch": [
                        {
                            "check": upsert_check,
                            "processors": [
                                {"mapping": upsert_doc_map},
                                {"mapping": 'root = {"op": "update-one", "id": this."$set"._id, "doc": this}'},
                            ],
                        },
                        {
                            "check": delete_check,
                            "processors": [
                                {"mapping": delete_filter_map},
                                {"mapping": 'root = {"op": "delete-one", "filter": this}'},
                            ],
                        },
                    ]
                },
            ]
        },
        "output": {"stdout": {}},
    }

    rows = [
        {
            "short_id": "s-pub-active",
            "language": "ja",
            "status": "published",
            "deleted_at": None,
            "cues": '[{"start_ms":0,"end_ms":1000,"text":"hi"}]',
        },
        {
            "short_id": "s-draft-active",
            "language": "ja",
            "status": "draft",
            "deleted_at": None,
            "cues": "[]",
        },
        {
            "short_id": "s-pub-deleted",
            "language": "ja",
            "status": "published",
            "deleted_at": "2025-02-03T01:02:03.456Z",
            "cues": "[]",
        },
        {
            "short_id": "s-draft-deleted",
            "language": "en",
            "status": "draft",
            "deleted_at": "2025-02-03T01:02:03.456Z",
            "cues": "[]",
        },
    ]

    outputs = run_benthos_pipeline(
        sub_pipeline,
        [json.dumps(rows), json.dumps([])],
    )
    assert len(outputs) == 4, f"expected 4 routed subtitle messages, got {len(outputs)}"

    assert outputs[0]["op"] == "update-one"
    assert outputs[0]["id"] == "s-pub-active:ja"
    assert outputs[0]["doc"]["$set"]["cues"] == [{"start_ms": 0, "end_ms": 1000, "text": "hi"}]

    assert outputs[1] == {"op": "delete-one", "filter": {"_id": "s-draft-active:ja"}}
    assert outputs[2] == {"op": "delete-one", "filter": {"_id": "s-pub-deleted:ja"}}
    assert outputs[3] == {"op": "delete-one", "filter": {"_id": "s-draft-deleted:en"}}


def main() -> int:
    if shutil.which("benthos") is None:
        raise RuntimeError("benthos command is required in PATH (run inside k8s/lint.sh nix-shell)")
    test_contents_stream()
    test_subtitles_stream()
    print("OK: Benthos contents.yaml and subtitles.yaml Bloblang mappings & switch routing verified.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
