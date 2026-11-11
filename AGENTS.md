# Minder

Personal task, list, reminder and schedule assistant. Telegram bot first (text, voice, documents, forwards, refine/update by chat); Android and web second.

## Project rules

- Follow the DOX framework: read every AGENTS.md on the path before editing; update the nearest owning AGENTS.md after meaningful changes
- No tombstones: docs, comments and commits describe only what exists now
- Write only requested or verified content; mark unchecked items "to check"
- Never add co-author trailers to commits. Hooks in `.githooks/` strip them; in a fresh clone run `git config core.hooksPath .githooks`
- Never write secrets to files, commits or chat. Secrets stay out of the repository
- Write all communication (chat replies, docs, UI copy, bot messages, notifications, digests, error messages) at about 80% of ASD-STE100: sentences of 20 words or fewer, active voice, simple tenses, one word for one meaning, commands for instructions. The exact rules: to check against the official specification
- Keep communication concise; ask clarifying questions one at a time
- Delegated workers and reviewers run on Sonnet 5 or 5.5; plans state contracts and tests, not implementation code

## Child DOX Index

- `.githooks/AGENTS.md`: commit-message hooks, setup and tests
- `docs/AGENTS.md`: requirements and design docs
- `brand/AGENTS.md`: brand guide website

Root owns: this file, `README.md`, `.gitignore`.
