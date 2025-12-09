#!/bin/bash
# 批量预测测试脚本（Bash 版本）

INPUT_FILE="${1:-example_input.txt}"
OUTPUT_FILE="${2:-test_results.xlsx}"
PROB_THRESH="${3:-0.001}"

echo "=================================================="
echo "CFM-ID 批量预测测试"
echo "=================================================="
echo "📤 输入文件: $INPUT_FILE"
echo "📊 概率阈值: $PROB_THRESH"
echo "⏳ 正在处理..."

if [ ! -f "$INPUT_FILE" ]; then
    echo "❌ 错误: 输入文件 '$INPUT_FILE' 不存在"
    exit 1
fi

curl -X POST "http://localhost:5001/predict/batch?prob_thresh=$PROB_THRESH" \
  -F "file=@$INPUT_FILE" \
  -o "$OUTPUT_FILE" \
  -w "\nHTTP状态码: %{http_code}\n" \
  -s

if [ $? -eq 0 ] && [ -f "$OUTPUT_FILE" ]; then
    FILE_SIZE=$(stat -f%z "$OUTPUT_FILE" 2>/dev/null || stat -c%s "$OUTPUT_FILE" 2>/dev/null || echo "0")
    FILE_SIZE_KB=$((FILE_SIZE / 1024))
    echo "✅ 预测完成！"
    echo "📁 结果已保存到: $OUTPUT_FILE"
    echo "📏 文件大小: ${FILE_SIZE_KB} KB"
    exit 0
else
    echo "❌ 预测失败"
    exit 1
fi

