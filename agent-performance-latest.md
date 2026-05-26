# Agent Performance 2026-05-26

**Run**: agent-performance-analyzer | 2026-05-26T13:52Z

## Summary
- 17 agents analyzed (12 active, 5 non-functional/inactive)
- Quality: 61.2/100 (↓ -2.3 from 63/100 on May 24)
- Effectiveness: 60.8/100 (↓ -3.3 from 61/100 on May 24)  
- Total outputs: 32 (issues: 26, PRs: 6; ↓ -66% from 95 last week)
- Issue close rate: 69.2% (18/26)
- PR merge rate: 50.0% (3/6; ↓ from 68.8% last week)

## Ecosystem Health: 🔴 DECLINING
- Quality down 2.3 points in 2 days
- Effectiveness down 3.3 points in 2 days
- Output volume down 66% (95 → 32)
- PR merge rate down 18.8 percentage points
- 3 agents completely non-functional (Architecture Guardian, Code Simplifier, Security Red Team)

## Pattern Detector Results
- **under_creation**: 3 (Code Simplifier, Architecture Guardian, Dead Code Remover)
- **repetition**: 1 (Testify Expert - pathparser_test.go)
- **inconsistency**: 2 (Caveman Optimizer, Function Namer)
- **clean**: 5 (Go Fan, Duplicate Code Detector, Contribution Check, Copilot SWE Agent, Doc Healer)

## Top Performers (unchanged from May 25)
- **Contribution Check**: 95/100 quality, 98/100 effectiveness (~100% close rate)
- **Copilot SWE Agent**: 90/100 quality, 92/100 effectiveness (67% merge rate, high-quality refactoring)
- **Agent Performance Analyzer**: 85/100 quality, 90/100 effectiveness (meta-monitoring)
- **Go Fan**: 85/100 quality, 88/100 effectiveness (80% close rate)

## Critical Non-Functional Agents (UNCHANGED - No Progress)
- 🚨 **Architecture Guardian**: OFFLINE 16+ days (#3033 open 8+ days, no action)
- 🚨 **Code Simplifier**: Silent failure day 4 (#3064 open 2 days, no action)
- 🚨 **Security Red Team**: Config error (#3066 open 2 days, no action)

## Underperformers
- **Dead Code Remover**: 35/100 quality, 30/100 effectiveness (0% merge rate)
- **Function Namer**: 40/100 quality, 45/100 effectiveness (inactive, fixes pending #3063, #3065)
- **Caveman Optimizer**: 45/100 quality, 40/100 effectiveness (50% recent, fix pending #3062)

## Issues Created This Run
- None (created comprehensive discussion report instead)
- Tracking implementation of May 24 improvement issues (#3062-#3065)

## Recommendations for Human Action (URGENT)
1. 🚨 **CRITICAL**: Fix Architecture Guardian (#3033) — offline 16+ days, requires gh aw compile
2. 🚨 **CRITICAL**: Debug Code Simplifier (#3064) — 4 days silent failure
3. 🚨 **CRITICAL**: Fix Security Red Team config (#3066) — target directories not found
4. 🚨 **CRITICAL**: Trigger Metrics Collector — 52 days stale (last: April 4)
5. ⚠️ **HIGH**: Implement pending fixes (#3062, #3063, #3065)
6. 📝 Review: Dead Code Remover detection (0% merge rate)
7. 📝 Close: Stale recovery issues #3022, #3031

## Ecosystem Trends (2-day comparison)
- Quality: 61.2/100 (↓ -2.3 from May 24)
- Effectiveness: 60.8/100 (↓ -3.3 from May 24)
- Output volume: 32 (↓ -66% from 95 on May 24)
- Active agents: 12/17 (71%) vs 10/17 (59%) on May 24 — slight improvement
- Critical offline: 3 agents (unchanged)
- PR merge rate: 50% (↓ -18.8pp from 68.8%)

**Primary decline drivers:**
1. Three agents completely non-functional (Architecture Guardian, Code Simplifier, Security Red Team)
2. Low-performing agents not yet fixed (Dead Code Remover, Caveman Optimizer, Function Namer)
3. Significant output volume drop suggests possible GitHub API issues or agent execution problems

**Positive signals:**
- Top performers remain stable (Contribution Check, Copilot SWE Agent, Go Fan)
- Issue close rate holding at ~70%
- Two more agents now active (12 vs 10)
