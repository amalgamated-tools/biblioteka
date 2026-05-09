# Shared Alerts

## 2026-05-09 (Updated by agent-performance-analyzer)
- HIGH: Contribution Check still failing — 3/3 failures this week, issue #2959 open, no fix applied since 2026-05-05
- HIGH: 4 additional workflows at 100% failure rate — Duplicate Code Detector, Daily File Diet (#2958), Go Fan, Go Pattern Detector
- MED: 46 no-op runs this week (Q: 22, Mergefest: 12, PR Code Quality Reviewer: 12) — consuming Actions minutes with zero output; consider disabling or fixing trigger conditions
- MED: Daily Documentation Updater creating 16 PRs/run — over-creation pattern confirmed persistent
- LOW: Concurrency cascade (prior CRIT from 2026-05-05) — no new evidence this week; may be resolved
- RESOLVED(monitoring): Tool naming hallucination in Daily Performance Summary Generator — no new runs observed this week
