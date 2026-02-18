package epub

import (
	"bytes"
	"strings"
)

// gutenbergPatterns contains case-insensitive patterns that indicate a
// Project Gutenberg license page.
var gutenbergPatterns = []string{
	"project gutenberg license",
	"gutenberg.org/license",
	"start of the project gutenberg license",
	"end of the project gutenberg license",
	"start of this project gutenberg ebook",
	"end of this project gutenberg ebook",
}

// gutenbergComboPatterns contains pairs of strings that together indicate a
// Gutenberg license page (both must appear, case-insensitive).
var gutenbergComboPatterns = [][2]string{
	{"project gutenberg", "terms of use"},
	{"full license", "gutenberg"},
}

// isGutenbergLicense checks whether data (raw XHTML) contains patterns
// indicating a Project Gutenberg license page.
func isGutenbergLicense(data []byte) bool {
	// Extract text to avoid matching inside tags/attributes.
	text, err := extractText(data)
	if err != nil {
		// Fallback: search raw bytes lowercased.
		text = string(bytes.ToLower(data))
	} else {
		text = strings.ToLower(text)
	}

	for _, pat := range gutenbergPatterns {
		if strings.Contains(text, pat) {
			return true
		}
	}
	for _, combo := range gutenbergComboPatterns {
		if strings.Contains(text, combo[0]) && strings.Contains(text, combo[1]) {
			return true
		}
	}
	return false
}

// RawContent reads the raw XHTML bytes of this chapter from the ePub archive.
// Leading UTF-8 BOM is stripped if present.
func (c Chapter) RawContent() ([]byte, error) {
	if c.book == nil {
		return nil, ErrInvalidChapter
	}
	data, err := c.book.readFile(c.Href)
	if err != nil {
		return nil, err
	}
	return stripBOM(data), nil
}

// TextContent extracts the plain text content from this chapter's XHTML.
// Block-level elements produce line breaks; script and style content is skipped.
func (c Chapter) TextContent() (string, error) {
	data, err := c.RawContent()
	if err != nil {
		return "", err
	}
	return extractText(data)
}

// BodyHTML extracts the inner HTML of the <body> element from this chapter's XHTML.
// Image paths are rewritten to ZIP-root-relative paths. Script and style elements
// and event handler attributes are stripped.
func (c Chapter) BodyHTML() (string, error) {
	data, err := c.RawContent()
	if err != nil {
		return "", err
	}
	// Rewrite image paths in the full document before extracting body,
	// so that html.Parse operates on a complete XHTML document.
	data = rewriteImagePaths(data, c.Href)
	return extractBodyHTML(data)
}
