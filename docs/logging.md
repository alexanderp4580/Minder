# Server logging

The Go service writes structured logs to standard error. Coolify shows them on the Logs tab of the Minder application.
Use this page to find out what the server did and why. The code contract is in [server/AGENTS.md](../server/AGENTS.md).

## Settings

| Variable | Values | Default |
|---|---|---|
| `LOG_LEVEL` | `debug`, `info`, `warn`, `error` | `info` |
| `LOG_FORMAT` | `text`, `json` | `text` |

An unknown value stops startup with an error that names the variable.
Info records each request, state change, model call, push attempt, reminder event and lifecycle event.
Debug adds transactions, saved entries, stream entries, worker ticks and successful sign-ins.
Health checks log at debug. Set `LOG_LEVEL=debug` in Coolify only while you investigate, then remove it.

## What the logs never hold

- Message text, item titles and notes. Logs carry lengths and IDs instead.
- Push tokens, JWTs, API keys and the Firebase service account.
  A push token appears as `token_fp`, the first four bytes of its SHA-256 hash.
- The mailbox of a rejected sign-in. Logs carry its domain.

## Correlation

- `request_id` ties one HTTP request together. The server returns it in the `X-Request-Id` header.
- `reply_id` and `trace_id` tie the lines of one assistant reply, from `message accepted` to `reply finished`.
- `item_id`, `change_id`, `occurrence_id` and `device_id` follow one object through its life.

## Events

Each line has `time`, `level` and `msg`. These fields appear on the lines named.

| Component | Messages | Key fields |
|---|---|---|
| Startup | `server configuration`, `database opened`, `migration applied`, `firebase push configured`, `Minder listening` | `revision`, `model_key_set`, `firebase_service_account_set`, `schema_version`, `migrations_applied` |
| Shutdown | `shutdown requested`, `http server stopped`, `background worker stopped`, `conversation closed`, `database closed` | `waited_ms` |
| HTTP | `http request`, `http panic`, `event stream opened`, `event stream closed` | `method`, `route`, `status`, `duration_ms`, `auth`, `error`, `error_code`, `entries_sent`, `reason` |
| Sign-in | `authentication rejected`, `signing keys fetched`, `signing key fetch failed` | `reason` is `missing_token`, `invalid_token` or `wrong_owner`; `detail` |
| Messages | `message accepted`, `message rejected`, `reply started`, `reply finished`, `reply failed`, `reply panicked` | `text_len`, `reply_id`, `kinds`, `error` |
| Assistant | `assistant reply started`, `usage reserved`, `usage settled`, `assistant reply finished`, `assistant reply failed`, `assistant reply blocked` | `items_in_context`, `prompt_bytes`, `cost_eur`, `month_cost_eur`, `limit_eur`, `stage` |
| Model | `model request`, `model response`, `model request failed`, `model reply rejected` | `model`, `status`, `finish_reason`, `tool_calls`, `prompt_tokens`, `cost_usd`, `stage`, `reason`, `body` |
| Changes | `change previewed`, `change confirmed`, `change confirm repeated`, `change discarded`, `change undone`, `change refreshed`, and the matching `rejected` lines | `action`, `change_id`, `item_id`, `version`, `fields`, `error_code` |
| Items | `item saved`, `item deleted`, `reminder schedule set`, `reminder schedule cancelled` | `reminder_active`, `reminder_at`, `nag_minutes`, `status`, `created` |
| Reminders | `reminder fired`, `reminder action`, `reminder action rejected` | `scheduled_at`, `late_seconds`, `next`, `action`, `occurrence_id` |
| Push | `push device registered`, `push device replaced`, `deliveries queued`, `push delivered`, `push sent`, `push delivery failed`, `push rejected`, `push failed`, `push device removed` | `device_id`, `token_fp`, `devices`, `status`, `fcm_status`, `fcm_error_code`, `token_invalid`, `stage` |
| Worker | `reminder worker started`, `reminder worker heartbeat`, `reminder worker tick failed` | see Heartbeat |
| Store | `slow transaction`, `usage reservation refused`, `transaction commit failed` | `duration_ms`, `waited_ms` |

## Heartbeat

The reminder worker logs `reminder worker heartbeat` at startup and every five minutes.

| Field | Meaning |
|---|---|
| `push_sender` | `true` when Firebase is configured |
| `items`, `devices` | Stored items and registered push devices |
| `schedules_active` | Reminders that have not fired yet, or that nag |
| `occurrences_active` | Fired reminders still waiting for Done, Open or a snooze |
| `deliveries_pending`, `deliveries_due` | Pushes not yet accepted by Firebase, and those due now |
| `oldest_due_seconds` | Age of the oldest due push. A growing value means pushes are stuck |
| `next_schedule_at` | Time of the next reminder |

## A reminder did not arrive

Follow these steps in order. Stop at the first step that fails.

1. Find `item saved` for the item. `reminder_active=false` means no reminder was set.
   The item needs `remind_at` or a profile that sets one. A due time alone sets none.
2. Find `reminder schedule set` for the same `item_id`. Check `at` and `nag_minutes`.
3. Find `reminder fired` at that time. Without it, check the heartbeat for a running worker.
4. Read `deliveries queued`. `devices=0` means no phone is registered. Enable notifications in the app.
   The server also logs `no push devices registered; reminder stays in chat only`.
5. Find `push delivered` for the `device_id`. This line means Firebase accepted the message.
6. On `push delivery failed` or `push rejected`, read `fcm_status` and `fcm_error_code`.
   The worker retries every minute. `push device removed` means Firebase rejected the token as `UNREGISTERED`.
   Open the app and enable notifications again to register a new token.
7. If Firebase accepted the message and nothing showed on the phone, the fault is on the phone: notification permission, battery restrictions or a stale install.

## A reply failed

1. Find `message accepted` and note `reply_id`.
2. Search for that `reply_id` or `trace_id`. The last error line names the stage.
3. `model request failed` shows the provider status and the first 512 bytes of its error body.
   `model reply rejected` shows why the server refused a reply: `unreadable`, `truncated`, `no_choice` or `cost_missing`.
4. `assistant reply blocked` with `reason=monthly_limit` shows the spend and the limit.
