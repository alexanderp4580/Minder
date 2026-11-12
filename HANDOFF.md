# Handoff

State on 2026-11-12. Branch `m0-chat`, worktree `Minder-m0-chat`. This file is temporary. Update it at each step. Delete it when the owner merges.

## Done

- M0 chat: server, Flutter app, Cloudflare Access, Coolify deploy. The owner approved it on a phone.
- Reminders and push (`cce3855`): scheduling with catch-up, nagging, FCM, notification actions. The owner confirmed that it works on a phone.
- The app uses repository interfaces and use cases. Only `data` knows API routes.
- Production runs `1c0efe9`. Firebase settings are runtime variables on Coolify.

## In progress: M1 remainder (lists, profiles, recurrence)

Owner order: finish M1 without questions.

Decisions:
- A list is a name on an item. The server keeps one spelling for each name, ignoring case.
- A profile has a name, minutes before the due time, and a nag interval. Five built-ins are seeded: Daily task, Chore, Birthday, Day event, Trip. The owner can edit them and add more.
- An item with a profile and a due time gets its reminder from the profile. A reminder on the item wins. An item with neither stays silent.
- Recurrence: daily, weekly, monthly, yearly, with an interval. Fixed repeats keep the schedule. "After completion" counts from the day you finish. Completing a recurring item moves its due time forward. Undo restores it.
- Deferred: reset and clone of lists, subtasks, labels, priority, profile statuses.

Steps:
- [x] Domain: recurrence, profile, validation, next due time (tests pass).
- [x] Store: profile table, seeding, profile edits reschedule reminders, list names (tests pass).
- [x] Command layer: list, profile and recurrence on items; profile preview, confirm, undo (tests pass).
- [x] Assistant: new fields, `propose_profile`, lists and profiles in the model context, list filter (tests pass; one real-model run).
- [x] App: labels for the new preview fields (widget test passes).
- [x] Docs: `server/AGENTS.md`, `docs/master-plan.md`, `docs/m1-spec.md`, `docs/evidence/m1.md`.
- [x] Final checks.
- [x] Commit (`4f8515b`), push, APK 0.3.0 built (arm64, SHA-256 starts `94d444a3`).
- [x] Deployed `e2ab375` to production without `force`: 28 seconds, `running:healthy`. A forced deploy takes about 5.5 minutes because it rebuilds every layer. Use `force` only when a build input outside the Dockerfile layers changes.
- [x] APK 0.3.0 sent to Telegram. APK 0.3.1 hides the notification button and text once notifications are on.
- [x] Location field and follow-up questions (tests pass, real-model run in `docs/evidence/m1.md`).
- [x] Several changes in one message, cost on each reply (tests pass, Haiku run in `docs/evidence/m1.md`).
- [x] Deployed `124494d` (76 seconds, healthy). APK 0.3.3 sent to Telegram.
- [ ] Owner phone check of lists, profiles, recurrence and follow-up questions.
- Open: date-only requests behave inconsistently. "Buy milk tomorrow" gets no due time and no question. "Call mum on Sunday" gets a question about the time.
- Open: a weaker model can drop a change from a multi-change message without an error. Free OpenRouter models work for plumbing tests. Rate limits apply.

## In progress: start date, duration, priority and the calendar page

Owner order: add the three fields and a calendar page.

Decisions:
- `start_at` is when an item begins. `due_at` is the deadline. An appointment has a start and a duration. A task has a deadline.
- `duration_minutes`: 1 to 10,080. `priority`: 1 (highest) to 5. Zero means not set.
- The start must not be after the deadline.
- A profile reminder counts from the deadline, or from the start when there is no deadline.
- A repeating item needs a deadline or a start. Completing it moves start, deadline and reminder by the same distance. Across a daylight-saving change, the start of an item that also has a deadline can move one hour against the local clock.
- Calendar page: month grid and a day agenda, read-only, opened from the chat header. Items sit on their start day, else their deadline day. Times show in the device zone.
- `GET /v1/items?from=&to=` returns the items in a range.

Steps:
- [x] Domain: fields, validation, anchor rules, recurrence shift (tests pass).
- [x] Command and assistant: proposals, previews, prompt (tests pass).
- [x] API: `GET /v1/items` (tests pass).
- [x] App: item model, repository, use case, calendar controller, calendar screen, navigation, tests (19 Flutter tests pass; web check passes).
- [x] Docs: `docs/calendar-spec.md`, `docs/screens.md`, `docs/m1-spec.md`, AGENTS files, evidence.
- [x] Branding hosted at `/branding/` behind sign-in (tests pass). It is private. A public page needs a Cloudflare Access bypass for that path. The owner decides.
- Docker context: `.dockerignore` lists the folders the image build may read. A new folder needs an entry there. The first branding deploy failed for this reason. Test a new COPY with `podman build` on a small Dockerfile before a deploy.
- [x] Deployed `1c0efe9` (172 seconds; unhealthy for the first seconds, then `running:healthy`). Production has the calendar, the new fields and `/branding/`. The public address returns 401 without sign-in. The content at `/branding/` is not checked through the live host.
- [x] UI spacing pass (owner: the app felt tight around headers and did not match the guide). Header without divider, time next to the sender, 44 px buttons, 56 px composer, shared button components, `ui_spec_test.dart`, mockup and `foundation.md` updated. Not deployed. Not on a phone yet.
- Owner fixes after the spacing pass: the primary button is last in its row; the text field is 44 px and shares its centre line with Send; calendar day cells are 56 px. The field was short in desktop browsers only, because they default to compact `VisualDensity`. The theme now pins standard density. Button, icon button and field styles live in `Tokens.theme`; screens use plain Material widgets. The Snooze menu is a `MenuAnchor`.
- Known gap: the guide uses Carbon icons and the app uses Material icons.
- [ ] APK 0.4.0 (calendar and spacing) is not built. The owner stopped the build. Build it only when the owner asks.
- [ ] Phone check of the calendar.

## Checks

Run from the worktree root:
- `toolchain/run go go test -race ./...` and `toolchain/run go go vet ./...`
- `toolchain/run flutter flutter test` and `toolchain/run flutter flutter analyze`
- `toolchain/reminder-local-check` after a web build.

Go race tests, Go vet, Flutter tests and Flutter analyze pass. Both web scenarios pass (`local-check` needs `toolchain/model_fixture.py` running).

## Secrets and access

- TokenCrypt grants last 30 minutes and the vault locks again. The owner unlocks it in the website.
- `firebase_minder_service_account` and `coolify_api_root_token` are the secrets this work uses.
- The Coolify app is `ciw3jk5gkgiql8kfdomeyys2`. A push to GitHub does not deploy. Deploy needs an explicit POST to `/deploy`.

## Open

- Android emulator check of notifications: no emulator container exists.
- `docs/evidence/m0.md` does not yet list the reminder phone result.
