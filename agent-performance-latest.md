# Agent Performance 2026-05-28

**Run**: agent-performance-analyzer | 2026-05-28T14:09Z | [§26579852725](https://github.com/amalgamated-tools/biblioteka/actions/runs/26579852725)

## Summary
- 19 workflows total (17 agent workflows, 2 utility/meta)
- 12/19 clean agents (63%) - significant improvement from 47% on May 27
- Quality: 63% excellent (agents scoring 80-100)
- Effectiveness: Strong top performers, 3 critical non-functional agents
- Last 7 days: 44 runs, 12.4M tokens, 174M effective tokens, 100% from Copilot engine
- Issue close rate: 69.2% (stable)
- PR merge rate: 50.0% (declining from 68.8% May 24, baseline 84% Apr 4)

## Ecosystem Health: 🟡 MODERATE (Improving but Critical Issues Persist)
- Clean percentage: 63% (↑ +16pp from May 27's 47%)
- 15/19 agents with 100% success rate in last 7 days
- Top performers stable and exemplary
- **3 CRITICAL non-functional agents persist with NO PROGRESS**

## Pattern Detection Results (May 21-28, 2026)
- **clean**: 12 agents (63%) - Contribution Check, PR Triage, Daily Doc Updater, Go Fan (recovered), Daily Malicious Code Scan, Daily Doc Healer, Duplicate Code Detector, Typist, Daily File Diet, Go Pattern Detector, Security Red Team, Agent Performance Analyzer
- **silent_failure**: 2 agents (11%) - Code Simplifier (day 5), Dead Code Remover (0% merge rate)
- **under_creation**: 2 agents (11%) - Daily Caveman Optimizer (0/3 PRs merged), Daily Go Function Namer (3/3 wontfix)
- **repetition**: 1 agent (5%) - Testify Expert (pathparser_test.go)
- **scope_creep**: 1 agent (5%) - Function Namer
- **inconsistency**: 1 agent (5%) - Code Refiner (3/3 cancelled)
- **offline**: 1 agent (5%) - Architecture Guardian (16+ days)

## Top Performers (Stable & Exemplary)
1. **Contribution Check**: 98/100 quality, 98/100 effectiveness (10 runs, 100% success, every 4h)
2. **Daily Documentation Updater**: 90/100 quality, 88/100 effectiveness (perfect success, 866k tokens)
3. **Daily Documentation Healer**: 88/100 quality, 86/100 effectiveness (perfect success, #3022 resolved)
4. **Go Fan**: 85/100 quality, 88/100 effectiveness (RECOVERED from over-creation, 1.44M tokens)
5. **Typist - Go Type Analysis**: 85/100 quality, 85/100 effectiveness (1.58M tokens, comprehensive)

## Critical Non-Functional Agents (NO PROGRESS - Day 4)
- 🚨 **Architecture Guardian**: OFFLINE 16+ days (#3033 open 9+ days, **ZERO HUMAN ACTION**)
- 🚨 **Code Simplifier**: Silent failure day 5 (#3064 open 3 days, **ZERO HUMAN ACTION**)
- 🚨 **Dead Code Remover**: 0% merge rate (0/1 PRs), functionally inactive, **NO ISSUE CREATED**

## Underperformers (Fixes Awaiting Implementation)
- **Function Namer**: 40/100 quality, 45/100 effectiveness (3/3 wontfix, #3063/#3065 pending)
- **Caveman Optimizer**: 45/100 quality, 40/100 effectiveness (0/3 PRs merged, #3062 pending)
- **Testify Expert**: 70/100 quality, 65/100 effectiveness (repetition, #3052 dedup needed)
- **Code Refiner**: 0/100 (3/3 cancelled, config/trigger issues)
- **Security Red Team**: 75/100 quality, 70/100 effectiveness (running but #3066 config error)

## Actions This Run
- ✅ Comprehensive Agent Performance Report discussion created
- ✅ Pattern detection completed (12 clean, 2 silent failure, 2 under-creation, 3 other patterns)
- ✅ 19 agents analyzed across 7 days (May 21-28)
- ✅ Detailed rankings, quality analysis, effectiveness measurement
- ✅ 13 prioritized recommendations (4 critical, 3 high, 2 medium, 4 strategic)
- ✅ Shared alerts updated with May 28 status

## Recommendations for Human Action (ESCALATED - CRITICAL)
1. 🚨 **CRITICAL**: Fix Architecture Guardian (#3033) - 16+ days offline, run `gh aw compile`
2. 🚨 **CRITICAL**: Debug Code Simplifier (#3064) - day 5 silent failure, add logging
3. 🚨 **CRITICAL**: Investigate Dead Code Remover - 0% merge rate, create issue
4. 🚨 **CRITICAL**: Trigger Metrics Collector - 52+ days stale (last: April 4)
5. ⚠️ **HIGH**: Implement pending fixes (#3062, #3063, #3065, #3066)
6. 📝 Close stale recovery issues (#3022, #3031)
7. 📝 Add Testify Expert deduplication (#3052)
8. 📝 Debug Code Refiner cancellation pattern

## Ecosystem Trends (May 27 → May 28)
- Clean percentage: 63% (↑ +16pp from 47%) ✅ MAJOR IMPROVEMENT
- Quality: 63% excellent (80-100 score range)
- Effectiveness: Top 8 agents maintaining excellence
- Success rate: 15/19 agents with 100% success (7 days)
- PR merge rate: 50% (↓ -18.8pp from May 24; -34pp from Apr 4 baseline) ⚠️
- Issue close rate: 69.2% (→ stable)
- Silent failures: 2 agents (new concern)

**Improvement drivers:**
1. Go Fan recovered from over-creation to clean status (+1 clean)
2. Most agents showing 100% success rates in last 7 days
3. Top performers maintaining excellent scores
4. Pattern detection showing 63% clean (vs 47% yesterday)

**Decline drivers:**
1. Three critical agents still offline/non-functional (Architecture Guardian 16+d, Code Simplifier 5d, Dead Code Remover inactive)
2. PR merge rate continuing to decline (-18.8pp in 3 days, -34pp from baseline)
3. Silent failure pattern in 2 agents (new)
4. Zero implementation progress on 5 improvement issues from May 24-27
5. Metrics Collector still stale 52+ days

**Positive signals:**
- Major improvement in clean percentage (+16pp)
- 15/19 agents with perfect success rates
- Top performers exemplary and stable
- Go Fan recovery demonstrates pattern correction works
- High-frequency agents (Contribution Check, PR Triage) maintaining reliability
