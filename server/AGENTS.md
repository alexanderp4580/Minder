# Server

Go HTTP service for M0. `domain` owns values. `store` alone writes SQL.
`command` owns item changes, previews, confirmation and undo. `assistant` proposes changes only.
`api` owns routes, authentication and event streams. `cmd/minder` wires the process.
All timestamps use UTC. The default display zone is Europe/Amsterdam.
Run `../toolchain/run go go test ./...` and `../toolchain/run go go vet ./...` from this folder.
Use test-first changes and real SQLite files for transaction tests.

`store/schema.sql` defines schema version 1. JSON item rows enforce status, version and title constraints.
A durable event cursor records changes to existing entries. Reconnects replay from that cursor.
Reserve model costs before a request; settle them from provider usage. Unknown costs keep the reservation.

Model context includes preview, result and error entries as structured assistant messages.
The latest user message stays last. Confirmed changes must not become new proposals on the next request.

`GET` and `HEAD /brand/minder-icon.png` serve only the embedded public branding PNG.
The web client, scripts, chat and all mutation routes still require a valid owner assertion in production.

Items carry `list`, `profile` and `recurrence`. `domain` owns recurrence rules, `NextDue` and `EffectiveReminder`.
Profiles live in the `profile` table (schema version 3). `store` seeds the five built-ins once and never overwrites edits.
A profile edit reschedules the reminders of items that use it in the same transaction.
`command` owns profile previews with the same Confirm, conflict and Undo rules as items. Completing a repeating item moves its due time.
The assistant proposes profile changes through `propose_profile`. Spec: `../docs/m1-spec.md`.
Items carry an optional `location`. The system prompt tells the model to ask one follow-up when a missing detail matters, and not to ask again after a decline.
`command.PreviewAll` previews up to five proposals in one transaction. Entries carry `cost_eur`: the model request cost sits on the first entry of a reply.
