$ErrorActionPreference = "Stop"

$root = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
$admin = Join-Path $root "admin"
$runtime = Join-Path $root ".runtime"
New-Item -ItemType Directory -Force -Path $runtime | Out-Null

Push-Location $admin
try {
  & npm run dev -- --host 0.0.0.0 --port 5173 2>&1 | Tee-Object -FilePath (Join-Path $runtime "admin.script.log")
  if ($LASTEXITCODE -ne 0) { throw "admin exited with code $LASTEXITCODE" }
} finally {
  Pop-Location
}
