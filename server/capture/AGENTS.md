# Purpose

Email capture application service.

# Ownership

capture package.

# Local Contracts

Use persistence/blob/model interfaces. Only validated command previews may change items. Preserve originals before acknowledging ingestion; no infrastructure imports.

Review is one appointment at a time. After extraction every new candidate is `queued`; the first extraction posts the model's `Summary` unchanged when present and stores its `Notification`, then `review.Advance` publishes the first queued candidate (its question or its preview) and announces `Appointment i of N: <title>` when there is more than one. Confirm and Discard (in `command`) publish the next and end with `Done: X added, Y skipped.`. A regenerated candidate keeps its place in the order. Only the current candidate can have an open question; open questions post as one `question` entry (`q-<capture>-<revision>`) after the email card, and answering or dismissing turns it into a short `minder` note. A targeted answer supersedes only its candidate. Every answer goes through the model. Dismiss closes an open review at its current revision: it discards non-terminal candidates, fails their pending previews, keeps the source and changes no items. Capture contexts and answers stay separate from normal chat. Preserve independent and terminal candidates. Publish validated previews, coverage and job completion in one transaction.
An extraction runs once per revision. Any failure ends the attempt as `failed` with one fixed sentence (`domain.FailureReason.Sentence`) in the email entry and one push; nothing retries automatically, and Retry is the owner's button (it starts a new revision, has no call cap and keeps originals). The count of candidates dropped for bad evidence is kept in `Capture.Unverified`.
The unverified count posts as its own server note.
Pushes: the first publication uses the stored model notification (`review.PushBody`), or `An email needs your review.` when missing; a failure pushes `Minder could not read an email.`; the owner's own answers, confirms and discards push nothing. A push is delivered only while its revision is current.

# Work Guidance

Use synthetic fixtures; preserve original content and privacy.

# Verification

`toolchain/run go go test -race ./capture`.

# Child DOX Index

None.
