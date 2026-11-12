# Handoff

State on 2026-11-12. Branch `m0-chat`, worktree `Minder-m0-chat`.
Keep this file current until the owner merges.

## Current source

- Chat, item and profile previews, Confirm, Discard, Undo and open-item queries.
- Lists, profiles, recurrence, reminders, nagging and Android FCM actions.
- Start, deadline, duration, priority and location fields.
- Read-only calendar with a month grid and day agenda.
- Theme spacing, standard visual density, shared Material styles and golden tests.
- Private `/branding/` guide and public OAuth logo.

The owner approved M0 and reminder delivery on a phone.
The owner requested completion of M1 and the calendar without questions.
Execution stays inline, without sub-agents. Phone approval still gates merging to `main`.

## Review and refactor

The current review isolates authentication, process settings, JSON decoding and Android channel messages.
It also fixes calendar request ordering and stale notification permission state.
Details and checks: [docs/evidence/review-refactor.md](docs/evidence/review-refactor.md).
Architecture: [docs/architecture.md](docs/architecture.md).

Production now requires explicit `CF_ISSUER`, `CF_AUDIENCE` and `OWNER_EMAIL` settings.
Set them in Coolify before deploying this build. The review did not inspect hosted settings.
`cmd/minder` contains no Cloudflare settings or owner identity.

## Deployment and APK status

The prior handoff records production commit `1c0efe9`, with calendar fields and `/branding/`.
This review did not check the live deployment or deploy changes.
The spacing changes and refactor remain local to `m0-chat`.

APK 0.3.3 was sent through Telegram. APK 0.4.0 is not built.
The owner stopped that build. Build it only when the owner asks.

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

## Checks still needed

- Phone checks of lists, profiles, recurrence, follow-up questions and calendar behaviour.
- Android emulator notification check; the prior handoff records no emulator container.
- Date-only handling: "Buy milk tomorrow" can omit the due time without a question.
- Multi-change completeness: a weaker model can omit a change without an error.
- The app uses Material icons. The guide specifies Carbon icons.
- Live `/branding/` content after sign-in needs a check. The prior unauthenticated request returned 401.
- `docs/evidence/m0.md` needs the reminder phone result recorded.

Deferred product work: list reset and clone, subtasks, labels, profile statuses and voice.

## Local verification

Run from the worktree root:

- `toolchain/run go go test -race ./...`
- `toolchain/run go go vet ./...`
- `toolchain/run go gofmt -l .`
- `toolchain/run flutter flutter test`
- `toolchain/run flutter flutter analyze`
- `toolchain/run flutter dart format --set-exit-if-changed lib test`
- Build web, then run chat, reminder and calendar browser checks.

`toolchain/local-check` needs `toolchain/model_fixture.py` on port 8181.
The reminder and calendar scripts start their own fixture. Run these checks in sequence.

## Secrets and access

Secrets stay in TokenCrypt or process variables.
TokenCrypt grants last 30 minutes. The vault can require the owner to unlock it on the website.
Deployment uses `coolify_api_root_token`; Firebase uses `firebase_minder_service_account`.
