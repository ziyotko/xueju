$ErrorActionPreference = "Stop"

$root = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
$backend = Join-Path $root "backend"
$runtime = Join-Path $root ".runtime"
New-Item -ItemType Directory -Force -Path $runtime | Out-Null

if (-not $env:GOCACHE) { $env:GOCACHE = Join-Path $runtime "gocache" }
if (-not $env:APP_PORT) { $env:APP_PORT = "8080" }

Push-Location $backend
try {
  & go run ./cmd/api 2>&1 | Tee-Object -FilePath (Join-Path $runtime "backend.script.log")
  if ($LASTEXITCODE -ne 0) { throw "backend exited with code $LASTEXITCODE" }
} finally {
  Pop-Location
}
