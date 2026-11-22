#!/usr/bin/env python3
"""
k8s/base/infra/postgres/migrations/001〜011 を実際の PostgreSQL 16 インスタンスへ適用し、
冪等再実行、010 の片側 NULL 補完・updated_at 保持・トリガー復元、
011 の新規適用 / 旧版(0a4a427)からの更新 / 再実行時の未編集 draft 保持 / down→up を検証する。
"""

from __future__ import annotations

import os
import re
import shutil
import subprocess
import tempfile
import time
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
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

    def psql_query(self, db_name: str, sql: str) -> list[str]:
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
        return [line for line in proc.stdout.splitlines() if line != ""]


def migration_files(max_num: int | None = None) -> list[Path]:
    files = sorted(MIGRATIONS_DIR.glob("*.up.sql"))
    if max_num is not None:
        files = [f for f in files if int(f.name.split("_", 1)[0]) <= max_num]
    return files


def test_010_partial_null_and_trigger_restoration(pg: TempPostgres) -> None:
    db = "test_010"
    pg.create_db(db)
    for f in migration_files(9):
        pg.psql_file(db, f)

    pg.psql_query(
        db,
        """
        INSERT INTO contents (id, short_id, content_type, channel_id, title, status, published_at, created_at, updated_at)
        VALUES
            ('11111111-1111-1111-1111-111111111111', 's-null-created', 'video', 'bluray', 'Only Created Null', 'ready', NULL, NULL, '2024-01-02 03:04:05+00'),
            ('22222222-2222-2222-2222-222222222222', 's-null-updated', 'video', 'bluray', 'Only Updated Null', 'ready', NULL, '2024-01-01 00:00:00+00', NULL),
            ('33333333-3333-3333-3333-333333333333', 's-null-both', 'video', 'bluray', 'Both Null', 'ready', '2024-02-01 00:00:00+00', NULL, NULL);
        """,
    )

    pg.psql_file(db, MIGRATIONS_DIR / "010_require_content_timestamps.up.sql")

    rows = pg.psql_query(
        db,
        """
        SELECT short_id,
               to_char(created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD HH24:MI:SS'),
               to_char(updated_at AT TIME ZONE 'UTC', 'YYYY-MM-DD HH24:MI:SS')
        FROM contents
        ORDER BY short_id;
        """,
    )
    assert rows == [
        "s-null-both|2024-02-01 00:00:00|2024-02-01 00:00:00",
        "s-null-created|2024-01-02 03:04:05|2024-01-02 03:04:05",
        "s-null-updated|2024-01-01 00:00:00|2024-01-01 00:00:00",
    ], f"unexpected 010 backfill result: {rows}"

    nullable = pg.psql_query(
        db,
        """
        SELECT column_name, is_nullable
        FROM information_schema.columns
        WHERE table_name = 'contents' AND column_name IN ('created_at', 'updated_at')
        ORDER BY column_name;
        """,
    )
    assert nullable == ["created_at|NO", "updated_at|NO"], f"expected NOT NULL constraints: {nullable}"

    # 補完後にトリガーが再び有効になり、通常更新で updated_at が進むことを検証
    pg.psql_query(
        db,
        "UPDATE contents SET title = 'After 010' WHERE short_id = 's-null-created';",
    )
    after_update = pg.psql_query(
        db,
        "SELECT updated_at > '2024-01-02 03:04:05+00'::timestamptz FROM contents WHERE short_id = 's-null-created';",
    )
    assert after_update == ["t"], f"contents_updated_at trigger did not fire after 010: {after_update}"


def test_011_fresh_and_upgrade_from_old_011_and_rerun(pg: TempPostgres) -> None:
    # 1. 新規環境への適用 + 再実行時の新規未編集 draft 保持 + down -> up
    db_fresh = "test_011_fresh"
    pg.create_db(db_fresh)
    for f in migration_files(10):
        pg.psql_file(db_fresh, f)

    pg.psql_query(
        db_fresh,
        """
        INSERT INTO contents (id, short_id, content_type, channel_id, title, status)
        VALUES ('11111111-1111-1111-1111-111111111111', 's1', 'video', 'bluray', 'Video 1', 'ready');
        INSERT INTO subtitle_tracks (id, content_id, language, status, created_at, updated_at)
        VALUES
            ('aaaa1111-1111-1111-1111-111111111111', '11111111-1111-1111-1111-111111111111', 'ja', 'draft', '2024-03-01 00:00:00+00', '2024-03-01 00:00:00+00'),
            ('aaaa2222-2222-2222-2222-222222222222', '11111111-1111-1111-1111-111111111111', 'en', 'published', '2024-03-01 00:00:00+00', '2024-03-01 00:00:00+00');
        INSERT INTO subtitle_cues (track_id, seq, start_ms, end_ms, text, original_text)
        VALUES ('aaaa1111-1111-1111-1111-111111111111', 1, 500, 1500, 'same text', 'same text');
        """,
    )

    pg.psql_file(db_fresh, MIGRATIONS_DIR / "011_add_subtitle_manually_edited.up.sql")
    rows = pg.psql_query(
        db_fresh,
        """
        SELECT language, manually_edited, to_char(updated_at AT TIME ZONE 'UTC', 'YYYY-MM-DD HH24:MI:SS')
        FROM subtitle_tracks
        ORDER BY language;
        """,
    )
    assert rows == [
        "en|t|2024-03-01 00:00:00",
        "ja|t|2024-03-01 00:00:00",
    ], f"fresh 011 must protect all existing tracks without bumping updated_at: {rows}"

    # 移行後に新規作成した未編集 draft が、全マイグレーション再実行後も FALSE のまま保たれること
    pg.psql_query(
        db_fresh,
        """
        INSERT INTO subtitle_tracks (id, content_id, language, status)
        VALUES ('aaaa3333-3333-3333-3333-333333333333', '11111111-1111-1111-1111-111111111111', 'fr', 'draft');
        """,
    )
    for f in migration_files():
        pg.psql_file(db_fresh, f)

    fr_state = pg.psql_query(
        db_fresh,
        "SELECT manually_edited FROM subtitle_tracks WHERE id = 'aaaa3333-3333-3333-3333-333333333333';",
    )
    assert fr_state == ["f"], f"new unedited draft must stay manually_edited=FALSE on migration rerun: {fr_state}"

    # down -> up で再適用可能であること
    pg.psql_file(db_fresh, MIGRATIONS_DIR / "011_add_subtitle_manually_edited.down.sql")
    pg.psql_file(db_fresh, MIGRATIONS_DIR / "011_add_subtitle_manually_edited.up.sql")
    all_protected = pg.psql_query(
        db_fresh,
        "SELECT COUNT(*) FROM subtitle_tracks WHERE manually_edited = FALSE;",
    )
    assert all_protected == ["0"], f"expected all tracks protected after down->up: {all_protected}"

    # 2. 旧版 011 (0a4a427: 本文差分のみ TRUE、時刻のみ手動編集は FALSE) からの更新検証
    db_legacy = "test_011_from_legacy"
    pg.create_db(db_legacy)
    for f in migration_files(10):
        pg.psql_file(db_legacy, f)

    pg.psql_query(
        db_legacy,
        """
        INSERT INTO contents (id, short_id, content_type, channel_id, title, status)
        VALUES ('11111111-1111-1111-1111-111111111111', 's1', 'video', 'bluray', 'Video 1', 'ready');
        INSERT INTO subtitle_tracks (id, content_id, language, status, created_at, updated_at)
        VALUES
            ('bbbb1111-1111-1111-1111-111111111111', '11111111-1111-1111-1111-111111111111', 'ja', 'draft', '2024-03-01 00:00:00+00', '2024-03-01 00:00:00+00'),
            ('bbbb2222-2222-2222-2222-222222222222', '11111111-1111-1111-1111-111111111111', 'en', 'draft', '2024-03-01 00:00:00+00', '2024-03-01 00:00:00+00');
        INSERT INTO subtitle_cues (track_id, seq, start_ms, end_ms, text, original_text)
        VALUES
            ('bbbb1111-1111-1111-1111-111111111111', 1, 200, 1200, 'same text', 'same text'),
            ('bbbb2222-2222-2222-2222-222222222222', 1, 200, 1200, 'edited text', 'original text');
        -- 旧版 011 (0a4a427) の状態を再現
        ALTER TABLE subtitle_tracks ADD COLUMN IF NOT EXISTS manually_edited BOOLEAN NOT NULL DEFAULT FALSE;
        UPDATE subtitle_tracks st
        SET manually_edited = TRUE
        WHERE manually_edited = FALSE
          AND EXISTS (
              SELECT 1 FROM subtitle_cues cu
              WHERE cu.track_id = st.id AND cu.text <> cu.original_text
          );
        """,
    )

    before_upgrade = pg.psql_query(
        db_legacy,
        """
        SELECT language, manually_edited, to_char(updated_at AT TIME ZONE 'UTC', 'YYYY-MM-DD HH24:MI:SS')
        FROM subtitle_tracks
        ORDER BY language;
        """,
    )
    assert len(before_upgrade) == 2
    assert before_upgrade[0].startswith("en|t|") and before_upgrade[1] == "ja|f|2024-03-01 00:00:00", (
        f"unexpected legacy 011 setup: {before_upgrade}"
    )
    en_updated_before = before_upgrade[0].split("|")[2]

    # 改訂版 011 を適用すると、時刻のみ編集済みの ja も TRUE に補完され、双方の updated_at も維持される
    pg.psql_file(db_legacy, MIGRATIONS_DIR / "011_add_subtitle_manually_edited.up.sql")
    after_upgrade = pg.psql_query(
        db_legacy,
        """
        SELECT language, manually_edited, to_char(updated_at AT TIME ZONE 'UTC', 'YYYY-MM-DD HH24:MI:SS')
        FROM subtitle_tracks
        ORDER BY language;
        """,
    )
    assert after_upgrade == [
        f"en|t|{en_updated_before}",
        "ja|t|2024-03-01 00:00:00",
    ], f"upgraded 011 must protect timing-only edited draft (ja): {after_upgrade}"

    # 次のマイグレーション再実行では新規の未編集 draft が FALSE のまま保たれること
    pg.psql_query(
        db_legacy,
        """
        INSERT INTO subtitle_tracks (id, content_id, language, status)
        VALUES ('bbbb3333-3333-3333-3333-333333333333', '11111111-1111-1111-1111-111111111111', 'de', 'draft');
        """,
    )
    pg.psql_file(db_legacy, MIGRATIONS_DIR / "011_add_subtitle_manually_edited.up.sql")
    de_state = pg.psql_query(
        db_legacy,
        "SELECT manually_edited FROM subtitle_tracks WHERE id = 'bbbb3333-3333-3333-3333-333333333333';",
    )
    assert de_state == ["f"], f"new unedited draft after legacy upgrade must stay manually_edited=FALSE: {de_state}"


def extract_pg_lock_query() -> str:
    content_go = (ROOT / "api" / "internal" / "repository" / "postgres" / "content.go").read_text(encoding="utf-8")
    col_match = re.search(r"const\s+contentColumns\s*=\s*`([^`]+)`", content_go)
    lock_match = re.search(r"lockContentForUpdateQuery\s*=\s*`\s*SELECT\s*`\s*\+\s*contentColumns\s*\+\s*`([^`]+)`", content_go)
    assert col_match is not None and lock_match is not None, "failed to extract lockContentForUpdateQuery from content.go"
    return f"SELECT {col_match.group(1)} {lock_match.group(1)}".strip()


def test_set_archived_two_connection_concurrency(pg: TempPostgres) -> None:
    db = "test_set_archived_conc"
    pg.create_db(db)
    for f in migration_files():
        pg.psql_file(db, f)

    cid = "44444444-4444-4444-4444-444444444444"
    pg.psql_query(
        db,
        f"""
        INSERT INTO contents (id, short_id, content_type, channel_id, title, status)
        VALUES ('{cid}', 'conc-1', 'video', 'bluray', 'Conc Video', 'ready');
        """,
    )

    lock_sql = extract_pg_lock_query().replace("$1", f"'{cid}'")

    # 1. 同時アーカイブ: 接続1がロックを保持してアーカイブした後にコミットし、
    # 接続2の FOR UPDATE が待機後にコミット済み archived_at を返すことを検証
    conn1 = subprocess.Popen(
        ["psql", "-h", str(pg.sock_dir), "-U", "postgres", "-d", db, "-v", "ON_ERROR_STOP=1", "-t", "-A"],
        stdin=subprocess.PIPE,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
    )
    assert conn1.stdin is not None and conn1.stdout is not None
    conn1.stdin.write(
        f"BEGIN;\n"
        f"UPDATE contents SET archived_at = '2025-05-01 10:00:00+00' WHERE id = '{cid}' AND deleted_at IS NULL "
        f"RETURNING to_char(archived_at AT TIME ZONE 'UTC', 'YYYY-MM-DD HH24:MI:SS'), to_char(updated_at AT TIME ZONE 'UTC', 'YYYY-MM-DD HH24:MI:SS');\n"
    )
    conn1.stdin.flush()

    # 接続1の UPDATE が完了して行ロックを保持するまで少し待つ
    for _ in range(50):
        locks = pg.psql_query(db, "SELECT COUNT(*) FROM pg_locks WHERE locktype = 'tuple' OR (locktype = 'transactionid' AND mode = 'ExclusiveLock');")
        if locks and int(locks[0]) >= 1:
            break
        time.sleep(0.02)

    conn2 = subprocess.Popen(
        [
            "psql",
            "-h",
            str(pg.sock_dir),
            "-U",
            "postgres",
            "-d",
            db,
            "-v",
            "ON_ERROR_STOP=1",
            "-t",
            "-A",
            "-c",
            f"BEGIN; WITH locked AS ({lock_sql}) SELECT to_char(archived_at AT TIME ZONE 'UTC', 'YYYY-MM-DD HH24:MI:SS'), to_char(updated_at AT TIME ZONE 'UTC', 'YYYY-MM-DD HH24:MI:SS') FROM locked; COMMIT;",
        ],
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
    )

    # 接続2が接続1の行ロック待ちに入るのを確認してから接続1をコミットする
    time.sleep(0.1)
    conn1_out, conn1_err = conn1.communicate("COMMIT;\n", timeout=5)
    assert conn1.returncode == 0, f"conn1 failed: {conn1_err}"
    conn2_out, conn2_err = conn2.communicate(timeout=5)
    assert conn2.returncode == 0, f"conn2 failed: {conn2_err}"

    conn1_lines = [l for l in conn1_out.splitlines() if "|" in l]
    conn2_lines = [l for l in conn2_out.splitlines() if "|" in l]
    assert len(conn1_lines) == 1 and len(conn2_lines) == 1, f"unexpected outputs: conn1={conn1_out!r} conn2={conn2_out!r}"
    assert conn2_lines[0] == conn1_lines[0], f"conn2 FOR UPDATE must return committed state {conn1_lines[0]}, got {conn2_lines[0]}"

    # 2. 同時削除 + 復元: 接続1が論理削除してコミットした後、接続2の FOR UPDATE が0件になることを検証
    conn1_del = subprocess.Popen(
        ["psql", "-h", str(pg.sock_dir), "-U", "postgres", "-d", db, "-v", "ON_ERROR_STOP=1", "-t", "-A"],
        stdin=subprocess.PIPE,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
    )
    assert conn1_del.stdin is not None
    conn1_del.stdin.write(f"BEGIN;\nUPDATE contents SET deleted_at = CURRENT_TIMESTAMP WHERE id = '{cid}' AND deleted_at IS NULL;\n")
    conn1_del.stdin.flush()
    time.sleep(0.05)

    conn2_del = subprocess.Popen(
        [
            "psql",
            "-h",
            str(pg.sock_dir),
            "-U",
            "postgres",
            "-d",
            db,
            "-v",
            "ON_ERROR_STOP=1",
            "-t",
            "-A",
            "-c",
            f"BEGIN; WITH locked AS ({lock_sql}) SELECT COUNT(*) FROM locked; COMMIT;",
        ],
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
    )
    time.sleep(0.1)
    _, del1_err = conn1_del.communicate("COMMIT;\n", timeout=5)
    assert conn1_del.returncode == 0, f"conn1_del failed: {del1_err}"
    del2_out, del2_err = conn2_del.communicate(timeout=5)
    assert conn2_del.returncode == 0, f"conn2_del failed: {del2_err}"
    del2_counts = [l for l in del2_out.splitlines() if l.isdigit()]
    assert del2_counts == ["0"], f"conn2 FOR UPDATE after concurrent Delete must see 0 rows, got {del2_out!r}"


def main() -> int:
    for cmd in ("initdb", "pg_ctl", "createdb", "psql"):
        if shutil.which(cmd) is None:
            raise RuntimeError(f"{cmd} is required in PATH (run inside k8s/lint.sh nix-shell)")
    if os.geteuid() == 0:
        raise RuntimeError("PostgreSQL initdb cannot run as root")

    with tempfile.TemporaryDirectory() as tmp:
        with TempPostgres(Path(tmp)) as pg:
            test_010_partial_null_and_trigger_restoration(pg)
            test_011_fresh_and_upgrade_from_old_011_and_rerun(pg)
            test_set_archived_two_connection_concurrency(pg)

    print("OK: PostgreSQL migrations 001-011 and 2-connection SetArchived concurrency verified.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
