param(
    [string]$Mode = "redis"
)

$Token = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiItMjIwMzA3OTk3Nzk4ODA5NiIsImV4cCI6MTc3NTAyMTYzOCwiaWF0IjoxNzc1MDE0NDM4fQ.sGCmOEc3tUNrdDp7NBYJ0YxeO9Pc-E-x_uJmlt8Sg00"
$BaseUrl = "http://localhost:8080"
$Concurrent = 200
$TotalReqs = 2000
$PostIDs = @(999)

$Success = 0
$Failed = 0

$Headers = @{
    "Content-Type" = "application/json"
    "Authorization" = "Bearer $Token"
}

$Start = [System.Diagnostics.Stopwatch]::StartNew()

$runspacePool = [runspacefactory]::CreateRunspacePool(1, $Concurrent)
$runspacePool.Open()
$runspaces = @()

for ($idx = 0; $idx -lt $TotalReqs; $idx++) {
    $ps = [powershell]::Create().AddScript({
        param($idx, $PostIDs, $BaseUrl, $Headers)
        $PostID = $PostIDs[$idx % $PostIDs.Count]
        $Direction = if ($idx % 2 -eq 0) { 1 } else { -1 }
        $Body = @{ post_id = $PostID; direction = $Direction } | ConvertTo-Json

        try {
            $Resp = Invoke-RestMethod -Uri "$BaseUrl/api/v1/vote" -Method Post -Headers $Headers -Body $Body -TimeoutSec 10
            if ($Resp.code -eq 1000) { return "success" } else { return "failed" }
        } catch {
            return "failed"
        }
    }).AddArgument($idx).AddArgument($PostIDs).AddArgument($BaseUrl).AddArgument($Headers)

    $ps.RunspacePool = $runspacePool
    $runspaces += [PSCustomObject]@{
        Pipe = $ps
        Handle = $ps.BeginInvoke()
    }
}

foreach ($r in $runspaces) {
    $result = $r.Pipe.EndInvoke($r.Handle)
    $r.Pipe.Dispose()
    if ($result -eq "success") { $Success++ } else { $Failed++ }
}

$runspacePool.Close()
$runspacePool.Dispose()

$Start.Stop()
$Elapsed = $Start.Elapsed.TotalSeconds
$Total = $Success + $Failed
$QPS = $Total / $Elapsed

Write-Host ""
Write-Host "=== Benchmark Results ($Mode mode) ==="
Write-Host "Total Requests:  $TotalReqs"
Write-Host "Concurrency:     $Concurrent"
Write-Host "Success:         $Success"
Write-Host "Failed:          $Failed"
Write-Host "Duration:        $([math]::Round($Elapsed, 2))s"
Write-Host "QPS:             $([math]::Round($QPS, 2))"
