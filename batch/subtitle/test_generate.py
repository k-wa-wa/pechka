import unittest
from unittest.mock import MagicMock, patch
import generate

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
        mock_cur.fetchone.side_effect = [(None,), ("track-1", "published")]
        reason = generate.find_unsafe_overwrite_reason(mock_conn, "c1", "ja")
        self.assertIn("published", reason)

        # Case 5: existing track has edited cues
        mock_cur.fetchone.side_effect = [(None,), ("track-1", "draft"), (3,)]
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

    def test_register_track_success(self):
        mock_conn = MagicMock()
        mock_cur = MagicMock()
        mock_conn.cursor.return_value.__enter__.return_value = mock_cur
        mock_cur.fetchone.side_effect = [(1,), ("track-uuid",)]

        segments = [{"start_ms": 0, "end_ms": 1000, "text": "Hello", "flagged": False}]
        with patch("generate.psycopg2") as mock_pg:
            generate.register_track(mock_conn, "active-c1", "ja", segments)
            mock_pg.extras.execute_values.assert_called_once()

        mock_conn.commit.assert_called_once()
        mock_conn.rollback.assert_not_called()

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
