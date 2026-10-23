# エージェント開発・検証ガイドライン (pechka)

すべての開発者およびエージェントは、以下のルールと検証手順を厳格に遵守すること。

---

## 1. 検証・品質チェック手順（MUST）

コード変更やテスト追加を行った後は、変更したモジュールに応じて**以下のコマンドを実行し成功を確認すること**。

### Frontend (`frontend/`)
```bash
cd frontend && npm run check
# 内訳: typecheck, lint (Biome), check:stories, check:tests, test
```

### Go モジュール群 (`api/`, `batch/etl/`, `batch/refresh-latest-playlist/`, `batch-tech-feed/*`)
```bash
# フォーマットと構文チェック
gofmt -s -w .
go vet ./...

# テスト実行
go test -race ./...
```

### Python バッチ (`batch/generate-thumbnail/`, `batch/subtitle/`)
```bash
python3 -m unittest discover -s batch/generate-thumbnail -p "test_*.py"
python3 -m unittest discover -s batch/subtitle -p "test_*.py"
```

### リポジトリ全体のテストファイル存在チェック
新規実装やファイル追加を行った場合、対応するテストファイルが漏れていないか必ず確認すること。
```bash
./scripts/check-required-tests.sh
```

### K8s マニフェスト変更時
```bash
./k8s/lint.sh
```

---

## 2. 操作・開発ルール

- **Git 操作の禁止**: ユーザーからの特別な指示がない限り、コミット・プッシュ・ブランチ作成などの Git 操作（`git` コマンド）は実行しないこと。
- **テストの必須化**: 新しいコンポーネントや関数を追加する際は、必ず対応する `*.test.tsx`, `*_test.go`, `test_*.py` をセットで作成すること。
- **テストサーバのバインド**: 単体テスト内で起動する HTTP サーバは必ず `127.0.0.1` / `localhost` にバインドし、`0.0.0.0` を使用しないこと。本番シークレットをハードコードしないこと。
- **Storybook ストーリーの追加**: `frontend/components/` に新しいコンポーネントを追加した際は、必ず `*.stories.tsx` も作成すること（`npm run check:stories` で検証）。
