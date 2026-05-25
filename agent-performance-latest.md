# Agent Performance 2026-05-25

**Run**: agent-performance-analyzer | 2026-05-25T13:54Z

## Summary
- 17 workflows analyzed (10 active, 7 inactive)
- Quality: 60.7/100 (↓ -2.3 from 63/100 yesterday)
- Effectiveness: 57.7/100 (↓ -3.3 from 61/100 yesterday)
- Total outputs: 95 (issues: 63, PRs: 32, merge rate: 68.8%)

## Key Changes from 2026-05-24
- **Overall ecosystem health declining**: Both quality and effectiveness down ~3 points
- **Architecture Guardian**: Still OFFLINE 16+ days (was 15+); #3033 remains OPEN (CRITICAL)
- **Code Simplifier**: Still silent (4th day); #3064 investigation pending
- **Function Namer**: Still inactive; #3063, #3065 fixes pending
- **Caveman Optimizer**: Still low merge rate (22.2%); #3062 fix pending
- **Dead Code Removal**: New underperformer identified (0% merge rate)
- **Security Red Team**: Configuration issue detected (#3066)

## Pattern Detector Results
- **over_creation**: 1 (Caveman Optimizer - 78% rejection rate)
- **under_creation**: 6 (Architecture Guardian offline, Code Simplifier silent, Function Namer inactive, Dead Code low output, Duplicate Code low output, Security Red Team config error)
- **repetition**: 1 (Testify Expert - pathparser_test.go)
- **scope_creep**: 1 (Function Namer - historical)
- **inconsistency**: 2 (Code Simplifier erratic, Performance Analyzer meta behavior)
- **clean**: 3 (Copilot SWE, Contribution Check, Go Fan)

## Top Performers (unchanged)
- **Copilot SWE Agent**: 90.9% merge rate, 20/22 PRs merged
- **Contribution Check**: 94.1% close rate, consistent daily reports
- **Daily Testify Expert**: 94.1% close rate, high-value issues (minor repetition)
- **Go Fan**: 90.9% close rate, pattern resolved

## Underperformers
- **Architecture Guardian**: OFFLINE 16+ days (CRITICAL, #3033)
- **Code Simplifier**: 4th day silent failure (#3064)
- **Caveman Optimizer**: 22.2% merge rate, over-production (#3062)
- **Function Namer**: Inactive after scope issues (#3063, #3065)
- **Dead Code Removal**: 0% merge rate (0/1 PRs merged)
- **Security Red Team**: Configuration error (#3066)

## Issues Created This Run
- None (meta-orchestrator created 4 issues yesterday: #3062-3065)
- Creating performance report discussion instead

## Recommendations for Human Action
1. 🚨 URGENT: Fix Architecture Guardian (16+ days offline, requires gh aw compile)
2. 🚨 URGENT: Address improvement issues #3062-3065 (created yesterday)
3. 🚨 URGENT: Trigger Metrics Collector manually (51 days stale)
4. Review: Dead Code Removal accuracy (0% merge suggests false positives)
5. Review: Security Red Team configuration (#3066)

## Ecosystem Trends
- Quality declining: 60.7/100 (↓ -2.3)
- Effectiveness declining: 57.7/100 (↓ -3.3)
- Output volume stable: 95 outputs
- Active agents: 10/17 (59%)
- Inactive agents: 7/17 (41% - too high)
