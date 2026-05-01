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
  const [currentLevel, setCurrentLevel] = useState(-1);

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
    <div className="space-y-2 p-2" style={{ background: "#000" }}>
      <video
        ref={videoRef}
        controls
        className="w-full aspect-video"
        style={{ background: "#000" }}
      />
      {levels.length > 0 && (
        <div className="flex gap-2 flex-wrap px-1 pb-1">
          <span className="text-sm" style={{ color: "var(--gh-muted)" }}>画質:</span>
          <button
            onClick={() => selectLevel(-1)}
            className="text-xs px-2 py-0.5 rounded transition-colors"
            style={currentLevel === -1
              ? { background: "var(--nf-red)", color: "#fff" }
              : { background: "var(--tag-bg)", color: "var(--gh-muted)" }}
          >
            自動
          </button>
          {levels.map((level, i) => (
            <button
              key={i}
              onClick={() => selectLevel(i)}
              className="text-xs px-2 py-0.5 rounded transition-colors"
              style={currentLevel === i
                ? { background: "var(--nf-red)", color: "#fff" }
                : { background: "var(--tag-bg)", color: "var(--gh-muted)" }}
            >
              {level.height ? `${level.height}p` : `${Math.round((level.bitrate ?? 0) / 1000)}kbps`}
            </button>
          ))}
        </div>
      )}
    </div>
  );
}
