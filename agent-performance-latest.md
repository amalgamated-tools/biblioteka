# Agent Performance 2026-05-27

**Run**: agent-performance-analyzer | 2026-05-27T14:04Z

## Summary
- 22 workflows total (17 agent workflows analyzed, 5 utility/trigger)
- 12/17 active agents (71%) - slight improvement from 59% on May 24
- Quality: 61.2/100 (↓ -2.3 from May 24)
- Effectiveness: 60.8/100 (↓ -3.3 from May 24)
- Total outputs: 32 (issues: 26, PRs: 6; ↓ -66% from 95 last week)
- Issue close rate: 69.2% (18/26)
- PR merge rate: 50.0% (3/6; ↓ -18.8pp from 68.8% May 24)

## Ecosystem Health: 🔴 DECLINING (Worsening)
- Quality down 2.3 points in 2 days
- Effectiveness down 3.3 points in 2 days
- Output volume down 66%
- PR merge rate down 18.8pp (vs April 4 baseline: down 34pp to 50% from 84%)
- **3 agents completely non-functional with NO PROGRESS on fixes**

## Pattern Detection Results
- **clean**: 8 agents (47%) - Contribution Check, Copilot SWE Agent, Agent Performance Analyzer, Go Fan (recovered), Doc Healer, Duplicate Code Detector, Daily Doc Updater, Malicious Code Scanner
- **under_creation**: 8 agents (47%) - Architecture Guardian (16+ days), Code Simplifier (4 days), Security Red Team (2 days), Dead Code Remover, Function Namer, File Diet, Go Pattern Detector
- **repetition**: 1 agent - Testify Expert (pathparser_test.go)
- **over_creation**: 1 agent - Caveman Optimizer (3 PRs, 0 merged)
- **scope_creep**: 1 agent - Function Namer (3/3 issues closed wontfix)
- **inconsistency**: 3 agents - Caveman Optimizer, Function Namer, Code Simplifier

## Top Performers (Stable)
- **Contribution Check**: 95/100 quality, 98/100 effectiveness (~100% close rate)
- **Copilot SWE Agent**: 90/100 quality, 92/100 effectiveness (67% merge rate)
- **Agent Performance Analyzer**: 85/100 quality, 90/100 effectiveness (meta-monitoring)
- **Go Fan**: 85/100 quality, 88/100 effectiveness (80% close rate, recovered from over-creation)

## Critical Non-Functional Agents (NO PROGRESS - Day 3)
- 🚨 **Architecture Guardian**: OFFLINE 16+ days (#3033 open 8+ days, **NO ACTION**)
- 🚨 **Code Simplifier**: Silent failure day 4 (#3064 open 2 days, **NO ACTION**)
- 🚨 **Security Red Team**: Config error (#3066 open 2 days, **NO ACTION**)

## Underperformers (Fixes Awaiting Implementation)
- **Dead Code Remover**: 35/100 quality, 30/100 effectiveness (0% merge rate) - needs investigation
- **Function Namer**: 40/100 quality, 45/100 effectiveness (inactive, #3063/#3065 pending)
- **Caveman Optimizer**: 45/100 quality, 40/100 effectiveness (0/3 PRs merged, #3062 pending)
- **Testify Expert**: 70/100 quality, 65/100 effectiveness (repetition, #3052 dedup needed)

## Actions This Run
- ✅ Created comprehensive discussion: "Agent Performance Report - Week of May 20-27, 2026"
- ✅ Pattern detection completed (8 clean, 8 under-creation, 3 inconsistency, 1 repetition, 1 over-creation)
- ✅ No new improvement issues (5 from May 24 still awaiting implementation)
- ✅ Shared alerts updated with May 27 status

## Recommendations for Human Action (UNCHANGED - URGENT)
1. 🚨 **CRITICAL**: Fix Architecture Guardian (#3033) - 16+ days offline, requires gh aw compile
2. 🚨 **CRITICAL**: Debug Code Simplifier (#3064) - 4 days silent failure
3. 🚨 **CRITICAL**: Fix Security Red Team config (#3066) - target directories not found
4. 🚨 **CRITICAL**: Trigger Metrics Collector - 52 days stale (last: April 4)
5. ⚠️ **HIGH**: Implement pending fixes (#3062, #3063, #3065)
6. 📝 Investigate Dead Code Remover detection (0% merge rate)
7. 📝 Add Testify Expert deduplication (#3052)
8. 📝 Close stale recovery issues (#3022, #3031)

## Ecosystem Trends (May 24 → May 27)
- Quality: 61.2/100 (↓ -2.3 from 63.5)
- Effectiveness: 60.8/100 (↓ -3.3 from 64.1)
- Output volume: 32 (↓ -66% from 95)
- Active agents: 12/17 = 71% (↑ +12pp from 59%)
- PR merge rate: 50% (↓ -18.8pp from 68.8%; vs baseline -34pp from 84%)
- Issue close rate: 69.2% (→ stable)

**Decline drivers:**
1. Three agents completely offline (Architecture Guardian 16+d, Code Simplifier 4d, Security Red Team 2d)
2. Low performers not yet fixed (pending implementation of May 24 issues)
3. Metrics 52 days stale preventing accurate trend analysis
4. No human action on critical issues despite escalation

**Positive signals:**
- Top performers stable (8 clean agents at 47%)
- Issue close rate holding ~70%
- Slightly more agents active (71% vs 59%)
- No new agent failures in past 2 days
