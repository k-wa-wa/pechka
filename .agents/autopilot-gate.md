PR の Preview 環境に対して以下を実際に実行して確認する。

CI（型検査・`go test`・Storybook・モックを使った VRT）は、実際の API・DB・nginx・フロントエンドを
結合した状態を検証しない。ここでは結合後の Preview に HTTP リクエストを送って確認する。

## 守ること: 読み取り専用

Preview の MinIO は本番の実体を共有している（`k8s/overlays/preview/README.md`）。
書き込みは本番のバケットに入るため、**GET / HEAD 以外のリクエスト（POST・PUT・PATCH・DELETE）は送らない**。
`/api/v1/admin/...` の書き込み系や `/api/v1/contents/ingest` も対象外。
書き込み系を変更する PR は、実行せずハンドラのコードとテストを読んで確認する。

## 検証対象

PR ごとの Preview 環境 `https://pechka-pr-<PR番号>.wpcapp.net` に対して確認する。

```bash
BASE="https://pechka-pr-<PR番号>.wpcapp.net"
```

Preview は PR への push から数分後に立ち上がり、DB の seed とマイグレーションが終わるまで API が 5xx を返す。
`/health` は nginx が直接返すため、API の準備完了の判定には使わない。
API が応答するまで 60 秒間隔で最大 10 回リトライする。

```bash
for i in $(seq 1 10); do
  code=$(curl -sS -o /dev/null -w "%{http_code}" --max-time 15 "${BASE}/api/v1/contents?limit=1" || echo "000")
  echo "attempt $i: HTTP $code"
  [ "$code" = "200" ] && break
  sleep 60
done
```

10 回とも 200 にならない（`000`・5xx）場合、コードの良否は判定できない。
以降の確認は行わず、どのコマンドがどう失敗したかを事実として報告する。

## 確認項目

### 1. フロントエンドのページ

```bash
page() { # $1=path $2=期待するHTTPステータス $3=HTMLに含まれる文字列
  code=$(curl -sS --max-time 30 "${BASE}$1" -o /tmp/gate.html -w "%{http_code}")
  [ "$code" = "$2" ] && grep -q -- "$3" /tmp/gate.html && echo "OK: $1" || echo "NG: $1 (HTTP $code, 期待 $2)"
}
page /               200 "<title>pechka</title>"
page /admin          200 "Admin — pechka"
page /contents/nope  404 "<title>pechka</title>"
```

### 2. API（契約は `api/openapi.yaml`）

```bash
api() { # $1=path $2=期待するHTTPステータス
  code=$(curl -sS --max-time 30 "${BASE}$1" -o /tmp/gate.json -w "%{http_code}")
  [ "$code" = "$2" ] && python3 -c "import json;json.load(open('/tmp/gate.json'))" 2>/dev/null \
    && echo "OK: $1" || echo "NG: $1 (HTTP $code, 期待 $2)"
}
api "/api/v1/contents?limit=3"           200   # 公開の一覧（MongoDB）
api "/api/v1/search?q=a&limit=3"         200   # 全文検索（Elasticsearch）
api "/api/v1/search"                     400   # q 未指定
api "/api/v1/contents/does-not-exist"    404
api "/api/v1/admin/contents?limit=3"     200   # 管理用の一覧（PostgreSQL）
```

- 一覧と検索が空配列でも、それだけで不合格にしない。Preview の MongoDB と Elasticsearch は
  本番から複製されず空の場合がある。PostgreSQL だけが本番から seed される。
- 管理用の一覧は、seed とマイグレーションの結果を反映する。ここが 5xx、または件数が 0 の場合は不合格とする
  （`total` があれば 1 以上、配列なら 1 件以上）。
- 応答の形は `api/openapi.yaml` と一致していること。食い違う場合は不合格とする。

### 3. メディア配信（nginx から本番共有の MinIO への経路）

管理用の一覧（PostgreSQL 由来）から `ready` のコンテンツを 1 件選び、HLS を辿る。

```bash
SID=$(curl -sS --max-time 30 "${BASE}/api/v1/admin/contents?limit=100" | python3 -c '
import json,sys
d=json.load(sys.stdin); items=d["contents"] if isinstance(d,dict) else d
print(next((c["short_id"] for c in items if c["status"]=="ready"), ""))')
echo "short_id: ${SID:-なし}"
if [ -n "$SID" ]; then
  curl -sS --max-time 30 -o /tmp/gate.m3u8 -w "master.m3u8: HTTP %{http_code} %{content_type}\n" "${BASE}/resources/hls/${SID}/master.m3u8"
  head -1 /tmp/gate.m3u8                                  # #EXTM3U であること
  V=$(grep -v '^#' /tmp/gate.m3u8 | grep -v '^$' | head -1)
  curl -sS --max-time 30 -o /tmp/gate-v.m3u8 -w "${V}: HTTP %{http_code}\n" "${BASE}/resources/hls/${SID}/${V}"
  SEG=$(grep -v '^#' /tmp/gate-v.m3u8 | grep -v '^$' | head -1)
  curl -sS --max-time 30 -r 0-1023 -o /dev/null -w "${SEG}: HTTP %{http_code}\n" "${BASE}/resources/hls/${SID}/${SEG}"
fi
curl -sS --max-time 30 -o /dev/null -w "存在しないリソース: HTTP %{http_code}\n" "${BASE}/resources/hls/nope/master.m3u8"
```

- master.m3u8 が 200 で先頭が `#EXTM3U`、バリアントのプレイリストが 200、セグメントが 200 または 206、
  存在しないリソースが 404 であること。
- `ready` のコンテンツが 1 件も無い場合は、この項目を確認できなかった事実だけを報告する。

### 4. 公開コンテンツの詳細（一覧が空でない場合のみ）

```bash
SID=$(curl -sS --max-time 30 "${BASE}/api/v1/contents?limit=100" | python3 -c '
import json,sys
d=json.load(sys.stdin); print(next((c["short_id"] for c in d if c["status"]=="ready"), ""))')
if [ -n "$SID" ]; then
  api "/api/v1/contents/${SID}" 200
  api "/api/v1/contents/${SID}/variants" 200
  page "/contents/${SID}" 200 "<title>"
  TK=$(curl -sS "${BASE}/api/v1/contents/${SID}" | python3 -c 'import json,sys;print(json.load(sys.stdin).get("thumbnail_key") or "")')
  [ -n "$TK" ] && curl -sS -o /dev/null -w "thumbnail: HTTP %{http_code} %{content_type}\n" "${BASE}/thumbnails/${TK#thumbnails/}"
else
  echo "公開の一覧に ready のコンテンツが無い（Preview の MongoDB が空の可能性）。詳細は確認できなかった。"
fi
```

すべて 200 で、サムネイルが `image/jpeg` であること。一覧が空で確認できない場合は、その事実だけを報告する。

### 5. この PR の変更点

- この PR が変更・追加した API の挙動や画面は、差分を読み、同じ要領で GET して確認する。
  正常系だけでなく、その変更が影響しうる異常系（不正なパラメータ、存在しない ID など）も 1 つ以上叩く。
- PR にマイグレーションが含まれる場合は、管理用の一覧が 200 で件数があることを、マイグレーション適用後の
  スキーマでの動作確認とみなす。
- 書き込み系の変更は上記の方針どおり実行せず、コードとテストで確認する。
