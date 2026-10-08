Write-Host ">>> [方案 2: 应用层轻量聚合方案] 启动 k6 压测任务..." -ForegroundColor Cyan
$k6Job = Start-Process -FilePath "k6.exe" -ArgumentList "run benchmark/k6_post_detail_preload.js" -PassThru -NoNewWindow

Write-Host ">>> 等待 45 秒使并发攀升至 100~150 VUs 峰值阶段..." -ForegroundColor Yellow
Start-Sleep -Seconds 45

Write-Host ">>> 开始在峰值期间采集方案 2 的 pprof 剖析数据..." -ForegroundColor Cyan
try {
    # 1. 抓取 20 秒 CPU 采样
    Write-Host ">>> [1/4] 采集 CPU profile (20s 采样)..." -ForegroundColor Yellow
    Invoke-WebRequest -Uri "http://127.0.0.1:8080/debug/pprof/profile?seconds=20" -OutFile "benchmark/cpu_agg.pprof" -TimeoutSec 35
    Write-Host ">>> CPU Profile 采集成功: benchmark/cpu_agg.pprof" -ForegroundColor Green

    # 2. 采集 Mutex profile
    Write-Host ">>> [2/4] 采集 Mutex profile..." -ForegroundColor Yellow
    Invoke-WebRequest -Uri "http://127.0.0.1:8080/debug/pprof/mutex" -OutFile "benchmark/mutex_agg.pprof" -TimeoutSec 15
    Write-Host ">>> Mutex Profile 采集成功: benchmark/mutex_agg.pprof" -ForegroundColor Green

    # 3. 采集 Block profile
    Write-Host ">>> [3/4] 采集 Block profile..." -ForegroundColor Yellow
    Invoke-WebRequest -Uri "http://127.0.0.1:8080/debug/pprof/block" -OutFile "benchmark/block_agg.pprof" -TimeoutSec 15
    Write-Host ">>> Block Profile 采集成功: benchmark/block_agg.pprof" -ForegroundColor Green

    # 4. 采集 Heap profile
    Write-Host ">>> [4/4] 采集 Heap profile..." -ForegroundColor Yellow
    Invoke-WebRequest -Uri "http://127.0.0.1:8080/debug/pprof/heap" -OutFile "benchmark/heap_agg.pprof" -TimeoutSec 15
    Write-Host ">>> Heap Profile 采集成功: benchmark/heap_agg.pprof" -ForegroundColor Green
} catch {
    Write-Host ">>> 抓取 pprof 出现异常: $_" -ForegroundColor Red
}

Write-Host ">>> 等待 k6 压测结束..." -ForegroundColor Yellow
$k6Job.WaitForExit()
Write-Host ">>> k6 压测完成，退出码: $($k6Job.ExitCode)" -ForegroundColor Green

if (Test-Path "benchmark/preload_summary.json") {
    Copy-Item -Path "benchmark/preload_summary.json" -Destination "benchmark/agg_summary.json" -Force
    Write-Host ">>> 压测报告已归档至 benchmark/agg_summary.json" -ForegroundColor Green
}
