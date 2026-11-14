# Guided email review

Date: 2026-11-14. App 0.10.2+31. Spec: [guided email review design](../superpowers/specs/2026-11-14-guided-email-review-design.md). Plan: [guided email review plan](../superpowers/plans/2026-11-14-guided-email-review.md). Branch `guided-email-review`, deployed for phone testing and not merged.

## Why

The owner forwarded an email with several appointments. Extraction failed twice (the model's answer was cut off at 4,000 tokens, with about 3,200 spent on reasoning, and a correction attempt was rejected), and the server silently retried every 5 s, 30 s, 2 min and 10 min. A later retry worked but the review was hard to follow: "Teams" landed in the address field with an Open in Google Maps action, two duration questions shared one chat message and both vanished when one was answered, and the follow-up sent a second "An email needs your review" push.

## Current state

- One extraction attempt per revision. A failure shows one fixed sentence in the email entry and sends one push; the owner taps Retry. No automatic retries, no call cap.
- `max_tokens` is 100,000 (the model allows 128,000); one email's request may be 400,000 bytes.
- Valid candidates are kept when others fail evidence; the review says how many could not be verified. Each rejected attempt logs a reason code, never content.
- The review is one appointment at a time, driven by the server in chat: the model's opening message, an `Appointment i of N` announcement, the question or preview of the current candidate, Confirm or Discard (the skip), and a `Done` summary. Every answer goes through the model.
- Online meetings: the prompt tells the model to leave the address empty; no code judges a location.
- Additional facts use an ordered `details` list. Meeting links use the label Meeting link and open from previews, email details and calendar details. Details support replacement, clear and Undo.
- Pushes: the model's stored `notification` on the first publication (fallback `An email needs your review.`), `Minder could not read an email.` on failure, nothing for the owner's own answers. The server posts `summary` unchanged once and keeps the unverified count in a separate note.

## Earlier checks (before Revision 2)

| Check | Result |
|---|---|
| Tests first | Each rule above had a failing test before its code: limits, failure once and reasons, the reason log without content, partial candidates, queued publishing, confirm and discard advancing, push counts and silence after answers, every answer reaching the model and the card states |
| `toolchain/email-local-check` | Passed twice against compiled web, signed ingestion and real SQLite: the trip email (hotel, then the flight question), and a three-appointment email with a Teams meeting walked to a `Done: 2 added, 1 skipped.` summary; the Teams location was empty |
| Real model, 2026-11-14 | A synthetic three-appointment dental email through a local server: three candidates, the Teams follow-up had an empty address, `- Online meeting (Microsoft Teams)` in notes, the opening message, `Appointment 1 of 3` and the first question appeared; cost about €0.0015 |

## Revision 2 checks, 2026-11-14

- Details command tests failed before the domain fields existed, then passed: preview, Confirm, replacement, clear, Undo, trimming, blank pairs and size rejection. The provider request test verifies details and clear_details on chat and extraction tools.
- Flutter checks cover detail rows, missing-field compatibility and opening the meeting link. A preview tap test failed with a null launch target, then passed after the preview used LinkedText.
- Model-text tests failed before the extraction fields existed, then passed: unchanged opening and persisted push text, missing-summary silence, push fallback, no repeated opening after answers and a separate unverified note.
- `toolchain/email-local-check` exited 0 with model-written fixture openings, model-applied duration replies and meeting-link details persisted in real SQLite.
- A real OpenRouter call through signed ingestion on a local server returned three synthetic dental appointments in states pending/queued/queued. It wrote an opening and notification, put `https://teams.example/join/braces` in Meeting link, and left location and website empty for the Teams follow-up. Provider cost: $0.0017248. No real mailbox data.
- Inline branch review found that a provider failure during evidence correction lost already verified candidates. The regression test failed with no candidates, then passed after retaining the first verified result while the request context remains active.

- `toolchain/check` exited 0 on the revised source: 163 Flutter tests, Go race tests/vet, Flutter analysis/formatting, architecture/DOX, Worker and toolchain/hook checks. Release web build exited 0. Its optional Wasm dry-run reported existing unsupported packages, and the icon-font optimizer reported the absent Cupertino font. The JavaScript web build succeeded.

## APK 0.10.1 (earlier release)

Source `fbbe18d`. All five final browser scenarios exited 0: chat, reminder, calendar, place lookup and email.

Release arm64 APK: `.local/artifacts/Minder-0.10.1-arm64.apk` (ignored), 19,031,438 bytes. SHA-256 `e7225a914cc245a8c3db1016ddf515402c762f0bb4295dd57fb23c4f3bb55eeb`. `apksigner verify --print-certs` exited 0 for this APK and 0.10.0; both report signer `7bbc3aba5e417ffe87efa00c53c183c6490dc61905ce7231aaf99f87666d2572`. `aapt2 dump badging` reports `io.liveleds.minder`, version 0.10.1, Android version code 2030 (pubspec 30 plus the arm64 split offset); the earlier APK reports 29.

Telegram confirmed one upload: message 2029, file size 19,031,438. The APK needs the matching server changes to exercise the new email flow. The owner requested the matching server deployment for phone testing. Merge still awaits approval.

## Initial deployment for phone testing

The owner requested deployment on 2026-11-14 after receiving the APK. Pushed `guided-email-review` and pinned Coolify to `50b968228b470261dce76da1b47a205d2a256f45`. Deployment `fy2qz1rqsyklwp5uvutdbfaw` finished; the application reports `running:healthy`. Startup logs show schema 5, zero migrations, configured model/Firebase/Serper credentials and no warnings or errors in the first 80 lines. The server still logs revision=unknown; the completed Coolify deployment identifies the commit.

Rollback target: branch `main`, commit `679441b3bcf730e80e880167b25ba5787c27d70b`, captured from the healthy application before deployment. `main` remains unchanged. No production email was injected or replayed; the owner can test with a fresh multi-appointment email using APK 0.10.1.

## Phone feedback: Confirm and working indicator

The owner reported that the first Confirm appeared to do nothing, although the review advanced to item 2 of 4, and asked for an agent activity indicator. Read-only inspection of structured production state found three confirmed event candidates plus one confirmed questionnaire task. All four have saved items, confirmed pending changes and confirmed stored preview cards. Four is the item count, including the questionnaire; it is not a fourth appointment.

A synthetic client regression reproduced the stale button with the event stream silent: the Confirm HTTP response advanced the poll cursor past the earlier confirmed-preview event. The same race could skip the closed-question entry when an answer response arrived first. ChatController now keeps its history/event-stream watermark separate from the latest visible cursor. Only history and ordered stream events advance that watermark.

The Working spinner now stays above the composer, outside the scrolling message list, as an accessible live status. It covers email received/processing states as well as normal chat waiting. It clears when extraction finishes or fails; email completion does not clear unrelated chat waiting. Six focused tests passed after reproducing the missing preview/question updates, missing email indicator and off-screen indicator in long reviews. `toolchain/check` exited 0 with 169 Flutter tests, Go race/vet, analysis/format and repository checks. Release web built. The email browser scenario passed with the stream blocked for first Confirm and a held model answer; the Working screenshot is `.local/email-working.png`. All five browser scenarios exited 0: email, chat, reminder, calendar and place lookup. APK 0.10.2 built and passed signature/package/version verification.

Source `ee99d879c3c4b03d9e84ec4772ba884745173c0e`. APK `.local/artifacts/Minder-0.10.2-arm64.apk` is 19,031,438 bytes, SHA-256 `92a2360ce3276b904fe068dbfc334ce4e25cdc6704071fb2d1bfe4fdc368c738`, package `io.liveleds.minder`, version code 2031. Its signer matches 0.10.1. The owner reported no delivery after the initial upload. The initial local upload, authorized local resend and server-side transfer each returned HTTP 504. At the owner’s request, tried a text test message over POST and GET; both returned HTTP 502. getMe succeeded in 0.08 seconds. No message or file receipt exists; delivery remains unconfirmed. No further attempt is running.

Coolify deployment `jtptw3myyrdsxktwodwgeias` finished at this commit and the application reports running:healthy. Startup logs show schema 5, zero migrations, configured model/Firebase/Serper credentials and no warnings or errors in the first 80 lines. Runtime revision still reports unknown; the completed deployment identifies the commit. Rollback target: branch guided-email-review at `50b968228b470261dce76da1b47a205d2a256f45`. Server source and schema did not change. The branch remains unmerged; phone approval of 0.10.2 is pending. This patch changes only the client and local verification fixtures.

## Fresh-start reset

At the owner's request, stopped the production container and backed up SQLite and email files to the private server directory `/var/backups/minder/fresh-start-20261114T155318Z`. Verified the backup database before clearing data. After restart, item, chat/event, pending/audit, reminder and capture/provenance tables contain zero rows, the email source directory is empty, and the five built-in profiles exist. Preserved one push registration and 79 cost-accounting rows. SQLite integrity and foreign-key checks passed; the container's `/v1/health` returned `{"status":"ok"}`. No source or deployment change accompanied this reset.

## DOX

Updated the owning domain, command, assistant, provider, capture, review, client and toolchain contracts. Parent ownership and child indexes remain unchanged because this revision adds no package or layer. Review ran inline, as the owner requested; no independent reviewer.

## Limits

- Not tried on a phone. The 2026-11-14 email that failed was not replayed; its exact validation failure is unknown.
- A push is not sent when the first extraction finds nothing to review.
- A failure push is sent once per revision; Retry starts a new revision.
