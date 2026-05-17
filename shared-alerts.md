# Shared Alerts

## 2026-05-17 (Updated by agent-performance-analyzer)
- ONGOING CRITICAL: Architecture Guardian lock-file out of sync — failed May 9, May 11, May 13, no run May 14/15/16/17; issue #2981; run `gh aw compile` to fix (8 days unresolved)
- ONGOING HIGH: Code Refiner — 0 runs (previously cancelling), systemic instability; needs investigation
- CRITICAL NEW: Daily Testify Expert — confirmed duplication: created issues #3015 (2026-05-16) AND #3019 (2026-05-17) for IDENTICAL file (internal/pathparser/pathparser_test.go); pattern also seen in #3006, #3012; agent needs dedup guard + closed-issue state check URGENTLY
- ONGOING MED: Daily Documentation Updater over-creates PRs (~16/run) — needs throttle/dedup; issue #2986; cascading into Doc Updater Auto-Merge stall
- ONGOING: Doc Updater Auto-Merge — stalled, blocked by upstream Doc Updater PR flood; will self-resolve if #2986 is fixed
- MONITOR: PR Triage Agent — improving (1/2 success); continue monitoring
- EXPECTED: Q(18), PR Code Quality Reviewer(17), Mergefest(15), CLA(13) — PR-triggered skips, normal; tracked in #2956
- RECOMMEND CLOSE: Daily Documentation Healer — 7+ consecutive clean days; issue #2972 should be closed now
- STABLE: Agentic Maintenance 5/5 (100%) — consistently top performer
- STABLE: Contribution Check 3/3 (100%) — no alert needed
- LOW: Metrics Collector shared memory stale 43 days (last update 2026-04-04) — metrics collector needs re-trigger
