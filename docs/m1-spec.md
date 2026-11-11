# M1: items, lists, profiles and recurrence

Status 2026-11-12. The owner asked for M1 to finish without questions. Reminders, nagging and catch-up shipped earlier: [notifications-spec.md](notifications-spec.md). This slice adds the rest of M1. Evidence: [evidence/m1.md](evidence/m1.md).

## Scope

**In**
- Lists: an item can carry a list name.
- Profiles: reminder defaults for a kind of item. Five built-ins and your own.
- Recurrence: daily, weekly, monthly and yearly repeats, fixed or after completion.
- The assistant sets all three and answers "what is on Groceries".

**Out (later)**
- Reset and clone of lists, subtasks, labels, priority, projects, profile statuses, loudness.

## Behaviour

- A list is a name. The server keeps one spelling for each name and ignores case when it matches. An empty list disappears with its last item.
- A profile has a name, minutes before the due time (0 to 20,160) and a nag interval (0 to 1,440).
- Built-in profiles: Daily task (at due, nag 15), Chore (at due, no nag), Birthday (1 day before), Day event (1 hour before), Trip (2 days before). These values are first guesses. You tune them.
- An item with a profile and a due time gets its reminder from the profile. A reminder set on the item wins. An item with neither gets no reminder.
- A profile edit moves the reminders of every item that uses it. A built-in profile cannot be deleted. A custom profile cannot be deleted while an item uses it.
- A repeating item needs a due time. Fixed repeats move to the next slot after now and keep the hour in the item's timezone, also across daylight-saving changes. Monthly repeats on day 29 to 31 use the last day of a short month. "After completion" counts from the day you finish.
- Completing a repeating item keeps it open and moves its due time. A reminder set on the item keeps its distance to the due time. Undo restores the old times.
- Every change still shows a preview. Profile changes use the same preview, Confirm, conflict and Undo rules as items.

## Interfaces

Item fields: `list`, `profile` (profile id), `recurrence` (`every`, `interval`, `after_completion`).
Proposal fields: `list`, `clear_list`, `profile`, `clear_profile`, `recurrence`, `clear_recurrence`, and for profiles `profile_id`, `profile_name`, `profile_remind_before_minutes`, `profile_nag_minutes` with `expected_version`.
Actions: `create_profile` and `update_profile` join the item actions.
Model tools: the three item tools take the new fields. `propose_profile` adds or changes a profile. `list_items` takes an optional `list`.
The model context holds the list names and all profiles.
Schema version 3 adds the `profile` table. Built-in rows are inserted once and never overwritten.

## Acceptance

- Names and values match this file in domain, store and command tests.
- No preview writes data. Confirm applies once. Undo restores items and profiles.
- Existing reminder tests still pass.
- A real model call proposes a list, a profile, a repeat and a profile change.
