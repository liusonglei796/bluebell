# PowerShell: 同时压测和采集 pprof
# 1. 后台启动压测
Start-Process powershell -ArgumentList "-Command", "go run D:/download/project/bluebell/cmd/bench/main.go"
Start-Sleep -Seconds 2

# 2. 采集 CPU profile (10秒)
Write-Host "开始采集 CPU profile..."
Invoke-WebRequest -Uri "http://127.0.0.1:8080/debug/pprof/profile?seconds=10" -OutFile "D:/download/project/bluebell/cpu.prof"

Write-Host "采集完成！"
