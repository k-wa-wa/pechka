# 台本生成から公開までを担うイメージ。
# 映像の描画は Remotion(Node/React)、音声とタイムラインは Python が受け持つ。
# Node 側の方が要求が重い(Chrome の同梱が要る)ため、Node のイメージを土台にして
# Python を足す形にしている。
FROM node:22-bookworm-slim

# fonts-noto-cjk がないと Chrome が日本語を豆腐(□)で描画する。
# Remotion (Chrome Headless) の起動に必要な依存ライブラリを追加。
RUN apt-get update && apt-get install -y --no-install-recommends \
    python3 \
    python3-pip \
    ffmpeg \
    fonts-noto-cjk \
    libnspr4 \
    libnss3 \
    libatk1.0-0 \
    libatk-bridge2.0-0 \
    libcups2 \
    libdrm2 \
    libdbus-1-3 \
    libxkbcommon0 \
    libxcomposite1 \
    libxdamage1 \
    libxfixes3 \
    libxrandr2 \
    libgbm1 \
    libasound2 \
    libpango-1.0-0 \
    libcairo2 \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# 依存の解決とブラウザの取得はソースより先に済ませ、レイヤーキャッシュを効かせる。
# `browser ensure` でイメージに Chrome を焼き込むのは、Job 起動のたびに
# ダウンロードが走るのを避けるため(docs/406 で字幕バッチの課題として挙がった事象)。
COPY remotion/package.json remotion/package-lock.json ./remotion/
RUN cd remotion && npm ci && npx remotion browser ensure

COPY requirements.txt .
RUN pip3 install --no-cache-dir --break-system-packages -r requirements.txt

COPY main.py .
COPY techfeed/ ./techfeed/
COPY remotion/ ./remotion/
# from-script 経路が既定で参照する台本と、その画像。
COPY examples/ ./examples/

ENTRYPOINT ["python3", "main.py"]
