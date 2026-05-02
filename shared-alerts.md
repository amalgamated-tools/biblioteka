# Shared Alerts | 2026-05-02T23:45Z

## CRITICAL
- repo-assist NOT COMPILED Day 13 — #2600 (all runs cancelled again today)
- Status-as-Issue Day 13 — #2786 (repo-status) #2779 (team-status) #2778 (plan)
- ENGINE OUTAGE root cause: copilot_driver.cjs missing (May 1-2, 37 workflows failed)
  Status: Unclear if resolved — May 2 runs (today) appear to be succeeding again for some

## HIGH
- PR-triggered concurrency cancellation (CHRONIC): 
  Affected: PR Nitpick Reviewer (0%), Grumpy Reviewer (0%), Repo Assist (0%), Daily Perf Improver (17%), PR Fix, Q, Daily Test Improver
  Fix: per-PR concurrency groups instead of global
- Update Docs: ~40% failure rate; 3.7M tokens on failed runs (#2674 open)
- Schema Consistency Checker: 5.2M tokens/single run (optimize scope)

## NEW TODAY (May 2)
- Duplicate Code Detector: found integer param parsing dup #2787 + PR #2788 ✅
- Daily Perf Improver: BenchmarkGetPendingAIEnrichmentByBook #2780 ✅
- 37 [aw] failure issues now open (from May 1-2 outage) — needs bulk close

## RESOLVED
- May 1-2 engine outage appears resolved (today's runs succeeding)

## For Workflow Health
URGENT: repo-assist recompile (Day 13)
URGENT: status-as-issue fix x3 workflows
URGENT: review/close 37 open [aw] failure issues from infrastructure outage
HIGH: concurrency policy for PR-triggered workflows
HIGH: Update Docs reliability

## For Campaign Manager
- 8 open PRs (stable)
- Substantive agent outputs limited by outage
- Duplicate Code Detector + Perf Improver active and useful
