#!/bin/bash

# スクリプトの目的: ソースコードファイルのパスと内容を1つのファイルにまとめる

# 出力ファイル名を定義
output_file="source_code_for_gemini.txt"

# 含めるファイルの拡張子を定義
extensions=(
    ".go"
    "Dockerfile"
    "go.mod"
    ".toml"
    ".env"
    "Makefile"
    "*.proto"
    "LICENSE"
)

# 現在のディレクトリを取得
current_dir=$(pwd)

# 出力ファイルをクリアする
> "$output_file"

# 指定された拡張子のファイルを見つけ、「internal」と「tmp_scrape」ディレクトリを除外する
find "$current_dir" -type f \( -name "*${extensions[0]}" -o -name "${extensions[1]}" -o -name "*${extensions[2]}" -o -name "*${extensions[3]}" -o -name "*${extensions[4]}" -o -name "*${extensions[5]}" -o -name "*${extensions[6]}" -o -name "*${extensions[7]}" \) ! -path "*/internal/*" ! -path "*/tmp_scrape/*" -print | while read -r file; do
    # ファイルパスを出力
    echo "$file" >> "$output_file"
    echo "" >> "$output_file"  # 空行を追加

    # ファイル内容を出力
    cat "$file" >> "$output_file"
    echo "" >> "$output_file"  # 空行を追加
done

# 標準出力に結果を表示
echo "Source code saved to $output_file."
