# Email Capture Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task, inline without sub-agents as required by the project. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Turn authenticated forwarded emails and attachments into reviewed Minder items, with follow-up questions, source access and travel dates.

**Architecture:** A Cloudflare Email Worker hands raw mail directly to a machine-authenticated Go endpoint. Go persists the original and a durable extraction job, publishes capture updates through existing chat history, and creates item changes only through the command service. Flutter presents captures, questions, downloads and the resulting calendar items.

**Tech Stack:** Go 1.27.2, SQLite, Flutter/Dart, JavaScript Worker with Web Crypto, existing OpenRouter and Firebase adapters. Bounded PDF conversion runs in an infrastructure adapter using Poppler tools installed in the runtime image; image understanding uses the configured model capability.

**Spec:** [Forwarded email capture](../specs/2026-11-12-email-capture-design.md).

## Global Constraints

- Use the existing `Minder-m0-chat` linked worktree, currently on `notification-body-tap`. Preserve the notification work and uncommitted keyboard fix. Do not merge or push.
- Run inline without sub-agents. Before each task, read its full applicable AGENTS.md chain and record its baseline, tests, decisions and commit in a plan-specific ignored ledger.
- Never put actual forwarding addresses, credentials or production email fixtures in Git, logs or examples. Use reserved example domains in tests. Use an agent Git identity for new commits so personal addresses do not enter new commit metadata.
- Exact recipient comes from `EMAIL_RECIPIENT`; the agreed deployment address is `assistant@minder.liveleds.io`. Allowed forwarding senders come only from required `EMAIL_ALLOWED_SENDERS`; missing/invalid values fail closed.
- No Cloudflare content buffer, offline spool or catch-all. The owner may re-forward if no capture appears. Persist originals and a job on Minder before successful handoff.
- Raw MIME limit 20 MiB; at most 50 MIME parts, 25 attachments and 10 MiB per decoded attachment. Automatic document processing covers at most 50 pages/images per source and ten model calls.
- Always review before item changes. No automatic acceptance, outbound booking actions or batch Confirm. Reuse validation, optimistic versions, audit, reminder transactions and Undo.
- Initial HTTP handoff timeout 15 seconds; signature window five minutes. Five total extraction attempts with delays 5 seconds, 30 seconds, 2 minutes and 10 minutes. Respect the existing shared monthly model budget.
- Only store executes SQL. Domain/application code has no provider, HTTP, filesystem or process dependencies. Add new package boundaries and DOX owners with their first code.
- Local implementation includes deployment assets and runbook. Keep live DNS, Cloudflare provisioning, production migration and APK delivery separate from local verification; report their status truthfully.

## Review Focus

- A commit followed by a dropped HTTP response: re-forwarding cannot silently duplicate an accepted capture/item (Tasks 2, 3 and 5).
- A return flight arrives on another local date or across DST: preserve the actual instants and both zones, never calculate from clock labels (Tasks 1, 4 and 7).
- A PDF parses only partly or a calendar invite contains a revision: expose missing coverage and stale updates instead of showing success (Tasks 4 and 5).
- Two captures need answers while normal chat is awaiting a reply: answers and busy indicators remain associated with the right request (Tasks 5 and 7).
- A forwarded original is from an unlisted sender but the actual forwarder is allowed: accept it; reject forged outer From/authentication headers and credential reuse on owner routes (Tasks 3 and 6).

## Task 1: Event and date contracts through the existing item pipeline

**Files:** Modify `server/domain/model.go`, `server/domain/recurrence.go`, `server/command/attributes.go`, `server/command/command.go`, `server/store/store.go`, `server/api/api.go`, `server/assistant/prompt.go`, `server/openrouter/client.go`; create `server/domain/event.go`, `server/domain/event_test.go`, `server/command/event_test.go`, `server/store/event_test.go`.

**Interfaces:** Extend `domain.Item` and `domain.Proposal` with kind, `end_at`, `start_timezone`, `end_timezone`, `start_date`, `end_date`, `due_date` and matching clear flags. Produce `domain.Overlaps(Item, time.Time, time.Time, *time.Location) bool` for inclusive-start/exclusive-end calendar selection and date-only display; keep legacy `PlacedAt`/Anchor behaviour for existing timestamp items.

- [ ] Write failing table-driven tests: `TestEventDates` rejects simultaneous date/timestamp values, invalid dates and an end without/before start; accepts a 14-night stay; `TestFlightZones` compares UTC instants across Amsterdam/New York DST; `TestLegacyItems` preserves reminders/recurrence for existing rows; `TestCalendarOverlap` includes a stay starting before a queried month and excludes checkout as an occupied night.
- [ ] Run `toolchain/run go go test ./domain ./command ./store`; observe the expected missing-field/function failures.
- [ ] Implement additive event/date validation, proposal application/clear flags, preview field labels and model tool schemas. Existing rows default to task without changing timestamps. Extend recurrence and reminder projection explicitly: preserve legacy behaviour; date-only values do not manufacture timed reminders. Pass a validated viewing timezone into calendar selection for date-only values and retain the 62-day limit.
- [ ] Run `toolchain/run go go test -race ./...`; expected all tests pass, including long events and legacy command tests. Update affected package contracts and commit only this task's files.

## Task 2: Capture data, source files and durable jobs

**Files:** Create `server/domain/capture.go`, `server/port/capture.go`, `server/store/capture.go`, `server/store/capture_test.go`, `server/store/migrations/004_email_capture.sql`, `server/mailfiles/{AGENTS.md,files.go,files_test.go}`; modify `server/port/persistence.go`, relevant parent DOX indexes and `toolchain/architecture-check`/checker tests.

**Interfaces:** Domain values `Capture`, `CaptureCandidate`, `CaptureAttachment`, `CaptureJob`, `CaptureAnswer`, `CaptureEvidence`, `CaptureReceipt`, `CaptureEnvelope` and `CaptureDocument`; fields carry opaque IDs, state/revision, source/part references and domain proposals. Extend `port.Transaction` with capture reads/writes and source links so commands can mutate them atomically. `mailfiles.Store` implements `SaveOriginal(ctx context.Context, id string, raw io.Reader) (digest string, size int64, err error)`, `SaveAttachment`, `Open`, `DeleteSource` through a capture-owned blob interface; filesystem access stays here.

- [ ] Write failing SQLite integration tests: `TestIngestAtomicReceipt`, `TestClaimLeaseRecovery`, `TestCandidateRevisionUnique`, `TestItemSourceLinksRollback`, `TestDeletedSourceTombstone`; file tests verify binary hashes, path traversal rejection, orphan cleanup and write/fsync/rename failure with no referenced partial file. Use temporary databases/directories.
- [ ] Run `toolchain/run go go test ./store ./mailfiles`; observe missing capture storage failures.
- [ ] Add the numbered migration without changing old files. Persist captures, attachments, jobs/attempts/leases, candidate revisions, answers, many-to-many item links and request receipts. Stage durable blobs before commit; clean only unreferenced temporary files. Receipts bind request ID and raw digest; retain minimal tombstones after source deletion.
- [ ] Run `toolchain/run go go test -race ./...` and structure checks; expected green suite and old-schema upgrades. Update DOX and commit.

## Task 3: Fail-closed configuration and machine-authenticated ingress

**Files:** Create `server/mailauth/{AGENTS.md,auth.go,auth_test.go}`, `server/capture/{AGENTS.md,service.go,ingest.go,ingest_test.go}`, `server/config/email.go`, `server/config/email_test.go`, `server/api/captures.go`, `server/api/captures_test.go`; modify `server/api/api.go`, `server/config/config.go`, package indexes and architecture checker/tests.

**Interfaces:** `mailauth.FromEnvironment(func(string) string) (*Verifier, error)` validates configuration without printing values; `Verifier.Authenticate(*http.Request, []byte, time.Time) (domain.CaptureEnvelope, error)` checks the signed envelope and allowlist. `capture.Service.Ingest(ctx context.Context, envelope domain.CaptureEnvelope, raw []byte) (captureID string, duplicate bool, err error)` consumes Task 2 persistence/blob ports. `api.CaptureIngestor` exposes that method via a separately authenticated handler at `/v1/email/ingest`.

- [ ] Write failing tests: disabled feature rejects ingress while legacy startup works; enabled capture rejects missing/malformed allowlists and secrets; only single outer From with trusted aligned authentication is allowed; wrong recipient, spoofed authentication headers, altered body/metadata, expired signatures and replay-ID/content conflicts fail; a valid committed duplicate returns the original ID. Assert no actual configured addresses or secrets in errors/logs.
- [ ] Run `toolchain/run go go test ./mailauth ./capture ./config ./api`; observe expected missing ingress/configuration failures.
- [ ] Define one versioned signing format shared with Task 6: newline-separated version, Unix timestamp, request ID, base64url canonical-envelope JSON and lowercase SHA-256 raw-body digest; sign with HMAC-SHA256 and compare in constant time. Decode a base64 environment key of at least 32 random bytes. Permit only five minutes of skew. Limit request size before processing; authenticate before persistence/model access. Owner routes never recognize the machine credentials. Return 201/200 with capture ID, 401/403, 409, 413 or safe 5xx as specified. Commit original, Received entry and extraction job before success.
- [ ] Run `toolchain/run go go test -race ./...`; expected green including a dropped-response retry and closed/full storage failure. Update config/API/architecture contracts and commit.

## Task 4: Bounded MIME, calendar, PDF and image extraction adapters

**Files:** Create `server/mailparse/{AGENTS.md,mime.go,calendar.go,document.go,parse_test.go}`, synthetic fixtures under `server/mailparse/testdata/`; modify `server/assistant/model.go`, `server/openrouter/client.go`, `server/openrouter/client_test.go`, `server/go.mod`, `server/go.sum`, `Dockerfile`, package indexes and architecture checker/tests.

**Interfaces:** `mailparse.Parser.Parse(ctx context.Context, raw io.Reader) (domain.CaptureDocument, error)` produces Task 2 document values with source parts, exact attachment bytes, text/images and a per-part coverage manifest. `domain.CaptureDocument` carries trusted original-message date when available and separate forwarding envelope. Extend model messages additively for text/image parts; preserve text-only Complete callers. PDF subprocess execution is adapter-owned with context deadlines, page/output bounds and fixed argument construction, never shell interpolation.

- [ ] Write failing fixture tests: multipart alternatives/encoded filenames/nested `.eml` preserve bytes; body HTML strips scripts and never fetches remote assets; hostile MIME nesting/oversize parts reject visibly; unsupported/encrypted files remain source attachments; text PDF and scanned pages expose coverage limits; ICS timezone/UID/sequence/cancellation and body conflicts retain evidence. Test a 51-page document cannot report complete coverage.
- [ ] Run `toolchain/run go go test ./mailparse ./openrouter`; observe missing parsing/multimodal behaviour.
- [ ] Implement bounded standard-library MIME parsing; select and pin a maintained ICS parser after checking its primary documentation and fixture behaviour. Use Poppler `pdftotext`/`pdftoppm` through the adapter for PDF text/pages. Add runtime executables to Docker. Use configured OpenRouter image capability; unsupported capability gives a visible extraction warning and preserved files. Never silently select a different paid model. Keep total processing within spec limits and all external content untrusted.
- [ ] Run `toolchain/run go go test -race ./...` plus real Poppler fixture conversion in the built runtime; expected byte-identical originals, explicit partial coverage and unchanged existing text model requests. Record capabilities/provider-retention checks separately from mocked tests. Update DOX and commit.

## Task 5: Capture orchestration, questions, proposals, updates and deletion

**Files:** Create `server/capture/{worker.go,extract.go,review.go,worker_test.go,review_test.go}`, `server/assistant/capture.go`, `server/assistant/capture_test.go`, `server/command/capture.go`, `server/command/capture_test.go`; modify `server/conversation/service.go`, `server/api/captures.go`, `server/port/capture.go`, `server/store/capture.go`, `server/command/command.go`, `server/domain/capture.go`, `server/push/fcm.go` if its existing payload adapter needs a generic review type.

**Interfaces:** `capture.Extractor.Extract(context.Context, domain.CaptureDocument, []domain.CaptureAnswer) (domain.CaptureExtraction, error)` returns candidates, evidence and questions without item writes. `capture.Service.Get(id string) (domain.Capture, error)`, `Answer(ctx context.Context, id string, revision int, text string) (domain.Entry, error)`, `Retry(id string) (domain.Entry, error)`, `Run(context.Context)` and authenticated attachment access consume Tasks 2–4. `domain.Entry` gains optional capture summary/reference; command proposals bind capture/candidate/revision and source IDs. Capture deletion is a new version-checked command action with normal preview/Confirm.

- [ ] Write failing tests for two captures plus normal chat; material question vs optional detail; six candidates without five-tool-call truncation; answer supersedes only affected revisions; stale Confirm cannot apply; concurrent/double Confirm creates one item/audit/source link; reschedule/cancellation matching preserves unrelated segments; re-forward reports duplicates; zero-actionable email remains visible; deleted source revokes downloads and blocks delayed replay recreation.
- [ ] Run `toolchain/run go go test ./capture ./assistant ./command ./conversation ./api`; observe missing workflow behaviour.
- [ ] Implement one lease-driven extraction worker and capture-isolated context. Share existing usage reservations across chat/document calls; persist attempt/call counts before provider calls. Publish candidates, question entry, cursor and job state atomically, using a transaction-local command preview helper instead of nested transactions. Respect the five-proposal boundary through explicit groups, with no missing candidates. Apply source links/audit/reminders in the command transaction. A new answer cannot reopen terminal candidates. Ordinary ambiguous chat asks which capture; explicitly targeted answers bypass unrelated chat busy state.
- [ ] Add retry clock tests for 5s/30s/2m/10m and five attempts; budget/model-call exhaustion keeps originals and visible recovery controls. Generic review notifications use the existing push transport with a deduplicated outbox record; no lock-screen source text. Source deletion removes raw/decoded files, excerpts, pending model inputs and public access, keeps confirmed items and a replay tombstone, and explains backup retention.
- [ ] Run `toolchain/run go go test -race ./...`; expected all transactional, lifecycle, cost and privacy tests pass. Update DOX and commit.

## Task 6: Cloudflare Worker and cross-language ingress tests

**Files:** Create `email-worker/{AGENTS.md,worker.mjs,worker.test.mjs,wrangler.toml,README.md}`, `toolchain/email-ingress-vectors.json`; modify root DOX index, `toolchain/check`, `toolchain/AGENTS.md`, `server/mailauth/auth_test.go`.

**Interfaces:** Export `email(message, env, ctx)` handler plus testable admission/signing helpers. Task 3 signing vectors use invented addresses and fixed binary MIME digests. Worker reads only the spec environment names and sends the exact MIME body plus signed canonical envelope; no persistent storage bindings.

- [ ] Write failing Node tests using Web Crypto and injected fetch/message streams: allowed outer sender carrying an arbitrary forwarded original, forged sender/authentication data, absent env, altered payload, exact binary forwarding, 413, timeout/non-2xx and returned capture ID. No handler success after an unacknowledged POST. Go verifies signatures generated from the same fixed vectors and rejects altered fields.
- [ ] Run `node --test email-worker/worker.test.mjs` and `toolchain/run go go test ./mailauth`; observe missing worker implementation/vector handling.
- [ ] Implement the handler, exact-recipient/outer-From admission and shared HMAC serialization. Establish which authentication results Cloudflare controls using primary documentation and deployed tests; do not trust arbitrary raw headers. If provenance cannot be established, fail closed and report the integration blocker. Supply service-token headers, a 15-second abort, raw-body/part limits, fixed HTTPS destination and sanitized errors. Wrangler configuration contains names/placeholders only, no sender values or secrets.
- [ ] Run Node and Go cross-language suites; expected valid signatures interoperate and invalid cases never call ingestion. Add Worker tests to the main check command; update ownership docs and commit.

## Task 7: Flutter capture review, downloads and travel calendar rendering

**Files:** Create `app/lib/domain/{capture.dart,capture_repository.dart}`, `app/lib/application/capture_controller.dart`, `app/lib/ui/capture_view.dart`, `app/lib/data/{attachment_download.dart,attachment_download_stub.dart,attachment_download_web.dart}`, `app/test/{capture_test.dart,capture_api_test.dart}`; modify `app/lib/{main.dart,domain/entry.dart,domain/item.dart,data/wire.dart,data/chat_api.dart,application/chat_controller.dart,ui/entry_view.dart,ui/chat_screen.dart,ui/calendar_screen.dart,ui/format.dart}`, `app/test/calendar_test.dart`, and Android files `app/android/app/src/main/kotlin/io/liveleds/minder/{MainActivity.kt,AttachmentDownloads.kt}`, `app/android/app/src/main/AndroidManifest.xml`, `app/android/app/src/main/res/xml/attachment_paths.xml`.

**Interfaces:** `CaptureRepository.getCapture(String id)`, `answerCapture(String id, int revision, String text)`, `retryCapture(String id)`, `downloadAttachment(String captureId, String attachmentId)` return typed captures/entries/download bytes. `CaptureController` owns selected capture/revision, targeted answer/action state and disposal guards. `AttachmentDownload.save(name, mediaType, bytes)` lives in data: web Blob download; Android MethodChannel writes a sanitized cache file and opens it via a narrowly configured FileProvider. Capture state travels through the existing chat cursor.

- [ ] Write failing controller/API/widget tests: targeted answers amid multiple pending captures and unrelated chat; Received/Processing/Failed/zero-item/partial attachment states; disabled/stale/confirmed candidates; six individually reviewable candidates; authenticated byte downloads with safe names; disposal/retry; notifications reveal no subject; flight local zones and a 14-night hotel across month boundaries. Add the corresponding chat/capture HTML mockup and rendered PNG under `docs/mockups/` with synthetic examples, keeping the existing visual system.
- [ ] Run `toolchain/run flutter flutter test`; observe missing capture types/routes/widgets and event rendering.
- [ ] Implement typed transport mappings and controllers, capture summary/Answer/Retry/source access and candidate evidence around existing preview controls. Preserve the uncommitted keyboard-following change and normal chat waiting semantics; capture updates must not clear a still-pending unrelated chat reply. Keep 44px controls, existing headers/backgrounds/nav and narrow-screen/text-scale behaviour. Keep new tests isolated from unrelated existing goldens; update only intentional new rendering with reviewable mockups.
- [ ] Run `toolchain/run flutter flutter test`, analyze and format; expected full suite green including keyboard regressions. Build Android debug/release to validate native download integration; verify binary download on emulator without using a physical phone. Update layer/platform/test DOX and commit this task's changes without silently absorbing the earlier keyboard diff.

## Task 8: Composition, integration evidence and deployment package

**Files:** Modify `server/cmd/minder/{main.go,lifecycle.go,lifecycle_test.go}`, `server/config/config_test.go`, `README.md`, `Dockerfile`, `.dockerignore`, `docs/{requirements.md,master-plan.md,architecture.md,logging.md}`, `HANDOFF.md`; create `toolchain/{email-local-check,email-check.cjs}`, `docs/evidence/email-capture.md`, `docs/email-capture-operations.md`. Update affected DOX indexes and architecture checker tests.

**Interfaces:** Compose capture service with parser, blob storage, model extractor, transaction store and notification transport only when enabled. Start and cancel/wait for the capture worker alongside existing reminder/conversation lifetimes. `toolchain/email-local-check` uses synthetic mail, temporary SQLite/source directories and deterministic model replies; it owns any local ports it starts and restores them on exit.

- [ ] Write failing lifecycle/integration tests for disabled capture, enabled configuration, shutdown during extraction, durable restart and signed delivery through compiled Flutter web. Browser flow: deliver an email with PDF/ICS, answer a missing date, confirm one candidate, inspect calendar/source download, send an update/cancellation and exercise retry/duplicate rejection.
- [ ] Run focused lifecycle and email integration checks; observe the missing composition/integration failure.
- [ ] Wire the runtime and Docker document tools; preserve existing services/ownership. Document exact environment names, private secret setup, persistent source mount, backup/restore, path-scoped Access policy/service token, subdomain-only MX/routing and rollback. Prepare concrete deploy commands/configuration without changing live DNS or deploying. Check the current provider's document capabilities, retention settings and costs against primary sources; record what still requires account readback.
- [ ] Run `toolchain/check` (includes Worker tests), build release web, then run existing chat, reminder and calendar scenarios sequentially plus the new email scenario. Verify Go race/vet, Flutter analyze/format and real temporary-SQLite failure tests. Record commands, exit codes and actual results in evidence; no claims of live delivery/phone approval from local tests.
- [ ] Review the complete diff inline per project no-subagent rule, fix important findings with failing regression tests, and repeat only affected checks plus the full suite when needed. Scan staged content/new commit identities for private sender values. Update the handoff with implementation, runtime and pending deployment status; keep unrelated dirty files intact. Commit the final implementation/evidence, retain the branch and present the deployment package for any required live-service approval.

## Coverage and execution handoff

Tasks 1–2 establish the shared types/transactions; Tasks 3 and 6 establish trusted delivery; Tasks 4–5 implement extraction and review; Task 7 presents them on Android/web; Task 8 proves the composed system. Every spec acceptance check maps to these tasks; original-file retention, deletion, duplicate recovery and budget handling each have explicit tests.

The owner approved the specification by requesting implementation. Inline execution is already the project preference. This plan is complete for review; execution has not started. After the required plan review, proceed continuously through all tasks, keeping live-service changes separate from local readiness.
