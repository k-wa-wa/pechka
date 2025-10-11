FROM python:3.12

RUN apt -y update && \
  apt -y install libgl1

# clang-18インストール（arm対策）
RUN apt install -y lsb-release wget software-properties-common gnupg && \
  wget -qO- https://apt.llvm.org/llvm.sh | bash -s -- 18 && \
  update-alternatives \
    --install /usr/bin/clang clang /usr/bin/clang-18 1 \
    --slave /usr/bin/clang++ clang++ /usr/bin/clang++-18 \
    --slave /usr/bin/lldb lldb /usr/bin/lldb-18 \
    --slave /usr/bin/lld lld /usr/bin/lld-18

WORKDIR /app
RUN wget https://github.com/exo-explore/exo/archive/d4cc2cf13d9e46e68b25d0c94bfcda6327afacd0.zip && \
  unzip d4cc2cf13d9e46e68b25d0c94bfcda6327afacd0.zip

WORKDIR /app/exo-d4cc2cf13d9e46e68b25d0c94bfcda6327afacd0

COPY patch_llama.py exo/inference/tinygrad/models/llama.py
RUN pip install llvmlite
RUN pip install -e .

# ENV DEBUG=2
CMD [ "exo" ]