import React from "react";
import { Composition } from "remotion";
import { Digest, msToFrames } from "./Digest";
import { Manifest } from "./types";
import sample from "./sample-manifest.json";

const fallback = sample as unknown as Manifest;

export const RemotionRoot: React.FC = () => (
  <Composition
    id="Digest"
    component={Digest}
    // 実尺は manifest 側が持っている。props が渡ればそちらで上書きする。
    durationInFrames={Math.max(1, msToFrames(fallback.total_ms, fallback.fps))}
    fps={fallback.fps}
    width={1920}
    height={1080}
    defaultProps={{ manifest: fallback }}
    calculateMetadata={({ props }) => ({
      durationInFrames: Math.max(1, msToFrames(props.manifest.total_ms, props.manifest.fps)),
      fps: props.manifest.fps,
    })}
  />
);
