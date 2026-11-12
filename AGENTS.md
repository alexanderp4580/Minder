# DOX framework

- Agent must follow DOX instructions across any edits

## Core Contract

- AGENTS.md files are binding work contracts for their subtrees
- Work products, source materials, instructions, records, assets, and durable docs must stay understandable from the nearest applicable AGENTS.md plus every parent AGENTS.md above it

## Read Before Editing

1. Read the root AGENTS.md
2. Identify every file or folder you expect to touch
3. Walk from the repository root to each target path
4. Read every AGENTS.md found along each route
5. If a parent AGENTS.md lists a child AGENTS.md whose scope contains the path, read that child and continue from there
6. Use the nearest AGENTS.md as the local contract and parent docs for repo-wide rules
7. If docs conflict, the closer doc controls local work details, but no child doc may weaken DOX

Do not rely on memory. Re-read the applicable DOX chain in the current session before editing.

## Update After Editing

Every meaningful change requires a DOX pass before the task is done.

Update the closest owning AGENTS.md when a change affects:

- purpose, scope, ownership, or responsibilities
- durable structure, contracts, workflows, or operating rules
- required inputs, outputs, permissions, constraints, side effects, or artifacts
- user preferences about behavior, communication, process, organization, or quality
- AGENTS.md creation, deletion, move, rename, or index contents

Update parent docs when parent-level structure, ownership, workflow, or child index changes. Update child docs when parent changes alter local rules. Remove stale or contradictory text immediately. Small edits that do not change behavior or contracts may leave docs unchanged, but the DOX pass still must happen.

## Hierarchy

- Root AGENTS.md is the DOX rail: project-wide instructions, global preferences, durable workflow rules, and the top-level Child DOX Index
- Child AGENTS.md files own domain-specific instructions and their own Child DOX Index
- Each parent explains what its direct children cover and what stays owned by the parent
- The closer a doc is to the work, the more specific and practical it must be

## Child Doc Shape

- Create a child AGENTS.md when a folder becomes a durable boundary with its own purpose, rules, responsibilities, workflow, materials, or quality standards
- Work Guidance must reflect the current standards of the project or user instructions; if there are no specific standards or instructions yet, leave it empty
- Verification must reflect an existing check; if no verification framework exists yet, leave it empty and update it when one exists

Default section order:
- Purpose
- Ownership
- Local Contracts
- Work Guidance
- Verification
- Child DOX Index

## Style

- Keep docs concise, current, and operational
- Document stable contracts, not diary entries
- Put broad rules in parent docs and concrete details in child docs
- Prefer direct bullets with explicit names
- Do not duplicate rules across many files unless each scope needs a local version
- Delete stale notes instead of explaining history
- Trim obvious statements, repeated rules, misplaced detail, and warnings for risks that no longer exist

## Closeout

1. Re-check changed paths against the DOX chain
2. Update nearest owning docs and any affected parents or children
3. Refresh every affected Child DOX Index
4. Remove stale or contradictory text
5. Run existing verification when relevant
6. Report any docs intentionally left unchanged and why


Source: https://github.com/agent0ai/dox. License: [docs/dox-license.txt](docs/dox-license.txt).

## User Preferences

- Minder is a personal task assistant. Flutter targets Android and web; Go and SQLite run on Coolify.
- Write concise English. Use active voice, familiar words and short sentences. Use stop-slop for prose.
- Write verified facts; mark unverified claims as such. Describe current contracts rather than change history.
- Keep secrets in TokenCrypt or process variables. Do not put credentials in files, commits, logs or chat.
- Keep architecture boundaries small and explicit. Preserve existing behaviour during refactors.
- Keep Cloudflare configuration and identity values out of `cmd/minder`.
- Never add co-author trailers. Git hooks in `.githooks/` strip them; configure `core.hooksPath` in fresh clones.
- Agree scope and acceptance before each product slice. Then execute autonomously within the approved scope.
- Current work runs inline without sub-agents. Use the existing `Minder-m0-chat` worktree on `m0-chat`.
- Outside inline execution, delegated workers use Sonnet 5 or 5.5 unless the owner names another model.
- Preserve unrelated edits. Do not reset, clean, discard work, merge or publish without authorization.
- Phone approval gates merging to `main`. The owner authorized orbital preview APK builds and Telegram delivery.
- Authorized APK delivery uses the existing Telegram bot and chat. Application ID: `io.liveleds.minder`.
- Do not change global agent settings during project work.

## Development workflow

- Read relevant installed skills from `~/.codex/skills/`. Announce first use and follow the project approval rules.
- Use using-superpowers and brainstorming for intent; writing-plans for multi-step work; executing-plans for inline execution.
- Use test-driven-development for behaviour and systematic-debugging for failures. Reproduce defects before fixing them.
- Keep a design, task order, interfaces, constraints, test assertions and expected results for architectural work.
- Tests exercise behaviour and boundaries. Keep SQL integration tests on real temporary SQLite databases.
- Review requirements and code quality before completion. Resolve critical and important defects before proceeding.
- Use verification-before-completion. Record fresh commands, exit status and observed results before claiming success.
- Use finishing-a-development-branch within existing authorization. Preserve the branch when integration needs phone approval.
- Update DOX contracts, indexes and the handoff. Keep one canonical contract for each responsibility.

## Ownership

Root owns `README.md`, `PRODUCT.md`, `HANDOFF.md`, `.gitignore`, `Dockerfile` and `.dockerignore`.
`HANDOFF.md` records source, deployment and phone status. `docs/architecture.md` describes current dependencies.
Secrets and local databases under `.local/` are ignored and are not documentation sources of truth.

## Verification

Run `toolchain/check` from the repository root for dependencies, DOX, Go, Flutter and hook checks.
Build release web and run the three local browser scenarios for changes that cross the client/server boundary.

## Child DOX Index

- `.githooks/AGENTS.md`: commit hooks and their tests
- `app/AGENTS.md`: Flutter client, platform assets and tests
- `brand/AGENTS.md`: branding source and guide
- `docs/AGENTS.md`: specifications, plans, evidence and mockups
- `server/AGENTS.md`: Go service and adapters
- `toolchain/AGENTS.md`: pinned runners and verification scripts
