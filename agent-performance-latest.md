# Agent Performance — 2026-05-02 | 23:45Z

## Summary
55 workflows configured | 50 runs (7d) | 37 infrastructure failures (May 1-2) | 8 open PRs | Day 13 repo-assist uncompiled

## Top Performers
Agentic Triage 90 | Contribution Guidelines Checker 88 | Duplicate Code Detector 85 | Glossary Maintainer 82 | Repository Quality Improver 80 | Metrics Collector 80

## Critical Issues
- ENGINE OUTAGE (May 1-2): copilot_driver.cjs missing caused 37 failures across 34 workflows — infrastructure issue, not agent quality
- repo-assist: NOT compiled Day 13, all runs cancelled/failing (#2600)
- Status-as-Issue Day 13: daily-repo-status #2786, daily-team-status #2779, daily-plan #2778
- PR-triggered cancellation cascade: PR Nitpick (0%), Grumpy Reviewer (0%), Daily Perf Improver (17%) — concurrency policy
- Update Docs: ~40% failure rate, 3.7M tokens on failed runs (#2674)

## Quality Outputs This Week
- Duplicate Code Detector: #2787 (integer param parsing dup) + PR #2788 ✅
- Repository Quality Improver: #2706 (mutation auditability, useful)
- Daily Perf Improver: #2780 (BenchmarkGetPendingAIEnrichmentByBook)
- Documentation Unbloat: #2782 (condensed SSRF bullets)
- Update Docs: #2785, #2776, #2777 (audit action docs)
- Daily Documentation Updater: 16 PRs in April period

## vs Apr 30
- [aw] failures: 1→37 ❌ (but 95% caused by infrastructure outage, not agent quality)
- repo-assist: 12→13 days uncompiled ❌
- status-as-issue: 11→13 ❌
- PR-triggered concurrency: unresolved ❌
- Duplicate Code Detector: new quality output ✅

## Resource Concerns
- Schema Consistency Checker: 5.2M tokens/single run (needs scoping)
- Update Docs: 3.7M tokens on failed run

## Discussion Created
"Agent Performance Report — 2026-05-02" (Audits)
