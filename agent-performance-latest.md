# Agent Performance 2026-06-09

**Run**: agent-performance-analyzer | 2026-06-09T13:48Z | [§27210614193](https://github.com/amalgamated-tools/biblioteka/actions/runs/27210614193)

## Summary
- 17 workflows analyzed | Quality: 64/100 (↑+9) | Effectiveness: 56/100 (↑+6)
- Ecosystem: ⚠️ RECOVERING (was DEGRADED 3 consecutive days; 4 agents recovered today)
- Healthy: 9/17 (53%, ↑ from 41%)
- RECOVERED: Go Fan ✅, Architecture Guardian ✅ (last run June 8), Daily Doc Healer ✅, Daily Malicious Scan ✅
- ONGOING CRITICAL: Contribution Check day 12, Efficiency Improver day 7, Code Refiner day 9+
- NEW: Typist - Go Type Analysis (day 2, succeeding)

## Agent Scores
| Agent | Quality | Effectiveness | Notes |
|-------|---------|--------------|-------|
| Agentic Maintenance | 90 ✅ | 95 ✅ | 4 runs today, perfect |
| Daily Testify Expert | 82 ✅ | 80 ✅ | Issue #3204 today |
| Daily Caveman Optimizer | 82 ✅ | 78 ✅ | PR #3191 open (title needs conventional commit format) |
| Daily Documentation Healer | 80 ✅ | 77 ✅ | RECOVERED after June 8 transient failure |
| Daily Malicious Code Scan | 75 ✅ | 72 ✅ | RECOVERED after June 8 transient failure |
| PR Triage Agent | 75 ✅ | 72 ✅ | Running normally |
| Duplicate Code Detector | 74 ✅ | 75 ✅ | Issues #3198-3200 → PRs #3201-3203 pipeline working |
| copilot-swe-agent | 72 ✅ | 65 ✅ | 53% merge rate; 8 open PRs accumulating |
| Architecture Guardian | 68 ⚠️ | 65 ⚠️ | BACK - last run June 8 success; watch for recurrence |
| Typist - Go Type Analysis | 70 🆕 | 65 🆕 | Day 2, succeeding; establishing baseline |
| Go Fan | 65 ⚠️ | 55 ⚠️ | RECOVERED; issue #3205 today; watch for recurrence |
| Code Simplifier | 55 ⚠️ | 48 ⚠️ | 0% merge rate on open PRs |
| Daily Documentation Updater | 48 ⚠️ | 43 ⚠️ | PR pile-up; scope overlap with Doc Healer |
| Dead Code Removal Agent | 48 ⚠️ | 38 ⚠️ | 0% merge rate on open PRs |
| Contribution Check | 72 🔴 | 12 🔴 | Day 12 failures; content valid; post-output infra broken |
| Efficiency Improver | 20 🔴 | 10 🔴 | Day 7 failures; no outputs since May 31 |
| Code Refiner | 8 🔴 | 5 🔴 | Day 9+ all cancelled; disable immediately |

## PR Merge Rates (30 days)
- dependabot: 79% (11/14)
- veverkap: 83% (5/6)
- copilot-swe-agent: 53% (10/19) — acceptable
- amalgamated-bot: 22% (7/32) — low; dragged by Code Simplifier + Dead Code Remover (0% each)

## Key Changes vs June 8
- ✅ RECOVERED: Go Fan (now succeeded, issue #3205 created)
- ✅ RECOVERED: Daily Doc Healer (June 8 failure confirmed transient)
- ✅ RECOVERED: Daily Malicious Scan (June 8 failure confirmed transient)
- ✅ ACTIVE: Architecture Guardian (ran June 8; June 6-7 gap resolved)
- 🔴 ONGOING: Contribution Check day 12 (content valid but infra still broken)
- 🔴 ONGOING: Efficiency Improver day 7 (still failing)
- 🔴 ONGOING: Code Refiner day 9+ (still cancelled/skipped)
- ⚠️ WATCH: copilot-swe-agent 8 open PRs (accumulation risk)
- 🆕 STABLE: Typist - Go Type Analysis (day 2 success)

## Recommendations
1. 🔴 Fix Contribution Check post-output infra step (day 12, ~1-2h)
2. 🔴 Disable Code Refiner (day 9+ all-cancelled, pure CI waste)
3. 🔴 Diagnose Efficiency Improver regression (started June 3)
4. ⚠️ Monitor Go Fan + Architecture Guardian for recurrence
5. ⚠️ Gate Code Simplifier + Dead Code Remover on open PR count
6. ⚠️ Merge or close 8 copilot-swe-agent open PRs before adding more
