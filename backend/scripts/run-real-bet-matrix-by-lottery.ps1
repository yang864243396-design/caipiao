# 按彩种分批跑 real-bet-matrix
# 用法:
#   .\scripts\run-real-bet-matrix-by-lottery.ps1 -SmallFirst
#   .\scripts\run-real-bet-matrix-by-lottery.ps1 -Status
#   .\scripts\run-real-bet-matrix-by-lottery.ps1 -RetryFailed
#   .\scripts\run-real-bet-matrix-by-lottery.ps1 -Only tron_ffc_1m,eth_ffc_1m
#   .\scripts\run-real-bet-matrix-by-lottery.ps1 -SyncSummary
param(
    [string]$OutDir = "data/real-bet-matrix/by-lottery",
    [string]$Summary = "data/real-bet-matrix/batch-summary.jsonl",
    [string]$MergedReport = "data/real-bet-matrix/all-results.jsonl",
    [string]$Delay = "2s",
    [double]$Unit = 2,
    [string]$Account = "",
    [string[]]$Only = @(),
    [string[]]$SkipLotteries = @(),
    [switch]$SmallFirst,
    [switch]$StopOnFail,
    [switch]$Status,
    [switch]$RetryFailed,
    [switch]$MergeOnly,
    [switch]$SyncSummary,
    [switch]$ReportOnly,
    [switch]$FreshSummary,
    [switch]$AcceptSkipOnly,
    [string]$ReportOut = "../docs/real-bet-matrix-test-report.md"
)

$ErrorActionPreference = "Continue"
$BackendRoot = Split-Path $PSScriptRoot -Parent
Set-Location $BackendRoot

function Get-LotteryMatrixItems {
    $dry = & go run ./cmd/real-bet-matrix -dry-run 2>&1 | Out-String
    $items = [System.Collections.Generic.List[object]]::new()
    foreach ($line in ($dry -split "`n")) {
        if ($line -match '^\s+(\S+):\s+(\d+)\s+plays') {
            $items.Add([pscustomobject]@{ Code = $Matches[1]; Plays = [int]$Matches[2] })
        }
    }
    if ($items.Count -eq 0) {
        throw "无法解析 dry-run 彩种列表"
    }
    return $items
}

function Convert-ToCount {
    param($Value)
    if ($null -eq $Value) { return 0 }
    if ($Value -is [array]) {
        if ($Value.Count -eq 0) { return 0 }
        return [int]$Value[0]
    }
    return [int]$Value
}

function Get-LatestSummaryMap {
    param($Rows)
    $latest = @{}
    foreach ($row in $Rows) {
        $latest[$row.lotteryCode] = [pscustomobject]@{
            lotteryCode = $row.lotteryCode
            plays       = Convert-ToCount $row.plays
            ok          = Convert-ToCount $row.ok
            skip        = Convert-ToCount $row.skip
            fail        = Convert-ToCount $row.fail
            exitCode    = Convert-ToCount $row.exitCode
            state       = $row.state
            failSample  = $row.failSample
            outFile     = $row.outFile
            startedAt   = $row.startedAt
            finishedAt  = $row.finishedAt
        }
    }
    return $latest
}

function Get-ResultCounts {
    param([string]$Path)
    $okCount = 0; $skipCount = 0; $failCount = 0
    $failSample = ""
    if (-not (Test-Path $Path)) {
        return @{ ok = 0; skip = 0; fail = 0; failSample = "" }
    }
    foreach ($line in Get-Content $Path -Encoding UTF8) {
        if ([string]::IsNullOrWhiteSpace($line)) { continue }
        if ($line -match '"status"\s*:\s*"ok"') { $okCount++ }
        elseif ($line -match '"status"\s*:\s*"skip"') { $skipCount++ }
        elseif ($line -match '"status"\s*:\s*"fail"') {
            $failCount++
            if (-not $failSample -and $line -match '"error"\s*:\s*"([^"]*)"') {
                $failSample = $Matches[1]
            }
        }
    }
    return @{ ok = $okCount; skip = $skipCount; fail = $failCount; failSample = $failSample }
}

function Read-SummaryRows {
    param([string]$Path)
    if (-not (Test-Path $Path)) { return @() }
    $rows = @()
    foreach ($line in Get-Content $Path -Encoding UTF8) {
        if ([string]::IsNullOrWhiteSpace($line)) { continue }
        try {
            $rows += ($line | ConvertFrom-Json)
        } catch {
            Write-Warning "skip invalid summary line: $line"
        }
    }
    return $rows
}

function Get-LotteryState {
    param($Row, [int]$Plays, [switch]$AcceptSkipOnly)
    if ($null -eq $Row) { return "pending" }
    $fail = Convert-ToCount $Row.fail
    $ok = Convert-ToCount $Row.ok
    $skip = Convert-ToCount $Row.skip
    $exit = Convert-ToCount $Row.exitCode
    if ($fail -gt 0 -or $exit -ne 0) { return "failed" }
    if ($ok -gt 0) { return "done" }
    if ($AcceptSkipOnly -and ($ok + $skip) -ge $Plays) { return "skip-only" }
    return "incomplete"
}

function Show-BatchStatus {
    param($Items, $SummaryRows)
    $latest = Get-LatestSummaryMap -Rows $SummaryRows

    $totalPlays = 0; $totalOk = 0; $totalSkip = 0; $totalFail = 0
    $done = 0; $failed = 0; $pending = 0; $skipOnly = 0

    Write-Host ("{0,-22} {1,5} {2,4} {3,4} {4,4} {5,4} {6}" -f "lottery", "plays", "ok", "skip", "fail", "exit", "state")
    Write-Host ("-" * 72)
    foreach ($lot in ($Items | Sort-Object Code)) {
        $row = $latest[$lot.Code]
        $outPath = Join-Path $OutDir ("{0}.jsonl" -f $lot.Code)
        if (Test-Path $outPath) {
            $counts = Get-ResultCounts -Path $outPath
            if (($counts.ok + $counts.skip + $counts.fail) -gt 0) {
                $row = [pscustomobject]@{
                    lotteryCode = $lot.Code
                    ok          = $counts.ok
                    skip        = $counts.skip
                    fail        = $counts.fail
                    exitCode    = if ($latest[$lot.Code]) { Convert-ToCount $latest[$lot.Code].exitCode } else { 0 }
                }
            }
        }
        $state = Get-LotteryState -Row $row -Plays $lot.Plays -AcceptSkipOnly:$AcceptSkipOnly
        $ok = if ($row) { Convert-ToCount $row.ok } else { 0 }
        $skip = if ($row) { Convert-ToCount $row.skip } else { 0 }
        $fail = if ($row) { Convert-ToCount $row.fail } else { 0 }
        $exit = if ($row) { Convert-ToCount $row.exitCode } else { "-" }
        Write-Host ("{0,-22} {1,5} {2,4} {3,4} {4,4} {5,4} {6}" -f $lot.Code, $lot.Plays, $ok, $skip, $fail, $exit, $state)
        $totalPlays += $lot.Plays; $totalOk += $ok; $totalSkip += $skip; $totalFail += $fail
        switch ($state) {
            "done" { $done++ }
            "failed" { $failed++ }
            "skip-only" { $skipOnly++ }
            default { $pending++ }
        }
    }
    Write-Host ("-" * 72)
    Write-Host "lotteries=$($Items.Count) done=$done skip-only=$skipOnly failed=$failed pending=$pending"
    Write-Host "rows ok=$totalOk skip=$totalSkip fail=$totalFail / $totalPlays"
    Write-Host "summary=$Summary"
    Write-Host "outDir=$OutDir"
}

function Merge-LotteryReports {
    param([string]$OutDir, [string]$Target)
    New-Item -ItemType Directory -Force -Path (Split-Path $Target -Parent) | Out-Null
    if (Test-Path $Target) { Remove-Item $Target -Force }
    $files = Get-ChildItem -Path $OutDir -Filter "*.jsonl" -File | Sort-Object Name
    foreach ($f in $files) {
        Get-Content $f.FullName -Encoding UTF8 | Add-Content -Path $Target -Encoding UTF8
    }
    Write-Host "merged $($files.Count) files -> $Target"
}

function Get-JsonlTimeRange {
    param([string]$Path)
    $startedAt = ""
    $finishedAt = ""
    if (-not (Test-Path $Path)) {
        return @{ startedAt = $startedAt; finishedAt = $finishedAt }
    }
    foreach ($line in Get-Content $Path -Encoding UTF8) {
        if ([string]::IsNullOrWhiteSpace($line)) { continue }
        try {
            $row = $line | ConvertFrom-Json
        } catch {
            continue
        }
        if ($row.at) {
            if (-not $startedAt) { $startedAt = $row.at }
            $finishedAt = $row.at
        }
    }
    return @{ startedAt = $startedAt; finishedAt = $finishedAt }
}

function Sync-SummaryFromResults {
    param($Items, [string]$SummaryPath, [string]$OutDir)
    if (Test-Path $SummaryPath) {
        $bak = "$SummaryPath.bak.$(Get-Date -Format 'yyyyMMdd-HHmmss')"
        Copy-Item $SummaryPath $bak
        Write-Host "archived summary -> $bak"
    }
    $lines = [System.Collections.Generic.List[string]]::new()
    foreach ($lot in ($Items | Sort-Object Code)) {
        $outFile = Join-Path $OutDir ("{0}.jsonl" -f $lot.Code)
        if (-not (Test-Path $outFile)) {
            continue
        }
        $counts = Get-ResultCounts -Path $outFile
        if (($counts.ok + $counts.skip + $counts.fail) -le 0) {
            continue
        }
        $times = Get-JsonlTimeRange -Path $outFile
        $exitCode = if ($counts.fail -gt 0) { 1 } else { 0 }
        $rowSnapshot = [pscustomobject]@{
            ok       = $counts.ok
            skip     = $counts.skip
            fail     = $counts.fail
            exitCode = $exitCode
        }
        $state = Get-LotteryState -Row $rowSnapshot -Plays $lot.Plays -AcceptSkipOnly:$AcceptSkipOnly
        $summaryObj = [ordered]@{
            lotteryCode = $lot.Code
            plays       = $lot.Plays
            ok          = $counts.ok
            skip        = $counts.skip
            fail        = $counts.fail
            exitCode    = $exitCode
            state       = $state
            failSample  = $counts.failSample
            outFile     = $outFile
            startedAt   = $times.startedAt
            finishedAt  = $times.finishedAt
        }
        $lines.Add(($summaryObj | ConvertTo-Json -Compress))
    }
    Set-Content -Path $SummaryPath -Value $lines -Encoding UTF8
    Write-Host "synced summary -> $SummaryPath ($($lines.Count) lotteries)"
}

New-Item -ItemType Directory -Force -Path $OutDir | Out-Null
New-Item -ItemType Directory -Force -Path (Split-Path $Summary -Parent) | Out-Null

$items = Get-LotteryMatrixItems
if ($SkipLotteries.Count -eq 0) {
    $SkipLotteries = @()
}
$summaryRows = Read-SummaryRows -Path $Summary

if ($Status) {
    Show-BatchStatus -Items $items -SummaryRows $summaryRows
    exit 0
}

if ($MergeOnly) {
    Merge-LotteryReports -OutDir $OutDir -Target $MergedReport
    exit 0
}

if ($SyncSummary) {
    Sync-SummaryFromResults -Items $items -SummaryPath $Summary -OutDir $OutDir
    Merge-LotteryReports -OutDir $OutDir -Target $MergedReport
    Show-BatchStatus -Items $items -SummaryRows (Read-SummaryRows -Path $Summary)
    exit 0
}

if ($ReportOnly) {
    & go run ./cmd/real-bet-matrix-report -out $ReportOut -account $Account -unit $Unit -note "manual report"
    exit 0
}

if ($FreshSummary -and (Test-Path $Summary)) {
    $bak = "$Summary.bak.$(Get-Date -Format 'yyyyMMdd-HHmmss')"
    Move-Item $Summary $bak
    Write-Host "archived summary -> $bak"
    $summaryRows = @()
}
if ($FreshSummary -and (Test-Path $OutDir)) {
    Remove-Item -Path (Join-Path $OutDir "*.jsonl") -Force -ErrorAction SilentlyContinue
    Write-Host "cleared $OutDir"
}
if ($FreshSummary -and (Test-Path $MergedReport)) {
    Remove-Item $MergedReport -Force -ErrorAction SilentlyContinue
}

$latestSummary = Get-LatestSummaryMap -Rows $summaryRows

$ordered = if ($SmallFirst) { $items | Sort-Object Plays, Code } else { $items | Sort-Object Code }
if ($Only.Count -gt 0) {
    $ordered = $ordered | Where-Object { $Only -contains $_.Code }
}
if ($SkipLotteries.Count -gt 0) {
    $ordered = $ordered | Where-Object { $SkipLotteries -notcontains $_.Code }
}

if ($RetryFailed) {
    $ordered = $ordered | Where-Object {
        $state = Get-LotteryState -Row $latestSummary[$_.Code] -Plays $_.Plays -AcceptSkipOnly:$AcceptSkipOnly
        $state -in @("failed", "incomplete", "pending")
    }
    Write-Host "RetryFailed: $($ordered.Count) lotteries"
} else {
    $doneCodes = @()
    foreach ($lot in $ordered) {
        $state = Get-LotteryState -Row $latestSummary[$lot.Code] -Plays $lot.Plays -AcceptSkipOnly:$AcceptSkipOnly
        if ($state -eq "done" -or $state -eq "skip-only") {
            $doneCodes += $lot.Code
        }
    }
    if ($doneCodes.Count -gt 0) {
        $ordered = $ordered | Where-Object { $doneCodes -notcontains $_.Code }
        Write-Host "resume: skip $($doneCodes.Count) completed lotteries"
    }
}

if ($ordered.Count -eq 0) {
    Write-Host "nothing to run"
    Show-BatchStatus -Items $items -SummaryRows $summaryRows
    exit 0
}

Write-Host "batch lotteries=$($ordered.Count) delay=$Delay unit=$Unit out=$OutDir"
Write-Host "summary=$Summary"
Write-Host ""

$batchIdx = 0
$runOk = 0; $runSkip = 0; $runFail = 0

foreach ($lot in $ordered) {
    $batchIdx++
    $outFile = Join-Path $OutDir ("{0}.jsonl" -f $lot.Code)
    $goArgs = @(
        "run", "./cmd/real-bet-matrix",
        "-lottery", $lot.Code,
        "-delay", $Delay,
        "-unit", "$Unit",
        "-max-period-wait", "5m",
        "-verify",
        "-out", $outFile,
        "-truncate"
    )
    if ($Account) {
        $goArgs += @("-account", $Account)
    }

    Write-Host "=== [$batchIdx/$($ordered.Count)] $($lot.Code) plays=$($lot.Plays) ==="
    $started = Get-Date -Format "o"
    & go @goArgs
    $exitCode = $LASTEXITCODE

    $counts = Get-ResultCounts -Path $outFile
    $runOk += $counts.ok; $runSkip += $counts.skip; $runFail += $counts.fail

    $rowSnapshot = [pscustomobject]@{
        ok       = $counts.ok
        skip     = $counts.skip
        fail     = $counts.fail
        exitCode = $exitCode
    }
    $state = Get-LotteryState -Row $rowSnapshot -Plays $lot.Plays -AcceptSkipOnly:$AcceptSkipOnly

    $summaryObj = [ordered]@{
        lotteryCode = $lot.Code
        plays       = $lot.Plays
        ok          = $counts.ok
        skip        = $counts.skip
        fail        = $counts.fail
        exitCode    = $exitCode
        state       = $state
        failSample  = $counts.failSample
        outFile     = $outFile
        startedAt   = $started
        finishedAt  = (Get-Date -Format "o")
    }

    ($summaryObj | ConvertTo-Json -Compress) | Add-Content -Path $Summary -Encoding UTF8

    Write-Host ("    => ok={0} skip={1} fail={2} exit={3} state={4}" -f $counts.ok, $counts.skip, $counts.fail, $exitCode, $summaryObj.state)
    if ($counts.failSample) {
        Write-Host ("    failSample: {0}" -f $counts.failSample)
    }
    Write-Host ""

    if ($StopOnFail -and ($exitCode -ne 0 -or $counts.fail -gt 0)) {
        Write-Host "StopOnFail: 中止于 $($lot.Code)"
        exit $exitCode
    }
}

Merge-LotteryReports -OutDir $OutDir -Target $MergedReport

Write-Host "batch run done lotteries=$($ordered.Count) ok=$runOk skip=$runSkip fail=$runFail"
Show-BatchStatus -Items $items -SummaryRows (Read-SummaryRows -Path $Summary)

& go run ./cmd/real-bet-matrix-report -out $ReportOut -account $Account -unit $Unit -note "用户端 PlaceBet(real) + 封盘/无盘等待下期 + web_bets 对账"

if ($runFail -gt 0) { exit 2 }
exit 0
