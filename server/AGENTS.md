# Server

Go HTTP service. `domain` owns values and rules. `store` alone writes SQL. `command` owns item and profile changes: preview, confirm, discard and undo. `assistant` proposes changes only. `api` owns routes and event streams. `access` owns request authentication and identity settings.
`config` validates process settings. `cmd/minder` wires the process.
All timestamps use UTC. The default display zone is Europe/Amsterdam.

## Checks

Run these from this folder; each must pass before a change is done:
- `../toolchain/run go go test -race ./...`
- `../toolchain/run go go vet ./...`
- `../toolchain/run go gofmt -l .` prints nothing

Write the failing test first. Use real SQLite files for transaction tests.

## Storage

- `store/migrations/NNN_name.sql` are the schema steps. File N takes the database from version N-1 to N. `Open` applies the missing steps, each in one transaction, and refuses a database from a newer release. Add a new file for every schema change; never edit an old one.
- Item, chat and profile rows keep their data as JSON. Item rows enforce status, version and title constraints.
- A durable event cursor records changes to existing entries. Reconnects replay from that cursor.
- `store` seeds the five built-in profiles once and never overwrites edits.

## Rules in `domain`

- `Validate` and `ValidateProfile` name the bad field with `*Invalid`.
- `Complete` finishes an item. A repeating item stays open and moves to its next occurrence. Chat, the reminder card and the notification all use it.
- `Anchor` is the deadline, else the start; reminders and repeats count from it. `PlacedAt` is the start, else the deadline; the calendar uses it.
- `EffectiveReminder`: a reminder on the item wins; otherwise the profile gives one from the anchor; otherwise none.

## Changes and the assistant

- A profile edit reschedules the reminders of the items that use it in the same transaction.
- `command.PreviewAll` previews up to five proposals in one transaction. One invalid proposal leaves no preview behind.
- The model context holds at most 200 open items, newest change first, plus the list names and profiles. Finished items never reach the model.
- The system prompt asks for one follow-up question when a missing detail matters, and not again after a decline.
- Reserve model costs before a request; settle them from provider usage. Unknown costs keep the reservation. A reply's cost (`cost_eur`) sits on its first entry.
- Model context includes preview, result and error entries as structured assistant messages. The latest user message stays last. Confirmed changes must not become new proposals.
- Error codes: `invalid_field` 400, `conflict` 409, `not_found` 404, `limit_reached` 429, `model_error` 502, `internal_error` 500. Model and internal errors show a fixed message, never their details.

## Runtime boundaries

- `api.Authenticator` accepts a request and returns an identity error. Keep provider headers and JWT logic in `access`.
- `access.FromEnvironment` requires explicit issuer, audience and owner settings. It returns no authenticator only in development or tests.
- `config.Load` rejects unknown environments, custom production model endpoints and non-finite budget values.
- Keep deployment identities out of `cmd/minder`.

## Routes

`api.New` registers every route with a method and path pattern. `GET /v1/health` and `GET /brand/minder-icon.png` need no sign-in. Everything else needs a valid owner assertion in production, and writes under `/v1/` must come from this site.
An unknown `/v1/` path answers 404 and never falls through to the web app.
`GET /v1/items?from=&to=` returns the items placed in a range of at most 62 days.
`/branding/` serves only `index.html`, `app-icon.svg` and `app-icon.png` from `BRAND_DIR`. Empty `BRAND_DIR` turns it off. The Dockerfile copies those three files to `/srv/branding`.
A panic in the assistant becomes an error entry and frees the server for the next message.

Specs: `../docs/m1-spec.md`, `../docs/calendar-spec.md`, `../docs/notifications-spec.md`.
