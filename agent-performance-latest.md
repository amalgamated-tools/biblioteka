# Agent Performance 2026-06-02

**Run**: agent-performance-analyzer | 2026-06-02T14:20Z | [§26825830691](https://github.com/amalgamated-tools/biblioteka/actions/runs/26825830691)

## Summary
- 15 workflows analyzed (13 established + 2 stabilizing)
- 6/15 clean agents (40%) — Contribution Check, Agentic Maintenance, Go Fan, Daily Testify Expert, Daily Malicious Code Scan, Typist
- Quality: 57/100 average (↓ -1 from June 1, 58)
- Effectiveness: 55/100 average (↓ -2)
- Pattern detection: 6 clean, 1 over-creation, 2 scope-creep, 1 repetition, 3 inconsistency, 2 offline, 1 under-creation

## PR Merge Rates (all-time)
- Efficiency Improver: 100% (2/2) ✅
- Testify Expert: 100% (3/3) ✅
- Go Fan: ~90% ✅
- Daily Documentation Updater: 50% ⚠️
- Function Namer: 50% ⚠️
- Caveman Optimizer: 25% 🔴 (3 open duplicate PRs as of today)
- Code Refiner: 0% (1 open unmerged) 🔴
- Code Simplifier: 0% (3 closed) 🔴
- Dead Code Remover: 0% (3 closed including #3142 new today) 🔴

## Ecosystem Health: 🟡 MODERATE–DECLINING (Day 8-12, Caveman Worsening)
- Caveman Optimizer: DETERIORATING — now 3 duplicate open PRs (#3123, #3131, #3138)
- Architecture Guardian: STILL NO RUNS (0 in last 7+ days — likely stalled)
- Code Refiner: STILL BROKEN (all cancelled/skipped, #3119 unmerged 7+ days)
- Dead Code Remover: NEW PR #3142 today — likely same rejection pattern (0% all-time)
- Code Simplifier: #3126 CLOSED — 0% merge rate persists

## Top Performers
1. **Contribution Check**: 98/100, 100% success, every 4h ✅
2. **Go Fan**: 85/100, 100% success, 90% PR merge rate ✅
3. **Daily Testify Expert**: 82/100, 100% PR merge rate ✅
4. **Agentic Maintenance**: 80/100, 100% success ✅
5. **Daily Malicious Code Scan**: 78/100, 100% success ✅

## Critical Issues
- 🚨 **Caveman Optimizer**: 3 simultaneous duplicate PRs (#3123, #3131, #3138) — WORSENING
- 🚨 **Architecture Guardian**: 0 runs last 7 days — stalled/offline
- 🚨 **Code Refiner**: 0% success, all cancelled/skipped, functionally offline
- 🚨 **Code Simplifier**: 0% PR merge rate — scope-creep (3rd rejection)
- 🚨 **Dead Code Remover**: 0% PR merge rate — scope-creep (#3142 new, likely rejected)

## Actions This Run
- Agent Performance report issue created (Week of June 2, 2026)
- Shared memory updated

## Recommendations for Human Action
1. 🚨 CRITICAL: Close 2 of 3 Caveman duplicate PRs (#3123, #3131 — keep #3138 newest)
2. 🚨 CRITICAL: Fix Architecture Guardian / confirm if retired
3. 🚨 CRITICAL: Investigate Code Refiner cancellation root cause
4. 🚨 Review Code Simplifier + Dead Code Remover scope — adjust prompts or disable
