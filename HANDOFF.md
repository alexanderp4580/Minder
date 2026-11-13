# Handoff

State on 2026-11-12. Production runs commit `2033008` from branch `main`.
Branch `m0-chat` is merged into `main`. The worktree `Minder-m0-chat` is on branch `notification-body-tap`, which has unmerged work (see "Open work").

## Current source

- Chat, item and profile previews, Confirm, Discard, Undo and open-item queries.
- Lists, profiles, recurrence, reminders, nagging and Android FCM actions.
- Start, deadline, duration, priority and location fields.
- Read-only calendar with a month grid, spaced week rows and a scrolling day agenda.
- Theme spacing, standard visual density, shared Material styles and golden tests.
- Animated orbital backgrounds with distinct Calendar and Chat compositions, 60 percent dimmer than in APK 0.4.2.
- A two-item bottom bar (Chat, Calendar). Headers show a title and a rule that fades left to right.
- Structured server logs for every request, change, model call, reminder and push. See [docs/logging.md](docs/logging.md).
- Private `/branding/` guide and public OAuth logo.

The owner approved M0 and reminder delivery on a phone.
The owner requested completion of M1 and the calendar without questions.
The owner asked for the merge to `main` and the deployment on 2026-11-12. Work ran inline, without sub-agents.

## Architecture and DOX

All six approved architecture cleanup steps are complete. The local checks passed; source and deployment status remain separate.

The Go domain and application services depend on interfaces, not SQLite or provider adapters.
`port.Transaction` preserves atomic item, reminder, event and audit changes.
Reminder scheduling and actions run in `reminder`; domain rules choose snooze times and recurrence.
`conversation` owns message acceptance, busy state, replies and cancellation.
`openrouter`, `push`, `access`, `store` and `api` are adapters. Main composes them and waits for shutdown.

Flutter controllers use domain repositories directly. Data adapters own HTTP, JSON, OAuth and Android messages.
ProviderScope owns shared API/auth clients. Disposed controllers stop their own work without closing shared repositories.

Agent0ai DOX now covers Go packages, Flutter layers, platform folders, documentation and tooling.
`toolchain/check` enforces dependencies and DOX ownership, then runs the local suites.
Architecture: [docs/architecture.md](docs/architecture.md).
Evidence: [docs/evidence/architecture-cleanup.md](docs/evidence/architecture-cleanup.md).

`CF_ISSUER`, `CF_AUDIENCE` and `OWNER_EMAIL` are configured in Coolify as runtime-only variables.
Read-back matched the live Cloudflare app and owner policy. They apply on the next deployment.
`cmd/minder` contains no Cloudflare identity settings.

## Calendar spacing

The owner approved the roomier calendar preview. The Flutter page, brand guide, foundation,
calendar spec and HTML mockup now share the same layout.
The page uses 24 px gutters, 8 px week gaps, a separate month/year heading and 20 px card padding.
Today sits beside the agenda date. The whole page scrolls on short screens.
Narrow-screen checks cover 320 and 360 px with 1.4× text; day targets remain at least 44 px wide.
Evidence: [docs/evidence/calendar-spacing.md](docs/evidence/calendar-spacing.md).

## Orbital backgrounds

The owner chose Orbital and approved animated backgrounds for Chat and Calendar, plus APK delivery.
`OrbitalBackground` displays fixed artwork exported from the canonical SVGs in brand/.
Calendar follows measured reference curves; Chat uses a shifted, tighter composition.
Atmosphere, edge gradients and soft glow stay still. Short highlights travel along cached paths over 180 seconds.
Static artwork and foreground are isolated from highlight repaints.
Reduced motion keeps the scene still. Hidden routes and backgrounded apps stop ticking.
Tests verify movement, lifecycle pauses, distinct scenes and stable foreground builds.
Brand guide, foundation, mockups and goldens follow the approved direction.
Phone frame timing and battery use remain unmeasured.
The owner approved reconstructing the artwork and moving highlights along its fixed paths.
The SVGs own geometry and lighting; toolchain/export-orbits.py keeps Flutter and HTML exports aligned.
The calendar spacing and opaque cards remain unchanged.

## Open work

### Branch `notification-body-tap` (not merged, not pushed, not deployed)

- `be4aeff`: the reminder notification has no content intent. Tapping its body does nothing.
- `6c115a5`: Done and Snooze run in the background. `ReminderActionReceiver` starts a headless Flutter engine at the Dart entry point `notificationAction` and calls the reminder action with the app login. Open still opens the app.
- App version 0.4.8+17. APK 0.4.8 reached Telegram once (message 1975). The server is unchanged.
- Verified on the emulator with a direct broadcast, not a rendered button. Evidence and limits: [docs/evidence/notification-actions.md](docs/evidence/notification-actions.md).
- Not checked: a physical phone, Android 12 and later background limits, battery restrictions, and the failure message with notifications on.
- Merge after the owner approves the buttons on a phone. No deployment is needed.

### Keyboard auto-scroll fix

`ChatScreen` now follows list metrics after layout while pinned to the bottom.
Keyboard opening and stepped dismissal, the returning bottom bar and composer resizing keep the latest content visible.
User scrolling away pauses viewport following. Returning to the bottom or a new message cursor resumes it.

The regression test reproduced a 244 px gap after keyboard opening before the fix.
It covers sending during keyboard dismissal, preserving the reading position, resuming at the bottom and receiving a reply.
Verification: `toolchain/check` exited 0 on 2026-11-12: 56 Flutter tests, analyze, format, Go race tests/vet, toolchain/hook tests, architecture, DOX and orbital export checks.
Physical phone verification remains pending. This change has no APK delivery or deployment yet.
The email extractor allows one evidence correction per batch (`96d1778`); see the email capture section.

Leftovers on this machine:

- A local `minder` server on `127.0.0.1:18190` and `toolchain/model_fixture.py` on port 8181 may still run. Stop them by process name, not with `pkill -f`, which also kills the calling shell.
- The emulator `emulator-5584` has the debug x86_64 build 0.4.8 installed over 0.1.1, and a port forward for 18190. Remove it with `adb reverse --remove-all`. `adb unroot` was already run.

Send each APK once. A 504 from Telegram can mean the file arrived.

## Email capture implementation

Local app version: 0.5.1+19 on `notification-body-tap`. Production runs `fb99b50` with email capture live: Worker `minder-email-ingress`, routing for the `minder` subdomain and an Access service-token policy. A real forward from an allowed Gmail address reached the server and produced one candidate on 2026-11-12 (Worker log: all stages, status 201). APK 0.5.0+18 reached Telegram (message 1990). App 0.5.1+19 adds the Do nothing button and clearer zero-item wording. Phone review, attachment download and Confirm remain unverified.
The owner approved the specification and inline implementation. The Worker, Go ingestion/parser/review pipeline, Flutter review flow and deployment runbook are present.

Minder retains originals and attachments, asks targeted follow-up questions and requires Confirm for each item change. Travel items preserve both time zones and hotel checkout dates. Sources remain accessible from calendar items.
The Worker delivers directly to Minder without a Cloudflare content buffer. Both ends require private `EMAIL_ALLOWED_SENDERS` configuration and aligned sender authentication; source values stay outside Git.

Verification and limits: [email capture evidence](docs/evidence/email-capture.md).
Activation, private environment parameters, persistence and rollback: [operations](docs/email-capture-operations.md).
Live Cloudflare/DNS/Access setup, provider account readback and phone approval remain pending. No emulator is available for native attachment opening.
Keep the branch for phone review; preserve the earlier keyboard and notification work.

## Navigation, header and composer

The bottom bar has two 56 px items. The selected item has a Signal icon and a 2 px Signal bar on its top edge.
The bar hides while the keyboard is open. A system inset sits below the bar.
The header rule fades from 50 percent Text colour at the left to clear at the right.
The composer is 68 px high: 8 px above the field, 16 px below it.
Evidence: [docs/evidence/navigation-and-logging.md](docs/evidence/navigation-and-logging.md).

## Logging

The server logs with `log/slog`. `LOG_LEVEL` and `LOG_FORMAT` set level and format.
The reminder worker logs a heartbeat every five minutes with the device count, active schedules and due pushes.
Logs never hold message text, item titles, tokens or keys.
Reference and tracing steps: [docs/logging.md](docs/logging.md).

## Deployment and APK status

Production runs `2033008`, deployed on 2026-11-12 as Coolify deployment `wehc4yesilvupuyue3pzpp4v`. The app is `running:healthy`.
The Coolify application builds from `main`. It built from `m0-chat` before this deployment.
The first production heartbeat reported one registered device, five items and no active schedule.
Notifications did not arrive on 2026-11-12. The old build logged nothing that explains it. Watch the next reminder with the new logs.

APK 0.4.6 carries the bar, the header rule, the composer spacing and the dimmer lines. Telegram confirmed one upload.
The build uses the cached debug signing key, so it installs over earlier builds. Phone approval of the look remains pending.
Earlier APK evidence: [docs/evidence/orbital-artwork.md](docs/evidence/orbital-artwork.md).

The Coolify app is `ciw3jk5gkgiql8kfdomeyys2`.
A Git push does not deploy. Deployment needs an explicit POST to `/deploy`.
The prior handoff recommends normal deploys; forced deploys rebuild all layers.
`.dockerignore` controls the build context. Check a new COPY source before deploying.

## Current behaviour

- List names share one spelling, ignoring case.
- Five built-in profiles provide editable reminder defaults. An explicit item reminder takes precedence.
- Recurrence supports daily, weekly, monthly and yearly intervals, including repeats after completion.
- Completing a repeating item advances its times. Undo restores the prior occurrence.
- `start_at` marks the start. `due_at` marks the deadline. Start must not follow deadline.
- Duration accepts 1 to 10,080 minutes. Priority accepts 1 to 5; zero means unset.
- Reminder and recurrence anchors use the deadline, then the start.
- Calendar placement uses the start, then the deadline. Times use the device zone.
- Repeats move start, deadline and reminder by the same elapsed distance.
- Across daylight-saving changes, a start tied to a deadline can move one hour against the local clock.
- `GET /v1/items?from=&to=` accepts ranges up to 62 days.

Specs: [docs/m1-spec.md](docs/m1-spec.md), [docs/calendar-spec.md](docs/calendar-spec.md),
[docs/notifications-spec.md](docs/notifications-spec.md).

## Product and owner follow-ups

- Phone checks of lists, profiles, recurrence, follow-up questions and calendar behaviour.
- Phone check of the bottom bar, the header rule and the composer spacing.
- Trace the next real reminder with [docs/logging.md](docs/logging.md). Production shows one device and no active schedule.
- Pass the Git revision into the image build so `server configuration` shows it.
- Android emulator notification check; the prior handoff records no emulator container.
- Date-only handling: "Buy milk tomorrow" can omit the due time without a question.
- Multi-change completeness: a weaker model can omit a change without an error.
- The app uses Material icons. The guide specifies Carbon icons.
- Live `/branding/` content after sign-in needs a check. The prior unauthenticated request returned 401.
- `docs/evidence/m0.md` needs the reminder phone result recorded.

Deferred product work: list reset and clone, subtasks, labels, profile statuses and voice.

## Local verification

Run `toolchain/check` from the worktree root. Individual checks include:

- `toolchain/run go go test -race ./...`
- `toolchain/run go go vet ./...`
- `toolchain/run go gofmt -l .`
- `toolchain/run flutter flutter test`
- `toolchain/run flutter flutter analyze`
- `toolchain/run flutter dart format --set-exit-if-changed lib test`
- Build web, then run chat, reminder and calendar browser checks.

The chat, reminder, calendar and email browser scripts start their own isolated fixtures. Run them in sequence.

## Secrets and access

Secrets stay in TokenCrypt or process variables.
TokenCrypt grants last 30 minutes. The vault can require the owner to unlock it on the website.
Deployment uses `coolify_api_root_token`; Firebase uses `firebase_minder_service_account`.
