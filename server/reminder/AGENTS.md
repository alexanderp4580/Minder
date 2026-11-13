# Purpose

Reminder scheduling, actions and delivery orchestration.

# Ownership

Due schedule processing, late/nag entries, Done, acknowledgement, snoozes, device validation and worker retries.

# Local Contracts

Persist schedule, occurrence, chat and audit changes in one transaction.
Actions are idempotent: repeated actions return the recorded result. Reject stale generations.
After each firing the schedule advances to the item's next offset (`NextReminder`); nag applies only after the last. Use domain completion and snooze rules. Check deliveries remain current before sending.
Use a sender interface; FCM belongs in `push`. Worker cancellation completes before storage closes.

# Work Guidance

Write failing behaviour tests before fixes. Preserve current wire and database contracts.

# Verification

From the root: `toolchain/run go go test -race ./reminder`.
Run `toolchain/architecture-check` for package boundaries.

# Child DOX Index

None.
