param([string]$BuildId = (Get-Date -Format 'yyyy-MM-dd-HHmmss'))
$ErrorActionPreference = 'Stop'
if ($BuildId -notmatch '^[a-zA-Z0-9_-]+$') { throw 'BuildId must contain only letters, digits, underscores or hyphens.' }
$taskRoot = Split-Path -Parent $PSScriptRoot
$taskStage = Join-Path $taskRoot ('work/package-' + [Guid]::NewGuid().ToString('N'))
$taskPayload = Join-Path $taskStage 'haojiayuan'
$taskRelease = Join-Path $taskRoot 'release'
New-Item -ItemType Directory -Path $taskPayload -Force | Out-Null
New-Item -ItemType Directory -Path $taskRelease -Force | Out-Null
function Copy-PackageFile([string]$RelativePath) {
    $taskSource = Join-Path $taskRoot $RelativePath
    if (-not (Test-Path -LiteralPath $taskSource -PathType Leaf)) { throw "Missing package input: $RelativePath" }
    $taskTarget = Join-Path $taskPayload $RelativePath
    New-Item -ItemType Directory -Path (Split-Path -Parent $taskTarget) -Force | Out-Null
    Copy-Item -LiteralPath $taskSource -Destination $taskTarget
}
foreach ($taskFile in @('.gitattributes','README.md','THIRD_PARTY_NOTICES.md','start.cmd','.env.example','.dockerignore','Dockerfile','compose.yaml','backend/go.mod','backend/go.sum','frontend/package.json','frontend/package-lock.json','frontend/index.html','frontend/tsconfig.json','frontend/vite.config.ts','scripts/build.ps1','scripts/build.sh','scripts/run.ps1','scripts/run.sh','scripts/package.ps1','docker/Caddyfile','bin/haojiayuan.exe','bin/haojiayuan')) {
    Copy-PackageFile $taskFile
}
# Only these project-owned directories and file types are distributable.
$taskDirectories = @{
    'backend/cmd' = @('.go'); 'backend/internal' = @('.go');
    'frontend/src' = @('.ts','.tsx','.css','.svg');
    'frontend/public' = @('.svg','.png','.txt','.webmanifest','.json','.ico');
    'frontend/dist' = @('.html','.js','.css','.svg','.png','.txt','.webmanifest','.json','.ico');
    'docs' = @('.md','.txt'); 'legal' = @('.txt')
}
foreach ($taskDirectory in $taskDirectories.Keys) {
    $taskSourceDirectory = Join-Path $taskRoot $taskDirectory
    if (-not (Test-Path -LiteralPath $taskSourceDirectory -PathType Container)) { throw "Missing directory: $taskDirectory" }
    foreach ($taskItem in Get-ChildItem -LiteralPath $taskSourceDirectory -File -Recurse) {
        if ($taskDirectories[$taskDirectory] -contains $taskItem.Extension.ToLowerInvariant()) {
            Copy-PackageFile ($taskItem.FullName.Substring($taskRoot.Length + 1))
        }
    }
}
$taskManifest = @()
foreach ($taskItem in Get-ChildItem -LiteralPath $taskPayload -File -Recurse | Sort-Object FullName) {
    $taskRelative = $taskItem.FullName.Substring($taskPayload.Length + 1).Replace('\','/')
    $taskManifest += (Get-FileHash -LiteralPath $taskItem.FullName -Algorithm SHA256).Hash.ToLowerInvariant() + '  ' + $taskRelative
}
[IO.File]::WriteAllLines((Join-Path $taskPayload 'SHA256SUMS.txt'), $taskManifest, [Text.UTF8Encoding]::new($false))
$taskArchive = Join-Path $taskRelease ('haojiayuan-six-games-' + $BuildId + '.zip')
if (Test-Path -LiteralPath $taskArchive) { throw "Archive already exists: $taskArchive" }
Compress-Archive -LiteralPath $taskPayload -DestinationPath $taskArchive -CompressionLevel Optimal
$taskArchiveHash = (Get-FileHash -LiteralPath $taskArchive -Algorithm SHA256).Hash.ToLowerInvariant()
[IO.File]::WriteAllText(($taskArchive + '.sha256'), ($taskArchiveHash + '  ' + (Split-Path -Leaf $taskArchive) + "`n"), [Text.UTF8Encoding]::new($false))
Write-Output $taskArchive
Write-Output "SHA256 $taskArchiveHash"
