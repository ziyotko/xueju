param([Parameter(Mandatory = $true)][string]$BackupFile)
$ErrorActionPreference = "Stop"

$resolved = (Resolve-Path -LiteralPath $BackupFile).Path
if (-not $env:MYSQL_BACKUP_DSN_HOST -or -not $env:MYSQL_BACKUP_PASSWORD) {
  throw "Set MYSQL_BACKUP_DSN_HOST and MYSQL_BACKUP_PASSWORD before restoring."
}

$confirmation = Read-Host "Restore $resolved into the xueju database? Type RESTORE to continue"
if ($confirmation -ne "RESTORE") { throw "Restore cancelled." }

$env:MYSQL_PWD = $env:MYSQL_BACKUP_PASSWORD
try {
  Get-Content -LiteralPath $resolved -Raw | & mysql --host=$env:MYSQL_BACKUP_DSN_HOST --user=$env:MYSQL_BACKUP_USER xueju
  if ($LASTEXITCODE -ne 0) { throw "mysql restore failed with exit code $LASTEXITCODE" }
} finally {
  Remove-Item Env:MYSQL_PWD -ErrorAction SilentlyContinue
}
Write-Output "Restore completed. Run the API health check and smoke tests before accepting the database."
