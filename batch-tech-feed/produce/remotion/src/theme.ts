export const theme = {
  bg: "#0d1117",
  panel: "#161b22",
  fg: "#e6edf3",
  muted: "#8b949e",
  dim: "#484f58",
  accent: "#58a6ff",
  accentSoft: "rgba(88, 166, 255, 0.12)",
  border: "#30363d",
} as const;

/** コンテナには fonts-noto-cjk を入れる。手元確認用にヒラギノ以降も並べている。 */
export const FONT =
  '"Noto Sans CJK JP", "Noto Sans JP", "Hiragino Sans", "Hiragino Kaku Gothic ProN", "Yu Gothic", sans-serif';

export const MONO =
  '"Noto Sans Mono CJK JP", "SFMono-Regular", "Menlo", monospace';
