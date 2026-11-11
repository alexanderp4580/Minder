# Minder

Personal task, list, reminder and schedule assistant. Flutter app first (Android and web from one codebase) with an in-app assistant for text and voice. Backend: Go and SQLite on Coolify.

## Project rules

- Follow the DOX framework: read every AGENTS.md on the path before editing; update the nearest owning AGENTS.md after meaningful changes
- No tombstones: docs, comments and commits describe only what exists now
- Write only requested or verified content; mark unchecked items "to check"
- Never add co-author trailers to commits. Hooks in `.githooks/` strip them; in a fresh clone run `git config core.hooksPath .githooks`
- Never write secrets to files, commits or chat. Secrets stay out of the repository
- Write all communication (chat replies, docs, UI copy, assistant messages, notifications, digests, error messages) at about 80% of ASD-STE100: sentences of 20 words or fewer, active voice, simple tenses, one word for one meaning, commands for instructions. The exact rules: to check against the official specification
- Keep communication concise; ask clarifying questions one at a time
- Delegate scoped research, drafting, implementation and review to cheaper agents. Give each clear ownership, context and acceptance criteria; the primary agent coordinates and integrates
- Run delegated workers and reviewers on Sonnet 5 or 5.5. Use another model only when the user names it for a specific task
- Preserve tokens. Plans state task order, interfaces, test assertions, fixed constraints and commands with expected results; they do not contain implementation code. The primary agent never writes the implementation itself; workers write it test-first
- Agree a concrete scope and acceptance criteria for each slice before executing it; then work autonomously within it. Phone approval is required before merging
- Superpowers and stop-slop are installed for both agents. Codex: `~/.codex/skills/`. Claude Code: Superpowers as the `superpowers@claude-plugins-official` plugin; stop-slop in `~/.claude/skills/stop-slop/`. Use them through the Development workflow below
- To confirm with the user for Minder: application ID, APK delivery through the existing Telegram bot and chat, and phone approval before a change merges to `main`

## Development workflow

Read the applicable installed `SKILL.md` at the stage where it applies, plus any referenced template or material needed for that action. Codex reads skills from `~/.codex/skills/<name>/`; Claude Code invokes them with the Skill tool (Superpowers names carry the `superpowers:` prefix). Announce first use briefly. User instructions and this project's approval and prose rules take precedence over conflicting skill defaults. Load relevant skills, not the entire collection for each task.

| Stage | Skills |
|---|---|
| Start and understand intent | `using-superpowers`, `brainstorming` |
| Prepare a multi-step implementation plan | `writing-plans` |
| Execute delegated tasks | `subagent-driven-development` |
| Parallel independent tasks | `dispatching-parallel-agents` |
| Implement behavior; investigate failures | `test-driven-development`, `systematic-debugging` |
| Request review; assess feedback | `requesting-code-review`, `receiving-code-review` |
| Check isolation; finish a verified branch | `using-git-worktrees`, `finishing-a-development-branch` |
| Claim completion; edit prose | `verification-before-completion`, `stop-slop` |

Use `executing-plans` for inline execution only if requested or delegation is unavailable. Use `writing-skills` when creating or changing a skill. Do not modify global agent settings as part of a project task.

### Scope and plan

Classify work as a research spike, a bounded change to an existing flow, or architectural work. State intent, constraints, and observable success; distinguish confirmed requirements from proposals. For architectural or multi-step work, prepare a design and a task plan with a linked spec, exact file responsibilities, interfaces, dependencies, fixed constraints, test assertions, commands, and expected results. Check for ambiguity, missing requirements, and unnecessary layers. Use checkboxes for tasks and name the behavior each task proves. Group setup and docs with their tested deliverable; split tasks where independent review is useful. A small existing-flow change needs only a concise design and checks.

Present the concrete slice design, planned work, and acceptance criteria together for one user approval. Then execute within that scope without repeated approval gates. Re-ask for scope changes or required permissions. Each product slice needs its own scope approval.

### Delegation and workspace

Inspect Git status, branch, and worktree state before changes. Reuse existing isolation; use `using-git-worktrees` when feature isolation is needed. Include the intended workspace and branch in the slice approval packet. Start product implementation on `main` only with explicit consent for that branch. Preserve unrelated edits and establish the relevant baseline. Documentation-only work may stay in the existing checkout. Do not reset, clean, merge, publish, or discard user work without authorization.

Give cheaper workers clean, bounded contexts: task and spec text, relevant DOX chain, owned files, interfaces, constraints, acceptance checks, and required output. Set model and reasoning effort explicitly from the live allowed choices. Keep dependent tasks sequential; parallelize only independent work without shared writes. Reuse the responsible worker for fixes. The coordinator integrates results and checks the final tree rather than treating a worker's success report as proof.

### Tests and debugging

For behavior changes, write a focused behavior test, run it to observe the intended failure, implement the minimum fix, rerun it, then refactor with checks green. Add regression tests for reproduced bugs. Prefer real behavior and boundary fakes over tests that mirror implementation. Verify docs and configuration with relevant checks instead of inventing unit tests. Add dependencies and abstractions when current behavior needs them.

For unexpected behavior, reproduce it, inspect evidence and recent changes, trace the cause, and test one hypothesis with one small change. Avoid bundles of speculative fixes. Repeated failed fixes require examining the design and reporting the evidence before expanding scope.

### Review and evidence

After each delegated implementation task, obtain one independent review covering requirements and code quality. Re-review a fix only when the review found Critical or Important defects, and scope it to the fix diff. Run one whole-branch review at the end. Supply the spec, task, precise diff or commit range, constraints, and observed checks without the coordinator's full conversation. Reviewers work read-only and do not dispatch more agents. Resolve unclear feedback before changing dependent code; validate suggestions against code and prior project decisions, and push back with evidence when they conflict. Fix critical and important defects before advancing; recheck the affected behavior. Record minor follow-ups.

Run fresh, relevant local checks on the final integrated tree before claiming success. Record commands, exit status, and observed results as passed, failed, or not run with reasons. Check acceptance criteria individually; distinguish automated tests, emulator tests, and user phone approval. Required failed or unrun local checks keep the slice incomplete. After implementation, use `finishing-a-development-branch` to handle integration within existing authorization; preserve a branch when integration is not authorized.

### Documentation and handoff

Update the owning DOX contracts and indexes, reread whole changed documents, and check their links and agreement. Use `stop-slop` and its references for targeted editing of documentation and UI copy: remove filler, vague claims, and formulaic prose; retain technical meaning, factual qualifiers, uncertainty, and exact values. Keep one canonical copy of each contract. Report changed files, verification evidence, and remaining gaps in a short handoff.

## Child DOX Index

- `.githooks/AGENTS.md`: commit-message hooks, setup and tests
- `docs/AGENTS.md`: requirements and design docs
- `brand/AGENTS.md`: brand guide website

Root owns: this file, `README.md`, `.gitignore`.
