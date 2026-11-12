# Minder master plan

Status 2026-11-11. A new project. Requirements are in [requirements.md](requirements.md); this plan holds the architecture decisions and the order of work.

## What it is

A personal assistant for tasks, lists, reminders, calendar events and schedules. You talk to it by text or voice, and it makes reminders hard to ignore.

```
you (text, voice, share) ─▶ assistant ─▶ preview ─▶ Confirm ─▶ items in SQLite
                                                                   │
                      reminder engine ◀────────────────────────────┤
                            │                                      ▼
                  push + nagging on the phone            chat answers
```

Rules:
- The Flutter app is the main client. One codebase builds Android and web.
- No reminder is lost. After a restart the engine catches up on missed reminders.
- Running cost is at most €10 per month.
- Model and speech keys stay on the server.

## Architecture

Decided (see requirements §17):

| Area | Decision |
|---|---|
| Client | Flutter, one codebase for Android and web. The web build is also the debug target on the Steam Deck |
| Backend | Go, one process, SQLite, one container on Coolify |
| Data ownership | The server owns the data and validates every command. The app needs a connection to read or change data |
| Push | Firebase Cloud Messaging |
| Assistant | Confirm first for every request that creates or changes data |
| Access | Public HTTPS behind Cloudflare Access with a browser sign-in. The server verifies the token |
| Model | Claude Haiku 5.5 through OpenRouter |
| Email capture | Cloudflare Email Routing at `assistant@minder.liveleds.io`; existing OpenRouter parsing; always review, retain originals/attachments, and ask follow-up questions |
| Speech | A speech model through OpenRouter, English only |
| Live updates | Server-sent events for the open app. Polling is the fallback |
| Backup | Coolify backups of a snapshot directory |

Open questions are in requirements §16.

The command interface does not depend on the client. A Telegram bot, an outside agent or another client can use it later.

## Agent management (proposal)

Requirement: an AI agent manages everything and the data keeps its structure (requirements §1.2). The in-app assistant is the first agent. The mechanism below is a proposal:

- One command interface carries every change. It does not depend on the client, so outside agents can use it later. The app screens and the agent use the same commands, for example create item, update item, set status, change profile.
- The server validates each command against the schema. It rejects invalid commands with an error that tells the agent what to fix.
- Structure lives in data: allowed statuses, profile fields and enums. Free text goes only in text fields.
- Each command first produces a preview of the change.
- An audit log records every change with its actor (you or the agent). Any change can be undone.
- Queries return structured results, so the agent reads state without guessing.

## Designs to write before the build plan

1. **Screens and flows.** The app's screens, navigation and flows, as a spec with mockups (`screens.md`). It shows what the data model and the commands must support.
2. **Data model and command interface.** Items, lists, projects, labels, profiles and statuses; the command set and schema; validation; audit log and undo. The engine and the assistant depend on it.
3. **Reminder engine.** How the server schedules reminders, repeats nags until you acknowledge them, and catches up after a restart.
4. **Assistant.** How a message becomes a parsed preview and then a confirmed change through the command interface. Where the model is called. How the €10 limit is enforced.

## Milestones (proposal)

The milestones follow the P0 list in the requirements. Sizes are not estimated yet; each slice gets a size when its design is written.

| Milestone | Delivers |
|---|---|
| M0 Working chat | Go service and Flutter chat app end to end: text assistant with preview and Confirm, items, debug APK. Part B adds Cloudflare sign-in and the Coolify deploy. Spec: [m0-spec.md](m0-spec.md) |
| M1 Items and reminder engine | Items, lists, profiles and recurrence in SQLite; scheduler with catch-up; nag state. Specs: [notifications-spec.md](notifications-spec.md), [m1-spec.md](m1-spec.md) |
| M2 Notifications | Firebase push, Android notification with Done, Snooze and Open, sticky nagging |
| M3 Chat summaries | Today, upcoming and list answers in chat. Other views only if you ask |
| M4 Voice and Edit | Voice recording, the speech model on OpenRouter, and Edit in the preview card |
| M5 Backup | Daily snapshot and export, Coolify backup, one tested restore |

Email capture accepts authenticated forwards only from `EMAIL_ALLOWED_SENDERS`, a required comma-separated environment parameter on the Worker and Go server. Configure actual addresses privately through deployment environment settings. Missing or invalid configuration fails closed; no hardcoded sender fallback.
Email delivery uses a direct Worker-to-server handoff without a temporary Cloudflare content buffer. The owner can re-forward if no capture appears; Minder persists and retries work after ingestion.

Email capture now has an agreed direction and a [complete subsystem specification](superpowers/specs/2026-11-12-email-capture-design.md). The owner approved implementation; the [implementation plan](superpowers/plans/2026-11-12-email-capture.md) is complete for review. Code execution and live provisioning have not started.

P1 follows: email forwarding, List, Today and Calendar views, Kanban and Table views, digests, voice capture, day/week/month summaries and phone calendar sync.

## Risks

- **Reminder delivery on Android.** Battery limits and Doze can delay push messages. The server and Firebase push deliver reminders. Measure on a real phone from M2.
- **Custom notification layout.** Community sources say notification text cannot use a custom font. The brand layout may need images. Check before M2.
- **Cost.** Model and speech calls must fit €10 per month. Measure the cost per request in M0.
- **Web exposure.** The web client holds personal data. Decide the access method before the web build leaves the Steam Deck.
- **Parsing and transcription quality.** Not measured. Test with real English examples in M0.

## Order

1. Write the four designs, starting with the screens and flows.
2. Present the M0 slice for approval, with design, plan and acceptance criteria.
3. Build M0. Then M1 to M5, one slice at a time. Each slice needs its own approval.

Product planning also lives in [requirements.md](requirements.md) and the brand guide in `../brand/`.
