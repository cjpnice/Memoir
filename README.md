# Memoir / 集忆

[English](README.md) | [中文文档](README.zh-CN.md)

**Memoir** is a local-first AI-powered photo album creator. Import your travel, family, or everyday photos, let AI analyze and curate them, then generate beautiful narrative albums you can export as HTML or publish directly to GitHub Pages.

Built for people who have thousands of meaningful photos sitting in folders—the images matter, but the work of choosing, sequencing, and sharing them is easy to postpone forever.

## Why Memoir?

Taking photos is effortless. Revisiting them is not. A single weekend can leave behind hundreds of frames: near-duplicates, blurred candids, and small details that may feel ordinary today but become precious years later.

Memoir asks: **if AI can understand images, can it help preserve memory instead of just generating more content?**

Three principles guide the project:

- **Local first** — Your photos, project state, and exports stay on your machine by default
- **Human in the loop** — AI recommends and drafts; you decide, edit, and publish
- **Narrative output** — Albums with cover, pacing, captions, and closing notes, not just photo grids

## Features

### Core Workflow

- **Smart Import** — Upload JPG, PNG, HEIC, and HEIF files with real-time progress tracking
- **AI Analysis** — Quality, preservation, and story scores with recommendations, issues, crop suggestions, and caption seeds
- **Intelligent Grouping** — Automatically cluster similar shots and recommend the best representative frame
- **Human Review** — Review AI decisions, delete images, undo edits, and override recommendations
- **Narrative Album Generation** — Create editable albums with pages, titles, body copy, image placement, and social post drafts
- **Album Editing** — Reorder pages, undo/redo changes, and refine the narrative flow

### Export & Publishing

- **HTML Export** — Standalone interactive albums with photo zoom and responsive layout
- **Long Image Export** — Single PNG for easy sharing on messaging apps
- **Share Links** — Local web server URLs for quick previews
- **GitHub Pages Publishing** — Publish albums directly to GitHub Pages with:
  - Automatic album cover and metadata extraction
  - Beautiful landing page listing all published albums
  - Real-time upload progress tracking
  - Auto-generated album homepage that updates with each publish

### Additional Features

- **Memory Map** — Visualize projects on an interactive map using EXIF location data
- **Social Post Generation** — AI-generated captions for Xiaohongshu, WeChat Moments, and other platforms with recommended image sets
- **Settings Persistence** — AI model and GitHub Pages configuration saved locally
- **Project Organization** — Multiple projects with titles, descriptions, locations, and visual themes

## Tech Stack

- **Frontend**: Next.js 16, React 19, TypeScript, lucide-react, react-map-gl
- **Backend**: Go 1.26, Gin web framework
- **AI**: Eino framework, OpenAI-compatible multimodal models
- **Storage**: Local JSON files and media directory (no external database)
- **Media Processing**: Go image libraries, HEIC/HEIF conversion, Chrome-based PDF rendering
- **Deployment**: Docker Compose, single-file Go binaries with embedded frontend

## Quick Start

### Prerequisites

- Go 1.26+
- Node.js 22+
- npm

### Development Setup

1. **Install dependencies**:
   ```bash
   make setup
   ```

2. **Configure environment**:
   ```bash
   cp .env.example .env
   ```
   
   Edit `.env` and set your OpenAI-compatible API credentials:
   ```bash
   OPENAI_BASE_URL=https://api.openai.com/v1
   OPENAI_API_KEY=your-api-key-here
   OPENAI_MODEL=gpt-4o-mini
   ```

3. **Start the API server** (in one terminal):
   ```bash
   make dev-api
   ```

4. **Start the web frontend** (in another terminal):
   ```bash
   make dev-web
   ```

5. **Open the app**:
   ```
   http://localhost:3000
   ```

The frontend calls the Go API at `http://127.0.0.1:8090` by default.

### Docker Setup

```bash
cp .env.example .env
# Edit .env with your API credentials
make docker-up
```

Services:
- Web: `http://localhost:3000`
- API: `http://localhost:8090`

### Single-File Release

Build a standalone binary with the frontend embedded:

```bash
make package
```

Release binaries are written to `dist/`. Run the matching binary to start a local server.

## Usage Workflow

### 1. Create a Project

Click "新建项目" on the homepage. Set a title, description, location (optional), tone, and visual theme.

### 2. Import Photos

Upload JPG, PNG, HEIC, or HEIF files. The app shows real-time upload progress and recent imports.

### 3. AI Analysis

Click "开始分析" to let the AI model:
- Score each photo for quality, preservation value, and story potential
- Identify issues (blur, exposure, composition)
- Suggest crops and improvements
- Group similar shots
- Generate caption seeds

### 4. Review Decisions

Navigate to the "审核" tab to:
- See AI recommendations for each image
- Override decisions (keep, exclude, improve then keep)
- Delete unwanted photos
- Filter and sort by various criteria

### 5. Generate Album

Click "生成相册" to create a narrative album with:
- Cover page with title and intro
- Themed pages with titles, body text, and image placement
- Closing notes
- Social media post drafts

### 6. Edit Album

In the "相册" tab:
- Reorder pages by drag-and-drop
- Edit page titles and body text
- Adjust image placement
- Undo/redo changes

### 7. Export

Choose an export format in the "导出" tab:

- **HTML** — Interactive web album with photo zoom
- **Long Image** — Single PNG for messaging apps
- **Share Link** — Local preview URL
- **GitHub Pages** — Publish online (requires setup)

Export results are persisted—refreshing the page won't lose them.

## GitHub Pages Publishing

Publish albums to GitHub Pages for online access and sharing.

### Setup

1. **Create a GitHub repository**:
   - Go to [github.com/new](https://github.com/new)
   - Create a public repository (e.g., `my-albums`)
   - Initialize with a README

2. **Generate a Personal Access Token**:
   - Go to [GitHub Token Settings](https://github.com/settings/tokens/new)
   - Select the `repo` scope
   - Generate and copy the token (starts with `ghp_`)

3. **Configure in Memoir**:
   - Click the ⚙ settings button on the homepage
   - Fill in the GitHub Pages card:
     - **Owner**: Your GitHub username
     - **Repo**: Repository name (e.g., `my-albums`)
     - **Branch**: `main` (default)
     - **Token**: Your personal access token
   - Click "保存 GitHub 设置"

4. **Enable GitHub Pages** (one-time):
   - Go to your repository's Settings → Pages
   - Source: "Deploy from a branch"
   - Branch: `main`, folder: `/ (root)`
   - Save

### Publishing

1. Open a project with a generated album
2. Go to the "导出" tab
3. Select "GitHub Pages" export type
4. Click "发布到 GitHub Pages"
5. Watch real-time progress as images and HTML are uploaded
6. Copy the published URL when complete

Your album will be available at:
```
https://{username}.github.io/{repo}/albums/{album-slug}/
```

### Album Homepage

Memoir automatically creates and maintains a landing page at the repository root:
```
https://{username}.github.io/{repo}/
```

The homepage displays:
- Grid of album cards with cover images
- Album titles and intro text
- Publication dates
- Responsive layout for mobile and desktop

The homepage updates automatically when you publish new albums. You can also manually refresh it from the settings page.

### Manual Refresh

If you delete albums or need to regenerate the homepage:
1. Open settings (⚙ button on homepage)
2. Click "刷新相册首页" in the GitHub Pages section

## Configuration

### Environment Variables

Copy `.env.example` to `.env` and configure:

```bash
# Server
PORT=8090
DATA_DIR=./data

# CORS (empty for permissive local dev)
ALLOWED_ORIGINS=

# Upload limits
MAX_UPLOAD_MB=256
MAX_UPLOAD_FILES=200

# Frontend API endpoint
NEXT_PUBLIC_API_BASE_URL=http://127.0.0.1:8090

# OpenAI-compatible multimodal model (required for AI features)
OPENAI_BASE_URL=
OPENAI_API_KEY=
OPENAI_MODEL=gpt-4o-mini

# Optional: separate image editing model
OPENAI_IMAGE_BASE_URL=
OPENAI_IMAGE_API_KEY=
OPENAI_IMAGE_MODEL=gpt-image-1.5
```

### AI Model Requirements

Memoir works with any OpenAI-compatible multimodal model that supports image input. Popular options:

- OpenAI GPT-4o / GPT-4o-mini
- Anthropic Claude (via compatible proxy)
- Local models with OpenAI-compatible API

Without an API key, the app starts normally but analysis and album generation return configuration errors.

### Settings Persistence

AI model and GitHub Pages settings are saved to:
```
{DATA_DIR}/ai-settings.json
{DATA_DIR}/github-settings.json
```

Settings configured in the web UI override environment variables.

## Project Structure

```
Memoir/
├── apps/web/              # Next.js frontend
│   ├── app/               # App router pages
│   ├── components/        # React components
│   │   └── dashboard/     # Main workspace views
│   ├── lib/               # Utilities and API client
│   └── public/            # Static assets
├── cmd/
│   ├── api/               # Development API entry point
│   └── memoir/            # Single-file release entry point
├── internal/
│   ├── ai/                # AI analyzer and model adapters
│   ├── app/               # Service layer and business logic
│   ├── config/            # Configuration management
│   ├── domain/            # Domain models (Project, Album, Image)
│   ├── httpapi/           # HTTP handlers and routing
│   ├── media/             # Image processing and storage
│   ├── store/             # JSON file persistence
│   └── webassets/         # Embedded frontend assets
├── scripts/               # Build and release scripts
├── compose.yaml           # Docker Compose configuration
└── Makefile               # Development commands
```

## Common Commands

```bash
make setup        # Install frontend dependencies
make dev          # Run API and web together
make dev-api      # Run the Go API server
make dev-web      # Run Next.js dev server
make test         # Run tests and build checks
make build        # Build Go packages and web app
make package      # Create single-file release binaries
make fmt          # Format Go code
make clean        # Remove data/ and build artifacts
make docker-up    # Start Docker Compose services
make docker-down  # Stop Docker Compose services
```

## AI and Privacy

- **Local by default** — All photos, project data, and exports stay on your machine
- **API calls** — Only image analysis requests are sent to your configured AI provider
- **No telemetry** — Memoir does not collect usage data or analytics
- **Your control** — You choose which photos to analyze and which to publish

Do not commit `.env`, `data/`, private photos, API keys, or personal albums to version control.

## Contributing

Memoir is pre-1.0 and actively developed. Areas where help is welcome:

- **Testing** — Add frontend interaction tests and API integration tests
- **Export formats** — PDF, EPUB, print-ready layouts
- **Album themes** — Additional visual themes and page layouts
- **AI providers** — Support for more multimodal model providers
- **Deployment** — Kubernetes manifests, security hardening, self-hosting guides
- **Documentation** — Tutorials, video guides, and troubleshooting

Please read [CONTRIBUTING.md](CONTRIBUTING.md) and [SECURITY.md](SECURITY.md) before contributing.

## License

MIT License. See [LICENSE](LICENSE).

## Acknowledgments

- Built with [Next.js](https://nextjs.org/), [React](https://react.dev/), and [Go](https://go.dev/)
- AI integration via [Eino](https://github.com/cloudwego/eino) framework
- Icons by [Lucide](https://lucide.dev/)
- Maps powered by [MapLibre GL](https://maplibre.org/)

---

**Made with ❤️ for preserving meaningful memories**
