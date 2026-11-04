# syntax=docker/dockerfile:1
#
# 上流の AivisSpeech Engine のイメージから、ビルド用の道具を除いたイメージを作る。
#
# 上流の Dockerfile は実行用の段 (runtime-env) にも build-essential・cmake・git を入れている。
# これは arm64 で Python の依存をソースからビルドするためのもので (VOICEVOX/voicevox_engine#770)、実行には要らない。
# trivy-operator でスキャンすると脆弱性が 6,000 件を超え (大半は build-essential → libc6-dev に引きずられて入る linux-libc-dev)、
# 報告が Kubernetes のリソースの保存上限 (2MiB) を超えて保存できない。道具とその依存を消して、報告を上限に収める。
#
# 元にするイメージは、2026-11-04 時点で本番が動かしている上流 :latest と同じ digest で固定する
# (2026-04-22 に上流の master からビルドされたもの)。中身を変えないため、正式版 (1.2.0 など) には上げていない。
# 版を上げるときは、上流の目的の版の digest にこの FROM を差し替え、CI の動作確認が通ることを確かめる。
FROM ghcr.io/aivis-project/aivisspeech-engine@sha256:d53bf4052ecd86e6bc26b7ce8e6836bd3a16c56d1b9ed09ef47955da367052cd

ARG DEBIAN_FRONTEND=noninteractive

# build-essential・cmake・git と、それだけが必要としていたパッケージ
# (gcc・g++・make・libc6-dev・linux-libc-dev・binutils・perl・openssh-client など) を消す。
# curl・ca-certificates・gosu は手で入れたもの (manual) なので残る。gosu は上流の CMD が使う。
#
# 一緒に消えるライブラリのうち、エンジンが使うものは無いことを確かめてある。
# - libgomp1: イメージ内のどの共有ライブラリもリンクしていない。
# - libsqlite3-0・libreadline8: Python 標準の sqlite3・readline モジュールが使えなくなるが、
#   エンジンとその依存はどちらも import しない。
# 消したあとも音声合成が通ることは CI (.github/workflows/aivisspeech.yml) で毎回確かめる。
# 将来エンジンが要るライブラリを消してしまったら、purge の前に `apt-mark manual <パッケージ>` で残す。
RUN set -eux; \
    apt-get purge -y --auto-remove build-essential cmake git; \
    apt-get clean; \
    rm -rf /var/lib/apt/lists/*

# ENTRYPOINT・CMD・WORKDIR・ユーザーは上流のまま引き継ぐ (上書きしない)。
