param([string]$Go = 'go')
$ErrorActionPreference = 'Stop'
$taskRoot = Split-Path -Parent $PSScriptRoot
Set-Location -LiteralPath (Join-Path $taskRoot 'frontend')
& npm.cmd ci
if ($LASTEXITCODE -ne 0) { throw 'npm ci 失败' }
& npm.cmd run build
if ($LASTEXITCODE -ne 0) { throw '前端构建失败' }
Set-Location -LiteralPath (Join-Path $taskRoot 'backend')
& $Go test ./...
if ($LASTEXITCODE -ne 0) { throw 'Go 测试失败' }
& $Go vet ./...
if ($LASTEXITCODE -ne 0) { throw 'Go vet 失败' }
New-Item -ItemType Directory -Force -Path (Join-Path $taskRoot 'bin') | Out-Null
& $Go build -trimpath -o (Join-Path $taskRoot 'bin/haojiayuan.exe') ./cmd/haojiayuan
if ($LASTEXITCODE -ne 0) { throw '后端构建失败' }
Write-Host '构建完成。执行 scripts/run.ps1 启动。'
