# Purpose

Item and profile changes through transaction interfaces.

# Ownership

Preview batches, Confirm, Discard, Undo, conflict refresh, attribute application and preview fields.

# Local Contracts

Use `port.Transaction`; do not import storage implementations.
One invalid proposal rolls back its batch. Confirm checks versions and records audit/results atomically.
Pending changes do not expire. Undo refuses a changed current version. Preserve storage errors.

Confirming or discarding a candidate (email or pasted text) publishes the next queued one through `review.Advance` and, when the last is resolved, posts the `Done` summary, in the same transaction.

Event kind, end, zones and date-only attributes participate in previews, Confirm and Undo like existing item fields.

Capture-bound previews validate candidate revision on Confirm. Confirm/Discard update source state and provenance atomically. Source deletion requires a separate irreversible preview and redacts captured history before file cleanup.

Pending item previews with a start or deadline carry `reminder_choices_minutes` and the chosen `reminder_offsets_minutes`; `SetReminders` edits them while the change is pending and conflicts afterwards. Selecting offset reminders clears any explicit remind_at in both the pending item and proposal, including when all offsets are deselected. Dates and source deadlines stay unchanged.

`delete_item` previews what a delete removes and changes nothing. Confirm calls `PurgeItem` and deletes each linked email source that no other item uses (redacted like `delete_capture`), then records an audit row marked `ItemDeleted` and a result with `removed_entry_ids`. The delete's own preview and result stay. Undo of a delete restores the item and its reminder schedule, not its cards or email source. A repeating item adds a `recurrence` row ("All occurrences removed"): a delete takes the whole series. A stale `expected_version` is a conflict.


Detail lists participate in preview, Confirm, replace, clear and Undo. The preview carries `details` as `Label: value` lines.

Booking/contact fields participate in previews, Confirm, clear and Undo. Before editing older items, promote labelled note metadata to separate fields; clearing a field also removes its legacy note lines so it cannot reappear. JSON-row storage needs no schema migration.

`CommitAgentTurn(turn, outcome)` is the only place an agent outcome is stored. In one transaction it posts an offer or the model's text, applies the review patch and previews the proposals, records the cost on the first entry, and ends the turn as `done` with its entry IDs. Committing a finished turn returns its entries and writes nothing; committing a failed turn conflicts. One invalid proposal or patch rolls the whole turn back. It never confirms an item. An arrival turn with no review closes the capture as reviewed and posts the model's text.
A review patch targets a source: an email's capture, or pasted text (a chat source), whose capture and source record are created on first publication. The patch's expected revision must equal the capture's (0 for pasted text with no review). Candidates without an ID are new and queue; candidates with an ID update that candidate in place (it must exist, not be confirmed or discarded, and quote its current revision), withdraw its preview, keep its old evidence when the patch sends none and return it to the queue at the new revision. Any update bumps the review revision; a first publication or additions do not. Omitted candidates stay as they are. Updating the open question closes its question entry. The first publication stores the model's notification and queues one push; later patches push nothing. Stale revisions, unknown candidates and terminal candidates fail with a conflict and change nothing.
`FailAgentTurn(turn, cause)` ends a turn as `failed` in one transaction with one visible report: an arrival fails its source through `review.Fail` (unless the source was deleted or moved on); any other turn posts an `error` entry under the turn's ID with the safe `domain.Describe` text. Failing a finished turn does nothing.

`AddTask(title)` and `MoveTask(id, version, status)` serve the board. Each runs in one transaction, writes an audit row with actor `owner` and writes no preview or chat entry. `AddTask` creates an open task with no date. `MoveTask` accepts open, in progress and done, needs the current version (`ErrConflict` otherwise), refuses an item that is not on the board (`This task is on the calendar.`), sets or clears `completed_at`, and returns the item unchanged for a move to its current status. The item write runs the normal reminder projection, so a done task closes its reminder. Logs: `board item added`, `board item moved` (`item_id`, `from`, `to`, `version`); never titles. `update_item` and `create_item` apply `Proposal.Status` (`open` or `in_progress`; done and cancelled keep their own actions); leaving done clears `completed_at`.

# Work Guidance

Write failing behaviour tests before fixes. Preserve current wire and database contracts.

# Verification

From the root: `toolchain/run go go test -race ./command`.
Run `toolchain/architecture-check` for package boundaries.

# Child DOX Index

None.
