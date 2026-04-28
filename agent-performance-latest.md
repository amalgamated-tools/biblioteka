# Agent Performance — 2026-04-28 | Run: 23:47Z

## Key Metrics
- 55 workflows (all Copilot) | 7 [aw] failures | 26 open PRs | status-as-issue Day 9
- repo-assist NOT compiled Day 10 (CRITICAL) | SSRF 4-way dup | groups.go 4-way dup

## Top (score/100)
agentic-triage 90 | glossary-maintainer 85 | schema-consistency-checker 80
repository-quality-improver 80 | daily-perf-improver 80 | daily-qa 75

## Needs Improvement
repo-assist 10 (Day 10 not compiled #2600) | daily-repo-status 15 (#2645) | daily-team-status 15 (#2639)
code-simplifier 20 (groups.go 4x: #2486+#2520+#2593+#2630, PRs #2487+#2521+#2594+#2631)
duplicate-code-detector 20 (SSRF 4x: #2503+#2557+#2579+#2647, PRs #2504+#2558+#2580+#2648)
dependabot-pr-bundler 25 (8 open bundles) | issue-arborist 45 (NEW #2633)

## Regressions vs Apr 27
SSRF escalated to 4-way ❌ | groups.go 4th pair created ❌ | dependabot 8 open ❌ | issue-arborist new failure ❌

## Improvements
reading-groups PRs: 4→1 open ✅

## Discussion Created
"Agent Performance Report — 2026-04-28" (Audits)
