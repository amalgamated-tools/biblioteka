---
on:
  schedule: weekly on sunday around 06:00
  workflow_dispatch: null
permissions:
  actions: read
  contents: read
network:
  allowed:
  - defaults
  - node
imports:
- shared/reporting.md
- shared/safe-output-app.md
safe-outputs:
  create-discussion:
    category: artifacts
    close-older-discussions: true
    expires: 1d
    max: 1
description: Generates a comprehensive summary of GitHub Actions artifacts usage across all workflows in the repository
engine: copilot
features:
  copilot-requests: true
sandbox:
  agent: awf
source: github/gh-aw/.github/workflows/artifacts-summary.md@e2ae16398626875962d19c1d5aeca50298fa68da
strict: true
timeout-minutes: 15
tools:
  bash: true
  edit: null
  github:
    toolsets:
    - actions
    - repos
---
# Artifacts Summary

Generate a comprehensive summary table of GitHub Actions artifacts usage in the repository ${{ github.repository }}.

## Task Requirements

1. **Analyze all workflows** in the repository to identify which ones generate artifacts
2. **Collect artifact data** for recent workflow runs (last 30 days recommended)
3. **Generate a summary table** with the following columns:
   - Workflow Name
   - Total Artifacts Count
   - Total Size (in MB/GB)
   - Average Size per Artifact
   - Latest Run Date
   - Status (Active/Inactive)

## Analysis Instructions

Please:

1. **List all workflows** in the repository using the GitHub API
2. **For each workflow**, get recent runs and their artifacts
3. **Calculate statistics**:
   - Total number of artifacts per workflow
   - Total size of all artifacts per workflow
   - Average artifact size
   - Most recent run date
4. **Create a markdown table** with the summary
5. **Include insights** such as:
   - Which workflows generate the most artifacts
   - Which workflows use the most storage
   - Trends in artifact usage
   - Recommendations for optimization

## Output Format

Create an issue with a markdown table like this:

```markdown
# Artifacts Usage Report

| Workflow Name | Artifacts Count | Total Size | Avg Size | Latest Run | Status |
|---------------|-----------------|------------|----------|------------|--------|
| workflow-1    | 45             | 2.3 GB     | 52 MB    | 2024-01-15 | Active |
| workflow-2    | 12             | 456 MB     | 38 MB    | 2024-01-10 | Active |

## Insights & Recommendations
[Your analysis and recommendations here]
```

## Important Notes

- Focus on workflows that actually generate artifacts (skip those without any)
- Convert sizes to human-readable formats (MB, GB)
- Consider artifact retention policies in your analysis
- Include both successful and failed runs in the analysis, ignore cancelled runs

**Important**: If no action is needed after completing your analysis, you **MUST** call the `noop` safe-output tool with a brief explanation. Failing to call any safe-output tool is the most common cause of safe-output workflow failures.

```json
{"noop": {"message": "No action needed: [brief explanation of what was analyzed and why]"}}
```
