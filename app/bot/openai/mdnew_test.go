package openai

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMarkdownNewClient_Get(t *testing.T) {
	tbl := []struct {
		name            string
		response        string
		statusCode      int
		expectedTitle   string
		expectedContent string
		err             string
	}{
		{
			name: "real markdown.new format with frontmatter",
			response: "Title: Test Article\n\nURL Source: https://example.com\n\nMarkdown Content:\n" +
				"---\ntitle: Test Article\ndescription: some desc\n---\n\n# Test Article\n\nSome content here.",
			statusCode:      http.StatusOK,
			expectedTitle:   "Test Article",
			expectedContent: "# Test Article\n\nSome content here.",
		},
		{
			name: "markdown.new format without frontmatter",
			response: "Title: Page Title\n\nURL Source: https://example.com\n\nMarkdown Content:\n" +
				"# Page Title\n\nBody text.",
			statusCode:      http.StatusOK,
			expectedTitle:   "Page Title",
			expectedContent: "# Page Title\n\nBody text.",
		},
		{
			name:            "plain markdown without headers",
			response:        "# Direct Heading\n\nParagraph content.",
			statusCode:      http.StatusOK,
			expectedTitle:   "Direct Heading",
			expectedContent: "# Direct Heading\n\nParagraph content.",
		},
		{
			name: "frontmatter title differs from header title",
			response: "Title: Header Title\n\nMarkdown Content:\n" +
				"---\ntitle: FM Title\n---\n\nBody.",
			statusCode:      http.StatusOK,
			expectedTitle:   "Header Title",
			expectedContent: "Body.",
		},
		{
			name:       "completely empty response",
			response:   "",
			statusCode: http.StatusOK,
			err:        "empty content from markdown.new for http://example.com",
		},
		{
			name:       "server error",
			response:   "error",
			statusCode: http.StatusInternalServerError,
			err:        "can't get markdown for http://example.com: status 500",
		},
	}

	for _, tt := range tbl {
		t.Run(tt.name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodGet, r.Method)
				assert.Equal(t, "/http://example.com", r.URL.Path)
				w.WriteHeader(tt.statusCode)
				_, err := w.Write([]byte(tt.response))
				require.NoError(t, err)
			}))
			defer ts.Close()

			mc := MarkdownNewClient{
				API:    ts.URL,
				Client: ts.Client(),
			}

			title, content, err := mc.Get("http://example.com")
			if tt.err == "" {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedTitle, title)
				assert.Equal(t, tt.expectedContent, content)
			} else {
				require.Error(t, err)
				assert.Equal(t, tt.err, err.Error())
			}
		})
	}
}

func TestParseMarkdownNewResponse(t *testing.T) {
	tbl := []struct {
		name          string
		input         string
		expectedTitle string
		expectedBody  string
	}{
		{
			name: "full markdown.new response",
			input: "Title: My Article\n\nURL Source: https://example.com\n\nMarkdown Content:\n" +
				"---\ntitle: My Article\ndescription: desc\nimage: img.png\n---\n\n# My Article\n\nContent here.",
			expectedTitle: "My Article",
			expectedBody:  "# My Article\n\nContent here.",
		},
		{
			name: "quoted frontmatter title, no header title",
			input: "Markdown Content:\n---\ntitle: \"Quoted Title\"\n---\n\nBody text.",
			expectedTitle: "Quoted Title",
			expectedBody:  "Body text.",
		},
		{
			name:          "plain markdown, no headers at all",
			input:         "# Direct Heading\n\nParagraph.",
			expectedTitle: "Direct Heading",
			expectedBody:  "# Direct Heading\n\nParagraph.",
		},
		{
			name:          "no title anywhere",
			input:         "Markdown Content:\nJust plain text without any structure.",
			expectedTitle: "",
			expectedBody:  "Just plain text without any structure.",
		},
		{
			name: "header title takes precedence over frontmatter",
			input: "Title: From Header\n\nMarkdown Content:\n" +
				"---\ntitle: From Frontmatter\n---\n\nBody.",
			expectedTitle: "From Header",
			expectedBody:  "Body.",
		},
		{
			name: "single-quoted frontmatter title",
			input: "Markdown Content:\n---\ntitle: 'Single Quoted'\n---\n\nBody.",
			expectedTitle: "Single Quoted",
			expectedBody:  "Body.",
		},
	}

	for _, tt := range tbl {
		t.Run(tt.name, func(t *testing.T) {
			title, body := parseMarkdownNewResponse(tt.input)
			assert.Equal(t, tt.expectedTitle, title)
			assert.Equal(t, tt.expectedBody, body)
		})
	}
}

func TestStripFrontmatter(t *testing.T) {
	tbl := []struct {
		name          string
		input         string
		expectedTitle string
		expectedBody  string
	}{
		{
			name:          "with frontmatter",
			input:         "---\ntitle: FM Title\nauthor: test\n---\nBody content",
			expectedTitle: "FM Title",
			expectedBody:  "Body content",
		},
		{
			name:          "no frontmatter",
			input:         "Just text",
			expectedTitle: "",
			expectedBody:  "Just text",
		},
		{
			name:          "frontmatter without title",
			input:         "---\nauthor: test\n---\nBody",
			expectedTitle: "",
			expectedBody:  "Body",
		},
		{
			name:          "incomplete frontmatter (no closing)",
			input:         "---\ntitle: Broken\nBody text",
			expectedTitle: "",
			expectedBody:  "---\ntitle: Broken\nBody text",
		},
	}

	for _, tt := range tbl {
		t.Run(tt.name, func(t *testing.T) {
			title, body := stripFrontmatter(tt.input)
			assert.Equal(t, tt.expectedTitle, title)
			assert.Equal(t, tt.expectedBody, body)
		})
	}
}
