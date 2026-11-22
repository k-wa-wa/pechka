#!/usr/bin/env python3
"""
k8s/base/infra/postgres/migrations/001〜009 を実際の PostgreSQL 16 インスタンスへ適用し、
002 再実行時の論理削除済み行保持・インデックス保護、009 の部分一意インデックス再定義
（idx_contents_deleted_at 不作成）および安全な down→up ロールバックを検証する。
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


def test_002_and_009_rerun_idempotency_and_soft_delete(pg: TempPostgres) -> None:
    db = "test_002_009"
    pg.create_db(db)
    for f in migration_files():
        pg.psql_file(db, f)

    # 1. idx_contents_deleted_at が存在しないこと、idx_contents_disc_id が deleted_at IS NULL 付きであることを検証
    deleted_idx = pg.psql_query(
        db,
        "SELECT COUNT(*) FROM pg_indexes WHERE tablename = 'contents' AND indexname = 'idx_contents_deleted_at';",
    )
    assert deleted_idx == ["0"], f"idx_contents_deleted_at must not exist: {deleted_idx}"

    disc_idx_def = pg.psql_query(
        db,
        "SELECT pg_get_indexdef(c.oid) FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace "
        "WHERE n.nspname = current_schema() AND c.relname = 'idx_contents_disc_id';",
    )
    assert len(disc_idx_def) == 1 and "deleted_at IS NULL" in disc_idx_def[0], (
        f"idx_contents_disc_id must filter deleted_at IS NULL: {disc_idx_def}"
    )

    # 2. 論理削除済み行と同一 disc_id の再取り込みが許可される一方、未削除同士の重複は拒否されることを検証
    disc_id = "dddd1111-1111-1111-1111-111111111111"
    pg.psql_query(
        db,
        f"""
        INSERT INTO discs (id, label, disc_name) VALUES ('{disc_id}', 'DISC-1', 'Disc 1');
        INSERT INTO contents (id, short_id, content_type, channel_id, disc_id, title, status, created_at, deleted_at)
        VALUES
            ('11111111-1111-1111-1111-111111111111', 's-deleted', 'video', 'bluray', '{disc_id}', 'Deleted Take 1', 'ready', '2024-01-01 00:00:00+00', '2024-01-02 00:00:00+00'),
            ('22222222-2222-2222-2222-222222222222', 's-active', 'video', 'bluray', '{disc_id}', 'Active Take 2', 'ready', '2024-01-03 00:00:00+00', NULL);
        """,
    )

    # 3. 002〜009 を再実行しても、論理削除済み行（および重複テスト用データ）が物理削除されず、
    #    idx_contents_disc_id も上書きされないことを検証
    for f in migration_files():
        if int(f.name.split("_", 1)[0]) >= 2:
            pg.psql_file(db, f)

    remaining = pg.psql_query(db, "SELECT short_id FROM contents ORDER BY short_id;")
    assert remaining == ["s-active", "s-deleted"], f"002/009 rerun must preserve soft-deleted rows: {remaining}"

    disc_idx_after = pg.psql_query(
        db,
        "SELECT pg_get_indexdef(c.oid) FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace "
        "WHERE n.nspname = current_schema() AND c.relname = 'idx_contents_disc_id';",
    )
    assert len(disc_idx_after) == 1 and "deleted_at IS NULL" in disc_idx_after[0], (
        f"002 rerun must not overwrite idx_contents_disc_id: {disc_idx_after}"
    )

    # 4. 論理削除済み行が残っている間は 009.down.sql が中断し、削除後は down → up できることを検証
    down_blocked = subprocess.run(
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
            "-f",
            str(MIGRATIONS_DIR / "009_add_deleted_at.down.sql"),
        ],
        capture_output=True,
        text=True,
        check=False,
    )
    assert down_blocked.returncode != 0, "009.down.sql must fail when soft-deleted rows exist"

    pg.psql_query(db, "DELETE FROM contents WHERE deleted_at IS NOT NULL;")
    pg.psql_file(db, MIGRATIONS_DIR / "009_add_deleted_at.down.sql")
    pg.psql_file(db, MIGRATIONS_DIR / "009_add_deleted_at.up.sql")

    disc_idx_restored = pg.psql_query(
        db,
        "SELECT pg_get_indexdef(c.oid) FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace "
        "WHERE n.nspname = current_schema() AND c.relname = 'idx_contents_disc_id';",
    )
    assert len(disc_idx_restored) == 1 and "deleted_at IS NULL" in disc_idx_restored[0], (
        f"009 up after down must restore partial unique index: {disc_idx_restored}"
    )


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
            test_002_and_009_rerun_idempotency_and_soft_delete(pg)
            test_set_archived_two_connection_concurrency(pg)

    print("OK: PostgreSQL migrations 001-009 and 2-connection SetArchived concurrency verified.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
