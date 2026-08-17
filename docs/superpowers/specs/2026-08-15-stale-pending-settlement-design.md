# Stale Pending Settlement Recovery Design

## Goal

Allow a real-money scheme to continue betting new third-party periods while preserving and compensating historical accepted bets that remain locally `pending`.

## Observed failure

`inst-1-1786600137736` contains accepted third-party orders from period `85321787` that remain pending. The payout synchronizer calls `QuerySettlement`, but its fallback search only reads pages 1 through 3 of the third-party bet list. Historical orders outside that window are never found. The scheme worker then treats any accepted pending record as a global block, so no later period can be placed.

## Required behavior

1. A current target period remains blocked if this scheme already has a local claim for that same third-party period.
2. A pending accepted order for an earlier third-party period remains a financial record and is never deleted, re-submitted, or locally settled from draw data.
3. Earlier pending orders do not prevent a new, strictly-current target period from being placed.
4. Payout synchronization continues to query historical pending orders and settles them only from a confirmed third-party response.
5. Diagnostics expose outstanding pending order identifiers, periods, ages, and whether they block the currently open target period.

## Design

`Worker.hasUnsettledGuajiBet` becomes period-scoped. It keeps the existing same-period claim check, but only blocks a historical accepted record when its authoritative `third_party_period` is empty or is not strictly before the fresh current open period. This retains conservative behavior for ambiguous records while releasing known older periods.

`guaji.Client.GetWebBet` retains the detail endpoint and recent-list fast path, then receives a bounded historical page-search capability. The payout synchronizer uses it for accepted pending orders outside the recent window, with a small request budget per order and a rotating persisted scan cursor per account. It only applies a settlement after the upstream marks the record settled; transport errors and unconfirmed records remain pending.

The admin runtime diagnostic reports accepted pending orders for the instance, including the third-party period, bet id, placement time, and a `blocksCurrentPeriod` flag. This lets operators distinguish a financial recovery backlog from a current-period duplicate guard.

## Safety constraints

- Never infer win/loss or payout from local draws for real third-party records.
- Never delete, recreate, or re-place an accepted third-party bet.
- Never allow a second local claim for the current third-party period.
- A missing or non-numeric third-party period stays blocking until settlement succeeds.

## Verification

- Unit tests cover accepted historical pending records that no longer block a later open period, same or ambiguous periods that remain blocking, and diagnostics classification.
- Guaji client tests cover recent-list lookup and historical paged lookup without unbounded requests.
- Targeted Go tests and server build must pass before deployment.
