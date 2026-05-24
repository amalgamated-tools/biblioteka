# Agent Performance 2026-05-24

**Run**: agent-performance-analyzer | 2026-05-24T13:21Z

## Summary
- 14 workflows analyzed (10 active, 4 event-triggered)
- Quality: 63/100 (stable from yesterday)
- Effectiveness: 61/100 (stable from yesterday)
- 9 behavioral patterns across 5 agents

## Key Changes from 2026-05-23
- **Architecture Guardian**: Now 15+ days offline (was 14+); #3033 remains OPEN (CRITICAL)
- **Go Fan**: Confirmed stable - no new issue today (monitor period complete)
- **Function Namer**: 3 improvement issues created (scope restriction)
- **Caveman Optimizer**: 1 improvement issue created (output validation)
- **Code Simplifier**: 1 investigation issue created (silent failure, 3rd day)
- **Testify Expert**: Stable at partial recovery level
- **Metrics Collector**: Still stale 49 days (needs manual trigger)

## Pattern Detector Results
- over_creation: 2 active (Caveman, Function Namer; Go Fan resolved)
- repetition: 1 (Testify Expert)
- inconsistency: 1 (Testify Expert; PR Triage not re-analyzed)
- under_creation: 2 (Architecture Guardian offline, Code Simplifier silent)
- scope_creep: 1 (Function Namer)
- clean: 8 agents (↑1 from yesterday - Go Fan resolved)

## Top Performers (unchanged)
- **Agentic Maintenance**: 100% success, backbone agent
- **Duplicate Code Detector**: High-value outputs, triggers downstream PRs
- **Contribution Check**: 100% clean reports
- **Dead Code Removal**: Quality PRs, consistent

## Underperformers
- **Architecture Guardian**: OFFLINE 15+ days (CRITICAL, #3033)
- **Function Namer**: 100% rejection rate, scope creep (issues created today)
- **Caveman Optimizer**: 0% merge rate, over-production (issue created today)
- **Code Simplifier**: 3rd day silent failure (issue created today)
- **Testify Expert**: Repetition pattern (stable at partial recovery)

## Issues Created This Run
- Function Namer scope restriction (3 separate issues with examples)
- Caveman Optimizer output validation gate
- Code Simplifier silent failure investigation
- Stale issue cleanup recommendations (#3022, #3031)

## Recommendations for Human Action
1. URGENT: Fix Architecture Guardian (15+ days offline, requires gh aw compile)
2. URGENT: Trigger Metrics Collector manually (49 days stale)
3. Review and merge: Function Namer scope restriction fixes
4. Review and merge: Caveman Optimizer validation gate
5. Close recovered issues: #3022, #3031

