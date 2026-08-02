> [!IMPORTANT]
> 未レビュー

# 技術ダイジェスト自動生成パイプライン 技術調査・構成検討

技術に関する最新情報をバッチで収集し、**同一内容の「動画」と「記事」**を自動生成して pechka 上で閲覧できるようにする仕組みの技術調査と構成案である。ユーザーストーリーは [102_tech_digest_user_stories.md](./102_tech_digest_user_stories.md) にある。

## 1. 出発点と前提

### 1.1 参照イメージ

参照として示された動画は「編集ソフトも動画生成AIも使わずに、解説動画を作っています【制作の裏側を解説】」（<https://www.youtube.com/watch?v=Jul3isnP5qQ>）である。

> 注: 動画ページは認証なしでは説明文まで取得できず、確認できたのはタイトルのみである。以下はタイトルが示す方向性からの推定を前提に置いている。

タイトルから読み取れる方針は明確で、**Sora 等の動画生成 AI で映像そのものを作るのではなく、スライドとナレーションをコードで合成して解説動画を組み立てる**アプローチである。この方針は本パイプラインでも採用する。理由は以下の通り。

- 動画生成 AI は事実を正確に描けない。技術解説という題材では図・コード・箇条書きの正確さが本質であり、生成映像は害にしかならない
- スライド合成なら **CPU だけで完結**し、既存の k8s ノード上のバッチとして無理なく回る（GPU 前提にならない）
- 台本（テキスト）が中間生成物として残るため、**同じ台本から記事も生成できる**。「動画と記事を似た内容で作り分ける」という要求に構造的に直結する

### 1.2 既存資産の棚卸し（最重要）

調査の結果、**pechka はこの機能に必要なピースの大半をすでに持っている**ことが分かった。新規に作るのは実質「収集」「台本生成」「スライド＋音声のレンダリング」の 3 つだけで、配信・検索・編集・管理はすべて既存機構に載る。

| 必要なもの | 既存資産 | 状態 |
| :-- | :-- | :-- |
| バッチ実行基盤 | Argo Workflows（`etl-bluray`, `subtitle-gen`） | そのまま使える |
| 動画の ABR 配信 | `batch/etl` transform/load（ffmpeg → HLS → MinIO → PostgreSQL） | MP4 を入力に流し込むだけ |
| カタログ・検索 | PostgreSQL → Benthos CDC → MongoDB / Elasticsearch | そのまま使える |
| 字幕 | `subtitle_tracks` / `subtitle_cues` + WebVTT 動的生成 + admin 編集 UI | **台本と同型のデータモデル。Whisper を通さず直接投入できる** |
| サムネイル | `batch/generate-thumbnail` | タイトルスライドで代替可能（後述） |
| 記事の受け皿 | `content_type: document`（`docs/101` Phase 6、型定義のみ済み） | 先行実装する形になる |
| LLM | nuage-cluster の `lm-server`（Ollama / ROCm / `batiai/qwen3.6-27b:iq3`） | HTTP で叩ける |
| ヘッドレスブラウザ | `frontend` に Playwright 導入済み（e2e 用） | 参考にはなるが、映像は Remotion が自前の Chrome を使う（§2.4） |

つまり本件は「新しいサブシステムの追加」ではなく、**既存 ETL パイプラインへの新しい入力経路の追加**として設計するのが正しい。

---

## 2. 技術選定

### 2.1 情報収集

RSS と公式 API のみを対象とする。

| ソース | 取得方法 | 備考 |
| :-- | :-- | :-- |
| Hacker News | Firebase API（`hacker-news.firebaseio.com`） | 公式・無料・認証不要。`topstories` + `item` |
| Zenn / Qiita トレンド | RSS | |
| Publickey / InfoQ Japan | RSS | |
| クラウド・OSS 公式ブログ（AWS, GCP, Kubernetes, CNCF 等） | RSS | 一次情報として優先度を上げる |
| GitHub Releases | GitHub REST API | ウォッチ対象リポジトリの新リリース。自分が実際に使っている OSS に絞ると価値が高い |
| arXiv（cs.*） | arXiv API | 任意。ノイズが多いので初期スコープ外 |

**法務・規約上の方針**: `docs/406` では歌詞サイトのスクレイピングを利用規約を理由に見送っている。同じ判断基準をここでも適用する。

- **HTML スクレイピングはしない**。RSS / 公式 API で取得できるものだけを対象にする
- 記事本文の全文を保存・再配布しない。DB に保持するのは URL・タイトル・要約対象として一時取得したテキストまでとし、**成果物（動画・記事）に載せるのは「見出し + 自分の言葉による要約 + 出典リンク」に限定**する
- `robots.txt` と各サイトの利用規約を尊重する。取得間隔も常識的な範囲に抑える

### 2.2 選別・台本生成（LLM）

**本番は `agy` CLI を採る。**

当初はローカルの `lm-server`（Ollama）も検討されたが、台本の品質と安定性を重視し `agy` CLI に一本化している。

処理は 2 段に分ける。段ごとにプロバイダを変えられる構成にしておくと、選別だけローカルに落として API 呼び出しを減らす、といった調整ができる。

1. **選別（filter）**: 収集した数十〜数百件から今日取り上げる 3〜5 件を選ぶ。分類タスクに近く、ローカル LLM でも十分に務まる
2. **台本化（compose）**: 選ばれた記事から構造化台本を書く。品質が効くのはここ。`agy` CLI を使う

出力は構造化 JSON に固定する（後述）。`agy -p` に台本スキーマを渡して JSON を書かせ、`batch-tech-feed` 側の `script.py` で検証してから後段へ流す。**バッチが受け取る境界は常に検証済みの台本であり、LLM の出力をそのままレンダリングに渡さない。**

#### 台本の中間形式（このパイプラインの中核）

LLM の出力を**構造化 JSON に固定**する。これが動画・記事・字幕の 3 つに分岐する単一のソースになる。

```json
{
  "digest_date": "2026-08-01",
  "title": "今日の技術トピック 3選",
  "intro_narration": "こんにちは。今日の技術トピックを3件お届けする。",
  "sections": [
    {
      "seq": 1,
      "slide": {
        "layout": "bullets",
        "title": "Kubernetes 1.35",
        "subtitle": "XXX が GA に",
        "items": ["XXX が GA に昇格", "ベータ時からの変更点は 2 つ", "アップグレード時の注意"]
      },
      "narration": [
        { "text": "1つ目は、Kubernetes 1.35 のリリースについて。", "focus": null },
        { "text": "目玉は XXX の GA 昇格である。", "focus": 0 },
        { "text": "本番投入を見送っていた構成でも、そのまま使えるようになる。", "focus": 0 }
      ],
      "article_md": "## Kubernetes 1.35 で XXX が GA に\n\n（記事側の本文。動画のナレーションより情報密度を上げ、コード片や表を含めてよい）",
      "sources": [
        { "title": "Kubernetes v1.35 Release Notes", "url": "https://…", "publisher": "kubernetes.io" }
      ]
    }
  ]
}
```

設計上の要点:

- `narration` は**文単位の配列**にする。TTS の合成単位・スライドの遷移単位・字幕 cue の単位がすべてこれに揃い、タイミング計算が 1 種類で済む
- 各文が持つ `focus` は「スライドのどの項目まで出して、どれを強調するか」を指す。`null` は状態の据え置きを意味し、**1 項目を複数の文で語る**構成を自然に書けるようにする（据え置いた文では絵が変わらない）
- `article_md` を `narration` とは別フィールドで持つ。「似た内容だが同一ではない」という要求（読み物には図表とコードが要る、聞き物には冗長さが要る）を素直に表現できる
- `sources` を必須にする。出典リンクの明示は §2.1 の方針の実装そのものである

> この形式は MVP-1（`batch-tech-feed`）で実装・検証済みである。実際に動く定義は `batch-tech-feed/techfeed/script.py`、サンプルは `batch-tech-feed/examples/script_sample.json` にある。`article_md` は記事化（MVP-4）で使うフィールドで、MVP-1 では未使用。

### 2.3 音声合成（TTS）

**第一候補は AivisSpeech Engine**。

| 候補 | 評価 |
| :-- | :-- |
| **AivisSpeech Engine** | VOICEVOX 互換 HTTP API。ONNX Runtime ベースで **CPU だけで高速動作**。Docker で一発起動。Style-Bert-VITS2 系のため日本語品質が高い。エンジン本体は LGPL-3.0 |
| VOICEVOX ENGINE | API 互換のため差し替え可能。実績は最も豊富。ただしキャラクターごとに利用条件・クレジット表記義務がある |
| Style-Bert-VITS2 直 | 品質は最高評価だが GPU 前提寄りで、API サーバ化の手間がかかる |
| Kokoro | 軽量・組み込みやすいが、日本語の解説ナレーション用途では上記に劣る |

API が互換なので **AivisSpeech と VOICEVOX は実質差し替え可能**であり、初手を誤っても後戻りが小さい。

注意点が 2 つある。

1. AivisSpeech Engine は公式に「一般的な PC での**単一ユーザー利用**を想定しており、多数のリクエストを高速に捌く API サーバ用途には最適化されていない」と明言している。→ **バッチからは並列度 1〜2 で逐次呼び出す**。数分の動画なら文の数は 50〜150 程度で、逐次でも十分間に合う
2. 音声モデル（話者）ごとにライセンスが異なる（ACML / ACML-NC / CC0 等）。**採用する話者のライセンスを確認し、リポジトリに記録する**。個人利用が前提なので制約は軽いが、記録は残す

#### タイミングは音声から導出する（「編集ソフト不要」の核心）

TTS の出力 WAV から各文の**実尺 (ms) が確定する**。これをタイムラインの正とし、

- スライドの切り替え時刻
- 字幕 cue の `start_ms` / `end_ms`

をすべて音声長から機械的に導出する。**手で打つタイムラインを一切持たない**。これが「編集ソフトを使わずに動画ができる」ことの実体であり、再生成が常に冪等になる理由でもある。

### 2.4 映像レンダリング

> **経緯**: 当初は Playwright で静止画を焼く方式（下表 B）を採り、実装・検証まで済ませた。
> その後「もっと動きを入れたい」という要求が出たため再評価し、**Remotion（A）へ移行した**。
> 判断を変えた理由は下の §2.4.1 に残す。同じ検討を繰り返さないためである。

| 案 | 内容 | 評価 |
| :-- | :-- | :-- |
| **A. Remotion** | React で動画を記述し Chrome で全フレームをレンダリング | **採用**。表現力が唯一の上限なし。ライセンスは個人／従業員 3 名以下なら無料で、個人利用の本件は該当する |
| B. HTML → Playwright で PNG → ffmpeg で合成 | 絵 1 枚 = PNG 1 枚。静止画 + 音声を ffmpeg で結合 | 描画量が「フレーム数」ではなく「絵の数」に比例するため桁違いに軽い。ただし**表現できるのは離散的な動きまで** |
| C. manim / MoviePy | 数式アニメーション、Python 動画合成 | 用途が違う（manim）、表現力が不足（MoviePy） |

#### 2.4.1 B 案から A 案へ判断を変えた理由

当初 A 案を退けた根拠は2つあり、**片方は誤りだった**。

- **「CPU のみだと重い」は誇張だった。** 1920x1080 の1フレームは実測 47ms（単プロセス）で、10分の動画でも単プロセス 14 分。Remotion は並列に描くため実測ではさらに速く、96 秒の動画が描画+エンコード込みで 1 分弱で焼けている。夜間バッチの予算には十分収まる
- **ライセンスは障害ではなかった。** 個人利用は無料であり、制約として併記したのは不正確だった

**正しかったのは「動きが離散的なら B が有利」という点だけである。** 箇条書きが1項目ずつ出るだけなら、2886 フレーム描いて 11 種類の絵しか出さないのは無駄で、実際 B 案では描画 141 枚（全フレーム方式の約 1/20）で済んでいた。

しかしこの利点は**動きがまばらなときにしか成立しない**。連続的な動き（図が線を1本ずつ描く、数値がカウントアップする、画像にゆっくり寄る）を1つでも入れた瞬間、毎フレーム内容が変わるので選択的レンダリングは何も節約しなくなる。そのとき手元に残るのは「自作の劣化版フレームステッパー」であり、Remotion に対する優位性が消える。

移行コストが小さいことも後押しになった。**このパイプラインの核である「音声の実尺からタイムラインを導く」はレンダラに依存しない**（§2.3）。`manifest.json` の `start_ms` / `end_ms` はそのままフレーム番号に写るため、台本の検証・TTS・タイムライン導出・音声合成はすべて残り、差し替えたのは描画層だけである。

副次的な獲得物として **Remotion Studio のライブプレビュー**がある。「レンダーして mp4 を開き直す」ループがタイムラインのスクラブに変わり、デザインを詰める速度が変わる。

#### 2.4.2 責務の分割

```
Python (batch-tech-feed/techfeed/)          Remotion (batch-tech-feed/remotion/)
  台本の検証                             映像の描画
  音声合成 → 実尺の実測      manifest    音声の多重化
  タイムライン導出        ──────────▶   mp4 の書き出し
```

`manifest.json` が両者の唯一の契約であり、各文の start/end に加えて**スライドの状態まで**含む。描画側が `focus` の解釈をやり直さずに済み、字幕の投入（MVP-2）も同じ物を読めるようにするためである。

#### 2.4.3 動きの2つの時間軸

動きは**文ごと**と**セクションごと**の2軸に分かれる。取り違えると動きが毎文リセットされるため、追加時にどちらへ乗せるかを必ず決める。

| 軸 | 用途 | 挙動 |
| :-- | :-- | :-- |
| 文ごと | 項目の出現、ハイライトの移動、絵の切り替え | 文が変わるたびに 0 に戻る |
| セクションごと | 図が育つ、画像がゆっくり寄る | セクションを通して単調に進む |

実装当初は連続的な動きも文の軸に乗せてしまい、**図が文を跨ぐたびに最初から描き直された**。動きを足すときに最初に踏む種類の誤りなので、README にも残してある。

#### 2.4.4 図と画像

技術解説では箇条書きより図の方が速く伝わる場面が多い。**Mermaid を React コンポーネントとして描く `diagram` レイアウト**を用意した。図の元は台本中のテキストなので、**外部から画像を取ってこずに図解が増やせる**のが要点であり、§2.1 の方針（本文の全文を保存・再配布しない）と衝突しない。

図はセクションの尺を使って育つ。ノードが順に浮かび、そのあとを追って矢印が `stroke-dashoffset` で伸び、最後にラベルと矢尻が出る。**これが B 案では描けなかった類の動きであり、移行の主目的である。** SVG 自体はソース文字列をキーにキャッシュし、毎フレームは要素ごとの `opacity` / `stroke-dashoffset` を計算した `<style>` を当てるだけにしている。

スクリーンショット等の画像は `figure` レイアウトで扱う。ただし**外部 URL は受け付けず、ローカルファイルのみ**とする。他サイトの画像を動画に焼き込むのは「見出し + 自分の要約 + 出典リンク」に留める方針を越えるためである。外部画像（記事の OGP 等）を取り込むかどうかは別途の判断とする（§4.4）。

### 2.5 配信（新規実装はほぼ不要）

生成した `digest.mp4` を**既存の ETL に流し込む**。

- **動画**: `batch/etl` の transform（ffmpeg ABR → HLS → MinIO）と load（`contents` / `video_variants` 登録）をそのまま再利用する。`etl-bluray` が MKV を入力にしているところを MP4 に変えるだけで、後段は完全に同一
- **字幕**: 台本 + TTS 実尺から**決定論的に生成できる**ため Whisper を通す必要がない。`subtitle_tracks` / `subtitle_cues` に直接 INSERT する。`docs/406` のバッチが持つ「published / 人手編集済みのトラックは上書きしない」ガードは同じ理由で踏襲する
- **サムネイル**: `batch/generate-thumbnail` で動画からフレームを抽出するより、**タイトルスライドの PNG をそのまま使う方が速くて綺麗**。生成済みの画像があるのだから解析する理由がない

### 2.6 記事の格納

`content_type` に **`article`** を追加する。

`document`（Phase 6 で計画済み）に相乗りする案も検討したが、`document` は NAS 上の PDF・テキストの取り込み先として想定されており、ビューアの実装（Markdown レンダリング vs PDF ビューア）も配信経路も異なる。`content_type` は `VARCHAR` なので値の追加コストはゼロであり、フロント側のビューア振り分けが素直になる方を採る。

本文の置き場は既存原則（書き込み PostgreSQL / 読み取り MongoDB / 検索 Elasticsearch）に従う。

```sql
-- migration 005（案）
CREATE TABLE digests (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    digest_date  DATE NOT NULL UNIQUE,
    status       VARCHAR(20) NOT NULL DEFAULT 'draft',  -- draft | published | error
    script_json  JSONB NOT NULL,        -- 台本そのもの。再生成の入力として保持する
    created_at   TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at   TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE articles (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    content_id  UUID NOT NULL UNIQUE REFERENCES contents(id) ON DELETE CASCADE,
    body_md     TEXT NOT NULL,
    sources     JSONB NOT NULL DEFAULT '[]',
    created_at  TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

-- 動画コンテンツと記事コンテンツを同じダイジェストに紐付ける
ALTER TABLE contents ADD COLUMN digest_id UUID REFERENCES digests(id);
```

Benthos のマッピングで MongoDB 側に `digest_id` と **`counterpart_short_id`（同じダイジェストのもう一方の形式）を非正規化**して持たせる。既存の非正規化方針（JOIN 不要）に沿っており、フロントは詳細画面で追加の問い合わせなしに「記事で読む / 動画で見る」を出せる。これが**「その日の気分で使い分ける」の実装**である。

Elasticsearch には `body_md` を入れる。記事本文の全文検索が既存の同期パイプラインだけで手に入る。

---

## 3. 全体構成

```
[RSS / Hacker News API / GitHub Releases API]
      │
      ▼ (Argo CronWorkflow: tech-digest / 毎日 03:00 JST)
┌─────────────────────────────────────────────────────────┐
│ ① collect    候補記事の収集 → candidates.json            │
│ ② compose    LLM で選別 + 台本生成 → script.json          │
│                  └─ agy CLI                               │
│ ③ synthesize AivisSpeech Engine で文ごとに WAV + 実尺     │
│ ④ render     スライド HTML → Playwright → PNG 群          │
│ ⑤ mux        ffmpeg: PNG + WAV → digest.mp4              │
└─────────────────────────────────────────────────────────┘
      │
      ├──► [既存 batch/etl transform] ABR HLS → MinIO
      │         └──► [既存 batch/etl load] contents / video_variants
      │
      ├──► subtitle_tracks / subtitle_cues（台本 + 実尺から決定論的に生成）
      │
      ├──► articles（content_type='article' の contents + body_md）
      │
      └──► thumbnail（タイトルスライドの PNG を MinIO へ）
                │
                ▼ (既存 Benthos CDC)
        [MongoDB] contents / subtitles   [Elasticsearch] 全文検索
                │
                ▼
        [Next.js] 一覧 → 詳細画面で「動画で見る / 記事で読む」を切替
```

### 3.1 コンポーネント構成

ディレクトリの形は Bluray 取り込み（`batch/etl`）に揃えた。Go 側は `main.go` が
サブコマンドを振り分け、各工程を `cmd/` に、共通の小道具を `shared/` に置く。
イメージは工程ごとに `Dockerfile.<component>` を用意する。

Python 側を Python にしているのは、`batch/generate-thumbnail` / `batch/subtitle`
（いずれも ffmpeg を伴うメディア処理バッチ）の前例に倣ったためである。

```
batch-tech-feed/
├── main.go              # Go エントリ（収集）
├── cmd/collect.go       # RSS / Hacker News / GitHub Releases
├── shared/shared.go     # HTTP・文字列処理の小道具
├── Dockerfile.collect   # → tech-feed-collect
│
├── main.py              # Python エントリ
├── techfeed/
│   ├── candidates.py    # 収集結果の受け取り
│   ├── compose.py       # LLM に台本を書かせる
│   ├── llm.py           # agy CLI 呼び出し
│   ├── script.py        # 台本の検証
│   ├── timeline.py      # 実尺からタイムラインを導出
│   ├── synthesize.py    # TTS
│   ├── narration.py     # 文ごとの wav を1本にまとめる
│   ├── renderer.py      # Remotion の呼び出し
│   ├── media.py         # ffmpeg / ffprobe
│   ├── hls.py           # mp4 → HLS バリアント
│   ├── storage.py       # MinIO
│   ├── catalog.py       # PostgreSQL
│   └── publish.py       # 公開の一連の流れ
├── remotion/src/        # 映像（Root / Digest / Slide / Mermaid）
├── Dockerfile.produce   # → tech-feed
│
├── sources.json         # 情報源
└── examples/            # 手書き台本のサンプル
```

### 3.2 実行時間の見積もり

5 分程度の動画 1 本あたりの粗い見積もり。

| 工程 | 実行場所 | 目安 |
| :-- | :-- | :-- |
| collect | k8s (CPU) | 1 分未満 |
| compose | `agy` CLI | 1〜数分（台本数千字） |
| synthesize | k8s (CPU, 逐次) | 数分（100 文程度） |
| render（Remotion。映像＋音声多重化） | k8s (CPU) | 数分（並列度に依存） |
| transcode (ABR) | k8s (CPU) | 数分〜十数分（既存 ETL と同じ特性） |

合計で 30 分程度に収まる想定であり、夜間バッチとして問題ない。

MVP-1 の実測（6 セクション / 20 文 / 96 秒の動画、`--engine mock`、Apple Silicon 上）では、**build 全体で約 68 秒**（mock TTS + 音声連結 + 2887 フレームの描画 + エンコード）。上表の見積もりには余裕がある。mp4 は 8.1 MB。

---

## 4. 主要な論点

実装に入る前に判断が必要なもの。

### 4.1 自動公開するか、人手レビューを挟むか

`docs/406` の字幕は「常に `draft` で登録し、admin 画面でのレビューを経て `published`」という設計になっている。同じ思想を適用すると、ダイジェストも毎朝レビューが必要になる。

- **レビュー必須案**: LLM の誤要約がそのまま蓄積するのを防げる。ただし毎日手を動かす必要があり、「バッチで自動生成」の価値が半減する
- **自動公開案**: 朝起きたら見られる。誤りは気づいたときに admin で直すか消す。個人利用であり外部に配信するわけではないため、実害は小さい
- **折衷案（推奨）**: 自動で `published` にするが、**LLM に自己検証させて低信頼のセクションに `flagged` を立て**、admin 一覧で目立たせる。`subtitle_cues.flagged` と同じ考え方で、既存の運用感に揃う

### 4.2 LLM をローカルで完結させるか — 決着済み

**`agy` CLI に一本化する**ことに決めた（§2.2）。台本の品質がこの機能の価値を直接決めるため、そこを未知数に賭けない判断である。

### 4.3 動画と記事の関係

「似た内容で作り分ける」を、

- (a) **1 台本 → 2 レンダラ**（本案）: 内容の一貫性が保証され、実装も単純
- (b) 独立生成: 記事は記事らしく、動画は動画らしく最適化できるが、LLM 呼び出しが 2 倍になり内容がずれる

(a) を採り、媒体差は `narration` と `article_md` をフィールドで分けることで吸収する。

### 4.4 外部の画像を取り込むか — 未決

現状、動画に載る図は **Mermaid で自前で描いたもの**と**手元のローカル画像**に限っている（§2.4）。記事の OGP 画像や、参照先ページのスクリーンショットは取り込んでいない。

- **取り込まない（現状）**: §2.1 の方針と一貫する。図は `diagram` で自前で描けるため、技術解説としては実用上ほぼ困らない
- **取り込む**: 見栄えは上がるが、他サイトの画像を自分の動画に焼き込むことになる。個人利用・非公開であっても、方針としては一段踏み込むことになる

判断を保留したまま実装は「取り込まない」側に倒してある。必要になったら、取り込み可否を情報源ごとに設定で持つのが妥当と考える。

---

## 5. 想定するフェーズ分割

| フェーズ | 内容 | 完了条件 |
| :-- | :-- | :-- |
| **D0（完了）** | 台本形式の確定、手書き台本での疎通、動き・図・画像、Remotion への移行 | 手で書いた `script.json` から動画 MP4 が焼ける → `batch-tech-feed` |
| **D2（完了）** | 配信経路への接続 | 生成した MP4 が HLS 化され pechka で再生できる → `main.py publish` |
| D1 | TTS 常駐化（実音声） | AivisSpeech を常駐させ、無音から実音声に切り替える |
| **D3（完了）** | 収集 + LLM 台本生成 | 実ソースから台本が自動生成される → `main.py daily` |
| D4 | 記事化（`content_type: article`） | 同じ台本から記事が生成され閲覧できる |
| D5 | 動画/記事の切替 UI と日次 CronWorkflow | 毎朝自動で 1 本増えている |
| D6 | 品質管理（flagged、admin からの再生成） | 誤りを見つけて直す運用が回る |

### 5.1 D0 の実装と検証結果

`batch-tech-feed` として実装した（詳細は同ディレクトリの README）。**§2.4 の B 案（HTML → Playwright → PNG → ffmpeg）が成立することを実測で確認した。**

検証したこと:

- サンプル台本（6 セクション / 20 文、全レイアウト）から 1920x1080 / 30fps / 96.3 秒 / 8.1 MB / 2887 フレームの mp4 が音声つきで生成される
- **抜き取った動画フレームが、同じフレーム番号で描いた composition の静止画と一致する**（h264 の劣化ぶんを除き PSNR 43〜46 dB）。タイムラインが端から端まで意図どおりであることの確認になっている
- **図がセクションの尺を使って育つ** — ノードが順に浮かび、矢印が伸び、ラベルと矢尻が続く
- 日本語が正しく描画される（macOS ではヒラギノ、コンテナでは `fonts-noto-cjk`）
- 台本の形式エラー（未知の layout、範囲外の `focus`、空の `items`、外部 URL の画像）が描画前に JSON パス付きで弾かれる

設計上、確認できて意味が大きかった点:

- **タイムラインがレンダラから独立している**ことが、実際にレンダラを差し替えて裏づけられた。Playwright 方式から Remotion 方式へ移す際に手を入れたのは描画層だけで、台本の検証・TTS・タイムライン導出・音声合成はそのまま動いた
- **Mermaid を React で描けるため、外部から画像を取ってこずに図解を増やせる**。§2.1 の方針と衝突しない
- **`--engine mock`（文字数から尺を概算した無音）でパイプライン全体を回せる**。尺の測定経路は本番と同じ ffprobe なので、TTS エンジンを立てずにタイミングの組み立てまで検証できる。CI に載せられる

実装中に踏んで、繰り返しやすいと判断した誤り:

- **連続的な動きを「文ごと」の時間軸に乗せた**。図が文を跨ぐたびに最初から描き直された。動きを足すときは文の軸かセクションの軸かを必ず決める（§2.4.3）
- **Remotion の props を `{"manifest": ...}` で包まずに渡した**。`defaultProps` とマージされて既定値が生き残り、**エラーにならないまま 1 秒の動画**が出てきた
- **mermaid の CSS に負けた**。mermaid は自前の `<style>` を SVG の内側に埋め込み、その規則は id で始まるため詳細度が高い。`!important` が要る
- **辺の class を `edgePath` だと思って探した**。実際は `edgePaths`（複数形）で `\b` が効かず1件も拾えず、辺だけ最初から見えていた

D1 以降への引き継ぎ:

- `manifest.json`（各文の start/end とスライドの状態）が既に出力されている。字幕（`subtitle_cues`）はこれをそのまま流し込めばよく、Whisper は不要
- 工程が `synthesize` / `render` に分かれ、中間成果物がディスクに残るため、話者やデザインだけを変えた作り直しができる（docs/102 US-5.3）
- デザインを詰めるときは Remotion Studio が使える（README 参照）

### 5.2 D2 の実装と検証結果

**Bluray 取り込みとは完全に独立したジョブ**として `batch-tech-feed/` に実装した。

Bluray 取り込みと同じく Argo Workflows の **WorkflowTemplate**（`tech-feed`）として
呼び出せるようにしてある。エントリポイントは2つ。

| entrypoint | 台本の出どころ |
| :-- | :-- |
| `daily`（既定） | 収集 → LLM が書く |
| `from-script` | ConfigMap の `script.json`（手書き） |

```
collect (Go)  ──▶  /tmp/candidates.json  ──▶  produce (Python)
```

**収集だけを別 Job（Go）に切り出している。** ネットワーク待ちが主な処理で、
Python + Node + Chrome を積んだ重いイメージを持ち出す必要がないためである。副次的に、
収集側は外部ストレージも DB も触らないので**認証情報を一切必要としない**。

段をまたぐ受け渡しは `/tmp` のファイルを Argo の `outputs.parameters` で渡す形にした。
Bluray 取り込みが `mkv-files.json` / `short-id` を受け渡しているのと同じである。

一方 produce 側（台本生成 → 音声 → 描画 → 公開）は1コンテナのままにしてある。中間成果物
（wav 群・mp4・HLS セグメント）が大きく、これらを段ごとに受け渡すと PVC か artifact
経由の仕組みが要るのに対し、分けて得られるものが乏しいため。

**情報源は API キーを必要としない。** RSS・Hacker News Firebase API・GitHub Releases API の
いずれも未認証で叩けることを実測で確認している。GitHub だけ未認証時のレート制限が
60 req/hour と低いが、1リポジトリ1リクエストなので現状の規模では問題にならない。

コード・イメージ・ビルドスクリプト・DB の同一性キーも共有していない。**DB スキーマは変更なし**。
共有するのは配信先（MinIO の `resources/hls` と PostgreSQL の `contents`）と、
デプロイ経路（Argo CD が `k8s/base` を同期する）だけである。

その代償として **ABR ラダーとマスタープレイリスト生成のロジックが2箇所に存在する**。ただし
狙いが違うため揃える必要もない — 画面収録に近い内容では 480p まで落とすと文字が潰れて読めず、
帯域を節約する意味がないので、tech-feed 側は `original` と `720p` の2本だけにしている。

ローカルに PostgreSQL 16 と MinIO を立てて検証した（モックではなく実物）。

- **既存のマイグレーション 001〜004 のみ**を適用した DB に対し、`contents` 1行 + `video_variants` 3行が登録される（スキーマ変更なしで動く）
- MinIO に 26 個の `.ts` / `.m3u8` と `master.m3u8`、サムネイル 1 枚が正しい Content-Type で置かれる
- **アップロード済みの `master.m3u8` を HTTP 経由で ffprobe すると、1920x1080 と 1280x720 の
  2 レンディションが読め、720p は実際にデコードできる**（プレーヤーと同じ経路での確認）
- 同じ `--source-key` で再実行しても `contents` は 1 行のまま、`short_id` と `published_at` も変わらない
- 別の `--source-key` なら別コンテンツとして増える
- Benthos が MongoDB へ流すのと同じ SQL を実行し、`disc_label: null`（ディスクなし）でも
  期待どおりの JSON になることを確認した

#### 同一性キー — スキーマを変えずに済ませた

当初は `contents.source_key` 列を足す migration を書いたが、**不要だった**ので取り下げた。

`short_id` は VARCHAR(50) UNIQUE で、API・フロント・バッチのどこでも文字列として扱われている
（数値としてパースしている箇所はない）。したがって取り込み元のキーから決定的に導けば、
既存の UNIQUE 制約がそのまま「再実行しても増えない」を担保する。

```
source_key "tech-feed:2026-08-01"  →  short_id "tech-feed-2026-08-01"
```

Bluray の `disc_id` とは独立した仕組みであり、共有スキーマには一切触れていない。
副次的に URL も読みやすくなる（`/contents/tech-feed-2026-08-01`）。


---

## 6. 採用しなかった案

- **動画生成 AI（Sora 等）で映像を作る案**: 技術解説では図・コード・箇条書きの正確さが本質であり、生成映像は正確さを担保できない。参照動画の方針とも一致しない
- **`content_type: document` に記事を相乗りさせる案**: §2.6 の通り、想定用途とビューア実装が異なるため `article` を新設する
- **静止画を並べる案（Playwright → PNG → ffmpeg）**: 一度は採用して実装まで済ませたが、連続的な動きを入れると利点が消えるため Remotion に移した（§2.4.1）
- **記事本文を MinIO に静的ファイルとして置く案**: 編集のたびにファイル再生成とキャッシュ無効化が要る。`docs/406` で静的 VTT を見送ったのと同じ理由で、DB 保持 + CDC 同期に揃える
- **HTML スクレイピングによる収集**: §2.1 の通り、`docs/406` の判断基準を踏襲して RSS / 公式 API に限定する
