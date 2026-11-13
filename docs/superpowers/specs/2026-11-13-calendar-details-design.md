# Calendar item details and links

Status: approved by the owner on 2026-11-13 ("Fix all"). Inline execution, no sub-agents.

## Intent

An appointment card in the calendar says "Departs" and "Arrives", and it hides every detail the server holds. The owner wants the details, a Google Maps link for the address, tappable phone numbers and email addresses, and similar.

## Cause

- The card adds "Departs" and "Arrives" chips to any timed item. Items have only three kinds (task, event, reminder), so an appointment is labelled like a flight.
- The client `Item` drops `notes`, `kind` and the reminder offsets. Only the address survives, as a plain chip.

## Goals

- Appointments say "Starts" and "Ends". "Departs" and "Arrives" appear only when the start and end timezones differ.
- A tap on a card expands it to show Starts or Departs, Ends or Arrives, Duration, Type, Address, Notes and Reminders.
- The address links to Google Maps. Phone numbers, email addresses and web links in the address and notes are tappable.

## Non-goals

- No new screen, no server change, no editing from the calendar, no contacts integration.
- No Markdown, no hidden link text, no automatic calling or sending.

## Approaches

1. **Expand the card in place.** Chosen: the owner decided that no screen beyond Chat and Calendar is needed, and the chat preview card already shows these rows.
2. A detail screen. Rejected: a third screen.
3. Always show every detail. Rejected: long notes make the agenda hard to scan.

## Design

- **Link finder.** A pure function in the domain layer splits text into plain, phone, email and web spans.
  - A web link starts with `https://` or `www.`. Trailing `.,;:!?)` stays outside it.
  - An email is `name@domain.tld`.
  - A phone number has 7 to 15 digits, may use `+`, spaces, dashes, dots and brackets, and is not glued to letters or digits. It must start with `+`, `00`, `0` or `(`, or have separators and at least 10 digits. ISO dates such as `2026-11-20` and references such as `ABC123456789` are not numbers.
  - Targets are `tel:`, `mailto:` and `https:`. Nothing else opens.
- **Maps.** `https://www.google.com/maps/search/?api=1&query=<address>`.
- **Opening.** A `LinkLauncher` interface in the domain layer, implemented with `url_launcher` in the data layer, passed to the calendar screen like the source repository. Android needs `<queries>` entries for `tel`, `mailto` and `https`.
- **Item model.** `Item` gains `notes`, `kind` and `reminderOffsets`.
- **Card.** Tapping a card expands or collapses it. A "Details" / "Hide details" text button (44 px) appears when the item has details. The collapsed card keeps today's time, title and tags, without the Departs and Arrives chips unless the zones differ. The expanded card adds the rows above. Notes show as plain lines, at most 600 characters, with links.
- **Security.** Notes come from email and are untrusted. The visible text is the link target. Tapping opens an external app; nothing opens on its own.

## Constraints

- Test first. `toolchain/check` passes. DOX updated.
- Calendar goldens change only after the new render is viewed.
- No deployment is needed; the APK carries the change.

## Left to the builder

Row spacing, the wording of the reminder row, and the link colour (Text colour with an underline).

## Known limits

- A phone number the finder rejects stays plain text. Nobody has tested the links on a phone.
- Google Maps opens in the Maps app or a browser, as the phone decides.
