# Forwarded email capture

Status: specification complete for owner review, 2026-11-12. Implementation planning and infrastructure changes have not started.
Requirements: [../../requirements.md](../../requirements.md). Current system: [../../architecture.md](../../architecture.md).

## Intent and agreed choices

The owner forwards emails to `assistant@minder.liveleds.io`. Minder reads the body and attachments,
keeps the originals, and proposes appointments, bookings, flights, hotel stays, tasks, deadlines and reminders.
One email may produce several items. Minder asks follow-up questions when details matter.
The owner must review and confirm every proposed item change, including updates and cancellations.

The owner chose Cloudflare Email Routing and the existing OpenRouter model setup.
Allowed forwarding addresses come from the required environment parameter `EMAIL_ALLOWED_SENDERS`.
Configure the owner-approved addresses privately at deployment. Do not put actual addresses in tracked files, examples, application defaults or Worker source.
Cloudflare receives readable messages. OpenRouter and the selected model provider receive the content needed for extraction.
The pipeline is not end-to-end encrypted. Neither confirmation nor low model confidence changes these privacy boundaries.

The owner decisions above and direct delivery below are agreed. This spec supplies concrete defaults for the remaining details; written-spec approval is still pending.

## Scope and examples

| Input | Expected proposals |
|---|---|
| Appointment confirmation | Appointment with explicit start, location and end if supplied |
| Flight itinerary | One event per flight segment, with departure and arrival locations, local times, zones and booking reference |
| Hotel booking | One stay spanning check-in through checkout, with address, dates and reference |
| Invoice | Payment task with deadline, amount/currency and reference in notes |
| Delivery notice | Delivery event/window or tracking task, according to the stated information |
| Newsletter with several dates | Separate candidates for the actionable dates; a question about alternatives rather than importing every mentioned date |
| Changed booking | Proposed changes to the matched existing items |
| Cancellation | Proposed cancellation of the matched existing items |

Capture covers tasks, events and reminders. It does not reply to senders, pay invoices, open booking links,
make bookings, accept invitations on the sender's system, or sync another calendar.
Follow-up answers and reviews happen in Minder chat. Outbound email is outside this slice.

## Receiving and delivery

Agreed architecture:

`sender → Cloudflare Email Routing → Email Worker → authenticated HTTPS POST → Minder originals + SQLite job → extraction → chat review → Confirm`

The owner chose a direct handoff without a temporary Cloudflare buffer. If no review appears in the app,
the owner checks the failure and can re-forward the email. We do not promise delivery recovery while Minder is offline.
Cloudflare routing to Workers and subdomains is documented; the actual account and delivery behaviour still need deployment checks.

Delivery contract:

- Worker accepts only recipient `assistant@minder.liveleds.io` and forwarding senders listed in `EMAIL_ALLOWED_SENDERS`. Reject other senders before handing content to Minder or the model. Require a single valid outer From mailbox that matches the allowlist (case-insensitive, with no automatic plus-address or dot-alias expansion), and a passing DMARC result aligned to that outer From from trusted Cloudflare authentication results. A forged From header or authentication header supplied inside the message grants no access. The embedded original sender may be anyone; the allowlist applies to the person forwarding the email, not the quoted message or attachment. Minder revalidates the signed metadata and allowlist at ingestion.
- `EMAIL_ALLOWED_SENDERS` is a required comma-separated list of bare email addresses. Trim surrounding whitespace, validate each address and compare case-insensitively. Empty, missing or malformed values fail closed: the Worker rejects capture and the Go service refuses to enable email ingestion. No built-in address fallback. Configure the same value in the Worker environment and Go process environment; both enforce it independently. Configuration changes require applying the new environment to both services.
- Worker posts the exact raw MIME bytes and trusted envelope/authentication metadata directly to Minder over HTTPS. No R2, KV, queue or other application-managed Cloudflare content storage. No body/attachment logging.
- Ingestion authenticates the Worker independently of browser sign-in. Use a dedicated Cloudflare Access service token at the edge plus an application HMAC over the timestamp, request ID, trusted metadata and raw-body digest. Validate the signature and a five-minute timestamp window. Keep and rotate credentials through secrets configuration.
- `POST /v1/email/ingest` is a machine-authenticated route, not an unauthenticated bypass. The Worker identity cannot call owner chat, item or download routes. Existing routes retain owner authentication.
- Go verifies and saves the raw email to its persistent volume, then commits the source, fingerprint, ingestion status entry and extraction job before returning success with the capture ID. Attachment decoding runs from the saved raw email; malformed parts can fail extraction without losing the source.
- Persist a unique raw-message fingerprint and source ID atomically. Duplicate delivery returns the existing capture ID without another job. A timeout after commit followed by re-forwarding must not silently duplicate previews or confirmed items.
- Worker treats only a successful durable-ingestion response as successful handoff. Surface non-success/timeouts as handler failures; do not return success after a failed POST. Verify actual SMTP bounce/retry behaviour in a deployed fault test. Do not assume Cloudflare retries Worker delivery.
- No persistent pre-ingestion retries or offline spool. Once Minder accepts the email, its own durable jobs handle extraction retries and restart recovery. A Received/Processing entry distinguishes accepted mail from an email that has not reached Minder.
- Back up originals and SQLite together on Minder's persistent volume. Record safe delivery/extraction error codes without including message content.

Use a 15-second handoff timeout; ingestion only stores and enqueues the email, never waits for model parsing.
A successful new ingestion returns HTTP 201 with the capture ID; an exact duplicate returns HTTP 200 with the same ID.
Return 401/403 for failed machine/sender checks, 413 for size limits and a safe 5xx for storage failures.
A duplicate request ID with changed content is a conflict; a valid replay within the timestamp window of the same committed request returns its original capture ID. Expired signatures are rejected even for previously committed requests.

Configure routing for the `minder.liveleds.io` subdomain only. Preserve root-domain Private Email records.
No catch-all address. Keep the existing web hostname and routes working.
Test externally before considering the address ready. Credentials stay in TokenCrypt or process variables.

## Configuration

| Environment parameter | Where | Contract |
|---|---|---|
| `EMAIL_CAPTURE_ENABLED` | Go | Defaults to false, preserving existing deployments. True enables ingestion/jobs and requires valid capture settings. Worker configuration is mandatory whenever deployed. |
| `EMAIL_ALLOWED_SENDERS` | Worker and Go | Required comma-separated bare addresses. Actual values stay in deployment secrets/environment, never Git, fixtures or examples. |
| `EMAIL_RECIPIENT` | Worker and Go | Required, set to the agreed capture address; one exact recipient, no catch-all. |
| `EMAIL_INGEST_URL` | Worker | Required HTTPS URL for the ingestion route. Never derived from email content. |
| `EMAIL_INGEST_HMAC_KEY` | Worker and Go | Required secret, at least 32 random bytes; separate from owner credentials and model keys. |
| `EMAIL_ACCESS_CLIENT_ID`, `EMAIL_ACCESS_CLIENT_SECRET` | Worker | Dedicated Access service-token credentials, sent through the documented service-token headers. |
| `EMAIL_SOURCE_DIR` | Go | Required persistent-volume directory for raw/decoded files; refuse to enable capture if it cannot be written. |

Provision a path-scoped Access application/policy for ingestion. The existing owner application continues to protect all other routes.
The Go signature check remains required even if the origin can be reached without the Cloudflare proxy.
Reject capture requests while the feature is disabled. Bad capture settings must not silently enable broad access.
Mask sender values in startup logs and errors. Tests use invented addresses on reserved example domains.
Record configuration presence and validation outcomes, not values. Worker and Go allowlists must agree before rollout.

## Parsing and attachments

Always preserve the exact original `.eml` and the decoded bytes of every accepted attachment.
An attachment belongs to its source email; confirmed items link to that source rather than duplicating files.
Review cards and later item queries provide authenticated access to originals and attachment downloads.

Initial extraction formats: plain text, HTML converted to text, calendar `.ics`, PDF text, and JPEG/PNG images or scanned PDFs through a model-capable document/image adapter.
Validate the configured model's actual capabilities before enabling those formats. Keep the provider configurable.
Unsupported, encrypted, corrupt or password-protected files remain downloadable and show a warning.
Ask the owner for missing information or a readable replacement; never silently treat unreadable files as parsed.
Do not ask for passwords in chat.

Use bounded MIME parsing and a maintained calendar parser. Do not execute attachments, macros, scripts,
HTML, or embedded instructions. Do not load remote images or fetch links. Email and attachment text are untrusted source data,
not instructions that can authorize commands or expose credentials.

Admission defaults: 20 MiB raw MIME, 50 MIME parts, 25 attachments, and 10 MiB per decoded attachment.
Enforce the effective minimum of these limits and current platform limits. Reject oversize messages visibly; do not truncate and accept them.
Process documents in bounded jobs, up to 50 PDF pages or images per source automatically.
Preserve larger documents but mark extraction incomplete and ask for the relevant pages. Do not silently omit content.
Do not unpack general archives in this slice.

Prefer structured `.ics` fields where they exist, but report conflicts with the email body instead of silently choosing.
Handle calendar UID, recurrence identifier, sequence and cancellation method for matching and review.
Recurring schedules that cannot map to Minder's recurrence model require clarification; never approximate them silently.
Extract titles, dates, start/end, zones, locations, deadlines, durations, reference numbers, amounts and useful links.
Each candidate records evidence: source part/page and a short supporting excerpt.
Published results include a coverage manifest for every MIME part: parsed, not actionable, unsupported, unreadable or deferred by a limit.
Do not label a capture fully parsed while any source part needs attention. Keep the original intact even when extraction is incomplete. Body-only proposals may remain reviewable alongside unreadable-attachment warnings; show the incompleteness before Confirm.

## Dates and event representation

The current item model has start, deadline, duration and one timezone. It has no event end, all-day/date-only values,
source links or attachments. Implementing those contracts is part of this subsystem; hotel checkout must not become a task deadline.

Required model additions:

- Item kind: task, event or reminder; existing rows migrate to task without changing their schedule.
- Explicit `end_at` for timed events, distinct from `due_at` for deadlines. End requires a start and must follow it. End is optional if unknown and not necessary for the event. Derive elapsed duration from start/end; reject contradictory supplied duration. Do not apply the existing one-week duration cap to an explicitly bounded long stay.
- `start_timezone` and `end_timezone` as IANA zones for timed travel, while storing instants in UTC. Preserve the original local time and source evidence for review.
- `start_date`, `end_date` and `due_date` are ISO calendar dates, mutually exclusive with their corresponding timestamp fields. A date-range end requires a date start and must follow it. Ranges use an exclusive end. A single-day event can use a start date alone. Missing time stays missing; no invented midnight appointment or payment cutoff.
- Source IDs and attachment manifests. A source may link to many items; an item may link to multiple update emails.

Display both local departure and arrival times with zones for flights. Never calculate flight duration by subtracting clock labels in different zones.
Use an explicit source timezone or resolve the location to a zone. Ask if the location, date, year or daylight-saving interpretation is ambiguous.
Europe/Amsterdam is the owner default for local appointments, displayed as an assumption when the source supplies no zone.
Anchor relative phrases to the original message date when reliable, not the date the email was forwarded.

A hotel stay without check-in/out times remains a date range. Show it on each occupied date with checkout identified separately in the stay details. Checkout is not an extra occupied night.
Validate recurrence with the new event fields: move each occurrence's start/end together without turning the end into a deadline, and keep date-only recurrence on local calendar dates. Unsupported travel recurrence asks for clarification.
The calendar must include events overlapping the requested range, rather than only events starting inside it.
Preserve the existing 62-day query limit. Timed multi-day events use overlap; date-only ranges use the viewing zone's dates.
Date-only deadlines appear in the date's agenda without an invented time. Reminder generation for date-only values requires an explicit reviewed reminder time;
existing timed profiles must not manufacture one. Show any proposed reminders in the review before confirmation.

## Review and follow-up conversation

Persist each email as a capture with its own context. New emails do not replace an unanswered capture.
Questions, answers, candidates and source access carry the capture ID, separate from unrelated chat.
A reply from a capture's Answer action targets it explicitly. If a free chat reply could refer to several captures, ask which one.

Create one source summary in chat, with sender, subject, received date, attachment status and proposed item count.
The owner can open an individual candidate, answer a question, change details in chat, Confirm or Discard.
Keep independent candidates reviewable while another candidate needs clarification.
Ask focused questions about material omissions; optional missing details do not block capture without a reason.
For example, an online meeting URL can supply a location; a flight's known airport needs no question for a street address.

After an answer, regenerate only the affected candidates. Supersede their prior previews so a stale Confirm cannot apply old details.
Retain already confirmed items and their source links. Show changed fields and unresolved assumptions.
Reuse existing command validation, version checks, audit and Undo. Extraction never writes final items or schedules reminders.
Confirm applies one candidate atomically with its source links, audit and reminder state. A failure leaves it pending with an actionable error.
Confirm is idempotent: a repeated request returns the prior result and creates no second item or audit action.
No batch Confirm in the first slice. Discard does not delete the original source or another candidate.

Send one generic Android notification when a capture needs review or an answer. Avoid subject, attachment contents and booking references on the lock screen.
The open app receives capture updates through the existing event stream; reconnect recovers them from durable history.

### Example conversation

- Minder: "Your booking contains two flights and a hotel stay. The return flight date is missing. What date is it?"
- The owner can review the outbound flight and hotel while answering the return-flight question.
- After the answer, Minder shows a separate preview for each candidate with source evidence, dates/zones, attachments and any proposed reminder.
- Confirming the hotel creates only that stay. The flights remain pending until the owner confirms or discards each one.
- A later cancellation email proposes a change to the matched booking. It does not cancel anything on arrival.

## Duplicates and booking updates

Use a hash of exact raw input for transport idempotency. Do not rely solely on Message-ID.
A re-forward can have different MIME and Message-ID: match embedded original identifiers, calendar UID/sequence,
booking reference and segment/date information, then show possible duplicates for review.
Never automatically delete or merge existing items based on model similarity.

An exact transport retry is silent and does not incur another extraction charge after a completed result.
A likely re-forward reports the existing capture/item and asks whether there is new information.
Different content with the same Message-ID must not disappear as a duplicate.
A cancellation or reschedule produces version-checked proposals linked to both source emails.
If several items match, ask the owner which booking is affected. Preserve unrelated segments and hotel nights.
Stale booking updates must not overwrite a newer version; show the sequence/date conflict.

## Responsibilities and interfaces

| Component | Owns | Boundary |
|---|---|---|
| Email Worker | Admission, trusted envelope, direct HTTPS handoff | Machine-authenticated ingestion; no persistent content storage, model calls or item changes |
| Go capture service | Ingestion, durable jobs, deduplication, source links, question context | Source/job repository, blob store, extractor and command interfaces |
| Document adapters | MIME, PDF/image and calendar conversion | Bounded source parts and evidence; no SQL or item writes |
| Assistant | Extraction orchestration and cost accounting | Existing model and proposal contracts extended for capture context |
| Command service | Validated previews, Confirm, Discard, Undo | Existing transaction boundary extended for source links and event fields |
| SQLite/blob adapters | Job leases, source metadata, attachment bytes | SQLite transactions and atomic persistent-volume writes |
| Flutter | Capture summary, candidate reviews, answers and downloads | Typed repositories; no transport maps in UI |

Owner-authenticated routes: `GET /v1/captures/{id}`, `POST /v1/captures/{id}/answers`,
`POST /v1/captures/{id}/retry`, a version-checked source-deletion preview through the command interface, and `GET /v1/captures/{id}/attachments/{attachmentId}`.
Original `.eml` is a named attachment. Downloads use opaque IDs, attachment disposition, safe filenames,
no content sniffing and no public caching. Metadata/text rendering escapes untrusted content.
The separate `POST /v1/email/ingest` route accepts only the authenticated Worker as specified above. It does not accept owner follow-up answers or item commands.

Capture status: received, processing, needs-answer, ready-for-review, reviewed, or failed.
Candidate status is separate: needs-answer, pending, confirmed, discarded or superseded.
A capture remains needs-answer while any candidate or attachment coverage needs clarification; other candidates can still be confirmed.
It becomes reviewed when all active candidates have terminal outcomes and all coverage warnings are acknowledged. A zero-item email shows "No actionable items found" with the source and an option to explain what to capture.
Retry does not reopen confirmed/discarded candidates. A source with previous results keeps them visible while failed work retries.

Persist sources, attachment manifests, jobs/leases, candidate revisions, follow-up answers, item-source links and minimal ingestion receipts.
Each candidate revision binds its source IDs, evidence, parsed fields, question state and command preview ID.
Use relational uniqueness for ingestion fingerprints and active candidate revisions. Keep item mutations inside the existing transaction interface.
Publish capture changes on the durable chat cursor; no second client synchronization channel is required.
Jobs use leases so a restart can reclaim abandoned work. Publish results and job completion atomically.
Stage blob writes before committing references; clean up orphaned temporary files after interrupted ingestion.

## Failure handling, retention and budget

Use the existing monthly model budget and cost reservations. Account for every text/image/document call.
Budget exhaustion keeps the email and a resumable job; no retry loop can bypass the budget.
Retry transient storage/provider failures with bounded exponential backoff. Use retry delays of 5 seconds, 30 seconds, 2 minutes and 10 minutes; after five total attempts, show Failed and Retry. Authentication, unsupported-format and validation errors do not enter a transient retry loop.
Unknown provider costs retain the existing reservation. A lease retry may repeat a provider charge after a crash;
idempotent result publication prevents duplicate previews, not all duplicate billing.

Limit automatic extraction to ten model calls per source; keep partial results explicit and ask how to proceed if more work is needed.
Aim for a preview within 30 seconds for a short email when services are healthy. Large documents may take longer; show Processing.
Never promise a successful parse for every forwarded file.

Retain server originals and attachments until the owner explicitly deletes the source. This slice does not promise encryption at rest; record the deployed disk/backup protection before production rather than assuming it. Discard and Undo preserve provenance.
Provide a source-delete action with confirmation: remove raw and decoded content, revoke download access and clear source excerpts;
confirmed items remain unless separately removed. Retain a minimal ingestion fingerprint tombstone so a delayed transport retry cannot recreate deleted content. Explain that older backup copies remain until backup expiry.
Include originals, source links and attachment hashes in backup/restore checks.

Cloudflare's documented routing analytics include sender, recipient, subject, IDs and processing data for 31 days.
Our Worker adds no persistent content storage. Minder retains the originals after durable ingestion.
Cloudflare still processes readable content and retains routing analytics; do not describe that as retaining nothing. External model retention depends on the configured provider;
verify and record the actual policy and routing settings before production. Do not claim zero retention.
Logs contain capture/job IDs, counts, sizes, stage, durations and safe error codes, never content, credentials or subject lines.

## Acceptance checks

1. Forward a real email to the configured address. The server ingests it while the app is closed and later shows a review.
2. A booking with PDF and binary/image attachments preserves byte-identical downloads and the original `.eml`.
3. An itinerary crossing time zones and daylight saving produces correct segment instants, durations and displayed zones.
4. A multi-night hotel appears throughout its stay; checkout stays distinct from a payment deadline. A date-only invoice retains its date without an invented hour.
5. Multiple captures and answers remain associated with their own sources. Answering a question supersedes stale previews only for affected candidates.
6. No items or reminder schedules change before Confirm. Confirm, conflict, Discard and Undo preserve current contracts.
7. Transport retries, re-forwarded copies and a crash after ingest commit do not silently duplicate confirmed items or completed extraction results.
8. Reschedule and cancellation emails identify matches for review and preserve unrelated booking segments.
9. With Minder offline, delivery fails without a successful handoff or Cloudflare content buffer. After recovery, re-forwarding produces a review. An accepted job survives a server restart; extraction exhaustion and budget exhaustion appear in the app.
10. Each configured allowed address can forward an authenticated email from any original sender. Reject every other outer sender, unapproved aliases, ambiguous/multiple From mailboxes, and allowlisted From values with failed or missing trusted aligned authentication. Spoofed Authentication-Results headers do not grant access. Wrong recipients, invalid machine credentials/signatures, replayed ingestion requests, malformed MIME, oversize files, hostile HTML and prompt injection cannot authorize changes or expose data. Machine credentials grant no owner-route access.
11. Unsupported/password-protected documents show their extraction limits and retain originals. Missing content never appears as a completed parse.
12. Source downloads require owner authentication and never enter public caches. Deletion revokes them; backup restore retains consistent source links.
13. Test large candidate sets without the current five-proposal limit silently dropping dates. Use reviewed candidate groups and separate bounded proposal batches.
14. Run Go race tests/vet, Flutter tests/analyze/format, architecture/DOX checks and compiled-web chat/reminder/calendar scenarios. Add real MIME/ICS/PDF/image fixtures and temporary SQLite integration tests.
15. Verify provider capability, actual Cloudflare admission/failure behaviour, account costs and content-retention settings before enabling production routing. Phone review gates merging as in the existing project workflow.

## Review points and rollout

The complete specification includes direct delivery, environment-based admission, attachment limits, source retention/deletion, event/date model additions and individual candidate confirmation.
Actual sender addresses are sensitive deployment data and must not enter Git.
Implementation starts after written-spec review and an implementation plan; this document does not mark the feature as built or deployed.

After approval, write the implementation plan. Suggested dependency order: event/date and provenance contracts;
direct receiving and server-side persistence; document extraction; capture conversation and review; deployment and real forwarding checks.
Keep infrastructure changes separate from local code verification. Do not change live mail DNS during specification work.

## Verified platform references

Checked 2026-11-12. The configuration and HTTP contracts above are design choices. Account configuration and end-to-end delivery remain unverified.

- [Cloudflare subdomain routing](https://developers.cloudflare.com/email-service/configuration/subdomains/).
- [Email Worker handler](https://developers.cloudflare.com/email-service/api/route-emails/email-handler/).
- [Email analytics and 31-day retention](https://developers.cloudflare.com/email-service/observability/metrics-analytics/).
