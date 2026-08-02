import React from "react";
import { AbsoluteFill, Audio, interpolate, staticFile, useCurrentFrame, useVideoConfig } from "remotion";
import { Manifest } from "./types";
import { theme } from "./theme";
import { Slide } from "./Slide";

/** 絵が切り替わるときのクロスフェードの長さ。長いと間延びし、短いと見落とす。 */
const CROSSFADE_FRAMES = 13;

export const msToFrames = (ms: number, fps: number) => Math.round((ms / 1000) * fps);

export const Digest: React.FC<{ manifest: Manifest }> = ({ manifest }) => {
  const frame = useCurrentFrame();
  const { fps } = useVideoConfig();
  const entries = manifest.entries;

  // いま読み上げている文を探す。文の境界がそのまま絵の境界である。
  let idx = 0;
  while (idx + 1 < entries.length && msToFrames(entries[idx + 1].start_ms, fps) <= frame) {
    idx++;
  }
  const cur = entries[idx];
  const prev = idx > 0 ? entries[idx - 1] : undefined;

  const start = msToFrames(cur.start_ms, fps);
  const local = frame - start;
  const duration = msToFrames(cur.end_ms, fps) - start;

  // 連続的な動き(図が育つ、画像がゆっくり寄る)はセクション全体を使い切る。
  // 文ごとの local を使うと、次の文に移った瞬間に動きが最初へ巻き戻ってしまう。
  const first = entries.find((e) => e.section_seq === cur.section_seq)!;
  const last = [...entries].reverse().find((e) => e.section_seq === cur.section_seq)!;
  const sectionStart = msToFrames(first.start_ms, fps);
  const sectionLocal = frame - sectionStart;
  const sectionDuration = msToFrames(last.end_ms, fps) - sectionStart;

  // 直前の絵と同じなら重ねる意味がないので、クロスフェードは省く。
  const changed = prev ? JSON.stringify(prev.state) !== JSON.stringify(cur.state) : false;
  const fade = changed
    ? interpolate(local, [0, CROSSFADE_FRAMES], [0, 1], { extrapolateRight: "clamp" })
    : 1;

  return (
    <AbsoluteFill style={{ background: theme.bg }}>
      {manifest.narration ? <Audio src={staticFile(manifest.narration)} /> : null}

      {/* 下の絵は不透明のまま置き、上の絵をフェードインさせる。両方を同時にフェード
          すると、変化していない領域まで合成の都合で一瞬暗くなる。 */}
      {changed && prev && fade < 1 ? (
        <Slide
          state={prev.state}
          local={local + (msToFrames(prev.end_ms, fps) - msToFrames(prev.start_ms, fps))}
          duration={msToFrames(prev.end_ms, fps) - msToFrames(prev.start_ms, fps)}
          sectionLocal={sectionLocal}
          sectionDuration={sectionDuration}
        />
      ) : null}

      <AbsoluteFill style={{ opacity: fade }}>
        <Slide state={cur.state} prev={prev?.state} local={local} duration={duration}
               sectionLocal={sectionLocal} sectionDuration={sectionDuration} />
      </AbsoluteFill>
    </AbsoluteFill>
  );
};
