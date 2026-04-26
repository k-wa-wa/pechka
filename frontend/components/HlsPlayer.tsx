"use client";

import { useEffect, useRef, useState } from "react";
import Hls, { type Level } from "hls.js";
import { type MongoVariant } from "@/lib/api";
import { hlsUrl } from "@/lib/utils";

interface Props {
  variants: MongoVariant[];
}

export function HlsPlayer({ variants }: Props) {
  const videoRef = useRef<HTMLVideoElement>(null);
  const hlsRef = useRef<Hls | null>(null);
  const [levels, setLevels] = useState<Level[]>([]);
  const [currentLevel, setCurrentLevel] = useState(-1); // -1 = auto

  const masterVariant = variants.find((v) => v.variant_type === "master");

  useEffect(() => {
    const video = videoRef.current;
    if (!video || !masterVariant) return;

    const src = hlsUrl(masterVariant.hls_key);

    if (Hls.isSupported()) {
      const hls = new Hls();
      hlsRef.current = hls;
      hls.loadSource(src);
      hls.attachMedia(video);
      hls.on(Hls.Events.MANIFEST_PARSED, (_, data) => {
        setLevels(data.levels);
      });
      hls.on(Hls.Events.LEVEL_SWITCHED, (_, data) => {
        setCurrentLevel(data.level);
      });
      return () => {
        hls.destroy();
        hlsRef.current = null;
      };
    } else if (video.canPlayType("application/vnd.apple.mpegurl")) {
      video.src = src;
    }
  }, [masterVariant]);

  function selectLevel(index: number) {
    if (!hlsRef.current) return;
    hlsRef.current.currentLevel = index;
    setCurrentLevel(index);
  }

  return (
    <div className="space-y-2">
      <video
        ref={videoRef}
        controls
        className="w-full rounded-lg bg-black aspect-video"
      />
      {levels.length > 0 && (
        <div className="flex gap-2 flex-wrap">
          <span className="text-sm text-gray-500">画質:</span>
          <button
            onClick={() => selectLevel(-1)}
            className={`text-sm px-2 py-0.5 rounded ${currentLevel === -1 ? "bg-blue-600 text-white" : "bg-gray-200 text-gray-800"}`}
          >
            自動
          </button>
          {levels.map((level, i) => (
            <button
              key={i}
              onClick={() => selectLevel(i)}
              className={`text-sm px-2 py-0.5 rounded ${currentLevel === i ? "bg-blue-600 text-white" : "bg-gray-200 text-gray-800"}`}
            >
              {level.height ? `${level.height}p` : `${Math.round((level.bitrate ?? 0) / 1000)}kbps`}
            </button>
          ))}
        </div>
      )}
    </div>
  );
}
