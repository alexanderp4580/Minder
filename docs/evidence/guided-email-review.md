# Guided email review

Date: 2026-11-14. App 0.10.0+29. Spec: [guided email review design](../superpowers/specs/2026-11-14-guided-email-review-design.md). Plan: [guided email review plan](../superpowers/plans/2026-11-14-guided-email-review.md). Branch `guided-email-review`, not merged or deployed.

## Why

The owner forwarded an email with several appointments. Extraction failed twice (the model's answer was cut off at 4,000 tokens, with about 3,200 spent on reasoning, and a correction attempt was rejected), and the server silently retried every 5 s, 30 s, 2 min and 10 min. A later retry worked but the review was hard to follow: "Teams" landed in the address field with an Open in Google Maps action, two duration questions shared one chat message and both vanished when one was answered, and the follow-up sent a second "An email needs your review" push.

## Current state

- One extraction attempt per revision. A failure shows one fixed sentence in the email entry and sends one push; the owner taps Retry. No automatic retries, no call cap.
- `max_tokens` is 100,000 (the model allows 128,000); one email's request may be 400,000 bytes.
- Valid candidates are kept when others fail evidence; the review says how many could not be verified. Each rejected attempt logs a reason code, never content.
- The review is one appointment at a time, driven by the server in chat: an opening message with the count, an `Appointment i of N` announcement, the question or preview of the current candidate, Confirm or Discard (the skip), and a `Done` summary. Duration chips are applied without the model.
- Online meetings keep no address (prompt and `IsVirtualVenue`); Maps appears only for a real address.
- Pushes: `N appointments need your review` on the first publication, `Minder could not read an email.` on failure, nothing for the owner's own answers.

## Checks

| Check | Result |
|---|---|
| Tests first | Each rule above had a failing test before its code: limits, failure once and reasons, the reason log without content, partial candidates, queued publishing, confirm and discard advancing, push counts and silence after answers, duration parsing and local application, virtual venues (shared table in Go and Dart), the Maps row and the card states |
| `toolchain/email-local-check` | Passed twice against compiled web, signed ingestion and real SQLite: the trip email (hotel, then the flight question), and a three-appointment email with a Teams meeting walked to a `Done: 2 added, 1 skipped.` summary; the Teams location was empty |
| Real model, 2026-11-14 | A synthetic three-appointment dental email through a local server: three candidates, the Teams follow-up had an empty address, `- Online meeting (Microsoft Teams)` in notes, the opening message, `Appointment 1 of 3` and the first question appeared; cost about €0.0015 |

## APK

`.local/artifacts/Minder-0.10.0-arm64.apk` (ignored by Git), 19,283,210 bytes, SHA-256 `302dac8cc7b22ce02d0c58050131d617208b176bb5976a8a62c98487411b7015`. aapt2 reports `io.liveleds.minder`, version 0.10.0, code 29. The signer SHA-256 `7bbc3aba…d2572` matches earlier APKs. Telegram confirmed one upload (message 2028). `toolchain/check` exited 0 on the final tree. The app needs the matching server deploy: an older server publishes every appointment at once, which this app still shows.

## Limits

- Not tried on a phone. The 2026-11-14 email that failed was not replayed; its exact validation failure is unknown.
- A push is not sent when the first extraction finds nothing to review.
- A failure push is sent once per revision; Retry starts a new revision.
