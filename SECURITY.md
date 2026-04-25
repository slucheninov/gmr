# Security Policy

## Supported versions

`gmr` is a single-binary CLI distributed via tagged GitHub Releases. Only the
**latest released minor version** receives security fixes. Older versions are
not patched — please upgrade.

| Version | Supported |
|---|---|
| 0.6.x | ✅ |
| < 0.6 | ❌ (Bash-era; please migrate to the Go binary) |

## Reporting a vulnerability

**Please do not open a public GitHub issue for security problems.**

Use one of the private channels below so we can investigate and patch before
the issue becomes public:

1. **Preferred** — [GitHub Security Advisories](https://github.com/slucheninov/gmr/security/advisories/new).
   This creates a private advisory, lets us collaborate on a fix, and
   automatically requests a CVE if applicable.
2. If GitHub is unavailable to you, contact the maintainer directly via the
   email listed on their [GitHub profile](https://github.com/slucheninov)
   with the subject prefix `[gmr-security]`.

When reporting, please include:

- A clear description of the issue and its impact.
- Steps to reproduce, or a minimal proof-of-concept.
- Affected version(s) (`gmr --version`).
- Any suggested fix or mitigation, if you have one.

## What to expect

| Stage | Target |
|---|---|
| Acknowledgement | within **3 business days** |
| Initial triage / severity assessment | within **7 business days** |
| Fix released (typical) | within **30 days** of acknowledgement, sooner for critical issues |
| Public disclosure | coordinated with the reporter; usually after a fix is shipped |

We aim to credit reporters in the release notes unless they prefer to remain
anonymous.

## Scope

In scope:

- Vulnerabilities in the `gmr` Go source code (`cmd/`, `internal/`).
- Issues in our GitHub Actions workflows that could compromise the supply
  chain (e.g. release artifact tampering).
- Dependency vulnerabilities affecting the produced binary.

Out of scope (please report upstream instead):

- Vulnerabilities in `gh`, `glab`, or `git` itself.
- Vulnerabilities in third-party AI provider APIs (Gemini, Claude, OpenAI).
- Issues that require a compromised local machine or stolen API key — `gmr`
  trusts the operator and the env vars they export.
- Social-engineering scenarios where the user pastes an attacker-controlled
  diff into a repo (the AI prompt is not a security boundary).

## Hardening notes for users

- Treat AI provider API keys as secrets — they are billed per-token and
  exfiltration costs you money. Use a dedicated key with rate limits.
- `gmr` reads the staged diff and sends a truncated copy to your chosen LLM
  provider. **Do not run `gmr` on repositories whose diff content cannot be
  shared with the provider** (private credentials, customer data, regulated
  PHI/PII). Use `GMR_MAX_DIFF=0` or simply `git commit` manually in that case.
- Verify release-asset checksums (`checksums.txt`) before installing.

Thank you for helping keep `gmr` and its users safe.
