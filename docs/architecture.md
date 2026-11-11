# Minder architecture (draft)

Status 2026-11-11. This is a proposal for the Go service and the Flutter app. The decided points come from [requirements.md](requirements.md) §17 and [master-plan.md](master-plan.md). Everything else is a proposal. Items marked "?" are open or not researched. Checks done on 2026-11-11 are in "Checks done".

Scope: the foundation (milestone M0) and the interfaces that M1 to M4 depend on. Screens other than chat are not designed. The chat screen is in [foundation.md](foundation.md).

## Decided

- Flutter client, one codebase for Android and web.
- Go backend, one process, SQLite, one container on Coolify.
- The server owns the data and validates every command. The app needs a connection.
- Push through Firebase Cloud Messaging.
- Model and speech keys stay on the server.
- Every assistant request that creates or changes data needs Confirm.
- One command interface carries every change. Cost is at most €10 per month.
- Public HTTPS behind Cloudflare Access. Web signs in through the browser. Android signs in through Cloudflare managed OAuth, which opens the browser once. The Go service verifies the Cloudflare token.
- A reminder is a notification and a chat message. Acting on either stops the nag.
- Done and Snooze act at once, with an Undo button.
- The model is Claude Haiku 5.5 through OpenRouter. One OpenRouter key serves the model and the speech model.
- Voice uses a speech model through OpenRouter, English only.

## System

```
Flutter app (Android, web)
   │  HTTPS: commands, queries, event stream
   ▼
Go service ── command layer ──▶ SQLite
   │             ▲  ▲
   │             │  └── reminder engine ──▶ FCM ──▶ phone
   │             └───── assistant ──▶ model and speech APIs
   └── audit log, backup job
```

One process runs four parts: the HTTP API, the command layer, the reminder engine and the assistant. They share one SQLite file.

## Go service

### Packages (proposal)

| package | owns | depends on |
|---|---|---|
| `domain` | item, list, label, profile, reminder, pending change; validation rules | nothing |
| `store` | SQLite access, migrations, transactions | `domain` |
| `command` | the command set, validation, preview, apply, audit log, undo | `domain`, `store` |
| `engine` | reminder scheduling, nag state, catch-up | `command`, `store`, `push` |
| `push` | FCM sender behind an interface | nothing |
| `assistant` | message to command proposal, model and speech clients behind interfaces, cost meter | `command` |
| `api` | HTTP routes, auth check, event stream | `command`, `assistant`, `engine` |
| `cmd/minder` | wiring and startup | all |

Rule: `api`, `engine` and `assistant` change data only through `command`. No package writes SQL except `store`.

### Command interface

- A command has a name, a typed payload and an actor (`owner` or `agent`).
- Commands (first set): create item, update item, set status, delete item, create list, set label, change profile, acknowledge reminder, snooze reminder.
- Every command runs in two steps:
  1. **Preview.** The server validates the payload and returns the old and new values. It stores a pending change with the version of each item it reads.
  2. **Apply.** Confirm applies the stored pending change in one transaction and writes an audit record. A pending change never expires. If an item changed after the preview, Confirm fails with an error and the assistant offers a new preview.
- Done and Snooze skip the preview and apply at once. The result shows an Undo button.
- A rejected command returns an error that names the field and the fix. The assistant uses the same error text.
- Undo applies the inverse of an audit record. Every command defines its inverse.

### Data model (outline)

Tables: `item`, `list`, `label`, `item_label`, `profile`, `reminder`, `nag_state`, `pending_change`, `audit`, `push_device`, `usage` (model and speech cost). Field list: requirements §3.1. Full schema is a separate design.

Rules:
- Times are stored in UTC with the IANA zone name of the item. Default zone Europe/Amsterdam.
- Statuses, types and nag modes are enums in the schema, not free text.
- Every row has `created_at` and `updated_at`. Items have `completed_at`.

### Reminder engine

- A reminder has a due time, an item and a nag state (next fire, count, acknowledged or not).
- One loop wakes at the next due time. It does not poll every second.
- After a restart the loop reads all unacknowledged reminders with a due time in the past. It sends each one marked late (requirements §6.2).
- A fired reminder creates a message for the chat and a push message. The engine repeats it at the nag interval until you acknowledge it. An escalating nag shortens the interval.
- Target: a reminder fires within 1 minute of its time (requirements §14).
- Tests use a fake clock and a fake push sender.

### Assistant

- Input: text, or audio that the speech client turns into text. Output: a list of commands with a preview, or an answer to a question.
- The model client and the speech client are interfaces. The text model is Claude Haiku 5.5 and the speech model is a separate model. Both go through OpenRouter.
- The assistant never writes data. It calls the preview step and returns the result. Confirm goes to the command layer.
- A cost meter records tokens and audio seconds per request in `usage`. The service refuses a request when the month total would pass the limit The limit is €5 per month for model and speech together.

### HTTP API (proposal)

- `POST /v1/commands`: send a command; returns a preview or an applied result.
- `POST /v1/commands/{id}/confirm` and `.../discard`: act on a pending change.
- `GET /v1/items`, `GET /v1/lists`: queries return structured JSON.
- `POST /v1/chat/messages`: send text or audio; returns the assistant reply.
- `GET /v1/events`: server-sent events. The open app holds one connection. The server writes an event for each new chat message and nag, and a ping every 20 seconds.
- `GET /v1/chat?after=<id>`: chat entries after an id. The app calls it after a dropped connection to fetch what it missed.
- `POST /v1/devices`: register an FCM token.
- All bodies are JSON. Errors carry a code, a field and a short message.

### Access and auth

- Cloudflare Access sits in front of the service. The Go service verifies the signed `Cf-Access-Jwt-Assertion` header on every request: signature, expiry and the application audience tag. It accepts only the owner's identity. Minder stores no passwords. Cloudflare recommends the header over the `CF_Authorization` cookie.
- Web: the browser signs in and Cloudflare sets the cookie. The Flutter web build runs on the same origin as the API, so requests carry it.
- Android: a native app cannot follow the browser redirect. Cloudflare managed OAuth fixes this. Access returns 401 with discovery metadata, the app runs an authorization code flow in the system browser, and Access issues an opaque access token. The app sends it on each request. Cloudflare turns it into the signed header for the Go service.
- Defaults: access token 15 minutes, refreshed with a refresh token. The grant session sets how long until you sign in again. Cloudflare suggests 1 to 2 weeks for agents. A native app uses a private-use redirect scheme that is listed exactly in the Access application.
- The client must support RFC 8707 (resource indicator). Whether a Flutter OAuth library supports this and dynamic client registration is not verified. The first M0 spike proves it. The fallback is a Cloudflare service token that you paste into the app once.
- Without a valid token the service answers 401 and the app starts the sign-in again.
- FCM push does not pass through Cloudflare. The app registers its device token after sign-in.
- The Android Done and Snooze notification actions call the API in the background. They need a valid access token. If it has expired, the app refreshes it. If the refresh token has expired too, the action opens the app to sign in.

### Cloudflare setup (done 2026-11-11)

| item | value |
|---|---|
| Hostname | `minder.liveleds.io`, A record to the Coolify server, proxied |
| Why not `*.io.liveleds.io` | Those records are DNS-only, so traffic skips Cloudflare and Access. They also lie outside the free certificate |
| Team domain | `liveleds.cloudflareaccess.com` |
| Access application | "Minder", self-hosted, id `5eaa7708-2f25-43e6-b570-af5422bd5941` |
| Audience tag | `11f11785388ae33f2f78007aa26db1bf047457ff7f19758eddd797ef4c151c54` (an identifier, not a secret) |
| Policy | "Owner only": allow one email, `alexanderp4580@gmail.com` |
| Sign-in | One-time PIN by email. Session 24 hours |
| Managed OAuth | On. Dynamic client registration on. Redirect `io.liveleds.minder:/oauth/callback`. Access token 15 minutes. Grant session 336 hours (14 days) |
| Zone SSL mode | Full |

Observed on 2026-11-11: a request without a token gets `401` with a `WWW-Authenticate` header and OAuth discovery metadata from Cloudflare. Nothing runs at the origin yet.

Rules:
- The Go service must still verify the token. A request sent straight to the server IP skips Cloudflare.
- The application ID in the redirect URI must match the Android application ID. Changing one changes the other.
- The Cloudflare API token lives in TokenCrypt as `cloud_flare_access_token`. It can edit Access applications and DNS on the zone.

### Backup

- Coolify has two backup types that apply. Its SQLite database resource has manual and scheduled backups, and the docs mark SQLite support as experimental. Its storage backup archives volume and directory mounts as `.tar.gz` files on a schedule (default daily, keeps 7).
- Minder runs its own Go service with its own SQLite file, so the storage backup applies. The docs warn that an archive taken during writes can be inconsistent.
- Design: a daily job inside the service writes a consistent snapshot (SQLite online backup) plus a JSON and `.ics` export into a backup directory. Coolify archives that directory. The live database file is not archived.
- Coolify stores archives on the same server by default. It can also upload to S3-compatible storage, SFTP, FTPS or WebDAV. You chose Coolify backups as enough. Backups stay on the same server. You accepted this risk.
- Coolify does not restore storage archives from the dashboard. A restore is manual: download, unpack, replace the file. Test one restore in M5.

## Flutter app

### Layers (proposal)

| layer | role |
|---|---|
| `data` | API client, DTOs, FCM token registration, event stream client |
| `domain` | models and repositories (interfaces) that the UI uses |
| `application` | controllers that hold state: conversation, reminders, connection |
| `ui` | screens and widgets built from the foundation tokens |

Rules:
- The UI never calls the API client. It reads state from controllers.
- The app holds no business rules. The server validates. The app shows the error text it gets.
- No data without a connection. The app shows an error and Retry.

### State and packages

- State management: Riverpod. Controllers are plain classes that tests run without widgets.
- HTTP: Dart `http`. JSON models: `freezed` and `json_serializable` (? not researched).
- Push: `firebase_messaging` and `flutter_local_notifications`. See the M2 spike.
- Fonts: Michroma and Chakra Petch as bundled assets, not loaded from the network in the app.
- Icons: Carbon icons as SVG assets (? package choice).

### Design tokens

One `tokens.dart` holds the colours, type sizes, spacing and shapes from [foundation.md](foundation.md). Widgets read tokens only. No hex value appears in a widget.

### Chat controller

The conversation is a list of entries: Minder message, your message, event line, nag frame, preview card. One controller owns the list.

States of one request: `idle`, `sending`, `waiting` ("Working"), `preview`, `error`. Voice adds `recording` and `transcribing`.

- Send: add your message, call the API, add the reply.
- Preview: show the card. Confirm calls confirm and replaces the card with the result and an Undo button. Discard removes the card.
- Events: the controller listens to the event stream while the app is open. After a drop it reconnects and fetches entries after the last known id. A reminder adds a nag frame. Done and Later call acknowledge and snooze.
- After an offline period the controller asks for entries since the last known id.

### Platform differences

| area | Android | Web |
|---|---|---|
| Push | FCM, notification actions | none in v1 (? web push later) |
| Voice | microphone permission, record to file | browser microphone, needs HTTPS (the site is HTTPS) |
| Reminders closed | server pushes | web shows reminders only while open |

### Tests

- Unit tests for controllers with a fake repository.
- Golden tests for the chat entries against the foundation tokens.
- An integration test on the web build against a local Go service.
- Android notification actions: a manual check on a real phone.

## Repository layout (proposal)

```
server/   Go module: cmd/minder, domain, store, command, engine, push, assistant, api
app/      Flutter project: lib/{data,domain,application,ui}, assets/
docs/     specs and mockups
```

Each folder gets its own `AGENTS.md` when it is created.

## Build and deploy

- One Dockerfile for the Go service. SQLite file on a mounted volume. Coolify deploys from Git.
- The Flutter web build is static files. The Go service serves them from the same container and origin as the API.
- The toolchain is pinned in a container so the Steam Deck and CI build the same way (master-plan M0).
- APK delivery through the existing Telegram bot and chat (root `AGENTS.md`).
- Secrets come from the environment of the container. None are in the repository.

## M0

The M0 scope, tasks and acceptance criteria are in [m0-spec.md](m0-spec.md).

## Checks done on 2026-11-11

| item | result | source |
|---|---|---|
| Haiku 5.5 price | OpenRouter lists `anthropic/claude-haiku-5.5` with tool support at $0.10 per million input tokens and $0.50 per million output tokens for prompts up to 100,000 tokens. Over that, five times more. A chat request is far below 100,000 tokens | [Anthropic pricing](https://platform.claude.com/docs/en/about-claude/pricing) |
| Speech on OpenRouter | Exists. `POST /api/v1/audio/transcriptions` takes base64 audio, a model name and an optional `language` code. Multipart upload cap 25 MB. Upstream timeout 60 s per request. Models are listed with `GET /api/v1/models?output_modalities=transcription` | [OpenRouter speech-to-text](https://openrouter.ai/docs/guides/overview/multimodal/stt) |
| Speech price | The Models API lists about 20 models. Prices look like USD per second of audio: from 0.0000033 (Qwen3 ASR 0.6B) to 0.0001 (Fish Audio). At 0.00005 per second, a 10-second message costs about $0.0005. Units need confirming on one real call | OpenRouter Models API |
| Cloudflare and streams | Cloudflare streams by default and does not buffer. Reports say a quiet stream drops near 100 seconds. A ping every 20 seconds avoids it. No source covers Access with streams, so M0 tests it | [Cloudflare response buffering](https://developers.cloudflare.com/network/response-buffering) |
| Android sign-in | Managed OAuth is the supported way for a non-browser client. Details in "Access and auth" | [Managed OAuth](https://developers.cloudflare.com/cloudflare-one/access-controls/applications/http-apps/managed-oauth/) |
| Token check in Go | Validate the `Cf-Access-Jwt-Assertion` header against the application audience tag and the team's signing keys | [Validate JWTs](https://developers.cloudflare.com/cloudflare-one/access-controls/applications/http-apps/authorization-cookie/validating-json/) |
| Coolify backup | See "Backup" | [SQLite](https://coolify.io/docs/databases/sqlite), [storage backups](https://coolify.io/docs/core/persistent-storage/storage-mounts/backups) |
| Push on Android | FCM delivers high-priority messages at once, even in Doze. Normal priority waits. FCM lowers the priority of an app whose high-priority messages show no notification to the user. Action buttons need `flutter_local_notifications`. Use a data message, handle it in a background callback, and show the notification there | [FCM priority](https://firebase.google.com/docs/cloud-messaging/android/message-priority), [FlutterFire](https://firebase.flutter.dev/docs/messaging/notifications) |

Cost example: a request with 2,000 input tokens and 300 output tokens costs about $0.00035. 1,000 requests cost about $0.35. The €5 limit is not a constraint at personal volume.

## Choices made from the checks

- HTTP client: Dart `http` (my choice, the official package, no research). Push: `firebase_messaging` with `flutter_local_notifications`.
- The reminder push is a high-priority data message. The background callback shows the notification with Done, Snooze and Open.
- The speech model is chosen from the Models API in M4. Start with a low-price model and compare accuracy on 20 real English messages.

## Spikes by milestone

| milestone | spike |
|---|---|
| M0 | Flutter Android signs in through Cloudflare managed OAuth and calls the Go service. Fallback: service token |
| M0 | A test client holds the event stream through Cloudflare Access for 5 minutes. Fallback: poll every 5 seconds |
| M2 | A real phone receives a high-priority data message in Doze and shows Done, Snooze and Open. Taps reach the Go service |
| M4 | One speech call through OpenRouter returns text and a cost value |
| M5 | The backup job writes a snapshot, Coolify archives it, and one manual restore works |

## Open decisions

- None.
