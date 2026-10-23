import unittest
from unittest.mock import MagicMock, patch
import analyze

class TestAnalyze(unittest.TestCase):
    def test_get_duration_seconds_success(self):
        with patch("subprocess.run") as mock_run:
            mock_run.return_value = MagicMock(stdout="123.45\n")
            duration = analyze.get_duration_seconds("dummy.mkv")
            self.assertEqual(duration, 123.45)

    def test_get_duration_seconds_error(self):
        with patch("subprocess.run") as mock_run:
            mock_run.return_value = MagicMock(stdout="invalid\n")
            duration = analyze.get_duration_seconds("dummy.mkv")
            self.assertEqual(duration, 0.0)

    def test_select_thumbnails(self):
        with patch("analyze.brightness_score", side_effect=[5.0, 50.0, 80.0, 20.0]):
            frames = ["f1.jpg", "f2.jpg", "f3.jpg", "f4.jpg"]
            selected = analyze.select_thumbnails(frames, max_count=2)
            # f1 is < 10 (filtered), f3 has 80, f2 has 50
            self.assertEqual(selected, ["f3.jpg", "f2.jpg"])

    def test_upload_thumbnails(self):
        mock_boto = MagicMock()
        mock_client = MagicMock()
        mock_boto.client.return_value = mock_client
        with patch.object(analyze, "boto3", mock_boto):
            keys = analyze.upload_thumbnails(
                thumbnails=["/tmp/t1.jpg"],
                short_id="sid1",
                bucket="test-bucket",
                minio_url="minio:9000",
                access_key="key",
                secret_key="sec",
                use_ssl=False,
            )
            self.assertEqual(keys, ["thumbnails/sid1/thumb_01.jpg"])
            mock_client.upload_file.assert_called_once()

    def test_register_assets(self):
        mock_conn = MagicMock()
        mock_cur = MagicMock()
        mock_conn.cursor.return_value.__enter__.return_value = mock_cur
        analyze.register_assets(mock_conn, "c-uuid", ["thumbnails/sid1/thumb_01.jpg"], "sid1/thumb_01.jpg")
        self.assertEqual(mock_cur.execute.call_count, 2)
        mock_conn.commit.assert_called_once()

if __name__ == "__main__":
    unittest.main()
