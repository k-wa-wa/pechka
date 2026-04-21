FROM ubuntu:22.04

ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update && apt-get install -y \
    ffmpeg \
    postgresql-client \
    wget \
    curl \
    && rm -rf /var/lib/apt/lists/*

# Install MinIO client
RUN wget -q "https://dl.min.io/client/mc/release/linux-amd64/mc" -O /usr/local/bin/mc \
    && chmod +x /usr/local/bin/mc

COPY transcode.sh /usr/local/bin/transcode.sh
COPY generate_master.sh /usr/local/bin/generate_master.sh
RUN chmod +x /usr/local/bin/transcode.sh /usr/local/bin/generate_master.sh

ENTRYPOINT ["/usr/local/bin/transcode.sh"]
