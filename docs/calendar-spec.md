# Calendar page

Status 2026-11-13. The owner asked for a calendar page. The page shows items and can delete them. Chat stays the place where you create and change items. Design rules: [foundation.md](foundation.md). Evidence: [evidence/m1.md](evidence/m1.md).

## Behaviour

- Chat and Calendar use the same title row, padding and fading rule. The header has no buttons. A shared bottom bar opens the other screen (see [foundation.md](foundation.md)).
- The page shows one month as a grid. Weeks start on Monday. The grid always shows six weeks, so days of the next and last month show dimmed.
- Day targets are 56 px high, with 8 px gaps between weeks and an inset 44 px selection frame. Each day shows up to three small squares for its items. Today is underlined. The selected day has a red frame. Opening the page selects today.
- The month and year have separate lines. Grouped arrows on the right move one month. Today sits beside the agenda date and returns to the current month. Moving the month selects its first day.
- Under the grid, the agenda lists the items of the selected day, earliest first. Each entry shows the time, the title and small tags for duration, priority, location, list and repeat. A finished item is struck through. Each card has a Delete control (a 44 px square at the top right). It opens a dialog with the server preview of what the delete removes, with Delete and Keep it. Leaving the dialog any other way keeps the item and discards the preview. A repeating item says that all occurrences go. Delete removes the item and its data and cannot be undone from the calendar; Undo is on the chat result card.
- An item sits on its start day, else its deadline day. An item with neither does not show. An item with a start shows "09:00". An item with only a deadline shows "Due 17:00".
- An empty day says "Nothing planned."
- Without a connection, the page keeps its last items and shows the error with a Retry button.
- Agenda times show in the device time zone. Expanded start/end rows use source wall clocks and timezones when supplied.
- The authored Calendar orbital artwork sits behind the page. Motion and accessibility rules live in the foundation.
- The page scrolls as one column. Calendar content is at most 520 px wide; the shared header is at most 760 px. Use 24 px content gutters at 360 px and above.
- Short screens and enlarged text can scroll the agenda into view. Day rows grow to keep dates and markers visible.
- Agenda cards have 20 px padding and an 18 px title. Shared spacing rules live in the brand guide and foundation.

- Disclosure rows use a neutral white label, down/up chevron and square interaction states. The 44 px target aligns with the card text under a Line divider. Source-dialog actions use a right chevron. The approved content-control rules live in `../brand/index.html` and the foundation.
- Tap a card or its 44 px Details button to expand it in place. Hide details collapses it. Available rows show Starts, Ends, Due, Duration, Type (Appointment for event), Address, Phone, Email, Website, Booking reference, Notes and Reminders. Two different nonempty start/end timezones use Departs and Arrives. A timezone without a timestamp adds no row.
- The address appears as a wrapping white underlined block with an external arrow and “Opens Google Maps”. It opens a Google Maps search. Phone, Email, Website and Booking reference have separate rows. Only remaining instructions appear in Notes. Labelled metadata in older notes is separated for display without a server write; conflicting values and calendar UID/sequence Ref lines stay in Notes. Notes show plain bullets capped at 600 characters, with underlined phone numbers, email addresses and HTTPS links. Dates and booking references stay plain text. Links open only after a tap, through the dialer, email app or browser; a failed launch shows a message. Only HTTPS, tel and mailto targets open.

## Data

- `GET /v1/items?from=<time>&to=<time>` returns the items placed in `[from, to)`. The range must be at most 62 days. The page asks for the six-week grid.
- The page loads when it opens and when you move the month. It does not update while it is open, because you cannot change items on it.

## Item fields behind it

- `start_at`: when the item begins. `due_at`: the deadline. The start must not be after the deadline.
- `duration_minutes`: 1 to 10,080. `priority`: 1 (highest) to 5. Zero means not set.
- A profile reminder counts from the deadline, else from the start.
- A repeating item needs a start or a deadline. Completing it moves the start, the deadline and the reminder by the same distance. Across a daylight-saving change, the start of an item that also has a deadline can move one hour against the local clock.
- The assistant sets `start_at` for appointments and meetings, and `due_at` only for deadlines.

## Not in this page

- Creating or changing items, drag and drop, week and day views, search, colour, phone calendar sync. The approved layout is in `mockups/calendar.html` and `mockups/calendar.png`. The golden image `../app/test/goldens/calendar.png` shows the current design.
