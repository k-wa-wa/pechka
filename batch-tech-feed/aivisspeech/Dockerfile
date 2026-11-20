# syntax=docker/dockerfile:1
#
# 上流の AivisSpeech Engine (正式版 1.2.0) のイメージから、ビルド用の道具を除き、
# OS と Python の既知脆弱性を抑えた実行用イメージを作る。
#
# 上流の Dockerfile は実行用の段 (runtime-env) にも build-essential・cmake・git を入れている。
# これは arm64 で Python の依存をソースからビルドするためのもので (VOICEVOX/voicevox_engine#770)、実行には要らない。
# trivy-operator でスキャンすると脆弱性が 6,000 件前後になり (大半は build-essential → libc6-dev に引きずられて入る linux-libc-dev)、
# 報告が Kubernetes のリソースの保存上限 (2MiB) を超えて保存できない。道具とその依存を消して、報告を上限に収める。
#
# 元にするイメージは、開発版 (1.3.0-dev) ではなく正式版最新の 1.2.0 (2026-04-30) のマルチアーキテクチャ index digest で固定する。
# 意図せず開発版や未検証のビルドを取り込まないため、:1.2.0 や :latest のタグ参照ではなく digest で指定する。
# 版を上げるときは、上流の正式版リリースの digest にこの FROM を差し替え、CI の動作確認が通ることを確かめる。
FROM ghcr.io/aivis-project/aivisspeech-engine@sha256:5799c94c3594501be80eeda62106f15439a858c1630d25d1385576581433b192

ARG DEBIAN_FRONTEND=noninteractive

# 1. OS のビルド用の道具を先に消してから、残る実行時パッケージへセキュリティ更新を当てる。
#    - build-essential・cmake・git と、それだけが必要としていたパッケージ
#      (gcc・g++・make・libc6-dev・linux-libc-dev・binutils・perl・openssh-client など) を消す。
#      curl・ca-certificates・gosu は上流が明示的に入れたもの (manual) なので残る。gosu は上流の CMD が使う。
#      一緒に消えるライブラリのうち、エンジンが使うものは無いことを CI の音声合成テストで毎回確かめる。
#      将来エンジンが要るライブラリを消してしまったら、purge の前に `apt-mark manual <パッケージ>` で残す。
#    - その後 `apt-get upgrade -y --no-install-recommends` で、CVE-2026-111497 (libbrotli1) を含む
#      Ubuntu 22.04 のセキュリティ更新をすべて当てる。
# 2. Python の依存は、1.2.0 のエンジン制約・音声合成コアを壊さない範囲で trivy が修正版ありと出すものを上げる。
#    - 上流 1.2.0 は uv sync を使っており .venv 内に pip が入っていないため、/opt/python/bin/uv で更新と整合確認を行う。
#    - starlette==1.10.0: CVE-2026-48710 (CISA KEV) 対策。fastapi 0.136.0 (>=0.46.0) と互換。
#    - anyio・certifi・filelock・fsspec・idna・python-multipart・urllib3: 周辺ランタイム/HTTP 依存の CVE を解消する。
#    - nltk (<=3.8.1)・transformers (<4.50.0) は Style-Bert-VITS2 側の制約で据え置き、
#      setuptools (<82) は pyworld の pkg_resources 依存のため据え置く。onnx (1.21.0) は推論モデル中核のため据え置く。
# 3. 上流がビルド時に残した /home/user/.cache/uv と、実行に不要なビルド道具 (/opt/python 側の uv・setuptools) を消す。
#    .cache/uv を消さないと古い .dist-info が残り、trivy が旧版の脆弱性を検出し続ける。
#    /opt/python 側の pip は最新版へ上げ、CI の `pip --python .venv/bin/python check` に残す。
RUN set -eux; \
    apt-get purge -y --auto-remove build-essential cmake git; \
    apt-get update; \
    apt-get upgrade -y --no-install-recommends; \
    apt-get clean; \
    rm -rf /var/lib/apt/lists/*; \
    gosu user /opt/python/bin/uv pip install --python /opt/aivisspeech-engine/.venv/bin/python --no-cache --upgrade \
      'starlette==1.10.0' \
      'anyio==4.16.0' \
      'certifi==2026.10.23' \
      'filelock==4.2.12' \
      'fsspec==2026.9.0' \
      'idna==3.20' \
      'python-multipart==0.0.34' \
      'urllib3==2.8.0'; \
    gosu user /opt/python/bin/uv pip check --python /opt/aivisspeech-engine/.venv/bin/python; \
    rm -rf /home/user/.cache/uv; \
    /opt/python/bin/pip3 install --no-cache-dir --upgrade 'pip==26.3'; \
    /opt/python/bin/pip3 uninstall -y uv setuptools

# ENTRYPOINT・CMD・WORKDIR・ユーザーは上流のまま引き継ぐ (上書きしない)。
