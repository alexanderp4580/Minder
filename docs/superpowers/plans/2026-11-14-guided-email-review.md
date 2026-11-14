# Guided Email Review Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans (the project runs inline, without sub-agents). Steps use checkbox syntax.

**Goal:** An emailed set of appointments is reviewed one at a time with clear messages; failures are visible and never retried silently; no assistant-imposed limits.

**Architecture:** `domain` gets the new capture state, reasons and venue rule. `assistant` validates extractions into kept and dropped candidates. `capture` publishes candidates one at a time and fails without retrying. `command` advances the flow when an item is confirmed or discarded. The app changes its Maps rule and card states.

**Tech Stack:** Go (`toolchain/run go`), SQLite, Flutter (`toolchain/run flutter`), Python fixtures, Playwright.

**Spec:** [docs/superpowers/specs/2026-11-14-guided-email-review-design.md](../specs/2026-11-14-guided-email-review-design.md)

## Global Constraints

- One responsibility per file and function; no copy-pasted branches. The owner rejected sloppy code.
- Logs carry reason codes, counts and ids, never email content, quotes or addresses.
- Tests first; real temporary SQLite for store and capture tests.
- Keep the evidence validation, the lease and conflict rules, and the €5 monthly limit.
- English copy, short sentences. Fixed failure sentences, no model text shown as an error.
- Do not read the owner's real mailbox; use synthetic emails only.

## Review Focus

- A failed extraction never schedules another attempt; Retry works after any number of calls.
- Answer to the current question while another candidate is queued: the queued one stays queued and its question is not lost.
- One candidate with bad evidence among valid ones: valid ones are kept and the count of dropped ones is shown.
- "Microsoft Teams" or "Zoom" as the only location text is cleared; a real address with "Teams" in a street name is kept.
- Confirming the last candidate shows the summary and queues no push; the owner's answer never queues a push.
- A duration chip when the question is not about duration goes to the model, not applied as a duration.

---

### Task 1: Remove the limits

**Files:** `server/openrouter/client.go`, `client_test.go`; `server/assistant/capture.go`, `capture_test.go`; `server/capture/review.go`, `review_test.go`.

**Interfaces:** `maxOutputTokens = 100000`; the email context limit is `maxCaptureBytes = 400000`; `reserveCaptureCall` still counts `Calls` but never refuses; `Retry` no longer checks `Calls`.

- [ ] **Step 1: Failing tests:** `TestRequestsAllowOneHundredThousandOutputTokens` (payload `max_tokens` 100000); `TestAnEmailOfOneHundredKilobytesIsAccepted` and `TestAnEmailOverFourHundredKilobytesIsRefused`; `TestRetryWorksAfterManyCalls` (capture with `Calls: 50` and status failed retries).
- [ ] **Step 2:** `toolchain/run go go test ./openrouter ./assistant ./capture` — FAIL.
- [ ] **Step 3:** Implement; delete the call cap branch and its message.
- [ ] **Step 4:** PASS. Commit `Remove the extraction limits`.

### Task 2: Fail once, say why, push once

**Files:** `server/capture/worker.go`, `server/domain/capture.go`, `server/domain/errors.go`, `server/assistant/capture.go`, tests.

**Interfaces:**
- `type FailureReason string` constants `ReasonCutOff`, `ReasonEvidenceMissing`, `ReasonNoReview`, `ReasonUnreadable`, `ReasonLimit`, `ReasonOther`, each with `Sentence() string` (`I could not read this email: <fixed reason>. The original is kept.`).
- `type ModelError` gains `Reason FailureReason`; `decodeCaptureExtraction` and the truncation check set it.
- `fail(c, now, cause)` has no `permanent` parameter and no delays: status `failed`, `NextAt` 0, entry text from the reason, one `capture_failed` delivery queued.
- Log line `email extraction rejected` with `reason`, `attempt`, `candidates`, never content.

- [ ] **Step 1: Failing tests:** `TestAFailedExtractionIsNotRetriedAutomatically` (one `ProcessNext` fails; later `ProcessNext` calls with a moving clock do nothing); `TestTheFailureEntryNamesTheReason` (table over the reasons); `TestAFailureQueuesOneCaptureFailedPush`; `TestRejectedExtractionsLogReasonsNotContent`; `TestRetryRunsOnceMore`.
- [ ] **Step 2:** `toolchain/run go go test ./capture ./assistant ./domain` — FAIL.
- [ ] **Step 3:** Implement. Release-claim paths on cancelled contexts stay.
- [ ] **Step 4:** PASS. Commit `Fail an email once and say why`.

### Task 3: Keep the valid candidates

**Files:** `server/assistant/capture.go`, `server/domain/capture.go` (`Capture.Unverified int`, `json:"unverified,omitempty"`), `server/capture/worker.go`, tests.

**Interfaces:** `decodeCaptureExtraction(response, document) (CaptureExtraction, dropped int, error)`; an extraction with candidates but none valid returns `ReasonEvidenceMissing`; `CaptureExtraction.Dropped int` carries the count to the worker, which stores it in `Capture.Unverified` and adds the card line `I could not verify N appointments from this email.`

- [ ] **Step 1: Failing tests:** `TestOneBadQuoteKeepsTheValidCandidates`, `TestNoValidCandidateFailsWithTheEvidenceReason`, `TestTheDroppedCountReachesTheCapture`, plus the existing validation tests adapted.
- [ ] **Step 2–4:** run `toolchain/run go go test ./assistant ./capture` FAIL then PASS. Commit `Keep valid appointments when one cannot be verified`.

### Task 4: One appointment at a time

**Files:** `server/domain/capture.go` (state `queued`), `server/capture/worker.go` and a new `server/capture/flow.go`, `server/command/capture.go`, `server/capture/questions.go`, tests.

**Interfaces:** `func publishNext(tx port.Transaction, s *Service, c *domain.Capture) (announced bool, err error)` publishes the first candidate in `queued` state (preview, or its question) after an announcement entry `Appointment i of N: <title>`; `func (s *Service) opening(c domain.Capture) string` builds `I found N appointments in "<subject>". We will go through them one at a time.` (singular, items). `updateCaptureCandidate` calls the same publisher after a confirm or discard and posts `Done: X added, Y skipped.` when none remain. Extraction results arrive as `queued` except none; the first is published immediately.

- [ ] **Step 1: Failing tests:** `TestExtractionPublishesOnlyTheFirstCandidate`, `TestTheOpeningMessageCountsAppointments` (1, 3, items), `TestConfirmingPublishesTheNext`, `TestDiscardingPublishesTheNext`, `TestTheLastStepPostsADoneSummary`, `TestAQuestionIsOnlyPostedForTheCurrentCandidate`, `TestAnsweringTheCurrentQuestionKeepsQueuedQuestions`.
- [ ] **Step 2–4:** `toolchain/run go go test ./capture ./command ./domain` FAIL then PASS. Commit `Guide the owner through appointments one at a time`.

### Task 5: Notifications

**Files:** `server/capture/worker.go`, `server/domain/capture.go` (`CaptureDelivery.Kind`, `Count`), `server/store/capture.go`, store migration if a column is needed, tests.

**Interfaces:** push body `N appointments need your review` / `1 appointment needs your review` for the first publication; `Minder could not read an email` for a failure; none after an owner answer or confirm/discard.

- [ ] **Step 1: Failing tests:** `TestFirstPublicationPushesTheCount`, `TestAnAnswerQueuesNoPush`, `TestAFailurePushesOnce`, store test for the delivery fields.
- [ ] **Step 2–4:** `toolchain/run go go test ./capture ./store` FAIL then PASS. Commit `Say what each email push is about`.

### Task 6: Duration chips without the model

**Files:** new `server/capture/duration.go` and test; `server/capture/review.go`.

**Interfaces:** `func parseDuration(text string) (minutes int, ok bool)` accepts `30 minutes`, `45 minutes`, `1 hour`, `90 minutes`, `1 hour 30 minutes`, `2 hours` (case-insensitive, trimmed); `answerCandidate` applies a duration to a candidate whose question mentions how long or duration: sets `duration_minutes`, clears the question, publishes the preview, no model call; any other text goes to the model.

- [ ] **Step 1: Failing tests:** `TestParseDuration` (table incl. rejects `1 hour please`, `soon`), `TestADurationAnswerSkipsTheModel` (extractor call count unchanged), `TestADurationOnANonDurationQuestionGoesToTheModel`.
- [ ] **Step 2–4:** PASS. Commit `Apply duration answers without the model`.

### Task 7: Online meetings are not addresses

**Files:** `server/domain/venue.go` and test; `server/assistant/capture.go` (prompt and `normaliseExtraction`).

**Interfaces:** `func IsVirtualVenue(location string) bool` true for text made only of virtual words (teams, microsoft teams, zoom, google meet, webex, online, video call, phone call, telephone, skype); false for a street address. `normaliseExtraction` clears such `Location`. Prompt: online meetings leave `location` empty, put the join link in `website`, and add the note `- Online meeting (<service>)`.

- [ ] **Step 1: Failing tests:** `TestIsVirtualVenue` (table: `Teams`, `Microsoft Teams meeting`, `Zoom`, `Online`, `Google Meet`, true; `Teamsstraat 4, Utrecht`, `Zoomlaan 1`, `Kerkstraat 1`, `Meeting room Teams-hoek 2`... false), `TestNormaliseClearsAVirtualLocation`, `TestThePromptSaysOnlineMeetingsHaveNoLocation`.
- [ ] **Step 2–4:** PASS. Commit `Keep online meetings out of the address`.

### Task 8: App states and the Maps rule

**Files:** `app/lib/domain/links.dart` (`isVirtualVenue`), `app/lib/ui/calendar_item_details.dart`, `app/lib/domain/capture.dart` (`unverified`, `queued`), `app/lib/ui/capture_view.dart`, `app/pubspec.yaml` (0.10.0+29), tests, `app/lib/ui/AGENTS.md`, `app/lib/domain/AGENTS.md`.

**Interfaces:** `bool isVirtualVenue(String location)` with the same cases as the server table; the address block offers Maps only when false. `Capture.unverified`; candidate state `queued` renders as `Waiting` after the current one; the card shows `I could not verify N appointments from this email.`

- [ ] **Step 1: Failing tests:** `virtual_venue_test.dart` (shared cases), a calendar-details widget test (no Maps action for `Teams`, action for a street), a capture-view test (queued appointments, the unverified line, the failure text).
- [ ] **Step 2:** `toolchain/run flutter flutter test` for those files — FAIL.
- [ ] **Step 3–4:** Implement; PASS. Commit `Show email progress and hide Maps for online meetings`.

### Task 9: Fixtures, browser scenario, real model

**Files:** `toolchain/model_fixture.py`, `email-check.cjs`, `email-local-check`, `toolchain/AGENTS.md`.

- [ ] **Step 1:** The model fixture returns three candidates (one with a Teams location, two with duration questions) for a synthetic three-appointment email; the email browser scenario walks all three: opening message, appointment 1 with a duration chip, Skip on 2, Confirm 3, summary.
- [ ] **Step 2:** `toolchain/email-local-check` — PASS.
- [ ] **Step 3:** Real model on the same synthetic email through a local server (TokenCrypt keys): record what it returns; the Teams meeting has an empty location.

### Task 10: Documentation, APK and verification

- [ ] **Step 1:** DOX pass for `server/capture`, `server/assistant`, `server/domain`, `server/command`, `server/store` if touched, `server/openrouter`, `app/lib/domain`, `app/lib/ui`, `toolchain`; evidence `docs/evidence/guided-email-review.md`; `HANDOFF.md`; the earlier email design's retry paragraph updated.
- [ ] **Step 2:** `toolchain/check` exits 0; build web and the release APK (version 0.10.0+29), verify the signer, send to Telegram once.
- [ ] **Step 3:** Stop before merging or deploying; ask the owner.

---

## Revision 2 (owner review, 2026-11-14)

The owner rejected three hand-written rules and one misplaced field. Tasks, each test-first:

- **R1 Remove the chip parser.** Delete `capture/duration.go`, `domain.ParseDuration` and their tests; every answer goes through the model. The model fixture sets a duration from the answer.
- **R2 Remove the venue word list.** Delete `domain/venue.go`, the clearing in `normaliseExtraction`, the Dart `isVirtualVenue` and the `Where` row. Maps shows for any location. The prompt keeps saying online meetings have no address.
- [x] **R3 Add `details`.** `domain.Detail{Label, Value}`; `Item.Details`, `Proposal.Details`, `Proposal.ClearDetails`; validation limits; `setAttributes` and `itemFields`; tool schema property `details` and `clear_details`; prompt lines (capture: meeting link goes in `details`; chat: facts that fit no field); Dart `Item.details`, wire, details rows, preview row.
- [x] **R4 Agent-written text.** `CaptureExtraction.Summary` and `.Notification` (`json:"summary"`, `json:"notification"`); `extract_capture` schema properties; `Capture.Notification`; the worker posts `Summary` as the opening on the first extraction and stores `Notification`; `review.Opening` is deleted; `review.PushBody` returns the failure sentence, else the stored notification, else `An email needs your review.`
- **R5 Verify.** Fixtures, browser scenario, real model run, `toolchain/check`, DOX, evidence, APK 0.10.1+30, one Telegram upload. No merge or deploy.
