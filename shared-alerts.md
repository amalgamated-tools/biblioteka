# Shared Alerts

## 2026-05-14 (Updated by agent-performance-analyzer)
- ONGOING HIGH: Architecture Guardian lock-file out of sync — failed May 9, May 11, May 13, no run May 14; issue #2981; run `gh aw compile` to fix (5+ days unresolved)
- ONGOING MED: Daily Documentation Updater over-creates PRs (~16/run) — needs throttle/dedup; issue #2986
- ONGOING MED: Daily Testify Expert duplicate smtp issues — #2977 (May 11) + #2984 (May 12), both OPEN
- ONGOING MED: Daily Testify Expert re-opened previously auto-closed issue (#2991 ssrf, was fixed in #2960) — no closed-issue state check
- ONGOING: Code Refiner — 0% success (2 cancellations), systemic instability; no issue yet
- ONGOING: Doc Updater Auto-Merge — 4/4 skipped, blocked by upstream Doc Updater PR flood
- EXPECTED: Q(22), PR Code Quality Reviewer(16), Mergefest(15), CLA(13) — PR-triggered skips, normal; tracked in #2956
- RECOMMEND CLOSE: Daily Documentation Healer — 4th consecutive clean day; issue #2972 should be closed
- STABLE: Contribution Check 3/3 (100%) — no alert needed
- STABLE: Agentic Maintenance 5/5 (100%) — consistently top performer
- LOW: Metrics Collector shared memory stale 40+ days (last update 2026-04-04)
