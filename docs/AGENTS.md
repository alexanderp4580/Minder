# Purpose

Product requirements and design docs for Minder.

# Ownership

- `requirements.md`: product requirements, open questions and architecture decisions
- `master-plan.md`: architecture, designs to write, milestones, risks and order of work
- Later design specs, plans and evidence files per slice

# Local Contracts

- `requirements.md` is the single source of requirements; refinements edit it in place
- `master-plan.md` and `requirements.md` must agree; a decision changed in one changes in the other in the same edit
- Decisions the user made are written as decided; undecided items stay in the "Open questions" section of `requirements.md`
- Facts about other projects, services and libraries are verified against their source before they are written
- When a screen spec exists, its mockup and the mockup index change in the same edit

# Work Guidance

Re-read a whole doc after editing and fix contradictions. Use stop-slop for prose.

# Verification

- Reread whole changed documents; check links, DOX indexes and agreement between documents
- Run `git diff --check` and inspect the changed files before reporting completion

# Child DOX Index

No child instruction files.
