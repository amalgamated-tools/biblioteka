# Agent Performance 2026-06-07

**Run**: agent-performance-analyzer | 2026-06-07T13:22Z | [§27093759465](https://github.com/amalgamated-tools/biblioteka/actions/runs/27093759465)

## Summary
- 15 workflows analyzed | Quality: 60/100 (↓-2) | Effectiveness: 48/100 (↓-3)
- Ecosystem: 🔴 DEGRADED | Clean: 7/15 (47% healthy)
- ESCALATED: Contribution Check day 10 failures (now 10 consecutive)
- NEW CRITICAL: Efficiency Improver 5 consecutive daily failures (new alert)
- ESCALATED: Architecture Guardian + Go Fan both missing 2+ days
- PERSISTENT: Code Refiner day 8 skip/cancel loop

## Agent Scores
| Agent | Quality | Effectiveness | Notes |
|-------|---------|--------------|-------|
| Agentic Maintenance | 90 ✅ | 95 ✅ | 7+ days perfect, 4 runs today |
| Daily Testify Expert | 82 ✅ | 80 ✅ | 7/7, issue #3173 open |
| Daily Caveman Optimizer | 82 ✅ | 78 ✅ | 7/7, PR #3161 merged |
| Daily Doc Healer | 80 ✅ | 78 ✅ | 7/7, PR #3174 open |
| Daily Malicious Scan | 78 ✅ | 75 ✅ | 1/1 today |
| PR Triage Agent | 75 ✅ | 72 ✅ | running correctly |
| Duplicate Code Detector | 74 ✅ | 75 ✅ | issues #3181/#3182 → PRs #3183/#3184 |
| Architecture Guardian | 60 ⚠️ | 45 ⚠️ | MISSING June 6-7 (2 days, escalated) |
| Go Fan | 60 ⚠️ | 45 ⚠️ | MISSING June 6-7 (2 days, escalated) |
| Code Simplifier | 55 ⚠️ | 48 ⚠️ | 0% merge rate 7 days; quality improving |
| Daily Doc Updater | 48 ⚠️ | 43 ⚠️ | PR pile-up: #3162/#3174/#3185 open |
| Dead Code Remover | 48 ⚠️ | 38 ⚠️ | 0% merge rate; PR #3176 open |
| Efficiency Improver | 20 🔴 | 10 🔴 | NEW: 5 consecutive daily failures |
| Contribution Check | 12 🔴 | 8 🔴 | DAY 10 infra failures; content still valid |
| Code Refiner | 8 🔴 | 5 🔴 | DAY 8 skip/cancel loop |

## PR Merge Rates (14 days)
- amalgamated-bot: 20% (4/20) — problematic; Code Simplifier + Dead Code = 0 merges
- copilot-swe-agent: 55% (6/11) — acceptable; 3 new refactor PRs today
- dependabot: 100% (6/6)

## Key Changes vs June 6
- 🔴 NEW: Efficiency Improver tracked — 5 consecutive failures since June 3
- 🔴 ESCALATED: Contribution Check now day 10 (was day 5)
- ⚠️ ESCALATED: Architecture Guardian + Go Fan both now 2 days missing
- ✅ Duplicate Code Detector ecosystem effect: issues → copilot-swe PRs same day
- → Code Refiner: day 8 (unchanged, persistent)

## Recommendations
1. 🔴 Fix Contribution Check — post-output infra step (day 10, ~1-2h fix)
2. 🔴 Disable Code Refiner — day 8, CI waste
3. 🔴 Diagnose Efficiency Improver — successful before June 2 (PR #3116), failing since June 3
4. ⚠️ Investigate Architecture Guardian + Go Fan scheduling (both missing June 6-7)
5. ⚠️ Gate Dead Code Remover + Code Simplifier on open PR count
6. ⚠️ Resolve Daily Doc Updater / Daily Doc Healer scope overlap
