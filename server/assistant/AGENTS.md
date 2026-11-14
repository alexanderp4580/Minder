# Purpose

Model orchestration through model, proposal and persistence interfaces.

# Ownership

Model contract, system prompt, context selection, proposal batches and request cost accounting.

# Local Contracts

No HTTP client or provider SDK belongs here. Model implementations live in adapters.
Keep at most 200 open items in context. Never apply proposed changes. Reserve cost before calling the model.
Settle known provider costs; retain reservations for unknown costs. Put a reply cost on its first entry only.

Extraction asks as a thoughtful assistant would: any number of questions, no fixed field checklist, up to three quick replies each (`options`), notes as a short `- ` list, address in `location`, `reminder_offsets_minutes` from the allowed choices (invalid values are dropped). Extraction puts phone, email, website and booking_reference in separate proposal fields, preserving exact values and leading zeros. Notes hold remaining instructions and calendar UID/sequence Ref lines. Missing optional metadata does not trigger a fixed question checklist. An appointment with a known start and no end/duration asks how long it takes, with 30 minutes and plausible alternatives. Empty duration options receive defaults. Displayed titles, instructions, questions and options are English; addresses, proper names and identifiers remain exact. Evidence remains literal original-source text for validation. Email extraction uses capture-only context, source evidence and structured candidates/questions. Share usage accounting with normal chat; external text cannot authorize writes. Permit one correction per extraction batch, asked for when the first answer failed or lost candidates; charge both calls and keep exact source-evidence validation. Candidates whose evidence checks out are kept and the rest are dropped and counted (`CaptureExtraction.Dropped`); the extraction fails only when none are valid, with a `domain.FailureReason` on the error. Each rejected attempt logs `email extraction rejected` (reason code, attempt, never content). There is no per-source call cap; one email's request may be up to 400,000 bytes. Online meetings (Teams, Zoom, Meet, phone calls) have no address: the prompt tells the model to leave `location` empty; the code never judges a location.

Chat tools are `propose_create`, `propose_update`, `propose_complete`, `propose_cancel` (status `cancelled`) and `propose_delete` (removes the item and its data), plus `propose_profile`, `list_items` (open items, shown to the owner) and `find_finished_items`. The prompt keeps done, cancelled and deleted apart. Deleted items and their cards leave the history and context. The agent sees the last 20 chat entries and at most 200 open items. `find_finished_items` (`query`, `list`) reaches done and cancelled items: the server makes one more model call with the matches as untrusted data (newest first, at most 50; the grown request stays under 120,000 bytes). That call answers about them or proposes a cancel or delete. The model must call the tool alone; any other calls in that reply are dropped, and a second lookup is an error. Both calls are billed and the reply carries both costs.

Lookups are optional (`WithLookups`). With a finder set, `Message.Purpose` on the first message is `PurposeOffer`, the prompt gains `lookupRule` and the model may call `offer_lookup` (`kind`: maps, places or search; `query`) alone, once per owner message. The server stores only the fixed-format offer entry (`domain.OfferText`) and drops other calls; an unknown kind or blank query is an error. The offer is the owner's approval of that exact search. Only when the owner's text is `Look it up` and the newest entry before it is an offer (error entries and earlier Look it up messages are skipped, so Retry works) does the server run that kind and query through the `Finder` and add the hits to the request as untrusted data (`lookupMessage`; at most three hits). The model then gets its normal call: it proposes, asks the owner, or offers one more search, which is a new card and a new tap. A failed or empty search says so. Any other text, a stale offer or a missing finder runs no search. There is no search budget and no loop; the prompt never mentions a limit and tells the model that `offer_lookup` is how it searches. Logs carry `kind` and `query_len`, never the query.

# Work Guidance

Write failing behaviour tests before fixes. Preserve current wire and database contracts.

# Verification

From the root: `toolchain/run go go test -race ./assistant`.
Run `toolchain/architecture-check` for package boundaries.

# Child DOX Index

None.
