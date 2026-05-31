package app

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"strings"
	"time"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

type githubClient struct {
	owner  string
	repo   string
	branch string
	token  string
}

type githubContentResponse struct {
	SHA     string `json:"sha"`
	Content string `json:"content"`
	Name    string `json:"name"`
	Path    string `json:"path"`
	Type    string `json:"type"`
}

type githubTreeEntry struct {
	Path string `json:"path"`
	Type string `json:"type"`
	SHA  string `json:"sha"`
}

type githubTreeResponse struct {
	Tree []githubTreeEntry `json:"tree"`
}

type githubPutRequest struct {
	Message string `json:"message"`
	Content string `json:"content"`
	Branch  string `json:"branch"`
	SHA     string `json:"sha,omitempty"`
}

type githubPutResponse struct {
	Content struct {
		SHA  string `json:"sha"`
		Path string `json:"path"`
	} `json:"content"`
	Commit struct {
		SHA string `json:"sha"`
	} `json:"commit"`
}

func newGitHubClient(owner, repo, branch, token string) *githubClient {
	return &githubClient{
		owner:  owner,
		repo:   repo,
		branch: branch,
		token:  token,
	}
}

func (g *githubClient) contentsURL(path string) string {
	return fmt.Sprintf("https://api.github.com/repos/%s/%s/contents/%s", g.owner, g.repo, path)
}

func (g *githubClient) treeURL(sha string) string {
	return fmt.Sprintf("https://api.github.com/repos/%s/%s/git/trees/%s", g.owner, g.repo, sha)
}

func (g *githubClient) doRequest(method, url string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+g.token)
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return http.DefaultClient.Do(req)
}

// getFileSHA returns the SHA of an existing file, or empty string if not found.
func (g *githubClient) getFileSHA(path string) (string, error) {
	url := g.contentsURL(path) + "?ref=" + g.branch
	resp, err := g.doRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return "", nil
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("GitHub API error %d: %s", resp.StatusCode, string(body))
	}

	var content githubContentResponse
	if err := json.NewDecoder(resp.Body).Decode(&content); err != nil {
		return "", err
	}
	return content.SHA, nil
}

// putFile creates or updates a file at the given path.
func (g *githubClient) putFile(path, message string, content []byte) error {
	sha, err := g.getFileSHA(path)
	if err != nil {
		return fmt.Errorf("check existing file: %w", err)
	}

	payload := githubPutRequest{
		Message: message,
		Content: base64.StdEncoding.EncodeToString(content),
		Branch:  g.branch,
		SHA:     sha,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	resp, err := g.doRequest("PUT", g.contentsURL(path), bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("GitHub API error %d: %s", resp.StatusCode, string(respBody))
	}
	return nil
}

// getAlbumDirs returns the list of album directory names under albums/.
func (g *githubClient) getAlbumDirs() ([]string, error) {
	url := g.contentsURL("albums") + "?ref=" + g.branch
	resp, err := g.doRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API error %d: %s", resp.StatusCode, string(body))
	}

	var entries []githubContentResponse
	if err := json.NewDecoder(resp.Body).Decode(&entries); err != nil {
		return nil, err
	}

	var dirs []string
	for _, e := range entries {
		if e.Type == "dir" {
			dirs = append(dirs, e.Name)
		}
	}
	return dirs, nil
}

// getAlbumTitle reads the first <title> tag from an album's index.html.
func (g *githubClient) getAlbumTitle(slug string) string {
	url := g.contentsURL("albums/"+slug+"/index.html") + "?ref=" + g.branch
	resp, err := g.doRequest("GET", url, nil)
	if err != nil {
		return slug
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return slug
	}

	var content githubContentResponse
	if err := json.NewDecoder(resp.Body).Decode(&content); err != nil {
		return slug
	}

	// Content is base64 encoded with newlines
	raw, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(content.Content, "\n", ""))
	if err != nil {
		return slug
	}

	htmlStr := string(raw)
	titleStart := strings.Index(htmlStr, "<title>")
	titleEnd := strings.Index(htmlStr, "</title>")
	if titleStart >= 0 && titleEnd > titleStart {
		title := htmlStr[titleStart+7 : titleEnd]
		title = strings.TrimSuffix(title, " - Memoir")
		return strings.TrimSpace(title)
	}
	return slug
}

// albumMeta holds metadata extracted from a published album's HTML.
type albumMeta struct {
	slug      string
	title     string
	intro     string
	coverPath string // relative path from repo root, e.g. "albums/slug/images/cover.jpg"
}

// extractCoverImage reads the first img src from the cover page section.
// The cover page has class "page-cover" and a "cover-image" div with an <img>.
func extractCoverImage(albumHTML string) string {
	// Find the cover page section
	coverStart := strings.Index(albumHTML, "page-cover")
	if coverStart < 0 {
		return ""
	}
	// Find the cover-image div
	coverDiv := strings.Index(albumHTML[coverStart:], "cover-image")
	if coverDiv < 0 {
		return ""
	}
	searchFrom := coverStart + coverDiv
	// Find the first <img src="..."> after cover-image
	imgTag := strings.Index(albumHTML[searchFrom:], "<img")
	if imgTag < 0 {
		return ""
	}
	imgStart := searchFrom + imgTag
	srcStart := strings.Index(albumHTML[imgStart:], "src=\"")
	if srcStart < 0 {
		return ""
	}
	srcStart += imgStart + 5 // skip past src="
	srcEnd := strings.Index(albumHTML[srcStart:], "\"")
	if srcEnd < 0 {
		return ""
	}
	return albumHTML[srcStart : srcStart+srcEnd]
}

// extractAlbumIntro reads the intro paragraph from the cover page section.
// The cover page has a "cover-copy" div with a <p> after the <h1> title.
func extractAlbumIntro(albumHTML string) string {
	coverStart := strings.Index(albumHTML, "page-cover")
	if coverStart < 0 {
		return ""
	}
	copyStart := strings.Index(albumHTML[coverStart:], "cover-copy")
	if copyStart < 0 {
		return ""
	}
	searchFrom := coverStart + copyStart
	// Skip the <h1>...</h1> tag
	h1End := strings.Index(albumHTML[searchFrom:], "</h1>")
	if h1End < 0 {
		return ""
	}
	searchFrom += h1End + 5
	// Find the next <p>...</p>
	pStart := strings.Index(albumHTML[searchFrom:], "<p>")
	if pStart < 0 {
		return ""
	}
	searchFrom += pStart + 3
	pEnd := strings.Index(albumHTML[searchFrom:], "</p>")
	if pEnd < 0 {
		return ""
	}
	intro := albumHTML[searchFrom : searchFrom+pEnd]
	// Strip any inner HTML tags
	intro = stripHTMLTags(intro)
	if len([]rune(intro)) > 120 {
		intro = string([]rune(intro)[:120]) + "…"
	}
	return strings.TrimSpace(intro)
}

// stripHTMLTags removes HTML tags from a string, keeping only text content.
func stripHTMLTags(s string) string {
	var b strings.Builder
	inTag := false
	for _, r := range s {
		if r == '<' {
			inTag = true
			continue
		}
		if r == '>' {
			inTag = false
			continue
		}
		if !inTag {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// readAlbumHTML fetches an album's index.html and returns its content.
func (g *githubClient) readAlbumHTML(slug string) (string, error) {
	url := g.contentsURL("albums/"+slug+"/index.html") + "?ref=" + g.branch
	resp, err := g.doRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	var content githubContentResponse
	if err := json.NewDecoder(resp.Body).Decode(&content); err != nil {
		return "", err
	}

	raw, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(content.Content, "\n", ""))
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

// extractAlbumMeta reads an album's HTML to get title, intro, and cover image path.
func (g *githubClient) extractAlbumMeta(slug string) albumMeta {
	meta := albumMeta{slug: slug, title: slug}

	htmlStr, err := g.readAlbumHTML(slug)
	if err != nil {
		return meta
	}

	// Title from <title> tag
	titleStart := strings.Index(htmlStr, "<title>")
	titleEnd := strings.Index(htmlStr, "</title>")
	if titleStart >= 0 && titleEnd > titleStart {
		title := htmlStr[titleStart+7 : titleEnd]
		title = strings.TrimSuffix(title, " - Memoir")
		if trimmed := strings.TrimSpace(title); trimmed != "" {
			meta.title = trimmed
		}
	}

	meta.intro = extractAlbumIntro(htmlStr)

	// Cover image: relative to album dir (e.g. "images/xxx.jpg"), convert to repo-relative
	coverSrc := extractCoverImage(htmlStr)
	if coverSrc != "" {
		meta.coverPath = "albums/" + slug + "/" + coverSrc
	}

	return meta
}

// buildAlbumListingHTML generates a root index.html that lists all published albums.
func (g *githubClient) buildAlbumListingHTML() (string, error) {
	dirs, err := g.getAlbumDirs()
	if err != nil {
		return "", fmt.Errorf("list albums: %w", err)
	}

	var albums []albumMeta
	for _, dir := range dirs {
		albums = append(albums, g.extractAlbumMeta(dir))
	}

	var b strings.Builder
	b.WriteString(`<!doctype html>
<html lang="zh-CN">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Memoir - 我的相册集</title>
<style>
  @import url('https://fonts.googleapis.com/css2?family=Noto+Serif+SC:wght@400;600&family=Inter:wght@300;400;500;600&display=swap');

  :root {
    --bg: #faf7f2;
    --bg-card: #ffffff;
    --text: #2d2420;
    --text-soft: #6b5d50;
    --text-muted: #9b8b7b;
    --accent: #8b2635;
    --accent-hover: #6d1e2a;
    --border: #e8e0d4;
    --shadow: rgba(45, 36, 32, 0.08);
    --serif: 'Noto Serif SC', 'Source Han Serif SC', 'Songti SC', Georgia, serif;
    --sans: 'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', system-ui, sans-serif;
  }

  * { margin: 0; padding: 0; box-sizing: border-box; }

  body {
    font-family: var(--sans);
    background: var(--bg);
    color: var(--text);
    min-height: 100vh;
    -webkit-font-smoothing: antialiased;
  }

  .page {
    max-width: 960px;
    margin: 0 auto;
    padding: 4rem 2rem 3rem;
  }

  /* Header */
  .site-header {
    text-align: center;
    margin-bottom: 3.5rem;
  }

  .site-header h1 {
    font-family: var(--serif);
    font-size: 2.4rem;
    font-weight: 600;
    letter-spacing: -0.02em;
    color: var(--text);
    margin-bottom: 0.4rem;
  }

  .site-divider {
    width: 40px;
    height: 2px;
    background: var(--accent);
    margin: 1rem auto;
    border-radius: 1px;
  }

  .site-header .subtitle {
    color: var(--text-muted);
    font-size: 0.95rem;
    font-weight: 300;
    letter-spacing: 0.02em;
  }

  /* Album count badge */
  .album-count {
    display: inline-flex;
    align-items: center;
    gap: 0.4rem;
    padding: 0.35rem 0.9rem;
    background: var(--accent);
    color: white;
    border-radius: 100px;
    font-size: 0.8rem;
    font-weight: 500;
    margin-top: 1rem;
  }

  /* Grid */
  .album-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
    gap: 1.8rem;
  }

  /* Album card */
  .album-card {
    display: flex;
    flex-direction: column;
    background: var(--bg-card);
    border: 1px solid var(--border);
    border-radius: 14px;
    overflow: hidden;
    text-decoration: none;
    color: inherit;
    transition: transform 0.25s cubic-bezier(0.4, 0, 0.2, 1),
                box-shadow 0.25s cubic-bezier(0.4, 0, 0.2, 1),
                border-color 0.25s ease;
  }

  .album-card:hover {
    transform: translateY(-4px);
    box-shadow: 0 12px 32px var(--shadow);
    border-color: var(--accent);
  }

  /* Cover image */
  .album-cover {
    position: relative;
    width: 100%;
    padding-top: 65%;
    background: linear-gradient(135deg, #f0ebe4, #e8e0d4);
    overflow: hidden;
  }

  .album-cover img {
    position: absolute;
    top: 0;
    left: 0;
    width: 100%;
    height: 100%;
    object-fit: cover;
    transition: transform 0.4s cubic-bezier(0.4, 0, 0.2, 1);
  }

  .album-card:hover .album-cover img {
    transform: scale(1.04);
  }

  .album-cover::after {
    content: '';
    position: absolute;
    bottom: 0;
    left: 0;
    right: 0;
    height: 40%;
    background: linear-gradient(transparent, rgba(0,0,0,0.12));
    pointer-events: none;
  }

  /* Placeholder when no cover image */
  .album-cover--empty {
    display: flex;
    align-items: center;
    justify-content: center;
    position: absolute;
    top: 0;
    left: 0;
    width: 100%;
    height: 100%;
  }

  .album-cover--empty svg {
    width: 48px;
    height: 48px;
    color: var(--text-muted);
    opacity: 0.4;
  }

  /* Card body */
  .album-body {
    padding: 1.2rem 1.3rem 1.4rem;
    flex: 1;
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }

  .album-title {
    font-family: var(--serif);
    font-size: 1.15rem;
    font-weight: 600;
    line-height: 1.4;
    color: var(--text);
  }

  .album-intro {
    font-size: 0.85rem;
    color: var(--text-soft);
    line-height: 1.6;
    flex: 1;
  }

  .album-footer {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-top: 0.6rem;
    padding-top: 0.7rem;
    border-top: 1px solid var(--border);
  }

  .album-footer span {
    font-size: 0.75rem;
    color: var(--text-muted);
  }

  .album-arrow {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 28px;
    height: 28px;
    border-radius: 50%;
    background: var(--bg);
    color: var(--text-muted);
    transition: background 0.2s, color 0.2s;
  }

  .album-card:hover .album-arrow {
    background: var(--accent);
    color: white;
  }

  /* Empty state */
  .empty-state {
    text-align: center;
    padding: 4rem 2rem;
  }

  .empty-icon {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 80px;
    height: 80px;
    border-radius: 50%;
    background: var(--bg-card);
    border: 2px dashed var(--border);
    color: var(--text-muted);
    margin-bottom: 1.5rem;
  }

  .empty-state h2 {
    font-family: var(--serif);
    font-size: 1.3rem;
    font-weight: 500;
    margin-bottom: 0.5rem;
  }

  .empty-state p {
    color: var(--text-muted);
    font-size: 0.9rem;
  }

  /* Footer */
  .site-footer {
    text-align: center;
    margin-top: 4rem;
    padding-top: 2rem;
    border-top: 1px solid var(--border);
    color: var(--text-muted);
    font-size: 0.8rem;
  }

  .site-footer a {
    color: var(--accent);
    text-decoration: none;
  }

  .site-footer a:hover {
    text-decoration: underline;
  }

  /* Responsive */
  @media (max-width: 640px) {
    .page { padding: 2.5rem 1.2rem 2rem; }
    .site-header h1 { font-size: 1.8rem; }
    .album-grid { grid-template-columns: 1fr; gap: 1.2rem; }
    .album-cover { padding-top: 56%; }
  }

  @media (min-width: 641px) and (max-width: 960px) {
    .album-grid { grid-template-columns: repeat(2, 1fr); }
  }
</style>
</head>
<body>
<div class="page">
`)

	b.WriteString("  <header class=\"site-header\">\n")
	b.WriteString("    <h1>Memoir</h1>\n")
	b.WriteString("    <div class=\"site-divider\"></div>\n")
	b.WriteString("    <p class=\"subtitle\">我的照片故事集</p>\n")
	if len(albums) > 0 {
		b.WriteString(fmt.Sprintf("    <div class=\"album-count\">%d 本相册</div>\n", len(albums)))
	}
	b.WriteString("  </header>\n\n")

	if len(albums) == 0 {
		b.WriteString(`  <div class="empty-state">
    <div class="empty-icon">
      <svg width="36" height="36" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
        <rect x="3" y="3" width="18" height="18" rx="2" ry="2"/>
        <circle cx="8.5" cy="8.5" r="1.5"/>
        <polyline points="21 15 16 10 5 21"/>
      </svg>
    </div>
    <h2>还没有发布的相册</h2>
    <p>从 Memoir 发布你的第一本相册，它就会出现在这里。</p>
  </div>
`)
	} else {
		b.WriteString("  <div class=\"album-grid\">\n")
		for _, a := range albums {
			b.WriteString("    <a class=\"album-card\" href=\"albums/")
			b.WriteString(html.EscapeString(a.slug))
			b.WriteString("/\">\n")

			// Cover image
			if a.coverPath != "" {
				b.WriteString("      <div class=\"album-cover\">\n")
				b.WriteString("        <img src=\"")
				b.WriteString(html.EscapeString(a.coverPath))
				b.WriteString("\" alt=\"")
				b.WriteString(html.EscapeString(a.title))
				b.WriteString("\" loading=\"lazy\">\n")
				b.WriteString("      </div>\n")
			} else {
				b.WriteString(`      <div class="album-cover">
        <div class="album-cover--empty">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
            <rect x="3" y="3" width="18" height="18" rx="2" ry="2"/>
            <circle cx="8.5" cy="8.5" r="1.5"/>
            <polyline points="21 15 16 10 5 21"/>
          </svg>
        </div>
      </div>
`)
			}

			b.WriteString("      <div class=\"album-body\">\n")
			b.WriteString("        <h2 class=\"album-title\">")
			b.WriteString(html.EscapeString(a.title))
			b.WriteString("</h2>\n")
			if a.intro != "" {
				b.WriteString("        <p class=\"album-intro\">")
				b.WriteString(html.EscapeString(a.intro))
				b.WriteString("</p>\n")
			}
			b.WriteString("        <div class=\"album-footer\">\n")
			b.WriteString("          <span>查看相册</span>\n")
			b.WriteString("          <span class=\"album-arrow\">\n")
			b.WriteString("            <svg width=\"14\" height=\"14\" viewBox=\"0 0 24 24\" fill=\"none\" stroke=\"currentColor\" stroke-width=\"2\" stroke-linecap=\"round\" stroke-linejoin=\"round\"><line x1=\"5\" y1=\"12\" x2=\"19\" y2=\"12\"/><polyline points=\"12 5 19 12 12 19\"/></svg>\n")
			b.WriteString("          </span>\n")
			b.WriteString("        </div>\n")
			b.WriteString("      </div>\n")
			b.WriteString("    </a>\n")
		}
		b.WriteString("  </div>\n")
	}

	b.WriteString("\n  <footer class=\"site-footer\">\n")
	b.WriteString("    <p>由 <a href=\"https://github.com\" target=\"_blank\">Memoir</a> 生成 · ")
	b.WriteString(time.Now().Format("2006-01-02"))
	b.WriteString("</p>\n")
	b.WriteString("  </footer>\n")
	b.WriteString("</div>\n</body>\n</html>")

	return b.String(), nil
}

// publishAlbumListing regenerates and uploads the root index.html.
func (g *githubClient) publishAlbumListing() error {
	listingHTML, err := g.buildAlbumListingHTML()
	if err != nil {
		return err
	}
	return g.putFile("index.html", "Update album listing", []byte(listingHTML))
}

// slugify converts a string to a URL-safe slug.
// For CJK characters, uses the original characters; for others, lowercases and replaces spaces/special chars with hyphens.
func slugify(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "album"
	}
	s = norm.NFC.String(s)
	var b strings.Builder
	prevDash := false
	for _, r := range s {
		switch {
		case unicode.Is(unicode.Han, r):
			b.WriteRune(r)
			prevDash = false
		case r == '-' || r == '_':
			if !prevDash && b.Len() > 0 {
				b.WriteRune('-')
				prevDash = true
			}
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(unicode.ToLower(r))
			prevDash = false
		default:
			if !prevDash && b.Len() > 0 {
				b.WriteRune('-')
				prevDash = true
			}
		}
	}
	result := strings.Trim(b.String(), "-")
	if result == "" {
		return "album"
	}
	// Limit length
	if len([]rune(result)) > 60 {
		result = string([]rune(result)[:60])
	}
	return result
}
