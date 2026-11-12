# Email capture operations

Specification: [email capture](superpowers/specs/2026-11-12-email-capture-design.md).
The feature is off by default. A local build or Git push does not create the email address or deploy the server.

## Runtime configuration

Configure actual sender values privately. Do not put them in Git, examples, test fixtures, shell history or logs.
Use TokenCrypt/process variables or the provider's secret editor. Both the Worker and Go enforce the allowlist.

| Setting | Server | Worker |
|---|---|---|
| `EMAIL_CAPTURE_ENABLED` | Set true to enable ingestion and its background worker; false preserves existing deployments | Not used; a deployed email handler requires valid configuration |
| `EMAIL_ALLOWED_SENDERS` | Required comma-separated bare addresses | Same privately configured value |
| `EMAIL_RECIPIENT` | Required exact recipient | Same exact recipient |
| `EMAIL_INGEST_HMAC_KEY` | Base64 key representing at least 32 random bytes | Same secret |
| `EMAIL_SOURCE_DIR` | Writable directory inside the persistent volume, for example `/data/email` | Not used |
| `EMAIL_INGEST_URL` | Not used | Fixed HTTPS ingestion URL ending `/v1/email/ingest` |
| `EMAIL_ACCESS_CLIENT_ID`, `EMAIL_ACCESS_CLIENT_SECRET` | Access policy at the edge admits this identity; Go still checks HMAC | Dedicated Access service-token credentials |

The server uses its existing OpenRouter configuration and monthly model budget. Each email retains a call count across retries.
Poppler `pdfinfo`, `pdftotext` and `pdftoppm` must be present; the Docker runtime installs `poppler-utils`.
JPEG/PNG and scanned pages require the configured model to accept image input. No silent paid-model substitution occurs.
Check that capability and the selected provider's content-retention settings before live capture. Fixture tests do not verify a real model or retention policy.

## Activation order

1. Back up the database and persistent volume. The additive database migrations create captures, receipts, source links and a review-notification outbox.
2. Deploy the reviewed server image with the capture feature enabled and source directory on the persistent volume. Confirm that the existing app, reminders and owner authentication still work.
3. Create a path-scoped Cloudflare Access application/policy for `/v1/email/ingest`, admitting only the dedicated service token. Preserve the owner-only policy for every other route.
4. Configure the Worker environment and secrets privately. Its manifest has no content storage bindings. Deploy `email-worker/worker.mjs` with Wrangler and verify the fixed destination and clock.
5. Enable Email Routing only for the `minder.liveleds.io` subdomain. Route `assistant@minder.liveleds.io` to the Worker. Preserve root-domain Private Email records and the existing web hostname. Do not enable a catch-all.
6. Forward from each permitted sender. Verify Cloudflare's first trusted authentication result, a capture receipt, an app review and byte-identical attachment download. Verify an unlisted/forged sender fails before model processing.
7. Test server downtime and a failed Worker POST. Record the actual bounce/retry behaviour; the app's absence of a capture is the owner's signal to investigate and re-forward. This design has no Cloudflare recovery buffer.
8. On a phone, answer a question, confirm one item, inspect travel dates and open an attachment. Phone approval still gates merging under the project workflow.

No live DNS, Access policy, Worker or production deployment changed during local implementation.

## Normal operation

A durable Received entry means the server saved the exact original and extraction job. The one-second job worker parses it,
publishes review candidates/questions and queues one generic notification per source revision/device.
The source remains available if parsing fails or the monthly budget is exhausted.

Only Confirm changes items and reminder schedules. Other candidates stay independent. A targeted clarification supersedes that candidate's old preview.
Transport retries return the original capture receipt. Different MIME can require review as a possible re-forward or booking update.
Version checks prevent stale updates and confirmations from overwriting newer items.

Admission limits are 20 MiB of raw MIME, 50 MIME parts, 25 attachments and 10 MiB per decoded attachment.
Automatic document conversion stops at 50 pages/images. Conversion, unsupported/encrypted files and model limits leave visible coverage warnings or a recoverable failure.
Do not treat a warning as evidence that an attachment was read.

## Failure tracing

Use capture/job IDs, stage, counts and safe error codes. Never copy email bodies, subjects, addresses, credentials or attachments into operational logs.
The `email extraction completed` event reports revision and candidate count. Existing request and model logs provide timing and cost boundaries.

- No Received entry: check routing, trusted sender authentication, Worker configuration, Access service token, HMAC key, request clock and ingestion reachability. Re-forward after correcting the problem.
- Received/Processing: inspect the job lease, provider readiness, source storage and document tools. A crashed worker's lease can be reclaimed after five minutes.
- Failed: inspect the safe error, retry after correcting configuration or budget, or provide readable material. Transient extraction delays are 5 seconds, 30 seconds, 2 minutes and 10 minutes; five total attempts then stop.
- Missing review notification: inspect the source revision/device outbox and existing FCM registration. The capture remains in chat even if push fails.
- Wrong proposed facts: clarify the affected candidate. Do not confirm it and expect the model to repair it later.

## Sources, deletion and backups

Keep SQLite and `EMAIL_SOURCE_DIR` in the same backup set. Originals and decoded files use private permissions and opaque IDs;
attachment filenames never become storage paths. Backups must restore both data and source links. Confirm current host/backup encryption before production; the app does not claim end-to-end or application-level disk encryption.

Delete email source creates an explicit irreversible preview. Confirm commits a minimal tombstone, revokes downloads,
redacts source summaries/excerpts in history and invalidates pending source previews. Confirmed items remain.
The worker removes private source files and resumes interrupted cleanup after restart. It serializes attachment writes against deletion so files cannot reappear afterward.
Source deletion cannot retract content already sent to a model provider or remove older backups before their retention expires.

For a restore check, stop writes, take a consistent SQLite snapshot and copy the source directory, restore both to an isolated instance,
verify attachment SHA-256 values and source links, then test one pending review. Do not restore over production as a verification step.

## Local checks and rollback

- `toolchain/check`: Worker tests, structure/DOX checks, Go race/vet and Flutter tests/analyze/format.
- `toolchain/email-local-check`: signed ingress and a compiled-web review against synthetic mail, an isolated model fixture and real temporary SQLite/source storage.
- `toolchain/local-check`, `toolchain/reminder-local-check`, `toolchain/calendar-local-check`: regression scenarios, each with its own fixture port.
- `node --test email-worker/worker.test.mjs`: sender admission and shared Go/JavaScript signing vectors.

For rollback, disable the email routing rule first, then set `EMAIL_CAPTURE_ENABLED=false`. Preserve originals and database backups.
Do not start an older binary against a newer schema: the existing migration guard refuses it. Use a matching image or an approved backup restore.
