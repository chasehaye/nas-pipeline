# Security Policy

## Reporting a vulnerability

Please report security issues **privately** — do not open a public issue. Use
GitHub's private vulnerability reporting ("Report a vulnerability" under the
repository's **Security** tab), or contact the maintainer directly.

You'll get an acknowledgement within a few days and updates through to a fix.

## Sensitive data

This project handles data that must never be committed or shared publicly:

- **SWIM / Solace credentials** — supplied at runtime via environment variables
  / secrets, never stored in the repo.
- **LADD Industry file** — Controlled Unclassified Information (**CUI**). It is
  mounted at runtime and must never be committed, attached to an issue or PR, or
  included in logs or screenshots. The `filter` service fails closed without it.

When reporting a bug, **redact** any credentials, CUI, or aircraft identifiers
tied to blocked flights before sharing logs or data samples.

## Supported versions

Pre-1.0: only the latest `main` is supported.
