# Agent Performance 2026-06-05

**Run**: agent-performance-analyzer | 2026-06-05T13:52Z | [§27018816726](https://github.com/amalgamated-tools/biblioteka/actions/runs/27018816726)

## Summary
- 15 workflows analyzed
- 5/15 clean agents (33%) — Contribution Check, Go Fan, Agentic Maintenance, Typist, Duplicate Code Detector
- Quality: 58/100 average (↓ -1 from June 4 — Architecture Guardian failure)
- Effectiveness: 54/100 average (↓ -3 from June 4)
- Key concern: Code Refiner now 6 days broken (all-skip)
- New observation: Architecture Guardian issue #3155 (failure) still open

## PR Merge Rates (updated)
- Go Fan: ~90% ✅
- Contribution Check: N/A (issues only) ✅
- Agentic Maintenance: N/A (no PR outputs) ✅
- Caveman Optimizer: ~38% ⚠️ — new PR #3161 open (re-targeting contribution-check again)
- Daily Doc Updater: ~33% ⚠️ — PR #3162 open today
- Code Simplifier: 0% 🔴 (2+ rejected PRs)
- Dead Code Remover: 0% 🔴 (2 rejected PRs)
- Code Refiner: 0% 🔴 (PR #3119 stale 6 days, all runs skipped)

## Ecosystem Health: 🟡 MODERATE — SLIGHT DECLINE
- ✅ Caveman Optimizer: duplicate crisis still resolved, but #3161 is new risk
- ⚠️ Architecture Guardian: issue #3155 "failed" still open (new June 4)
- 🔴 Code Refiner: all-skip pattern day 6 (PR #3119 now 6 days stale)
- 🔴 Code Simplifier: 0% merge rate persists
- 🔴 Dead Code Remover: 0% merge rate, previous PRs closed
- 🔴 Metrics Collector: STALE 62 days (last: 2026-04-04)

## Top Performers
1. **Contribution Check**: 98/100 ✅ (issue #3157 today)
2. **Go Fan**: 85/100 ✅ (issue #3165 today)
3. **Agentic Maintenance**: 82/100 ✅ (4/4 runs today)
4. **Typist**: 78/100 ✅
5. **Duplicate Code Detector**: 72/100 ✅ (issues #3158, #3159 today)

## Changes (June 4 → June 5)
- 🔴 Code Refiner: now 6 days broken (no change)
- ⚠️ Architecture Guardian: issue #3155 still open (no fix yet)
- ⚠️ Caveman Optimizer: new PR #3161 — same area as previously rejected #3131
- ↔️ Daily Doc Updater: new PR #3162 (over-creation continues)
- ↔️ Dead Code Remover: new run success, but PR outcome TBD

## Reactive Agents (healthy skips)
- PR Code Quality Reviewer: 10/10 skip (expected — no qualifying PRs)
- Q: 12/12 skip (slash command, not invoked)
- Mergefest: 10/10 skip (no qualifying PRs)
- CLA Assistant: 7/7 skip (no triggers)

## Actions This Run
- Agent Performance issue created (Week of June 5, 2026)
- Shared memory updated

## Recommendations for Human Action
1. 🔴 Fix or retire Code Refiner (6-day broken streak, PR #3119 stale)
2. 🔴 Narrow Code Simplifier scope (0% merge rate, waste of review time)
3. 🔴 Narrow Dead Code Remover scope (0% merge rate)
4. 🔴 Fix Metrics Collector (62 days stale — observability gap)
5. ⚠️ Investigate Architecture Guardian failure (#3155)
6. ⚠️ Reduce Daily Doc Updater PR volume (16 PRs/run, 33% merge rate)
7. ⚠️ Monitor Caveman PR #3161 (risk of new duplicate cycle)

## Pattern Detector Results (confirmed)
- Code Refiner: under-creation + inconsistency (regression from prior productivity)
- Code Simplifier: over-creation + scope-creep
- Dead Code Removal Agent: over-creation
- Daily Documentation Updater: over-creation
- Caveman Optimizer: repetition (re-targeting same area as rejected #3131)
- Architecture Guardian: inconsistency (intermittent failures)
- Contribution Check, Agentic Maintenance, Go Fan, Typist, Duplicate Code Detector: clean (no patterns)
- PR Code Quality Reviewer, Q, Mergefest, Metrics Collector: under-creation (dormant)
