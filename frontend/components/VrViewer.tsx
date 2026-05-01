"use client";

import { useEffect, useRef } from "react";
import { type MongoVariant } from "@/lib/api";
import { hlsUrl } from "@/lib/utils";

interface Props {
  variants: MongoVariant[];
}

export function VrViewer({ variants }: Props) {
  const containerRef = useRef<HTMLDivElement>(null);

  const masterVariant = variants.find((v) => v.variant_type === "master");
  const src = masterVariant ? hlsUrl(masterVariant.hls_key) : "";

  useEffect(() => {
    if (!containerRef.current || !src) return;

    import("aframe").then(() => {
      if (!containerRef.current) return;
      containerRef.current.innerHTML = `
        <a-scene embedded style="width:100%;height:500px">
          <a-videosphere src="#vr-video" rotation="0 -90 0"></a-videosphere>
          <a-assets>
            <video id="vr-video" src="${src}" crossorigin="anonymous" autoplay loop></video>
          </a-assets>
          <a-camera></a-camera>
        </a-scene>
      `;
    });
  }, [src]);

  return (
    <div>
      <div ref={containerRef} className="w-full overflow-hidden" style={{ height: 500, background: "#000" }} />
      <p className="text-sm px-3 py-1.5" style={{ color: "var(--gh-muted)" }}>
        マウスドラッグまたはデバイスを動かして視点を変更できます
      </p>
    </div>
  );
}
