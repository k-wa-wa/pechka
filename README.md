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

### 1. 一括セットアップ

```bash
bash scripts/dev-up.sh
```

このスクリプトは以下を自動で実行します:
- Kind クラスタの作成 (`pechka-cluster`)
- namespace の作成と local overlay の適用 (`k8s/overlays/local` - Argo Workflows 本体と必要な RBAC 設定も自動でデプロイされます)
- API・フロントエンド・全 ETL コンポーネントのイメージビルド
- Kind クラスタへのイメージロードと Deployment 再起動

個別に実行したい場合:

```bash
# クラスタ作成とマニフェスト適用のみ (Argo Workflows 含む)
bash scripts/kind-setup.sh

# イメージビルド・ロード・Deployment 再起動のみ
bash scripts/build-and-load.sh
```

### 2. ポートフォワードとアクセス確認

```bash
# バックグラウンドで Pechka Nginx のポートフォワード起動
nohup kubectl port-forward svc/nginx -n pechka 8000:80 > /tmp/nginx-pf.log 2>&1 &

# バックグラウンドで Argo Workflows UI のポートフォワード起動
nohup kubectl port-forward -n argo svc/argo-server 2746:2746 --address 0.0.0.0 > /tmp/argo-pf.log 2>&1 &

# 疎通確認
curl -s http://localhost:8000/api/v1/contents
```

ブラウザで以下にアクセスできます：
- **Pechka Web UI**: [http://localhost:8000](http://localhost:8000)
- **Argo Workflows Web UI**: [http://localhost:2746](http://localhost:2746)

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

#### 4.3. ETL パイプラインの実行（Argo Workflows）

Argo Workflows を用いたパイプラインの実行は、マニフェストファイル（YAML）の適用、または Argo Web UI から行います。

**マニフェスト（YAML）を使用した実行方法:**

ダミー MKV が配置されている状態で、[test-workflow.yaml](file:///home/nixos/ghq/github.com/k-wa-wa/pechka/etl/test-workflow.yaml) を適用して実行します。

```bash
kubectl create -f etl/test-workflow.yaml
```

**Argo Web UI を使用した実行方法:**

1. Argo Web UI にアクセスし、**Workflow Templates** 画面を開きます。
2. `etl-bluray` テンプレートを選択します。
3. **Submit** ボタンをクリックし、パラメータ入力ダイアログを開きます。
4. 以下の項目を入力し、実行（Submit）します：
   - **Entrypoint**: `manual` （デフォルトは `auto` になっているため、必ず `manual` に切り替えてください）
   - **disc-label**: `TEST_DISC_001`
   - **content-title**: `サンプル動画コンテンツ`

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
