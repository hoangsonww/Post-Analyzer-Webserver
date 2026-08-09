# Security Policy

## Supported Versions

| Version | Supported          |
|---------|--------------------|
| latest  | :white_check_mark: |

## Reporting a Vulnerability

**Please do not open a public GitHub issue for security vulnerabilities.**

If you discover a security vulnerability in Post Analyzer Webserver, please report it responsibly:

1. **Report privately**: open a [GitHub Security Advisory](../../security/advisories/new) on this repository.
2. **Include**:
   - Description of the vulnerability
   - Steps to reproduce
   - Potential impact
   - Suggested fix (if any)

You should receive an acknowledgment within **48 hours**. We will work with you to understand the issue and coordinate a fix before any public disclosure.

## Scope

The following are in scope:

- Authentication and authorization flaws (JWT handling, ABAC policy evaluation)
- Injection vulnerabilities (SQL injection, XSS, command injection)
- Sensitive data exposure (credentials, tokens, PII leaks)
- Broken access control (cross-user or cross-role data access via `/api/v1` or `/web/*`)
- RPC boundary issues (Kitex/Thrift) that leak or corrupt data across services
- Dependency vulnerabilities with a known exploit

The following are **out of scope**:

- Issues in third-party services (managed Postgres/Redis/Kafka providers, cloud platforms)
- Denial of service via rate limiting (we acknowledge this and plan to address it)
- Self-hosted deployment misconfigurations (e.g., using demo credentials in production)

## Disclosure Policy

- We follow [coordinated disclosure](https://en.wikipedia.org/wiki/Coordinated_vulnerability_disclosure).
- We aim to release a fix within **14 days** of confirming a vulnerability.
- Credit will be given to reporters in the release notes unless they prefer to remain anonymous.
