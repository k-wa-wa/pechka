# pechka

動画・画像・VR・ドキュメント等、あらゆる種類のコンテンツを NAS やディスク等から取り込み、ブラウザで閲覧・配信するホームメディア基盤。Bluray に限らず様々なデータソース（NAS、ファイルサーバ、各種メディア）への対応を視野に入れ、コンテンツを AI へのインプットとして活用することも中核的なユースケースとして想定している。

## 概要

既存のホームオートメーション関連コードを整理し、以下の構成で再構築しています。
詳細は `docs/` フォルダを参照してください。

## ドキュメント

- [101_requirements.md](docs/101_requirements.md) — 要件定義書
- [201_architecture.md](docs/201_architecture.md) — アーキテクチャ設計書
- [405_bluray_ingestion_pipeline.md](docs/405_bluray_ingestion_pipeline.md) — Bluray ETL パイプライン設計

## 現在のステータス

- **Phase 0: クリーンアップ + ドキュメント整備** (進行中 - PR #3)
- **Phase 1: インフラ・DB 基盤** (準備中)
- **Phase 2: API Service 実装 (Go)** (準備中)
- **Phase 3: フロントエンド実装 (Next.js)** (準備中)
- **Phase 4: Bluray ETL パイプライン実装** (準備中)
- **Phase 5: 最適化・運用整備** (準備中)

## ローカル開発環境の起動・動作確認手順

Kind (Kubernetes in Docker) を使ってローカルに本番同等の環境を構築します。

### 1. クラスタの作成とデプロイ

```bash
# クラスタ作成
kind create cluster --name pechka-cluster

# namespace 作成（kustomize が自動作成しないため必要）
kubectl apply -f k8s/namespace.yaml

# dev overlay を適用（全リソースを一括デプロイ）
kubectl apply -k k8s/overlays/dev
```

### 2. コンテナイメージのビルドと Kind へのロード

```bash
# ETL イメージをビルドして Kind クラスタにロード
bash etl/build-and-load.sh

# API・フロントエンドイメージをロード
kind load docker-image \
  ghcr.io/k-wa-wa/pechka-api:latest \
  ghcr.io/k-wa-wa/pechka-frontend:latest \
  --name pechka-cluster

# Deployment を再起動してイメージを反映
kubectl rollout restart deployment/api deployment/frontend -n pechka
kubectl rollout status deployment/api deployment/frontend -n pechka
```

### 3. ポートフォワードとアクセス確認

```bash
# バックグラウンドでポートフォワード起動
nohup kubectl port-forward svc/nginx -n pechka 8000:80 > /tmp/nginx-pf.log 2>&1 &

# 疎通確認
curl -s http://localhost:8000/api/v1/contents
```

ブラウザで **[http://localhost:8000](http://localhost:8000)** にアクセスできます。

---

### 4. テストデータ投入（K8s ETL パイプライン）

Bluray ドライブなしで MKV を手動生成し、本番同様のパイプラインを回します。

#### 4.1. `pechka-ffmpeg` イメージを Kind にロード

```bash
kind load docker-image pechka-ffmpeg:latest --name pechka-cluster
```

#### 4.2. ダミー MKV の生成

```bash
kubectl exec -n pechka etl-importer -- mkdir -p /mnt/bluray/mkv/TEST_DISC_001

kubectl run make-dummy-mkv \
  --image=pechka-ffmpeg:latest \
  --namespace pechka \
  --image-pull-policy=IfNotPresent \
  --restart=Never \
  --overrides='{"spec":{"volumes":[{"name":"bluray-volume","persistentVolumeClaim":{"claimName":"nfs-bluray-pvc"}}],"containers":[{"name":"make-dummy-mkv","image":"pechka-ffmpeg:latest","imagePullPolicy":"IfNotPresent","volumeMounts":[{"name":"bluray-volume","mountPath":"/mnt/bluray"}],"command":["ffmpeg","-f","lavfi","-i","testsrc=duration=5:size=640x360:rate=30","-f","lavfi","-i","sine=frequency=1000:duration=5","-c:v","libx264","-c:a","aac","-y","/mnt/bluray/mkv/TEST_DISC_001/title_00.mkv"]}]}}' \
  && kubectl wait --for=condition=complete pod/make-dummy-mkv --timeout=120s -n pechka

kubectl delete pod make-dummy-mkv -n pechka
```

#### 4.2. ETL コンテナイメージのビルドと Kind へのロード

以下のスクリプトを実行して、すべての ETL コンポーネントを Docker イメージとしてビルドし、Kind クラスターへロードします。

```bash
bash etl/build-and-load.sh
```

#### 4.3. ETL パイプラインの実行

`etl/run-pipeline.sh` は、実行環境の Kubernetes コンテキスト（Kind）を自動検出し、クラスター上に配置された環境毎のマニフェスト（パッチ適用済みのプレースホルダー定義）を動的に読み込んで Job をインスタンス化します。

```bash
bash etl/run-pipeline.sh \
  --disc-label TEST_DISC_001 \
  --title "サンプル動画コンテンツ" \
  --skip-extract
```

**オプション説明:**
| オプション | 説明 |
|---|---|
| `--disc-label` | ディスクラベル（MKV ファイルの親ディレクトリ名） |
| `--title` | コンテンツタイトル |
| `--skip-extract` | Bluray 物理デバイスによる MKV 抽出をスキップ（テスト用 MKV を使用する場合） |
| `--skip-thumbnail` | サムネイル生成ジョブをスキップする場合に指定 |

#### 4.4. 動作確認

パイプライン完了後、以下で確認できます。

```bash
# API からコンテンツ一覧を取得 (ポートフォワード 8000:80 を実行している場合)
curl -s http://localhost:8000/api/v1/contents | jq '.[0].title'

# MongoDB への同期確認（Benthos CDC）
kubectl exec -n pechka mongodb-0 -- \
  mongosh --quiet -u pechka -p changeme --authenticationDatabase admin \
  --eval 'db.getSiblingDB("pechka").contents.countDocuments()'

# Elasticsearch への同期確認
kubectl exec -n pechka elasticsearch-0 -- \
  curl -s 'http://localhost:9200/pechka_contents/_count'
```

正常時はブラウザで [http://localhost:8000](http://localhost:8000) にアクセスするとコンテンツ一覧が表示されます。
