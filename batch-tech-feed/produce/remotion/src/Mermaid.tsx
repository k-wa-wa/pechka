import React, { useEffect, useMemo, useState } from "react";
import { continueRender, delayRender } from "remotion";
import mermaid from "mermaid";
import { theme, FONT } from "./theme";

mermaid.initialize({
  startOnLoad: false,
  theme: "base",
  darkMode: true,
  themeVariables: {
    background: theme.bg,
    mainBkg: theme.panel,
    primaryColor: theme.panel,
    primaryTextColor: theme.fg,
    primaryBorderColor: theme.border,
    nodeBorder: theme.border,
    nodeTextColor: theme.fg,
    clusterBkg: "#12171e",
    clusterBorder: theme.border,
    lineColor: theme.accent,
    textColor: theme.muted,
    // 既定だと矢印ラベルの背後に不透明の箱が敷かれ、暗い背景から浮いてしまう。
    edgeLabelBackground: theme.bg,
    fontSize: "26px",
    fontFamily: FONT,
  },
  // wrappingWidth を広げないと、日本語のノード名が語の途中で折り返される。
  flowchart: { padding: 18, nodeSpacing: 60, rankSpacing: 80, wrappingWidth: 320, useMaxWidth: false },
});

/**
 * 同じ図を毎フレーム描き直すと Mermaid のレイアウト計算が支配的になるため、
 * ソース文字列をキーに SVG をキャッシュする。Remotion は複数タブで並列に描画するが、
 * キャッシュはタブごとに効けば十分に元が取れる。
 */
const cache = new Map<string, string>();
let uid = 0;

export const useMermaid = (source: string): string | null => {
  const [svg, setSvg] = useState<string | null>(() => cache.get(source) ?? null);
  const handle = useMemo(
    () => (cache.has(source) ? null : delayRender(`mermaid: ${source.slice(0, 40)}`)),
    [source]
  );

  useEffect(() => {
    if (handle === null) return;
    let alive = true;
    // Mermaid は毎回ユニークな ID を要求する(同じ ID だと内部キャッシュで描画が壊れる)
    mermaid
      .render(`mmd-${uid++}`, source)
      .then((r) => {
        cache.set(source, r.svg);
        if (alive) setSvg(r.svg);
        continueRender(handle);
      })
      .catch((err) => {
        // 図の書き損じでレンダリング全体を落とさない。図の代わりにエラーを出す。
        const message = err instanceof Error ? err.message : String(err);
        const fallback = `<text x="0" y="40" fill="#f85149" font-size="28">mermaid error: ${message}</text>`;
        cache.set(source, `<svg xmlns="http://www.w3.org/2000/svg" width="900" height="80">${fallback}</svg>`);
        if (alive) setSvg(cache.get(source)!);
        continueRender(handle);
      });
    return () => {
      alive = false;
    };
  }, [handle, source]);

  return svg;
};

/**
 * 図を進行度 p (0..1) にあわせて少しずつ現す。ノードが順に浮かび上がり、
 * そのあとを追って矢印が伸びる。
 *
 * これは静止画を並べる方式では描けなかった類の動きで、Remotion に移した主な理由である
 * (docs/407 §2.4)。要素ごとの opacity と stroke-dashoffset を毎フレーム計算するだけなので、
 * SVG 自体を再生成する必要はない。
 */
export const revealCss = (svg: string, p: number): string => {
  // mermaid v11 の構造に合わせている:
  //   ノード  <g class="node default" id="…-flowchart-A-0">
  //   辺      <g class="edgePaths"> の中の <path class="… flowchart-link" id="…-L_A_C_0">
  // 辺を `edgePath` で探すと、実際の class が `edgePaths`(複数形)なので \b が効かず
  // 1件も拾えない。拾えないと辺の規則が丸ごと出力されず、辺だけ最初から見えてしまう。
  const nodeIds = [...svg.matchAll(/<g class="[^"]*\bnode\b[^"]*"[^>]*id="([^"]+)"/g)].map((m) => m[1]);
  const edgeCount = (svg.match(/class="[^"]*\bflowchart-link\b/g) ?? []).length;

  // mermaid は自前の <style> を SVG の内側に埋め込み、その規則は id で始まるため
  // 素の class セレクタより詳細度が高い。!important を付けないと静かに負ける。
  const rules: string[] = [];
  // ノードを先に、少しずつずらして出す。
  nodeIds.forEach((id, i) => {
    const start = nodeIds.length > 1 ? (i / nodeIds.length) * 0.5 : 0;
    const o = clamp01((p - start) / 0.25);
    rules.push(`#${cssEscape(id)} { opacity: ${o.toFixed(3)} !important; }`);
  });
  // 矢印はノードが出そろってから伸ばす。行き先が無い線が先に見えると落ち着かない。
  if (edgeCount > 0) {
    const edgeP = clamp01((p - 0.55) / 0.3);
    // 実長を測らずに済むよう、どの辺より長い dasharray を敷いてオフセットで隠す。
    rules.push(
      `.edgePaths path { stroke-dasharray: 3000 !important; stroke-dashoffset: ${(3000 * (1 - edgeP)).toFixed(0)} !important; }`,
      `.edgeLabels { opacity: ${clamp01((p - 0.72) / 0.2).toFixed(3)} !important; }`,
      `.edgePaths marker path, .arrowMarkerPath { opacity: ${clamp01((p - 0.85) / 0.15).toFixed(3)} !important; }`
    );
  }
  return rules.join("\n");
};

const clamp01 = (v: number) => Math.max(0, Math.min(1, v));
// mermaid の id には `-` 以外の記号が混ざりうるため、CSS セレクタとして安全に escape する。
const cssEscape = (id: string) => id.replace(/([^\w-])/g, "\\$1");
