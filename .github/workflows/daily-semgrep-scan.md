---
on:
  schedule: daily
  workflow_dispatch: null
permissions:
  contents: read
  issues: read
  pull-requests: read
  security-events: read
imports:
- shared/mcp/semgrep.md
safe-outputs:
  create-code-scanning-alert:
    driver: Semgrep Security Scanner
description: Daily Semgrep security scan for SQL injection and other vulnerabilities
name: Daily Semgrep Scan
source: github/gh-aw/.github/workflows/daily-semgrep-scan.md@e2ae16398626875962d19c1d5aeca50298fa68da
---
Scan the repository for SQL injection vulnerabilities using Semgrep.

**Important**: If no action is needed after completing your analysis, you **MUST** call the `noop` safe-output tool with a brief explanation. Failing to call any safe-output tool is the most common cause of safe-output workflow failures.

```json
{"noop": {"message": "No action needed: [brief explanation of what was analyzed and why]"}}
```
