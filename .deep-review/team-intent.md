# Team Intent Brief

## Summary
Polish-with-active-hardening mode. The team (one human + multiple AI agents) is grinding through accessibility and consistency cleanup, with a parallel and *very active* second-front on auth-boundary, SSRF, and audit-logging hardening. No reverts of code (one docs revert only), no fix-on-fix loops. The signal is wide and clean: bugs land, get fixed cleanly, no thrash. Confidence is **high**.

## Bug-class mix (last 2 months)
| Class | % |
|---|---:|
| ux-flow (accessibility, form labelling, focus/contrast, ARIA semantics) | 45 |
| auth-tenancy (cross-user visibility, group membership checks, signup gate, audit-on-sensitive-action) | 15 |
| api-contract (DTO/types, sentinel error wording, response shape, JSON nullability) | 15 |
| concurrency-data-integrity (atomic owner insert, errgroup over WaitGroup, queries via errgroup) | 10 |
| performance-reliability (response-size caps, SSRF safe HTTP clients) | 10 |
| deploy-infra (JWT secret defaults & startup validation, migration timestamp format) | 5 |

## Team focus
- Accessibility hardening is the dominant cleanup theme — required-field semantics, contrast, focus management, live regions, list-item semantics. Many small, surgical commits in this cluster.
- SSRF surface gets repeatedly tightened wherever a URL is admin-configurable or sent to an external system; the team has had to revisit this on at least three distinct external integrations.
- Audit-log consistency is a slow-burn theme — entity-type labels, metadata fields, and resource labels keep landing in fixes. Auditing on sensitive actions appears to be a moving target.
- Group/membership-driven visibility (annotations, reading lists, progress) keeps surfacing fixes — the rule "is this resource visible to this user via group membership" is being learned-in-prod.
- Sentinel-error wording, name normalization, and error→HTTP-status mapping are being unified across the DB layer; an inconsistency in this mapping at any one call site is a likely defect.
- Concurrency fixes are appearing in narrow, specific places (atomic group-owner insert, errgroup over WaitGroup) — the team is finding races as they hit them, not preemptively. Unfixed races in similar code shapes are plausible.

## Mode
**polish** — Many small, focused fixes; one docs-only revert in 2 months; steady single-purpose commits; no fix-on-fix chains. The auth/security hardening commits are mixed *into* a polish cadence rather than appearing as firefighting bursts.

## Confidence
**high** — ~80+ signal commits in the window cluster cleanly into the buckets above; both quantitative (file churn lines up with the buckets) and qualitative (commit bodies) evidence agree.

## Trust-critical surface categories
- **Multi-user visibility boundaries.** Any action whose result depends on group membership, reading-list sharing, or annotation visibility flags. The team has been actively fixing both sides of this rule (showing too much, hiding too much).
- **Admin-configurable external URL inputs.** Any endpoint where an admin can plug in a URL that the server later fetches (LLM endpoints, OIDC issuer discovery, SMTP host, future webhook). SSRF defense must be present at *connection time*, not just at config-validation time, due to DNS rebinding.
- **Auth-credential storage and verification.** Any path where a credential is hashed, compared, or rotated — including the non-standard hash chain used by the KOReader sync protocol and the URL-embedded device sync token.
- **Signup / registration gate enforcement.** Any path that creates a new user record — direct signup, OIDC first-time callback, account linking — must consistently honor the registration-disabled setting and the first-user-becomes-admin invariant.
- **Audit-log emission on sensitive actions.** Any create/update/delete on a user-owned resource, plus admin actions, must produce an audit entry with the agreed entity_type / metadata shape.
- **Background-job concurrency on shared write paths.** The same row (book, author, sidecar file, reading progress) can be touched by a scan, a retry, an upload, and a user edit at the same time. Atomicity at the row/file level is the invariant to test.
- **User-file pipeline (upload → metadata → cover → sidecar → reorganize → DB).** Multi-step, cross-storage workflow with retries; orphans, half-written sidecars, and over-permissive paths are the recurring failure class.
- **OPDS / KOSync / Kobo protocol boundaries.** External clients with their own retry behavior and credential semantics; any per-user scoping miss here is amplified by automated devices that re-issue requests indefinitely.
