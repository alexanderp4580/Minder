# Minder — Requirements Spec

Version 0.1 · 2026-11-11 · Owner: Alexander

## 1. Goal

A single personal system for tasks, lists, reminders, calendar events and schedules. Capture should take almost no effort (chat, email forwarding, voice). Reminders should be hard to ignore, and it should help with dreaded "painful" tasks. It must cost nothing to run.

### 1.1 Interaction model

- **Telegram bot:** main way to talk to Minder. Send text, voice, photos, documents and forwards; refine and update existing items by chat.
- **Web client:** second main interface, for browsing and editing in the views of §4.
- **Android app:** notifications only (reminders, sticky nagging, Done/Snooze actions). It is not a place to manage items.

## 2. Scope

**In:** tasks, lists, projects, calendar events, reminders, recurring items, email-to-item parsing, chat-based capture, Android notifications, multiple views.

**Out (v1):** team collaboration, multi-user sharing, iOS, desktop native apps, billing.

## 3. Core concepts

| Concept | Description |
|---|---|
| Item | Base unit. Has a type: **Task**, **Event** or **Reminder** |
| List | Named collection of items (e.g. Groceries, Admin, Painful) |
| Project | Groups lists or items around one goal |
| Label | Free-form tag. Multiple per item |
| Inbox | Default landing spot for captured items that are not yet sorted |

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
- Attachments (incl. original email)
- Source: manual, chat, email, voice
- Effort / dread rating (see §7)
- Created, updated, completed timestamps

## 4. Views (tabs)

Each list or project can be opened in any view. The view choice is remembered per list.

| View | Requirements |
|---|---|
| **List** | Default. Sort/group by due date, priority, label, list. Inline check-off. Collapsible subtasks |
| **Kanban** | Columns per status or custom buckets. Drag and drop between columns. WIP limit optional |
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

### 5.2 Chat (Telegram)

- Send a text message to the bot to create an item via natural language parsing.
- Forward any message to the bot to create an item with the original content attached.
- Voice messages are transcribed and then parsed.
- Photos/screenshots: extract text and parse.
- The bot replies with a parsed preview and buttons: **Confirm / Edit / Discard**.
- Commands: today, upcoming, list contents, mark done, snooze, search.

### 5.3 Email forwarding

- A dedicated address / mailbox to forward emails to.
- Each email is parsed automatically into one or more items (events, tasks, reminders).
- Extracts: title, date/time, end time, location, deadlines, reference numbers, links.
- Handles: booking confirmations, appointments, invoices/bills (due date), deliveries, meeting invites (.ics), school/club newsletters (multiple dates).
- The original email is kept as an attachment on the item.
- Confirmation goes via chat before saving. Optional auto-accept when confidence is high.
- Unparseable emails go to the Inbox with a flag.

## 6. Reminders & notifications

### 6.1 Channels

- Android app notifications (primary for hard-to-ignore reminders)
- Telegram messages (secondary; can also carry Done/Snooze buttons)
- Configurable per item, per list and globally

### 6.2 Sticky / nagging behaviour

- **Persistent mode:** a reminder re-fires at a set interval (e.g. every 15 min) until it is explicitly acknowledged.
- Swiping the notification away does **not** count as acknowledged.
- Acknowledge actions in the notification itself: **Done**, **Snooze** (15 m / 1 h / tonight / tomorrow / custom), **Open**.
- Escalation: optionally raise priority/sound after N ignored nags, and mirror to Telegram.
- Insistent/ringing mode for the highest priority.
- Quiet hours with exceptions for top priority.

### 6.3 Scheduled digests

- Morning digest: today's events, due tasks and 1–3 painful tasks.
- Evening digest (optional): what got done, what slipped, tomorrow preview.
- Weekly review prompt: stale items, overdue items, empty lists.

## 7. Painful-task support

- Items can be marked **painful** (dread rating 1–5).
- **Breakdown:** split a painful task into small first steps (≤ 5 min), suggested automatically and editable.
- Show only the **next step**, not the whole task, in digests and notifications.
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
- Available in the web client and via chat.

## 11. Android client

- Native app whose job is notifications (see §1.1).
- Notification actions per §6.2.
- Receives reminders from the server; open question: what it shows when tapped (see §16).

## 12. Web client

- All views from §4.
- Keyboard shortcuts for quick-add and navigation.
- Installable as a web app.

## 13. Data & privacy

- Self-hosted. All data stays on own infrastructure where possible.
- Single user with authentication. Chat bot only responds to the owner's account.
- Daily automatic backup, with export of everything to an open format (JSON + .ics).
- Import from common task apps (CSV at minimum).
- Clearly state which external services (if any) see email content, and allow a fully local parsing option.

## 14. Non-functional

| Requirement | Target |
|---|---|
| Cost | €0 running cost |
| Reminder accuracy | Fires within 1 minute of scheduled time |
| Capture latency | Chat/email → preview within 30 s |
| Availability | Personal use; tolerate brief downtime, but no reminder may be lost (catch up on restart) |
| Platforms | Telegram + web + Android notifications |
| Timezone | Europe/Amsterdam default; DST-correct; per-event timezone for travel |
| Language | English UI; parsing understands English and Dutch input |
| Writing style | All text the user reads (bot messages, notifications, digests, errors, interface) follows about 80% of ASD-STE100: short sentences, active voice, one meaning per word |

## 15. Priorities

| Priority | Features |
|---|---|
| **P0 (MVP)** | Items, lists, List + Calendar + Today views, chat capture with confirm, Android push with sticky nagging + Done/Snooze, recurring items, backup |
| **P1** | Email forwarding + parsing, Kanban + Table views, digests, painful-task breakdown, phone calendar sync |
| **P2** | Gantt view, saved filters, voice/photo capture, rot detection, stats, import |

## 16. Open questions

- Auto-accept parsed emails above a confidence threshold, or always confirm?
- Should nag interval escalate (15 → 10 → 5 min) or stay fixed?
- Shared lists with others (e.g. partner) in the future?
- Is fully local email parsing a hard requirement or a preference?
- What does the Android app show when a notification is tapped: a minimal item screen, or the web client?
- Does the Android app also get a home-screen widget, share-sheet quick-add or offline use?
