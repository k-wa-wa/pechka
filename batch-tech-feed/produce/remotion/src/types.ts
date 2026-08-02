/** manifest.json の型。Python 側 (digest/timeline.py to_manifest) と対になる契約である。 */

export type Layout = "title" | "bullets" | "code" | "figure" | "diagram";

export type SlideState = {
  layout: Layout;
  header: string;
  title: string;
  subtitle: string;
  items: string[];
  /** items のうち何件目までを表示するか */
  revealed: number;
  /** いま読み上げている項目の index */
  highlight: number | null;
  code: string;
  language: string;
  /** data URI に畳まれた画像 */
  image: string;
  caption: string;
  /** Mermaid のソース */
  diagram: string;
  sources: string[];
  section_seq: number;
  section_total: number;
};

export type Entry = {
  seq: number;
  section_seq: number;
  text: string;
  start_ms: number;
  end_ms: number;
  audio: string;
  state: SlideState;
};

export type Manifest = {
  digest_date: string;
  title: string;
  fps: number;
  total_ms: number;
  /** public dir からの相対名。renderer.py が --public-dir で out ディレクトリを渡す */
  narration: string;
  entries: Entry[];
};
