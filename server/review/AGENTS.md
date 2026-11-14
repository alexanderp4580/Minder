# Purpose

The one-at-a-time review of an email's items.

# Ownership

Which candidate is published next and what the chat says about it: the `Appointment i of N` announcement, the closing summary, the push text and the question entries.

# Local Contracts

`Advance(tx, capture, preview)` publishes the first `queued` candidate when none is `pending` or `needs-answer`: its question (and the question entry) or its preview through the given `PreviewFunc`. It changes only the capture it is given and the chat entries it writes; the caller stores the capture. Superseded candidates are not counted. `Summary` words the closing count. `PushBody` returns the fixed failure sentence, else `Capture.Notification` unchanged, else `An email needs your review.`. Progress nouns are `appointment` when every live candidate is an event, otherwise `item`. Question helpers (`PostQuestions`, `CloseQuestions`, `HasOpenQuestions`) live here so `capture` and `command` share them.
Depends on `domain` and `port` only. It never applies a change.

# Work Guidance

Write failing behaviour tests before fixes. Keep messages short and English.

# Verification

From the root: `toolchain/run go go test -race ./review`.
Run `toolchain/architecture-check` for package boundaries.

# Child DOX Index

None.
