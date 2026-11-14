# One agent for chat and email

Status: draft for owner review. The owner chose one agent and the same flow for forwarded emails and emails pasted into chat. This document specifies that change; implementation has not started. It supersedes the proposed patch that would merely add email access to the existing separate chat flow.

## Outcome

Minder understands incoming material, asks questions, proposes items and answers follow-ups through one agent. The owner can forward an email or paste it into chat and receive the same review behavior. Source access continues during review and after confirmation, until the owner deletes the source. Existing confirmed items remain unchanged by this migration.

One agent means one instruction set, conversation context builder and model/tool execution loop. It can take multiple model turns to read material or use tools. There is no separate email extraction persona, hidden specialist or extraction-only model request. Background jobs prepare files and schedule turns; they do not independently interpret appointments or answer the owner.

## Inputs and source preparation

Forwarded MIME messages retain their authenticated ingestion, private original storage, deduplication and bounded attachment conversion. That processing produces source documents, with metadata, part IDs, text, images and coverage. It then schedules an incoming-source turn in the conversation.

Pasted text enters as an ordinary owner message. The agent decides whether the owner wants an explanation, a summary, item proposals or something else. No email keyword detector selects a different prompt or workflow. When the agent proposes items from pasted material, their provenance references the retained chat source. The paste is not rejected by the current 4,000-character chat restriction if it fits the existing 400,000-byte source budget; input validation, transport and client constraints must agree.

Source preparation does not call another model to decide facts. External email text, attachment contents and quoted source passages are untrusted data and cannot authorize mutations. The owner's surrounding request remains the instruction. No remote links are fetched merely because they occur in a source.

## Agent context and tools

The shared agent receives owner messages, source arrivals, review state and previous tool outcomes as distinct records. An arrival is source data, not a fabricated owner command. Incoming actionable appointments retain the established behavior of starting a review; ordinary pasted material follows the owner's request.

The agent can find retained sources, read their parts and inspect the current review. The current review's source reference remains available even when its original card leaves the recent-history window. Older sources can be found by metadata or query. Attachment images are supplied to the same agent when needed. Missing, deleted or unreadable parts produce explicit tool results; the agent must explain those limits instead of claiming it checked them.

All requests use the shared instructions and tool family. Remove the separate PurposeEmail/extract_capture execution path after its callers move. Lookup approval and existing item/profile/list proposals keep their present authorization rules. Reading retained sources does not need a separate approval, trigger extraction, send another review push or modify an item.

## Reviews and follow-up messages

The agent can publish a group of source-linked candidate proposals and questions using the existing review presentation. The server validates and persists those tool results; it never decides facts or parses a natural-language answer into a duration.

Both a quick-reply tap and a typed message enter the shared agent turn with the applicable source, review revision and candidate reference. These references indicate conversational context, not a command to overwrite the candidate. The agent decides whether the owner answered a question, asked about evidence, corrected a value or changed the subject.

The agent updates a candidate by its stable ID and expected revision. A targeted update preserves other queued, confirmed and discarded candidates. A follow-up answer does not require returning the entire extraction batch, eliminating the current omission failure. If a message ambiguously refers to several sources or candidates, the agent asks which one.

Confirm and Discard remain explicit actions. Conversational answers never silently confirm an item. Existing previews, source links, review progress and the questionnaire task remain supported.

## Facts and missing details

The shared instructions distinguish source facts, owner-provided values and assumptions. An end time or duration requires support from source material or an owner answer. A typical appointment length, pending record or earlier assistant statement is not evidence. With a start and no supported length, ask for the length; do not populate a guessed end time.

Candidate evidence refers to the supporting source parts. Literal quote validation remains a mechanical check, not a claim that every field has been semantically verified. The agent must inspect the relevant source when challenged, identify unsupported values plainly and propose their removal or correction. Avoid claims that code can establish semantic support through date parsers or word lists. Acceptance includes a real-model synthetic source with a start time and no end or duration.

## Scheduling, failures and retention

Accepted source arrivals and owner turns are durable and serialized through the same agent execution mechanism. Receipt can return before processing finishes. Incoming material cannot overwrite an active owner turn or be silently dropped when chat is busy. A process restart resumes pending work without duplicating saved proposals or notifications.

Working remains visible while a turn is active. Every failed accepted turn receives a new visible failure message. Source-arrival review/failure pushes retain their delivery guarantees; normal conversational answers do not create unsolicited review pushes. Retry remains explicit. Usage accounting records all model calls.

Source deletion makes it unavailable to tools and future context, including stale review references. Retention and provenance apply to pasted sources as well as forwarded sources. Existing explicit item deletion and email-source deletion semantics must stay consistent. No production reset or replay of the owner's email is part of implementation verification.

## Acceptance

- Forward a synthetic three-appointment email with a questionnaire, and paste equivalent text with the same owner request. Both reach the same instructions and agent/tool loop and can produce the same four-item review. There is no extraction-only model call.
- Ask what to bring, where an appointment happens, and what a questionnaire requires during review and after confirmation. Answers use the retained source and attachments.
- Supply a start without a duration. No unsupported end or one-hour duration appears. Ask where a proposed value came from; the agent checks the original rather than defending the pending record.
- Answer the second candidate's duration question by quick reply and by typing. Only that candidate changes; all four candidates and the first confirmed item remain intact.
- Ask a follow-up while a review question is open. The message is answered rather than forced into the question-answer extraction path.
- Find an older source after its card leaves recent chat history. Deleting it makes subsequent reads return unavailable.
- Submit a pasted email longer than 4,000 characters within the 400,000-byte source budget. It reaches the shared agent intact. Oversized input fails visibly before acceptance.
- Exercise concurrent arrival, model failure and restart. Work is serialized, failures appear in chat, and no duplicate proposals or pushes result.
- Keep existing chat, reminders, calendar, lookups, confirmation and usage-accounting checks passing. Use synthetic sources in tests and logs; never commit private source content.

## Delivery

Implementation follows a reviewed plan after this design is approved. Work stays inline on guided-email-review. Verify server, client and browser changes, build an APK if the client changes, deploy the tested feature commit and provide the LAN download. Keep main unmerged until phone approval. Record production status separately from local checks.
