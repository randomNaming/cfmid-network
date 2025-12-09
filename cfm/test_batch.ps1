# PowerShell 批量预测测试脚本

$url = "http://localhost:5001/predict/batch?prob_thresh=0.001"
$inputFile = "example_input.txt"
$outputFile = "results.xlsx"

Write-Host "=================================================="
Write-Host "CFM-ID 批量预测测试"
Write-Host "=================================================="
Write-Host "📤 输入文件: $inputFile"
Write-Host "📊 概率阈值: 0.001"
Write-Host "⏳ 正在处理..."

if (-not (Test-Path $inputFile)) {
    Write-Host "❌ 错误: 输入文件 '$inputFile' 不存在" -ForegroundColor Red
    exit 1
}

try {
    $fileBytes = [System.IO.File]::ReadAllBytes($inputFile)
    $fileName = [System.IO.Path]::GetFileName($inputFile)
    
    $boundary = [System.Guid]::NewGuid().ToString()
    $LF = "`r`n"
    
    $bodyLines = @(
        "--$boundary",
        "Content-Disposition: form-data; name=`"file`"; filename=`"$fileName`"",
        "Content-Type: text/plain$LF",
        [System.Text.Encoding]::UTF8.GetString($fileBytes),
        "--$boundary--$LF"
    ) -join $LF
    
    $bodyBytes = [System.Text.Encoding]::UTF8.GetBytes($bodyLines)
    
    $response = Invoke-WebRequest -Uri $url -Method Post -Body $bodyBytes -ContentType "multipart/form-data; boundary=$boundary" -ErrorAction Stop
    
    if ($response.StatusCode -eq 200) {
        [System.IO.File]::WriteAllBytes($outputFile, $response.Content)
        $fileSize = (Get-Item $outputFile).Length / 1KB
        Write-Host "✅ 预测完成！" -ForegroundColor Green
        Write-Host "📁 结果已保存到: $outputFile"
        Write-Host "📏 文件大小: $([math]::Round($fileSize, 2)) KB"
    } else {
        Write-Host "❌ 错误: HTTP $($response.StatusCode)" -ForegroundColor Red
        Write-Host "响应内容: $($response.Content)"
    }
} catch {
    Write-Host "❌ 错误: $_" -ForegroundColor Red
    if ($_.Exception.Response) {
        $reader = New-Object System.IO.StreamReader($_.Exception.Response.GetResponseStream())
        $responseBody = $reader.ReadToEnd()
        Write-Host "响应内容: $responseBody" -ForegroundColor Red
    }
    exit 1
}

