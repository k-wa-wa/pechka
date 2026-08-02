import React from "react";
import { theme, MONO } from "./theme";

const codeStyle: React.CSSProperties = {
  fontFamily: MONO,
  fontSize: "0.88em",
  color: theme.accent,
  background: theme.accentSoft,
  padding: "2px 10px",
  borderRadius: 6,
};

/**
 * 台本中の `...` をインラインコードとして描く。コマンド名やフラグが地の文から
 * 切り出されていると、技術的な内容では読み取りがはっきり速くなる。
 */
export const Rich: React.FC<{ text: string }> = ({ text }) => (
  <>
    {text.split(/(`[^`]+`)/g).map((part, i) =>
      part.startsWith("`") && part.endsWith("`") && part.length > 1 ? (
        <code key={i} style={codeStyle}>
          {part.slice(1, -1)}
        </code>
      ) : (
        <React.Fragment key={i}>{part}</React.Fragment>
      )
    )}
  </>
);
