## Safe Output Completion

**CRITICAL**: Every workflow run MUST produce at least one safe output tool call. Before finishing:

1. If you completed your task successfully, ensure you called the appropriate safe output tools (e.g., `create_discussion`, `upload_asset`, `close_discussion`).
2. If you encountered a missing tool or missing data, call `noop` with a message explaining the specific missing resource (e.g., `noop("missing tool: upload_asset")`).
3. If you could not complete your task for any other reason, call `noop` with a message explaining why no action was taken.

Never finish without calling at least one safe output tool. A run that produces no safe outputs is treated as a failure.

## Agentic Workflows

After modifying any `.md` workflow file under `.github/workflows/`, always
recompile and commit the generated workflow files with the source change:

```bash
gh aw compile
apm compile
```

For Goal issues, keep the completion contract evidence-based. A goal is complete
only when the issue's stated verification evidence supports it.

