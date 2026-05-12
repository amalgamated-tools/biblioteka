# Shared Alerts

## 2026-05-12 (Updated by agent-performance-analyzer)
- REGRESSION HIGH: Architecture Guardian failed again (#2981, May 11) — fix in #2966 did not hold; lock file issue persists
- REGRESSION MED: Daily Documentation Updater failed today (#2986); also over-creates PRs (~16/run)
- NEW MED: Daily Testify Expert duplicate issues — #2977 (May 11) = #2984 (May 12), same file, no dedup logic
- ONGOING HIGH: Q(22), PR Code Quality Reviewer(14), Mergefest(15) all 100% skip — issue #2956 still open
- IMPROVING: Daily Documentation Healer — 2nd clean day; issue #2972 still open, monitor for 3rd
- STABLE: Contribution Check 3/3 (100%) — marking stable, no alert needed
- STABLE: Agentic Maintenance 6/6 (100%) — consistently top performer
- MONITOR: Doc Updater Auto-Merge — 4/4 non-success runs, may be blocked
- MONITOR: Code Refiner — 3/3 non-success, possible skip/broken trigger
- LOW: Metrics Collector shared memory stale 38+ days (last update 2026-04-04)
- RESOLVED(monitoring): Concurrency cascade — no new evidence
