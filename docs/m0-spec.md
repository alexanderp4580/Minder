# M0: working chat, server and app

Status 2026-11-11. The owner approved implementation and APK delivery. Evidence is in [evidence/m0.md](evidence/m0.md). Architectural work. The owner answered the open items: no agents, OpenRouter for the model, Cloudflare later, Edit left out. Design rules: [foundation.md](foundation.md). Architecture and checks: [architecture.md](architecture.md). Screen: [screens.md](screens.md) and `mockups/chat.html`.

## Goal

You open the app on the web or on Android. You type a request. Minder shows a preview. You tap Confirm and the item exists. You ask what is open, and Minder lists it. You mark an item done. Part A runs on your own machine and network with a fixed owner. Part B puts it on the Coolify server behind Cloudflare Access. Part B waits until you set up Cloudflare.

## Scope

**In**
- Go service: health, auth check, SQLite, items, commands with preview, confirm, discard and undo, chat history, event stream, text assistant on Claude Haiku 5.5 through OpenRouter, cost meter.
- Flutter app (web and Android): the chat screen, text composer, preview card, result with Undo, error with Retry, sign-in.
- Part A: server and app run locally with a fixed owner. A debug APK reaches Telegram. The phone reaches the local server over HTTPS through `tailscale serve`.
- Part B (later): sign-in through Cloudflare Access, one container on Coolify, web build served by the Go service.

**Out (later milestones)**
- Reminders, nagging, push, notifications and the nag frame: M1 and M2.
- Voice and the speech model: M4.
- Profiles, lists, labels, recurrence, subtasks, email: not in M0.
- Backup job and Coolify backup: M5.
- Gear and Settings, and every other screen: not designed.

## Behaviour

Chat entries in M0: your message, Minder text, a "Working" entry, a preview card, a result entry with Undo, an error entry with Retry.

Requests the assistant handles:
- **Create:** "Call the plumber on Thursday at 10." Preview card with the new item. Confirm creates it.
- **Change:** "Move the dentist to Friday." Preview card with the old value struck through. Confirm updates the item. The assistant can change title, due date and notes.
- **Complete:** "Mark the dentist done." Preview card. Confirm sets the status to done.
- **Ask:** "What is open?" Minder lists open items in a message. No preview.
- **Unclear request:** Minder asks one question. It changes nothing.

Rules:
- Every create, change and complete from the assistant shows a preview. Nothing is saved before Confirm.
- A preview never expires. If the item changed after the preview, Confirm fails and Minder offers a new preview.
- Discard drops the preview. Edit is not in M0. To change a proposal, you discard it and ask again.
- Undo reverses the last confirmed change. It works on one level for each result entry.
- All text from Minder follows about 80% of ASD-STE100.
- A request over the monthly cost limit (€5) gets the message "The monthly limit is reached." Nothing is sent to the model.
- Without a connection, the app shows an error entry and Retry.

## Data

Item fields in M0:

| field | rule |
|---|---|
| `id` | server-generated, stable |
| `title` | required, 1 to 200 characters |
| `status` | enum: open, in_progress, done, cancelled. M0 uses open and done |
| `due_at` | optional, stored in UTC |
| `timezone` | IANA name, default Europe/Amsterdam |
| `notes` | optional text, up to 4,000 characters |
| `version` | integer, increases on every change |
| `created_at`, `updated_at`, `completed_at` | UTC |

Other tables: `chat_entry`, `pending_change`, `audit`, `usage`. Full column lists belong to the schema file written in task 1.

## Interfaces

### HTTP API

All bodies are JSON. Every route except `/v1/health` needs a valid Cloudflare token.

| route | request | response |
|---|---|---|
| `GET /v1/health` | none | `status`, `version` |
| `GET /v1/chat?after=<id>&limit=<n>` | optional cursor | list of chat entries, oldest first |
| `POST /v1/chat/messages` | `text` | your entry and the id of the reply, which arrives as an event |
| `POST /v1/changes/{id}/confirm` | none | result entry with the applied change and an undo id |
| `POST /v1/changes/{id}/discard` | none | updated preview entry marked discarded |
| `POST /v1/undo/{id}` | none | result entry |
| `GET /v1/events` | none | event stream |

Chat entry fields: `id`, `kind` (user, minder, preview, result, error), `text`, `created_at`, and for a preview or result: `change` with `action`, `item_title`, `fields` (each with `name`, `old`, `new`), `state` (pending, confirmed, discarded, failed) and `undo_id` when available.

Events: `entry` (a new or changed chat entry) and `ping` (every 20 seconds). The stream uses `text/event-stream`. The server sets headers that stop proxy buffering.

Errors: HTTP status, `code`, `field` when it applies, and a short `message` in STE. Codes: `invalid_field`, `conflict` (item changed), `not_found`, `limit_reached`, `model_error` (HTTP 502), `internal_error` (HTTP 500), `unauthorized`.

### Command layer

| command | payload | validation | inverse |
|---|---|---|---|
| create_item | title, due_at, timezone, notes | title length, notes length, valid zone | delete the created item |
| update_item | item id, expected version, changed fields | item exists, version matches, field rules | restore the old values |
| complete_item | item id, expected version | item exists, status is open or in_progress | restore the old status and clear `completed_at` |

Every command has two steps, preview and apply (architecture.md). Apply runs in one transaction and writes an audit record with the actor (`owner` or `agent`).

### Assistant contract

- Input: the new message and the last 20 chat entries.
- The model is `anthropic/claude-haiku-5.5` through the OpenRouter chat API, with tool calls. OpenRouter lists the same price as Anthropic: $0.10 per million input tokens and $0.50 per million output tokens (checked 2026-11-11). The client is an interface with a real and a fake implementation.
- The model has four tools: propose create, propose update, propose complete, list items. A propose tool calls the preview step. It never applies.
- The system prompt states: English only, STE style, ask one question when the request is unclear, never invent an item id.
- The cost meter stores tokens and the cost for each request. It reads the cost value that OpenRouter returns with each response. The monthly total adds these values.

### Auth

- The service verifies `Cf-Access-Jwt-Assertion`: signature against the team keys, expiry, audience tag and the owner's email.
- `AUTH_MODE=dev` skips the check and uses a fixed owner. Part A runs in this mode. The service refuses to start in `dev` mode when `ENV=production`, so it cannot go on the public server without the check.

## Flutter app

- Entries render from the chat controller state. Widgets read `tokens.dart` only.
- The chat controller states: idle, sending, waiting, error. The event stream adds entries.
- Confirm, Discard and Undo call the API and update the entry from the response.
- On start the controller loads the last 50 entries and opens the event stream. After a drop it reconnects and loads entries after the last id.
- Part A: no sign-in. The app talks to the local server in dev mode.
- Part B: web sign-in uses the Cloudflare cookie. Android sign-in uses managed OAuth (architecture.md, spike table).

## Fixed constraints

- No secret in the repository. Keys come from the environment. The OpenRouter key stays in TokenCrypt. A real model call uses `tokencrypt run --with openrouter_minder_token`.
- Only `store` writes SQL. Only `command` changes data.
- Times are UTC in storage and Europe/Amsterdam by default in display.
- The text on screen follows the foundation rules. No hex value in a widget.
- Application ID: `io.liveleds.minder`. It is the reverse of `liveleds.io`, a domain the owner holds. It can change until the first release.

## Task plan

Each task starts with a failing test, then the minimum code, then green. A worker writes the code. The coordinator checks the result. After each task one independent review checks requirements and quality.

| # | task | behaviour it proves | depends on |
|---|---|---|---|
| 0 | Toolchain and skeleton | `server/` and `app/` build and test inside pinned podman images on the Steam Deck. Each folder has its `AGENTS.md` | none |
| 1 | Store and schema | Migrations run on an empty file and a second run changes nothing. Item rows keep UTC and zone | 0 |
| 2 | Command layer | Preview shows old and new values. Confirm applies once. A changed item makes Confirm fail with `conflict`. Undo restores. Audit records the actor | 1 |
| 3 | Auth middleware | A valid token passes. A bad signature, expired token, wrong audience and wrong owner each give 401. `dev` mode refuses to start in production | 0 |
| 4 | Chat API and events | Posting a message stores it and emits an entry event. Entries load after a cursor. The stream sends a ping every 20 seconds and survives a client reconnect | 2, 3 |
| 5 | Assistant | With a fake model: create, change, complete and ask requests produce the right previews or answers. An unclear request produces one question and no change. The cost meter blocks a request at the limit. A manual run proves one real Haiku call through OpenRouter | 4 |
| 6 | App: tokens and chat entries | Golden images of each entry type match the approved mockup | 0 |
| 7 | App: controller and API client | With a fake server, a send gives a preview. Confirm gives a result. Discard and Undo work. A dropped stream reconnects and fills the gap. Errors show Retry | 4 (contract), 6 |
| 8 | Sign-in (Part B) | Web build signs in and calls the API through Cloudflare Access. Android signs in through managed OAuth and refreshes the token. A real test client holds the stream for 5 minutes | 3, 7 |
| 9 | Deploy (Part B) | One container runs on Coolify. The Go service serves the web build and the API from one origin. `GET /v1/health` answers through Cloudflare Access | 5, 7 |
| 10 | End to end | Part A: the criteria marked A pass on your machine, and a debug APK reaches Telegram. Part B: the criteria marked B pass on the deployed system | 7 (Part A), 8 and 9 (Part B) |

Part A is tasks 0 to 7 and the Part A half of task 10. Part B is tasks 8 and 9 and the Part B half of task 10.

Tasks 1 to 5 (server) and tasks 6 and 7 (app) can run in parallel after task 0, because they share only the API contract in this file.

Commands and expected results (names fixed in task 0):
- `server`: `go test ./...` exits 0 with no failing test. `go vet ./...` prints nothing.
- `app`: `flutter test` exits 0. `flutter analyze` prints "No issues found".
- Both run through the podman images, so the Steam Deck needs no local Go or Flutter.
- Task 2 evidence: a test that applies a create, a change and an undo on a real SQLite file and compares rows.
- Task 4 evidence: a test with a real HTTP server and a client that disconnects during a stream.

## Acceptance criteria

Each criterion needs evidence recorded as a command, an exit status and an observed result. Criteria 1 to 8, 13 and 14 and the (A) parts apply to Part A. The (B) parts apply to Part B.

1. `go test ./...`, `go vet ./...`, `flutter test` and `flutter analyze` pass in the pinned images. Automated.
2. A create request ends with the item in SQLite, one audit record and an Undo that removes it. Automated.
3. A change on an item edited after the preview fails with `conflict` and a new preview. Automated.
4. No test path writes an item before Confirm. Automated.
5. A request at the cost limit returns `limit_reached` and makes no model call. Automated.
6. A request with an invalid token gets 401 in each of the four cases in task 3, and `dev` mode refuses to start in production. Automated.
7. The stream sends a ping every 20 seconds and a reconnect loses no entry. Automated.
8. The chat entry goldens match the mockup within the tolerance set in task 6. Automated.
9. (B) The deployed web app signs in, accepts "Call the plumber on Thursday at 10", shows a preview and creates the item after Confirm. Manual on the deployed system.
10. (A) The same create flow works in the web build and in the Android debug build against the local server. Manual, on a phone. (B) The same on the deployed system.
11. (B) The deployed stream delivers an event within 2 seconds and stays open 5 minutes. Manual. If it fails, switch to polling every 5 seconds and record the result.
12. (A) One real call through OpenRouter returns a preview, and its cost shows in `usage`. Manual.
13. The text on screen contains no sentence over 20 words. Manual read of every string.
14. Owning docs are updated, links resolve and `git diff --check` is clean. Automated.

Not in the criteria: reminders, push, voice, backup, and any screen other than chat.

## Owner actions

- Part A: The key exists as `openrouter_minder_token` in TokenCrypt. I request it for the real-call checks only.
- Part B: the Cloudflare Access application exists (see architecture.md, "Cloudflare setup"). Coolify needs an application for the container and the domain `minder.liveleds.io`. That step is task 9.
- Approve the slice (see below).

## Approval packet

- **Workspace and branch:** a git worktree `m0-chat` from `main`. Tell me if you want `main` instead.
- **Execution:** I write the code myself, test-first, task by task. No sub-agents, as you asked. This overrides the project rule that workers write the code. One review pass per task is my own check against the spec.
- **Phone approval:** required before anything merges to `main`.

## Open items

- ? Part A phone check: `tailscale serve` puts HTTPS in front of the local server, so the app never uses plain HTTP. To check: HTTPS is enabled on your tailnet and the phone runs Tailscale. If this fails, the Android check moves to Part B.
- ? Whether `AUTH_MODE=dev` stays after Part B. It stays for local work, with the production guard.
