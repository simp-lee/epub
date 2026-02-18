package epub

import (
	"bytes"
	"errors"
	"io"
	"net/url"
	"regexp"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// entityNameToNumeric maps lowercase HTML entity names to their XML numeric
// character references. encoding/xml does not recognise HTML named entities,
// so we convert them before parsing OPF/NCX files.
var entityNameToNumeric = map[string][]byte{
	"nbsp": []byte("&#160;"), "mdash": []byte("&#8212;"), "ndash": []byte("&#8211;"),
	"hellip": []byte("&#8230;"),
	"lsquo": []byte("&#8216;"), "rsquo": []byte("&#8217;"),
	"ldquo": []byte("&#8220;"), "rdquo": []byte("&#8221;"),
	"copy": []byte("&#169;"), "reg": []byte("&#174;"), "trade": []byte("&#8482;"),
	"bull": []byte("&#8226;"), "middot": []byte("&#183;"),
	"eacute": []byte("&#233;"), "egrave": []byte("&#232;"),
	"ecirc": []byte("&#234;"), "euml": []byte("&#235;"),
	"aacute": []byte("&#225;"), "agrave": []byte("&#224;"),
	"acirc": []byte("&#226;"), "auml": []byte("&#228;"),
	"iacute": []byte("&#237;"), "igrave": []byte("&#236;"),
	"icirc": []byte("&#238;"), "iuml": []byte("&#239;"),
	"oacute": []byte("&#243;"), "ograve": []byte("&#242;"),
	"ocirc": []byte("&#244;"), "ouml": []byte("&#246;"),
	"uacute": []byte("&#250;"), "ugrave": []byte("&#249;"),
	"ucirc": []byte("&#251;"), "uuml": []byte("&#252;"),
	"ntilde": []byte("&#241;"), "ccedil": []byte("&#231;"),
	"times": []byte("&#215;"), "divide": []byte("&#247;"),
	"deg": []byte("&#176;"), "para": []byte("&#182;"), "sect": []byte("&#167;"),
	"laquo": []byte("&#171;"), "raquo": []byte("&#187;"),
	"iexcl": []byte("&#161;"), "iquest": []byte("&#191;"),
}

// htmlEntityPattern matches common HTML named entities case-insensitively.
var htmlEntityPattern = regexp.MustCompile(
	`(?i)&(nbsp|mdash|ndash|hellip|lsquo|rsquo|ldquo|rdquo|copy|reg|trade|bull|middot|` +
		`eacute|egrave|ecirc|euml|aacute|agrave|acirc|auml|iacute|igrave|icirc|iuml|` +
		`oacute|ograve|ocirc|ouml|uacute|ugrave|ucirc|uuml|ntilde|ccedil|` +
		`times|divide|deg|para|sect|laquo|raquo|iexcl|iquest);`)

// preprocessHTMLEntities replaces common HTML named entities with their
// numeric character references so that encoding/xml can parse the data.
// The matching is case-insensitive to handle non-standard ePub content.
func preprocessHTMLEntities(data []byte) []byte {
	return htmlEntityPattern.ReplaceAllFunc(data, func(match []byte) []byte {
		// Extract entity name between & and ;, lowercase for lookup.
		name := strings.ToLower(string(match[1 : len(match)-1]))
		if replacement, ok := entityNameToNumeric[name]; ok {
			return replacement
		}
		return match
	})
}

// blockTags is the set of tags that should insert a newline when encountered
// during text extraction.
var blockTags = map[atom.Atom]bool{
	atom.P:          true,
	atom.Br:         true,
	atom.Div:        true,
	atom.H1:         true,
	atom.H2:         true,
	atom.H3:         true,
	atom.H4:         true,
	atom.H5:         true,
	atom.H6:         true,
	atom.Li:         true,
	atom.Tr:         true,
	atom.Blockquote: true,
	atom.Hr:         true,
}

// skipTags is the set of tags whose content should be skipped during text extraction.
var skipTags = map[atom.Atom]bool{
	atom.Script: true,
	atom.Style:  true,
}

var selfClosingSkipTagPattern = regexp.MustCompile(`(?is)<(script|style)\b([^>]*)/>`)

func normalizeSelfClosingSkipTags(htmlData []byte) []byte {
	if !selfClosingSkipTagPattern.Match(htmlData) {
		return htmlData
	}
	return selfClosingSkipTagPattern.ReplaceAll(htmlData, []byte(`<$1$2></$1>`))
}

// extractText extracts the plain text content from HTML data.
// Block-level elements (<p>, <br>, <div>, <h1>-<h6>, <li>, <tr>) produce line
// breaks. Content inside <script> and <style> tags is skipped.
func extractText(htmlData []byte) (string, error) {
	htmlData = normalizeSelfClosingSkipTags(htmlData)
	tokenizer := html.NewTokenizer(bytes.NewReader(htmlData))

	var buf strings.Builder
	skipDepth := 0 // depth inside a skip tag
	lastWasNewline := true

	for {
		tt := tokenizer.Next()
		switch tt {
		case html.ErrorToken:
			err := tokenizer.Err()
			if errors.Is(err, io.EOF) {
				return strings.TrimSpace(buf.String()), nil
			}
			return "", err

		case html.StartTagToken:
			tn, _ := tokenizer.TagName()
			a := atom.Lookup(tn)
			if skipTags[a] {
				skipDepth++
				continue
			}
			if skipDepth > 0 {
				continue
			}
			if blockTags[a] {
				if buf.Len() > 0 && !lastWasNewline {
					buf.WriteByte('\n')
					lastWasNewline = true
				}
			}

		case html.SelfClosingTagToken:
			tn, _ := tokenizer.TagName()
			a := atom.Lookup(tn)
			if skipDepth > 0 {
				continue
			}
			if blockTags[a] {
				if buf.Len() > 0 && !lastWasNewline {
					buf.WriteByte('\n')
					lastWasNewline = true
				}
			}

		case html.EndTagToken:
			tn, _ := tokenizer.TagName()
			a := atom.Lookup(tn)
			if skipTags[a] && skipDepth > 0 {
				skipDepth--
			}

		case html.TextToken:
			if skipDepth > 0 {
				continue
			}
			raw := string(tokenizer.Text())
			// Collapse internal whitespace runs to single spaces, but preserve
			// non-empty content so that inline elements keep their spacing.
			text := collapseWhitespace(raw)
			if text != "" {
				buf.WriteString(text)
				lastWasNewline = strings.HasSuffix(text, "\n")
			}
		}
	}
}

// collapseWhitespace replaces runs of whitespace characters (spaces, tabs,
// newlines) with a single space. Returns empty string if the input is all whitespace.
// Leading and trailing whitespace is preserved as a single space so that
// inter-element spacing (e.g., between inline tags) is maintained.
func collapseWhitespace(s string) string {
	var buf strings.Builder
	inSpace := false
	hasNonSpace := false
	for _, r := range s {
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			inSpace = true
		} else {
			if inSpace && buf.Len() > 0 {
				buf.WriteByte(' ')
			}
			buf.WriteRune(r)
			inSpace = false
			hasNonSpace = true
		}
	}
	if !hasNonSpace {
		return ""
	}
	result := buf.String()
	// Preserve leading whitespace as a single space.
	if len(s) > 0 && isWhitespace(rune(s[0])) {
		result = " " + result
	}
	// Preserve trailing whitespace as a single space.
	if inSpace {
		result = result + " "
	}
	return result
}

// isWhitespace returns true if r is a whitespace character.
func isWhitespace(r rune) bool {
	return r == ' ' || r == '\t' || r == '\n' || r == '\r'
}

// extractBodyHTML parses HTML data, finds the <body> element, and renders its
// children back to an HTML string. Elements <script>, <style> are removed.
// Event handler attributes (onclick, onload, etc.) are stripped.
func extractBodyHTML(htmlData []byte) (string, error) {
	doc, err := html.Parse(bytes.NewReader(htmlData))
	if err != nil {
		return "", err
	}

	body := findElement(doc, atom.Body)
	if body == nil {
		// No body found; return empty string.
		return "", nil
	}

	// Clean the body subtree.
	cleanNode(body)

	// Render children of body.
	var buf bytes.Buffer
	for c := body.FirstChild; c != nil; c = c.NextSibling {
		if err := html.Render(&buf, c); err != nil {
			return "", err
		}
	}
	return strings.TrimSpace(buf.String()), nil
}

// findElement performs a depth-first search for a node with the given atom tag.
func findElement(n *html.Node, a atom.Atom) *html.Node {
	if n.Type == html.ElementNode && n.DataAtom == a {
		return n
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if result := findElement(c, a); result != nil {
			return result
		}
	}
	return nil
}

// cleanNode recursively removes <script> and <style> elements and strips
// event handler attributes from the subtree rooted at n.
func cleanNode(n *html.Node) {
	var next *html.Node
	for c := n.FirstChild; c != nil; c = next {
		next = c.NextSibling
		if c.Type == html.ElementNode && (c.DataAtom == atom.Script || c.DataAtom == atom.Style) {
			n.RemoveChild(c)
			continue
		}
		if c.Type == html.ElementNode {
			stripEventAttributes(c)
		}
		cleanNode(c)
	}
}

// stripEventAttributes removes all event handler attributes (on*) from the node.
func stripEventAttributes(n *html.Node) {
	cleaned := n.Attr[:0]
	for _, attr := range n.Attr {
		keyLower := strings.ToLower(attr.Key)
		if strings.HasPrefix(keyLower, "on") {
			continue
		}
		if isURIAttribute(attr) && !isSafeURI(attr.Val) {
			continue
		}
		cleaned = append(cleaned, attr)
	}
	n.Attr = cleaned
}

// isURIAttribute reports whether attr is an HTML attribute that may contain
// a URL and should be protocol-sanitized.
func isURIAttribute(attr html.Attribute) bool {
	if attr.Key == "href" || attr.Key == "src" {
		return true
	}
	if attr.Namespace == "xlink" && attr.Key == "href" {
		return true
	}
	if attr.Key == "xlink:href" {
		return true
	}
	return false
}

// isSafeURI validates URI values for href/src-like attributes.
// Allowed values:
//   - relative paths and fragments
//   - schemes: http, https, mailto
//   - data:image/*
func isSafeURI(raw string) bool {
	v := strings.TrimSpace(raw)
	if v == "" {
		return true
	}
	if strings.HasPrefix(v, "#") || strings.HasPrefix(v, "/") || strings.HasPrefix(v, "./") || strings.HasPrefix(v, "../") || strings.HasPrefix(v, "?") {
		return true
	}

	u, err := url.Parse(v)
	if err != nil {
		return false
	}

	if u.Scheme == "" {
		return true
	}

	scheme := strings.ToLower(u.Scheme)
	switch scheme {
	case "http", "https", "mailto":
		return true
	case "data":
		lower := strings.ToLower(v)
		return strings.HasPrefix(lower, "data:image/")
	default:
		return false
	}
}

// rewriteImagePaths rewrites relative image paths in HTML data to absolute
// ZIP-internal paths, using basePath as the reference location.
// It handles <img src="..."> and <image xlink:href="...">.
func rewriteImagePaths(htmlData []byte, basePath string) []byte {
	doc, err := html.Parse(bytes.NewReader(htmlData))
	if err != nil {
		// If parsing fails, return the input unchanged.
		return htmlData
	}

	rewriteImageNode(doc, basePath)

	var buf bytes.Buffer
	if err := html.Render(&buf, doc); err != nil {
		return htmlData
	}
	return buf.Bytes()
}

// rewriteImageNode recursively walks the DOM tree, rewriting image paths.
func rewriteImageNode(n *html.Node, basePath string) {
	if n.Type == html.ElementNode {
		switch n.DataAtom {
		case atom.Img:
			rewriteAttr(n, "", "src", basePath)
		case atom.Image:
			// SVG <image> uses xlink:href or href
			rewriteAttr(n, "xlink", "href", basePath)
			rewriteAttr(n, "", "href", basePath)
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		rewriteImageNode(c, basePath)
	}
}

// rewriteAttr rewrites a specific attribute from a relative path to an absolute
// ZIP-internal path. namespace is the XML namespace prefix (empty for no namespace).
func rewriteAttr(n *html.Node, namespace, key, basePath string) {
	for i, attr := range n.Attr {
		if matchAttr(attr, namespace, key) {
			val := attr.Val
			if val == "" || strings.HasPrefix(val, "http://") || strings.HasPrefix(val, "https://") || strings.HasPrefix(val, "data:") || hasURIScheme(val) {
				continue
			}
			if resolved := resolveRelativePath(basePath, val); resolved != "" {
				n.Attr[i].Val = resolved
			}
		}
	}
}

// hasURIScheme reports whether s starts with a URI scheme like "mailto:" or
// "javascript:".
func hasURIScheme(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	// RFC 3986: URI scheme must start with a letter.
	if !((s[0] >= 'A' && s[0] <= 'Z') || (s[0] >= 'a' && s[0] <= 'z')) {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == ':' {
			return i > 1
		}
		if !(c == '+' || c == '-' || c == '.' || (c >= '0' && c <= '9') || (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z')) {
			return false
		}
	}
	return false
}

// matchAttr checks if an html.Attribute matches the given namespace and key.
func matchAttr(attr html.Attribute, namespace, key string) bool {
	if namespace == "" {
		return attr.Key == key && attr.Namespace == ""
	}
	// For namespaced attributes, x/net/html may store them in different ways.
	// Check both namespace field and prefixed key.
	if attr.Namespace == namespace && attr.Key == key {
		return true
	}
	if attr.Key == namespace+":"+key {
		return true
	}
	return false
}
