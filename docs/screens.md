# Minder screens (draft)

Status 2026-11-11. Derived from [requirements.md](requirements.md). The app has two screens: Chat and a Calendar that shows and deletes items ([calendar-spec.md](calendar-spec.md)). The owner decided that no other screen is needed now. Design rules are in [foundation.md](foundation.md). The first build slice is in [m0-spec.md](m0-spec.md).

All names, dates, counts and times in the text and in the mockup are illustrative.

## Email capture in chat

The [email capture spec](superpowers/specs/2026-11-12-email-capture-design.md) defines source summaries, attachments, coverage warnings and candidate-specific follow-up questions inside Chat.
The main composer handles custom replies to the current review question. Quick replies disappear as soon as an answer is sent; a refused send restores them and the typed text. Questions offer “Or give a custom response below.” instead of another input field. Each item uses the existing Confirm/Discard preview; clarifying one item leaves independent previews valid.
Expanded email candidates show the same labelled address, contact, booking reference and notes fields as calendar details. The review heading and extracted descriptions are English; original-source quotes and internal evidence IDs are not displayed. Original attachments remain available. A missing appointment duration offers 30 minutes and alternatives, both in the expanded item and the separate question message.
Source deletion first creates an irreversible preview. Mockup: [email capture](mockups/email-capture.html).

A task's Scheduled date is when the owner plans to do it. Deadline is the source's cutoff, kept separate from that choice. A questionnaire chosen for three days before the first appointment appears on that scheduled day, with a notification for the work time. Changing notification offsets preserves the task date and deadline and replaces any older explicit reminder.

## Calendar event bars — approval draft

Requested on 2026-11-14; awaiting owner approval. [Calendar review mock](mockups/calendar-events-review.html) and [render](mockups/calendar-events-review.png) propose named coloured bars in the month instead of dots. Multi-day events span their dates and continue with the same name and colour across week boundaries. A selected day's full event names appear in the agenda. Day targets remain at least 44 px; compact bars describe events and day taps select the agenda.

The draft has two party examples: 17:00–20:00 on one day, and 17–20 November across four dates. Ambiguous ranges need clarification rather than a guess. Both use Event, Starts/Ends and a plain Location value for Home. Hotel checkout wording applies only to an actual stay. Maps links require a real address rather than an arbitrary location label; that semantic decision belongs to the agent, not a word list. Event colours are a proposed calendar extension to the neutral brand palette. Use the six existing label tokens: Blue, Teal, Green, Amber, Lilac and Slate. Red remains reserved for its five brand uses. The same item colour appears on every span, its agenda card, details accent and named colour control; Workshop is Blue in all four places. A colour never replaces the item name.

The revised mock uses the shared stylesheet, header, 56 px navigation, Carbon 11.92.0 icons, 44 px selection frames and 8/16/24/32 px spacing. Empty date rows retain 56 px targets; rows with named bars grow by 24 px per event lane. Content scrolls. This row growth and event colour treatment await approval; the canonical brand guide remains unchanged.

Each item has an editable colour. The details and expanded agenda offer a named six-colour picker with Cancel and Apply colour. Chat creation and editing accept colour requests and show the same picker in the existing item preview. If colour is omitted, the assistant proposes a palette colour in that preview without a separate required question. An explicit colour wins and remains unchanged during unrelated edits. Chat previews use Edit, Discard and Confirm; selecting a swatch changes the proposal, and Confirm saves it. The mock demonstrates a Teal dinner creation and a Blue-to-Amber Workshop edit through the one neutral main composer. Its scripted examples and changes held only until reload are labelled. This is a review artifact; the app, prompt and approved calendar remain unchanged.

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
- Shared header: title "Chat" and a rule below it that fades left to right. The Calendar screen uses the same treatment. A bottom bar with Chat and Calendar sits below the input field.
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

Month grid and agenda with a Delete control on each card, with the fixed Calendar orbital artwork behind the content. The month heading and year sit beside grouped month arrows.
Today sits beside the selected date in the agenda header.
The whole page scrolls on short screens. Spacing and selection rules are in [foundation.md](foundation.md).
Behaviour is in [calendar-spec.md](calendar-spec.md). The mockup is `mockups/calendar.html` with `mockups/calendar.png`.

## Open decisions

- ? How you reach the Inbox, lists and profiles. They appear in chat answers until a screen exists.
- ? Edit in the preview card. M0 may leave it out (see [m0-spec.md](m0-spec.md)).
- ? The snooze options and whether Later opens them in a sheet or as chips in the chat.
- ? How you delete or cancel an item. The assistant can do it through a preview.
- ? Whether Start small keeps its name.
- ? The default values of each built-in profile (requirements §16).

## Mockup

`mockups/chat.html` with `mockups/chat.png`. It shows a nag frame with a countdown, your message and a preview card. M0 omits the nag frame and the countdown, because the reminder engine starts in M1.
