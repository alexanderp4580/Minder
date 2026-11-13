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
ChatScreen follows the newest cursor and keeps the list at the bottom through keyboard, composer and navigation viewport changes. User scrolling away pauses viewport following; returning to the bottom or receiving a new cursor resumes it.
CalendarTokens owns calendar dimensions. The calendar uses a centered, scrolling column with grouped month arrows and Today beside the agenda date.
Day targets stay at least 44 px wide; row height expands for enlarged text. Calendar goldens change only with an approved design.

OrbitalBackground owns decorative painting and animation lifecycle. Use distinct OrbitScene values for chat and calendar.
`orbital_artwork.dart` owns static painting, layout and cached path metrics. Paths remain fixed while highlights travel along them. Highlights ease in and out (`Curves.easeInOut`) over one loop of 68.571 s; their mean speed is 2.625 times the original 180 s linear loop and their brightness is unchanged.
`orbital_artwork_data.dart` is generated from the canonical SVGs in brand/; run toolchain/export-orbits.py after SVG changes.
Keep static artwork/glow and foreground isolated from moving highlight repaints. Stop motion for reduced motion, hidden routes and backgrounded apps.

QuestionView shows the open questions of an email as their own message, with agent-chosen quick replies and one free reply; quick replies answer through CaptureController. CaptureView shows processing/review status and what was found (title, when, address), a Do nothing button on every open review (not processing, failed or reviewed), a clear zero-item message without a proposals line, source attachments, extraction warnings and explicit source deletion preview. Existing preview controls apply individual candidates; no item changes happen on arrival.

Calendar items with source links open a read-only email/attachment dialog; authenticated downloads remain available after the original capture leaves recent chat.

Each calendar agenda card has a 44 px Delete icon button (tooltip `Delete`) in its top row. It opens `ItemDeleteDialog`, which shows the server preview (what goes with the item), then Delete or Keep it; the dialog cannot be dismissed from outside. The chat error block shows a message only (no Retry); sign-in keeps its button. ScreenHeader takes an optional `trailing` control; Chat shows a 44 px offline icon there (tap to retry) and no connection banner. Preview cards come from `previewRows`: the title shows once, Starts/Ends carry their timezone inline, Duration and Type read as words (Appointment for `event`), Address is the `location` field, notes show as plain-text bullets capped at 600 characters, and reminders are `ReminderChips` that call `setReminders` while the change is pending.

# Work Guidance

Keep responsibilities focused. Preserve existing widget appearance during refactors.

# Verification

From root: `toolchain/run flutter flutter test`, `toolchain/run flutter flutter analyze`, `toolchain/architecture-check`.

# Child DOX Index

None.
