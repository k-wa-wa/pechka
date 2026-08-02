import React from "react";
import { AbsoluteFill, interpolate, spring, useVideoConfig } from "remotion";
import { SlideState } from "./types";
import { theme, FONT, MONO } from "./theme";
import { Rich } from "./rich";
import { useMermaid, revealCss } from "./Mermaid";

type Props = {
  state: SlideState;
  /** 直前の状態。何が「新しく現れた」かを知るために使う */
  prev?: SlideState;
  /** この文が始まってからの経過フレーム */
  local: number;
  /** この文の長さ(フレーム) */
  duration: number;
  /** セクションが始まってからの経過フレーム。連続的な動きに使う */
  sectionLocal: number;
  /** セクション全体の長さ(フレーム) */
  sectionDuration: number;
};

const PAD = "72px 96px";

export const Slide: React.FC<Props> = ({ state, prev, local, duration, sectionLocal, sectionDuration }) => (
  <AbsoluteFill style={{ background: theme.bg, color: theme.fg, fontFamily: FONT, padding: PAD,
                         display: "flex", flexDirection: "column" }}>
    {state.layout !== "title" && <Header state={state} />}
    <div style={{ flex: 1, display: "flex", flexDirection: "column",
                  justifyContent: "center", minHeight: 0 }}>
      <Body state={state} prev={prev} local={local} duration={duration}
            sectionLocal={sectionLocal} sectionDuration={sectionDuration} />
    </div>
    {state.layout !== "title" && <Footer state={state} />}
  </AbsoluteFill>
);

const Header: React.FC<{ state: SlideState }> = ({ state }) => (
  <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center",
                color: theme.muted, fontSize: 26, letterSpacing: "0.04em",
                paddingBottom: 24, borderBottom: `1px solid ${theme.border}` }}>
    <div>{state.header}</div>
    <div style={{ color: theme.dim, fontVariantNumeric: "tabular-nums" }}>
      {state.section_seq} / {state.section_total}
    </div>
  </div>
);

const Footer: React.FC<{ state: SlideState }> = ({ state }) => (
  <div style={{ marginTop: "auto", paddingTop: 28, borderTop: `1px solid ${theme.border}`,
                color: theme.dim, fontSize: 22, display: "flex", gap: 28,
                visibility: state.sources.length ? "visible" : "hidden" }}>
    <span>出典</span>
    {state.sources.map((s, i) => (
      <span key={i} style={{ color: theme.muted }}>{s}</span>
    ))}
  </div>
);

const Heading: React.FC<{ state: SlideState }> = ({ state }) => (
  <>
    <div style={{ fontSize: 62, fontWeight: 700, lineHeight: 1.3, marginBottom: 12 }}>
      {state.title}
    </div>
    {state.subtitle ? (
      <div style={{ fontSize: 30, color: theme.muted, marginBottom: 44 }}>
        <Rich text={state.subtitle} />
      </div>
    ) : null}
  </>
);

const Caption: React.FC<{ text: string }> = ({ text }) =>
  text ? (
    <div style={{ fontSize: 26, color: theme.muted, textAlign: "center" }}>
      <Rich text={text} />
    </div>
  ) : null;

const Body: React.FC<Props> = ({ state, prev, local, duration, sectionLocal, sectionDuration }) => {
  switch (state.layout) {
    case "title":
      return <TitleBody state={state} local={local} />;
    case "code":
      return <CodeBody state={state} />;
    case "figure":
      return <FigureBody state={state} local={sectionLocal} duration={sectionDuration} />;
    case "diagram":
      return <DiagramBody state={state} local={sectionLocal} duration={sectionDuration} />;
    default:
      return <BulletsBody state={state} prev={prev} local={local} duration={duration}
                          sectionLocal={sectionLocal} sectionDuration={sectionDuration} />;
  }
};

const TitleBody: React.FC<{ state: SlideState; local: number }> = ({ state, local }) => {
  const { fps } = useVideoConfig();
  const rule = spring({ frame: local, fps, config: { damping: 200 }, durationInFrames: 30 });
  return (
    <div style={{ display: "flex", flexDirection: "column", alignItems: "center", textAlign: "center" }}>
      <div style={{ fontSize: 96, fontWeight: 700, lineHeight: 1.25 }}>{state.title}</div>
      {state.subtitle ? (
        <div style={{ marginTop: 40, fontSize: 40, color: theme.muted }}>{state.subtitle}</div>
      ) : null}
      <div style={{ width: 160 * rule, height: 5, background: theme.accent,
                    marginTop: 56, borderRadius: 3 }} />
    </div>
  );
};

const BulletsBody: React.FC<Props> = ({ state, prev, local, sectionLocal, sectionDuration }) => {
  const items = (
    <ul style={{ listStyle: "none", display: "flex", flexDirection: "column",
                 gap: state.image ? 24 : 28, margin: 0, padding: 0 }}>
      {state.items.map((text, i) => (
        <Item key={i} index={i} text={text} state={state} prev={prev}
              local={local} compact={Boolean(state.image)} />
      ))}
    </ul>
  );

  if (!state.image) {
    return <div><Heading state={state} />{items}</div>;
  }
  return (
    <div>
      <Heading state={state} />
      <div style={{ display: "grid", gridTemplateColumns: "1.05fr 0.95fr", gap: 64,
                    alignItems: "center", minHeight: 0 }}>
        {items}
        <Shot src={state.image} local={sectionLocal} duration={sectionDuration} maxHeight={620} />
      </div>
      <Caption text={state.caption} />
    </div>
  );
};

const Item: React.FC<{
  index: number; text: string; state: SlideState; prev?: SlideState;
  local: number; compact: boolean;
}> = ({ index, text, state, prev, local, compact }) => {
  const { fps } = useVideoConfig();
  const shown = index < state.revealed;
  const focused = state.highlight === index;
  // 直前の状態に無くて今ある項目が「新しく現れた」もの。
  const entering = shown && index >= (prev?.revealed ?? 0);

  const enter = entering
    ? spring({ frame: local, fps, config: { damping: 200, mass: 0.6 }, durationInFrames: 22 })
    : 1;
  // ハイライトの移動もフェードで繋ぐ。瞬間的に飛ぶと目が置いていかれる。
  const wasFocused = prev?.highlight === index;
  const focus = focused
    ? interpolate(local, [0, 12], [wasFocused ? 1 : 0, 1], { extrapolateRight: "clamp" })
    : interpolate(local, [0, 12], [wasFocused ? 1 : 0, 0], { extrapolateRight: "clamp" });

  return (
    <li style={{
      display: "flex", alignItems: "flex-start", gap: compact ? 20 : 26,
      fontSize: compact ? 34 : 40, lineHeight: 1.5,
      color: focused ? "#fff" : shown ? theme.fg : theme.muted,
      padding: compact ? "14px 22px" : "18px 26px",
      borderRadius: 12,
      borderLeft: `5px solid ${focus > 0 ? theme.accent : "transparent"}`,
      background: `rgba(88, 166, 255, ${(0.12 * focus).toFixed(3)})`,
      visibility: shown ? "visible" : "hidden",
      opacity: enter,
      transform: `translateY(${((1 - enter) * 26).toFixed(2)}px)`,
    }}>
      <span style={{ color: focus > 0 ? theme.accent : theme.dim,
                     fontSize: compact ? 28 : 34, lineHeight: 1.75,
                     fontVariantNumeric: "tabular-nums" }}>
        {String(index + 1).padStart(2, "0")}
      </span>
      <span><Rich text={text} /></span>
    </li>
  );
};

const CodeBody: React.FC<{ state: SlideState }> = ({ state }) => (
  <div>
    <Heading state={state} />
    {state.language ? (
      <div style={{ alignSelf: "flex-start", display: "inline-block", fontSize: 22,
                    color: theme.accent, background: theme.accentSoft,
                    padding: "6px 16px", borderRadius: 999, marginBottom: 20 }}>
        {state.language}
      </div>
    ) : null}
    <pre style={{ background: theme.panel, border: `1px solid ${theme.border}`, borderRadius: 14,
                  padding: "40px 48px", fontFamily: MONO, fontSize: 32, lineHeight: 1.6,
                  color: theme.fg, margin: 0, whiteSpace: "pre", overflow: "hidden" }}>
      {state.code}
    </pre>
  </div>
);

/**
 * 画像はゆっくり寄り続ける。止め絵のまま長く映すと画面が死んで見えるため。
 * 連続的な動きなので、静止画を並べる方式では表現できなかったものである。
 */
const Shot: React.FC<{ src: string; local: number; duration: number; maxHeight: number }> = ({
  src, local, duration, maxHeight,
}) => {
  const scale = interpolate(local, [0, Math.max(duration, 1)], [1, 1.04], {
    extrapolateRight: "clamp",
  });
  return (
    <div style={{ display: "flex", alignItems: "center", justifyContent: "center",
                  minHeight: 0, overflow: "hidden" }}>
      <img src={src} alt="" style={{
        maxWidth: "100%", maxHeight, objectFit: "contain",
        borderRadius: 14, border: `1px solid ${theme.border}`, background: theme.panel,
        transform: `scale(${scale})`,
      }} />
    </div>
  );
};

const FigureBody: React.FC<{ state: SlideState; local: number; duration: number }> = ({
  state, local, duration,
}) => (
  // 見出しは他レイアウトと同じ左寄せのまま、画像だけを中央に置く。
  // レイアウトが変わるたびに見出しの位置が動くと、視線が毎回さまよう。
  <div style={{ display: "flex", flexDirection: "column", minHeight: 0, gap: 24 }}>
    <Heading state={state} />
    <Shot src={state.image} local={local} duration={duration} maxHeight={600} />
    <Caption text={state.caption} />
  </div>
);

const DiagramBody: React.FC<{ state: SlideState; local: number; duration: number }> = ({
  state, local, duration,
}) => {
  const svg = useMermaid(state.diagram);
  // 図はセクションの尺いっぱいを使って現す。ナレーションが説明する順に絵が育つ。
  const p = interpolate(local, [0, Math.max(duration * 0.75, 1)], [0, 1], {
    extrapolateRight: "clamp",
  });

  return (
    <div>
      <Heading state={state} />
      <div style={{ display: "flex", alignItems: "center", justifyContent: "center",
                    minHeight: 0, flex: 1, marginBottom: 20 }}>
        {svg ? (
          <>
            <style>{revealCss(svg, p)}</style>
            <div
              style={{ width: "100%", display: "flex", justifyContent: "center" }}
              // mermaid が返す SVG をそのまま描く。中身は台本由来で外部入力ではない。
              dangerouslySetInnerHTML={{ __html: sizeSvg(svg) }}
            />
          </>
        ) : null}
      </div>
      <Caption text={state.caption} />
    </div>
  );
};

/** mermaid が svg にインラインで書く寸法を上書きし、スライド幅いっぱいに使わせる。 */
const sizeSvg = (svg: string) =>
  svg.replace(
    /<svg /,
    '<svg style="max-width:100%;width:100%;height:auto;max-height:560px" '
  );
