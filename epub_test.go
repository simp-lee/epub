package epub

import (
	"archive/zip"
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
)

// minimalEPubFiles returns the minimum set of files for a valid ePub.
func minimalEPubFiles() map[string]string {
	return map[string]string{
		"mimetype":               "application/epub+zip",
		"META-INF/container.xml": validContainerXML,
		"OEBPS/content.opf":      `<?xml version="1.0"?><package/>`,
	}
}

func TestOpen_Valid(t *testing.T) {
	fp := buildTestEPubFile(t, minimalEPubFiles())

	book, err := Open(fp)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer book.Close()

	if book.opfPath != "OEBPS/content.opf" {
		t.Errorf("opfPath = %q, want %q", book.opfPath, "OEBPS/content.opf")
	}
	if book.opfDir != "OEBPS" {
		t.Errorf("opfDir = %q, want %q", book.opfDir, "OEBPS")
	}
	if len(book.Warnings()) != 0 {
		t.Errorf("unexpected warnings: %v", book.Warnings())
	}
}

func TestNewReader_Valid(t *testing.T) {
	files := minimalEPubFiles()
	data := buildTestEPubBytes(t, files)

	book, err := NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("NewReader() error = %v", err)
	}
	defer book.Close()

	if book.opfPath != "OEBPS/content.opf" {
		t.Errorf("opfPath = %q, want %q", book.opfPath, "OEBPS/content.opf")
	}
	if book.closer != nil {
		t.Error("NewReader should not set closer")
	}
}

func TestOpen_MimetypeMissing(t *testing.T) {
	files := minimalEPubFiles()
	delete(files, "mimetype")
	fp := buildTestEPubFile(t, files)

	book, err := Open(fp)
	if err != nil {
		t.Fatalf("Open() error = %v; want success with warning", err)
	}
	defer book.Close()

	if len(book.Warnings()) == 0 {
		t.Error("expected warning for missing mimetype, got none")
	}
}

func TestOpen_MimetypeWrongContent(t *testing.T) {
	files := minimalEPubFiles()
	files["mimetype"] = "text/plain"
	fp := buildTestEPubFile(t, files)

	book, err := Open(fp)
	if err != nil {
		t.Fatalf("Open() error = %v; want success with warning", err)
	}
	defer book.Close()

	found := false
	for _, w := range book.Warnings() {
		if w != "" {
			found = true
		}
	}
	if !found {
		t.Error("expected warning for wrong mimetype, got none")
	}
}

func TestWarnings_DefensiveCopy(t *testing.T) {
	book := &Book{warnings: []string{"warning-a", "warning-b"}}

	got := book.Warnings()
	got[0] = "mutated"

	gotAgain := book.Warnings()
	if gotAgain[0] != "warning-a" {
		t.Fatalf("Warnings() exposed internal slice; got %q, want %q", gotAgain[0], "warning-a")
	}
}

func TestMetadata_DefensiveCopy(t *testing.T) {
	book := &Book{metadata: Metadata{
		Version:     "3.0",
		Titles:      []string{"Original Title"},
		Authors:     []Author{{Name: "Author A"}},
		Language:    []string{"en"},
		Identifiers: []Identifier{{Value: "id-1"}},
		Subjects:    []string{"Fiction"},
	}}

	md := book.Metadata()
	md.Titles[0] = "Mutated"
	md.Authors[0].Name = "Mutated Author"
	md.Language[0] = "fr"
	md.Identifiers[0].Value = "changed"
	md.Subjects[0] = "Changed"

	again := book.Metadata()
	if again.Titles[0] != "Original Title" ||
		again.Authors[0].Name != "Author A" ||
		again.Language[0] != "en" ||
		again.Identifiers[0].Value != "id-1" ||
		again.Subjects[0] != "Fiction" {
		t.Fatalf("Metadata() exposed internal state: %#v", again)
	}
}

func TestTOC_DefensiveCopy(t *testing.T) {
	book := &Book{toc: []TOCItem{{
		Title: "Parent",
		Href:  "ch1.xhtml",
		Children: []TOCItem{{
			Title: "Child",
			Href:  "ch1.xhtml#s1",
		}},
	}}}

	toc := book.TOC()
	toc[0].Title = "Mutated Parent"
	toc[0].Children[0].Title = "Mutated Child"

	again := book.TOC()
	if again[0].Title != "Parent" {
		t.Fatalf("TOC() exposed internal top-level item; got %q", again[0].Title)
	}
	if again[0].Children[0].Title != "Child" {
		t.Fatalf("TOC() exposed internal nested item; got %q", again[0].Children[0].Title)
	}
}

func TestLandmarks_DefensiveCopy(t *testing.T) {
	book := &Book{landmarks: []TOCItem{{Title: "Cover", Href: "cover.xhtml"}}}

	lm := book.Landmarks()
	lm[0].Title = "Mutated"

	again := book.Landmarks()
	if again[0].Title != "Cover" {
		t.Fatalf("Landmarks() exposed internal slice; got %q", again[0].Title)
	}
}

func TestChapters_DefensiveCopy(t *testing.T) {
	book := &Book{chapters: []Chapter{{ID: "ch1", Title: "Original", Href: "ch1.xhtml"}}}

	chapters := book.Chapters()
	chapters[0].Title = "Mutated"

	again := book.Chapters()
	if again[0].Title != "Original" {
		t.Fatalf("Chapters() exposed internal slice; got %q", again[0].Title)
	}
}

func TestOpen_DRM(t *testing.T) {
	files := minimalEPubFiles()
	files["META-INF/sinf.xml"] = "<sinf/>"
	fp := buildTestEPubFile(t, files)

	_, err := Open(fp)
	if !errors.Is(err, ErrDRMProtected) {
		t.Errorf("Open() error = %v, want ErrDRMProtected", err)
	}
}

func TestNewReader_DRM(t *testing.T) {
	files := minimalEPubFiles()
	files["META-INF/sinf.xml"] = "<sinf/>"
	data := buildTestEPubBytes(t, files)

	_, err := NewReader(bytes.NewReader(data), int64(len(data)))
	if !errors.Is(err, ErrDRMProtected) {
		t.Errorf("NewReader() error = %v, want ErrDRMProtected", err)
	}
}

func TestClose_Idempotent(t *testing.T) {
	fp := buildTestEPubFile(t, minimalEPubFiles())

	book, err := Open(fp)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}

	if err := book.Close(); err != nil {
		t.Fatalf("first Close() error = %v", err)
	}
	if err := book.Close(); err != nil {
		t.Fatalf("second Close() error = %v", err)
	}
}

func TestClose_NewReader(t *testing.T) {
	data := buildTestEPubBytes(t, minimalEPubFiles())

	book, err := NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("NewReader() error = %v", err)
	}

	// Close should succeed even though there's no underlying file.
	if err := book.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}

func TestReadFile(t *testing.T) {
	files := minimalEPubFiles()
	files["OEBPS/chapter1.xhtml"] = "<html>hello</html>"
	fp := buildTestEPubFile(t, files)

	book, err := Open(fp)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer book.Close()

	data, err := book.ReadFile("OEBPS/chapter1.xhtml")
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(data) != "<html>hello</html>" {
		t.Errorf("ReadFile() = %q, want %q", string(data), "<html>hello</html>")
	}
}

func TestReadFile_CaseInsensitive(t *testing.T) {
	files := minimalEPubFiles()
	files["OEBPS/Chapter1.xhtml"] = "<html>hello</html>"
	fp := buildTestEPubFile(t, files)

	book, err := Open(fp)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer book.Close()

	data, err := book.ReadFile("oebps/chapter1.xhtml")
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(data) != "<html>hello</html>" {
		t.Errorf("ReadFile() = %q, want %q", string(data), "<html>hello</html>")
	}
}

func TestReadFile_NotFound(t *testing.T) {
	fp := buildTestEPubFile(t, minimalEPubFiles())

	book, err := Open(fp)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer book.Close()

	_, err = book.ReadFile("nonexistent.xhtml")
	if !errors.Is(err, ErrFileNotFound) {
		t.Errorf("ReadFile() error = %v, want ErrFileNotFound", err)
	}
}

func TestOpen_FontObfuscationWarning(t *testing.T) {
	files := minimalEPubFiles()
	files["META-INF/encryption.xml"] = `<?xml version="1.0" encoding="UTF-8"?>
<encryption xmlns="urn:oasis:names:tc:opendocument:xmlns:container"
            xmlns:enc="http://www.w3.org/2001/04/xmlenc#">
  <enc:EncryptedData>
    <enc:EncryptionMethod Algorithm="http://www.idpf.org/2008/embedding"/>
    <enc:KeyInfo/>
  </enc:EncryptedData>
</encryption>`
	fp := buildTestEPubFile(t, files)

	book, err := Open(fp)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer book.Close()

	found := false
	for _, w := range book.Warnings() {
		if w == "font obfuscation detected; obfuscated fonts may not render correctly" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected font obfuscation warning, got: %v", book.Warnings())
	}
}

func TestOpen_InvalidPath(t *testing.T) {
	_, err := Open("/nonexistent/path/test.epub")
	if err == nil {
		t.Error("Open() with invalid path should return error")
	}
}

func TestBookReader_Interface(t *testing.T) {
	// Compile-time check that Book implements bookReader.
	var _ bookReader = (*Book)(nil)
}

// buildTestEPubBytes creates an in-memory ZIP archive and returns the raw bytes.
func buildTestEPubBytes(t *testing.T, files map[string]string) []byte {
	t.Helper()
	buf := new(bytes.Buffer)
	zw := zip.NewWriter(buf)
	// Write mimetype first if present (ePub spec requires it as first entry).
	if mt, ok := files["mimetype"]; ok {
		fw, err := zw.Create("mimetype")
		if err != nil {
			t.Fatalf("buildTestEPubBytes: create mimetype: %v", err)
		}
		if _, err := io.WriteString(fw, mt); err != nil {
			t.Fatalf("buildTestEPubBytes: write mimetype: %v", err)
		}
	}
	for name, content := range files {
		if name == "mimetype" {
			continue
		}
		fw, err := zw.Create(name)
		if err != nil {
			t.Fatalf("buildTestEPubBytes: create %s: %v", name, err)
		}
		if _, err := io.WriteString(fw, content); err != nil {
			t.Fatalf("buildTestEPubBytes: write %s: %v", name, err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("buildTestEPubBytes: close: %v", err)
	}
	return buf.Bytes()
}

// ============================================================
// Integration Tests: Full Pipeline
// ============================================================

// TestIntegration_EPub2_EndToEnd exercises the full pipeline for an ePub 2 book:
// Open → Metadata → TOC → Chapters (content) → Cover → Close.
func TestIntegration_EPub2_EndToEnd(t *testing.T) {
	opf := `<?xml version="1.0" encoding="UTF-8"?>
<package version="2.0" xmlns="http://www.idpf.org/2007/opf" unique-identifier="bookid">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/" xmlns:opf="http://www.idpf.org/2007/opf">
    <dc:title>Pride and Prejudice</dc:title>
    <dc:creator opf:file-as="Austen, Jane" opf:role="aut">Jane Austen</dc:creator>
    <dc:language>en</dc:language>
    <dc:identifier id="bookid" opf:scheme="ISBN">978-0-14-143951-8</dc:identifier>
    <dc:publisher>Penguin Classics</dc:publisher>
    <dc:date>1813-01-28</dc:date>
    <dc:description>A classic novel of manners.</dc:description>
    <dc:subject>Fiction</dc:subject>
    <dc:subject>Romance</dc:subject>
    <meta name="cover" content="cover-img"/>
  </metadata>
  <manifest>
    <item id="ncx" href="toc.ncx" media-type="application/x-dtbncx+xml"/>
    <item id="ch1" href="chapter01.xhtml" media-type="application/xhtml+xml"/>
    <item id="ch2" href="chapter02.xhtml" media-type="application/xhtml+xml"/>
    <item id="ch3" href="chapter03.xhtml" media-type="application/xhtml+xml"/>
    <item id="cover-img" href="images/cover.jpg" media-type="image/jpeg"/>
  </manifest>
  <spine toc="ncx">
    <itemref idref="ch1"/>
    <itemref idref="ch2"/>
    <itemref idref="ch3" linear="no"/>
  </spine>
  <guide>
    <reference type="cover" title="Cover" href="chapter01.xhtml"/>
  </guide>
</package>`

	ncx := `<?xml version="1.0" encoding="UTF-8"?>
<ncx xmlns="http://www.daisy.org/z3986/2005/ncx/" version="2005-1">
  <navMap>
    <navPoint id="np1" playOrder="1">
      <navLabel><text>Chapter 1</text></navLabel>
      <content src="chapter01.xhtml"/>
    </navPoint>
    <navPoint id="np2" playOrder="2">
      <navLabel><text>Chapter 2</text></navLabel>
      <content src="chapter02.xhtml"/>
    </navPoint>
    <navPoint id="np3" playOrder="3">
      <navLabel><text>Chapter 3</text></navLabel>
      <content src="chapter03.xhtml"/>
    </navPoint>
  </navMap>
</ncx>`

	ch1 := `<?xml version="1.0" encoding="UTF-8"?>
<html xmlns="http://www.w3.org/1999/xhtml">
<head><title>Chapter 1</title></head>
<body>
<h1>Chapter 1</h1>
<p>It is a truth universally acknowledged.</p>
<img src="images/cover.jpg" alt="Cover"/>
</body>
</html>`

	ch2 := `<?xml version="1.0" encoding="UTF-8"?>
<html xmlns="http://www.w3.org/1999/xhtml">
<head><title>Chapter 2</title></head>
<body>
<h1>Chapter 2</h1>
<p>Mr. Bennet was among the earliest.</p>
</body>
</html>`

	ch3 := `<?xml version="1.0" encoding="UTF-8"?>
<html xmlns="http://www.w3.org/1999/xhtml">
<head><title>Chapter 3</title></head>
<body>
<h1>Chapter 3</h1>
<p>Not all that Mrs. Bennet wished.</p>
</body>
</html>`

	files := map[string]string{
		"mimetype":               "application/epub+zip",
		"META-INF/container.xml": validContainerXML,
		"OEBPS/content.opf":      opf,
		"OEBPS/toc.ncx":          ncx,
		"OEBPS/chapter01.xhtml":  ch1,
		"OEBPS/chapter02.xhtml":  ch2,
		"OEBPS/chapter03.xhtml":  ch3,
		"OEBPS/images/cover.jpg": "FAKE-JPEG-DATA",
	}
	fp := buildTestEPubFile(t, files)

	book, err := Open(fp)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer book.Close()

	// No warnings expected for a well-formed ePub 2.
	if len(book.Warnings()) != 0 {
		t.Errorf("unexpected warnings: %v", book.Warnings())
	}

	// --- Metadata ---
	md := book.Metadata()
	if md.Version != "2.0" {
		t.Errorf("Metadata.Version = %q, want %q", md.Version, "2.0")
	}
	if len(md.Titles) != 1 || md.Titles[0] != "Pride and Prejudice" {
		t.Errorf("Metadata.Titles = %v, want [Pride and Prejudice]", md.Titles)
	}
	if len(md.Authors) != 1 || md.Authors[0].Name != "Jane Austen" {
		t.Errorf("Metadata.Authors = %v", md.Authors)
	}
	if md.Authors[0].FileAs != "Austen, Jane" {
		t.Errorf("Authors[0].FileAs = %q, want 'Austen, Jane'", md.Authors[0].FileAs)
	}
	if md.Authors[0].Role != "aut" {
		t.Errorf("Authors[0].Role = %q, want 'aut'", md.Authors[0].Role)
	}
	if md.Publisher != "Penguin Classics" {
		t.Errorf("Publisher = %q", md.Publisher)
	}
	if md.Date != "1813-01-28" {
		t.Errorf("Date = %q", md.Date)
	}
	if len(md.Identifiers) != 1 || md.Identifiers[0].Scheme != "ISBN" {
		t.Errorf("Identifiers = %v", md.Identifiers)
	}
	if len(md.Subjects) != 2 {
		t.Errorf("Subjects = %v", md.Subjects)
	}

	// --- TOC ---
	toc := book.TOC()
	if len(toc) != 3 {
		t.Fatalf("TOC() returned %d items, want 3", len(toc))
	}
	if toc[0].Title != "Chapter 1" || toc[0].SpineIndex != 0 {
		t.Errorf("toc[0] = %+v", toc[0])
	}
	if toc[1].Title != "Chapter 2" || toc[1].SpineIndex != 1 {
		t.Errorf("toc[1] = %+v", toc[1])
	}
	if toc[2].Title != "Chapter 3" || toc[2].SpineIndex != 2 {
		t.Errorf("toc[2] = %+v", toc[2])
	}

	// --- Chapters ---
	chapters := book.Chapters()
	if len(chapters) != 3 {
		t.Fatalf("Chapters() returned %d, want 3", len(chapters))
	}
	if chapters[0].Title != "Chapter 1" {
		t.Errorf("chapters[0].Title = %q", chapters[0].Title)
	}
	if !chapters[0].Linear || !chapters[1].Linear {
		t.Error("ch1 and ch2 should be linear")
	}
	if chapters[2].Linear {
		t.Error("ch3 should be non-linear")
	}

	// Chapter content access.
	text, err := chapters[0].TextContent()
	if err != nil {
		t.Fatalf("TextContent() error = %v", err)
	}
	if !strings.Contains(text, "universally acknowledged") {
		t.Errorf("chapter 1 text missing expected content")
	}

	body, err := chapters[0].BodyHTML()
	if err != nil {
		t.Fatalf("BodyHTML() error = %v", err)
	}
	if !strings.Contains(body, "<h1>Chapter 1</h1>") {
		t.Errorf("BodyHTML missing h1 tag")
	}
	// Image paths should be rewritten.
	if !strings.Contains(body, "OEBPS/images/cover.jpg") {
		t.Errorf("BodyHTML did not rewrite image path")
	}

	// --- Cover ---
	cover, err := book.Cover()
	if err != nil {
		t.Fatalf("Cover() error = %v", err)
	}
	if cover.Path != "OEBPS/images/cover.jpg" {
		t.Errorf("Cover.Path = %q", cover.Path)
	}
	if cover.MediaType != "image/jpeg" {
		t.Errorf("Cover.MediaType = %q", cover.MediaType)
	}
	if string(cover.Data) != "FAKE-JPEG-DATA" {
		t.Errorf("Cover.Data = %q", string(cover.Data))
	}

	// --- Landmarks (ePub 2 has none) ---
	if lm := book.Landmarks(); lm != nil {
		t.Errorf("ePub 2 should have nil landmarks, got %v", lm)
	}
}

// TestIntegration_EPub3_EndToEnd exercises the full pipeline for an ePub 3 book:
// Open → Metadata (with refines) → TOC (nav document) → Chapters → Cover → Close.
func TestIntegration_EPub3_EndToEnd(t *testing.T) {
	opf := `<?xml version="1.0" encoding="UTF-8"?>
<package version="3.0" xmlns="http://www.idpf.org/2007/opf" unique-identifier="uid">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
    <dc:title id="t1">Main Title</dc:title>
    <dc:title id="t2">Subtitle</dc:title>
    <dc:creator id="c1">Author Name</dc:creator>
    <dc:language>en</dc:language>
    <dc:identifier id="uid">urn:uuid:abcd-1234</dc:identifier>
    <dc:publisher>ePub3 Press</dc:publisher>
    <dc:date>2025-01-01</dc:date>
    <dc:description>An ePub 3 test book.</dc:description>
    <dc:subject>Testing</dc:subject>
    <meta property="dcterms:modified">2025-01-15T00:00:00Z</meta>
    <meta refines="#t1" property="display-seq">1</meta>
    <meta refines="#t2" property="display-seq">2</meta>
    <meta refines="#c1" property="file-as">Name, Author</meta>
    <meta refines="#c1" property="role" scheme="marc:relators">aut</meta>
    <meta refines="#uid" property="identifier-type">UUID</meta>
  </metadata>
  <manifest>
    <item id="nav" href="nav.xhtml" media-type="application/xhtml+xml" properties="nav"/>
    <item id="ch1" href="chapter01.xhtml" media-type="application/xhtml+xml"/>
    <item id="ch2" href="chapter02.xhtml" media-type="application/xhtml+xml"/>
    <item id="cover-img" href="images/cover.png" media-type="image/png" properties="cover-image"/>
  </manifest>
  <spine>
    <itemref idref="ch1"/>
    <itemref idref="ch2"/>
  </spine>
</package>`

	nav := `<?xml version="1.0" encoding="UTF-8"?>
<html xmlns="http://www.w3.org/1999/xhtml" xmlns:epub="http://www.idpf.org/2007/ops">
<head><title>Navigation</title></head>
<body>
<nav epub:type="toc">
  <h2>Table of Contents</h2>
  <ol>
    <li><a href="chapter01.xhtml">First Chapter</a></li>
    <li><a href="chapter02.xhtml">Second Chapter</a></li>
  </ol>
</nav>
<nav epub:type="landmarks">
  <h2>Landmarks</h2>
  <ol>
    <li><a epub:type="bodymatter" href="chapter01.xhtml">Start of Content</a></li>
  </ol>
</nav>
</body>
</html>`

	ch1 := `<?xml version="1.0" encoding="UTF-8"?>
<html xmlns="http://www.w3.org/1999/xhtml">
<head><title>First Chapter</title></head>
<body>
<h1>First Chapter</h1>
<p>This is the first chapter of the ePub 3 book.</p>
</body>
</html>`

	ch2 := `<?xml version="1.0" encoding="UTF-8"?>
<html xmlns="http://www.w3.org/1999/xhtml">
<head><title>Second Chapter</title></head>
<body>
<h1>Second Chapter</h1>
<p>This is the second chapter.</p>
</body>
</html>`

	files := map[string]string{
		"mimetype":               "application/epub+zip",
		"META-INF/container.xml": validContainerXML,
		"OEBPS/content.opf":      opf,
		"OEBPS/nav.xhtml":        nav,
		"OEBPS/chapter01.xhtml":  ch1,
		"OEBPS/chapter02.xhtml":  ch2,
		"OEBPS/images/cover.png": "FAKE-PNG-DATA",
	}
	fp := buildTestEPubFile(t, files)

	book, err := Open(fp)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer book.Close()

	if len(book.Warnings()) != 0 {
		t.Errorf("unexpected warnings: %v", book.Warnings())
	}

	// --- Metadata ---
	md := book.Metadata()
	if md.Version != "3.0" {
		t.Errorf("Version = %q, want 3.0", md.Version)
	}
	// Titles ordered by display-seq.
	if len(md.Titles) != 2 || md.Titles[0] != "Main Title" || md.Titles[1] != "Subtitle" {
		t.Errorf("Titles = %v, want [Main Title, Subtitle]", md.Titles)
	}
	if len(md.Authors) != 1 || md.Authors[0].Name != "Author Name" {
		t.Errorf("Authors = %v", md.Authors)
	}
	if md.Authors[0].FileAs != "Name, Author" {
		t.Errorf("Authors[0].FileAs = %q", md.Authors[0].FileAs)
	}
	if md.Authors[0].Role != "aut" {
		t.Errorf("Authors[0].Role = %q", md.Authors[0].Role)
	}
	if len(md.Identifiers) != 1 || md.Identifiers[0].Scheme != "UUID" {
		t.Errorf("Identifiers = %v", md.Identifiers)
	}
	if md.Publisher != "ePub3 Press" {
		t.Errorf("Publisher = %q", md.Publisher)
	}

	// --- TOC (from nav document) ---
	toc := book.TOC()
	if len(toc) != 2 {
		t.Fatalf("TOC() returned %d items, want 2", len(toc))
	}
	if toc[0].Title != "First Chapter" || toc[0].SpineIndex != 0 {
		t.Errorf("toc[0] = %+v", toc[0])
	}
	if toc[1].Title != "Second Chapter" || toc[1].SpineIndex != 1 {
		t.Errorf("toc[1] = %+v", toc[1])
	}

	// --- Landmarks ---
	lm := book.Landmarks()
	if len(lm) != 1 {
		t.Fatalf("Landmarks() returned %d items, want 1", len(lm))
	}
	if lm[0].Title != "Start of Content" {
		t.Errorf("landmark title = %q", lm[0].Title)
	}

	// --- Chapters ---
	chapters := book.Chapters()
	if len(chapters) != 2 {
		t.Fatalf("Chapters() returned %d, want 2", len(chapters))
	}
	if chapters[0].Title != "First Chapter" {
		t.Errorf("chapters[0].Title = %q", chapters[0].Title)
	}

	text, err := chapters[1].TextContent()
	if err != nil {
		t.Fatalf("TextContent() error = %v", err)
	}
	if !strings.Contains(text, "second chapter") {
		t.Errorf("chapter 2 text missing expected content")
	}

	// --- Cover (strategy 1: cover-image property) ---
	cover, err := book.Cover()
	if err != nil {
		t.Fatalf("Cover() error = %v", err)
	}
	if cover.Path != "OEBPS/images/cover.png" {
		t.Errorf("Cover.Path = %q", cover.Path)
	}
	if cover.MediaType != "image/png" {
		t.Errorf("Cover.MediaType = %q", cover.MediaType)
	}
}

// TestIntegration_MissingContainerXML_FallbackOPF tests that Open() succeeds
// when META-INF/container.xml is absent, falling back to .opf file scanning.
func TestIntegration_MissingContainerXML_FallbackOPF(t *testing.T) {
	opf := `<?xml version="1.0" encoding="UTF-8"?>
<package version="2.0" xmlns="http://www.idpf.org/2007/opf" unique-identifier="uid">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
    <dc:title>Fallback Test</dc:title>
    <dc:language>en</dc:language>
    <dc:identifier id="uid">test-fallback-001</dc:identifier>
  </metadata>
  <manifest>
    <item id="ch1" href="chapter.xhtml" media-type="application/xhtml+xml"/>
  </manifest>
  <spine>
    <itemref idref="ch1"/>
  </spine>
</package>`

	ch := `<html xmlns="http://www.w3.org/1999/xhtml"><body><p>Hello</p></body></html>`

	// No container.xml, no mimetype — just the OPF and a chapter.
	files := map[string]string{
		"content.opf":   opf,
		"chapter.xhtml": ch,
	}
	fp := buildTestEPubFile(t, files)

	book, err := Open(fp)
	if err != nil {
		t.Fatalf("Open() error = %v; expected fallback to .opf scan", err)
	}
	defer book.Close()

	// Should have a warning about missing mimetype.
	if len(book.Warnings()) == 0 {
		t.Error("expected warning for missing mimetype")
	}

	md := book.Metadata()
	if len(md.Titles) != 1 || md.Titles[0] != "Fallback Test" {
		t.Errorf("Metadata.Titles = %v", md.Titles)
	}

	chapters := book.Chapters()
	if len(chapters) != 1 {
		t.Fatalf("Chapters() returned %d, want 1", len(chapters))
	}

	text, err := chapters[0].TextContent()
	if err != nil {
		t.Fatalf("TextContent() error = %v", err)
	}
	if !strings.Contains(text, "Hello") {
		t.Error("chapter text missing expected content")
	}
}

// TestIntegration_HTMLEntitiesInOPF tests that HTML entities in the OPF metadata
// are correctly handled through the full Open → Metadata pipeline.
func TestIntegration_HTMLEntitiesInOPF(t *testing.T) {
	opf := `<?xml version="1.0" encoding="UTF-8"?>
<package version="2.0" xmlns="http://www.idpf.org/2007/opf" unique-identifier="uid">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
    <dc:title>Caf&eacute; &amp; Cr&egrave;me &mdash; A Story</dc:title>
    <dc:creator>Ren&eacute; M&uuml;ller</dc:creator>
    <dc:language>fr</dc:language>
    <dc:identifier id="uid">entity-test</dc:identifier>
    <dc:description>Characters: &laquo;hello&raquo; &ndash; &hellip;</dc:description>
  </metadata>
  <manifest>
    <item id="ch1" href="ch.xhtml" media-type="application/xhtml+xml"/>
  </manifest>
  <spine>
    <itemref idref="ch1"/>
  </spine>
</package>`

	files := map[string]string{
		"mimetype":               "application/epub+zip",
		"META-INF/container.xml": validContainerXML,
		"OEBPS/content.opf":      opf,
		"OEBPS/ch.xhtml":         `<html><body><p>Content</p></body></html>`,
	}
	fp := buildTestEPubFile(t, files)

	book, err := Open(fp)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer book.Close()

	md := book.Metadata()
	wantTitle := "Caf\u00e9 & Cr\u00e8me \u2014 A Story"
	if len(md.Titles) != 1 || md.Titles[0] != wantTitle {
		t.Errorf("Title = %q, want %q", md.Titles[0], wantTitle)
	}
	wantAuthor := "Ren\u00e9 M\u00fcller"
	if len(md.Authors) != 1 || md.Authors[0].Name != wantAuthor {
		t.Errorf("Author = %q, want %q", md.Authors[0].Name, wantAuthor)
	}
	wantDesc := "Characters: \u00abhello\u00bb \u2013 \u2026"
	if md.Description != wantDesc {
		t.Errorf("Description = %q, want %q", md.Description, wantDesc)
	}
}

// TestIntegration_HTMLEntitiesInNCX tests that HTML entities in the NCX TOC
// are correctly handled through the full Open → TOC pipeline.
func TestIntegration_HTMLEntitiesInNCX(t *testing.T) {
	opf := `<?xml version="1.0" encoding="UTF-8"?>
<package version="2.0" xmlns="http://www.idpf.org/2007/opf" unique-identifier="uid">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
    <dc:title>Entity TOC Test</dc:title>
    <dc:language>en</dc:language>
    <dc:identifier id="uid">entity-toc</dc:identifier>
  </metadata>
  <manifest>
    <item id="ch1" href="ch.xhtml" media-type="application/xhtml+xml"/>
    <item id="ncx" href="toc.ncx" media-type="application/x-dtbncx+xml"/>
  </manifest>
  <spine toc="ncx">
    <itemref idref="ch1"/>
  </spine>
</package>`

	ncx := `<?xml version="1.0" encoding="UTF-8"?>
<ncx xmlns="http://www.daisy.org/z3986/2005/ncx/" version="2005-1">
  <navMap>
    <navPoint id="np1" playOrder="1">
      <navLabel><text>Tom &amp; Jerry &mdash; Adventures</text></navLabel>
      <content src="ch.xhtml"/>
    </navPoint>
  </navMap>
</ncx>`

	files := map[string]string{
		"mimetype":               "application/epub+zip",
		"META-INF/container.xml": validContainerXML,
		"OEBPS/content.opf":      opf,
		"OEBPS/toc.ncx":          ncx,
		"OEBPS/ch.xhtml":         `<html><body><p>Content</p></body></html>`,
	}
	fp := buildTestEPubFile(t, files)

	book, err := Open(fp)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer book.Close()

	toc := book.TOC()
	if len(toc) != 1 {
		t.Fatalf("TOC() returned %d items, want 1", len(toc))
	}
	want := "Tom & Jerry \u2014 Adventures"
	if toc[0].Title != want {
		t.Errorf("toc[0].Title = %q, want %q", toc[0].Title, want)
	}
}

// TestIntegration_CaseInsensitivePaths tests that file references with
// mismatched case are resolved through case-insensitive fallback.
func TestIntegration_CaseInsensitivePaths(t *testing.T) {
	// container.xml references "OEBPS/content.opf" but the actual
	// ZIP entry uses different casing. Also the OPF references chapter
	// files with different casing than what is stored in the archive.
	container := `<?xml version="1.0" encoding="UTF-8"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
  <rootfiles>
    <rootfile full-path="OEBPS/Content.OPF" media-type="application/oebps-package+xml"/>
  </rootfiles>
</container>`

	opf := `<?xml version="1.0" encoding="UTF-8"?>
<package version="2.0" xmlns="http://www.idpf.org/2007/opf" unique-identifier="uid">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
    <dc:title>Case Test</dc:title>
    <dc:language>en</dc:language>
    <dc:identifier id="uid">case-test</dc:identifier>
  </metadata>
  <manifest>
    <item id="ch1" href="Chapter01.XHTML" media-type="application/xhtml+xml"/>
    <item id="ncx" href="toc.ncx" media-type="application/x-dtbncx+xml"/>
  </manifest>
  <spine toc="ncx">
    <itemref idref="ch1"/>
  </spine>
</package>`

	ncx := `<?xml version="1.0" encoding="UTF-8"?>
<ncx xmlns="http://www.daisy.org/z3986/2005/ncx/" version="2005-1">
  <navMap>
    <navPoint id="np1" playOrder="1">
      <navLabel><text>Chapter 1</text></navLabel>
      <content src="Chapter01.XHTML"/>
    </navPoint>
  </navMap>
</ncx>`

	// ZIP entries use lowercase but OPF references mixed case.
	files := map[string]string{
		"mimetype":               "application/epub+zip",
		"META-INF/container.xml": container,
		"OEBPS/content.opf":      opf, // stored as lowercase
		"OEBPS/toc.ncx":          ncx,
		"OEBPS/chapter01.xhtml":  `<html><body><p>Hello from ch1</p></body></html>`,
	}
	fp := buildTestEPubFile(t, files)

	book, err := Open(fp)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer book.Close()

	md := book.Metadata()
	if len(md.Titles) != 1 || md.Titles[0] != "Case Test" {
		t.Errorf("Metadata.Titles = %v", md.Titles)
	}

	chapters := book.Chapters()
	if len(chapters) != 1 {
		t.Fatalf("Chapters() returned %d, want 1", len(chapters))
	}

	// The chapter content should be readable despite case mismatch.
	raw, err := chapters[0].RawContent()
	if err != nil {
		t.Fatalf("RawContent() error = %v", err)
	}
	if !strings.Contains(string(raw), "Hello from ch1") {
		t.Errorf("RawContent missing expected text")
	}
}

// TestIntegration_BOM_InXHTML tests that XHTML files with a UTF-8 BOM are
// correctly processed through the full pipeline.
func TestIntegration_BOM_InXHTML(t *testing.T) {
	opf := `<?xml version="1.0" encoding="UTF-8"?>
<package version="2.0" xmlns="http://www.idpf.org/2007/opf" unique-identifier="uid">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
    <dc:title>BOM Test</dc:title>
    <dc:language>en</dc:language>
    <dc:identifier id="uid">bom-test</dc:identifier>
  </metadata>
  <manifest>
    <item id="ch1" href="ch.xhtml" media-type="application/xhtml+xml"/>
    <item id="ncx" href="toc.ncx" media-type="application/x-dtbncx+xml"/>
  </manifest>
  <spine toc="ncx">
    <itemref idref="ch1"/>
  </spine>
</package>`

	bomNCX := "\xEF\xBB\xBF" + `<?xml version="1.0" encoding="UTF-8"?>
<ncx xmlns="http://www.daisy.org/z3986/2005/ncx/" version="2005-1">
  <navMap>
    <navPoint id="np1" playOrder="1">
      <navLabel><text>BOM Chapter</text></navLabel>
      <content src="ch.xhtml"/>
    </navPoint>
  </navMap>
</ncx>`

	// Chapter content with BOM.
	bomChapter := "\xEF\xBB\xBF" + `<?xml version="1.0" encoding="UTF-8"?>
<html xmlns="http://www.w3.org/1999/xhtml">
<head><title>BOM</title></head>
<body><p>Content with BOM prefix.</p></body>
</html>`

	files := map[string]string{
		"mimetype":               "application/epub+zip",
		"META-INF/container.xml": validContainerXML,
		"OEBPS/content.opf":      opf,
		"OEBPS/toc.ncx":          bomNCX,
		"OEBPS/ch.xhtml":         bomChapter,
	}
	fp := buildTestEPubFile(t, files)

	book, err := Open(fp)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer book.Close()

	// TOC should be parsed despite BOM in NCX.
	toc := book.TOC()
	if len(toc) != 1 {
		t.Fatalf("TOC() returned %d items, want 1", len(toc))
	}
	if toc[0].Title != "BOM Chapter" {
		t.Errorf("toc[0].Title = %q", toc[0].Title)
	}

	// Chapter content should have BOM stripped.
	chapters := book.Chapters()
	if len(chapters) != 1 {
		t.Fatalf("Chapters() returned %d, want 1", len(chapters))
	}
	raw, err := chapters[0].RawContent()
	if err != nil {
		t.Fatalf("RawContent() error = %v", err)
	}
	if len(raw) >= 3 && raw[0] == 0xEF && raw[1] == 0xBB && raw[2] == 0xBF {
		t.Error("RawContent still has BOM")
	}

	text, err := chapters[0].TextContent()
	if err != nil {
		t.Fatalf("TextContent() error = %v", err)
	}
	if !strings.Contains(text, "Content with BOM prefix.") {
		t.Errorf("TextContent = %q, missing expected text", text)
	}
}

// TestIntegration_DRM_AdobeADEPT tests that an ePub with Adobe ADEPT DRM
// is rejected with ErrDRMProtected.
func TestIntegration_DRM_AdobeADEPT(t *testing.T) {
	files := map[string]string{
		"mimetype":               "application/epub+zip",
		"META-INF/container.xml": validContainerXML,
		"OEBPS/content.opf":      `<?xml version="1.0"?><package/>`,
		"META-INF/encryption.xml": `<?xml version="1.0" encoding="UTF-8"?>
<encryption xmlns="urn:oasis:names:tc:opendocument:xmlns:container"
            xmlns:enc="http://www.w3.org/2001/04/xmlenc#">
  <enc:EncryptedData>
    <enc:EncryptionMethod Algorithm="http://www.w3.org/2001/04/xmlenc#aes128-cbc"/>
    <KeyInfo xmlns="http://www.w3.org/2000/09/xmldsig#">
      <resource xmlns="http://ns.adobe.com/adept"/>
    </KeyInfo>
  </enc:EncryptedData>
</encryption>`,
	}
	fp := buildTestEPubFile(t, files)

	_, err := Open(fp)
	if !errors.Is(err, ErrDRMProtected) {
		t.Errorf("Open() error = %v, want ErrDRMProtected", err)
	}
}

// TestIntegration_DRM_AppleFairPlay tests that an ePub with Apple FairPlay DRM
// (sinf.xml) is rejected with ErrDRMProtected.
func TestIntegration_DRM_AppleFairPlay(t *testing.T) {
	files := map[string]string{
		"mimetype":               "application/epub+zip",
		"META-INF/container.xml": validContainerXML,
		"OEBPS/content.opf":      `<?xml version="1.0"?><package/>`,
		"META-INF/sinf.xml":      "<sinf/>",
	}
	fp := buildTestEPubFile(t, files)

	_, err := Open(fp)
	if !errors.Is(err, ErrDRMProtected) {
		t.Errorf("Open() error = %v, want ErrDRMProtected", err)
	}
}

// TestIntegration_FontObfuscationOnly tests that an ePub with only font
// obfuscation (no actual DRM) opens successfully with a warning.
func TestIntegration_FontObfuscationOnly(t *testing.T) {
	opf := `<?xml version="1.0" encoding="UTF-8"?>
<package version="2.0" xmlns="http://www.idpf.org/2007/opf" unique-identifier="uid">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
    <dc:title>Font Obfuscation Test</dc:title>
    <dc:language>en</dc:language>
    <dc:identifier id="uid">font-obf-test</dc:identifier>
  </metadata>
  <manifest>
    <item id="ch1" href="ch.xhtml" media-type="application/xhtml+xml"/>
  </manifest>
  <spine>
    <itemref idref="ch1"/>
  </spine>
</package>`

	files := map[string]string{
		"mimetype":               "application/epub+zip",
		"META-INF/container.xml": validContainerXML,
		"OEBPS/content.opf":      opf,
		"OEBPS/ch.xhtml":         `<html><body><p>Hello</p></body></html>`,
		"META-INF/encryption.xml": `<?xml version="1.0" encoding="UTF-8"?>
<encryption xmlns="urn:oasis:names:tc:opendocument:xmlns:container"
            xmlns:enc="http://www.w3.org/2001/04/xmlenc#">
  <enc:EncryptedData>
    <enc:EncryptionMethod Algorithm="http://www.idpf.org/2008/embedding"/>
    <enc:CipherData>
      <enc:CipherReference URI="OEBPS/fonts/myfont.otf"/>
    </enc:CipherData>
  </enc:EncryptedData>
  <enc:EncryptedData>
    <enc:EncryptionMethod Algorithm="http://ns.adobe.com/pdf/enc#RC"/>
    <enc:CipherData>
      <enc:CipherReference URI="OEBPS/fonts/another.ttf"/>
    </enc:CipherData>
  </enc:EncryptedData>
</encryption>`,
	}
	fp := buildTestEPubFile(t, files)

	book, err := Open(fp)
	if err != nil {
		t.Fatalf("Open() error = %v; font obfuscation should not block opening", err)
	}
	defer book.Close()

	// Must have font obfuscation warning.
	foundWarning := false
	for _, w := range book.Warnings() {
		if strings.Contains(w, "font obfuscation") {
			foundWarning = true
		}
	}
	if !foundWarning {
		t.Errorf("expected font obfuscation warning, got: %v", book.Warnings())
	}

	// Book should still be fully functional.
	md := book.Metadata()
	if len(md.Titles) != 1 || md.Titles[0] != "Font Obfuscation Test" {
		t.Errorf("Metadata.Titles = %v", md.Titles)
	}

	chapters := book.Chapters()
	if len(chapters) != 1 {
		t.Fatalf("Chapters() returned %d, want 1", len(chapters))
	}
	text, err := chapters[0].TextContent()
	if err != nil {
		t.Fatalf("TextContent() error = %v", err)
	}
	if !strings.Contains(text, "Hello") {
		t.Error("chapter content missing")
	}
}

// TestIntegration_NewReader_EPub2 tests the NewReader path with a complete ePub 2.
func TestIntegration_NewReader_EPub2(t *testing.T) {
	opf := `<?xml version="1.0" encoding="UTF-8"?>
<package version="2.0" xmlns="http://www.idpf.org/2007/opf" unique-identifier="uid">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
    <dc:title>Reader Test</dc:title>
    <dc:language>en</dc:language>
    <dc:identifier id="uid">reader-test</dc:identifier>
  </metadata>
  <manifest>
    <item id="ch1" href="ch.xhtml" media-type="application/xhtml+xml"/>
    <item id="ncx" href="toc.ncx" media-type="application/x-dtbncx+xml"/>
  </manifest>
  <spine toc="ncx">
    <itemref idref="ch1"/>
  </spine>
</package>`

	ncx := `<?xml version="1.0" encoding="UTF-8"?>
<ncx xmlns="http://www.daisy.org/z3986/2005/ncx/" version="2005-1">
  <navMap>
    <navPoint id="np1" playOrder="1">
      <navLabel><text>My Chapter</text></navLabel>
      <content src="ch.xhtml"/>
    </navPoint>
  </navMap>
</ncx>`

	files := map[string]string{
		"mimetype":               "application/epub+zip",
		"META-INF/container.xml": validContainerXML,
		"OEBPS/content.opf":      opf,
		"OEBPS/toc.ncx":          ncx,
		"OEBPS/ch.xhtml":         `<html><body><p>NewReader content</p></body></html>`,
	}
	data := buildTestEPubBytes(t, files)

	book, err := NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("NewReader() error = %v", err)
	}
	defer book.Close()

	md := book.Metadata()
	if len(md.Titles) != 1 || md.Titles[0] != "Reader Test" {
		t.Errorf("Metadata.Titles = %v", md.Titles)
	}

	toc := book.TOC()
	if len(toc) != 1 || toc[0].Title != "My Chapter" {
		t.Errorf("TOC = %+v", toc)
	}

	chapters := book.Chapters()
	if len(chapters) != 1 {
		t.Fatalf("Chapters() = %d, want 1", len(chapters))
	}
	text, err := chapters[0].TextContent()
	if err != nil {
		t.Fatalf("TextContent() error = %v", err)
	}
	if !strings.Contains(text, "NewReader content") {
		t.Error("missing expected text")
	}
}

// TestIntegration_ContentChapters_GutenbergFiltering tests the Gutenberg
// license detection through the full pipeline.
func TestIntegration_ContentChapters_GutenbergFiltering(t *testing.T) {
	opf := `<?xml version="1.0" encoding="UTF-8"?>
<package version="2.0" xmlns="http://www.idpf.org/2007/opf" unique-identifier="uid">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
    <dc:title>Gutenberg Book</dc:title>
    <dc:language>en</dc:language>
    <dc:identifier id="uid">gb-test</dc:identifier>
  </metadata>
  <manifest>
    <item id="ch1" href="ch1.xhtml" media-type="application/xhtml+xml"/>
    <item id="ch2" href="ch2.xhtml" media-type="application/xhtml+xml"/>
    <item id="lic" href="license.xhtml" media-type="application/xhtml+xml"/>
    <item id="ncx" href="toc.ncx" media-type="application/x-dtbncx+xml"/>
  </manifest>
  <spine toc="ncx">
    <itemref idref="ch1"/>
    <itemref idref="ch2"/>
    <itemref idref="lic"/>
  </spine>
</package>`

	ncx := `<?xml version="1.0" encoding="UTF-8"?>
<ncx xmlns="http://www.daisy.org/z3986/2005/ncx/" version="2005-1">
  <navMap>
    <navPoint id="np1" playOrder="1">
      <navLabel><text>Chapter 1</text></navLabel>
      <content src="ch1.xhtml"/>
    </navPoint>
    <navPoint id="np2" playOrder="2">
      <navLabel><text>Chapter 2</text></navLabel>
      <content src="ch2.xhtml"/>
    </navPoint>
  </navMap>
</ncx>`

	license := `<?xml version="1.0" encoding="UTF-8"?>
<html xmlns="http://www.w3.org/1999/xhtml">
<head><title>License</title></head>
<body>
<p>*** START OF THE PROJECT GUTENBERG LICENSE ***</p>
<p>This eBook is for the use of anyone anywhere at no cost.</p>
<p>*** END OF THE PROJECT GUTENBERG LICENSE ***</p>
</body>
</html>`

	files := map[string]string{
		"mimetype":               "application/epub+zip",
		"META-INF/container.xml": validContainerXML,
		"OEBPS/content.opf":      opf,
		"OEBPS/toc.ncx":          ncx,
		"OEBPS/ch1.xhtml":        `<html><body><p>Real content 1</p></body></html>`,
		"OEBPS/ch2.xhtml":        `<html><body><p>Real content 2</p></body></html>`,
		"OEBPS/license.xhtml":    license,
	}
	fp := buildTestEPubFile(t, files)

	book, err := Open(fp)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer book.Close()

	// All 3 chapters in spine.
	all := book.Chapters()
	if len(all) != 3 {
		t.Fatalf("Chapters() = %d, want 3", len(all))
	}

	// ContentChapters() should filter out license.
	content := book.ContentChapters()
	if len(content) != 2 {
		t.Fatalf("ContentChapters() = %d, want 2", len(content))
	}
	if content[0].ID != "ch1" || content[1].ID != "ch2" {
		t.Errorf("ContentChapters IDs = [%s, %s]", content[0].ID, content[1].ID)
	}

	// Verify cached chapters now have IsLicense set.
	all = book.Chapters()
	if !all[2].IsLicense {
		t.Error("chapters[2].IsLicense should be true")
	}
}

// TestIntegration_SpineRanges tests that TOC spine ranges are correctly computed
// across the full pipeline.
func TestIntegration_SpineRanges(t *testing.T) {
	opf := `<?xml version="1.0" encoding="UTF-8"?>
<package version="2.0" xmlns="http://www.idpf.org/2007/opf" unique-identifier="uid">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
    <dc:title>Range Test</dc:title>
    <dc:language>en</dc:language>
    <dc:identifier id="uid">range-test</dc:identifier>
  </metadata>
  <manifest>
    <item id="s0" href="s0.xhtml" media-type="application/xhtml+xml"/>
    <item id="s1" href="s1.xhtml" media-type="application/xhtml+xml"/>
    <item id="s2" href="s2.xhtml" media-type="application/xhtml+xml"/>
    <item id="s3" href="s3.xhtml" media-type="application/xhtml+xml"/>
    <item id="ncx" href="toc.ncx" media-type="application/x-dtbncx+xml"/>
  </manifest>
  <spine toc="ncx">
    <itemref idref="s0"/>
    <itemref idref="s1"/>
    <itemref idref="s2"/>
    <itemref idref="s3"/>
  </spine>
</package>`

	// TOC maps entry 1 → s0, entry 2 → s2 (skipping s1).
	ncx := `<?xml version="1.0" encoding="UTF-8"?>
<ncx xmlns="http://www.daisy.org/z3986/2005/ncx/" version="2005-1">
  <navMap>
    <navPoint id="np1" playOrder="1">
      <navLabel><text>Part 1</text></navLabel>
      <content src="s0.xhtml"/>
    </navPoint>
    <navPoint id="np2" playOrder="2">
      <navLabel><text>Part 2</text></navLabel>
      <content src="s2.xhtml"/>
    </navPoint>
  </navMap>
</ncx>`

	page := `<html><body><p>page</p></body></html>`
	files := map[string]string{
		"mimetype":               "application/epub+zip",
		"META-INF/container.xml": validContainerXML,
		"OEBPS/content.opf":      opf,
		"OEBPS/toc.ncx":          ncx,
		"OEBPS/s0.xhtml":         page,
		"OEBPS/s1.xhtml":         page,
		"OEBPS/s2.xhtml":         page,
		"OEBPS/s3.xhtml":         page,
	}
	fp := buildTestEPubFile(t, files)

	book, err := Open(fp)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer book.Close()

	toc := book.TOC()
	if len(toc) != 2 {
		t.Fatalf("TOC() = %d items, want 2", len(toc))
	}

	// Part 1: spine[0:2] (covers s0 and s1).
	if toc[0].SpineIndex != 0 || toc[0].SpineEndIndex != 2 {
		t.Errorf("toc[0] range = [%d:%d], want [0:2]", toc[0].SpineIndex, toc[0].SpineEndIndex)
	}
	// Part 2: spine[2:4] (covers s2 and s3, to end).
	if toc[1].SpineIndex != 2 || toc[1].SpineEndIndex != 4 {
		t.Errorf("toc[1] range = [%d:%d], want [2:4]", toc[1].SpineIndex, toc[1].SpineEndIndex)
	}
}
