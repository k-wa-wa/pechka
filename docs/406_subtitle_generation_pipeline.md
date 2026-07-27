# ライブ字幕自動生成パイプライン設計

## 1. 概要

既に取り込み済みのライブ映像コンテンツ（MinIO `mkv/` 配下）を対象に、日本語の字幕（文字起こしベース）を自動生成し、pechka上で配信・人手修正できるようにする。

元の設計（全自動ライブ字幕生成パイプライン設計書）は 5 Phase 構成（音声抽出 → Demucs ボーカル分離 → WhisperX 文字起こし → 歌詞検索・DPアライメント → 字幕多重化）だったが、k8s上でのCPU検証（PoC）の結果を踏まえ、**まず成立させられる範囲**に絞って実装した。Demucsによるボーカル分離、歌詞検索・DPアライメントによる表記補正・曲の特定はTODOとして切り出し、v1のスコープには含めない。曲の特定・歌詞の正誤判定は、admin画面からの人手修正で行う運用とする。

## 2. PoCで得られた知見

検証は使い捨てのk8s Jobで実施した（PoC資材は役目を終えたため削除済み）。4本のソース・パラメータで実行し、以下が判明している。

| # | 内容 | 結果 |
| :-- | :-- | :-- |
| 1 | GPU（lm-server, ROCm）ではなくk8s上のCPU（faster-whisper large-v3, int8, 4 threads）で実行 | 10分音声で約270〜280秒（約2.2倍速）。実用速度と判断し、GPU/ROCm整備は当面見送り |
| 2 | `vad_filter=True`（Silero VAD有効）で歌唱区間を処理 | **歌唱がほぼ丸ごと「非発話」と判定され消える**。9分の歌唱区間が4セグメント(12秒分)にまで削られた |
| 3 | 同区間を`vad_filter=False`で再処理 | 90セグメント・約10分フルの歌詞が出力され、2曲分の歌詞が文脈的に一貫して取れた |
| 4 | 別ソース（Blu-ray, 20分, VAD無効）を通し処理 | MCなしでいきなり歌唱から始まる構成でも良好。ただし**冒頭103秒（MKVのChapter境界と一致）が「ご視聴ありがとうございました」×3回→「ハー」の反復というハルシネーションループ**になった |

結論として、**歌唱を含むコンテンツでは`vad_filter=False`が必須**。無音・観客ノイズのみの区間ではハルシネーションが発生しうるため、後処理での検知フラグを実装している（§4.3）。

固有名詞の表記ゆれ（「野音」→ヤオン/矢音/矢尾/ヤンゴ、「BiSH」→美酒 等）も確認しているが、`initial_prompt`による補正は未検証（TODO）。

## 3. 全体アーキテクチャ

```
[入力: 既取り込み済み MKV (MinIO mkv/{DISC_LABEL}/*.mkv)]
       │
       ▼ (Argo Workflow: subtitle-gen / 手動実行)
[batch/subtitle: 音声抽出(ffmpeg) → 文字起こし(faster-whisper) → ハルシネーション検知]
       │
       ▼
[PostgreSQL] subtitle_tracks (status=draft) + subtitle_cues ─── 書き込み・編集の単一の情報源
       │
       ├─► [admin UI] cue単位で編集・削除・挿入、トラックのdraft/published切り替え
       │
       ▼ (Benthos CDC, 2秒ポーリング)
       published のみ → [MongoDB] subtitles コレクション ─── 配信専用の非正規化ドキュメント
       draft に戻された場合は delete-one で同期解除
       │
       ▼
[API] GET /v1/contents/:short_id/subtitles/:lang → WebVTTを動的生成
       │
       ▼
[フロントエンド] <video><track kind="subtitles" src="..."></video>
```

元設計との対応: 元Phase 1（音声抽出）はそのまま踏襲。元Phase 2（Demucs）はTODO。元Phase 3（WhisperX + wav2vec2アライメント）はfaster-whisper単体に簡略化（文字単位タイムスタンプ・カラオケ字幕は非対応）。元Phase 4（歌詞照合）はadmin画面での人手修正に置き換え。元Phase 5（多重化）はHLSコンテナへの多重化ではなく、WebVTTの動的配信に置き換えている。

## 4. 実装詳細

### 4.1 バッチ（`batch/subtitle/generate.py`）

`batch/generate-thumbnail`と同じPython製バッチの構成に倣っている。

- MinIOの署名付きURLをffmpegに直接渡して音声抽出する（`batch/generate-thumbnail`と同様、MKV全体をダウンロードしない）
- faster-whisper large-v3, int8, CPU 4 threads, `vad_filter=False`固定（§2の知見）
- 連続する同一テキストが2回以上続く場合、ハルシネーション疑いとして`flagged=true`を立てる（「ご視聴ありがとうございました」×3連続のようなループ対策）
- Postgresへ直接書き込み（既存トラックは削除して丸ごと入れ替え = 再実行は冪等）。**常に`status='draft'`で登録**し、自動公開はしない
- Demucs、`initial_prompt`、曲の特定・歌詞照合はいずれも未実装（TODO）

### 4.2 データモデル（PostgreSQL, migration 004）

```sql
subtitle_tracks (id, content_id, language, status[draft|published], model, created_at, updated_at)
subtitle_cues   (id, track_id, seq, start_ms, end_ms, text, original_text, flagged, updated_at)
```

- `status`はdraft/publishedの2値のみ。個別行のレビュー状態は`flagged`側で表現する
- `original_text`にWhisperの生出力を保持し、`text`を編集可能にする（diff/revert用。revert UI自体は未実装）
- `flagged`はcue単位。admin画面が該当行をハイライトし、人手で修正すると`flagged=false`に戻る（`UpdateCue`側で自動クリア）

### 4.3 配信（MongoDB + Benthos CDC）

`k8s/base/infra/benthos/benthos.yaml`に`subtitles.yaml`ストリームを追加した。

- 2秒間隔でPostgresをポーリングし、`subtitle_tracks` × `subtitle_cues`をJOINして`short_id:language`をキーにMongoDBの`subtitles`コレクションへ同期する
- **`status == "published"`のみ`upsert`、それ以外は`delete-one`**という`output.switch`分岐にしている。draftのまま/draftに戻された場合はMongoDBから確実に消えるため、公開ゲートとして機能する（既存の`contents.yaml`ストリームは行単位の削除を扱えていないため、これは意図的な改善）
- 既存の設計方針「読み取りはMongoDB経由」を踏襲。字幕配信はPostgres直読みにしない
- `contents.yaml`側にも`has_subtitles`（`EXISTS(... status='published')`）を追加し、フロントが字幕の有無を判定できるようにした

### 4.4 API（Go）

- `GET /v1/contents/:short_id/subtitles/:lang`: MongoDBの`subtitles`コレクションから読み、WebVTTを動的整形して返す（`Content-Type: text/vtt`, `Cache-Control: no-cache`）。draft は同期されないため、このハンドラは公開可否を判定する必要がない
- admin CRUD: `GET/PUT /v1/admin/contents/:content_id/subtitles`, `/v1/admin/subtitles/:track_id/status`, `/v1/admin/subtitles/:track_id/cues`, `/v1/admin/subtitles/cues/:cue_id`（既存の`AdminHandler`に相乗り）

### 4.5 フロントエンド

- `VideoPlayer.tsx`: `<video>`に`<track kind="subtitles" src="/api/v1/contents/{shortId}/subtitles/ja">`を追加。HLSの画質バリアント切り替え（`hls.js`のsrc差し替え）とは独立しているため、画質を切り替えても字幕表示が途切れない（HLSネイティブのSUBTITLESレンディションは採用しなかった。理由は§5参照）
- `AdminTable.tsx`に「字幕」ボタンを追加し、`SubtitleEditorModal.tsx`でcueの一覧表示・テキスト編集・挿入・削除・公開/非公開切り替えができる

### 4.6 Argo Workflow

`etl-bluray`（ingest-flow）とは分離した`subtitle-gen` WorkflowTemplate（`k8s/base/etl/subtitle-workflow.yaml`）とした。処理時間が長く（フル尺で数十分規模）、既存のHLS即時公開フローをブロックしたくないため。`content-id` / `mkv-path`を指定して手動実行する（バックフィル・新規取り込み後の追加実行いずれにも同じテンプレートを使う）。

## 5. 採用しなかった案

- **HLSネイティブのSUBTITLESレンディション**（`master.m3u8`に`#EXT-X-MEDIA:TYPE=SUBTITLES`を追加し、hls.jsの`subtitleTracks` APIで扱う）: `VideoPlayer.tsx`が画質バリアントを直接`src`切り替えする実装のため、全バリアントにSUBTITLES属性を持たせる改修が必要になり複雑。素の`<track>`要素の方がシンプルかつ画質切り替えに対して堅牢
- **字幕配信をPostgres直読みにする案**: admin系エンドポイントと同様に許容範囲内ではあったが、既存の「読み取りはMongoDB経由」という設計方針との一貫性を優先し、Benthos CDC経由に統一した
- **静的VTTファイルをMinIOに保存する案**: cue単位の編集と相性が悪く（編集の都度ファイル再生成・キャッシュ無効化が必要）、動的生成で十分軽量なため見送った

## 6. TODO（今回のスコープ外）

- **曲の特定・歌詞の正誤判定の自動化**: 元設計Phase 4相当（歌詞検索・Needleman-Wunsch DPアライメント）。歌詞サイト（UtaTen, J-Lyric.net等）のスクレイピングは技術的には検証済みだが、両サイトとも利用規約で「閲覧」以外の複製・自動収集を明示的に禁止しており、方針判断が保留中。当面はadmin画面での人手修正が正の手段
- **Demucsによるボーカル分離**: Phase 2前段としての効果は未検証。VAD無効での素通し処理である程度の精度が出ているため、優先度は高くない
- **`initial_prompt`による固有名詞補正**: 未検証。バンド名・会場名等の表記ゆれ対策として次に試す価値がある
- **複数音声トラックの扱い**: MKVに本編/コメンタリー/多言語音声等が複数ある場合の選択方針（`docs/405`から持ち越しの既知課題）
- **カラオケ字幕（`\k`タグ相当）**: 文字単位タイムスタンプが必要だが、faster-whisper単体では非対応（元設計のwav2vec2アライメント相当が必要）
- **専用コンテナイメージ化**: 現状はJob起動のたびにapt/pipインストール・モデルダウンロードが走る。ビルド済みイメージ（`etl-subtitle`）への焼き込み
- **cueのoriginal_textへのrevert UI**: データは保持しているが、admin画面に「ASR結果に戻す」ボタンは未実装
- **新規取り込み後の自動起動**: 現状は手動実行のみ。パイプラインが安定してから`ingest-flow`との連携や自動トリガーを検討する
