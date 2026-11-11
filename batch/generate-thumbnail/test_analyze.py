import unittest
from pathlib import Path
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

    def test_reconnect_options_precede_input_for_http(self):
        url = "http://nginx:9000/bucket/video.mkv"
        with patch("subprocess.run") as mock_run:
            mock_run.return_value = MagicMock(stdout="10\n")
            analyze.get_duration_seconds(url)
            probe = mock_run.call_args[0][0]
            analyze.extract_frame_at(url, 1, Path("/tmp/out.jpg"))
            ffmpeg = mock_run.call_args[0][0]
        for args in (probe, ffmpeg):
            idx = args.index(url)
            for opt in ("-reconnect", "-reconnect_streamed", "-reconnect_delay_max"):
                self.assertLess(args.index(opt), idx)
        self.assertLess(ffmpeg.index("-reconnect"), ffmpeg.index("-ss"))

    def test_no_reconnect_options_for_local_path(self):
        with patch("subprocess.run") as mock_run:
            mock_run.return_value = MagicMock(stdout="10\n")
            analyze.get_duration_seconds("/tmp/v.mkv")
            self.assertNotIn("-reconnect", mock_run.call_args[0][0])

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

    def test_resolve_thumbnail_key(self):
        s3_keys = [
            "thumbnails/sid1/thumb_01.jpg",
            "thumbnails/sid1/thumb_02.jpg",
        ]
        key = analyze.resolve_thumbnail_key(s3_keys)
        # S3完全キー形式のまま保持され、thumbnails/ プレフィックスが除去されないことを検証
        self.assertEqual(key, "thumbnails/sid1/thumb_01.jpg")
        self.assertTrue(key.startswith("thumbnails/"))

        with self.assertRaises(ValueError):
            analyze.resolve_thumbnail_key([])

    def test_register_assets(self):
        mock_conn = MagicMock()
        mock_cur = MagicMock()
        mock_conn.cursor.return_value.__enter__.return_value = mock_cur
        s3_key = "thumbnails/sid1/thumb_01.jpg"
        analyze.register_assets(mock_conn, "c-uuid", [s3_key], s3_key)
        self.assertEqual(mock_cur.execute.call_count, 2)
        # 1回目は assets テーブルへの INSERT
        first_call_args = mock_cur.execute.call_args_list[0]
        self.assertIn("INSERT INTO assets", first_call_args[0][0])
        self.assertEqual(first_call_args[0][1], ("c-uuid", s3_key))
        # 2回目は contents.thumbnail_key の UPDATE
        second_call_args = mock_cur.execute.call_args_list[1]
        self.assertIn("UPDATE contents SET thumbnail_key", second_call_args[0][0])
        self.assertEqual(second_call_args[0][1], (s3_key, "c-uuid"))
        mock_conn.commit.assert_called_once()

if __name__ == "__main__":
    unittest.main()
