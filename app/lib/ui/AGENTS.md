# Purpose

Flutter screens and presentation.

# Ownership

ChatScreen, CalendarScreen, ScreenHeader, BottomNav, EntryView, OrbitalBackground, time/cost labels and Tokens.theme.

# Local Contracts

Use controllers and typed domain values. Never build API routes or read raw transport maps.
Theme controls buttons, fields and standard visual density. Preserve 44 px controls and existing goldens.
Screens use Material icons and bundled fonts.
ScreenHeader owns title layout, responsive padding and the left-to-right fading rule below it for both screens. It has no buttons.
BottomNav owns the two-item Chat/Calendar bar: 56 px items, a Signal icon and top bar on the selected item, hidden while the keyboard is open. Each screen passes its own destination and switch action. Do not add presence/status copy to headers. Loading and errors belong in content.
ChatScreen shows the existing Working spinner above the composer, outside the scrolling message list while ChatController.thinking is true, including the agent reading an email and quick-reply or follow-up answers. The composer has no character limit: it takes a pasted email up to the controller's 400,000-byte limit. It clears when processing finishes or fails; unrelated chat waiting keeps it visible.
ChatScreen follows the newest cursor and keeps the list at the bottom through keyboard, composer and navigation viewport changes. User scrolling away pauses viewport following; returning to the bottom or receiving a new cursor resumes it.
CalendarTokens owns calendar dimensions. The calendar uses a centered, scrolling column with grouped month arrows and Today beside the agenda date.
Day targets stay at least 44 px wide; row height expands for enlarged text. Calendar goldens change only with an approved design.

OrbitalBackground owns decorative painting and animation lifecycle. Use distinct OrbitScene values for chat and calendar.
`orbital_artwork.dart` owns static painting, layout and cached path metrics. Paths remain fixed while highlights travel along them. Highlights ease in and out (`Curves.easeInOut`) over one loop of 68.571 s; their mean speed is 2.625 times the original 180 s linear loop and their brightness is unchanged.
`orbital_artwork_data.dart` is generated from the canonical SVGs in brand/; run toolchain/export-orbits.py after SVG changes.
Keep static artwork/glow and foreground isolated from moving highlight repaints. Stop motion for reduced motion, hidden routes and backgrounded apps.

`ChatMarkdown` renders Minder's own text (messages, results, questions, lookup offers) as Markdown with `flutter_markdown_plus`: bold, italic, lists, code, quotes, tables and links in Tokens colours, headings at body size. Phone numbers, emails and web addresses are linked by `findLinks` rules; only HTTPS, tel and mailto open, after a tap, through the `LinkLauncher`; images are never fetched. Preview and calendar notes use the same widget so chat and cards look alike. Your own messages, errors, previews and structured fields stay plain text. The agent keeps writing Markdown; there is no server-side stripping.

QuestionView shows open questions with quick replies and “Or give a custom response below.” The main ChatScreen composer is the only reply field, with neutral text and cursor colours. QuestionPrompt also exposes questions inside CaptureView details. Both shortcut copies use ChatController's shared answer state, hide choices immediately on submission and restore them on refusal. Main-composer replies carry review context without parsing the owner's words; the agent decides their meaning. Duration questions retain fallback choices. CaptureView keeps English structured fields, source attachments, coverage warnings, zero-item wording, Do nothing and source deletion. Evidence IDs and quotes stay hidden. Queued candidates show Waiting. Existing previews apply individual items; arrival changes none.

Calendar items with source links open a read-only email/attachment dialog; authenticated downloads remain available after the original capture leaves recent chat.

Each calendar agenda card has a 44 px Delete icon button (tooltip `Delete`) in its top row. It opens `ItemDeleteDialog`, which shows the server preview (what goes with the item), then Delete or Keep it. Tapping outside does nothing; Back or any other exit discards the preview, even while it loads, so no pending delete card stays in the chat. An error entry shows Retry only while it is the newest entry. The chat error block shows a message only (no Retry); sign-in keeps its button. ScreenHeader takes an optional `trailing` control; Chat shows a 44 px offline icon there (tap to retry) and no connection banner. Preview cards come from `previewRows`: the title shows once, Starts/Ends carry their timezone inline, Duration and Type read as words (Event for `event`), Address is the `location` field, notes render through `ChatMarkdown` as a list capped at 600 characters, and reminders are `ReminderChips` that call `setReminders` while the change is pending.


Calendar cards expand by tapping the card or its 44 px Details / Hide details button. `ContentAction` owns neutral text actions and the square keyboard focus ring (2 px Text outline, 3 px offset). The theme gives text actions white text, square bounds and neutral hover/press feedback. Disclosure rows use down/up chevrons and expose expansion state; source-dialog actions use a right chevron. Address blocks align with other fields, wrap their full text, show an external arrow and “Opens Google Maps”, and do not toggle the card when tapped. Detail labels have 4 px before values and fields have 16 px gaps; source zones have their own line.

`CalendarItemDetails` labels task starts Scheduled and due fields Deadline. An explicit notification appears when no offset reminders override it. Previews use the same task labels when the kind is available. Other detail rows retain duration, type, address, contacts, reference and notes. Additional item details show their supplied labels and use `LinkedText` for values, including meeting links. Preview rows retain these as literal `Details` lines with tappable links. Contact rows open their external apps; booking references remain selectable plain text with an explicit accessible value. New and legacy confirmation previews show the same labels. Starts/Ends become Departs/Arrives only with two different, nonempty timezones. Local wall clocks retain their source zone. Addresses open Google Maps; `LinkedText` makes visible HTTPS, phone and email spans tappable and disposes its recognizers. Notes keep the shared 600-character cap. A failed launch shows a snackbar. Confirmed preview cards end with a full-width footer: divider, 16 px top gap, 12 px inner top padding, a check icon and semibold text; other resolved states use an information icon.

A minder entry with two `options` is a place-lookup offer: a card labelled `LOOK UP · ASKS FIRST` with the text. While it is the newest entry it shows Skip (outlined) and Look it up (filled, 44 px). A tap calls `ChatController.send` with that label, so the choice is a normal message. The buttons wait while a send or action runs.

# Work Guidance

Keep responsibilities focused. Preserve existing widget appearance during refactors.

# Verification

From root: `toolchain/run flutter flutter test`, `toolchain/run flutter flutter analyze`, `toolchain/architecture-check`.

# Child DOX Index

None.
