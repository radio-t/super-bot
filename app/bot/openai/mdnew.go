package openai

import (
	"fmt"
	"io"
	"net/http"
	"strings"
)

// MarkdownNewClient extracts page content using markdown.new service.
// It implements the uKeeperGetter interface as a drop-in replacement for UKeeperClient.
type MarkdownNewClient struct {
	Client *http.Client
	API    string // base URL, e.g. "https://markdown.new"
}

// Get returns title and markdown content for the given link using markdown.new.
// markdown.new response format:
//
//	Title: <title>
//	URL Source: <url>
//	Markdown Content:
//	---
//	title: <title>
//	---
//	<markdown body>
func (m MarkdownNewClient) Get(link string) (title, content string, err error) {
	reqURL := strings.TrimSuffix(m.API, "/") + "/" + link
	resp, err := m.Client.Get(reqURL)
	if err != nil {
		return "", "", fmt.Errorf("can't get markdown for %s: %w", link, err)
	}
	defer resp.Body.Close() //nolint

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("can't get markdown for %s: status %d", link, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("can't read markdown response for %s: %w", link, err)
	}

	title, content = parseMarkdownNewResponse(string(body))
	if content == "" {
		return "", "", fmt.Errorf("empty content from markdown.new for %s", link)
	}

	return title, content, nil
}

// parseMarkdownNewResponse parses the markdown.new response format.
// It extracts title from the "Title:" header line, and markdown body from after "Markdown Content:".
// If "Markdown Content:" section contains YAML frontmatter, it is stripped from the body.
// Falls back to frontmatter title or first # heading if "Title:" header is missing.
func parseMarkdownNewResponse(raw string) (title, body string) {
	raw = strings.TrimSpace(raw)

	// look for "Title:" header at the top
	headerTitle := extractHeaderField(raw, "Title:")

	// look for "Markdown Content:" separator
	mdContent := raw
	if idx := strings.Index(raw, "Markdown Content:"); idx >= 0 {
		mdContent = strings.TrimSpace(raw[idx+len("Markdown Content:"):])
	}

	// strip YAML frontmatter from markdown content if present
	fmTitle, mdBody := stripFrontmatter(mdContent)
	if mdBody != "" {
		mdContent = mdBody
	}

	// title priority: header "Title:" > frontmatter title > first # heading
	title = headerTitle
	if title == "" {
		title = fmTitle
	}
	if title == "" {
		title = extractHeadingTitle(mdContent)
	}

	return title, mdContent
}

// extractHeaderField extracts value from a "Key: value" line at the beginning of text
func extractHeaderField(text, key string) string {
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, key) {
			return strings.TrimSpace(strings.TrimPrefix(line, key))
		}
		if line == "" {
			continue
		}
		// stop at first non-header, non-empty line that isn't our key
		if !strings.Contains(line, ":") {
			break
		}
	}
	return ""
}

// stripFrontmatter removes YAML frontmatter (---\n...\n---) and returns frontmatter title + body
func stripFrontmatter(text string) (title, body string) {
	text = strings.TrimSpace(text)
	if !strings.HasPrefix(text, "---") {
		return "", text
	}

	end := strings.Index(text[3:], "\n---")
	if end < 0 {
		return "", text
	}

	frontmatter := text[3 : 3+end]
	body = strings.TrimSpace(text[3+end+4:])

	for _, line := range strings.Split(frontmatter, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "title:") {
			title = strings.TrimSpace(strings.TrimPrefix(line, "title:"))
			title = strings.Trim(title, `"'`)
			break
		}
	}

	return title, body
}

// extractHeadingTitle extracts title from first # heading in markdown
func extractHeadingTitle(md string) string {
	for _, line := range strings.Split(md, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "# ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "# "))
		}
	}
	return ""
}
