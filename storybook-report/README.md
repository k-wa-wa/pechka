# storybook-report

frontend の Storybook ビルド成果物を配信するだけの静的コンテナイメージ。

CI (`.github/workflows/frontend.yml` の `push-storybook-report-image` ジョブ) が
`test-frontend` ジョブの `storybook-static` アーティファクトをこのディレクトリの
`storybook-static/` にダウンロードしてから `docker build` する。
frontend アプリ本体のイメージとは無関係で、preview 環境専用。
