# Security Policy

## Reporting a Vulnerability

If you discover a security vulnerability in Lelemon, please report it responsibly.

**Do NOT open a public GitHub issue for security vulnerabilities.**

Instead, please email us at: **security@lelemon.dev**

You will receive a response within 48 hours acknowledging your report. We will work with you to understand the scope of the issue and develop a fix before any public disclosure.

## What Qualifies

- Authentication or authorization bypass
- SQL injection, XSS, or other injection attacks
- Data leaks between tenants (project isolation bypass)
- Token/credential exposure
- Remote code execution

## What Does NOT Qualify

- Denial of service (unless trivially exploitable)
- Issues requiring physical access
- Social engineering
- Issues in dependencies (report those upstream)

## Supported Versions

| Version | Supported |
|---------|-----------|
| Latest release | Yes |
| Older releases | No |

We recommend always running the latest version.

## Disclosure Policy

- We will acknowledge receipt within 48 hours
- We will confirm the issue and determine its impact within 7 days
- We will release a fix within 30 days for critical issues
- We will credit reporters in the release notes (unless they prefer anonymity)
