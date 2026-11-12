# Minder screens (draft)

Status 2026-11-11. Derived from [requirements.md](requirements.md). The app has two screens: Chat and a read-only Calendar ([calendar-spec.md](calendar-spec.md)). The owner decided that no other screen is needed now. Design rules are in [foundation.md](foundation.md). The first build slice is in [m0-spec.md](m0-spec.md).

All names, dates, counts and times in the text and in the mockup are illustrative.

## UX rules

- Chat is the main workspace. Minder answers requests and posts reminders there. The calendar shows dates.
- Lists, views and summaries appear as Minder messages in the chat.
- Every action has a visible button.
- Done and Snooze act at once and show an Undo button. Every assistant request that creates or changes data shows a preview with Confirm, Edit and Discard.
- A label is free text with one colour from a fixed set of six. The colour always shows with the label text.
- The app reads and changes data through the server. Without a connection, the screen shows an error and a Retry button.
- All text follows about 80% of ASD-STE100: sentences of 20 words or fewer, active voice, one word for one meaning.
- The look follows the brand guide in `../brand/` and [foundation.md](foundation.md).

## Objects

| object | what it is |
|---|---|
| Item | Base unit. Has a title, type (task, event or reminder), status, due date and time, profile, labels, reminders, nag mode and notes |
| List | Named collection of items. Every item belongs to one list. New items go to the Inbox list until you sort them |
| Label | Free text with one of six colours: Blue #3B86F0, Teal #34D1BF, Green #9BE38A, Amber #E9A020, Lilac #C3B0FF, Slate #6F879F |
| Profile | Named set of reminder times, loudness, nag mode (fixed or escalating) and nag interval. Five are built in: Daily task, Chore, Birthday, Day event, Trip |

M0 builds items only. Lists, labels and profiles come later (see [m0-spec.md](m0-spec.md)).

## Chat

Fixed orbital artwork sits behind the content, with small highlights travelling along the paths. Chat uses a tighter composition than Calendar.

Purpose: create and change items, ask about them, and receive reminders, in one conversation.

Content, from top to bottom:
- Header: title "Chat", status line ("Minder is watching · 5 open") and the gear button.
- The conversation, oldest first, newest at the bottom.
- Composer: text field and microphone button.

Entry types:
- **Your message:** Panel fill and Edge outline, on the right.
- **Minder message:** plain text with a sender line and a time.
- **Event line:** a centred line between hairlines, for example "07:15 · Reminder 2 sent · no reply".
- **Nag frame:** a Minder message in the focus frame. Urgency label, item title, one helpful sentence, buttons Done, Later and Start small, and a countdown to the next reminder.
- **Preview card:** type of change, item title, changed fields (old value struck through, new value after it) and buttons Confirm, Edit and Discard.
- **Result:** what changed, with an Undo button.
- **Working:** shown while the server answers.
- **Error:** the problem and a Retry button.

Actions:
- Type and send, or tap the microphone to record. Send stops the recording. The server transcribes it and the transcript shows as your message.
- Confirm saves the change. Edit opens the proposed values for change. Discard drops the preview.
- Done acknowledges the nag and marks the item done. Later opens the snooze options: 15 min, 1 h, tonight, tomorrow, custom. Start small gives one small first step. Done and Later act at once.
- Questions such as "What is open?" get an answer with no preview.

States:
- Empty conversation: Minder greets you and says what is open.
- Waiting for the server: a Working entry.
- Failure: an error entry with Retry.
- A reminder arrives while the app is open: a nag frame appears at the bottom.
- A reminder arrives while the app is closed: an Android notification with Done, Snooze and Open. Open shows the chat at the nag frame.

Serves: requirements §1.2, §5.2, §6.2, §7.

## Calendar

Read-only month grid and agenda, with the fixed Calendar orbital artwork behind the content. The month heading and year sit beside grouped month arrows.
Today sits beside the selected date in the agenda header.
The whole page scrolls on short screens. Spacing and selection rules are in [foundation.md](foundation.md).
Behaviour is in [calendar-spec.md](calendar-spec.md). The mockup is `mockups/calendar.html` with `mockups/calendar.png`.

## Open decisions

- ? What the gear opens. M0 builds no Settings.
- ? How you reach the Inbox, lists and profiles. They appear in chat answers until a screen exists.
- ? Edit in the preview card. M0 may leave it out (see [m0-spec.md](m0-spec.md)).
- ? The snooze options and whether Later opens them in a sheet or as chips in the chat.
- ? How you delete or cancel an item. The assistant can do it through a preview.
- ? Whether Start small keeps its name.
- ? The default values of each built-in profile (requirements §16).

## Mockup

`mockups/chat.html` with `mockups/chat.png`. It shows a nag frame with a countdown, your message and a preview card. M0 omits the nag frame and the countdown, because the reminder engine starts in M1.
