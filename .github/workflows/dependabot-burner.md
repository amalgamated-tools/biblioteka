---
on:
  schedule: daily
  workflow_dispatch:
permissions:
  contents: read
  issues: write
  pull-requests: write
tools:
  github:
    toolsets: [repos, issues, pull_requests, code_security]
safe-outputs:
  create-issue:
    title-prefix: '[dependabot-burner] '
  add-comment:
    max: 10
    discussions: false
  update-issue:
    max: 20
imports:
  - shared/reporting.md
source: github/gh-aw/.github/workflows/dependabot-burner.md@e2db3a4a4d844e8337b59db4bf5c1d8f9458778d
---

# Dependabot Burner

- Find all open Dependabot PRs.
- Create bundle issues, each for exactly **one runtime + one manifest file**.

## Vulnerability Severity Triage

After identifying open Dependabot PRs, check the GitHub Advisory Database for vulnerability severity on each dependency update:

1. For each open Dependabot PR, inspect the PR title and body to identify the package name and updated version.
2. Use the `code_security` toolset to check for known advisories associated with the dependency being updated.
3. Based on severity, apply the following labels to the Dependabot PR (create the label if it doesn't exist):
   - `severity: critical` — for CVSS critical (9.0–10.0) vulnerabilities
   - `severity: high` — for CVSS high (7.0–8.9) vulnerabilities
   - `security` — for any PR that fixes a known CVE (critical or high)
4. For critical and high severity security PRs:
   - Add a comment to the PR explaining the severity and recommending expedited review.
   - Include the CVE identifier and a brief description of the vulnerability if available.
5. Security-fix PRs should be noted prominently in the bundle issue summary.

**Important**: If no action is needed after completing your analysis, you **MUST** call the `noop` safe-output tool with a brief explanation. Failing to call any safe-output tool is the most common cause of safe-output workflow failures.

```json
{"noop": {"message": "No action needed: [brief explanation of what was analyzed and why]"}}
```

