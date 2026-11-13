# Email review refinement Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans (project rule: inline, no sub-agents) to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make forwarded email a short conversation, make the item card readable, let an item hold several reminders and move connection status into a header icon.

**Architecture:** Items gain `reminder_offsets_minutes` (stored in the existing JSON item body, so no schema migration). The single per-item `reminder_schedule` row stays: it points at the next unfired offset and advances after each firing. Pending item previews carry reminder choices and can be edited before Confirm. Extraction returns structured questions with quick replies, posted as their own chat entry. Flutter formats fields for reading and shows chips and a header status icon.

**Tech Stack:** Go, SQLite, Flutter, OpenRouter tool schemas, Cloudflare Worker (unchanged).

**Spec:** [email review refinement](../specs/2026-11-13-email-review-refinement-design.md)

## Global Constraints

- Reminder offsets in minutes: 10, 30, 60, 120, 1440, 2880, 4320, 10080. At most 8 per item, sorted longest first, no duplicates.
- Display kind `event` as *Appointment*, `task` as *Task*, `reminder` as *Reminder*. Stored values do not change.
- Address row uses stored field `location`. Starts and Ends carry the timezone inline, for example `Fri 20 Nov 14:30 (Europe/Amsterdam)`. Duration reads `45 minutes` or `1 hour 30 minutes`.
- The title shows once per card. Notes render as a Markdown bullet list, plain text only: no links, images or HTML.
- Questions are the agent's judgment: no hard-coded field checklist, up to three quick replies each, typing always works.
- Connection icon appears only after 2 seconds of failure; no banner.
- Project rules: test first, DOX updated for each touched folder, `toolchain/check` green, no secrets in Git, never push to `main`, send each APK once.
- Existing stored items, captures (string questions) and single reminders load and behave as before.

## Review Focus

- An appointment starting sooner than the largest offset: offsets already past are dropped, never fired late. Pinned in Task 1.
- A date-only item (hotel nights) with offsets: anchor at 09:00 local on the start date. Pinned in Task 1.
- Toggling reminders on a preview that was already confirmed or discarded returns a conflict and the card refreshes. Pinned in Task 2.
- Captures saved before this change with `questions` as plain strings still load. Pinned in Task 3.
- Hostile notes text (Markdown links, HTML, 5,000 characters) shows as inert plain text, capped. Pinned in Task 5.

---

### Task 1: Reminder offsets in the domain, store and scheduler

**Files:**
- Modify: `server/domain/model.go` (Item, Proposal), `server/domain/recurrence.go`, `server/domain/reminder.go`, `server/store/reminders.go`, `server/reminder/service.go`, `server/command/*.go` (apply proposal fields)
- Test: `server/domain/reminder_policy_test.go`, `server/store/reminders_test.go`, `server/reminder/transaction_test.go`
- Docs: `server/domain/AGENTS.md`, `server/reminder/AGENTS.md`, `server/store/AGENTS.md`

**Interfaces:**
- Produces: `Item.ReminderOffsets []int` and `Proposal.ReminderOffsets []int` (JSON `reminder_offsets_minutes`), `Proposal.ClearReminderOffsets bool` (`clear_reminder_offsets`), `var ReminderOffsetChoices = []int{10,30,60,120,1440,2880,4320,10080}`, `func NormalizeOffsets(in []int) ([]int, error)` (sorted longest first, deduplicated, only allowed values, max 8), `func NextReminder(i Item, p *Profile, after time.Time) (time.Time, bool)`.
- Rule: `NextReminder` returns the earliest offset time strictly after `after`, anchored on `Anchor(i)`; a date-only item anchors at 09:00 in `i.Timezone`. With no offsets it falls back to `EffectiveReminder`.

- [ ] **Step 1: Write failing tests.** `TestNormalizeOffsets` (order, duplicates, `7` rejected, nine values rejected). `TestNextReminderDropsPastOffsets` (start in 3 hours, offsets 1 day, 1 hour, 10 minutes: first result is 1 hour before; after that firing the next is 10 minutes before; then none). `TestNextReminderAnchorsDateOnlyAt0900Local`. `TestScheduleAdvancesThroughOffsets` in `reminder/transaction_test.go`: after `ScheduleDue` fires the first offset, the schedule is active at the next offset and produces a second occurrence; after the last, nag rules apply as before. `TestSingleReminderUnchanged` pins the existing `remind_at` behaviour.
- [ ] **Step 2: Run** `toolchain/run go go test ./domain ./store ./reminder`. Expected: FAIL (undefined names).
- [ ] **Step 3: Implement** the fields and functions above. `ReminderProjection` gains `Offsets string` (comma-joined, so it stays comparable); `applyReminder` writes the schedule at `NextReminder(item, profile, now)`; `Service.ScheduleDue` advances to `NextReminder(item, profile, now)` after each firing before applying nag. `PutItem` validates offsets through `NormalizeOffsets`.
- [ ] **Step 4: Run** the same command. Expected: PASS; then `toolchain/run go go test -race ./...` Expected: PASS.
- [ ] **Step 5: Commit** `git add server && git commit -m "Hold several reminders per item"`.

### Task 2: Preview carries reminder choices and can change them

**Files:**
- Modify: `server/domain/model.go` (Change), `server/command/attributes.go`, `server/command/*.go`, `server/api/api.go`, `server/api/changes.go` (or the file that holds `confirm`/`discard`)
- Test: `server/command/*_test.go`, `server/api/api_test.go`

**Interfaces:**
- Consumes: Task 1 fields.
- Produces: `Change.ReminderOffsets []int` and `Change.ReminderChoices []int` (JSON `reminder_offsets_minutes`, `reminder_choices_minutes`), set on pending item previews that have a start or deadline; a preview field named `reminders` with text such as `1 day, 1 hour`; `func (s *Service) SetReminders(changeID string, offsets []int) (domain.Entry, error)`; route `POST /v1/changes/{id}/reminders` with body `{"offsets":[...]}` returning the updated preview entry.

- [ ] **Step 1: Failing tests.** `TestPreviewListsRemindersAndChoices`; `TestSetRemindersUpdatesThePendingProposalAndPreview`; `TestSetRemindersOnConfirmedChangeConflicts` (409); `TestSetRemindersRejectsUnknownOffsets` (400); API test for auth, unknown fields and the 409.
- [ ] **Step 2: Run** `toolchain/run go go test ./command ./api`. Expected: FAIL.
- [ ] **Step 3: Implement.** `itemFields` adds the `reminders` field; `SetReminders` rewrites the pending proposal and preview in one transaction; confirming applies the offsets through the Task 1 path.
- [ ] **Step 4: Run** the same command. Expected: PASS.
- [ ] **Step 5: Commit** `git commit -m "Let previews change their reminders before Confirm"`.

### Task 3: Extraction asks as an assistant, with structured questions

**Files:**
- Modify: `server/domain/capture.go`, `server/assistant/capture.go` (`captureInstructions`), `server/openrouter/client.go` (extract_capture schema), `server/capture/worker.go`
- Test: `server/domain/capture_test.go` (new), `server/assistant/capture_test.go`, `server/openrouter/*_test.go`

**Interfaces:**
- Produces: `type CaptureQuestion struct{ ID string; Text string; Options []string }` (`id`, `text`, `options,omitempty`) with `UnmarshalJSON` that also accepts a bare string; `Capture.Questions []CaptureQuestion`; `CaptureCandidate.Options []string` (`options,omitempty`) for its `Question`; `CaptureExtraction.Questions []CaptureQuestion`. The extract_capture schema allows `reminder_offsets_minutes` on proposals, `options` (max 3 strings) on questions and candidates.
- Prompt rules (replace the fixed location rule): ask any question a thoughtful assistant would, as many as needed, only when the answer would change the item and the email does not give it; never use a checklist; offer up to three short quick replies; put the address in `location`; write `notes` as a short `- ` list of relevant facts only (booking reference, what to bring, cancellation, contact), never restating title, time or address; choose `reminder_offsets_minutes` from the allowed list (appointments default 1 day and 1 hour); put the timezone in `start_timezone`/`end_timezone`.

- [ ] **Step 1: Failing tests.** `TestOldStoredCaptureWithStringQuestionsLoads`; `TestExtractionSchemaAllowsOptionsAndOffsets`; `TestExtractionKeepsQuestionOptionsAtThree`; `TestPromptDoesNotListRequiredFields` (the instructions contain no "needs a location" rule and do mention judgment).
- [ ] **Step 2: Run** `toolchain/run go go test ./domain ./assistant ./openrouter ./capture`. Expected: FAIL.
- [ ] **Step 3: Implement** the type, schema and prompt; the worker stores structured questions; invalid offsets from the model are dropped, not fatal.
- [ ] **Step 4: Run** the same command plus `toolchain/run go go test -race ./...`. Expected: PASS.
- [ ] **Step 5: Commit** `git commit -m "Let extraction ask structured questions with quick replies"`.

### Task 4: Questions arrive as their own chat message

**Files:**
- Modify: `server/capture/worker.go`, `server/capture/review.go`, `server/domain/model.go` (entry kind), `server/store/*.go` (entry kind allow-list if any)
- Test: `server/capture/review_test.go`, `server/api/captures_test.go`

**Interfaces:**
- Consumes: Task 3 types.
- Produces: entry kind `question` (`Entry.Capture` set) posted once per revision that has open questions; ID `q-<captureID>-<revision>`; the capture card entry itself never carries questions. An answer or Do nothing marks earlier question entries answered by bumping the capture snapshot they carry.

- [ ] **Step 1: Failing tests.** `TestQuestionsPostAsSeparateEntryAfterReceived` (order: capture card, then question entry); `TestNoQuestionEntryWhenNothingMissing`; `TestAnswerClosesThePreviousQuestionEntry`; `TestSeveralQuestionsShareOneEntry`; `TestQuickReplyTextIsAnAnswer` (answer with text equal to an option works like typing).
- [ ] **Step 2: Run** `toolchain/run go go test ./capture ./api`. Expected: FAIL.
- [ ] **Step 3: Implement.** Push notification fires for the first question entry or the first final preview, not for the received card (adjust `QueueCaptureNotifications` call sites).
- [ ] **Step 4: Run** the same command. Expected: PASS.
- [ ] **Step 5: Commit** `git commit -m "Post email questions as their own message"`.

### Task 5: Readable item card in Flutter

**Files:**
- Modify: `app/lib/ui/format.dart`, `app/lib/ui/entry_view.dart`, `app/lib/domain/change.dart` (or where `Change` lives), `app/lib/data/wire.dart`, `app/lib/domain/chat_repository.dart`, `app/lib/data/chat_api.dart`, `app/lib/application/chat_controller.dart`
- Create: `app/lib/ui/reminder_chips.dart`
- Test: `app/test/chat_test.dart`, `app/test/ui_spec_test.dart`, new `app/test/preview_card_test.dart`

**Interfaces:**
- Consumes: `Change.ReminderOffsets/ReminderChoices` (Task 2).
- Produces: `String durationText(int minutes)`; `String kindLabel(String kind)`; `ReminderChips({required List<int> choices, required List<int> selected, required ValueChanged<List<int>> onChanged, bool enabled})`; `ChatRepository.setChangeReminders(String changeId, List<int> offsets) -> Future<Entry>`; `ChatController.setReminders(String changeId, List<int> offsets)`.
- Rules: the header shows the title once and the `title` row is hidden when the item is new; `start_timezone`/`end_timezone` rows are hidden and their value is appended to Starts/Ends in parentheses; `duration_minutes` row label `Duration`; `location` label `Address`; `kind` value through `kindLabel`; `notes` renders as bullets, plain text, capped at 600 characters shown; chips 44 px high, wrap, tap toggles, "Clear" removes all.

- [ ] **Step 1: Failing tests.** `previews show the title once`, `starts and ends carry the timezone`, `duration reads with its unit` (`45 minutes`, `1 hour 30 minutes`), `address and appointment labels`, `notes render as an inert bullet list` (a markdown link and `<b>` stay literal; 5,000 characters are capped), `reminder chips toggle and send the new offsets`, `chips are disabled once confirmed`.
- [ ] **Step 2: Run** `toolchain/run flutter flutter test test/preview_card_test.dart`. Expected: FAIL.
- [ ] **Step 3: Implement** the formatters, the card and chips; a conflict from `setChangeReminders` refreshes the entry.
- [ ] **Step 4: Run** `toolchain/run flutter flutter test` and `flutter analyze`. Expected: all pass, no issues.
- [ ] **Step 5: Commit** `git commit -m "Show readable item previews with reminder choices"`.

### Task 6: Email card summary and question message in Flutter

**Files:**
- Modify: `app/lib/domain/capture.dart`, `app/lib/data/wire.dart`, `app/lib/ui/capture_view.dart`, `app/lib/ui/entry_view.dart`
- Create: `app/lib/ui/question_view.dart`
- Test: `app/test/capture_api_test.dart`, new `app/test/question_view_test.dart`

**Interfaces:**
- Consumes: Task 3 and Task 4 wire shapes.
- Produces: `CaptureQuestion` in the domain (`id`, `text`, `options`), `QuestionView({required Capture capture, required ChatController chat})` for entries of kind `question`. The email card lists what Minder found (title, when, address per candidate) with the source files and Do nothing; it never shows questions.

- [ ] **Step 1: Failing tests.** `email card lists found items without questions`, `question entry shows each question with its quick replies`, `tapping a quick reply sends it as the answer with the current revision`, `typing answers several questions at once`, `answered questions show as closed`.
- [ ] **Step 2: Run** `toolchain/run flutter flutter test test/question_view_test.dart`. Expected: FAIL.
- [ ] **Step 3: Implement** the widgets; Do nothing stays on every open review.
- [ ] **Step 4: Run** the full Flutter suite and analyze. Expected: pass.
- [ ] **Step 5: Commit** `git commit -m "Split the email card from its questions"`.

### Task 7: Connection status icon

**Files:**
- Modify: `app/lib/application/chat_controller.dart`, `app/lib/ui/screen_header.dart`, `app/lib/ui/chat_screen.dart`
- Test: `app/test/chat_test.dart`, `app/test/bottom_nav_test.dart`

**Interfaces:**
- Produces: `ChatController.showOffline` (true after 2 seconds of continuous failure, false on the next success), `ScreenHeader({required String title, Widget? trailing})`, a 44 px cloud-off icon button with semantics "Offline. Tap to retry" that calls `sync()`.
- Rule: transport failures no longer set `error`; a failed send still shows its inline Retry entry. Sign-in errors keep their block.

- [ ] **Step 1: Failing tests.** `a one second drop shows nothing`, `a three second drop shows the header icon`, `recovery hides the icon`, `no connection banner or Retry block appears for transport failures`, `a failed send keeps its inline Retry`.
- [ ] **Step 2: Run** `toolchain/run flutter flutter test test/chat_test.dart`. Expected: FAIL.
- [ ] **Step 3: Implement** with a grace `Timer` using an injectable clock/duration for tests.
- [ ] **Step 4: Run** the full suite, analyze and format. Expected: pass; goldens unchanged unless the header intentionally changes (review any diff).
- [ ] **Step 5: Commit** `git commit -m "Show connection status as a header icon"`.

### Task 8: End to end, docs, version and delivery

**Files:**
- Modify: `toolchain/email-local-check`, `toolchain/email-check.cjs`, `toolchain/model_fixture.py`, `app/pubspec.yaml` (to `0.6.0+20`), `HANDOFF.md`, `docs/evidence/email-capture.md`, `docs/logging.md`, `docs/superpowers/AGENTS.md`, affected folder `AGENTS.md` files and the spec status line.

- [ ] **Step 1: Extend the browser scenario**: synthetic appointment email → received card with found summary → question message with a quick reply → final card (title once, Appointment, timezone inline, `45 minutes`, Address, bullet notes, chips with agent choices) → toggle a chip → Confirm → check stored reminders and the calendar day.
- [ ] **Step 2: Run** `toolchain/check`, release web build, then the chat, reminder, calendar and email scenarios in sequence. Expected: every command exits 0.
- [ ] **Step 3: Update DOX and evidence.** Record commands, exit codes and observed results; state what remains unverified (phone, real model output).
- [ ] **Step 4: Commit, push `notification-body-tap`, pin the Coolify commit, deploy, confirm health.** Back up first per the runbook; migrations are additive.
- [ ] **Step 5: Build the arm64 release APK and send it once to Telegram.** Expected: Telegram `ok:true`.

## Execution handoff note

Rulings and deferrals belong in the ledger. Known deferrals: re-registering a stale push token on app resume, and relaxing the fixed location rule in normal chat (`server/assistant/prompt.go`), which this plan leaves unchanged.
