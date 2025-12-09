# cfmid-network
CFM‑ID (with Go wrapper) and Go API (with HTTP client)

## 功能特性

- ✅ 单分子预测 API (`/predict`)
- ✅ **批量预测 API (`/predict/batch`)** - 支持文件上传，批量处理多个分子
- ✅ **Excel 导出** - 批量预测结果自动导出为 Excel 文件，方便后续分析
- ✅ 健康检查端点 (`/healthz`)

## 快速开始

### 单分子预测

```bash
curl -X POST "http://localhost:5001/predict" \
  -d "smiles=CC(C)C(N)C(=O)O" \
  -d "prob_thresh=0.001"
```

### 批量预测（新功能）

```bash
# 使用示例文件
curl -X POST "http://localhost:5001/predict/batch?prob_thresh=0.001" \
  -F "file=@cfm/example_input.txt" \
  -o results.xlsx
```

详细使用说明请参考 [cfm/BATCH_USAGE.md](cfm/BATCH_USAGE.md)

## 输入文件格式

批量预测支持两种输入格式：

**格式 1：ID + SMILES**
```
M1 CC(C)C(N)C(=O)O
M2 CCO
M3 CC(=O)O
```

**格式 2：仅 SMILES（自动生成 ID）**
```
CC(C)C(N)C(=O)O
CCO
CC(=O)O
```

## 输出格式

批量预测结果以 Excel 文件形式返回，包含以下列：
- ID, SMILES, InChiKey, Formula, PMass
- Energy Level, m/z, Intensity, Fragment ID, Annotation

## 示例文件

- `cfm/example_input.txt` - 10 个示例分子
- `cfm/example_input_large.txt` - 28 个示例分子（更多类别）