# NFS → MinIO データ移行手順

## 概要

nuage-cluster 上の NFS (`/srv/nfs/hls`) に存在する既存 HLS コンテンツを MinIO の `pechka` バケットへ移行する一時 Job。

## 前提条件

- Sub-Task #22（nuage-cluster デプロイ）が完了していること
- `pechka` namespace に MinIO がデプロイ済みであること
- `pechka-minio` Secret（access-key / secret-key）と `pechka-config` ConfigMap が存在すること
- nuage-cluster の NFS サーバーアドレスが確認済みであること

## 実行手順

### 1. NFS サーバーアドレスの確認

```bash
# nuage-cluster 上で NFS サーバーの IP を確認する
kubectl get nodes -o wide  # または NFS サーバーのホスト名/IP を確認
export NFS_SERVER="<NFS_SERVER_IP_OR_HOSTNAME>"
```

### 2. Job マニフェストの `${NFS_SERVER}` を実際の値に置き換える

```bash
sed "s/\${NFS_SERVER}/$NFS_SERVER/g" nfs-to-minio-job.yaml > nfs-to-minio-job-resolved.yaml
```

### 3. Job の適用

```bash
kubectl apply -f nfs-to-minio-job-resolved.yaml
```

または kustomize 経由で適用する場合:

```bash
# マニフェスト内の ${NFS_SERVER} を先に置き換えてから適用すること
kubectl kustomize k8s/overlays/prod/migration | \
  sed "s/\${NFS_SERVER}/$NFS_SERVER/g" | \
  kubectl apply -f -
```

### 4. Job の進行状況を確認

```bash
kubectl -n pechka get job nfs-to-minio-migration
kubectl -n pechka logs -l app=nfs-to-minio-migration --follow
```

### 5. コピー完了の確認

Job のログに `Migration finished successfully.` が出力されれば完了。

#### MinIO への接続確認

```bash
# mc を使って pechka バケットのコンテンツを確認
kubectl -n pechka run mc-debug --rm -it --image=quay.io/minio/mc:latest -- sh
# mc alias set minio http://minio:9000 <ACCESS_KEY> <SECRET_KEY>
# mc ls minio/pechka/resources/hls/
# mc ls minio/pechka/thumbnails/
```

### 6. 移行後の動作確認

- フロントエンド `https://pechka.nuage.cluster.wpc/` にアクセスしてコンテンツ一覧が表示されることを確認
- HLS 動画が nginx → MinIO プロキシ経由で再生できることを確認
- サムネイルが正しく表示されることを確認

### 7. Job のクリーンアップ

確認完了後、一時ファイルを削除する:

```bash
kubectl -n pechka delete job nfs-to-minio-migration
rm -f nfs-to-minio-job-resolved.yaml
```

## コピー対象パス

| NFS パス | MinIO パス |
|----------|-----------|
| `/srv/nfs/hls/` | `pechka/resources/hls/` |
| `/srv/nfs/thumbnails/` | `pechka/thumbnails/` |

## トラブルシューティング

- **MinIO への接続エラー**: `pechka-minio` Secret の access-key / secret-key が正しいか確認する
- **NFS マウントエラー**: `${NFS_SERVER}` のアドレスが正しく設定されているか、NFS サーバーが起動しているか確認する
- **権限エラー**: NFS の export 設定でクラスターのノードからの読み取りが許可されているか確認する
