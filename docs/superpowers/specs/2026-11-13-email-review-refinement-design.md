# Email review refinement

Status: draft, awaiting owner approval. Date: 2026-11-13. Builds on [email capture](2026-11-12-email-capture-design.md).

## Intent

A forwarded email should feel like a short conversation, not a form. The owner saw the first real run on a phone and listed what felt wrong. The fixes make the review card readable, make events carry the right fields, and let one item hold several reminders.

## Goals

1. **Natural flow.** Three messages, in order:
   1. *Email received.* The email card appears at once. When extraction finishes, the same card shows what Minder found: each item's title, when, and where, plus the source file. It asks nothing.
   2. *One question, only if needed.* A separate chat message from Minder, short and specific, for example "Is the location important? If so, where is it?" The agent supplies zero to three quick-reply buttons, such as "Skip" or "Use the address in the email". Typing always works. Answering sends one reply and continues.
   3. *Final item.* When nothing is missing, Minder posts one complete item card for approval (Confirm, Discard, Do nothing).
2. **Item card.**
   - The title shows once.
   - Type reads *Appointment* where it now reads *Event*.
   - Starts and Ends are one row each, with the timezone in the value, for example "Fri 20 Nov, 14:30 (Europe/Amsterdam)". No separate timezone rows.
   - Duration is a row whose value carries the unit: "45 minutes", "1 hour 30 minutes".
   - Address is its own row, filled from the email. The label is *Address*; the stored field stays `location`.
   - Notes are a short Markdown list of relevant facts only: booking reference, what to bring, cancellation terms, contact. No restated title, date or address. No reminders.
3. **Reminders as choices, not notes.** The card shows chips. The agent preselects the sensible ones and the owner switches any on or off, or clears all. Offsets: 10 minutes, 30 minutes, 1 hour, 2 hours, 1 day, 2 days, 3 days, 1 week. An item can hold several reminders at once.
4. **Kind rename.** The stored kind `event` is shown as *Appointment*. Whether the stored value changes is the builder's call; existing items must keep working.
5. **Connection status.** Remove the "Connection lost / Retry" block that flashes in the chat. Show a small status icon at the top right of the header: hidden when connected, a clear icon when offline or retrying, tappable to retry. Short drops of about two seconds show nothing.

## Non-goals and anti-goals

- No new item kinds such as flight or hotel; those still use `event`, shown as Appointment, until the owner asks.
- No auto-confirm. Email text never writes items on its own.
- No visual redesign: keep the Orbital look, header rule and 44 px targets.
- Anti-goal: a question the email already answers. The agent must not ask for a location that is in the email.

## Constraints

- Project rules: DOX updated, test-first, `toolchain/check` green, inline work, no secrets, no merge to `main`.
- Reminders today are one `remind_at` plus `nag_minutes` per item, and one `reminder_schedule` row per item. Multi-reminder needs a new representation (for example `reminder_offsets_minutes` on the item and one schedule row per offset). Existing single reminders must migrate unchanged. Recurrence, snooze, Done and Undo keep working.
- Reminders anchor on the start, or on the due time for tasks. Date-only items use a fixed local time of the builder's choosing, recorded as a call.
- Production data stays intact: additive migration only, with the existing backup step.

## Decided by the owner (2026-11-13)

- Email received, then a separate question message, then a full item for approval.
- The first card carries the extracted information.
- Reminder choices run from minutes up to one week, with some preselected by the agent.
- Location example in the mockup was only an example; the question is asked only when needed.

## My calls (change any)

- Quick replies come from the agent per question, at most three, plus free typing. Guess.
- The first card updates in place when extraction finishes instead of posting a second message. Guess.
- A notification still arrives once per review: "Email received" for card one is silent; the push fires for the question or the final item, whichever comes first. Guess.
- Do nothing stays on every open review.
- Default agent preselection: appointments 1 day and 1 hour; tasks with a deadline 1 day. Guess.
- Offline icon appears after 2 seconds of failure. Guess.

## Left to the builder

Data model for multiple reminders, migration, wire format for quick replies and reminder offsets, the exact Markdown subset for notes, prompt changes, and the test plan.

## Acceptance

- A synthetic appointment email produces the three-message flow on compiled web and in the Flutter widget tests.
- The item card shows the title once, one row each for Starts, Ends, Duration, Address and Type, a notes list with no reminders, and reminder chips with the agent's choices on.
- Switching chips changes the stored reminders; each fires once at its offset in a local scheduler test, and Done or Undo still work.
- A forced two-second connection drop shows no banner; a ten-second drop shows the header icon and recovery clears it.
- Existing items and single reminders load and fire as before after the migration.
