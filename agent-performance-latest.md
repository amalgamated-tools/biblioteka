# Agent Performance — 2026-04-24
**Run:** 2026-04-24T23:44Z

## Snapshot
- 55 workflows registered, all Copilot; **repo-assist NOT compiled (6th day ⚠️ CRITICAL)**
- 44 open issues, estimated 8 new today
- 25 open PRs (all ready)
- [aw] Failures: **7 active** (stable, no new failures today)
- Status-as-issue pattern: **Day 5** — #2536, #2530, #2529 (+ NEW: #2528 chronicle adds to pattern)
- Groups.go refactoring duplication: **4 items** — issues #2486+#2520, PRs #2487+#2521
- Reading groups docs triple-duplicate PRs: #2418, #2480, #2481 (Day 3+)
- Open dependabot bundles stacking: 4 open (#2524, #2490, #2467, #2411)

## Top Performers
- **duplicate-code-detector**: #2538 (group membership dup) + PR #2539. Score: 88/100
- **glossary-maintainer**: #2535 clean glossary PR. Score: 78/100
- **unbloat-docs**: #2534 targeted docs cleanup. Score: 75/100
- **dependabot-pr-bundler**: #2524 clean deps bundle. Score: 73/100 (stacking issue)
- **tech-content-editorial-board**: #2532 security editorial. Score: 68/100

## Agents Needing Improvement
- **repo-assist**: NOT COMPILED + FAILED (#2389). Score: 10/100 — CRITICAL (6th day)
- **daily-repo-status**: Status-as-issue Day 5 (#2536). Score: 15/100
- **daily-team-status**: Status-as-issue Day 5 (#2530). Score: 15/100
- **daily-plan**: Status-as-issue Day 5 (#2529). Score: 20/100
- **daily-repo-chronicle**: Now creating issue #2528 (fixed failure but wrong output type). Score: 30/100
- **code-simplifier**: Groups.go duplication: issues #2486+#2520, PRs #2487+#2521. Score: 35/100
- **daily-doc-updater**: Triple reading groups docs PRs (#2418/#2480/#2481). Score: 40/100

## Improvements vs Yesterday (Apr 23)
- No new [aw] failures today ✅
- daily-repo-chronicle fixed its run failure (no new [aw] issue today) ✅
- sergo appears to have run (#2523 GA versions update) ✅

## Regressions vs Yesterday
- Status-as-issue pattern Day 5 (still no fix) ❌
- repo-assist NOT COMPILED Day 6 (escalating) ❌
- daily-repo-chronicle: fixed failure but now creates issues instead of discussions (#2528) — NEW type ❌
- Groups.go refactoring: new PR #2521 duplicates existing #2487 + new issue #2520 duplicates #2486 ❌
- Dependabot bundles stacking: 4 open bundle issues (#2524, #2490, #2467, #2411) ❌

## Critical Issues
1. CRITICAL: repo-assist NOT COMPILED (6th day) — #2389
2. CRITICAL: Status-as-Issue Pattern Day 5 — #2536, #2530, #2529, #2528 (+ historical: #2470, #2416, etc.)
3. HIGH: 7 active [aw] failures — #2494, #2442, #2441, #2440, #2405, #2390, #2389
4. HIGH: Groups.go refactor duplication (4 items) — #2486, #2520 (issues), #2487, #2521 (PRs)
5. HIGH: Triple duplicate reading groups docs PRs — #2418, #2480, #2481
6. MEDIUM: 4 open dependabot bundle issues stacking — #2524, #2490, #2467, #2411

## Discussions Created
"Agent Performance Report — 2026-04-24" in Audits category
