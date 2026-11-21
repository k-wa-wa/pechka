import unittest
from unittest.mock import MagicMock, patch
import generate


class FakeSubtitleCursor:
    def __init__(self, db):
        self.db = db
        self._result = None

    def __enter__(self):
        return self

    def __exit__(self, exc_type, exc, tb):
        return False

    def execute(self, sql, params=None):
        norm = " ".join(sql.split())
        if norm.startswith("SELECT deleted_at FROM contents WHERE id = %s"):
            c = self.db["contents"].get(params[0])
            self._result = None if c is None else (c["deleted_at"],)
        elif norm.startswith("SELECT 1 FROM contents WHERE id = %s AND deleted_at IS NULL FOR UPDATE"):
            c = self.db["contents"].get(params[0])
            self._result = (1,) if c is not None and c["deleted_at"] is None else None
        elif norm.startswith("SELECT id, status, manually_edited FROM subtitle_tracks WHERE content_id = %s AND language = %s"):
            t = self.db["tracks"].get((params[0], params[1]))
            self._result = None if t is None else (t["id"], t["status"], t["manually_edited"])
        elif norm.startswith("SELECT COUNT(*) FROM subtitle_cues WHERE track_id = %s AND text != original_text"):
            track_id = params[0]
            cnt = sum(1 for cu in self.db["cues"] if cu["track_id"] == track_id and cu["text"] != cu["original_text"])
            self._result = (cnt,)
        elif norm.startswith("DELETE FROM subtitle_tracks WHERE content_id = %s AND language = %s"):
            t = self.db["tracks"].pop((params[0], params[1]), None)
            if t is not None:
                self.db["cues"] = [cu for cu in self.db["cues"] if cu["track_id"] != t["id"]]
            self._result = None
        elif norm.startswith("INSERT INTO subtitle_tracks"):
            content_id, language, model = params
            track_id = f"track-{len(self.db['tracks']) + 1}-new"
            self.db["tracks"][(content_id, language)] = {
                "id": track_id,
                "status": "draft",
                "manually_edited": False,
                "model": model,
            }
            self._result = (track_id,)
        else:
            raise AssertionError(f"Unexpected SQL: {norm}")

    def fetchone(self):
        return self._result


class FakeSubtitleConn:
    def __init__(self, db):
        self.db = db
        self.committed = False
        self.rolled_back = False

    def cursor(self):
        return FakeSubtitleCursor(self.db)

    def commit(self):
        self.committed = True

    def rollback(self):
        self.rolled_back = True


def make_fake_db(*, status="draft", manually_edited=False, cues=None):
    return {
        "contents": {"c1": {"deleted_at": None}},
        "tracks": {
            ("c1", "ja"): {
                "id": "track-1",
                "status": status,
                "manually_edited": manually_edited,
                "model": "large-v3",
            }
        },
        "cues": list(cues) if cues is not None else [
            {"id": "cu-1", "track_id": "track-1", "text": "Hello", "original_text": "Hello", "start_ms": 0, "end_ms": 1000},
        ],
    }


class TestGenerate(unittest.TestCase):
    def test_flag_hallucinations(self):
        segments = [
            {"start_ms": 0, "end_ms": 1000, "text": "Hello"},
            {"start_ms": 1000, "end_ms": 2000, "text": "Repeated"},
            {"start_ms": 2000, "end_ms": 3000, "text": "Repeated"},
            {"start_ms": 3000, "end_ms": 4000, "text": "Repeated"},
            {"start_ms": 4000, "end_ms": 5000, "text": "World"},
        ]
        result = generate.flag_hallucinations(segments)
        self.assertFalse(result[0]["flagged"])
        self.assertTrue(result[1]["flagged"])
        self.assertTrue(result[2]["flagged"])
        self.assertTrue(result[3]["flagged"])
        self.assertFalse(result[4]["flagged"])

    def test_find_unsafe_overwrite_reason(self):
        mock_conn = MagicMock()
        mock_cur = MagicMock()
        mock_conn.cursor.return_value.__enter__.return_value = mock_cur

        # Case 1: content missing
        mock_cur.fetchone.side_effect = [None]
        reason = generate.find_unsafe_overwrite_reason(mock_conn, "missing", "ja")
        self.assertIn("does not exist", reason)

        # Case 2: content soft-deleted
        mock_cur.fetchone.side_effect = [("2026-01-01T00:00:00Z",)]
        reason = generate.find_unsafe_overwrite_reason(mock_conn, "deleted", "ja")
        self.assertIn("is deleted", reason)

        # Case 3: active content, no existing track
        mock_cur.fetchone.side_effect = [(None,), None]
        reason = generate.find_unsafe_overwrite_reason(mock_conn, "c1", "ja")
        self.assertIsNone(reason)

        # Case 4: existing track is published
        mock_cur.fetchone.side_effect = [(None,), ("track-1", "published", False)]
        reason = generate.find_unsafe_overwrite_reason(mock_conn, "c1", "ja")
        self.assertIn("published", reason)

        # Case 5: existing track marked manually_edited (insert/delete/timing-only edit)
        mock_cur.fetchone.side_effect = [(None,), ("track-1", "draft", True)]
        reason = generate.find_unsafe_overwrite_reason(mock_conn, "c1", "ja")
        self.assertIn("manually edited", reason)

        # Case 6: existing track has text != original_text cues
        mock_cur.fetchone.side_effect = [(None,), ("track-1", "draft", False), (3,)]
        reason = generate.find_unsafe_overwrite_reason(mock_conn, "c1", "ja")
        self.assertIn("3 manually edited cue(s)", reason)

    def test_register_track_deleted_parent_rolls_back(self):
        mock_conn = MagicMock()
        mock_cur = MagicMock()
        mock_conn.cursor.return_value.__enter__.return_value = mock_cur
        mock_cur.fetchone.return_value = None

        segments = [{"start_ms": 0, "end_ms": 1000, "text": "Hello", "flagged": False}]
        with self.assertRaises(RuntimeError):
            generate.register_track(mock_conn, "deleted-c1", "ja", segments)

        self.assertEqual(mock_cur.execute.call_count, 1)
        self.assertIn("deleted_at IS NULL FOR UPDATE", mock_cur.execute.call_args[0][0])
        mock_conn.rollback.assert_called_once()
        mock_conn.commit.assert_not_called()

    def test_register_track_protects_manual_edits_performed_during_transcription(self):
        segments = [{"start_ms": 0, "end_ms": 1000, "text": "Regenerated", "flagged": False}]

        # Case A: Track published during transcription
        db_pub = make_fake_db(status="published")
        conn_pub = FakeSubtitleConn(db_pub)
        with self.assertRaisesRegex(RuntimeError, "published"):
            generate.register_track(conn_pub, "c1", "ja", segments)
        self.assertTrue(conn_pub.rolled_back)
        self.assertFalse(conn_pub.committed)
        self.assertIn(("c1", "ja"), db_pub["tracks"])
        self.assertEqual(len(db_pub["cues"]), 1)

        # Case B: Cue manually inserted during transcription (text == original_text, manually_edited=True)
        db_insert = make_fake_db(
            manually_edited=True,
            cues=[
                {"id": "cu-1", "track_id": "track-1", "text": "Hello", "original_text": "Hello", "start_ms": 0, "end_ms": 1000},
                {"id": "cu-2", "track_id": "track-1", "text": "Inserted", "original_text": "Inserted", "start_ms": 1000, "end_ms": 2000},
            ],
        )
        conn_insert = FakeSubtitleConn(db_insert)
        with self.assertRaisesRegex(RuntimeError, "manually edited"):
            generate.register_track(conn_insert, "c1", "ja", segments)
        self.assertTrue(conn_insert.rolled_back)
        self.assertEqual(len(db_insert["cues"]), 2)

        # Case C: Cue timing-only update during transcription (text == original_text, manually_edited=True)
        db_timing = make_fake_db(
            manually_edited=True,
            cues=[
                {"id": "cu-1", "track_id": "track-1", "text": "Hello", "original_text": "Hello", "start_ms": 250, "end_ms": 1500},
            ],
        )
        conn_timing = FakeSubtitleConn(db_timing)
        with self.assertRaisesRegex(RuntimeError, "manually edited"):
            generate.register_track(conn_timing, "c1", "ja", segments)
        self.assertTrue(conn_timing.rolled_back)
        self.assertEqual(db_timing["cues"][0]["start_ms"], 250)

        # Case D: Cue deleted during transcription (remaining cues unmodified, manually_edited=True)
        db_delete = make_fake_db(manually_edited=True, cues=[])
        conn_delete = FakeSubtitleConn(db_delete)
        with self.assertRaisesRegex(RuntimeError, "manually edited"):
            generate.register_track(conn_delete, "c1", "ja", segments)
        self.assertTrue(conn_delete.rolled_back)
        self.assertEqual(db_delete["tracks"][("c1", "ja")]["id"], "track-1")

        # Case E: Legacy text != original_text even if manually_edited=False
        db_legacy = make_fake_db(
            manually_edited=False,
            cues=[
                {"id": "cu-1", "track_id": "track-1", "text": "Fixed", "original_text": "Hello", "start_ms": 0, "end_ms": 1000},
            ],
        )
        conn_legacy = FakeSubtitleConn(db_legacy)
        with self.assertRaisesRegex(RuntimeError, "1 manually edited cue"):
            generate.register_track(conn_legacy, "c1", "ja", segments)
        self.assertTrue(conn_legacy.rolled_back)
        self.assertEqual(db_legacy["cues"][0]["text"], "Fixed")

    def test_register_track_success(self):
        mock_conn = MagicMock()
        mock_cur = MagicMock()
        mock_conn.cursor.return_value.__enter__.return_value = mock_cur
        mock_cur.fetchone.side_effect = [(1,), None, ("track-uuid",)]

        segments = [{"start_ms": 0, "end_ms": 1000, "text": "Hello", "flagged": False}]
        with patch("generate.psycopg2") as mock_pg:
            generate.register_track(mock_conn, "active-c1", "ja", segments)
            mock_pg.extras.execute_values.assert_called_once()

        mock_conn.commit.assert_called_once()
        mock_conn.rollback.assert_not_called()

    def test_register_track_replaces_unedited_draft_success(self):
        db = make_fake_db(status="draft", manually_edited=False)
        conn = FakeSubtitleConn(db)
        segments = [{"start_ms": 0, "end_ms": 1000, "text": "Hello", "flagged": False}]
        with patch("generate.psycopg2") as mock_pg:
            generate.register_track(conn, "c1", "ja", segments)
            mock_pg.extras.execute_values.assert_called_once()

        self.assertTrue(conn.committed)
        self.assertFalse(conn.rolled_back)
        self.assertNotEqual(db["tracks"][("c1", "ja")]["id"], "track-1")

    @patch("generate.subprocess.run")
    def test_extract_audio_http(self, mock_run):
        generate.extract_audio("http://nginx:9000/bucket/video.mkv", "out.wav")
        mock_run.assert_called_once()
        args = mock_run.call_args[0][0]
        self.assertIn("-reconnect", args)
        self.assertIn("-reconnect_streamed", args)
        # 再接続オプションは入力オプションなので、-i より前に無いと効かない
        input_index = args.index("-i")
        for opt in ("-reconnect", "-reconnect_streamed", "-reconnect_delay_max"):
            self.assertLess(args.index(opt), input_index, f"{opt} must precede -i")

    @patch("generate.subprocess.run")
    def test_extract_audio_local(self, mock_run):
        generate.extract_audio("/path/to/video.mkv", "out.wav")
        mock_run.assert_called_once()
        args = mock_run.call_args[0][0]
        self.assertNotIn("-reconnect", args)

    @patch("socket.getaddrinfo")
    def test_resolve_postgres_dsn_multi_host(self, mock_getaddrinfo):
        mock_getaddrinfo.return_value = [
            (2, 1, 6, "", ("10.20.1.41", 5432)),
            (2, 1, 6, "", ("10.20.1.42", 5432)),
            (2, 1, 6, "", ("10.20.1.43", 5432)),
        ]
        dsn = "postgres://usr:pwd@postgres.external-service.svc.cluster.local:5432/db?target_session_attrs=read-write"
        resolved = generate.resolve_postgres_dsn(dsn)
        expected = "postgres://usr:pwd@10.20.1.41:5432,10.20.1.42:5432,10.20.1.43:5432/db?target_session_attrs=read-write"
        self.assertEqual(resolved, expected)

    @patch("socket.getaddrinfo")
    def test_resolve_postgres_dsn_single_host(self, mock_getaddrinfo):
        mock_getaddrinfo.return_value = [
            (2, 1, 6, "", ("127.0.0.1", 5432)),
        ]
        dsn = "postgres://usr:pwd@localhost:5432/db"
        resolved = generate.resolve_postgres_dsn(dsn)
        self.assertEqual(resolved, dsn)

    def test_resolve_postgres_dsn_already_multi_host(self):
        dsn = "postgres://usr:pwd@host1:5432,host2:5432/db"
        resolved = generate.resolve_postgres_dsn(dsn)
        self.assertEqual(resolved, dsn)


if __name__ == "__main__":
    unittest.main()
