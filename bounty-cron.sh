#!/bin/bash

# 5分おきにbounty.shを実行するデモ用（nohup用）スクリプト

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
BOUNTY_SCRIPT="$SCRIPT_DIR/bounty.sh"

echo "=== bounty-cron.sh を開始します ==="
while true
do
    echo "$(date): bounty.sh の実行を開始します"
    
    bash "$BOUNTY_SCRIPT"
    
    echo "$(date): 次の実行まで5分間待機します..."
    sleep 300
done
