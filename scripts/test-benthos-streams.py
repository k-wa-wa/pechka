#!/usr/bin/env python3
"""
k8s/base/infra/benthos/streams/{contents,subtitles}.yaml に定義された実際の
Bloblang 式（mapping / unarchive / document_map / filter_map / switch 分岐）を
そのまま抽出して本物の Benthos バイナリで実行検証する。
"""

from __future__ import annotations

import json
import os
import shutil
import subprocess
import tempfile
from pathlib import Path
import yaml

ROOT = Path(__file__).resolve().parent.parent
STREAMS_DIR = ROOT / "k8s" / "base" / "infra" / "benthos" / "streams"
MIGRATIONS_DIR = ROOT / "k8s" / "base" / "infra" / "postgres" / "migrations"


class TempPostgres:
    def __init__(self, tmpdir: Path) -> None:
        self.tmpdir = tmpdir
        self.data_dir = tmpdir / "pgdata"
        self.sock_dir = tmpdir / "sock"
        self.log_file = tmpdir / "postgres.log"
        self.sock_dir.mkdir(parents=True, exist_ok=True)

    def __enter__(self) -> "TempPostgres":
        subprocess.run(
            [
                "initdb",
                "-D",
                str(self.data_dir),
                "-U",
                "postgres",
                "-A",
                "trust",
                "--encoding=UTF8",
                "--locale=C",
            ],
            check=True,
            capture_output=True,
            text=True,
        )
        subprocess.run(
            [
                "pg_ctl",
                "-D",
                str(self.data_dir),
                "-l",
                str(self.log_file),
                "-o",
                f"-k {self.sock_dir} -h ''",
                "start",
            ],
            check=True,
            capture_output=True,
            text=True,
        )
        return self

    def __exit__(self, exc_type, exc, tb) -> None:
        subprocess.run(
            ["pg_ctl", "-D", str(self.data_dir), "-m", "immediate", "stop"],
            check=False,
            capture_output=True,
            text=True,
        )

    def create_db(self, db_name: str) -> None:
        subprocess.run(
            ["createdb", "-h", str(self.sock_dir), "-U", "postgres", db_name],
            check=True,
            capture_output=True,
            text=True,
        )

    def psql_file(self, db_name: str, path: Path) -> None:
        proc = subprocess.run(
            [
                "psql",
                "-h",
                str(self.sock_dir),
                "-U",
                "postgres",
                "-d",
                db_name,
                "-v",
                "ON_ERROR_STOP=1",
                "-f",
                str(path),
            ],
            capture_output=True,
            text=True,
            check=False,
        )
        if proc.returncode != 0:
            raise AssertionError(f"psql failed for {path}:\nSTDOUT:\n{proc.stdout}\nSTDERR:\n{proc.stderr}")

    def psql_query(self, db_name: str, sql: str) -> str:
        proc = subprocess.run(
            [
                "psql",
                "-h",
                str(self.sock_dir),
                "-U",
                "postgres",
                "-d",
                db_name,
                "-v",
                "ON_ERROR_STOP=1",
                "-t",
                "-A",
                "-c",
                sql,
            ],
            capture_output=True,
            text=True,
            check=False,
        )
        if proc.returncode != 0:
            raise AssertionError(f"psql query failed:\n{sql}\nSTDERR:\n{proc.stderr}")
        return proc.stdout.strip()


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


def query_subtitles_with_postgres(sql_query: str) -> list[dict]:
    """
    subtitles.yaml の実 SQL 全文（CASE WHEN ... json_agg / FROM / JOIN / WHERE）を
    実際の PostgreSQL 16 インスタンスでそのまま実行し、抽出行とキュー本文・順序を検証する。
    """
    raw_select = sql_query.strip().rstrip(";")
    with tempfile.TemporaryDirectory() as tmp:
        with TempPostgres(Path(tmp)) as pg:
            db = "subtitles_stream_test"
            pg.create_db(db)
            for mig in sorted(MIGRATIONS_DIR.glob("*.up.sql")):
                pg.psql_file(db, mig)

            pg.psql_query(
                db,
                """
                INSERT INTO contents (id, short_id, content_type, channel_id, title, status, deleted_at)
                VALUES
                    ('11111111-1111-1111-1111-111111111111', 's-active', 'video', 'bluray', 'Active', 'ready', NULL),
                    ('22222222-2222-2222-2222-222222222222', 's-deleted', 'video', 'bluray', 'Deleted', 'ready', NOW() - INTERVAL '1 hour');

                INSERT INTO subtitle_tracks (id, content_id, language, status, updated_at)
                VALUES
                    ('aaaa1111-1111-1111-1111-111111111111', '11111111-1111-1111-1111-111111111111', 'ja', 'published', NOW() - INTERVAL '1 hour'),
                    ('aaaa2222-2222-2222-2222-222222222222', '11111111-1111-1111-1111-111111111111', 'en', 'draft', NOW() - INTERVAL '1 hour'),
                    ('aaaa3333-3333-3333-3333-333333333333', '11111111-1111-1111-1111-111111111111', 'fr', 'draft', NOW() - INTERVAL '1 second'),
                    ('bbbb1111-1111-1111-1111-111111111111', '22222222-2222-2222-2222-222222222222', 'ja', 'published', NOW() - INTERVAL '1 hour'),
                    ('bbbb2222-2222-2222-2222-222222222222', '22222222-2222-2222-2222-222222222222', 'en', 'draft', NOW() - INTERVAL '1 hour');

                INSERT INTO subtitle_cues (id, track_id, seq, start_ms, end_ms, text, original_text)
                VALUES
                    ('00000000-0000-0000-0000-000000000002', 'aaaa1111-1111-1111-1111-111111111111', 2, 2000, 3000, 'third', 'third'),
                    ('00000000-0000-0000-0000-000000000001', 'aaaa1111-1111-1111-1111-111111111111', 1, 1000, 1500, 'second', 'second'),
                    ('00000000-0000-0000-0000-000000000003', 'aaaa1111-1111-1111-1111-111111111111', 0, 0, 900, 'first', 'first'),
                    ('00000000-0000-0000-0000-000000000004', 'aaaa2222-2222-2222-2222-222222222222', 0, 0, 1000, 'old-draft-hidden', 'old-draft-hidden'),
                    ('00000000-0000-0000-0000-000000000005', 'aaaa3333-3333-3333-3333-333333333333', 0, 0, 1000, 'recent-draft-must-clear', 'recent-draft-must-clear'),
                    ('00000000-0000-0000-0000-000000000006', 'bbbb1111-1111-1111-1111-111111111111', 0, 0, 1000, 'deleted-pub-must-clear', 'deleted-pub-must-clear'),
                    ('00000000-0000-0000-0000-000000000007', 'bbbb2222-2222-2222-2222-222222222222', 0, 0, 1000, 'deleted-draft-must-clear', 'deleted-draft-must-clear');
                """,
            )

            wrapper_sql = (
                "SELECT COALESCE("
                "json_agg(json_build_object("
                "'short_id', q.short_id, "
                "'language', q.language, "
                "'status', q.status, "
                "'deleted_at', q.deleted_at, "
                "'cues', q.cues::text"
                ")), '[]'::json) "
                f"FROM ({raw_select}) q;"
            )
            raw_json = pg.psql_query(db, wrapper_sql)
            return json.loads(raw_json)


def test_subtitles_stream() -> None:
    cfg = yaml.safe_load((STREAMS_DIR / "subtitles.yaml").read_text(encoding="utf-8"))
    sql_query = cfg["pipeline"]["processors"][0]["sql_raw"]["query"]
    extracted_rows = query_subtitles_with_postgres(sql_query)
    extracted_ids = {f"{r['short_id']}:{r['language']}" for r in extracted_rows}
    assert extracted_ids == {
        "s-active:ja",
        "s-active:fr",
        "s-deleted:ja",
        "s-deleted:en",
    }, f"unexpected extracted subtitle rows from SQL WHERE: {extracted_ids}"

    stream_processors = cfg["pipeline"]["processors"][1:]
    cases = cfg["output"]["fallback"][0]["retry"]["output"]["switch"]["cases"]
    upsert_case = cases[0]
    upsert_check = upsert_case["check"]
    upsert_mongo = upsert_case["output"]["mongodb"]
    upsert_op = upsert_mongo["operation"]
    upsert_doc_map = upsert_mongo["document_map"]

    delete_case = cases[1]
    delete_check = delete_case["check"]
    delete_mongo = delete_case["output"]["mongodb"]
    delete_op = delete_mongo["operation"]
    delete_filter_map = delete_mongo["filter_map"]

    assert upsert_op == "update-one" and upsert_mongo.get("upsert") is True, (
        f"case 0 must be update-one with upsert=true, got: {upsert_mongo}"
    )
    assert delete_op == "delete-one", f"case 1 must be delete-one, got: {delete_mongo}"

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
                                {"mapping": f'root = {{"op": {json.dumps(upsert_op)}, "id": this."$set"._id, "doc": this}}'},
                            ],
                        },
                        {
                            "check": delete_check,
                            "processors": [
                                {"mapping": delete_filter_map},
                                {"mapping": f'root = {{"op": {json.dumps(delete_op)}, "filter": this}}'},
                            ],
                        },
                    ]
                },
            ]
        },
        "output": {"stdout": {}},
    }

    outputs = run_benthos_pipeline(
        sub_pipeline,
        [json.dumps(extracted_rows), json.dumps([])],
    )
    assert len(outputs) == 4, f"expected 4 routed subtitle messages, got {len(outputs)}"

    by_id = {}
    for out in outputs:
        doc_id = out["id"] if "id" in out else out["filter"]["_id"]
        by_id[doc_id] = out

    assert by_id["s-active:ja"]["op"] == "update-one"
    assert by_id["s-active:ja"]["doc"]["$set"]["cues"] == [
        {"start_ms": 0, "end_ms": 900, "text": "first"},
        {"start_ms": 1000, "end_ms": 1500, "text": "second"},
        {"start_ms": 2000, "end_ms": 3000, "text": "third"},
    ]
    assert by_id["s-active:fr"] == {"op": "delete-one", "filter": {"_id": "s-active:fr"}}
    assert by_id["s-deleted:ja"] == {"op": "delete-one", "filter": {"_id": "s-deleted:ja"}}
    assert by_id["s-deleted:en"] == {"op": "delete-one", "filter": {"_id": "s-deleted:en"}}


def main() -> int:
    for cmd in ("benthos", "initdb", "pg_ctl", "createdb", "psql"):
        if shutil.which(cmd) is None:
            raise RuntimeError(f"{cmd} is required in PATH (run inside k8s/lint.sh nix-shell)")
    if os.geteuid() == 0:
        raise RuntimeError("PostgreSQL initdb cannot run as root")
    test_contents_stream()
    test_subtitles_stream()
    print("OK: Benthos contents.yaml and subtitles.yaml Bloblang mappings & switch routing verified.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
