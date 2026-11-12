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
Orbital tests cover highlight movement with a pixel-stable artwork layer, distinct scenes, stable foreground builds, reduced motion and lifecycle pauses. Screen goldens use reduced motion for deterministic captures.
`native_auth_probe.dart` is an emulator-only callback probe.

Capture tests cover source evidence, typed authenticated attachment bytes, targeted answers while normal chat waits, and hotel-night overlap without checkout.

# Work Guidance

Assert user-visible results and lifecycle effects rather than private method calls.

# Verification

`toolchain/run flutter flutter test`. Run analyze and dart format after changes.

# Child DOX Index

None.
