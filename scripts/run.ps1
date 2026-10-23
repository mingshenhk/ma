$ErrorActionPreference = 'Stop'
$taskRoot = Split-Path -Parent $PSScriptRoot
Set-Location -LiteralPath $taskRoot
if (-not $env:LISTEN_ADDR) { $env:LISTEN_ADDR = '127.0.0.1:8080' }
if (-not $env:DATA_DIR) { $env:DATA_DIR = Join-Path $taskRoot 'data' }
if (-not $env:STATIC_DIR) { $env:STATIC_DIR = Join-Path $taskRoot 'frontend/dist' }
New-Item -ItemType Directory -Force -Path $env:DATA_DIR | Out-Null
if (-not $env:ADMIN_KEY) {
    $taskKeyPath = Join-Path $env:DATA_DIR 'admin-key.txt'
    if (-not (Test-Path -LiteralPath $taskKeyPath)) {
        $taskRandom = [System.Security.Cryptography.RandomNumberGenerator]::Create()
        $taskBytes = New-Object byte[] 32
        $taskRandom.GetBytes($taskBytes)
        $taskRandom.Dispose()
        [IO.File]::WriteAllText($taskKeyPath, [Convert]::ToBase64String($taskBytes))
    }
    $env:ADMIN_KEY = [IO.File]::ReadAllText($taskKeyPath).Trim()
    Write-Host "管理密钥保存在 $taskKeyPath，请限制该目录访问权限。"
}
$taskBinary = Join-Path $taskRoot 'bin/haojiayuan.exe'
if (-not (Test-Path -LiteralPath $taskBinary)) { throw '缺少 bin/haojiayuan.exe，请先执行 scripts/build.ps1。' }
Write-Host "浏览器访问 http://$($env:LISTEN_ADDR) ，按 Ctrl+C 停止。"
& $taskBinary
if ($LASTEXITCODE -ne 0) { throw "服务退出：$LASTEXITCODE" }
