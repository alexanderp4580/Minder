# Delete and cancel items Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans (project rule: inline, no sub-agents) to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Chat and calendar can delete an item with its associated data, chat can cancel an item, and a failed send leaves no Retry block in chat.

**Architecture:** A new `delete_item` action reuses the preview, Confirm and Undo flow. A new store method purges the item's associated rows in the confirm transaction. The calendar calls a new preview endpoint and confirms through the existing change routes. The chat controller restores unsent text instead of keeping an error block.

**Tech Stack:** Go, SQLite, Flutter, OpenRouter tool schemas.

**Spec:** [delete items design](../specs/2026-11-13-delete-items-design.md)

## Global Constraints

- Test first. `toolchain/check` must pass. DOX updated for each touched folder.
- No migration. Logs carry IDs and counts, not titles or text.
- Do not push or deploy. Commit without co-author trailers.
- Existing cancel behaviour of `cancel_item` (status `cancelled`) is unchanged.

## Review Focus

- Deleting an item that shares an email source with another item keeps the source. Pinned in Task 2.
- A stale `expected_version` returns a conflict and deletes nothing. Pinned in Task 2.
- The delete's own preview and result entries survive the purge. Pinned in Task 2.
- Undo of a delete restores the item and its reminder schedule. Pinned in Task 2.
- A deleted item no longer appears in the agent's history or context. Pinned in Task 3.
- Text typed during an offline send is not lost. Pinned in Task 5.

---

### Task 1: Chat tools for cancel and delete

**Files:** Modify `server/openrouter/client.go` (tools list, line ~156), `server/assistant/assistant.go` (switch at ~223), `server/assistant/prompt.go` (rules); Test `server/assistant/assistant_test.go`, `server/openrouter/client_test.go` if it lists tools.

**Interfaces:** Produces tools `propose_cancel`, `propose_delete` (required `item_id`, `expected_version`); the assistant maps them to `Proposal.Action` `cancel_item` and `delete_item`.

- [ ] Failing tests: a model call to `propose_cancel` yields a proposal with `Action == "cancel_item"`; `propose_delete` yields `delete_item`; the system prompt contains the rule text for done, cancelled and deleted.
- [ ] Implement the two tools, the mapping and one prompt rule: use `propose_complete` for done, `propose_cancel` for cancel or no longer needed, `propose_delete` only for delete, remove or created by mistake.
- [ ] `toolchain/run go go test ./assistant/... ./openrouter/...` passes. Commit.

### Task 2: delete_item through preview, Confirm and Undo

**Files:** Modify `server/port/port.go` (Items), `server/store/store.go` (+ `server/store/purge.go`), `server/command/command.go` (+ `server/command/delete.go`), `server/domain/model.go` (`Change.RemovedEntryIDs`); Test `server/store/purge_test.go`, `server/command/delete_test.go`.

**Interfaces:**
- Consumes: `Transaction.DeleteItem`, `SourcesForItem`, `RedactCapture`, `PutCapture`.
- Produces: `Items.PurgeItem(id, keepPending string) (Purged, error)` with `Purged{EntryIDs []string; OrphanSources []string}`; `Change.RemovedEntryIDs []string json:"removed_entry_ids,omitempty"`; action `delete_item` in `Service.Preview` and `Confirm`.

- [ ] Failing store tests on a real temporary SQLite file: after `PurgeItem` the item, schedule, occurrences, push deliveries, earlier pending rows and audit rows, chat entries and events of that item are gone; the entry of `keepPending` stays; a source linked to another item is not in `OrphanSources`.
- [ ] Failing command tests: preview of `delete_item` returns a pending card (fields `item`, reminders, source); a stale `ExpectedVersion` returns `ErrConflict` and changes nothing; Confirm removes the item, tombstones orphan sources, keeps its own preview and result entries, sets `RemovedEntryIDs`, and returns a result with `UndoID`; Undo restores the item and its reminder schedule.
- [ ] Implement `PurgeItem` in store (JSON path queries over `pending_change.body`, `audit.body`, `reminder_occurrence`), the `delete_item` branches, and the audit for Undo.
- [ ] Full `toolchain/run go go test -race ./...`, vet and gofmt pass. Commit.

### Task 3: Agent no longer sees deleted data

**Files:** Test `server/assistant/assistant_test.go`.

- [ ] Failing test: after Confirm of a delete, `History(0, 20)` holds none of the item's earlier previews or results and `contextItems` excludes the item.
- [ ] Fix what the test shows and nothing else. Commit.

### Task 4: Calendar preview endpoint

**Files:** Modify `server/api/api.go` (route `POST /v1/items/{id}/delete`), new handler in `server/api/items_delete.go`; Test `server/api/api_test.go`.

**Interfaces:** Request `{"expected_version": int}`. Response the preview `Entry` (HTTP 200); 404 unknown item; 409 stale version.

- [ ] Failing tests for 200, 404 and 409; the preview then confirms through `/v1/changes/{id}/confirm`.
- [ ] Implement via `command.Service.Preview(Proposal{Action: "delete_item", ItemID, ExpectedVersion})`. Commit.

### Task 5: Chat controller: no Retry block, unsent text restored

**Files:** Modify `app/lib/application/chat_controller.dart`, `app/lib/ui/chat_screen.dart`; Test `app/test/chat_test.dart`.

**Interfaces:** Produces `ChatController.takeUnsentText()` (returns the text of a failed send once) and `hasUnsentText`. The screen puts that text in front of anything typed since. `error` holds a message for non-transport failures only, with no Retry button, and clears on the next send or action. The sign-in block is unchanged.

- [ ] Failing tests: a transport-failed send sets no `error`, shows no `Retry`, and the composer holds the text; the Retry block does not appear after recovery; a non-transport failure shows its message and no Retry; text typed while offline is kept.
- [ ] Implement; remove `_lastText`/`_retryAction` guard logic. Update the existing "a failed send keeps its inline Retry" test to the new rule. Commit.

### Task 6: Client handles removed entries; Delete in calendar

**Files:** Modify `app/lib/domain/entry.dart`, `app/lib/data/wire.dart`, `app/lib/domain/item_repository.dart`, `app/lib/data/chat_api.dart`, `app/lib/application/calendar_controller.dart`, `app/lib/application/chat_controller.dart` (`_merge`), `app/lib/ui/calendar_screen.dart`, `app/lib/ui/format.dart`; Test `app/test/chat_test.dart`, `app/test/calendar_test.dart`.

**Interfaces:** Consumes the Task 2 and 4 wire shapes. Produces `Change.removedEntryIds`, `ItemRepository.previewDelete(Item) → Future<Entry>`, `confirmDelete(String)`, `discardDelete(String)`, and a 44 px Delete button on each agenda card.

- [ ] Failing tests: a result carrying `removed_entry_ids` drops those entries from the chat; tapping Delete shows the preview dialog; Confirm removes the item from the month; Discard keeps it; the button is 44 px high.
- [ ] Implement; label `delete_item` as `DELETE ITEM`. Commit.

### Task 7: End to end, docs, version, delivery

- [ ] Browser check on compiled web with the fixture: create an item in chat, delete it by chat, confirm the chat cards vanish and Undo restores it; delete from the calendar; force an offline send and see the text return.
- [ ] Update DOX (`server/command`, `server/store`, `server/api`, `server/assistant`, `app/lib/ui`, `app/lib/application`, `app/test`, `docs/screens.md`, brand guide if the calendar card changes), `HANDOFF.md`, evidence `docs/evidence/delete-items.md`. Bump to 0.7.0+23.
- [ ] `toolchain/check` exit 0; build the arm64 APK; verify signature; send once through TokenCrypt. Do not deploy.
