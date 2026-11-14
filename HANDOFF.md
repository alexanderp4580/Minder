# Handoff

State on 2026-11-14. Production runs `dff6f38` from `guided-email-review` (Coolify deployment `7nxitpnns4vwk3yymn9w7yx9`, finished, running:healthy, schema 6). Read-only verification matched the running image tag to that full commit. Private whole-email and agent tracing is enabled until `2026-11-15T20:56:21Z` (15 November, 21:56 Amsterdam time). Runtime checks verified the startup flag, active capture deadline, 0700 directory permissions and no warnings or errors in the last 80 startup log lines. No production trace files existed at that check. No data reset, app version change or migration accompanied this deployment. The previous runtime is `b7eaf1e`, also schema 6. Earlier owner-requested resets and backup removal left no rollback data; a schema-5 binary refuses schema 6. The branch is pushed and unmerged; phone approval still gates merging.

## Required preparation tasks

The owner asked for a system-prompt nudge to recover the questionnaire task and useful timing choices. The prompt now treats required preparation and follow-up work as separate task candidates, including one questionnaire for several visits. It separates a deadline from the owner's choice of when to do the task or be reminded. It offers future choices before the earliest relevant appointment (for example one day, two days or a week before), respects earlier source deadlines, and updates only the answered task. It also states that each date boundary uses either a date or a timestamp, never both; a real-model run exposed a mixed deadline that prevented the reminder proposal from committing. No production keyword routing, timing parser, tool-loop or pipeline change was added. Source `dff6f38` is deployed and healthy as recorded above.

Acceptance: `toolchain/real-model-agent-check --preparation` runs the real production-default model against temporary local servers and SQLite. Its long synthetic email contains appointments on 2, 9 and 16 December 2026, stated lengths and a required questionnaire before the first visit. The baseline failed because the task had no timing question. After the prompt change, three independent runs each returned four items for forwarded and pasted email, with one-day, two-day and one-week options before the first appointment. Choosing two days before produced a proposed reminder on 30 November and left the other three proposals unchanged. Model calls/costs for these runs: 6/0.0142 USD, 7/0.0175 USD and 10/0.0243 USD. The original real-model regression check also exited 0 (11 calls, 0.0195 USD). `toolchain/check` exited 0 with 182 Flutter tests, Go race/vet, analysis/format and repository checks. Runtime model configuration has no MODEL override, matching the local default. Real-email and phone review of the prompt change remain pending.

## Temporary private diagnostics

The owner approved sensitive logs for testing and explicitly requested the whole email. Private capture records accepted original MIME byte for byte (including email headers and attachments), parsed documents, complete model request bodies, raw responses, review rejections and final agent outcomes. It captures the original even if the agent never reads it. HTTP authentication headers and process credentials are excluded. Normal logs no longer include provider error bodies, which may echo source text.

`AGENT_TRACE_DIR` and `AGENT_TRACE_UNTIL` enable capture with an explicit expiry. Production activation and runtime permissions are verified above; no production email had arrived in the trace directory at that check. Directory mode is 0700; files are 0600. Expiry stops new capture; existing files remain for inspection and explicit cleanup. Event formats and inspection: [logging](docs/logging.md#temporary-private-agent-traces). The tracing change itself altered no model instructions, application version or schema.

Verification on 2026-11-14: `toolchain/check` exited 0 (182 Flutter tests, Go race/vet, analysis/format and repository checks); the email browser flow exited 0 with tracing enabled. Trace inspection verified all six browser emails against their source SHA-256, 27 paired exchanges and private permissions. `toolchain/real-model-agent-check` exited 0: 12 model calls, 0.021 USD, six completed turns, original email hash verified, paired full requests/responses, one captured review rejection and no API key in the traces. The synthetic mail produced a dental visit and a questionnaire task, but the questionnaire had no timing question or options; this does not establish why the owner's email lost that task. Whole-email capture also passed a regression using an original larger than the model context budget. Expired/disabled capture, expiry during a call, malformed/provider replies and nonfatal write errors passed. Tests failed before implementation. Review ran inline under the owner's no-sub-agents preference. DOX contracts were updated; indexes stay unchanged because ownership boundaries did not change.

## One agent (deployed)

Spec: [one-agent design](docs/superpowers/specs/2026-11-14-one-agent-design.md). Plan: [plan](docs/superpowers/plans/2026-11-14-one-agent.md). Evidence: [one-agent evidence](docs/evidence/one-agent.md).

Chat, forwarded email and pasted email use one agent: one instruction set, one context builder and one tool loop (`find_sources`, `read_source`, `inspect_review`, `propose_review` and the chat tools). Owner messages, quick replies and source arrivals are durable agent turns, run one at a time; a turn commits atomically with its outcome. Preparation only parses and stores. Updates change one candidate by ID and revision; omitted candidates stay. The extraction path is removed. Long pastes (up to 400,000 bytes) reach the agent whole. No call cap: the request budget, turn timeout and monthly limit bound a loop.

Phone feedback on the deployed build (owner report, unverified by me; the production data was reset afterwards, so there is no stored evidence to inspect):
- The owner clarified that the agent handled the first appointment's one-hour length correctly. Duration is not an outstanding defect.
- The review had three items, not four. The questionnaire task was missing. The earlier build asked about it, with options for when to do it so it is done on time; the owner wants that item back, with timing options.
- Open question: why the agent stopped proposing the questionnaire task with timing options after the one-agent change. Reproduce with the real-model check and a synthetic email of the same shape before changing the prompt. Do not add keyword rules or parsers.

The owner approved temporary sensitive logging and then requested a system-prompt nudge for questionnaire reminders, with local synthetic verification. Current implementation and evidence are recorded in Required preparation tasks and Temporary private diagnostics above. The shorter old extraction prompt versus the shared prompt remains an unverified explanation for the missing item in the owner's earlier email. Private whole-email capture is deployed and enabled; inspect the next real email's trace rather than guessing about what it contained. No keyword rules, parsers or pipeline changes. Settings, event contents and cleanup: [logging](docs/logging.md#temporary-private-agent-traces).

Judgment calls: the code-side duplicate-booking check is gone (the prompt asks the model, and the real model did flag a duplicate); a bad review goes back to the model as a tool result instead of a correction call; the first-publication push is dropped if the review is answered before it is delivered, as before.

Local results: `toolchain/check` exit 0; five browser scenarios pass; a real-model run on synthetic mail (11 calls, 0.019 USD) kept the 13:00 start, set no end or length, asked how long, answered bring/questionnaire questions from the body and attachment, updated only the answered candidate, and gave the same review for pasted text. APK 0.11.0+33: `.local/artifacts/Minder-0.11.0-arm64.apk`, 19,031,442 bytes, SHA-256 `68360a5f84ffd13ff8ac6c75ebf6c4bef484089b1336a7319211fedf376673a7`, package `io.liveleds.minder`, version code 2033, same signer as 0.10.3. Download: http://192.168.8.226:8788/Minder-0.11.0-arm64.apk (full download hash verified). The APK matches the deployed server. Phone rendering and notifications are unreviewed.

## Failure reporting (0.10.3+32, deployed)

The earlier 0.10.3 phone review had four items: three appointments and a questionnaire task. The first was confirmed. A one-hour end time was initially treated as unsupported; the owner later clarified that the first appointment's one-hour length is correct. Duration is not an outstanding defect. The owner's targeted one-hour answer for item 2 reached the model. Its follow-up returned two candidates but failed the check requiring the answered item's original ID. The server stored status failed; it did not create the second preview. Omission versus renaming could not be distinguished because that model reply was not stored.

The failure was only on the earlier email card, leaving Questions answered as the newest chat message. Firebase accepted the failure push (HTTP 200), but Android always displayed generic review copy and ignored the supplied title/body. The patch appends one source-linked failure report per revision, restores a missing report for a stored failed review on startup without re-extracting or re-pushing, records a specific review_item_missing reason, and uses the supplied Android notification text. Retry remains on the email card. The existing confirmed item is unchanged.

Four new server regressions failed before the fix and passed after it. `toolchain/check` exited 0 (169 Flutter tests, Go race/vet, analysis/format and repository checks), release web/APK builds passed, and the email browser scenario passed with a deliberately rejected four-item follow-up and a visible final failure report. All five browser scenarios exited 0. APK signature/version and a full LAN download were verified. Deployment finished at b260753 and reports running:healthy. A read-only live check found one new final chat report for revision 2; all four candidates, the first confirmed item, three model calls and both sent push rows are unchanged. Startup restored exactly one report, with no warnings or errors. No emulator is available on this machine; Android rendering still needs phone verification. The reporting patch did not change proposal generation or repair the candidate-ID failure; the later one-agent implementation replaced that extraction path.

APK 0.10.3: .local/artifacts/Minder-0.10.3-arm64.apk, 19,031,438 bytes, SHA-256 19772ce14623ad7b4269e9edf943eda682724cfbb55b0a47f0a2c9b492dc1462, package io.liveleds.minder, Android version code 2032, same signer as 0.10.2. Download: http://192.168.8.226:8788/Minder-0.10.3-arm64.apk. The LAN server serves only APKs; keep it running. Future split arm64 updates must exceed 2032. No Telegram retry or database reset accompanied this patch. Branch remains unmerged for phone approval.

## Phone feedback patch (0.10.2+31)

The first reported Confirm saved its item. Read-only structured state showed three confirmed appointments and one confirmed questionnaire task, which explains four items. A client regression reproduced the stale Confirm button: the HTTP response skipped the earlier preview update by advancing the polling cursor. The patch separates the ordered-feed watermark from the latest visible cursor and covers response-first answers and in-flight polls too.

Working now stays above the composer during normal chat waiting and email received/processing states. It has a live accessibility label and stays visible while reading a long review. Six focused regressions passed. `toolchain/check` exited 0 with 169 Flutter tests, Go race/vet, analysis/format and repository checks. Release web and APK builds passed; all five browser scenarios exited 0. The email scenario blocks the event stream for Confirm and holds the model answer to verify Working. The web update is deployed and healthy. The server source is unchanged. The owner requested another backed-up reset before retrying 0.10.2; keep the feature branch unmerged until phone approval.

APK `.local/artifacts/Minder-0.10.2-arm64.apk`: 19,031,438 bytes, SHA-256 `92a2360ce3276b904fe068dbfc334ce4e25cdc6704071fb2d1bfe4fdc368c738`, same signer as 0.10.1, package `io.liveleds.minder`, version code 2031. The owner confirmed that the initial Telegram upload did not arrive and requested a test message. All three APK attempts (initial local upload, local resend, server-side transfer) returned HTTP 504. Two test-message attempts (POST and GET) returned HTTP 502. getMe succeeds, but no file or message delivery is confirmed. Avoid further uploads until the send endpoint works or the owner directs another attempt. The owner switched delivery to LAN. A detached HTTP server serves only the APK from `.local/lan-apk-download/` at `http://192.168.8.226:8788/Minder-0.10.2-arm64.apk`; PID file `.local/lan-apk-server.pid`, log `.local/lan-apk-server.log`. A full local download returned HTTP 200 with matching size and SHA-256. Phone download is not yet confirmed. Keep it running for the owner. Future split arm64 updates must exceed 2031.

## Fresh test data

2026-11-14, at the owner's request: cleared production items, chat/events, pending/audit records, reminders, captures, receipts, provenance and email files with no backup. Preserved 90 usage rows and one push registration. Integrity and foreign-key checks passed; the container restarted healthy and restores the five default profiles. The owner also asked to remove every backup: `/var/backups/minder` is empty and the Coolify `minder-email` backup is gone. No backups of Minder data exist now. The owner asked for another reset after the one-agent deployment; it also cleared the new `agent_turn` and `source` tables (usage 93 rows and the push registration kept, no backup, integrity ok). Close and reopen the phone app before testing.

## Guided email review (branch `guided-email-review`)

Revision 2 is complete at source commit `fbbe18d`. All five browser scenarios passed. APK 0.10.1 reached Telegram once (message 2029). Base `main` remains `679441b`. The branch is pushed and deployed for phone testing, and remains unmerged. Work stays inline in `Minder-m0-chat`; phone review and owner approval gate merging.

Spec: [guided email review design](docs/superpowers/specs/2026-11-14-guided-email-review-design.md). Plan: [guided email review plan](docs/superpowers/plans/2026-11-14-guided-email-review.md). Verification: [guided email review evidence](docs/evidence/guided-email-review.md).

- One worker extraction attempt per revision; visible failure and one push, owner-only Retry, no call cap. Output ceiling 100,000 tokens and email context up to 400,000 bytes. Evidence checks keep valid candidates and count rejected ones. One evidence-correction call may improve an extraction; a provider failure on that correction keeps the verified first result while the request context remains active.
- Chat reviews one appointment at a time. Confirm and Discard advance the queue. `Appointment i of N` and `Done` remain mechanical. Every answer, including quick replies, goes through the model. The model decides whether an item has an address; Maps appears for any supplied location.
- Items carry `details` as ordered label/value pairs, with replace, clear and Undo. Previews, email details and calendar details display them and open web links. Meeting links belong in Details, not Website. No schema migration: JSON rows store the fields.
- The model writes `summary` and `notification`. The server posts the opening unchanged on initial extraction, stores the notification for the first push and keeps the unverified count in its own note. Missing summary posts no opening; missing notification uses `An email needs your review.`. Failure push: `Minder could not read an email.`. Answers, Confirm and Discard queue no push.
- Fresh `toolchain/check` exited 0: 163 Flutter tests, Go race/vet, analysis/formatting and repository checks. A real-model synthetic three-appointment email returned a Teams meeting with its join link in Details, no address and no Website value, plus an opening and notification. All five final browser scenarios exited 0: chat, reminder, calendar, place lookup and email.
- Earlier app version: 0.10.1+30. APK `.local/artifacts/Minder-0.10.1-arm64.apk` is 19,031,438 bytes, SHA-256 `e7225a914cc245a8c3db1016ddf515402c762f0bb4295dd57fb23c4f3bb55eeb`; its signer matches 0.10.0. The split arm64 APK carries Android version code **2030** (pubspec 30 plus the ABI offset); future APK updates must exceed 2030. Use the same split build command with the next pubspec version. Telegram confirmed one upload (message 2029). The previous APK 0.10.0 (message 2028) lacks Revision 2; do not ship it.

Owner preferences: very short replies; no sub-agents; the model decides behavior and wording; no unrequested rules, word lists, answer parsers or caps. Merge requires approval. The owner authorized this deployment for phone testing after APK delivery. Current 0.10.3 delivery status is recorded in the failure reporting section above.

Unknown: the precise rejection reason for the owner's earlier multi-appointment email, and how the phone renders push title/body. No physical phone review of this revision yet.

App 0.9.3+28 carries the near-white M logo and the place lookup. The APK (`.local/artifacts/Minder-0.9.3-arm64.apk`, ignored by Git, 19,283,214 bytes, SHA-256 `1f4c7f4b09b31e5692b9bfaa8cafc6a4248eae6bc2ad75e58e4d38846403bfa7`, signer matches earlier APKs) reached Telegram once (message 2027). Phone approval of the logo and of the lookups is still pending.

## Current work: approved controls and booking fields

The owner approved the brand guide and clarified that phone, email, website and booking reference belong in separate fields like Address, not Notes. App 0.9.1+26 introduced these fields.

The client uses neutral square text actions, chevrons, the approved address block and separate contact/reference rows. New proposals and stored items carry optional `phone`, `email`, `website` and `booking_reference` strings. Notes retain remaining instructions. Clearly labelled older note metadata is separated for display; server edits promote it before clear/Undo so cleared data cannot reappear. Conflicts and conditional instructions remain visible. Bare calendar UID/sequence Ref lines stay in Notes.

HTML conversion preserves safe contact/link targets and cell separation before extraction. No schema migration is needed: item JSON already stores optional fields. The owner authorized delivery and deployment. Commit `d04c3a8` deployed successfully through Coolify (`ory0emc19xfx2eroyyrh48k6`); the application reports running:healthy. The previous verified runtime is `fc21e5e`. Phone approval of the app builds is still pending.

Local checks passed: 147 Flutter tests, Go race/vet, analysis/format, all four browser scenarios, and a real-model synthetic booking confirmed into temporary SQLite with separate fields. APK 0.9.0+25 built and its signature matches 0.8.0. Telegram delivery succeeded once on the authorized retry (message 2000). Source: server `03c51fd`, client/guide `04a3530`.

The later phone screenshot exposed raw evidence in the email card and missing duration suggestions. The 0.9.1 patch replaces quotes/internal IDs with shared structured fields, uses an English heading and extracted descriptions, and puts 30-minute/alternative choices inside expanded details. Source evidence stays literal for validation and original attachments remain available. The latest full check passed (150 Flutter tests). Browser flows passed, including inline duration submission and persisted fields. An accessible booking-reference defect found by the browser check was fixed before shipping. Source is `d04c3a8`. APK 0.9.1+26 reached Telegram once (message 2001, 19,289,894 bytes), with the matching signing certificate. The server deployment finished on that exact source commit. Main remains unchanged for phone review.

Verification and APK status: [calendar controls evidence](docs/evidence/calendar-controls.md).


## Calendar details 0.8.0

The owner approved calendar details and links with “Fix all”, asked to continue the current work, and requested a clearer Confirmed footer. That 0.8.0 slice needed no server change.

- `d0f0f68` fixes the stale connection message: dropped streams feed the delayed offline icon; a successful sync clears background error copy. Failed sends retain composer text.
- `4f45e7e`: calendar cards expand on a card tap or Details. Available rows retain kind, notes, reminders, source wall clocks and timezones. Starts/Ends become Departs/Arrives only across different zones. Address opens Google Maps; note links open the dialer, email app or browser. The launcher permits HTTPS, tel and mailto, and reports failed launches. Notes retain the 600-character cap.
- Confirmed preview cards have a divider, more space, a check icon and semibold text.
- APK 0.8.0+24 reached Telegram once (message 1998). Full repository checks (135 Flutter tests), release web/APK builds, matching signing certificate and all three browser scenarios passed. Phone review is pending. Details: [calendar details evidence](docs/evidence/calendar-details.md).


## Controls and address brand review

The owner requested a brand guide proposal before app changes, and clarified that the address section inside the card looks odd. `brand/controls-review.html` and its three-page PDF propose neutral disclosure labels with chevrons, aligned address blocks, white underlined links, destination hints and square interaction states. Desktop and mobile PNGs show the draft.

Telegram confirmed one PDF upload (message 1999, 42,266 bytes). Local checks passed at 1280, 390 and 320 px, plus 320 px at 1.4× text; fonts, disclosure toggling and the Maps target passed. The three A4 PDF pages fit without clipping.

The owner approved the guide on 2026-11-13 and clarified that contact/booking values need separate fields, like Address. Canonical guide and shared styles now include the approved treatment. Current implementation and verification: [calendar controls](docs/evidence/calendar-controls.md).


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
Earlier read-back matched the live Cloudflare app and owner policy; deployment owns activation.
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

Merged into `main` at `ab94726` and included in current production. App 0.7.0+23 (the server also gained the finished-items lookup, which needs no app change).
Chat can cancel, delete and look up finished items (a second model call, only on request). The calendar has a Delete control. A delete removes the item and its associated data. A failed send returns its text to the composer and leaves no Retry block.
APK 0.7.0 reached Telegram once (message 1995). Evidence and limits: [docs/evidence/delete-items.md](docs/evidence/delete-items.md). The delete server changes are deployed. Phone approval is pending.

## Place lookup

In `main` and deployed at `adb1bdf` (Coolify deployment `gn89izeozdgbno8bmucrxgxl`, running:healthy, schema 5, no migration, startup log `serper_key_set=true`, no warnings or errors in the first 60 log lines). App 0.9.2+27; an APK needs a code above the installed 26 to update.
When a request names a venue without an address, chat shows an offer card (Skip, Look it up). Look it up runs one Serper places search for the offered query and the next preview carries the address in location and the website in the website field. Skip and any other text run no search. The server needs `SERPER_API_KEY` (TokenCrypt secret `serper_api_key`) as a Coolify process variable; without it chat is unchanged.
Design: [docs/superpowers/specs/2026-11-13-place-lookup-design.md](docs/superpowers/specs/2026-11-13-place-lookup-design.md). Plan and evidence: [docs/superpowers/plans/2026-11-13-place-lookup.md](docs/superpowers/plans/2026-11-13-place-lookup.md), [docs/evidence/place-lookup.md](docs/evidence/place-lookup.md). APK 0.9.2 (`.local/artifacts/Minder-0.9.2-arm64.apk`, ignored by Git) reached Telegram once (message 2009). Nobody has tried a lookup against production or on a phone yet. Phone approval is pending.

Approved lookups: the model offers one search (`offer_lookup` with kind maps, places or search, and a query); the owner's tap runs exactly that search; the model reads the hits and proposes, asks, or offers another. No budget, no loop. Spec: [approved lookups design](docs/superpowers/specs/2026-11-14-approved-lookups-design.md); evidence: [docs/evidence/place-lookup.md](docs/evidence/place-lookup.md). In `main` and deployed at `679441b`; no app change. Not tried on a phone yet.

## App icon and faster highlights

In `main` and deployed (server side). App 0.6.2+22.
The launcher icon is the Michroma M, near-white with a glow at two corners and a lit rim, over the Orbital background, no red. Orbital highlights loop in 68.571 s with ease in and out (mean speed 2.625 times the original 180 s linear loop); their brightness is unchanged.
APK 0.6.1 reached Telegram once (message 1993; 102.857 s linear loop). APK 0.6.2 reached Telegram once (message 1994). Telegram credentials come from TokenCrypt (`telegram_astrabot_token`, `telegram_astrabot_chatid`).
Evidence and limits: [docs/evidence/app-icon.md](docs/evidence/app-icon.md). Phone approval is pending.

## Open work

### Notification actions and email capture (in `main`, deployed; phone approval pending)

- `be4aeff`: the reminder notification has no content intent. Tapping its body does nothing.
- `6c115a5`: Done and Snooze run in the background. `ReminderActionReceiver` starts a headless Flutter engine at the Dart entry point `notificationAction` and calls the reminder action with the app login. Open still opens the app.
- App version 0.4.8+17. APK 0.4.8 reached Telegram once (message 1975). The server is unchanged.
- Verified on the emulator with a direct broadcast, not a rendered button. Evidence and limits: [docs/evidence/notification-actions.md](docs/evidence/notification-actions.md).
- Not checked: a physical phone, Android 12 and later background limits, battery restrictions, and the failure message with notifications on.
- The source is already in main and production; phone verification of the buttons remains pending.

### Keyboard auto-scroll fix

`ChatScreen` now follows list metrics after layout while pinned to the bottom.
Keyboard opening and stepped dismissal, the returning bottom bar and composer resizing keep the latest content visible.
User scrolling away pauses viewport following. Returning to the bottom or a new message cursor resumes it.

The regression test reproduced a 244 px gap after keyboard opening before the fix.
It covers sending during keyboard dismissal, preserving the reading position, resuming at the bottom and receiving a reply.
Verification: `toolchain/check` exited 0 on 2026-11-12: 56 Flutter tests, analyze, format, Go race tests/vet, toolchain/hook tests, architecture, DOX and orbital export checks.
Physical phone verification remains pending. Later APKs and deployments include this change.
The email extractor allows one evidence correction per batch (`96d1778`); see the email capture section.

Leftovers on this machine:

- A local `minder` server on `127.0.0.1:18190` and `toolchain/model_fixture.py` on port 8181 may still run. Stop them by process name, not with `pkill -f`, which also kills the calling shell.
- The emulator `emulator-5584` has the debug x86_64 build 0.4.8 installed over 0.1.1, and a port forward for 18190. Remove it with `adb reverse --remove-all`. `adb unroot` was already run.

Send each APK once. A 504 from Telegram can mean the file arrived.

## Email capture implementation

Earlier app version: 0.6.0+20 (email review refinement: [design](docs/superpowers/specs/2026-11-13-email-review-refinement-design.md)). The initial live email release ran `fb99b50`: Worker `minder-email-ingress`, routing for the `minder` subdomain and an Access service-token policy. A real forward from an allowed Gmail address reached the server and produced one candidate on 2026-11-12 (Worker log: all stages, status 201). APK 0.5.0+18 reached Telegram (message 1990). App 0.5.1+19 adds the Do nothing button and clearer zero-item wording. Phone review, attachment download and Confirm remain unverified.

App 0.6.0+20 changes the flow: the email card lists what was found, questions arrive as their own message with agent-chosen quick replies, and the final card shows the title once, Starts/Ends with timezone, `45 minutes`, Address, bullet notes, Type Appointment and reminder chips (10 minutes to 1 week; the agent preselects). One item can hold several reminders: `reminder_offsets_minutes` on the item, one schedule row that advances to the next unfired offset. A header icon shows the connection only after 2 seconds offline.
Known gaps: the app does not re-register a stale push token on resume (tap Enable notifications); normal chat still asks for a location by fixed rule (`server/assistant/prompt.go`).
The owner approved the specification and inline implementation. The Worker, Go ingestion/parser/review pipeline, Flutter review flow and deployment runbook are present.

Minder retains originals and attachments, asks targeted follow-up questions and requires Confirm for each item change. Travel items preserve both time zones and hotel checkout dates. Sources remain accessible from calendar items.
The Worker delivers directly to Minder without a Cloudflare content buffer. Both ends require private `EMAIL_ALLOWED_SENDERS` configuration and aligned sender authentication; source values stay outside Git.

Verification and limits: [email capture evidence](docs/evidence/email-capture.md).
Activation, private environment parameters, persistence and rollback: [operations](docs/email-capture-operations.md).
Live ingestion was verified as described above. Phone approval and native attachment opening remain unverified.
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

Current runtime and APK status are recorded in Current work above and the linked evidence. Coolify builds `main` at a pinned `git_commit_sha`; it does not follow new commits by itself. The `adb1bdf` deployment ran schema version 5 with email capture, Firebase push and the Serper key configured; it applied no migration.
Notifications did not arrive on 2026-11-12. The old build logged nothing that explains it. Watch the next reminder with the new logs.

APK 0.4.6 carries the bar, the header rule, the composer spacing and the dimmer lines. Telegram confirmed one upload.
The build uses the cached debug signing key, so it installs over earlier builds. Phone approval of the look remains pending.
Earlier APK evidence: [docs/evidence/orbital-artwork.md](docs/evidence/orbital-artwork.md).

The Coolify app is `ciw3jk5gkgiql8kfdomeyys2`.
A Git push does not deploy. To deploy a commit: PATCH the application with `{"git_branch":"main","git_commit_sha":"<sha>"}` at `/api/v1/applications/<uuid>`, then POST `/api/v1/deploy?uuid=<uuid>&force=false`. Coolify rejects a GET. A deploy without the PATCH redeploys the old pinned commit and rebuilds nothing. Rollback is the same two calls with the branch and commit captured before deployment.
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
- Trace the next real reminder with [docs/logging.md](docs/logging.md). The last recorded reminder heartbeat was from the earlier deployment; inspect current runtime counts when tracing.
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
