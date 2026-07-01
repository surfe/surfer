# Security Policy

## Reporting a vulnerability

Please **do not** open public GitHub issues for security vulnerabilities.

Instead, report them privately to **security@surfe.com** (or use GitHub's
[private vulnerability reporting](https://docs.github.com/en/code-security/security-advisories/guidance-on-reporting-and-writing-information-about-vulnerabilities/privately-reporting-a-security-vulnerability)
on this repository).

Please include:

- A description of the issue and its impact
- Steps to reproduce (proof of concept if possible)
- Affected version (`surfer version`) and platform

We aim to acknowledge reports within 3 business days and will keep you updated on
remediation progress.

## Scope

This repository is the `surfer` CLI. Vulnerabilities in the Surfe API itself should
also be sent to security@surfe.com.

## Handling of credentials

- OAuth tokens are stored locally at `~/.surfer/tokens.json` with `0600` permissions.
- `surfer --debug` prints full HTTP request/response bodies, which may include
  personal data and bearer tokens. Do not share debug output publicly.
