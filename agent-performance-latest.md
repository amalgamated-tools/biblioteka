# Agent Performance 2026-05-09

**Run**: agent-performance-analyzer | 2026-05-09T13:20Z

## Summary
- 27 distinct workflows active in 7-day window (2026-05-02 to 2026-05-09)
- 100 runs analyzed: ~16 workflows succeeded (59%), 5 failed outright, 6 workflows with 100% skip rates
- vs prior report (2026-05-05): overall success rate slightly improved; cancellation storm appears resolved
- Concurrency cascade (prior CRIT): no new evidence of 8+ simultaneous cancellation bursts this week

## Top Performers
- **Agentic Maintenance**: 5/5 success (100%) — consistent, reliable
- **CodeQL**: 6/6 success (100%) — security scanning solid
- **Release Please**: 3/3 success (100%)
- **Dead Code Removal Agent**: 2/2 success (100%)
- **Daily security/quality scans** (Malicious Code, Security Red Team, Testify Expert, Go Function Namer): 1/1 each (100%)

## Underperformers
- **Contribution Check**: 0/3 (100% failure) — ongoing issue from prior report, still not fixed
- **Duplicate Code Detector**: 0/1 failed
- **Daily File Diet**: 0/1 failed — issue #2958 open
- **Go Fan**: 0/1 failed
- **Go Pattern Detector**: 0/1 failed
- **Q**: 0/22 (100% skipped — never produces output)
- **Mergefest + PR Code Quality Reviewer**: 0/12 each (100% skipped)
- **Daily Documentation Updater**: 1/1 success but created 16 PRs (over-creation concern)

## Key Findings
1. **HIGH - Contribution Check still failing**: 3 failures this week, open issue #2959 — no fix applied
2. **HIGH - 5 workflows at 100% failure rate**: Duplicate Code Detector, File Diet, Go Fan, Go Pattern Detector, Contribution Check
3. **MED - Q/Mergefest/PR Code Quality Reviewer permanently skipped**: 22+12+12 = 46 no-op runs consuming GitHub Actions minutes
4. **MED - Daily Doc Updater over-creation**: 16 PRs/run (flagged previously, still unaddressed per shared alerts)
5. **LOW - Concurrency cascade**: No new evidence this week; likely resolved or PR load reduced

## Discussion Created
- Yes (this run)

## Issues Created
- None new (existing issues #2958, #2959 already track key failures)
