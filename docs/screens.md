# Minder screens and flows (draft)

Status 2026-11-11. Derived from [requirements.md](requirements.md) and [master-plan.md](master-plan.md). Portrait phone first. Open decisions are marked "?" and listed in "Open decisions".

All names, dates, counts and times in the text and in the mockups are illustrative.

## UX rules

- Four bottom tabs: Today, Board, Calendar, Lists.
- The assistant bar (text field and microphone button) sits above the tabs on Today.
- Every action has a visible button. Drag is only a shortcut.
- Done and Snooze act at once. Every assistant request that creates or changes data shows a preview with Confirm, Edit and Discard. The assistant saves nothing before you confirm.
- A label is free text with one colour from a fixed set of six. The colour always shows with the label text.
- The app reads and changes data through the server. Without a connection, a screen shows an error and a Retry button.
- All text follows about 80% of ASD-STE100: sentences of 20 words or fewer, active voice, one word for one meaning.
- The look follows the brand guide in `../brand/`.

## Objects

| object | what it is |
|---|---|
| Item | Base unit. Has a title, type (task, event or reminder), status, due date and time, profile, labels, reminders, nag mode and notes |
| List | Named collection of items. Every item belongs to one list. New items go to the Inbox list until you sort them |
| Label | Free text with one of six colours: Blue #3B86F0, Teal #34D1BF, Green #9BE38A, Amber #E9A020, Lilac #C3B0FF, Slate #6F879F |
| Profile | Named set of reminder times, loudness, nag mode (fixed or escalating) and nag interval. Five are built in: Daily task, Chore, Birthday, Day event, Trip. You can edit them and add your own |

An item uses one profile. Reminders set on the item override the profile. Board columns map to item statuses: To do is open, Doing is in progress, Done is done.

## Map

```
Today ─ focus frame, groups ─▶ Item sheet
  ├─ assistant bar ──────────▶ Assistant (chat, voice, preview)
  └─ gear ───────────────────▶ Settings ─▶ Profiles ─▶ Profile editor
Board ─ cards, Move ─────────▶ Item sheet
Calendar ─ day, week, month ─▶ Item sheet
Lists ─ list of lists ───────▶ Items of one list ─▶ Item sheet
```

The gear button on each tab opens Settings. The item sheet opens from any item row or card.

## Today

Purpose: show what to do now and what else is due today.

Content, from top to bottom:
- Title, date and the gear button.
- Progress squares, one per item due today, with "done of total" (3 / 8). A filled square is a done item. The white square is the focus item.
- Focus frame with corner brackets. It shows the most urgent item: the oldest overdue item first, then the next due item. It shows title, label, profile and due time, and the buttons Done and Later.
- Group "Done today", collapsed, with a count.
- Groups Morning, Afternoon and Anytime, each with a count. Each row has a check button, title, label, profile and time. The focus item does not repeat in a group.
- Assistant bar and tabs.

Actions:
- Done on the focus frame or the check button on a row marks the item done at once.
- Later opens the snooze options: 15 min, 1 h, tonight, tomorrow, custom. The choice acts at once (?).
- Tap a row to open the item sheet. Tap "Done today" to expand it.
- Tap the text field or the microphone to open the Assistant.

States:
- Nothing due: the frame says "Nothing due today" and shows no buttons.
- All done: all squares are filled and the frame says "All done".
- Other overdue items show in the group of their time of day, with a red "overdue" label (?).
- No connection: error and Retry button.

Serves: requirements §4 (Today), §5.2, §6.2.

## Board

Purpose: one Kanban board for all items, not split by project.

Content:
- Title, gear button, filter chips (All, then one chip per label with its colour square) and the item count.
- Columns To do, Doing and Done, each with a count. They scroll sideways.
- Cards show title, due date, label and a Move button. An overdue card has a Signal border.

Actions:
- Move opens a sheet with the columns. Choose a column to move the card at once. Dragging a card is a shortcut.
- A label chip filters all columns. All removes the filter.
- Tap a card to open the item sheet.

States: empty column shows "No items". No connection: error and Retry button.

Serves: requirements §4 (Kanban).

## Calendar

Purpose: see items by day, week or month.

Content:
- Title, gear button and a switch with Day, Week and Month.
- Range line with previous and next buttons (9 – 15 Nov 2026).
- Week view: seven day cells, each with the day number and the count of items. The selected day has a frame. Below the cells: the items of the selected day, in time order, with a check button.
- Day view: the items of one day. Month view: a month grid with a count per day. Tap a day to select it.

Actions: switch view, move the range, select a day, check an item off, open the item sheet.

States: a day with no items shows "No items". No connection: error and Retry button.

Serves: requirements §4 (Calendar), §8. Only the Week view has a mockup; Day and Month need mockups before the build.

## Lists

Purpose: open lists of items and see how many are open.

List of lists:
- Title, gear button, count of lists and open items.
- One row per list: name and "open of total". Inbox is the first row.
- Button New list.

Items of one list (after you tap a list):
- Back button, list name, "open · total".
- Rows with check button, title, label, status tag and due date.
- Button Add item. It opens an empty item sheet.

States: an empty list shows "No items" and the Add item button.

Serves: requirements §3, §9.

## Assistant

Purpose: create and change data by text or voice.

Content:
- Back button and title.
- Chat messages. Your messages are on the right.
- Preview card in an assistant message: type of change, item title, the changed fields (old value struck through, new value below), the note "Nothing is saved before you confirm", and the buttons Confirm, Edit and Discard.
- Composer. Text mode: text field and microphone button. Voice mode: cancel button, a "Recording" label with a timer, a waveform and a send button.

Actions:
- Type and send a message, or tap the microphone to record. Send stops the recording. The server transcribes it and the transcript shows as your message.
- Confirm saves the change. Edit opens the item sheet with the proposed values. Discard drops the preview. Nothing is saved before Confirm.
- Questions (for example "What is due today?") get an answer with no preview.

States: waiting for the server shows "Working"; failure shows an error and a Retry button.

Serves: requirements §1.2, §5.2, §17.

## Item sheet

Purpose: read and edit one item.

Content, in order: title, type (Task, Event, Reminder), due date and time, profile, labels (with Add label), reminders (from the profile or set on the item, with Add reminder), nag mode (Fixed or Escalating), nag interval, notes.

Actions: Save, Done (marks the item done at once), Close (drops unsaved edits). The profile field opens the list of profiles. Add label asks for the label text and one of six colours.

States: new item opens with empty fields and the profile Daily task (?). A field with a bad value shows a short error under the field.

Serves: requirements §3.1, §6.2, §6.3.

## Profiles

Purpose: tune reminder behaviour by kind of item.

List: one row per profile with name, a "Built in" or "Custom" tag and a one-line summary (times, loudness, nag mode). Button Add profile.

Editor: name, reminder times (each with a remove button, and Add time), loudness (Soft, Normal, Loud, Ringing ?), nag mode (Fixed or Escalating), nag interval (minutes). Buttons Save and Cancel. The editor shows placeholder values. The real defaults are not decided.

Actions: edit any profile, including built-in ones; add a profile; remove a custom profile (?).

Serves: requirements §6.3.

## Settings

Purpose: show the connection to the server and the notification state.

Content:
- Pairing: status, method (?), button Pair again.
- Notifications: permission state, delivery through the server and push, and a row that opens Profiles.
- Account of the server: address (?), signed-in user (?), sign-in method (?).

Details marked "?" depend on decisions in requirements §16.

Serves: requirements §11, §13, §16.

## Flows

Create by assistant:
1. On Today, tap the text field or microphone. The Assistant opens.
2. Type or record a request, for example "Call the plumber on Thursday at 10".
3. The assistant shows a preview card.
4. Tap Confirm to save, Edit to change the values, or Discard to stop. The item then shows on Today, Board and Calendar.

Complete an item:
1. Tap the check button on a row, or Done on the focus frame.
2. The item moves to "Done today". The progress squares and the count update. The next most urgent item takes the focus frame.

Move on the board:
1. On Board, tap Move on a card.
2. Choose the target column. The card moves at once.
3. Dragging the card to another column does the same.

Edit a profile:
1. Tap the gear, then Profiles.
2. Tap a profile.
3. Change the fields. Tap Save.
4. Items that use the profile take the new values. Reminders set on an item stay as set (?).

## Open decisions

- ? Later on the focus frame uses the same options as Snooze.
- ? Other overdue items show in the group of their time of day with an "overdue" label.
- ? The assistant bar shows on Today only. You open the Assistant from other tabs by going to Today.
- ? Board shows items with the statuses open, in progress and done. Cancelled items do not show.
- ? How you delete or cancel an item by hand. The item sheet has no button for it.
- ? Loudness has four levels: Soft, Normal, Loud, Ringing.
- ? You can remove custom profiles but not built-in ones. The effect on items that use a removed profile is not decided.
- ? A new item starts with the profile Daily task.
- ? Items that use an edited profile take the new values, except reminders set on the item.
- ? Pairing method, server address and sign-in method. See requirements §16.
- ? The default values of each built-in profile. See requirements §16.

## Mockups

HTML in `mockups/`, each with a rendered `.png` next to it. Shared styles are in `mockups/common.css`.

| file | shows |
|---|---|
| `today` | title, progress squares, focus frame, groups, assistant bar, tabs |
| `board` | label chips, columns with cards, Move buttons |
| `calendar` | Day, Week and Month switch; week view with the items of Wed 11 Nov |
| `lists` | list of lists with counts; items of Trip Lisbon |
| `assistant` | chat with a preview card and a voice message in progress |
| `item` | item sheet |
| `profiles` | list of profiles; editor of Trip |
| `settings` | pairing, notifications, account of the server |

Counts agree on every screen: 13 items in total, 10 open and 3 done. Wed 11 Nov has 7 items due and 1 older overdue item, so Today shows 3 / 8 done.
