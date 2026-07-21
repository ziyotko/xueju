@echo off
setlocal

set GOOS=linux
set GOARCH=amd64
set CGO_ENABLED=0

go build -trimpath -o xueju ./cmd/api
if errorlevel 1 exit /b %errorlevel%

echo Built Linux amd64 binary: %CD%\xueju
