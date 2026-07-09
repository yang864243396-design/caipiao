# 从 by-lottery/*.jsonl 或 merged jsonl 生成 Markdown 测试报告
param(
    [string]$OutDir = "data/real-bet-matrix/by-lottery",
    [string]$MergedReport = "data/real-bet-matrix/all-results.jsonl",
    [string]$Summary = "data/real-bet-matrix/batch-summary.jsonl",
    [string]$ReportOut = "../docs/real-bet-matrix-test-report.md",
    [string]$Account = "",
    [double]$Unit = 2,
    [string]$Note = ""
)

$ErrorActionPreference = "Stop"
$BackendRoot = Split-Path $PSScriptRoot -Parent
Set-Location $BackendRoot

function Read-JsonLines {
    param([string]$Path)
    $rows = @()
    if (-not (Test-Path $Path)) { return $rows }
    foreach ($line in Get-Content $Path -Encoding UTF8) {
        if ([string]::IsNullOrWhiteSpace($line)) { continue }
        try { $rows += ($line | ConvertFrom-Json) } catch { }
    }
    return $rows
}

function Escape-Md {
    param([string]$Text)
    if ($null -eq $Text) { return "" }
    return ($Text -replace '\|', '\|' -replace "`r`n", " " -replace "`n", " ")
}

function Add-Line {
    param([System.Text.StringBuilder]$Builder, [string]$Line)
    [void]$Builder.AppendLine($Line)
}

$allRows = @()
if (Test-Path $MergedReport) {
    $allRows = Read-JsonLines -Path $MergedReport
}
if ($allRows.Count -eq 0 -and (Test-Path $OutDir)) {
    Get-ChildItem -Path $OutDir -Filter "*.jsonl" -File | Sort-Object Name | ForEach-Object {
        $allRows += Read-JsonLines -Path $_.FullName
    }
}

$ok = @($allRows | Where-Object { $_.status -eq "ok" })
$skip = @($allRows | Where-Object { $_.status -eq "skip" })
$fail = @($allRows | Where-Object { $_.status -eq "fail" })
$total = $allRows.Count

$byLottery = $allRows | Group-Object lotteryCode | Sort-Object Name
$skipByReason = $skip | Group-Object error | Sort-Object Count -Descending
$failByReason = $fail | Group-Object error | Sort-Object Count -Descending

$now = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
$passRate = if ($total -gt 0) { [math]::Round(100.0 * $ok.Count / $total, 2) } else { 0 }
$expectedTotal = 4384

$sb = New-Object System.Text.StringBuilder
Add-Line $sb "# 全彩种全玩法真实下单矩阵测试报告"
Add-Line $sb ""
Add-Line $sb "> 自动生成于 $now"
if ($Note) { Add-Line $sb "> $Note" }
Add-Line $sb ""
Add-Line $sb "## 1. 测试概览"
Add-Line $sb ""
Add-Line $sb '| 指标 | 数值 |'
Add-Line $sb '|------|------|'
Add-Line $sb "| 矩阵总行数 | $total / $expectedTotal |"
Add-Line $sb "| 成功 (ok) | $($ok.Count) |"
Add-Line $sb "| 跳过 (skip) | $($skip.Count) |"
Add-Line $sb "| 失败 (fail) | $($fail.Count) |"
Add-Line $sb "| 通过率 (ok/总数) | ${passRate}% |"
if ($Account) { Add-Line $sb "| 测试账号 | $Account |" }
Add-Line $sb "| 单注金额 (unit) | $Unit 元 |"
Add-Line $sb ""
Add-Line $sb "## 2. 按彩种汇总"
Add-Line $sb ""
Add-Line $sb '| 彩种 | 总数 | ok | skip | fail | 通过率 | 状态 |'
Add-Line $sb '|------|------|----|------|------|--------|------|'

foreach ($g in $byLottery) {
    $code = $g.Name
    $items = $g.Group
    $nOk = @($items | Where-Object { $_.status -eq "ok" }).Count
    $nSkip = @($items | Where-Object { $_.status -eq "skip" }).Count
    $nFail = @($items | Where-Object { $_.status -eq "fail" }).Count
    $nAll = $items.Count
    $rate = if ($nAll -gt 0) { [math]::Round(100.0 * $nOk / $nAll, 1) } else { 0 }
    $state = if ($nFail -gt 0) { "fail" } elseif ($nOk -eq $nAll) { "done" } elseif ($nOk -gt 0) { "partial" } elseif ($nSkip -eq $nAll) { "all-skip" } else { "incomplete" }
    Add-Line $sb ('| {0} | {1} | {2} | {3} | {4} | {5}% | {6} |' -f $code, $nAll, $nOk, $nSkip, $nFail, $rate, $state)
}

Add-Line $sb ""
Add-Line $sb "## 3. Skip 原因分布"
Add-Line $sb ""
if ($skipByReason.Count -eq 0) {
    Add-Line $sb "无 skip 记录。"
} else {
    Add-Line $sb '| 次数 | 原因 |'
    Add-Line $sb '|------|------|'
    foreach ($g in $skipByReason) {
        Add-Line $sb ('| {0} | {1} |' -f $g.Count, (Escape-Md $g.Name))
    }
}

Add-Line $sb ""
Add-Line $sb "## 4. 失败原因分布"
Add-Line $sb ""
if ($failByReason.Count -eq 0) {
    Add-Line $sb "无 fail 记录。"
} else {
    Add-Line $sb '| 次数 | 原因 |'
    Add-Line $sb '|------|------|'
    foreach ($g in $failByReason) {
        Add-Line $sb ('| {0} | {1} |' -f $g.Count, (Escape-Md $g.Name))
    }
}

Add-Line $sb ""
Add-Line $sb "## 5. 失败明细（最多 200 条）"
Add-Line $sb ""
if ($fail.Count -eq 0) {
    Add-Line $sb "无。"
} else {
    Add-Line $sb '| 彩种 | type/sub | 玩法 | rule | 错误 |'
    Add-Line $sb '|------|----------|------|------|------|'
    $shown = 0
    foreach ($r in $fail) {
        if ($shown -ge 200) { break }
        $ts = "$($r.typeId)/$($r.subId)"
        Add-Line $sb ('| {0} | {1} | {2} | {3} | {4} |' -f $r.lotteryCode, $ts, (Escape-Md $r.label), $r.ruleId, (Escape-Md $r.error))
        $shown++
    }
    if ($fail.Count -gt 200) {
        Add-Line $sb ""
        Add-Line $sb "> 另有 $($fail.Count - 200) 条失败，见 ``data/real-bet-matrix/all-results.jsonl``。"
    }
}

Add-Line $sb ""
Add-Line $sb "## 6. 结论"
Add-Line $sb ""
if ($total -lt $expectedTotal) {
    Add-Line $sb "- **进度**：矩阵尚未跑完（$total / $expectedTotal），以下为当前快照。"
}
if ($fail.Count -eq 0 -and $ok.Count -gt 0 -and $skip.Count -eq 0 -and $total -ge $expectedTotal) {
    Add-Line $sb "- **结论**：全矩阵真实下单全部成功。"
} elseif ($fail.Count -eq 0 -and $ok.Count -gt 0) {
    Add-Line $sb "- **结论**：无编码/下单失败；skip 多为封盘，属正常边界。"
} elseif ($fail.Count -gt 0) {
    Add-Line $sb "- **结论**：存在 fail，需按第 4–5 节修复 guajibet 编码或 solo/bets_nums 后重跑失败项。"
} else {
    Add-Line $sb "- **结论**：尚无成功下单，请检查 GUAJI_ENABLED、会员挂账 token、期号同步。"
}

Add-Line $sb ""
Add-Line $sb "## 7. 原始数据"
Add-Line $sb ""
Add-Line $sb '- 分彩种：`backend/data/real-bet-matrix/by-lottery/*.jsonl`'
Add-Line $sb '- 合并：`backend/data/real-bet-matrix/all-results.jsonl`'
Add-Line $sb '- 批次汇总：`backend/data/real-bet-matrix/batch-summary.jsonl`'
Add-Line $sb '- 跑批日志：`backend/data/real-bet-matrix/batch-run.log`'

$reportPath = Join-Path $BackendRoot $ReportOut
$reportDir = Split-Path $reportPath -Parent
New-Item -ItemType Directory -Force -Path $reportDir | Out-Null
[System.IO.File]::WriteAllText($reportPath, $sb.ToString(), [System.Text.UTF8Encoding]::new($false))
Write-Host "report -> $reportPath (rows=$total ok=$($ok.Count) skip=$($skip.Count) fail=$($fail.Count))"
