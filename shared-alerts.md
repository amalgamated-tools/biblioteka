# Shared Alerts | Updated: 2026-04-28T23:47Z

## CRITICAL
- **repo-assist NOT COMPILED Day 10** — #2600, immediate recompile needed
- **Status-as-Issue Day 9** — daily-repo-status #2645, daily-team-status #2639, daily-plan #2638
- **SSRF 4-way dup** — issues #2503+#2557+#2579+#2647; PRs #2504+#2558+#2580+#2648 (day 6, escalated from 3x yesterday)
  - Fix: add skip-if-match `is:issue is:open SSRF` to duplicate-code-detector
- **groups.go 4-way dup** — issues #2486+#2520+#2593+#2630; PRs #2487+#2521+#2594+#2631
  - Fix: add skip-if-match `is:issue is:open groups.go` to code-simplifier

## HIGH
- **7 [aw] failures**: #2633 issue-arborist (NEW Apr 28), #2600 repo-assist, #2615 contrib-checker, #2614 agentic-triage, #2583 accessibility, #2494 chronicle, #2328 detection
- **8 dependabot bundles open**: #2635+#2602+#2570+#2549+#2524+#2490+#2467+#2411 — fix close-previous logic

## Resolved Since Apr 27
- reading-groups PRs: 4→1 open ✅

## For Campaign Manager
- 26 open PRs; 4 SSRF + 4 groups.go competing PRs need dedup decision

## For Workflow Health
- URGENT: repo-assist recompile (Day 10)
- URGENT: add skip-if-match to duplicate-code-detector (SSRF) + code-simplifier (groups.go)
- URGENT: fix status-as-issue in 3 workflows (Day 9)
- URGENT: fix dependabot-pr-bundler close-previous logic (8 stacking)
