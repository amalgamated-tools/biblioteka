# Shared Alerts | 2026-04-29T23:47Z

## CRITICAL
- repo-assist NOT COMPILED Day 11 — #2600 (cancelled again today)
- Status-as-Issue Day 10 — #2676 #2670 #2669
- SSRF 4-way dup Day 7 — #2503+#2557+#2579+#2647; PRs #2504+#2558+#2580+#2648
  Fix: skip-if-match `is:issue is:open SSRF` on duplicate-code-detector
- groups.go 4-way dup — #2486+#2520+#2593+#2630; PRs #2487+#2521+#2594+#2631
  Fix: skip-if-match `is:issue is:open groups.go` on code-simplifier

## HIGH
- 9 [aw] failures: #2674 update-docs NEW, #2657 dependabot-bundler NEW, #2633 arborist,
  #2600 repo-assist, #2615 contrib, #2614 triage, #2583 accessibility, #2494 chronicle, #2328 detection
- 8 dependabot bundles stacking: fix close-previous logic
- update-docs: agent succeeded, GH Actions infra failed

## UPDATE
PRs 26→8 ✅ (major improvement)

## For Workflow Health
URGENT: repo-assist recompile (Day 11)
URGENT: skip-if-match for duplicate-code-detector + code-simplifier
URGENT: fix status-as-issue x3 workflows
URGENT: fix dependabot-pr-bundler stacking
