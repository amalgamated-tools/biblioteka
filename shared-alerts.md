# Shared Alerts

## 2026-05-13 (Updated by agent-performance-analyzer)
- ONGOING HIGH: Architecture Guardian lock-file out of sync — failed May 9, May 11, May 13; issue #2981; run `gh aw compile` to fix
- ONGOING MED: Daily Documentation Updater failed (#2986); also over-creates PRs (~16/run) — needs throttle/dedup
- NEW MED: Daily Testify Expert re-opened previously auto-closed issue (#2991 ssrf, was fixed in #2960) — no closed-issue state check
- ONGOING MED: Daily Testify Expert duplicate smtp issues — #2977 (May 11) + #2984 (May 12), both OPEN
- ONGOING HIGH: Q(21), PR Code Quality Reviewer(15), Mergefest(15) all 100% skip — PR-triggered, normal; tracked in #2956
- MONITOR: Code Refiner — 2/3 cancelled, 0% success, systemic instability
- MONITOR: Doc Updater Auto-Merge — 4/4 skipped, likely blocked (no PRs to auto-merge)
- RECOVERED: Daily Documentation Healer — 3rd consecutive clean day; issue #2972 still open but can be closed
- STABLE: Contribution Check 3/3 (100%) — no alert needed
- STABLE: Agentic Maintenance 6/6 (100%) — consistently top performer
- LOW: Metrics Collector shared memory stale 39+ days (last update 2026-04-04)
