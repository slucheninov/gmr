# Security Policy

## Supported versions

`gmr` is a single-binary CLI distributed via tagged GitHub Releases. Security
fixes are released only for the **latest published version**; older releases are
not patched. Please upgrade before reporting an issue that may already be fixed.

| Release | Supported |
|---|---|
| Latest GitHub Release | ✅ |
| Older releases | ❌ |
| Bash-era versions earlier than 0.6.0 | ❌ |

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
- Leakage of repository content or credentials caused by `gmr` itself.

Out of scope (please report upstream instead):

- Vulnerabilities in `gh`, `glab`, or `git` itself.
- Vulnerabilities in third-party AI provider APIs (Gemini, Claude, OpenAI).
- Issues that require a compromised local machine or a previously stolen API
  key — `gmr` trusts the operator and the environment variables they export.
- Social-engineering scenarios where the user pastes an attacker-controlled
  diff into a repo (the AI prompt is not a security boundary).

## Hardening notes for users

- Treat AI provider API keys as secrets. Use dedicated keys with appropriate
  limits, never commit them, and rotate them if they may have leaked.
- For commit generation, `gmr` first runs `git add -A`, then sends the staged
  diff (up to `GMR_MAX_DIFF` lines) and its diff stat to the configured provider.
  This can include previously unstaged and untracked files.
- For release generation, `gmr deploy` can send commit subjects and bodies from
  the previous release tag through `HEAD` to the configured provider.
- **Do not use AI-generating commands on content that cannot be shared with the
  selected provider** (credentials, customer data, regulated PHI/PII, or other
  confidential material). `GMR_MAX_DIFF` limits the payload size; it does not
  disable transmission, and non-positive values fall back to the default limit.
  Commit manually instead. An already committed feature branch can be submitted
  without AI generation, and `gmr status` does not call an AI provider.
- Set `GEMINI_BASE_URL`, `ANTHROPIC_BASE_URL`, or `OPENAI_BASE_URL` only to an
  endpoint you trust: the corresponding API key and repository-derived prompt
  are sent to that endpoint.
- Verify release-asset checksums (`checksums.txt`) before installing.

Thank you for helping keep `gmr` and its users safe.
