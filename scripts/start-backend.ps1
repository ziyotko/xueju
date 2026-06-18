$ErrorActionPreference = "Stop"

Set-Location "D:\WeChatProjects\xueju\backend"
$env:GOCACHE = "D:\WeChatProjects\xueju\.runtime\gocache"
$env:APP_PORT = "8080"

& "D:\Go\bin\go.exe" run ./cmd/api *> "D:\WeChatProjects\xueju\.runtime\backend.script.log"
