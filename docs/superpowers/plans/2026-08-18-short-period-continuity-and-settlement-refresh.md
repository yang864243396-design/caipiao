# Short-period continuity and settlement refresh implementation plan

1. Add lottery schedule tests proving a future-period provisional close can be promoted once to its real close, while an active close remains immutable.
2. Extend the period schedule snapshot/anchor with provisional and real-close metadata, and promote the same issue when it becomes active.
3. Add worker tests for coalesced lottery wake-ups, then let draw notifications trigger one bounded placement scan after strategy advancement.
4. Add payout batch tests proving a failed historical lookup does not prevent later rows from settling, and cap stale recovery work per account tick.
5. Implement per-row payout error aggregation and preserve diagnostics without deleting or falsely settling pending rows.
6. Add frontend polling-state tests, dynamically switch list polling on actual WebSocket state, and refresh scheme bet records from WebSocket events with disconnected fallback polling.
7. Run targeted Go and client tests, production client build, formatting, diff checks, and related package regression tests.
