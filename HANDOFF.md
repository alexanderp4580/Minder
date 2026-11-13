# Handoff

State on 2026-11-13. Production runs commit `ab94726`, the head of `main` and of `notification-body-tap`.
`main` holds `m0-chat`, `notification-body-tap`, `app-icon-orbital` and `delete-items`. The owner asked for the merge and the deployment on 2026-11-13. `origin/main` and `origin/notification-body-tap` are both at `ab94726`. The worktree `Minder-m0-chat` is on `fix-connection-message`.

## Current client work

The owner approved calendar details and links with “Fix all”, asked to continue the current work, and requested a clearer Confirmed footer. Work runs inline on `fix-connection-message`; keep this branch for phone review. No server change or deployment is needed.

- `d0f0f68` fixes the stale connection message: dropped streams feed the delayed offline icon; a successful sync clears background error copy. Failed sends retain composer text.
- `4f45e7e`: calendar cards expand on a card tap or Details. Available rows retain kind, notes, reminders, source wall clocks and timezones. Starts/Ends become Departs/Arrives only across different zones. Address opens Google Maps; note links open the dialer, email app or browser. The launcher permits HTTPS, tel and mailto, and reports failed launches. Notes retain the 600-character cap.
- Confirmed preview cards have a divider, more space, a check icon and semibold text.
- APK 0.8.0+24 reached Telegram once (message 1998). Full repository checks (135 Flutter tests), release web/APK builds, matching signing certificate and all three browser scenarios passed. Phone review is pending. Details: [calendar details evidence](docs/evidence/calendar-details.md).


## Controls and address brand review

The owner requested a brand guide proposal before app changes, and clarified that the address section inside the card looks odd. `brand/controls-review.html` and its three-page PDF propose neutral disclosure labels with chevrons, aligned address blocks, white underlined links, destination hints and square interaction states. Desktop and mobile PNGs show the draft.

Telegram confirmed one PDF upload (message 1999, 42,266 bytes). Local checks passed at 1280, 390 and 320 px, plus 320 px at 1.4× text; fonts, disclosure toggling and the Maps target passed. The three A4 PDF pages fit without clipping.

Status: awaiting owner approval. The app and canonical `brand/index.html` retain the current implementation. After approval, promote the agreed rules into the canonical guide and shared styles, then apply them to the calendar.


## Current source

- Chat, item and profile previews, Confirm, Discard, Undo and open-item queries.
- Lists, profiles, recurrence, reminders, nagging and Android FCM actions.
- Start, deadline, duration, priority and location fields.
- Calendar with a month grid, spaced week rows, a scrolling day agenda and a Delete control on each card (it only deletes; creating and changing items stay in chat).
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
Atmosphere, edge gradients and soft glow stay still. Short highlights travel along cached paths over about 69 seconds (68.571 s per loop), easing in and out.
Static artwork and foreground are isolated from highlight repaints.
Reduced motion keeps the scene still. Hidden routes and backgrounded apps stop ticking.
Tests verify movement, lifecycle pauses, distinct scenes and stable foreground builds.
Brand guide, foundation, mockups and goldens follow the approved direction.
Phone frame timing and battery use remain unmeasured.
The owner approved reconstructing the artwork and moving highlights along its fixed paths.
The SVGs own geometry and lighting; toolchain/export-orbits.py keeps Flutter and HTML exports aligned.
The calendar spacing and opaque cards remain unchanged.

## Delete and cancel items

In `main` and deployed at `ab94726`. App 0.7.0+23 (the server also gained the finished-items lookup, which needs no app change).
Chat can cancel, delete and look up finished items (a second model call, only on request). The calendar has a Delete control. A delete removes the item and its associated data. A failed send returns its text to the composer and leaves no Retry block.
APK 0.7.0 reached Telegram once (message 1995). Evidence and limits: [docs/evidence/delete-items.md](docs/evidence/delete-items.md). The server needs a deploy before delete works on a phone. Phone approval is pending.

## App icon and faster highlights

In `main` and deployed (server side). App 0.6.2+22.
The launcher icon is the Michroma M with a lit rim over the Orbital background, no red. Orbital highlights loop in 68.571 s with ease in and out (mean speed 2.625 times the original 180 s linear loop); their brightness is unchanged.
APK 0.6.1 reached Telegram once (message 1993; 102.857 s linear loop). APK 0.6.2 reached Telegram once (message 1994). Telegram credentials come from TokenCrypt (`telegram_astrabot_token`, `telegram_astrabot_chatid`).
Evidence and limits: [docs/evidence/app-icon.md](docs/evidence/app-icon.md). Phone approval is pending.

## Open work

### Notification actions and email capture (in `main`, deployed; phone approval pending)

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

Local app version: 0.6.0+20 on `notification-body-tap` (email review refinement: [design](docs/superpowers/specs/2026-11-13-email-review-refinement-design.md)). Production runs `fb99b50` with email capture live: Worker `minder-email-ingress`, routing for the `minder` subdomain and an Access service-token policy. A real forward from an allowed Gmail address reached the server and produced one candidate on 2026-11-12 (Worker log: all stages, status 201). APK 0.5.0+18 reached Telegram (message 1990). App 0.5.1+19 adds the Do nothing button and clearer zero-item wording. Phone review, attachment download and Confirm remain unverified.

App 0.6.0+20 changes the flow: the email card lists what was found, questions arrive as their own message with agent-chosen quick replies, and the final card shows the title once, Starts/Ends with timezone, `45 minutes`, Address, bullet notes, Type Appointment and reminder chips (10 minutes to 1 week; the agent preselects). One item can hold several reminders: `reminder_offsets_minutes` on the item, one schedule row that advances to the next unfired offset. A header icon shows the connection only after 2 seconds offline.
Known gaps: the app does not re-register a stale push token on resume (tap Enable notifications); normal chat still asks for a location by fixed rule (`server/assistant/prompt.go`).
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

Production runs `ab94726`, deployed on 2026-11-13 as Coolify deployment `bi6zx3vbht0tv4fvwsyxf3o2`. The app is `running:healthy`. Startup showed schema version 5, no migrations, email capture and Firebase push configured, and the reminder worker running with 9 items and 1 device.
The Coolify application builds the branch `notification-body-tap` at a pinned commit (`git_commit_sha`). It does not follow `main`.
The first production heartbeat reported one registered device, five items and no active schedule.
Notifications did not arrive on 2026-11-12. The old build logged nothing that explains it. Watch the next reminder with the new logs.

APK 0.4.6 carries the bar, the header rule, the composer spacing and the dimmer lines. Telegram confirmed one upload.
The build uses the cached debug signing key, so it installs over earlier builds. Phone approval of the look remains pending.
Earlier APK evidence: [docs/evidence/orbital-artwork.md](docs/evidence/orbital-artwork.md).

The Coolify app is `ciw3jk5gkgiql8kfdomeyys2`.
A Git push does not deploy. To deploy a commit: PATCH the application with `{"git_commit_sha": "<sha>"}` at `/api/v1/applications/<uuid>`, then POST `/api/v1/deploy?uuid=<uuid>&force=false`. Coolify rejects a GET. A deploy without the PATCH redeploys the old pinned commit and rebuilds nothing. Rollback is the same two calls with the previous commit (`0046c9a` before this release).
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
