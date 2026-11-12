# Purpose

Email capture application service.

# Ownership

capture package.

# Local Contracts

Use persistence/blob/model interfaces. Only validated command previews may change items. Preserve originals before acknowledging ingestion; no infrastructure imports.

Capture contexts and answers stay separate from normal chat. Preserve independent and terminal candidates. Publish validated previews, coverage and job completion in one transaction. Retries keep originals and obey attempt/model budgets.

# Work Guidance

Use synthetic fixtures; preserve original content and privacy.

# Verification

`toolchain/run go go test -race ./capture`.

# Child DOX Index

None.
