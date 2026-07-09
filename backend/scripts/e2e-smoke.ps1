# E2E smoke test aligned with docs/integration-checklist.md
# Usage: .\scripts\e2e-smoke.ps1 [-BaseUrl http://127.0.0.1:8081/api/v1]

param(
    [string]$BaseUrl = 'http://127.0.0.1:8081/api/v1'
)

$ErrorActionPreference = 'Continue'
$script:pass = 0
$script:fail = 0
$script:skip = 0
$script:results = @()

function Record($id, $ok, $detail) {
    $script:results += [PSCustomObject]@{ Id = $id; Ok = $ok; Detail = $detail }
    if ($null -eq $ok) { $script:skip++ }
    elseif ($ok) { $script:pass++ }
    else { $script:fail++ }
}

function Invoke-Api {
    param(
        [string]$Method = 'GET',
        [string]$Path,
        [hashtable]$Headers = @{},
        $Body = $null
    )
    $uri = "$BaseUrl$Path"
    $params = @{
        Uri         = $uri
        Method      = $Method
        Headers     = $Headers
        TimeoutSec  = 15
    }
    if ($null -ne $Body) {
        $params.Body = ($Body | ConvertTo-Json -Compress)
        $params.ContentType = 'application/json'
    }
    return Invoke-RestMethod @params
}

Write-Host "=== E2E Smoke @ $BaseUrl ===" -ForegroundColor Cyan

# 0.3 Health
try {
    $h = Invoke-Api -Path '/health'
    Record '0.3' ($h.code -eq 0 -and $h.data.db -eq 'up') "code=$($h.code) db=$($h.data.db)"
} catch {
    Record '0.3' $false $_.Exception.Message
    Write-Host "Backend unreachable; aborting." -ForegroundColor Red
    $script:results | Format-Table -AutoSize
    exit 1
}

# 1.x Auth
try {
    $bad = Invoke-Api -Method POST -Path '/client/auth/login' -Body @{ account = 'vs8888'; password = 'wrong' }
    Record '1.2' ($bad.code -ne 0) "expected fail code=$($bad.code)"
} catch {
    Record '1.2' $true 'HTTP error on bad login (acceptable)'
}

$clientTok = $null
try {
    $login = Invoke-Api -Method POST -Path '/client/auth/login' -Body @{ account = 'vs8888'; password = 'vs8888' }
    $clientTok = $login.data.accessToken
    Record '1.1' ($login.code -eq 0 -and $clientTok) "account=$($login.data.account)"
} catch {
    Record '1.1' $false $_.Exception.Message
}

$adminTok = $null
try {
    $alogin = Invoke-Api -Method POST -Path '/admin/auth/login' -Body @{ account = 'admin'; password = 'admin123' }
    $adminTok = $alogin.data.accessToken
    Record '1.3' ($alogin.code -eq 0 -and $adminTok) "roleId=$($alogin.data.roleId)"
} catch {
    Record '1.3' $false $_.Exception.Message
}

$clientH = @{ Authorization = "Bearer $clientTok" }
$adminH = @{ Authorization = "Bearer $adminTok" }

# 1.4 Maintenance on
try {
    $on = Invoke-Api -Method PUT -Path '/admin/operations/maintenance' -Headers $adminH -Body @{
        enabled = $true; title = 'E2E维护'; message = 'smoke test'
    }
    $pub = Invoke-Api -Path '/public/maintenance'
    Record '1.4' ($on.code -eq 0 -and $pub.data.enabled -eq $true) "public.enabled=$($pub.data.enabled)"
} catch {
    Record '1.4' $false $_.Exception.Message
}

# 1.6 Maintenance off
try {
    $off = Invoke-Api -Method PUT -Path '/admin/operations/maintenance' -Headers $adminH -Body @{ enabled = $false }
    $pub2 = Invoke-Api -Path '/public/maintenance'
    Record '1.6' ($off.code -eq 0 -and $pub2.data.enabled -eq $false) "public.enabled=$($pub2.data.enabled)"
} catch {
    Record '1.6' $false $_.Exception.Message
}

# Phase 1
foreach ($pair in @(
    @{ Id = '2.1'; Path = '/client/cloud/schemes/running' },
    @{ Id = '2.2'; Path = '/client/cloud/bet-records?mode=real' },
    @{ Id = '2.2-sim'; Path = '/client/cloud/bet-records?mode=sim' }
)) {
    try {
        $r = Invoke-Api -Path $pair.Path -Headers $clientH
        Record $pair.Id ($r.code -eq 0) 'ok'
    } catch { Record $pair.Id $false $_.Exception.Message }
}

# Phase 2
foreach ($pair in @(
    @{ Id = '3.1-profile'; Path = '/client/member/profile' },
    @{ Id = '3.1-wallet'; Path = '/client/member/wallet' },
    @{ Id = '3.2'; Path = '/client/orders/ledger?limit=5' },
    @{ Id = '3.3-bets'; Path = '/client/orders/bets?limit=5' },
    @{ Id = '3.3-chases'; Path = '/client/orders/chases?limit=5' },
    @{ Id = '3.4-ledger'; Path = '/client/orders/ledger?scope=team&limit=5' },
    @{ Id = '3.6'; Path = '/client/funds/records?limit=10' },
    @{ Id = '3.7'; Path = '/client/funds/recharge-channels' },
    @{ Id = '3.9'; Path = '/client/member/payout-accounts' }
)) {
    try {
        $r = Invoke-Api -Path $pair.Path -Headers $clientH
        Record $pair.Id ($r.code -eq 0) 'ok'
    } catch { Record $pair.Id $false $_.Exception.Message }
}

# 3.5 withdraw context
try {
    $wc = Invoke-Api -Path '/client/funds/withdraw/context' -Headers $clientH
    Record '3.5-ctx' ($wc.code -eq 0) "canWithdraw=$($wc.data.canWithdraw)"
} catch { Record '3.5-ctx' $false $_.Exception.Message }

# 3.8 demo recharge (unique amount to detect)
$rechargeAmt = 101.01
try {
    $ch = Invoke-Api -Path '/client/funds/recharge-channels' -Headers $clientH
    $channelId = $ch.data.items[0].id
    $wBefore = (Invoke-Api -Path '/client/member/wallet' -Headers $clientH).data.availableBalance
    $rc = Invoke-Api -Method POST -Path '/client/funds/recharge' -Headers $clientH -Body @{
        channelId = $channelId; amount = $rechargeAmt
    }
    $wAfter = (Invoke-Api -Path '/client/member/wallet' -Headers $clientH).data.availableBalance
    $delta = [math]::Round($wAfter - $wBefore, 2)
    Record '3.8' ($rc.code -eq 0 -and $rc.data.status -eq 'paid' -and $delta -gt 0) "order=$($rc.data.orderNo) credit=$($rc.data.actualCredit) delta=$delta"
} catch { Record '3.8' $false $_.Exception.Message }

# Phase 3
foreach ($pair in @(
    @{ Id = '4.1'; Path = '/client/copy-hall/rankings?lotteryCode=cq_ssc&board=master' },
    @{ Id = '4.2'; Path = '/client/schemes/share-catalog' },
    @{ Id = '4.2-priv'; Path = '/client/schemes' },
    @{ Id = '4.6'; Path = '/client/games/pk10/draws?limit=5' }
)) {
    try {
        $r = Invoke-Api -Path $pair.Path -Headers $clientH
        Record $pair.Id ($r.code -eq 0) 'ok'
    } catch { Record $pair.Id $false $_.Exception.Message }
}

try {
    $gd = Invoke-Api -Path '/client/games/pk10/detail' -Headers $clientH
    Record '4.4' ($gd.code -eq 0 -and $gd.data.lotteryCode) "issue=$($gd.data.currentIssue)"
} catch { Record '4.4' $false $_.Exception.Message }

# Phase 4
foreach ($pair in @(
    @{ Id = '5.1'; Path = '/client/content/announcements' },
    @{ Id = '5.2-faq'; Path = '/client/content/faq' },
    @{ Id = '5.2-help'; Path = '/client/content/help' }
)) {
    try {
        $r = Invoke-Api -Path $pair.Path -Headers $clientH
        Record $pair.Id ($r.code -eq 0) 'ok'
    } catch { Record $pair.Id $false $_.Exception.Message }
}

try {
    $fb = Invoke-Api -Method POST -Path '/client/content/feedback' -Headers $clientH -Body @{
        subject = 'E2E smoke'; content = "automated $(Get-Date -Format o)"
    }
    Record '5.3' ($fb.code -eq 0 -and $fb.data.id) "id=$($fb.data.id)"
} catch { Record '5.3' $false $_.Exception.Message }

# Phase 5 Admin
foreach ($pair in @(
    @{ Id = '6.1'; Path = '/admin/dashboard/kpi' },
    @{ Id = '6.2'; Path = '/admin/members?searchField=id&keyword=1' },
    @{ Id = '6.3-list'; Path = '/admin/funds/withdraw/orders' },
    @{ Id = '6.5'; Path = '/admin/schemes/instances?scope=share' },
    @{ Id = '6.6'; Path = '/admin/copy-hall/rankings?lotteryCode=tron_ffc_1m&board=master' },
    @{ Id = '6.7'; Path = '/admin/games/lottery-catalog' },
    @{ Id = '6.8'; Path = '/admin/funds/recharge-channels' },
    @{ Id = '6.11'; Path = '/admin/system/audit-logs?limit=5' },
    @{ Id = '6.12-lottery'; Path = '/admin/reports/lottery-stat' },
    @{ Id = '6.12-pnl'; Path = '/admin/reports/pnl' },
    @{ Id = 'bundle'; Path = '/admin/content/bundle' }
)) {
    try {
        $r = Invoke-Api -Path $pair.Path -Headers $adminH
        Record $pair.Id ($r.code -eq 0) 'ok'
    } catch { Record $pair.Id $false $_.Exception.Message }
}

try {
    $mem = Invoke-Api -Path '/admin/members/1' -Headers $adminH
    Record '6.2-detail' ($mem.code -eq 0) "id=$($mem.data.id)"
} catch { Record '6.2-detail' $false $_.Exception.Message }

# WS endpoints reachable (HTTP upgrade probe = expect 400/426 not connection refused)
foreach ($pair in @(
    @{ Id = 'WS-public'; Url = ($BaseUrl -replace '/api/v1','/api/v1/ws/public') },
    @{ Id = 'WS-client'; Url = ($BaseUrl -replace '/api/v1','/api/v1/ws/client') + "?token=$clientTok" },
    @{ Id = 'WS-admin'; Url = ($BaseUrl -replace '/api/v1','/api/v1/ws/admin') + "?token=$adminTok" }
)) {
    try {
        $resp = Invoke-WebRequest -Uri $pair.Url -TimeoutSec 5 -ErrorAction Stop
        Record $pair.Id ($resp.StatusCode -ge 200) "status=$($resp.StatusCode)"
    } catch {
        $ok = $_.Exception.Message -match '426|400|401|Upgrade'
        Record $pair.Id $ok $_.Exception.Message.Substring(0, [Math]::Min(60, $_.Exception.Message.Length))
    }
}

Write-Host ""
Write-Host "PASS: $($script:pass)  FAIL: $($script:fail)  SKIP: $($script:skip)" -ForegroundColor $(if ($script:fail -eq 0) { 'Green' } else { 'Yellow' })
$script:results | Format-Table -AutoSize
if ($script:fail -gt 0) { exit 1 }
