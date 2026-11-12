# Purpose

Product requirements and design docs for Minder.

# Ownership

- `requirements.md`: product requirements, open questions and architecture decisions
- `master-plan.md`: architecture, designs to write, milestones, risks and order of work
- `screens.md`: the chat screen, objects and open decisions
- `foundation.md`: design rules for the chat screen: colour, type, space, components, states
- `architecture.md`: current Go service and Flutter layers, runtime boundaries, routes and checks
- `m0-spec.md`: scope, interfaces, task plan and acceptance criteria of milestone M0
- `m1-spec.md`: scope, behaviour, interfaces and acceptance of the lists, profiles and recurrence slice
- `calendar-spec.md`: the read-only calendar page, its data and the start, duration and priority fields
- `mockups/`: HTML phone mockups of every screen with rendered PNGs (own AGENTS.md)
- `evidence/`: local and hosted verification results; architecture-cleanup.md records the current architecture checks
- `dox-license.txt`: Agent0ai DOX MIT notice
- `superpowers/`: approved architecture designs and execution plans

- `m0-execution.md`: approved execution order and checks
- `evidence/`: M0 and M1 verification results, real model costs, hosted access checks, Firefox callback proof and app/consent branding checks

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
- `evidence/AGENTS.md`: observed verification results and screenshots
- `superpowers/AGENTS.md`: approved designs and implementation plans
