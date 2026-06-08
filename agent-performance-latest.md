# Agent Performance 2026-06-08

**Run**: agent-performance-analyzer | 2026-06-08T14:19Z | [§27144015202](https://github.com/amalgamated-tools/biblioteka/actions/runs/27144015202)

## Summary
- 17 workflows analyzed | Quality: 55/100 (↓-5) | Effectiveness: 50/100 (↑+2)
- Ecosystem: 🔴 DEGRADED (3rd consecutive) | Healthy: 7/17 (41%, ↓ from 47%)
- NEW CRITICAL: Architecture Guardian day 3 missing (escalated from Warning)
- NEW CRITICAL: Go Fan now actively FAILING (was missing, now run 27124476632 failed)
- ONGOING: Contribution Check day 11, Code Refiner day 9, Efficiency Improver day 6
- NEW WATCH: Daily Doc Healer + Daily Malicious Scan both failed at 07:22 UTC (shared infra?)
- NEW AGENT: Typist - Go Type Analysis (first run today, succeeded)

## Agent Scores
| Agent | Quality | Effectiveness | Notes |
|-------|---------|--------------|-------|
| Agentic Maintenance | 90 ✅ | 95 ✅ | 5 runs today, perfect |
| Daily Testify Expert | 82 ✅ | 80 ✅ | Succeeded today |
| Daily Caveman Optimizer | 82 ✅ | 78 ✅ | PR #3191 created today |
| Daily Doc Healer | 75 ⚠️ | 73 ⚠️ | FIRST FAIL today 07:22 UTC (shared infra?) |
| PR Triage Agent | 75 ✅ | 72 ✅ | Running correctly |
| Duplicate Code Detector | 74 ✅ | 75 ✅ | Issue #3189 today → copilot PR #3190 |
| copilot-swe-agent | 72 ✅ | 65 ✅ | ~60% merge rate, 8 open PRs |
| Typist - Go Type Analysis | 70 🆕 | 65 🆕 | New today, 1 run success |
| Daily Malicious Scan | 73 ⚠️ | 70 ⚠️ | FIRST FAIL today 07:22 UTC (shared infra?) |
| Code Simplifier | 55 ⚠️ | 48 ⚠️ | 0% merge 7 days, today succeeded |
| Daily Doc Updater | 48 ⚠️ | 43 ⚠️ | PR pile-up, scope overlap with Doc Healer |
| Dead Code Remover | 48 ⚠️ | 38 ⚠️ | PR #3176 open, 0% merge rate |
| Go Fan | 25 🔴 | 15 🔴 | CRITICAL: now failing (run 27124476632), day 3 |
| Architecture Guardian | 20 🔴 | 15 🔴 | CRITICAL: day 3 missing, no output |
| Efficiency Improver | 20 🔴 | 10 🔴 | Day 6 failures (June 3-8) |
| Contribution Check | 12 🔴 | 8 🔴 | Day 11, 3 runs today all fail, content valid |
| Code Refiner | 8 🔴 | 5 🔴 | Day 9 cancelled, disable immediately |

## PR Merge Rates (30 days)
- dependabot: 100% (11/11)
- copilot-swe-agent: ~60% (9/~15) — acceptable
- amalgamated-bot: ~35% (7/~20) — low; drag from Code Simplifier + Dead Code Remover (0 merges each)

## Key Changes vs June 7
- 🔴 NEW CRITICAL: Architecture Guardian escalated (day 3 missing)
- 🔴 NEW CRITICAL: Go Fan now actively failing (was missing, escalated)
- ⚠️ NEW WATCH: Daily Doc Healer first failure (07:22 UTC)
- ⚠️ NEW WATCH: Daily Malicious Scan first failure (07:22 UTC, same minute as Doc Healer)
- 🆕 NEW: Typist - Go Type Analysis appeared (1 run, success)
- → Code Refiner day 9, Contribution Check day 11, Efficiency Improver day 6 (all unchanged)

## Recommendations
1. 🔴 Disable Code Refiner — day 9, CI waste (15 min effort)
2. 🔴 Fix Contribution Check post-output infra step (day 11, ~1-2h)
3. 🔴 Investigate Architecture Guardian + Go Fan — shared scheduling/secrets issue
4. 🔴 Diagnose Efficiency Improver — regression since June 3
5. ⚠️ Watch Doc Healer + Malicious Scan tomorrow — shared infra or transient?
6. ⚠️ Gate Code Simplifier + Dead Code Remover on open PR count
7. ⚠️ Define scope boundary between Daily Doc Updater and Daily Doc Healer
