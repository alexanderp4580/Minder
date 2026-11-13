# Delete and cancel items, quieter connection errors

Status: approved by the owner on 2026-11-13 ("Yes") as one slice. Inline execution, no sub-agents.

## Intent

The owner added an event by accident, and it confuses the agent when they add something similar. Chat has no cancel tool, so "cancel it" falls back to `propose_complete` and the item becomes `done`. The chat list also keeps a "Check your connection. Retry" block after the connection recovers, and the list pins it into view.

## Goals

- Chat can cancel (status `cancelled`) and delete an item. Calendar can delete an item.
- Delete removes the item and its associated data. Cancel keeps the item.
- A failed send leaves no block in the chat list. The text returns to the composer.

## Non-goals and anti-goals

- No bulk delete. No new screens. No change to the email capture flow beyond deleting an orphaned source.
- A delete that leaves chat cards, reminders or an unused email source behind has failed.
- After a connection drop recovers, no error block stays in the chat list.

## Approaches

1. **Add `propose_cancel` and `propose_delete` tools** and a `delete_item` action through the existing preview, Confirm and Undo flow. Chosen: it follows the flow of the other changes and keeps deletes confirmed.
2. Let `propose_update` set `status` to cancelled and add a hard-delete endpoint outside chat. Rejected: two paths for one change, and no preview in chat.
3. Prompt-only fix ("use update for cancel"). Rejected: the model has no tool that reaches `cancel_item`.

## Design

- **Tools and prompt.** `propose_cancel` maps to `cancel_item`, `propose_delete` to `delete_item`. Both need `item_id` and `expected_version`. One prompt rule separates done, cancelled and deleted.
- **Preview.** A `delete_item` preview shows the title, `Deleted`, the count of reminders, and whether an email source goes with it. Confirm applies it. Discard keeps the item.
- **What delete removes.** In one transaction: the item; its reminder schedule, occurrences and push deliveries; the chat entries of its earlier previews, results and reminders (and their events); its earlier pending changes and audit rows; its `item_source` links; and each linked email source that no other item still uses (redacted like `delete_capture`). The delete's own preview and result entries stay.
- **Undo.** The result card has Undo. Undo restores the item and its reminders. It does not restore chat cards or an email source; the card says so.
- **Client sync.** The result's `Change.removed_entry_ids` lists the chat entries removed. The app drops them from memory.
- **Calendar.** Each agenda card has a 44 px Delete button. It calls `POST /v1/items/{id}/delete` with `expected_version`, shows the preview in a dialog, then Confirm (`/v1/changes/{id}/confirm`) or Discard.
- **Retry.** A transport failure on send restores the text to the composer and sets no `error`. Other send failures show a message without a Retry button. It clears on the next send or action; a sync would clear it within seconds, before you could read it. The sign-in block is unchanged. Persisted `error` entries keep their Retry.

## Constraints

- Keep old migration files unchanged; this slice needs no migration.
- Logs carry IDs and counts, not titles or text.
- Phone approval gates the merge. No deployment without the owner's word.

## Left to the builder

Names of SQL helpers, dialog layout details, wording of the delete preview beyond the fields above.

## Known limits

- The delete's audit row keeps an item snapshot so Undo works. It is not purged later.
- Backups made before the delete keep their copies until they expire.
