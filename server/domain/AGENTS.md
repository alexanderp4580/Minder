# Purpose

Pure Go values and business rules.

# Ownership

Items, built-in profile defaults, proposals, chat values, recurrence, reminder policy, identifiers and operation errors.

# Local Contracts

Use the standard library only. Keep HTTP, SQL, SDKs and adapter imports out.
`Complete` advances recurrence; `SnoozeAt` uses the item zone. UTC storage times retain their item timezone.
`CaptureQuestion` (`id`, `text`, `options`) also decodes a bare string, the form older stored captures use. `ProjectReminder` derives a schedule projection; persistence writes it atomically with item/profile changes. An item may hold `reminder_offsets_minutes` (10, 30, 60, 120, 1440, 2880, 4320, 10080; `NormalizeOffsets` sorts longest first). `NextReminder` picks the earliest offset after a time, drops past offsets and anchors date-only items at 09:00 local.

Items distinguish event end from deadline, and date-only ranges from instants. Event ranges are half-open; Overlaps uses the viewing timezone. Travel retains start/end zones. Date-only recurrence advances local dates without creating a timed reminder.

Capture values carry source metadata, attachment manifests, coverage, candidate revisions and follow-up answers without provider or storage code.

The lookup offer (`Offer{Kind, Query}`, kinds maps, places, search) is a minder entry with fixed text, for example `Search Google Maps for "<query>"?` (`OfferText`, `ParseOffer`; an unknown kind or blank query is refused); `IsLookupReply` accepts `Look it up`. `Entry` marshals a computed `options` (`Skip`, `Look it up`) for a well-formed offer only; nothing stores it.

`Change.RemovedEntryIDs` (`removed_entry_ids`) lists chat entries a delete removed. `Audit.ItemDeleted` marks the audit row that lets Undo restore a deleted item.


Items carry `details`, an ordered label/value list. Proposals replace it with `details` or empty it with `clear_details`. Validation trims pairs, drops blank labels or values and accepts at most 10 pairs, labels up to 60 characters and values up to 2,048. `DetailsText` renders literal `Label: value` lines for previews.

Items and proposals carry optional `phone`, `email`, `website` and `booking_reference` strings, with clear flags on proposals. Phone/email/reference accept at most 200 characters; website accepts 2048. References retain leading zeros. `NormalizeBookingDetails` promotes clearly labelled legacy note lines, preserves conflicts and instructions, and leaves calendar UID/sequence Ref lines alone.

`FailureReason` (`ReasonCutOff`, `ReasonEvidence`, `ReasonNoReview`, `ReasonUnreadable`, `ReasonUnparsed`, `ReasonTimeout`, `ReasonLimit`, `ReasonOther`) gives each email failure one fixed sentence; `ReasonOf` classifies an error and `ModelError.Reason` carries it. `Capture.Unverified` counts dropped candidates; candidate state `queued` means extracted but not yet published. `AsksDuration` recognises duration questions. `CaptureExtraction.Summary` and `.Notification` carry the model's opening and push text; `Capture.Notification` stores the latter.

# Work Guidance

Write failing behaviour tests before fixes. Preserve current wire and database contracts.

# Verification

From the root: `toolchain/run go go test -race ./domain`.
Run `toolchain/architecture-check` for package boundaries.

# Child DOX Index

None.
