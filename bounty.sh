#!/bin/bash
set -euo pipefail

TIMESTAMP_FILE="$(dirname "$0")/.last_run_timestamp"
NOW=$(date -u +%Y-%m-%dT%H:%M:%SZ)
NEEDS_UPDATE=false

if [ ! -f "$TIMESTAMP_FILE" ]; then
    NEEDS_UPDATE=true
else
    LAST_RUN=$(cat "$TIMESTAMP_FILE")
    # 前回実行時から更新されたIssueやPRの数を取得
    UPDATED_COUNT=$(gh search issues --repo k-wa-wa/pechka --include-prs --updated ">=$LAST_RUN" --json number --jq 'length' 2>/dev/null || echo "0")
    
    if [ "$UPDATED_COUNT" -gt 0 ]; then
        NEEDS_UPDATE=true
    fi
fi

if [ "$NEEDS_UPDATE" = true ]; then
    echo "$(date): 更新が検出されました。処理を開始します"

    ~/.local/bin/claude --dangerously-skip-permissions -p "
    CLAUDE.md の自動開発タスクの手順に従って、GitHub 上のタスクを処理してください。
    タスクがないと判断した場合は、すぐに処理を終了してください。
    "
    # npx @google/gemini-cli --yolo "CLAUDE.md の自動開発タスクの手順に従って、GitHub 上のタスクを処理してください。claudeが途中で辞めたタスクがあるかもしれないので注意して"

    # 終了後にタイムスタンプを更新
    echo "$NOW" > "$TIMESTAMP_FILE"
    echo "$(date): 処理を終了し、タイムスタンプを更新しました"
else
    # 更新がない場合はそのまま終了
    echo "$(date): 変更はありません"
fi
