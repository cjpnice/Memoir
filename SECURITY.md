# Security Policy

[中文](SECURITY.zh-CN.md)

Memoir / 集忆 is currently a local-first, single-user, pre-1.0 project. Imported photos, generated albums, AI settings, API keys, and all files under `DATA_DIR` should be treated as private user data.

## Supported Versions

During pre-1.0, only the current `main` branch is maintained.

## Reporting A Vulnerability

If the repository is hosted on GitHub, please use GitHub Security Advisory private reporting first. If that is not enabled, contact the maintainer privately before disclosing exploitable details.

Please include:

- A short summary.
- Reproduction steps.
- Whether photos, album exports, share pages, API keys, or local files may be exposed.
- Operating system, browser, and runtime mode: local, Docker, or single-file app.
- Whether an AI provider was configured, and what provider type was used.

Do not upload private photos or real API keys. Reproduce with sanitized or publishable media whenever possible.

## Current Security Boundary

- Memoir has no built-in user authentication and should not be exposed directly to the public internet.
- Defaults are designed for local development. Add authentication, TLS, access control, and precise CORS origins before public deployment.
- Docker Compose is intended for local or trusted-network use, not as a ready-to-host public SaaS setup.
- The single-file app still starts a local web service; binding it to a public interface creates the same exposure risks.
- When an AI provider is configured, uploaded image content is sent to that provider. Use a self-hosted gateway for sensitive photos.
- `DATA_DIR` stores project state, media files, exports, and in-app AI settings. Protect that directory with appropriate local permissions and backups.

## Do Not Commit

- `.env`
- `data/`
- `dist/`
- `samples/demo-photos/`
- `images/`
- Real photos or private generated albums
- API keys, access tokens, or provider secrets

## Security Roadmap

- Explicit authentication and sharing permissions.
- Public deployment reverse-proxy examples.
- Stronger access control for exported share artifacts.
- Optional database and object-storage adapters.
- Frontend interaction and upload-boundary tests.
