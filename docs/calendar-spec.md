# Calendar page

Status 2026-11-12. The owner asked for a calendar page. The page is read-only. Chat stays the place where you create and change items. Design rules: [foundation.md](foundation.md). Evidence: [evidence/m1.md](evidence/m1.md).

## Behaviour

- The chat header has a Calendar button, a 44 px square on the right. The Calendar page has a matching Chat button at the same place.
- The page shows one month as a grid. Weeks start on Monday. The grid always shows six weeks, so days of the next and last month show dimmed.
- Each day cell is 56 px high, so the weeks have room between them. Each day shows up to three small squares for its items. Today is underlined. The selected day has a red frame. Opening the page selects today.
- The arrows move one month. Today returns to the current month. Moving the month selects its first day.
- Under the grid, the agenda lists the items of the selected day, earliest first. Each entry shows the time, the title and small tags for duration, priority, location, list and repeat. A finished item is struck through.
- An item sits on its start day, else its deadline day. An item with neither does not show. An item with a start shows "09:00". An item with only a deadline shows "Due 17:00".
- An empty day says "Nothing planned."
- Without a connection, the page keeps its last items and shows the error with a Retry button.
- Times show in the device time zone.

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

- Creating or changing items, drag and drop, week and day views, search, colour, phone calendar sync. Old drafts in `mockups/calendar.html` are not maintained. The golden image `../app/test/goldens/calendar.png` shows the current design.
