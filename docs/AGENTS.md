# Purpose

Product requirements and design docs for Minder.

# Ownership

- `requirements.md`: product requirements, open questions and architecture decisions
- `master-plan.md`: architecture, designs to write, milestones, risks and order of work
- `screens.md`: chat, calendar, objects and open decisions
- `foundation.md`: design rules for chat and calendar: colour, type, space, components, states
- `architecture.md`: current Go service and Flutter layers, runtime boundaries, routes and checks
- `m0-spec.md`: scope, interfaces, task plan and acceptance criteria of milestone M0
- `m1-spec.md`: scope, behaviour, interfaces and acceptance of the lists, profiles and recurrence slice
- `calendar-spec.md`: the read-only calendar page, its data and the start, duration and priority fields
- `email-capture-operations.md`: private configuration, activation, recovery, source retention and rollback
- `logging.md`: server log settings, events, fields, privacy limits and how to trace a missing reminder or a failed reply
- `mockups/`: HTML phone mockups of every screen with rendered PNGs (own AGENTS.md)
- `evidence/`: local and hosted verification results, architecture checks and calendar layout checks
- `dox-license.txt`: Agent0ai DOX MIT notice
- `superpowers/`: draft and approved subsystem/architecture designs and execution plans

- `m0-execution.md`: approved execution order and checks

# Local Contracts

- `requirements.md` is the single source of requirements; refinements edit it in place
- `master-plan.md` and `requirements.md` must agree; a decision changed in one changes in the other in the same edit
- Decisions the user made are written as decided; undecided items stay in the "Open questions" section of `requirements.md`
- Facts about other projects, services and libraries are verified against their source before they are written
- Calendar layout rules match the brand guide, foundation, calendar spec, mockup and Flutter rendering.
- When a screen spec exists, its mockup changes in the same edit

# Work Guidance

Re-read a whole doc after editing and fix contradictions. Use stop-slop for prose.

# Verification

- Reread whole changed documents; check links, DOX indexes and agreement between documents
- Run `git diff --check` and inspect the changed files before reporting completion

# Child DOX Index

- `mockups/AGENTS.md`: phone mockups, shared styles and rendering
- `evidence/AGENTS.md`: observed verification results and screenshots
- `superpowers/AGENTS.md`: draft and approved designs, plus implementation plans
