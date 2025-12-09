# 快速测试指南

## 服务状态

✅ Docker 容器已启动并运行
- 容器名称: `cfmid-wrapper`
- 服务地址: `http://localhost:5001`

## 测试端点

### 1. 健康检查

```bash
curl http://localhost:5001/healthz
```

预期响应: `OK`

### 2. 单分子预测

```bash
curl -X POST "http://localhost:5001/predict" \
  -d "smiles=CC(C)C(N)C(=O)O" \
  -d "prob_thresh=0.001"
```

### 3. 批量预测（新功能）

#### 方式 1: 使用 PowerShell 脚本（推荐，Windows）

```powershell
# 进入 cfm 目录
cd cfm

# 运行测试脚本
.\test_batch_simple.ps1
```

或者直接运行：
```powershell
cd cfm
.\test_batch_simple.ps1
```

#### 方式 2: 使用真正的 curl（如果已安装）

如果您的系统安装了真正的 curl（不是 PowerShell 别名），可以使用：

```bash
# Windows (Git Bash 或 WSL)
curl -X POST "http://localhost:5001/predict/batch?prob_thresh=0.001" \
  -F "file=@cfm/example_input.txt" \
  -o results.xlsx
```

**注意**: PowerShell 中的 `curl` 是 `Invoke-WebRequest` 的别名，不支持 `-F` 参数。请使用上面的 PowerShell 脚本。

#### 方式 3: 使用 PowerShell 直接调用（手动方式）

```powershell
# 进入 cfm 目录
cd cfm

# 使用 .NET HttpClient
$url = "http://localhost:5001/predict/batch?prob_thresh=0.001"
$httpClient = New-Object System.Net.Http.HttpClient
$multipartContent = New-Object System.Net.Http.MultipartFormDataContent
$fileStream = [System.IO.File]::OpenRead("example_input.txt")
$streamContent = New-Object System.Net.Http.StreamContent($fileStream)
$streamContent.Headers.ContentType = New-Object System.Net.Http.Headers.MediaTypeHeaderValue("text/plain")
$multipartContent.Add($streamContent, "file", "example_input.txt")
$response = $httpClient.PostAsync($url, $multipartContent).Result
$responseBytes = $response.Content.ReadAsByteArrayAsync().Result
[System.IO.File]::WriteAllBytes("results.xlsx", $responseBytes)
$fileStream.Close()
$httpClient.Dispose()
Write-Host "✅ 完成！结果已保存到 results.xlsx"
```

#### 方式 4: 使用 Python

```python
import requests

url = "http://localhost:5001/predict/batch"
params = {"prob_thresh": "0.001"}

with open("example_input.txt", "rb") as f:
    files = {"file": f}
    response = requests.post(url, params=params, files=files)
    
    if response.status_code == 200:
        with open("results.xlsx", "wb") as out:
            out.write(response.content)
        print("✅ 预测完成，结果已保存到 results.xlsx")
    else:
        print(f"❌ 错误: {response.status_code} - {response.text}")
```

#### 使用 Postman 或类似工具

1. 方法: `POST`
2. URL: `http://localhost:5001/predict/batch?prob_thresh=0.001`
3. Body 类型: `form-data`
4. 添加字段:
   - Key: `file` (类型: File)
   - Value: 选择 `example_input.txt` 文件
5. 发送请求
6. 保存响应为 `.xlsx` 文件

## 容器管理命令

```bash
# 查看容器状态
docker ps --filter name=cfmid-wrapper

# 查看日志
docker logs cfmid-wrapper

# 停止容器
docker stop cfmid-wrapper

# 启动容器
docker start cfmid-wrapper

# 重启容器
docker restart cfmid-wrapper

# 删除容器
docker rm -f cfmid-wrapper
```

## 示例文件

- `example_input.txt` - 10 个示例分子
- `example_input_large.txt` - 28 个示例分子（更多类别）

## 输入文件格式

支持两种格式：

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

## 输出说明

批量预测返回 Excel 文件，包含以下列：
- ID, SMILES, InChiKey, Formula, PMass
- Energy Level, m/z, Intensity, Fragment ID, Annotation

每个分子的每个碎片占一行，方便后续分析。

