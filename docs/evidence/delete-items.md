# Delete and cancel items, quieter connection errors

Date: 2026-11-13. Branch `delete-items` (from local `main`). App version 0.7.0+23. The finished-items lookup is server-only; the 0.7.0 APK needs no update for it. Not merged, not pushed, not deployed.
Spec: [delete items design](../superpowers/specs/2026-11-13-delete-items-design.md). Plan: [delete items plan](../superpowers/plans/2026-11-13-delete-items.md).

## Behaviour

- Chat has `propose_cancel` (status `cancelled`) and `propose_delete`. Before this change "cancel" fell through to `propose_complete` and set `done`, because the chat tool list had no cancel tool.
- Delete removes the item, its reminder schedule and push records, its earlier chat cards, pending changes and audit rows, and each linked email source that no other item uses. The delete's own preview and result stay. Undo restores the item and its reminders only.
- Chat can look up finished items. When the agent calls `find_finished_items`, the server adds one model call with the matching done and cancelled items (at most 50), so you can ask about them and cancel or delete one. Ordinary requests still cost one call.
- A repeating item's delete preview says all occurrences go.
- The calendar has a 44 px Delete control on each card. It shows the server preview in a dialog. Back or any other exit discards the preview.
- A failed send puts its text back in the composer. A dropped connection never adds a block to the chat. Other failures show a message without Retry. An error entry offers Retry only while it is the newest entry.

## Checks

| Check | Result |
|---|---|
| Red tests before each change | Each failed first: tools, purge, delete flow, route, composer restore, removed cards, calendar Delete, Back discard, newest-only Retry, repeat warning |
| Mutation check | Breaking `deleteEntries` failed `TestConfirmedDeleteRemovesTheItemItsCardsAndAnUnusedSource` and `TestDeletedItemLeavesTheAgentsContext`; restoring it passed |
| `toolchain/check` | Exit 0: 95 Flutter tests, Go race tests, vet, format, Worker tests, hooks, architecture, DOX, orbital exports |
| Finished-items tests | Failed first (unknown tool, oversized second request); pass with the lookup |
| Calendar golden | Updated after viewing the new card with its Delete control |

Artifact: `.local/artifacts/Minder-0.7.0-arm64.apk` (ignored by Git), 19,224,066 bytes, SHA-256 `3a2c35ba322adad6b9730d5c5d8287c47f667685ff845106bfbb5d867faeed2c`. aapt2 reports `io.liveleds.minder`, version 0.7.0, code 23. The signing certificate matches earlier APKs. Telegram confirmed one upload (message 1995).

## Limits

- The browser scenarios did not run. The owner interrupted the run and asked why it was needed, so it was skipped. The API test and the widget tests cover the same paths. Nobody has used delete on a phone.
- A finished-items lookup costs a second model call. The lookup matches the title text only, so a vague request may need a name.
- Nobody has tried the lookup against the real model; the tests use a scripted model.
- The delete's audit row keeps an item snapshot so Undo works. Nothing purges it later.
- Backups made before a delete keep their copies until they expire.
- A server-side error entry still shows Retry while it is the newest entry.
