# vrt-report

frontend の VRT (Playwright) 実行結果を配信するだけの静的コンテナイメージ。

CI (`.github/workflows/frontend.yml` の `push-vrt-report-image` ジョブ) が
`test-vrt` ジョブの `playwright-report` アーティファクトをこのディレクトリの
`playwright-report/` にダウンロードしてから `docker build` する。
frontend アプリ本体のイメージとは無関係で、preview 環境専用。
