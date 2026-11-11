# Purpose

Product requirements and design docs for Minder.

# Ownership

- `requirements.md`: product requirements, open questions and architecture decisions
- `master-plan.md`: architecture, designs to write, milestones, risks and order of work
- `screens.md`: the chat screen, objects and open decisions
- `foundation.md`: design rules for the chat screen: colour, type, space, components, states
- `architecture.md`: Go service and Flutter app architecture, checks done, spikes
- `m0-spec.md`: scope, interfaces, task plan and acceptance criteria of milestone M0
- `mockups/`: HTML phone mockups of every screen with rendered PNGs (own AGENTS.md)
- Later design specs, plans and evidence files per slice

- `m0-execution.md`: approved execution order and checks
- `evidence/`: M0 verification results, real model costs, hosted access checks and Firefox callback proof

# Local Contracts

- `requirements.md` is the single source of requirements; refinements edit it in place
- `master-plan.md` and `requirements.md` must agree; a decision changed in one changes in the other in the same edit
- Decisions the user made are written as decided; undecided items stay in the "Open questions" section of `requirements.md`
- Facts about other projects, services and libraries are verified against their source before they are written
- When a screen spec exists, its mockup changes in the same edit

# Work Guidance

Re-read a whole doc after editing and fix contradictions. Use stop-slop for prose.

# Verification

- Reread whole changed documents; check links, DOX indexes and agreement between documents
- Run `git diff --check` and inspect the changed files before reporting completion

# Child DOX Index

- `mockups/AGENTS.md`: phone mockups, shared styles and rendering
