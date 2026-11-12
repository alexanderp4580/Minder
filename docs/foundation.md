# Minder foundation (draft)

Status 2026-11-12. This file defines the shared design rules for chat and calendar. The values come from `../brand/index.html` and `mockups/common.css`. Where the two differ, the brand guide wins. Items marked "?" are not decided.

Scope: the chat screen and the calendar page ([calendar-spec.md](calendar-spec.md)). Other screens are not designed.

## Principles

- Chat is the main workspace. Minder answers requests and posts reminders there. The calendar shows dates.
- Minder nags in the conversation. A reminder is a message that stays until you act on it.
- Every action has a visible button. Typing is optional.
- Every request that creates or changes data shows a preview with Confirm, Edit and Discard.
- Everything on screen is Void, Panel, Edge or text colour. Signal red marks one thing at a time.

## Colour

| token | value | use |
|---|---|---|
| Void | #0a0a0a | screen background |
| Panel | #161616 | message, field, secondary button |
| Line | #3a3a3a | hairlines and dividers |
| Edge | #6b6b6b | outlines only, never text |
| Text | #f2f2f2 | main text |
| Muted | #b5b5b5 | secondary text, done fill |
| Dim | #8f8f8f | text-safe dim, the slash in numerals |
| Signal | #e0121a | the five uses below |
| Alert | #ff4448 | Signal-coloured text on Void |
| White | #ffffff | text on Signal |

Signal has five uses: the focus frame, an overdue or urgent state, the primary action, a selected or active state, live recording. Nothing else is red.

Label colours: Blue #3B86F0, Teal #34D1BF, Green #9BE38A, Amber #E9A020, Lilac #C3B0FF, Slate #6F879F. A label colour always shows with the label text.

Contrast ratios: to check against the brand guide for every pair used on the chat screen.

## Type

| role | face | size | weight | tracking |
|---|---|---|---|---|
| Screen title | Michroma, upper case | 22 px | 400 | 0.2 em |
| Countdown | Michroma | 20 px | 400 | 0.06 em |
| Item title in a frame | Chakra Petch | 20 px | 600 | normal |
| Card title | Chakra Petch | 18 px | 600 | normal |
| Button, primary | Chakra Petch | 16 px | 700 | 0.04 em |
| Message text | Chakra Petch | 15 px | 500 | normal, line height 1.4 |
| Button, secondary | Chakra Petch | 14 px | 700 | 0.04 em |
| Meta, time | Chakra Petch | 12 to 13 px | 500 | 0.02 em |
| Section label | Chakra Petch, upper case | 12 px | 600 | 0.1 em |

Rules:
- The smallest text is 12 px.
- Upper case is for short labels only: sender, card type, urgency.
- Buttons and meta text use mixed case.
- The app bundles its fonts. HTML guides and mockups load them from Google Fonts.
Titles use Michroma. All other text uses Chakra Petch.

## Space and shape

- Phone canvas 412 × 892 px. Side gutter 20 px. Gap between buttons 8 px.
- Header: 18 px above the title, no divider below. The status line sits 8 px under the title. The first message group starts 38 px under the status line.
- Message group: 16 px above the sender line, 6 px between the sender line and the message. The sender line carries the time: "Minder · 07:30" on the left, "You · 07:31" on the right, on the side of its message.
- Your message: 44 px left margin inside the gutter, so the box starts 64 px from the screen edge. Padding 10 px above and below, 12 px at the sides.
- Buttons in the chat are 44 px high. Secondary buttons have a Panel fill and a 1 px Edge outline. The brand guide sizes of 54 and 48 px apply to notifications.
- Buttons that share a row have equal gaps of 8 px. The primary button is always last in its row, so its cut corner sits at the bottom right of the row. Confirm is 1.3 times as wide as Discard. Snooze, Open and Done share one row.
- The composer is 56 px high including its hairline. The field is a 44 px box with a 1 px Edge border and 14 px side padding. It is exactly as tall as the 44 px Send button, with the text centred. Send is last.
- Hairlines (Line colour, 1 px) separate regions. Cards use an Edge outline.
- Corners are square. The primary button has one cut corner at the bottom right (12 px). The mic button has the same cut (10 px).
- Every control is at least 44 × 44 px.
- The focus frame is four Signal corner brackets, 14 px, 2 px stroke, with no full border.

## Calendar layout

The calendar follows the Layout section of the brand guide.

- Use 8 px inside control groups, 16 px between related rows, 24 px between sections and 32 px between major blocks.
- The calendar column is at most 520 px wide. Side gutters are 24 px at widths of 360 px and above.
- At narrower widths, reduce gutters to preserve seven 44 px day targets. The whole page scrolls on short screens.
- Leave 32 px after the screen header. Show the month at 24 px, with the year below at 14 px.
- Group the two 44 px month arrows with an 8 px gap. Today sits beside the agenda date.
- Leave 24 px before weekday labels, then 12 px before the date grid.
- Day targets are 56 px high, with 8 px between weeks. The selected frame is 44 × 44 px at normal text size.
- Grow day rows and selection frames when enlarged text needs more height. Keep the date and markers visible.
- Leave 32 px between the grid and agenda, and 16 px between the agenda heading and cards.
- Agenda cards use 20 px padding, an 18 px title and a Line outline. Leave 16 px between cards.

## Orbital background

- Three neutral ellipses with different centres, sizes and tilts sit behind the content. Static lighting anchors the upper-right and lower edge.
- Calendar uses broad, offset ellipses. Chat uses tighter arcs near the upper-right header.
- Lines use a 0.85 px stroke. Text-colour highlights fade from 60% at the top through 16% and 8% in the middle to 42% at the bottom.
- Static lighting peaks at 26% in the upper-right and 16% at the lower edge. Keep cards opaque.
- Rotate each ellipse around its own centre over 180 seconds. The static halo and foreground have separate repaint boundaries.
- Reduced motion keeps the composition still. Hidden routes and backgrounded apps stop animation.
- Cards stay opaque. Background decoration does not intercept taps or add screen-reader content.

## Icons

Carbon icons, pinned version @carbon/icons 11.92.0, loaded from jsDelivr. 18 to 22 px. They follow the text colour. An icon never stands alone without a label, except the gear and the microphone.

## Chat components

| component | what it is | colour rule |
|---|---|---|
| Header | title "Chat", status line, a 44 px square button on the right (Calendar) | the status square is Text colour |
| Status line | "Minder is watching · 5 open", upper case, 12 px | Muted |
| Event line | centred text between two hairlines, for example "07:15 · Reminder 2 sent · no reply" | Muted |
| Sender line | "Minder · 07:30" or "You · 07:31", above a message, on its side | Muted |
| Your message | Panel fill, Edge outline, right side, 64 px left margin | none |
| Minder message | plain text on Void, no outline | none |
| Nag frame | a Minder message in the focus frame: urgency label, item title, one helpful sentence, buttons Done, Later and Start small, a countdown | Signal brackets, Alert label, Done is primary |
| Countdown | time to the next reminder in Michroma, next to a row of 12 squares that fill as time passes | Muted fill |
| Preview card | Panel-less card on Void with an Edge outline: type of change, item title, old value struck through, new value, buttons Confirm, Edit, Discard | Confirm is primary |
| Composer, text | text field and mic button, 56 px high, hairline above | mic is Signal |
| Composer, voice | cancel button, "Recording" label with timer, waveform, send button | Signal outline and waveform |

Only one primary button is Signal on screen at a time (?). On the current mockup the nag frame and the preview card each show one.

## States

Each state needs a design before the build:
- Waiting for the server: "Working" with a status square.
- Error: the problem and a Retry button.
- Empty conversation: Minder greets and says what is open.
- Reminder fired while the app is closed: a notification with Done, Snooze and Open.
- Voice mode: recording, sending, transcript shown as your message.
- Long conversation: scroll, with the newest message at the bottom.

## Copy

- Follow about 80% of ASD-STE100: 20 words or fewer, active voice, one word for one meaning.
- A message says what happened or what Minder needs. It names the item.
- A nag message gives one small first step for a painful task.
- No filler, no praise, no emoji.

## Open decisions

- ? How you leave the conversation (settings, lists, board). The gear is the only exit today.
- ? Escalating nag wording and loudness. Defaults are in requirements §16.
- ? Whether the nag frame keeps its Done as the primary button when a preview card is also open.
- Orbital motion follows the background rules above. Countdown and progress motion remain design drafts.

## Mockup

`mockups/chat.html` and `mockups/chat.png` show the chat screen with a nag and a preview card.
