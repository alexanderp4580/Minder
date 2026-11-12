# Purpose

Flutter behaviour and rendering checks.

# Ownership

Controller tests, HTTP/native boundary tests, auth tests and goldens.

# Local Contracts

Use domain values or wire fixtures at the boundary under test. Tests own their repositories.
Golden files capture the current chat/calendar rendering. Do not update them to hide a regression.
Calendar spacing tests check the 8 px week gap, inset selection, Today placement and 320 and 360 px scrolling layouts with enlarged text.
`native_auth_probe.dart` is an emulator-only callback probe.

# Work Guidance

Assert user-visible results and lifecycle effects rather than private method calls.

# Verification

`toolchain/run flutter flutter test`. Run analyze and dart format after changes.

# Child DOX Index

None.
