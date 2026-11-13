# Minder architecture

Current source layout on 2026-11-12. Requirements: [requirements.md](requirements.md).
Design tokens: [foundation.md](foundation.md). Screens: [screens.md](screens.md).

## Runtime

The Flutter client runs on Android and web. One Go process serves the web build and HTTP API.
SQLite holds items, profiles, chat entries, pending changes, audit records, reminder schedules, devices and usage.
The reminder worker sends Android notifications through Firebase Cloud Messaging.
The assistant calls OpenRouter for text replies and proposed changes. Opt-in email capture uses the same model budget for isolated source extraction.
A Cloudflare Email Worker hands raw MIME directly to machine-authenticated ingress; the Go worker reads persisted sources and publishes reviewed candidates in chat.

The client needs a server connection. The server validates item changes before saving them.
Model and Firebase service account keys stay on the server.
Coolify runs the container with a persistent database volume.

## Go packages

| Package | Responsibility | Project dependencies |
|---|---|---|
| `domain` | Values, validation, recurrence, reminders, identifiers and profile defaults | None |
| `port` | Atomic transaction and persistence contracts | `domain` |
| `store` | SQLite adapter, migrations, transactions and projections | `domain`, `port` |
| `command` | Preview, Confirm, Discard, Undo and conflict refresh | `domain`, `port` |
| `assistant` | Context, proposals, cost reservations and model contract | `domain` |
| `conversation` | Message acceptance, concurrency, replies and cancellation | `domain` |
| `reminder` | Scheduling, actions, device validation and delivery worker | `domain`, `port` |
| `openrouter` | HTTP model adapter and tool schema | `domain`, `assistant` |
| `serper` | Serper places HTTP adapter | `domain`, `assistant` |
| `push` | FCM adapter and token classification | None |
| `access` | Owner authentication and Cloudflare JWT adapter | None |
| `config` | Process settings and startup validation | None |
| `api` | HTTP routes, responses, event stream and static files | `domain`, `conversation` |
| `capture` | Durable source ingestion, extraction jobs, isolated questions and review publication | `domain`, `port` |
| `mailparse` | Bounded MIME, calendar, PDF and image conversion | `domain` |
| `mailfiles` | Private atomic original/attachment files | None |
| `mailauth` | Signed Worker envelope/body admission and environment allowlists | `domain`, `config` |
| `cmd/minder` | Composition, signals and shutdown | Application services and adapters |

Only `store` executes SQL. Application services use repository interfaces and `port.Transaction`.
The SQLite adapter implements these contracts and rolls back all writes if a transaction callback fails.
Domain and application packages have no SQLite, HTTP client or provider SDK imports.

The API accepts query, command, message, reminder and authentication interfaces.
`conversation` validates and accepts messages, runs one reply at a time and records safe errors.
`Close` cancels and waits for active replies. Process shutdown also waits for the reminder worker before closing SQLite.
`openrouter` implements the assistant model interface. `serper` implements the optional place finder: the model may offer a lookup, and the server searches only after the owner taps Look it up on a fresh offer. `push` implements the reminder sender interface.

### Changes and storage

Item and profile proposals create durable previews. Confirm checks the stored version before applying a change.
A version conflict can return a replacement preview. Previews have no expiry.
Confirm records the change and audit entry in one transaction. Undo checks the current version before restoring data.
Done and Snooze on reminders act without a preview.
`domain.Complete` handles completion from chat and reminder actions, including recurrence.

Numbered SQL files in `store/migrations/` define schema upgrades.
`PRAGMA user_version` records the schema version. Startup refuses a database from a newer release.
Item, profile and chat rows contain JSON. Reminder and delivery tables hold scheduling state.
A durable event cursor records entry updates for reconnects.

The server stores timestamps in UTC and retains the item's IANA timezone. Events can retain separate start/end zones and an explicit end; date-only ranges remain local calendar dates and use an exclusive end. Calendar queries include overlap, with an optional viewing timezone.
The default item timezone is Europe/Amsterdam.
`domain.Anchor` uses the deadline, then the start, for reminders and recurrence.
`domain.PlacedAt` uses the start, then the deadline, for the calendar.

### Assistant and costs

The model proposes item and profile changes. The command service validates them before creating previews.
A message can produce up to five proposals. One invalid proposal rolls back the preview batch.
Context contains at most 200 open items, plus lists, profiles and recent chat entries.

The monthly limit defaults to €5. The conversion rate defaults to one euro per dollar.
Before each model call, the server reserves €0.02 times that rate.
The server settles the reservation from provider usage. Unknown costs retain the reservation.
Startup rejects zero, negative and non-finite limits or rates.
Production rejects a custom model endpoint.

### Reminders

The application reminder service processes due schedules. The worker runs it every 15 seconds and catches up after a restart.
A due reminder creates a chat entry and queues device deliveries.
Nag intervals repeat the reminder until the owner acts.
The worker checks that each delivery is current before sending it.
The worker chooses retry times. The SQLite adapter records delivery state.
FCM delivery failures retain retry state. An unregistered FCM token disables that device.

## HTTP and authentication

`api.New` registers method and path patterns. Unknown `/v1/` paths return 404.

| Method and path | Purpose |
|---|---|
| `GET /v1/health` | Health check, without sign-in |
| `GET /brand/minder-icon.png` | Public OAuth branding image |
| `GET /v1/chat?after=&limit=` | Chat history or changes after a cursor |
| `POST /v1/chat/messages` | Accept text and start an assistant reply |
| `GET /v1/events?after=` | Stream entry changes and pings |
| `GET /v1/items?from=&to=` | Calendar items across at most 62 days |
| `POST /v1/changes/{id}/confirm` | Apply a preview |
| `POST /v1/changes/{id}/discard` | Discard a preview |
| `POST /v1/undo/{id}` | Undo an audited change |
| `GET /v1/reminders` | Active reminder entries |
| `POST /v1/reminders/{id}/{action}` | Act on a reminder |
| `GET /v1/push/config` | Public Android Firebase identifiers, after sign-in |
| `POST /v1/push/devices` | Register a device token |

The opt-in `POST /v1/email/ingest` route requires dedicated Worker authentication: signed envelope/body, recipient and sender admission, plus the path-scoped Access service-token policy. It never grants owner privileges.
`GET /v1/captures/{id}`, its owner answer/retry/deletion-preview routes, and source attachment downloads use owner authentication.
All other routes require owner authentication in production, including the web client and `/branding/`.
The branding route serves three named assets from `BRAND_DIR`.
API writes with an Origin header must come from the same origin.
Errors expose fixed messages for model and internal failures.

The `access` adapter reads `Cf-Access-Jwt-Assertion` and validates its signature, issuer, audience, expiry and owner email.
The API depends on `Authenticate(*http.Request) error`; it has no Cloudflare header or configuration logic.
Authentication settings have no identity defaults. [README.md](../README.md) lists the required environment variables.
Development authentication requires `ENV=development` or `ENV=test`.

Web sign-in uses the browser session. Android uses managed OAuth through the system browser.
The Android adapter handles PKCE, state checks, registration, token storage and refresh.
Its callback is `io.liveleds.minder:/oauth/callback`.

The event stream sends pings every 20 seconds and checks the database every 200 milliseconds.
The client reconnects from its cursor and polls every five seconds as a fallback.

## Flutter layers

| Layer | Responsibility |
|---|---|
| `domain` | Typed models and repository/device interfaces |
| `data` | HTTP, JSON decoding, OAuth and the Android notification channel |
| `application` | Chat, calendar and notification state controllers |
| `ui` | Screens, entry widgets, display formatting and design tokens |
| `main.dart` | Riverpod wiring, navigation and app lifecycle |

Domain models contain no JSON factories or transport maps.
`data/wire.dart` maps server responses to domain values.
`data/notification_device.dart` maps native channel messages to typed registration and reminder actions.
Controllers use domain repositories directly. The notification controller reads typed device values.
ProviderScope closes shared API/auth clients; controllers close only their own subscriptions and callbacks.
The calendar controller ignores results from superseded requests.

`Tokens.theme` defines Material styles and standard visual density.
The app bundles Michroma and Chakra Petch fonts. It uses Material icons.
Chat and calendar goldens check rendering. Controller tests use repository fakes.
HTTP and notification tests exercise transport boundaries with controlled responses.

## Build and checks

The Dockerfile builds Flutter web and Go, then copies both into one runtime image.
`toolchain/run` pins Go and Flutter in Podman containers.
Local browser checks use compiled Flutter web, the Go service and temporary SQLite databases.
They cover chat changes, Undo, reminders and calendar navigation.

Architecture cleanup evidence: [evidence/architecture-cleanup.md](evidence/architecture-cleanup.md).
`toolchain/architecture-check` enforces package and layer imports. `toolchain/dox-check` validates documentation ownership.
Product verification: [evidence/m0.md](evidence/m0.md) and [evidence/m1.md](evidence/m1.md).
Phone approval gates merging. The handoff records deployment and APK status.

## Email source lifecycle

The complete contract is in [email capture](superpowers/specs/2026-11-12-email-capture-design.md); operating steps are in [email capture operations](email-capture-operations.md).
SQLite capture rows hold job leases, candidate revisions, answers and attachment manifests. Unique receipts and item-source links share the existing transaction boundary.
The source directory holds opaque private files. Ingress commits source and job before acknowledgment; there is no Cloudflare content buffer.
Extraction creates previews, never final items. Targeted answers invalidate affected previews. Confirm/Discard update capture state, audit, source links and reminders atomically.
Source deletion commits a tombstone/redaction before restart-safe file cleanup. Generic review notifications use a durable per-revision/device outbox.
CaptureController and typed client repositories keep transport details out of the UI. Source updates do not clear an unrelated chat request's waiting state.
