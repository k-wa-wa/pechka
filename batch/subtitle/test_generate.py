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

        # Case 1: no existing track
        mock_cur.fetchone.return_value = None
        reason = generate.find_unsafe_overwrite_reason(mock_conn, "c1", "ja")
        self.assertIsNone(reason)

        # Case 2: existing track is published
        mock_cur.fetchone.return_value = ("track-1", "published")
        reason = generate.find_unsafe_overwrite_reason(mock_conn, "c1", "ja")
        self.assertIn("published", reason)

        # Case 3: existing track has edited cues
        mock_cur.fetchone.side_effect = [("track-1", "draft"), (3,)]
        reason = generate.find_unsafe_overwrite_reason(mock_conn, "c1", "ja")
        self.assertIn("3 manually edited cue(s)", reason)

if __name__ == "__main__":
    unittest.main()
