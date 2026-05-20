# Shared Alerts

## 2026-05-20 (Updated by agent-performance-analyzer)
- UNCHANGED CRITICAL: Architecture Guardian — OFFLINE 11+ consecutive days (May 9-19); issue #3033 OPEN 1d; requires `gh aw compile` or manual YAML/config fix — no automated recovery
- ONGOING: Code Refiner — not observed running today; yesterday 2/2 cancelled; 7+ day systemic issue
- RECOVERED: Daily Documentation Updater — success today; prior issue #3031 resolved; over-creation pattern (#2986) still being monitored
- STABLE FRAGILE: Daily Documentation Healer — 2nd clean day; issues #3022 still open; do NOT close until 3+ stable days confirmed
- IMPROVING: Daily Caveman Optimizer — no new PR today; PR #3028 still unmerged; if merges tomorrow, pattern is resolved
- NEW REPETITION: Testify Expert — 3 daily issues in 3 days (#3021, #3029, #3038); each file valid but queue overload; needs dedup guard or rate limit
- STALE DRAFTS: Dead Code Removal PR #3025 (2d draft) + Caveman Optimizer PR #3028 (2d open) — both need human merge
- EXPECTED: Q, PR Code Quality Reviewer, CLA — event-triggered skips normal
- STABLE: Agentic Maintenance 5/5 (100%), Contribution Check 4/4 (100%)
- LOW: Metrics Collector shared memory stale 46+ days (last update 2026-04-04); recommend re-run
