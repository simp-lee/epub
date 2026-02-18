package epub

import (
	"errors"
	"strings"
	"testing"
)

// minimalOPF returns a valid OPF with the given spine items and manifest entries.
func chapterTestOPF() string {
	return `<?xml version="1.0" encoding="UTF-8"?>
<package xmlns="http://www.idpf.org/2007/opf" version="2.0" unique-identifier="uid">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
    <dc:title>Test Book</dc:title>
    <dc:language>en</dc:language>
    <dc:identifier id="uid">test-id-001</dc:identifier>
  </metadata>
  <manifest>
    <item id="ch1" href="chapter01.xhtml" media-type="application/xhtml+xml"/>
    <item id="ch2" href="chapter02.xhtml" media-type="application/xhtml+xml"/>
    <item id="ch3" href="chapter03.xhtml" media-type="application/xhtml+xml"/>
    <item id="ncx" href="toc.ncx" media-type="application/x-dtbncx+xml"/>
  </manifest>
  <spine toc="ncx">
    <itemref idref="ch1"/>
    <itemref idref="ch2"/>
    <itemref idref="ch3" linear="no"/>
  </spine>
</package>`
}

func chapterTestNCX() string {
	return `<?xml version="1.0" encoding="UTF-8"?>
<ncx xmlns="http://www.daisy.org/z3986/2005/ncx/" version="2005-1">
  <navMap>
    <navPoint id="np1" playOrder="1">
      <navLabel><text>Chapter One</text></navLabel>
      <content src="chapter01.xhtml"/>
    </navPoint>
    <navPoint id="np2" playOrder="2">
      <navLabel><text>Chapter Two</text></navLabel>
      <content src="chapter02.xhtml"/>
    </navPoint>
  </navMap>
</ncx>`
}

const chapterTestContainer = `<?xml version="1.0" encoding="UTF-8"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
  <rootfiles>
    <rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/>
  </rootfiles>
</container>`

const chapter01XHTML = `<?xml version="1.0" encoding="UTF-8"?>
<html xmlns="http://www.w3.org/1999/xhtml">
<head><title>Chapter One</title></head>
<body>
<h1>Chapter One</h1>
<p>Hello, world!</p>
<p>Second paragraph.</p>
</body>
</html>`

const chapter02XHTML = `<?xml version="1.0" encoding="UTF-8"?>
<html xmlns="http://www.w3.org/1999/xhtml">
<head><title>Chapter Two</title></head>
<body>
<h1>Chapter Two</h1>
<p>Goodbye, world!</p>
</body>
</html>`

const chapter03XHTML = `<?xml version="1.0" encoding="UTF-8"?>
<html xmlns="http://www.w3.org/1999/xhtml">
<head><title>Chapter Three</title></head>
<body>
<p>No TOC entry for this one.</p>
</body>
</html>`

func buildChapterTestEPub(t *testing.T) string {
	t.Helper()
	files := map[string]string{
		"mimetype":               "application/epub+zip",
		"META-INF/container.xml": chapterTestContainer,
		"OEBPS/content.opf":      chapterTestOPF(),
		"OEBPS/toc.ncx":          chapterTestNCX(),
		"OEBPS/chapter01.xhtml":  chapter01XHTML,
		"OEBPS/chapter02.xhtml":  chapter02XHTML,
		"OEBPS/chapter03.xhtml":  chapter03XHTML,
	}
	return buildTestEPubFile(t, files)
}

func TestChapters_SpineOrder(t *testing.T) {
	fp := buildChapterTestEPub(t)
	book, err := Open(fp)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer book.Close()

	chapters := book.Chapters()
	if len(chapters) != 3 {
		t.Fatalf("got %d chapters, want 3", len(chapters))
	}

	// Verify spine order and IDs.
	wantIDs := []string{"ch1", "ch2", "ch3"}
	for i, ch := range chapters {
		if ch.ID != wantIDs[i] {
			t.Errorf("chapters[%d].ID = %q, want %q", i, ch.ID, wantIDs[i])
		}
	}

	// Verify titles from TOC matching.
	if chapters[0].Title != "Chapter One" {
		t.Errorf("chapters[0].Title = %q, want %q", chapters[0].Title, "Chapter One")
	}
	if chapters[1].Title != "Chapter Two" {
		t.Errorf("chapters[1].Title = %q, want %q", chapters[1].Title, "Chapter Two")
	}
	// Chapter 3 has no TOC entry.
	if chapters[2].Title != "" {
		t.Errorf("chapters[2].Title = %q, want empty", chapters[2].Title)
	}

	// Verify Linear field.
	if !chapters[0].Linear {
		t.Error("chapters[0].Linear = false, want true")
	}
	if !chapters[1].Linear {
		t.Error("chapters[1].Linear = false, want true")
	}
	if chapters[2].Linear {
		t.Error("chapters[2].Linear = true, want false")
	}
}

func TestChapters_Cached(t *testing.T) {
	fp := buildChapterTestEPub(t)
	book, err := Open(fp)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer book.Close()

	c1 := book.Chapters()
	c2 := book.Chapters()
	if len(c1) != len(c2) {
		t.Fatal("cached Chapters() returned different lengths")
	}
	if len(c1) == 0 {
		t.Fatal("Chapters() returned empty result")
	}

	// Verify returned slices are independent copies.
	c1[0].Title = "mutated"
	if c2[0].Title == "mutated" {
		t.Error("Chapters() returned aliased slices")
	}

	// Verify cached internal data is still stable.
	c3 := book.Chapters()
	if c3[0].Title == "mutated" {
		t.Error("Chapters() exposed internal cached data")
	}
}

func TestChapter_RawContent(t *testing.T) {
	fp := buildChapterTestEPub(t)
	book, err := Open(fp)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer book.Close()

	chapters := book.Chapters()
	data, err := chapters[0].RawContent()
	if err != nil {
		t.Fatalf("RawContent: %v", err)
	}

	got := string(data)
	if !strings.Contains(got, "<h1>Chapter One</h1>") {
		t.Errorf("RawContent missing expected h1 tag, got:\n%s", got)
	}
	if !strings.Contains(got, "Hello, world!") {
		t.Errorf("RawContent missing expected text, got:\n%s", got)
	}
}

func TestChapter_RawContent_StripsBOM(t *testing.T) {
	// Build an ePub with a BOM-prefixed chapter.
	bom := "\xEF\xBB\xBF"
	chapterWithBOM := bom + chapter01XHTML

	files := map[string]string{
		"mimetype":               "application/epub+zip",
		"META-INF/container.xml": chapterTestContainer,
		"OEBPS/content.opf":      chapterTestOPF(),
		"OEBPS/toc.ncx":          chapterTestNCX(),
		"OEBPS/chapter01.xhtml":  chapterWithBOM,
		"OEBPS/chapter02.xhtml":  chapter02XHTML,
		"OEBPS/chapter03.xhtml":  chapter03XHTML,
	}
	fp := buildTestEPubFile(t, files)

	book, err := Open(fp)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer book.Close()

	chapters := book.Chapters()
	data, err := chapters[0].RawContent()
	if err != nil {
		t.Fatalf("RawContent: %v", err)
	}

	// BOM should be stripped.
	if len(data) >= 3 && data[0] == 0xEF && data[1] == 0xBB && data[2] == 0xBF {
		t.Error("RawContent did not strip BOM")
	}
	if !strings.Contains(string(data), "<h1>Chapter One</h1>") {
		t.Error("RawContent content is corrupt after BOM stripping")
	}
}

func TestChapter_TextContent(t *testing.T) {
	fp := buildChapterTestEPub(t)
	book, err := Open(fp)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer book.Close()

	chapters := book.Chapters()
	text, err := chapters[0].TextContent()
	if err != nil {
		t.Fatalf("TextContent: %v", err)
	}

	if !strings.Contains(text, "Chapter One") {
		t.Errorf("TextContent missing 'Chapter One', got:\n%s", text)
	}
	if !strings.Contains(text, "Hello, world!") {
		t.Errorf("TextContent missing 'Hello, world!', got:\n%s", text)
	}
	if !strings.Contains(text, "Second paragraph.") {
		t.Errorf("TextContent missing 'Second paragraph.', got:\n%s", text)
	}
	// Should not contain HTML tags.
	if strings.Contains(text, "<h1>") || strings.Contains(text, "<p>") {
		t.Errorf("TextContent contains HTML tags, got:\n%s", text)
	}
}

func TestChapter_BodyHTML(t *testing.T) {
	fp := buildChapterTestEPub(t)
	book, err := Open(fp)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer book.Close()

	chapters := book.Chapters()
	body, err := chapters[0].BodyHTML()
	if err != nil {
		t.Fatalf("BodyHTML: %v", err)
	}

	if !strings.Contains(body, "<h1>Chapter One</h1>") {
		t.Errorf("BodyHTML missing <h1> tag, got:\n%s", body)
	}
	if !strings.Contains(body, "<p>Hello, world!</p>") {
		t.Errorf("BodyHTML missing <p> tag, got:\n%s", body)
	}
	// Should not contain <body> wrapper.
	if strings.Contains(body, "<body>") {
		t.Errorf("BodyHTML should not contain <body> tag, got:\n%s", body)
	}
}

func TestChapter_BodyHTML_RewritesImagePaths(t *testing.T) {
	chapterWithImg := `<?xml version="1.0" encoding="UTF-8"?>
<html xmlns="http://www.w3.org/1999/xhtml">
<head><title>Images</title></head>
<body>
<p>Text</p>
<img src="images/cover.jpg" alt="Cover"/>
</body>
</html>`

	files := map[string]string{
		"mimetype":               "application/epub+zip",
		"META-INF/container.xml": chapterTestContainer,
		"OEBPS/content.opf": `<?xml version="1.0" encoding="UTF-8"?>
<package xmlns="http://www.idpf.org/2007/opf" version="2.0" unique-identifier="uid">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
    <dc:title>Image Test</dc:title>
    <dc:language>en</dc:language>
    <dc:identifier id="uid">test-img-001</dc:identifier>
  </metadata>
  <manifest>
    <item id="ch1" href="chapter01.xhtml" media-type="application/xhtml+xml"/>
    <item id="ncx" href="toc.ncx" media-type="application/x-dtbncx+xml"/>
  </manifest>
  <spine toc="ncx">
    <itemref idref="ch1"/>
  </spine>
</package>`,
		"OEBPS/toc.ncx": `<?xml version="1.0" encoding="UTF-8"?>
<ncx xmlns="http://www.daisy.org/z3986/2005/ncx/" version="2005-1">
  <navMap>
    <navPoint id="np1" playOrder="1">
      <navLabel><text>Chapter</text></navLabel>
      <content src="chapter01.xhtml"/>
    </navPoint>
  </navMap>
</ncx>`,
		"OEBPS/chapter01.xhtml": chapterWithImg,
	}
	fp := buildTestEPubFile(t, files)

	book, err := Open(fp)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer book.Close()

	chapters := book.Chapters()
	if len(chapters) == 0 {
		t.Fatal("no chapters")
	}

	body, err := chapters[0].BodyHTML()
	if err != nil {
		t.Fatalf("BodyHTML: %v", err)
	}

	// Image path should be rewritten from "images/cover.jpg" to "OEBPS/images/cover.jpg".
	if !strings.Contains(body, "OEBPS/images/cover.jpg") {
		t.Errorf("BodyHTML did not rewrite image path, got:\n%s", body)
	}
}

func TestChapter_RawContent_FileNotFound(t *testing.T) {
	// Build a chapter with invalid href.
	ch := Chapter{
		Href: "nonexistent.xhtml",
		book: &Book{},
	}

	// We need a Book with a zip reader. Use a minimal one.
	files := map[string]string{
		"mimetype": "application/epub+zip",
	}
	zr := buildTestZip(t, files)
	ch.book = &Book{zip: zr}

	_, err := ch.RawContent()
	if err == nil {
		t.Error("expected error for nonexistent file, got nil")
	}
}

func TestChapter_RawContent_ZeroValueChapter(t *testing.T) {
	var ch Chapter

	_, err := ch.RawContent()
	if !errors.Is(err, ErrInvalidChapter) {
		t.Fatalf("RawContent() error = %v, want ErrInvalidChapter", err)
	}
}

func TestChapter_ContentMethods_ZeroValueChapter(t *testing.T) {
	var ch Chapter

	_, err := ch.TextContent()
	if !errors.Is(err, ErrInvalidChapter) {
		t.Fatalf("TextContent() error = %v, want ErrInvalidChapter", err)
	}

	_, err = ch.BodyHTML()
	if !errors.Is(err, ErrInvalidChapter) {
		t.Fatalf("BodyHTML() error = %v, want ErrInvalidChapter", err)
	}
}

// --- Gutenberg license detection tests ---

const gutenbergLicenseXHTML = `<?xml version="1.0" encoding="UTF-8"?>
<html xmlns="http://www.w3.org/1999/xhtml">
<head><title>License</title></head>
<body>
<h1>Project Gutenberg License</h1>
<p>This eBook is for the use of anyone anywhere at no cost and with
almost no restrictions whatsoever.</p>
</body>
</html>`

const gutenbergEndXHTML = `<?xml version="1.0" encoding="UTF-8"?>
<html xmlns="http://www.w3.org/1999/xhtml">
<head><title>End</title></head>
<body>
<p>*** END OF THIS PROJECT GUTENBERG EBOOK PRIDE AND PREJUDICE ***</p>
<p>Updated editions will replace the previous one.</p>
</body>
</html>`

const gutenbergStartXHTML = `<?xml version="1.0" encoding="UTF-8"?>
<html xmlns="http://www.w3.org/1999/xhtml">
<head><title>Start</title></head>
<body>
<p>*** START OF THE PROJECT GUTENBERG LICENSE ***</p>
<p>License text here...</p>
<p>*** END OF THE PROJECT GUTENBERG LICENSE ***</p>
</body>
</html>`

const gutenbergComboXHTML = `<?xml version="1.0" encoding="UTF-8"?>
<html xmlns="http://www.w3.org/1999/xhtml">
<head><title>Terms</title></head>
<body>
<p>Project Gutenberg is a registered trademark.</p>
<p>Please read the terms of use carefully.</p>
</body>
</html>`

const gutenbergFullLicenseXHTML = `<?xml version="1.0" encoding="UTF-8"?>
<html xmlns="http://www.w3.org/1999/xhtml">
<head><title>Full License</title></head>
<body>
<p>Please see the full license available at gutenberg.org/license</p>
</body>
</html>`

const regularChapterXHTML = `<?xml version="1.0" encoding="UTF-8"?>
<html xmlns="http://www.w3.org/1999/xhtml">
<head><title>Normal</title></head>
<body>
<h1>A Normal Chapter</h1>
<p>This is regular content with no license information.</p>
</body>
</html>`

func TestIsGutenbergLicense(t *testing.T) {
	tests := []struct {
		name string
		data string
		want bool
	}{
		{"license title", gutenbergLicenseXHTML, true},
		{"end of ebook marker", gutenbergEndXHTML, true},
		{"license block", gutenbergStartXHTML, true},
		{"combo: project gutenberg + terms of use", gutenbergComboXHTML, true},
		{"gutenberg.org/license URL", gutenbergFullLicenseXHTML, true},
		{"regular chapter", regularChapterXHTML, false},
		{"empty body", `<html><body></body></html>`, false},
		{"gutenberg mention without license context", `<html><body><p>Inspired by Project Gutenberg ideals.</p></body></html>`, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isGutenbergLicense([]byte(tt.data))
			if got != tt.want {
				t.Errorf("isGutenbergLicense() = %v, want %v", got, tt.want)
			}
		})
	}
}

func gutenbergTestOPF() string {
	return `<?xml version="1.0" encoding="UTF-8"?>
<package xmlns="http://www.idpf.org/2007/opf" version="2.0" unique-identifier="uid">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
    <dc:title>Gutenberg Test</dc:title>
    <dc:language>en</dc:language>
    <dc:identifier id="uid">test-gb-001</dc:identifier>
  </metadata>
  <manifest>
    <item id="ch1" href="chapter01.xhtml" media-type="application/xhtml+xml"/>
    <item id="ch2" href="chapter02.xhtml" media-type="application/xhtml+xml"/>
    <item id="license" href="license.xhtml" media-type="application/xhtml+xml"/>
    <item id="ncx" href="toc.ncx" media-type="application/x-dtbncx+xml"/>
  </manifest>
  <spine toc="ncx">
    <itemref idref="ch1"/>
    <itemref idref="ch2"/>
    <itemref idref="license"/>
  </spine>
</package>`
}

func gutenbergTestNCX() string {
	return `<?xml version="1.0" encoding="UTF-8"?>
<ncx xmlns="http://www.daisy.org/z3986/2005/ncx/" version="2005-1">
  <navMap>
    <navPoint id="np1" playOrder="1">
      <navLabel><text>Chapter One</text></navLabel>
      <content src="chapter01.xhtml"/>
    </navPoint>
    <navPoint id="np2" playOrder="2">
      <navLabel><text>Chapter Two</text></navLabel>
      <content src="chapter02.xhtml"/>
    </navPoint>
  </navMap>
</ncx>`
}

func buildGutenbergTestEPub(t *testing.T) string {
	t.Helper()
	files := map[string]string{
		"mimetype":               "application/epub+zip",
		"META-INF/container.xml": chapterTestContainer,
		"OEBPS/content.opf":      gutenbergTestOPF(),
		"OEBPS/toc.ncx":          gutenbergTestNCX(),
		"OEBPS/chapter01.xhtml":  chapter01XHTML,
		"OEBPS/chapter02.xhtml":  chapter02XHTML,
		"OEBPS/license.xhtml":    gutenbergLicenseXHTML,
	}
	return buildTestEPubFile(t, files)
}

func TestChapters_GutenbergLicenseDetection(t *testing.T) {
	fp := buildGutenbergTestEPub(t)
	book, err := Open(fp)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer book.Close()

	// Chapters() alone must not set IsLicense (lazy detection).
	chapters := book.Chapters()
	if len(chapters) != 3 {
		t.Fatalf("got %d chapters, want 3", len(chapters))
	}
	for i, ch := range chapters {
		if ch.IsLicense {
			t.Errorf("before ContentChapters(): chapters[%d].IsLicense = true, want false", i)
		}
	}

	// ContentChapters() triggers detection and returns non-license chapters.
	content := book.ContentChapters()
	if len(content) != 2 {
		t.Fatalf("ContentChapters() returned %d chapters, want 2", len(content))
	}

	// After ContentChapters(), cached Chapters() has IsLicense set.
	chapters = book.Chapters()
	if chapters[0].IsLicense {
		t.Error("chapters[0].IsLicense = true, want false")
	}
	if chapters[1].IsLicense {
		t.Error("chapters[1].IsLicense = true, want false")
	}
	if !chapters[2].IsLicense {
		t.Error("chapters[2].IsLicense = false, want true")
	}
}

func TestContentChapters_ExcludesLicense(t *testing.T) {
	fp := buildGutenbergTestEPub(t)
	book, err := Open(fp)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer book.Close()

	content := book.ContentChapters()
	if len(content) != 2 {
		t.Fatalf("ContentChapters() returned %d chapters, want 2", len(content))
	}

	for _, ch := range content {
		if ch.IsLicense {
			t.Errorf("ContentChapters() included license chapter %q", ch.ID)
		}
	}

	// Verify IDs.
	if content[0].ID != "ch1" {
		t.Errorf("content[0].ID = %q, want %q", content[0].ID, "ch1")
	}
	if content[1].ID != "ch2" {
		t.Errorf("content[1].ID = %q, want %q", content[1].ID, "ch2")
	}
}

func TestContentChapters_NoLicensePages(t *testing.T) {
	fp := buildChapterTestEPub(t)
	book, err := Open(fp)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer book.Close()

	content := book.ContentChapters()
	all := book.Chapters()
	if len(content) != len(all) {
		t.Errorf("ContentChapters() = %d, Chapters() = %d; want equal when no license pages",
			len(content), len(all))
	}
}

func TestChapters_GutenbergEndOfEbook(t *testing.T) {
	// Test detection of "END OF THIS PROJECT GUTENBERG EBOOK" pattern.
	files := map[string]string{
		"mimetype":               "application/epub+zip",
		"META-INF/container.xml": chapterTestContainer,
		"OEBPS/content.opf": `<?xml version="1.0" encoding="UTF-8"?>
<package xmlns="http://www.idpf.org/2007/opf" version="2.0" unique-identifier="uid">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
    <dc:title>End Test</dc:title>
    <dc:language>en</dc:language>
    <dc:identifier id="uid">test-end-001</dc:identifier>
  </metadata>
  <manifest>
    <item id="ch1" href="chapter01.xhtml" media-type="application/xhtml+xml"/>
    <item id="ending" href="ending.xhtml" media-type="application/xhtml+xml"/>
    <item id="ncx" href="toc.ncx" media-type="application/x-dtbncx+xml"/>
  </manifest>
  <spine toc="ncx">
    <itemref idref="ch1"/>
    <itemref idref="ending"/>
  </spine>
</package>`,
		"OEBPS/toc.ncx":         gutenbergTestNCX(),
		"OEBPS/chapter01.xhtml": chapter01XHTML,
		"OEBPS/ending.xhtml":    gutenbergEndXHTML,
	}
	fp := buildTestEPubFile(t, files)

	book, err := Open(fp)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer book.Close()

	chapters := book.Chapters()
	if len(chapters) != 2 {
		t.Fatalf("got %d chapters, want 2", len(chapters))
	}

	// ContentChapters() triggers detection.
	content := book.ContentChapters()
	if len(content) != 1 {
		t.Fatalf("ContentChapters() returned %d chapters, want 1", len(content))
	}

	// After detection, Chapters() reflects IsLicense.
	chapters = book.Chapters()
	if chapters[0].IsLicense {
		t.Error("chapters[0] should not be license")
	}
	if !chapters[1].IsLicense {
		t.Error("chapters[1] should be license (END OF THIS PROJECT GUTENBERG EBOOK)")
	}
}
