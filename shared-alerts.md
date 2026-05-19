# Shared Alerts

## 2026-05-19 (Updated by agent-performance-analyzer)
- ESCALATED CRITICAL: Architecture Guardian — now OFFLINE 10+ consecutive days (failed May 9-15, no run May 16-19); issue #2981 now 8+ days unresolved; run `gh aw compile` to fix — manual intervention required
- ONGOING HIGH: Code Refiner — 2/2 cancelled today; 7+ day systemic cancellation; needs workflow config investigation
- NEW DUAL FAILURE: Daily Documentation Updater — (1) NEW outright failure today (issue #3031); (2) ONGOING over-creation ~16 PRs/run (#2986); both need fixing together
- RECOVERED: Daily Documentation Healer — 1/1 success today; issues #2972 + #3022 still open; do NOT close until 3+ stable days
- ONGOING CASCADE: Doc Updater over-creation (#2986) → Doc Healer stress → Doc Auto-Merge stall (3-agent cascade); fixing #2986+#3031 will resolve cascade
- NEW PATTERN: Daily Caveman Optimizer — over_creation + repetition; PR #3018 unmerged, PR #3028 created today; needs merge or throttle
- IMPROVING: PR Triage Agent — 1/2 (50%); improving from prior zero; continue monitoring
- TESTIFY EXPERT: today's #3029 is new file (ssrf_test.go); prior duplication pattern (#3019) still open; add dedup guard
- EXPECTED: Q, PR Code Quality Reviewer, Mergefest, CLA — event-triggered skips normal; tracked in #2956
- STABLE: Agentic Maintenance 6/6 (100%)
- STABLE: Contribution Check 3/3 (100%)
- LOW: Metrics Collector shared memory stale 45+ days (last update 2026-04-04)
