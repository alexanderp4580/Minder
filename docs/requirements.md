# Minder — Requirements Spec

Version 0.1 · 2026-11-11 · Owner: Alexander

## 1. Goal

A single personal system for tasks, lists, reminders, calendar events and schedules. Capture should take almost no effort (assistant chat, voice, email forwarding, share sheet). Reminders should be hard to ignore, and it should help with dreaded "painful" tasks. It must cost at most €10 per month to run.

### 1.1 Interaction model

- **Flutter app:** the main client. One codebase builds for Android and for the web. It holds the assistant chat (text and voice), the views of §4, the summaries and the notifications.
- **Backend:** a Go service with a SQLite database, in one container on the Coolify server. It stores the data, runs the reminder schedule, calls the language model and the speech service, and sends push messages through Firebase Cloud Messaging.
- **Telegram:** not part of v1. The assistant interface does not depend on the client, so a bot can be added later.

### 1.2 Agent management

The most important requirement: an AI agent can manage everything in Minder (items, lists, projects, labels, profiles and settings), and the data keeps its structure. The agent cannot create malformed or inconsistent data. The in-app assistant does this work. It uses a command interface that does not depend on the client, so outside agents can use the same interface later. The mechanism is in [master-plan.md](master-plan.md).

## 2. Scope

**In:** tasks, lists, projects, calendar events, reminders, recurring items, email-to-item parsing, assistant chat (text and voice), Android notifications, a web client, multiple views.

**Out (v1):** team collaboration, multi-user sharing, iOS, desktop native apps, a Telegram bot, billing.

## 3. Core concepts

| Concept | Description |
|---|---|
| Item | Base unit. Has a type: **Task**, **Event** or **Reminder** |
| List | Named collection of items (e.g. Groceries, Admin, Painful) |
| Project | Groups lists or items around one goal |
| Label | Free-form tag with a color from a fixed set of six. Multiple per item. The color always shows with the label text |
| Inbox | Default landing spot for captured items that are not yet sorted |
| Profile | A named set of reminder defaults: when to remind, how loud, and whether to nag. Built-in profiles are editable, and you can add your own |

### 3.1 Item fields

- Title (required)
- Description / notes (markdown)
- Type: task / event / reminder
- Status: open, in progress, done, cancelled
- Priority: 1–5
- Due date/time (tasks)
- Start + end date/time, all-day flag (events)
- Location (optional)
- Reminders: zero or more, absolute or relative to due/start
- Recurrence: daily, weekly, monthly, yearly, custom, plus "repeat X after completion"
- Subtasks / checklist
- Labels, list, project
- Profile (see §6.3)
- Attachments (incl. original email)
- Source: manual, chat, email, voice
- Effort / dread rating (see §7)
- Created, updated, completed timestamps

## 4. Views (tabs)

Each list or project can be opened in any view. The view choice is remembered per list.

| View | Requirements |
|---|---|
| **List** | Default. Sort/group by due date, priority, label, list. Inline check-off. Collapsible subtasks |
| **Kanban** | One board for all items, not split by project. Columns per status or custom buckets. Filter by label. Move an item with a Move button; drag and drop is a shortcut. WIP limit optional |
| **Table** | Spreadsheet-style. Choose visible columns. Sort/filter per column. Bulk edit |
| **Gantt / Timeline** | Items with start + due shown as bars. Drag to reschedule or resize |
| **Calendar** | Day / week / month / agenda. Shows events and dated tasks together. Drag to reschedule |
| **Today** | Overdue + due today + scheduled today + top painful items |
| **Upcoming** | Next 7 / 30 days, grouped by day |

### 4.1 Saved filters

- Create named filters (e.g. "Overdue high priority", "Waiting on others").
- A saved filter behaves like a list and supports all views.
- Filter on any field, with AND/OR.

## 5. Capture

### 5.1 Manual

- Quick-add bar with natural language parsing: `Call dentist tomorrow 10am #health !2 every 6 months`.
- Parse date, time, recurrence, label, priority, list.

### 5.2 Assistant chat (in the app)

- Type a message to the assistant to create an item through natural language parsing.
- Record a voice message. The server transcribes it and then parses it.
- Photos and screenshots: extract text and parse.
- The assistant replies with a parsed preview and the buttons **Confirm / Edit / Discard**.
- This applies to every assistant request that creates or changes data, including edits and deletes of existing items (for example "move the dentist to Friday"). The assistant changes nothing before you confirm.
- The assistant also answers questions: today, upcoming, list contents and search.

### 5.3 Email forwarding

- A dedicated address / mailbox to forward emails to.
- Each email is parsed automatically into one or more items (events, tasks, reminders).
- Extracts: title, date/time, end time, location, deadlines, reference numbers, links.
- Handles: booking confirmations, appointments, invoices/bills (due date), deliveries, meeting invites (.ics), school/club newsletters (multiple dates).
- The original email is kept as an attachment on the item.
- Confirmation happens in the app before saving. Optional auto-accept when confidence is high.
- Unparseable emails go to the Inbox with a flag.

## 6. Reminders & notifications

### 6.1 Channels

- Android app notifications (primary for hard-to-ignore reminders)
- Configurable per item, per list and globally

### 6.2 Sticky / nagging behaviour

- **Persistent mode:** a reminder re-fires at a set interval (e.g. every 15 min) until it is explicitly acknowledged.
- **Nag interval per item:** each item chooses a fixed interval or an escalating one (15 → 10 → 5 min).
- Swiping the notification away does **not** count as acknowledged.
- Acknowledge actions in the notification itself: **Done**, **Snooze** (15 m / 1 h / tonight / tomorrow / custom), **Open**.
- Escalation: optionally raise priority/sound after N ignored nags.
- Insistent/ringing mode for the highest priority.
- No global quiet hours. The profile of the item sets how loud a reminder is (see §6.3).
- **Catch-up:** a reminder is a critical message, so the server never skips one. After downtime it sends every missed reminder and marks it as late.

### 6.3 Profiles

- The kind of item decides how its reminders behave. A profile sets the reminder times, the loudness, the nag mode and the nag interval.
- Built-in profiles: Daily task, Chore, Birthday, Day event and Trip.
- You can edit every built-in profile and add your own.
- An item uses one profile. Reminders set on the item itself override the profile.
- A profile can also set the statuses an item moves through (for example Kanban columns). The details are not designed yet.
- Profiles are data, not code. You tune them by use.

### 6.4 Scheduled digests

- Morning digest: today's events, due tasks and 1–3 painful tasks.
- Evening digest (optional): what got done, what slipped, tomorrow preview.
- Weekly review prompt: stale items, overdue items, empty lists.

## 7. Painful-task support

- Items can be marked **painful** (dread rating 1–5).
- **Rot detection:** flag items with no progress for X days and prompt to do, reschedule, delegate or delete.
- **Daily cap:** surface at most N painful items per day.
- Track streaks / completions of painful items (simple stats).

## 8. Calendar

- Day/week/month/agenda views (see §4).
- Two-way sync with the phone's native calendar app.
- Import/export standard calendar files (.ics).
- Subscribe to external read-only calendars (e.g. holidays).
- Conflict warning when a new event overlaps an existing one.

## 9. Lists

- Simple checklist lists (groceries, packing) with no dates required.
- Reusable templates (e.g. packing list) that can be cloned.
- Reorder by drag.
- Uncheck-all / reset a list.

## 10. Search

- Full-text search across title, notes and attachments (incl. email bodies).
- Available in the app and through the assistant chat.

## 11. Android client

- Native app built with Flutter: chat, views, summaries and notifications.
- Push messages arrive through Firebase Cloud Messaging.
- Notification actions per §6.2.
- The app reads and changes data through the server. It needs a connection for this.

## 12. Web client

- Built from the same Flutter code as the Android app.
- All views from §4.
- Also the debug target on the Steam Deck.
- Exposure and sign-in: see §16.

## 13. Data & privacy

- Self-hosted. All data stays on own infrastructure where possible.
- Single user. The server accepts requests only from the paired owner device. Pairing and sign-in: see §16.
- Daily automatic backup, with export of everything to an open format (JSON + .ics).
- Import from common task apps (CSV at minimum).
- Clearly state which external services (if any) see email content, and allow a fully local parsing option.

## 14. Non-functional

| Requirement | Target |
|---|---|
| Cost | At most €10 per month for models and services, including parsing, voice transcription and hosting |
| Reminder accuracy | Fires within 1 minute of scheduled time |
| Capture latency | Message, voice or email → preview within 30 s |
| Availability | Personal use; tolerate brief downtime, but no reminder may be lost (catch up on restart) |
| Platforms | Android and web, from one Flutter codebase |
| Timezone | Europe/Amsterdam default; DST-correct; per-event timezone for travel |
| Language | English UI; parsing understands English and Dutch input |
| Writing style | All text the user reads (assistant messages, notifications, digests, errors, interface) follows about 80% of ASD-STE100: short sentences, active voice, one meaning per word |

## 15. Priorities

| Priority | Features |
|---|---|
| **P0 (MVP)** | Items, lists, List + Calendar + Today views, assistant chat with confirm (text), Android push with sticky nagging + Done/Snooze, recurring items, backup |
| **P1** | Email forwarding + parsing, Kanban + Table views, digests, voice capture, day/week/month summaries, phone calendar sync |
| **P2** | Gantt view, saved filters, photo capture, rot detection, stats, import |

## 16. Open questions

- Auto-accept parsed emails above a confidence threshold, or always confirm?
- Shared lists with others (e.g. partner) in the future?
- Is fully local email parsing a hard requirement or a preference?
- Do we keep Projects, now that the Kanban board shows all items?
- What are the default reminder times, loudness and nag settings of each built-in profile?
- Does a profile attach to an item, to a list, or to both?
- Which profiles use statuses, and what are they?
- Do items keep subtasks and checklists?
- Catch-up sends all missed reminders (you said "probably all"). Should very old ones, for example more than a week late, be grouped into one message?
- Which of share-sheet quick-add and a home-screen widget belong in v1?
- Do the Done and Snooze buttons act at once? Proposal: yes, because only assistant requests need confirmation.
- How does the web client reach you: tailnet only, or public behind an existing sign-in service? Minder does not write its own authentication.
- How does the app pair with the server, and does it reach the server over public HTTPS or over the tailnet? Leaning towards Cloudflare; not decided.
- Which language model and which speech service? Both must fit the €10 limit.

## 17. Architecture decisions

| Area | Decision |
|---|---|
| Client | Flutter, one codebase for Android and web |
| Backend | Go, one process, SQLite, one container on Coolify |
| Data ownership | The server owns the data and validates every command. The app needs a connection to read or change data |
| Push | Firebase Cloud Messaging |
| Keys | Model and speech keys stay on the server |
| Assistant | Confirm first for every request that creates or changes data |
| Agent management | The assistant manages everything through one command interface that validates every change. Outside agents can use it later |
| Budget | At most €10 per month |
