import ffmpeg
import numpy as np
from PIL import Image
import os
import re
import logging

logger = logging.getLogger(__name__)

class VideoScorer:
    def __init__(self, ideal_brightness=128):
        self.ideal_brightness = ideal_brightness

    def analyze_frame(self, file_path, timestamp):
        """
        Analyzes a single frame at a specific timestamp for brightness and audio volume.
        Returns a score from 0-200 (100 for brightness, 100 for volume).
        """
        try:
            brightness_score = self._get_brightness_score(file_path, timestamp)
            volume_score = self._get_volume_score(file_path, timestamp)
            total_score = brightness_score + volume_score
            logger.info(f"Timestamp {timestamp}s: Brightness={brightness_score:.2f}, Volume={volume_score:.2f}, Total={total_score:.2f}")
            return total_score
        except Exception as e:
            logger.error(f"Error analyzing frame at {timestamp}s: {e}")
            return 0

    def _get_brightness_score(self, file_path, timestamp):
        try:
            # Extract single frame as raw bytes
            out, _ = (
                ffmpeg
                .input(file_path, ss=timestamp)
                .output('pipe:', vframes=1, format='image2', vcodec='mjpeg', loglevel='error')
                .run(capture_stdout=True)
            )
            
            from io import BytesIO
            img = Image.open(BytesIO(out)).convert('L') # Convert to grayscale (Luminance)
            stat = np.array(img)
            avg_lum = np.mean(stat)
            
            diff = abs(self.ideal_brightness - avg_lum)
            score = 100.0 - (diff / self.ideal_brightness) * 100.0
            return float(max(0.0, score))
        except Exception as e:
            logger.error(f"Brightness calculation failed: {e}")
            return 0

    def _get_volume_score(self, file_path, timestamp):
        try:
            # Extract 3 seconds of audio and use volumedetect
            out, err = (
                ffmpeg
                .input(file_path, ss=timestamp, t=3)
                .output('null', format='null', af='volumedetect', vn=None, sn=None, dn=None)
                .run(capture_stdout=True, capture_stderr=True)
            )
            
            output = err.decode('utf-8')
            
            mean_vol = -91.0
            max_vol = -91.0
            
            mean_match = re.search(r"mean_volume:\s+([-\.\d]+)\s+dB", output)
            max_match = re.search(r"max_volume:\s+([-\.\d]+)\s+dB", output)
            
            if mean_match:
                mean_vol = float(mean_match.group(1))
            if max_match:
                max_vol = float(max_match.group(1))
                
            mean_score = ((mean_vol + 91.0) / 91.0) * 100.0
            max_score = ((max_vol + 91.0) / 91.0) * 100.0
            
            # 70% mean, 30% max weight
            score = (mean_score * 0.7) + (max_score * 0.3)
            return float(max(0.0, min(100.0, score)))
        except Exception as e:
            logger.error(f"Volume calculation failed: {e}")
            return 0
