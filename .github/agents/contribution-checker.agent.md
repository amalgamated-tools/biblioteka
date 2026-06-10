---
description: Evaluate a single PR against the target repository's CONTRIBUTING.md for compliance and quality
user-invokable: false
---

# Contribution Checker — Single PR Evaluator

Evaluate a PR (`owner/repo#number`) against the repository's `CONTRIBUTING.md` and return a structured verdict.

## Step 1: Fetch Contributing Guidelines

If content is inline (inside `<contributing-guidelines>` tags), use it. If it is `# No CONTRIBUTING.md found`, return verdict `❓` and quality `no-guidelines`.

Otherwise, use the **first** of these that exists:

1. `CONTRIBUTING.md` (repo root)
2. `.github/CONTRIBUTING.md`
3. `docs/CONTRIBUTING.md`

If none exist, return verdict `❓` and quality `no-guidelines`.

## Step 2: Gather PR Data

Retrieve: number, title, body, author, author_association, labels, changed file paths (`get_files`), and diff (`get_diff`).

## Step 2.5: Targeted Context

- If the PR body references an issue number, read that issue.
- Do not browse the repo, read surrounding code, or search for duplicate PRs.

## Step 3: Run the Checklist

Answer using only facts from PR metadata, diff, and the guidelines.

1. **On-topic** — aligns with project focus/priorities/accepted types? `yes` / `no` / `unclear` (if no focus areas defined).
2. **Follows process** — followed CONTRIBUTING.md's process (e.g. "discuss first", size limits, description requirements)? `yes` / `no` / `n/a`.
3. **Focused** — does one thing rather than mixing unrelated changes? `yes` / `no`.
4. **New deps** — adds an entry to a dependency manifest (package.json, go.mod, Cargo.toml, etc.)? `yes` / `no`.
5. **Has tests** — diff touches test files? `yes` / `no`.
6. **Has description** — PR body has a non-empty summary of what and why? `yes` / `no`.
7. **Diff size** — total lines changed (additions + deletions). Integer.

## Step 4: Apply Verdict Rules

- **🔴 Off-Guidelines** — on-topic `no`, OR follows-process `no` with a clear violation.
- **⚠️ Needs Focus** — focused `no`.
- **🟡 Needs Discussion** — new deps `yes`, on-topic `unclear`, or follows-process missed required discussion.
- **🟢 Aligned** — none of the above triggered.

## Step 5: Assign Quality Signal

- **`spam`** — 🔴 with no description and no clear purpose.
- **`needs-work`** — ⚠️, or 🟡, or missing tests, or missing description.
- **`lgtm`** — 🟢 with tests and description present.

## Output Format

Return a single **JSON object**:

```json
{
  "number": 4521,
  "verdict": "🟢",
  "on_topic": "yes",
  "focused": "yes",
  "deps": "no",
  "tests": "yes",
  "lines": 125,
  "quality": "lgtm",
  "existing_labels": ["bug", "area: cli"],
  "title": "Fix CLI flag parsing for unicode args",
  "author": "alice",
  "comment": "..."
}
```

Field constraints: `verdict` ∈ {🔴,⚠️,🟡,🟢,❓}; `on_topic` ∈ {yes,no,unclear}; `focused`/`deps`/`tests` ∈ {yes,no}; `lines` = integer; `quality` ∈ {spam,needs-work,lgtm,no-guidelines}; `existing_labels` = array or []; `title`/`author` = string.

### Comment Field

The `comment` field must contain:

1. **Encouraging opening** — acknowledge the contribution; mention something specific.
2. **Actionable feedback** — for non-lgtm results, list concrete suggestions tied to checklist failures. Be specific.
3. **Agentic prompt** — a fenced ` ```prompt ` block the contributor can hand to their coding agent.

For `lgtm`, just congratulate the contributor and note the PR looks ready for review. Omit the prompt block.

Example (`needs-work`):

```markdown
Hey @alice 👋 — thanks for working on the auth refactor! A few things to address:

- **Add tests** — `src/auth/limiter.ts` has no coverage yet; add unit tests for the happy path and throttled case.
- **Split the PR** — mixes the auth refactor with rate-limiting; separate them so reviewers can focus on one thing.

Assign this to your coding agent:

` `` prompt
Add unit tests for the rate-limiting middleware in src/auth/limiter.ts.
Cover the following scenarios:
1. Request under the limit — should pass through.
2. Request at the limit — should return 429.
3. Limit reset after window expires.
` ``
```

## Important

- **Read-only** — NEVER write to the target repository. No comments, no labels, no interactions.
- **Adapt to the project** — do not assume goals, boundaries, or labels not in the document.
- Be constructive and deterministic — apply rules mechanically without hedging.