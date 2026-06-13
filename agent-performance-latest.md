# Agent Performance 2026-06-13

**Run**: agent-performance-analyzer | 2026-06-13T13:24Z | [§27467955623](https://github.com/amalgamated-tools/biblioteka/actions/runs/27467955623)

## Summary
- 18 workflows analyzed | Quality: 60/100 (↓-4) | Effectiveness: 52/100 (↓-4)
- Ecosystem: ⚠️ MIXED - 7 healthy, 6 warning, 3 critical, 1 self (APAz failed 3d)
- Healthy: 7/18 (39%, ↓ from 53%)
- NEW AGENTS: Go Pattern Detector (daily, succeeding), Goal (approval-gated), Archie (dead)
- ECOSYSTEM RESTRUCTURING: 6 Daily agents renamed Weekly (transition causing disruption)

## Agent Scores
| Agent | Quality | Effectiveness | Notes |
|-------|---------|--------------|-------|
| Agentic Maintenance | 92 ✅ | 96 ✅ | 8/8 runs, multiple/day, perfect |
| Daily Documentation Healer | 80 ✅ | 78 ✅ | 5/5 ok, stable |
| PR Triage Agent | 78 ✅ | 76 ✅ | 5/5 ok, stable |
| Duplicate Code Detector | 76 ✅ | 76 ✅ | 5/5 ok, pipeline healthy |
| Typist - Go Type Analysis | 72 ✅ | 68 ✅ | 5/5 ok, day 6, establishing baseline |
| Go Pattern Detector | 70 🆕 | 67 🆕 | 6/6 ok, new, function-rename issues |
| Go Fan | 68 ✅ | 60 ✅ | 5/5 ok, fully recovered |
| copilot-swe-agent | 70 ⚠️ | 50 ⚠️ | 12 open PRs, 43% merge (↓ from 53%) |
| Architecture Guardian | 65 ⚠️ | 60 ⚠️ | REGRESSION: failed June 12 (4/5 ok) |
| Code Simplifier | 55 ⚠️ | 45 ⚠️ | 5/5 ok but 0% merge rate |
| Agent Performance Analyzer | 50 ⚠️ | 45 ⚠️ | SELF: 3 failures June 10-12 |
| Daily Documentation Updater | 48 ⚠️ | 42 ⚠️ | PR pile-up, scope overlap |
| Dead Code Removal Agent | 45 ⚠️ | 35 ⚠️ | INTERMITTENT: 3/5 ok, 2 fails |
| Efficiency Improver | 20 🔴 | 8 🔴 | DAY 13 failures; no outputs since May 31 |
| Contribution Check | 68 🔴 | 15 🔴 | DAY 17+; 1 success June 13; mostly failing |
| Code Refiner | 5 🔴 | 3 🔴 | DAY 17+ all cancelled; DISABLE IMMEDIATELY |

## Ecosystem Changes (June 9-13)
- 6 agents renamed Daily→Weekly: Caveman Optimizer, Security Red Team, Malicious Code Scan,
  Testify Expert, File Diet, Go Function Namer
- Weekly versions have 0 runs (not yet triggered on weekly schedule)
- Transition causing [aw] failure issues for "Daily" named versions

## PR Merge Rates (30d)
- dependabot: 72% (healthy)
- veverkap: 67% (healthy)
- copilot-swe-agent: 43% (↓ from 53%) — 12 open PRs backlog
- amalgamated-bot: 19% (↓ from 22%) — alarming; dragged by 0% merge agents

## Key Changes vs June 9
- ↓ APAz (me) failed 3 consecutive days June 10-12 (issues #3218, #3232)
- ↓ Architecture Guardian regressed again June 12 (issue #3252)
- ↓ Dead Code Removal Agent: new intermittent failures June 11-12 (issue #3250)
- ↓ copilot-swe-agent: 8→12 open PRs, declining merge rate
- ↓ Daily Security Red Team Agent: failed June 10-11 (issues #3222, #3233)
- ↓ Daily Testify Expert: failed June 12 (issue #3242)
- 🆕 Go Pattern Detector: new daily agent, 6/6 successes, function-rename plan issues
- 🆕 Goal workflow: new, waiting for approval
- ✅ Contribution Check: 1 success today (June 13 01:26 UTC) - possible recovery

## Recommendations
1. 🔴 Disable Code Refiner (day 17+ all-cancelled, pure CI waste)
2. 🔴 Diagnose Efficiency Improver regression (started June 3, now day 13)
3. 🔴 Fix Contribution Check post-output infra (day 17+ mostly failing)
4. ⚠️ Gate copilot-swe-agent: block new PRs until backlog < 5 open
5. ⚠️ Gate Code Simplifier + Dead Code Remover on open PR count (currently 0% merge)
6. ⚠️ Monitor Architecture Guardian for recurrence (failed again June 12)
7. ⚠️ Triage Weekly agent transition: verify all 6 renamed agents trigger correctly
8. ⚠️ Investigate APAz own failures (issues #3218, #3232)
