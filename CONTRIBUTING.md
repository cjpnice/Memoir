# Contributing

[中文](CONTRIBUTING.zh-CN.md)

Thank you for helping Memoir / 集忆. The project is pre-1.0, and the main goal is to make the local-first AI memory-curation workflow reliable, explainable, and reproducible.

## Local Development

You need Go 1.26+, Node.js 22+, and npm.

```bash
make setup
cp .env.example .env
make dev-api
make dev-web
```

You can develop without an API key. The app starts normally; analysis and album generation return an error prompting you to configure one. Upload, project management, and export flows work offline.

## Verification

Before opening a PR, run:

```bash
make test
```

This runs the web dialog policy check, Go tests, and the frontend production build. For documentation-only changes, mention the link and command checks you performed.

If your change affects release packaging, also run:

```bash
make package
```

## Code Boundaries

- Go API entrypoint: `cmd/api`
- Single-file app entrypoint: `cmd/memoir`
- Core workflows: `internal/app`
- AI interfaces and adapters: `internal/ai`
- HTTP routes and CORS behavior: `internal/httpapi`
- Uploads, thumbnails, image edits, and export media: `internal/media`
- Frontend app: `apps/web`

Prefer small, focused changes. When touching AI behavior, exports, uploads, security boundaries, or persisted data structures, add tests or document manual verification clearly.

## Documentation Rules

User-, contributor-, and deployment-facing docs should have Chinese and English companion files:

- Root docs: `README.md` (Chinese, default) and `README.en.md` (English)
- English docs: `docs/*.md`
- Chinese docs: `docs/zh-CN/*.md`

The two languages do not need to be literal translations, but they must describe the same facts, constraints, and commands.

## Privacy Rules

Do not commit:

- `.env` or any API key.
- Runtime state under `data/`.
- Real photos, private album exports, or share pages.
- Local media folders such as `samples/demo-photos/` or `images/`.
- Release artifacts under `dist/`.
- `apps/web/node_modules/`, `.next/`, or `out/`.

Use authorized media for screenshots and videos, and strip EXIF/GPS metadata before publishing final assets.

## Good First Areas

- Add frontend interaction tests for upload, review, album editing, and export flows.
- Add album themes and export templates.
- Strengthen Docker, self-hosting, reverse-proxy, and single-file app docs.
- Review AI provider compatibility, error messages, and privacy guidance.

## Pull Requests

Please include:

- What changed.
- Why it is needed.
- How you verified it.
- Whether it affects photo privacy, AI providers, export formats, single-file packaging, or public deployment.

If an issue involves private photos, reproduce it with sanitized or publishable media instead.
