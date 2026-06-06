# Agent Performance 2026-06-06

**Run**: agent-performance-analyzer | 2026-06-06T13:21Z | [§27063408099](https://github.com/amalgamated-tools/biblioteka/actions/runs/27063408099)

## Summary
- 19 workflows analyzed | Quality: 62/100 (↑+4) | Effectiveness: 51/100 (↓-3)
- Ecosystem: 🟠 DEGRADED | Clean: 7/19 (37%)
- NEW CRITICAL: Contribution Check broken (5 failures; content valid, infra step failing)
- PERSISTENT: Code Refiner day 7 broken (12 runs/day, 0 productive)

## Agent Scores
| Agent | Quality | Notes |
|-------|---------|-------|
| Agentic Maintenance | 85 ✅ | 4/4 today |
| Go Fan | 85 ✅ | ⚠️ No run today at 13:21 UTC |
| Daily Testify Expert | 78 ✅ | NEW 5/5, issue #3173 |
| Daily Doc Healer | 76 ✅ | NEW 5/5, PR #3174 |
| Duplicate Code Detector | 74 ✅ | issue #3171 |
| PR Triage Agent | 72 ✅ | scoped correctly |
| Malicious Scan | 71 ✅ | 1/1 |
| Architecture Guardian | 65 ⚠️ | recovering, #3155 open |
| Code Simplifier | 53 ⚠️ | PR #3177 better quality, watch |
| Dead Code Remover | 48 ⚠️ | 0% merge, PR #3176 open |
| Daily Doc Updater | 45 ⚠️ | over-creation, 3 open PRs |
| Contribution Check | 10 🔴 | 5 failures (infra bug) |
| Code Refiner | 8 🔴 | day 7 skip/cancel loop |

## Key Changes vs June 5
- 🔴 NEW: Contribution Check broken (was #1 at 98/100)
- ✅ Caveman: PR #3161 merged
- ✅ New agents: Daily Doc Healer + Daily Testify Expert (both 5/5)
- ⚠️ Doc PR pile-up: #3162 + #3174 + #3175 open simultaneously

## Recommendations
1. 🔴 Fix Contribution Check — post-output step failing (1-2h fix)
2. 🔴 Disable Code Refiner — 7 days broken, CI waste
3. ⚠️ Gate Doc Updater on open PRs; scope vs Doc Healer
4. 👀 Monitor Go Fan (no run today)
5. 👀 Track Code Simplifier PR #3177
