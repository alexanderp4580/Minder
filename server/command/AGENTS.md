# Purpose

Item and profile changes through transaction interfaces.

# Ownership

Preview batches, Confirm, Discard, Undo, conflict refresh, attribute application and preview fields.

# Local Contracts

Use `port.Transaction`; do not import storage implementations.
One invalid proposal rolls back its batch. Confirm checks versions and records audit/results atomically.
Pending changes do not expire. Undo refuses a changed current version. Preserve storage errors.

Event kind, end, zones and date-only attributes participate in previews, Confirm and Undo like existing item fields.

Capture-bound previews validate candidate revision on Confirm. Confirm/Discard update source state and provenance atomically. Source deletion requires a separate irreversible preview and redacts captured history before file cleanup.

Pending item previews with a start or deadline carry `reminder_choices_minutes` and the chosen `reminder_offsets_minutes`; `SetReminders` edits them while the change is pending and conflicts afterwards.

`delete_item` previews what a delete removes and changes nothing. Confirm calls `PurgeItem` and deletes each linked email source that no other item uses (redacted like `delete_capture`), then records an audit row marked `ItemDeleted` and a result with `removed_entry_ids`. The delete's own preview and result stay. Undo of a delete restores the item and its reminder schedule, not its cards or email source. A repeating item adds a `recurrence` row ("All occurrences removed"): a delete takes the whole series. A stale `expected_version` is a conflict.


Booking/contact fields participate in previews, Confirm, clear and Undo. Before editing older items, promote labelled note metadata to separate fields; clearing a field also removes its legacy note lines so it cannot reappear. JSON-row storage needs no schema migration.

# Work Guidance

Write failing behaviour tests before fixes. Preserve current wire and database contracts.

# Verification

From the root: `toolchain/run go go test -race ./command`.
Run `toolchain/architecture-check` for package boundaries.

# Child DOX Index

None.
