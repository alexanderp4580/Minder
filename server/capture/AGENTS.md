# Purpose

Email ingestion, source preparation, source reading and review delivery.

# Ownership

capture package.

# Local Contracts

Use persistence/blob interfaces; no infrastructure imports. Preserve originals before acknowledging ingestion. Ingestion also stores the source reference (`domain.SourceRef`, kind `email`).
An optional `Diagnostics` callback receives the complete accepted original MIME bytes (base64 in JSON), envelope and source ID after durable ingestion, even when no agent reads the source. Successful preparation records the parsed document with the arrival turn's trace ID. Capture applies no diagnostic truncation; existing ingestion and parser limits still apply. Main supplies the private recorder, which owns storage, expiry and write failures. Normal logs exclude source content.
The worker only prepares. `ProcessNext` claims a received email, parses the original, stores attachments, records subject, coverage and the source label, marks the capture `Prepared` and queues one `source_arrival` agent turn (`domain.ArrivalTurn`) in the same transaction. The capture stays `processing` until the agent publishes a review. Preparation calls no model and decides nothing about content.
A preparation failure ends the attempt as `failed` through `review.Fail` (one fixed sentence in the email entry, one new source-linked chat message, one push). Nothing retries automatically; Retry is the owner's button (new revision, originals kept, prepared again, new arrival turn). On startup, restore a missing failure report for each stored failed source once, without queuing another push. Shutdown releases a claimed source without consuming a retry.
`ReadSource(id)` returns a source's parts, attachments and coverage, and `FindSources(query)` lists matches (the store limits and refuses blank or oversized queries). Email originals parse through the existing parser; owner text loads as one text part with ID `text`. Every read checks deletion. A deleted or unknown source gives `ErrSourceUnavailable` or `ErrNotFound`; a kept source that cannot be read gives `ErrSourceUnreadable`. Reads change no capture, turn, push, entry or item.
`Dismiss` closes an open review at its current revision: it discards non-terminal candidates, fails their pending previews, keeps the source and changes no items. `Acknowledge` accepts unreadable parts so the review can finish. Answers to questions are agent turns (see `conversation`), not capture operations.
Pushes: the first publication pushes the stored model notification (`review.PushBody`), or `An email needs your review.`; a failure pushes `Minder could not read an email.`; the owner's own answers, confirms and discards push nothing. A push is delivered only while its revision is current.

# Work Guidance

Use synthetic fixtures; preserve original content and privacy.

# Verification

`toolchain/run go go test -race ./capture`.

# Child DOX Index

None.
