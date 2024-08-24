FROM python:3.12.5-slim-bookworm

COPY --from=ghcr.io/astral-sh/uv:latest /uv /bin/uv

WORKDIR /app

COPY pyproject.toml uv.lock requirements.txt ./
RUN uv venv /app/.venv
RUN uv pip install -r requirements.txt
ENV PATH="/app/.venv/bin:$PATH"

COPY main.py ./

EXPOSE 8000
CMD [ "uvicorn", "main:app",  "--host", "0.0.0.0", "--port", "8000" ]
