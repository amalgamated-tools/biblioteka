# Security Policy

## Supported Versions

Only the latest release of Biblioteka receives security fixes. Older versions are not actively patched.

| Version | Supported |
|---------|-----------|
| Latest  | ✅        |
| Older   | ❌        |

## Reporting a Vulnerability

**Do not open a public GitHub issue for security vulnerabilities.**

Please report security issues by emailing the maintainers privately or by using [GitHub's private security advisory feature](https://docs.github.com/en/code-security/security-advisories/guidance-on-reporting-and-writing/privately-reporting-a-security-vulnerability):

1. Go to the [**Security** tab](../../security) of this repository.
2. Click **"Report a vulnerability"**.
3. Fill in the details of the issue.

### What to include

A good vulnerability report includes:

- A clear description of the issue and its potential impact.
- Steps to reproduce (proof-of-concept code or a request/response pair is helpful).
- The affected component (e.g. auth middleware, JWT validation, OIDC handler).
- Suggested remediation if you have one.

### Response timeline

| Milestone | Target |
|-----------|--------|
| Acknowledgement | Within **3 business days** |
| Initial assessment | Within **7 business days** |
| Fix or mitigation | As soon as practicable; critical issues are prioritised |

We will credit reporters in the release notes unless you prefer to remain anonymous.

## Security considerations for self-hosted deployments

When running Biblioteka yourself:

- **Set `JWT_SECRET` to a strong random value** (`openssl rand -hex 32`) before the first launch. The default `change-me-in-production` value must not be used in production.
- **Enable `SECURE_COOKIES=true`** when serving over HTTPS to prevent the session cookie from being transmitted over plain HTTP.
- **Terminate TLS at a reverse proxy** (Caddy, nginx, Traefik). Do not expose port 8080 directly to the internet.
- **Keep Redis access restricted** to the host running Biblioteka. Redis itself has no authentication configured in the default `docker-compose.yml`.

See [docs/deployment.md](docs/deployment.md) for the full production checklist.
