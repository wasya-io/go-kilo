#!/bin/bash

# 変数定義
PROJECT_DIR=$(dirname "$(realpath "$0")")
LOG_PATTERN="log-*.json"

# スクリプトの説明
echo "ログファイル削除ユーティリティ"
echo "パターン: $LOG_PATTERN"
echo "ディレクトリ: $PROJECT_DIR"

# 削除対象ファイル表示
echo "削除対象ファイル:"
find "$PROJECT_DIR" -maxdepth 1 -type f -name "$LOG_PATTERN" | sort

# 削除確認
echo ""
read -p "上記のログファイルを削除しますか？ (y/n): " confirmation

if [[ $confirmation == [yY] || $confirmation == [yY][eE][sS] ]]; then
    # 削除実行
    find "$PROJECT_DIR" -maxdepth 1 -type f -name "$LOG_PATTERN" -delete
    echo "ログファイルを削除しました。"
else
    echo "キャンセルしました。"
fi