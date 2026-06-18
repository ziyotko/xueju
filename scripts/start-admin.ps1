$ErrorActionPreference = "Stop"

Set-Location "D:\WeChatProjects\xueju\admin"

& "C:\Program Files\nvm-nodejs\nodejs\npm.cmd" run dev -- --host 0.0.0.0 --port 5173 *> "D:\WeChatProjects\xueju\.runtime\admin.script.log"
