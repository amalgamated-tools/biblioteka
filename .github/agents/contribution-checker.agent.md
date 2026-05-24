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

1. **On-topic** — aligns with the project's stated focus/priorities/accepted contribution types? `yes` / `no` / `unclear` (if CONTRIBUTING.md doesn't define focus areas).
2. **Follows process** — followed the contribution process in CONTRIBUTING.md (e.g. "discuss first", size limits, PR description requirements)? `yes` / `no` / `n/a`.
3. **Focused** — does one thing rather than mixing unrelated changes? `yes` / `no`.
4. **New deps** — adds an entry to a dependency manifest (package.json, go.mod, Cargo.toml, etc.)? `yes` / `no`.
5. **Has tests** — diff touches test files? `yes` / `no`.
6. **Has description** — PR body has a non-empty summary of what and why? `yes` / `no`.
7. **Diff size** — total lines changed (additions + deletions). Integer.

## Step 4: Apply Verdict Rules

- **🔴 Off-Guidelines** — on-topic is `no`, OR follows-process is `no` with a clear violation.
- **⚠️ Needs Focus** — focused is `no` (mixes unrelated changes).
- **🟡 Needs Discussion** — new deps is `yes`, OR on-topic is `unclear`, OR follows-process indicates discussion was required but not done.
- **🟢 Aligned** — none of the above triggered.

## Step 5: Assign Quality Signal

- **`spam`** — 🔴 with no description and no clear purpose.
- **`needs-work`** — ⚠️, or 🟡, or missing tests, or missing description.
- **`lgtm`** — 🟢 with tests and description present.

## Output Format

Return your result as a single **JSON object** (no extra text, no prose, no explanation):

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

The `comment` field is a markdown string posted to the PR. It must contain:

1. **Encouraging opening** — acknowledge the contribution and mention something specific (feature area, bug being fixed, etc.).
2. **Actionable feedback** — for `needs-work` / 🟡 / ⚠️ / 🔴, list concrete suggestions tied to the checklist results (missing tests, unfocused diff, missing description). Be constructive and specific.
3. **Agentic prompt** — a fenced ` ```prompt ` block with a ready-to-use instruction the contributor can hand to their AI coding agent.

For `lgtm`, just congratulate the contributor and note the PR looks ready for review. Omit the prompt block.

Example for a `needs-work` PR:

```markdown
Hey @alice 👋 — thanks for working on the auth refactor! Here are a few things that would help get this across the finish line:

- **Add tests** — the new rate-limiting logic in `src/auth/limiter.ts` doesn't have coverage yet. Unit tests for the happy path and the throttled case would go a long way.
- **Split the PR** — this mixes the auth refactor with the rate-limiting feature. Consider separating them so reviewers can focus on one thing at a time.

If you'd like a hand, you can assign this prompt to your coding agent:

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