param (
    [int]$Duration = 20,
    [string]$BaseURL = "http://127.0.0.1:8080"
)

Write-Host ">>> 开始采集 pprof CPU Profile (${Duration}s)..." -ForegroundColor Cyan
Invoke-WebRequest -Uri "$BaseURL/debug/pprof/profile?seconds=$Duration" -OutFile "benchmark/cpu.pprof"
Write-Host ">>> CPU Profile 采集完成: benchmark/cpu.pprof" -ForegroundColor Green

Write-Host ">>> 采集 pprof Heap Profile..." -ForegroundColor Cyan
Invoke-WebRequest -Uri "$BaseURL/debug/pprof/heap" -OutFile "benchmark/heap.pprof"
Write-Host ">>> Heap Profile 采集完成: benchmark/heap.pprof" -ForegroundColor Green
