# Handoff

State on 2026-11-12. Branch `m0-chat`, worktree `Minder-m0-chat`. This file is temporary. Update it at each step. Delete it when the owner merges.

## Done

- M0 chat: server, Flutter app, Cloudflare Access, Coolify deploy. The owner approved it on a phone.
- Reminders and push (`cce3855`): scheduling with catch-up, nagging, FCM, notification actions. The owner confirmed that it works on a phone.
- The app uses repository interfaces and use cases. Only `data` knows API routes.
- Production runs `124494d`. Firebase settings are runtime variables on Coolify.

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
- Open: "Buy milk tomorrow" gets no due time. The model writes the date into the notes.
- Open: a weaker model can drop a change from a multi-change message without an error. Free OpenRouter models work for plumbing tests. Rate limits apply.

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
