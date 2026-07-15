$ErrorActionPreference = "Stop"

if (-not $env:MYSQL_BACKUP_DSN_HOST -or -not $env:MYSQL_BACKUP_PASSWORD) {
  throw "Set MYSQL_BACKUP_DSN_HOST and MYSQL_BACKUP_PASSWORD before running a backup."
}

$backupDir = Join-Path $PSScriptRoot "..\backups"
New-Item -ItemType Directory -Force -Path $backupDir | Out-Null
$timestamp = Get-Date -Format "yyyyMMdd-HHmmss"
$target = Join-Path $backupDir "xueju-$timestamp.sql"

$env:MYSQL_PWD = $env:MYSQL_BACKUP_PASSWORD
try {
  & mysqldump --host=$env:MYSQL_BACKUP_DSN_HOST --user=$env:MYSQL_BACKUP_USER --single-transaction --routines --events xueju --result-file=$target
  if ($LASTEXITCODE -ne 0) { throw "mysqldump failed with exit code $LASTEXITCODE" }
} finally {
  Remove-Item Env:MYSQL_PWD -ErrorAction SilentlyContinue
}
Write-Output "Backup created: $target"
