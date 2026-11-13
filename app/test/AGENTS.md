# Purpose

Flutter behaviour and rendering checks.

# Ownership

Controller tests, HTTP/native boundary tests, auth tests and goldens.

# Local Contracts

Use domain values or wire fixtures at the boundary under test. Tests own their repositories.
Golden files capture the current chat/calendar rendering. Do not update them to hide a regression.
Header parity tests compare the real Chat/Calendar headers at 320, 412 and 1280 px, reject presence/status text and header buttons, and check the fading rule (1 px, left to right). Chat tests check the composer spacing above the bar.
`chat_test.dart` covers bottom following through keyboard opening, sending and stepped keyboard dismissal, plus reading-position preservation and resumed following.
`bottom_nav_test.dart` covers the bar: items, centring, selected state, 56 px targets and keyboard hiding.
Calendar spacing tests check the 8 px week gap, inset selection, Today placement and 320 and 360 px scrolling layouts with enlarged text.
Orbital tests cover the 68.571 s highlight loop, ease in and out, highlight movement with a pixel-stable artwork layer, distinct scenes, stable foreground builds, reduced motion and lifecycle pauses. Screen goldens use reduced motion for deterministic captures.
`native_auth_probe.dart` is an emulator-only callback probe.

Connection status tests cover the grace period, the header icon, recovery, no banner for background drops, the restored composer text after a failed send (also with newer typed text), a server refusal message without Retry, a dropped event stream (icon only, no message), a background message that clears after a good sync, and removal of deleted cards. Calendar tests cover the 44 px Delete control, its preview dialog, Confirm, Keep it and Back. A chat test covers Retry on the newest error entry only. Question tests (`question_view_test.dart`) cover the found-items card, quick replies and typed replies. Preview card tests (`preview_card_test.dart`) cover the title once, inline timezones, units, labels, inert notes and reminder chips. Capture tests cover source evidence, typed authenticated attachment bytes, targeted answers while normal chat waits, Do nothing and zero-item wording, and hotel-night overlap without checkout.


`booking_details_test.dart` covers legacy conflicts, calendar Ref lines, leading zeros beyond the note cap and typed dial targets. Calendar tests cover separate contact/reference rows from both typed JSON and older notes.

`links_test.dart` covers adjacent phone labels, non-breaking spaces, optional international trunk markers, leading-zero numeric references, Dutch/international numbers, booking references, dates, punctuation, overlapping links and encoded maps queries. `link_launcher_test.dart` covers the scheme/target boundary and platform failures. Calendar tests cover zone labels, expandable rows, maps and note links, and 320 px layouts at 1.4× text. `preview_card_test.dart` covers the separated Confirmed footer and its narrow-screen golden.

# Work Guidance

Assert user-visible results and lifecycle effects rather than private method calls.

# Verification

`toolchain/run flutter flutter test`. Run analyze and dart format after changes.

# Child DOX Index

None.
