/**
 * graidr scorer — two-phase scoring
 *
 * Phase 1: Deterministic facts (computed via shell, not LLM)
 * Phase 2: LLM subjective scoring (only what GPT-4o can actually judge)
 *
 * Final score = deterministic + subjective per dimension (0-100)
 */

'use strict'
// eslint-disable-next-line @typescript-eslint/no-require-imports
const { execSync } = require('child_process')
// eslint-disable-next-line @typescript-eslint/no-require-imports
const https = require('https')
// eslint-disable-next-line @typescript-eslint/no-require-imports
const fs = require('fs')
// eslint-disable-next-line @typescript-eslint/no-require-imports
const path = require('path')

// ── endpoints ─────────────────────────────────────────────────

// Public Supabase Edge Function endpoint — not a secret, no credentials embedded
const SUBMIT_SCORE_URL = 'https://kyfbjdzfizzfpdjqdpyz.supabase.co/functions/v1/submit-score'
const GITHUB_MODELS_ENDPOINT = 'https://models.inference.ai.azure.com/chat/completions'
const MODEL_ID = 'gpt-4o'

// ── helpers ───────────────────────────────────────────────────

function req(url, opts, body) {
  return new Promise((resolve, reject) => {
    const r = https.request(url, opts, res => {
      let data = ''
      res.on('data', c => (data += c))
      res.on('end', () => {
        if (res.statusCode >= 400) reject(new Error(`HTTP ${res.statusCode}: ${data}`))
        else resolve(data ? JSON.parse(data) : null)
      })
    })
    r.on('error', reject)
    if (body) r.write(JSON.stringify(body))
    r.end()
  })
}

function sh(cmd) {
  try {
    return execSync(cmd, { encoding: 'utf8', stdio: ['pipe', 'pipe', 'pipe'] }).trim()
  } catch (err) {
    // Non-zero exit is expected for grep/find "not found" results — return empty string.
    // Re-throw only on signals (e.g. SIGKILL) which indicate a real system failure.
    if (err.signal) throw err
    return ''
  }
}

function shCount(cmd) {
  const out = sh(cmd)
  return out === '' ? 0 : parseInt(out, 10) || 0
}

// ── Phase 1: Deterministic facts ──────────────────────────────

function collectFacts() {
  const facts = {}

  // Structure facts
  facts.has_typescript = shCount("git ls-files -- '*.ts' '*.tsx' | head -1 | wc -l") > 0
  facts.has_tests = shCount("git ls-files -- '*.test.*' '*.spec.*' '*__tests__*' | head -1 | wc -l") > 0
  facts.has_build_script = false
  facts.has_lint_script = false
  facts.has_test_script = false
  try {
    const pkg = JSON.parse(sh('git show HEAD:package.json'))
    const scripts = pkg.scripts || {}
    facts.has_build_script = 'build' in scripts
    facts.has_lint_script = 'lint' in scripts
    facts.has_test_script = 'test' in scripts
  } catch { /* no package.json */ }

  // Find largest source file
  const fileSizes = sh("git ls-files -- '*.ts' '*.tsx' '*.js' '*.jsx' | grep -v node_modules | head -30 | xargs wc -l 2>/dev/null | grep -v total | sort -rn | head -1")
  if (fileSizes) {
    const match = fileSizes.match(/^\s*(\d+)\s+(.+)$/)
    if (match) {
      facts.largest_file_lines = parseInt(match[1], 10)
      facts.largest_file_name = match[2]
    }
  }
  facts.largest_file_lines = facts.largest_file_lines || 0
  facts.largest_file_name = facts.largest_file_name || ''

  // Safety facts
  facts.has_gitignore = fs.existsSync(path.join(process.cwd(), '.gitignore'))
  facts.env_in_gitignore = false
  if (facts.has_gitignore) {
    const gi = sh('cat .gitignore')
    facts.env_in_gitignore = /\.env/.test(gi)
  }
  facts.env_files_committed = shCount("git ls-files -- '.env*' | wc -l") > 0
  facts.potential_secrets = shCount("git ls-files -- '*.ts' '*.tsx' '*.js' '*.jsx' '*.json' | grep -Ev '(^|/)(package-lock\\.json|yarn\\.lock|pnpm-lock\\.yaml|npm-shrinkwrap\\.json)$' | xargs grep -Eil '(\\bsk-[a-zA-Z0-9]{20,}|\\bsk_live_[a-zA-Z0-9]+|\\bAKIA[0-9A-Z]{16}\\b|\\bpassword\\s*=\\s*\"[^\"]+\"|\\bapi_key\\s*=\\s*\"[^\"]+\")' 2>/dev/null | wc -l")
  facts.eval_usage = shCount("git ls-files -- '*.ts' '*.tsx' '*.js' '*.jsx' | xargs grep -l 'eval(' 2>/dev/null | wc -l") > 0

  // Completeness facts
  facts.has_readme = false
  const readmeLines = shCount("git show HEAD:README.md 2>/dev/null | wc -l")
  facts.has_readme = readmeLines > 10

  facts.has_lockfile = sh("git ls-files -- 'package-lock.json' 'yarn.lock' 'pnpm-lock.yaml' | head -1") !== ''
  const lockfileCount = shCount("git ls-files -- 'package-lock.json' 'yarn.lock' 'pnpm-lock.yaml' | wc -l")
  facts.mixed_lockfiles = lockfileCount > 1

  facts.console_log_count = shCount("git ls-files -- '*.ts' '*.tsx' '*.js' '*.jsx' | grep -v node_modules | grep -v '.test.' | grep -v '.spec.' | xargs grep -c 'console\\.log' 2>/dev/null | awk -F: '{s+=$2} END {print s}'")
  facts.todo_fixme_count = shCount("git ls-files -- '*.ts' '*.tsx' '*.js' '*.jsx' | grep -v node_modules | xargs grep -Eci 'TODO|FIXME|HACK' 2>/dev/null | awk -F: '{s+=$2} END {print s}'")

  facts.total_source_files = shCount("git ls-files -- '*.ts' '*.tsx' '*.js' '*.jsx' | grep -v node_modules | wc -l")

  return facts
}

// ── Deterministic scoring ─────────────────────────────────────

function computeDeterministicScores(facts) {
  let structure = 0
  if (facts.has_typescript)            structure += 10
  if (facts.has_tests)                 structure += 10
  if (facts.largest_file_lines < 300)  structure += 10
  if (facts.has_build_script)          structure += 10
  if (facts.has_lint_script)           structure += 10

  let safety = 0
  if (facts.has_gitignore)             safety += 10
  if (facts.env_in_gitignore)          safety += 10
  if (!facts.env_files_committed)      safety += 10
  if (facts.potential_secrets === 0)   safety += 10
  if (!facts.eval_usage)               safety += 10

  let completeness = 0
  if (facts.has_readme)                completeness += 10
  if (facts.has_lockfile)              completeness += 10
  if (!facts.mixed_lockfiles)          completeness += 10
  if (facts.console_log_count < 5)     completeness += 10
  if (facts.todo_fixme_count < 5)      completeness += 10

  return { structure, safety, completeness }
}

// ── Smart file sampling ───────────────────────────────────────

function collectContext(deep, facts) {
  // File tree
  const fileTree = sh(
    "find . -not -path '*/node_modules/*' -not -path '*/.git/*' -not -path '*/.next/*' -not -path '*/dist/*' -maxdepth 3 | sort | head -80"
  )

  // Tier 1: always include
  const tier1 = ['package.json', '.gitignore', 'README.md']

  // Tier 2: security-relevant files
  const tier2raw = sh(
    "git ls-files -- 'app/api/**' 'pages/api/**' 'middleware.*' '*.config.*' 'lib/auth*' 'src/lib/auth*' 'lib/db*' 'src/lib/db*' | head -5"
  )
  const tier2 = tier2raw ? tier2raw.split('\n').filter(Boolean) : []

  // Tier 3: largest files (where vibe-code complexity hides)
  const largestRaw = sh(
    "git ls-files -- '*.ts' '*.tsx' '*.js' '*.jsx' | grep -v node_modules | grep -v '.test.' | grep -v '.spec.' | head -30 | xargs wc -l 2>/dev/null | grep -v total | sort -rn | head -3 | awk '{print $2}'"
  )
  const tier3 = largestRaw ? largestRaw.split('\n').filter(Boolean) : []

  // Tier 4: entry points
  const tier4 = ['app/page.tsx', 'app/layout.tsx', 'src/app/page.tsx', 'src/app/layout.tsx', 'index.tsx', 'index.ts']

  // Deduplicate and collect
  const maxFiles = deep ? 12 : 8
  const allFiles = [...new Set([...tier1, ...tier2, ...tier3, ...tier4])]
  const maxLines = deep ? 80 : 60

  let fileSamples = ''
  let collected = 0
  for (const f of allFiles) {
    if (collected >= maxFiles) break
    try {
      const content = sh(`git show HEAD:${f}`)
      if (!content) continue
      const lines = content.split('\n').slice(0, maxLines).join('\n')
      fileSamples += `\n\n### ${f}\n\`\`\`\n${lines}\n\`\`\``
      collected++
    } catch {
      // file doesn't exist in this repo
    }
  }

  return { fileTree, fileSamples }
}

// ── Deep scan detection ───────────────────────────────────────

function isDeepScan() {
  if (process.env.DEEP_SCAN === 'true') return true
  try {
    const msg = sh('git log -1 --pretty=%B')
    if (msg.includes('[grade:deep]')) return true
  } catch { /* ignore */ }
  return false
}

// ── Parse LLM response ───────────────────────────────────────

function parseScore(text) {
  const json = text.replace(/^```(?:json)?\n?/, '').replace(/\n?```$/, '')
  return JSON.parse(json)
}

// ── Score with GitHub Models ──────────────────────────────────

async function scoreWithGitHubModels(context, facts, deterministicScores, deep) {
  const { fileTree, fileSamples } = context
  const ratedWith = deep ? 'gpt-4o-deep' : 'gpt-4o'
  console.log(`graidr: scoring with ${MODEL_ID}${deep ? ' (deep)' : ''}...`)

  const factsBlock = `## Pre-computed facts (verified, not estimated)
- TypeScript: ${facts.has_typescript ? 'yes' : 'no'}
- Tests found: ${facts.has_tests ? 'yes' : 'no'}
- Largest file: ${facts.largest_file_name} (${facts.largest_file_lines} lines)
- Build script: ${facts.has_build_script ? 'yes' : 'no'}
- Lint script: ${facts.has_lint_script ? 'yes' : 'no'}
- Test script: ${facts.has_test_script ? 'yes' : 'no'}
- .gitignore exists: ${facts.has_gitignore ? 'yes' : 'no'}
- .env in .gitignore: ${facts.env_in_gitignore ? 'yes' : 'no'}
- .env files committed: ${facts.env_files_committed ? 'YES — CRITICAL' : 'no'}
- Potential secrets in code: ${facts.potential_secrets}
- eval() usage: ${facts.eval_usage ? 'YES' : 'no'}
- README.md: ${facts.has_readme ? 'yes (>10 lines)' : 'missing or minimal'}
- Lock file: ${facts.has_lockfile ? 'yes' : 'no'}
- Mixed lock files: ${facts.mixed_lockfiles ? 'YES' : 'no'}
- console.log count: ${facts.console_log_count}
- TODO/FIXME/HACK count: ${facts.todo_fixme_count}
- Total source files: ${facts.total_source_files}

## Deterministic scores (already computed, you add subjective on top)
- Structure: ${deterministicScores.structure}/50
- Safety: ${deterministicScores.safety}/50
- Completeness: ${deterministicScores.completeness}/50`

  const userMessage = `${factsBlock}

## File tree
\`\`\`
${fileTree}
\`\`\`

## File contents (sampled — you are NOT seeing the entire repo)
${fileSamples}`

  const u = new URL(GITHUB_MODELS_ENDPOINT)
  const response = await req(
    GITHUB_MODELS_ENDPOINT,
    {
      hostname: u.hostname,
      path: u.pathname,
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${process.env.GITHUB_TOKEN}`,
        Accept: 'application/vnd.github+json',
        'X-GitHub-Api-Version': '2022-11-28',
      },
    },
    {
      model: MODEL_ID,
      messages: [
        {
          role: 'system',
          content: `You are Graidr, a code quality scorer for vibe-coded web apps.

You receive pre-computed facts and sampled file contents from a GitHub repo.
The deterministic scores (0-50 per dimension) are already calculated. Your job is to add SUBJECTIVE scores (0-50 per dimension) based on what you can see in the code.

IMPORTANT: You are seeing a PARTIAL sample of the repo. If you cannot verify something from the provided files, score it as 0 — do not guess or assume.

## Score 5 subjective criteria per dimension (0-10 each, max 50 per dimension):

### Structure (subjective)
- Consistent naming conventions (files, variables, functions): 0-10
- Components/functions are single-purpose and focused: 0-10
- Clear separation of concerns (UI vs logic vs data): 0-10
- No copy-paste / duplicate code patterns visible: 0-10
- Imports are organized and dependencies make sense: 0-10

### Safety (subjective)
- Error handling present on fetch/API calls (try/catch or .catch): 0-10
- Input validation on forms or API route handlers: 0-10
- No hardcoded URLs pointing to localhost or http://: 0-10
- Auth checks present where expected (protected routes/APIs): 0-10
- No risky patterns (innerHTML, string concatenation for queries): 0-10

### Completeness (subjective)
- Loading states exist (skeleton, spinner, loading text): 0-10
- Error states shown to user (not just console.error): 0-10
- Empty states handled (empty lists, no data): 0-10
- Default boilerplate removed (no leftover starter code): 0-10
- Project feels intentional, not abandoned mid-build: 0-10

## Repo category — classify this repo into EXACTLY ONE of these categories:
portfolio, saas, e-commerce, blog, dashboard, api, cli, library, docs, other

Pick the single best fit based on the repo's purpose and structure.

## Output format — return ONLY this JSON, no other text:
{
  "structure_subjective": <integer 0-50>,
  "safety_subjective": <integer 0-50>,
  "completeness_subjective": <integer 0-50>,
  "category": "<one of: portfolio, saas, e-commerce, blog, dashboard, api, cli, library, docs, other>",
  "top_issues": [
    "<specific actionable issue with file name if possible>",
    "<specific actionable issue>",
    "<specific actionable issue>"
  ],
  "doing_well": [
    "<specific thing done right>",
    "<specific thing done right>"
  ]
}

Rules:
- top_issues: 3-5 items. Be SPECIFIC — mention file names, line counts, function names.
- doing_well: 1-3 items. Genuine positives only, no filler.
- Each subjective score is the SUM of 5 criteria (each 0-10). Show your work in your reasoning but only output the JSON.
- If the repo is mostly empty or you can't assess a criterion, score it 0.`,
        },
        { role: 'user', content: userMessage },
      ],
      temperature: 0,
      max_tokens: 1024,
    },
  )

  const text = response.choices[0].message.content.trim()
  const subjective = parseScore(text)

  // Combine deterministic + subjective
  const score = {
    structure_score: Math.min(100, deterministicScores.structure + (subjective.structure_subjective || 0)),
    safety_score: Math.min(100, deterministicScores.safety + (subjective.safety_subjective || 0)),
    completeness_score: Math.min(100, deterministicScores.completeness + (subjective.completeness_subjective || 0)),
    category: subjective.category || 'other',
    top_issues: subjective.top_issues || [],
    doing_well: subjective.doing_well || [],
  }

  return { score, ratedWith }
}

// ── Submit score ──────────────────────────────────────────────

async function submitScore(score, ratedWith) {
  const u = new URL(SUBMIT_SCORE_URL)
  return req(
    SUBMIT_SCORE_URL,
    {
      hostname: u.hostname,
      path: u.pathname,
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${process.env.GITHUB_TOKEN}`,
      },
    },
    {
      github_repository: process.env.GITHUB_REPOSITORY,
      github_sha: process.env.GITHUB_SHA,
      branch: process.env.GITHUB_REF_NAME,
      structure_score: score.structure_score,
      safety_score: score.safety_score,
      completeness_score: score.completeness_score,
      category: score.category,
      top_issues: score.top_issues,
      doing_well: score.doing_well,
      rated_with: ratedWith,
    },
  )
}

// ── Grade helpers ─────────────────────────────────────────────

function gradeLabel(n) {
  if (n >= 90) return 'S'
  if (n >= 80) return 'A'
  if (n >= 70) return 'B'
  if (n >= 50) return 'C'
  return 'D'
}

function deltaStr(current, previous) {
  if (previous === null) return ''
  const d = current - previous
  if (d > 0) return ` ↑ +${d}`
  if (d < 0) return ` ↓ ${d}`
  return ' · 0'
}

// ── Post PR comment ───────────────────────────────────────────

async function postPRComment(score, ratedWith) {
  if (!process.env.PR_NUMBER) return

  const overall = Math.round((score.structure_score + score.safety_score + score.completeness_score) / 3)
  const grade = gradeLabel(overall)
  const repo = process.env.GITHUB_REPOSITORY || ''
  if (!repo.includes('/')) return
  const [owner, name] = repo.split('/')

  const issuesList = (score.top_issues || []).map(i => `- ${i}`).join('\n')
  const wellList = (score.doing_well || []).map(i => `- ${i}`).join('\n')

  const body = [
    `## Graidr score — **${overall}/100** · Grade **${grade}**`,
    '',
    '| Dimension | Score |',
    '|-----------|------:|',
    `| Structure | **${score.structure_score}** |`,
    `| Safety | **${score.safety_score}** |`,
    `| Completeness | **${score.completeness_score}** |`,
    '',
    '### Top issues',
    issuesList || '- None found',
    '',
    '### Doing well',
    wellList || '- N/A',
    '',
    `Scored with \`${ratedWith}\` · [graidr.tools](https://graidr.tools/repo/${owner}/${name}) · \`${(process.env.GITHUB_SHA || '').slice(0, 7)}\``,
  ].join('\n')

  await req(
    `https://api.github.com/repos/${owner}/${name}/issues/${process.env.PR_NUMBER}/comments`,
    {
      hostname: 'api.github.com',
      path: `/repos/${owner}/${name}/issues/${process.env.PR_NUMBER}/comments`,
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${process.env.GITHUB_TOKEN}`,
        'User-Agent': 'graidr-scorer/1.0',
      },
    },
    { body },
  )
}

// ── Print score card ──────────────────────────────────────────

function printScoreCard(score, ratedWith) {
  const overall = Math.round((score.structure_score + score.safety_score + score.completeness_score) / 3)
  const grade = gradeLabel(overall)

  console.log(`
  graidr — ${overall}/100 · ${grade}

  Structure     ${score.structure_score}
  Safety        ${score.safety_score}
  Completeness  ${score.completeness_score}

  Top issues:`)
  for (const issue of (score.top_issues || [])) {
    console.log(`    - ${issue}`)
  }
  if (score.doing_well?.length) {
    console.log(`\n  Doing well:`)
    for (const w of score.doing_well) {
      console.log(`    + ${w}`)
    }
  }
  console.log(`\n  Model: ${ratedWith}\n`)
}

// ── main ──────────────────────────────────────────────────────

async function main() {
  const deep = isDeepScan()

  console.log('graidr: collecting facts...')
  const facts = collectFacts()

  console.log('graidr: computing deterministic scores...')
  const deterministicScores = computeDeterministicScores(facts)

  console.log('graidr: sampling files...')
  const context = collectContext(deep, facts)

  const { score, ratedWith } = await scoreWithGitHubModels(context, facts, deterministicScores, deep)

  printScoreCard(score, ratedWith)

  console.log('graidr: submitting score...')
  await submitScore(score, ratedWith)

  if (process.env.PR_NUMBER) {
    console.log('graidr: posting PR comment...')
    await postPRComment(score, ratedWith)
  }

  console.log('graidr: done.')
}

main().catch(err => {
  console.error('graidr: non-fatal error —', err.message ?? err)
  process.exit(0) // never fail the pipeline
})