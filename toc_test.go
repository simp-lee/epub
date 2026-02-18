package epub

import (
	"strings"
	"testing"
)

func TestParseNCX_FlatTOC(t *testing.T) {
	ncxData := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<ncx xmlns="http://www.daisy.org/z3986/2005/ncx/" version="2005-1">
  <navMap>
    <navPoint id="np1" playOrder="1">
      <navLabel><text>Chapter 1</text></navLabel>
      <content src="chapter1.xhtml"/>
    </navPoint>
    <navPoint id="np2" playOrder="2">
      <navLabel><text>Chapter 2</text></navLabel>
      <content src="chapter2.xhtml"/>
    </navPoint>
    <navPoint id="np3" playOrder="3">
      <navLabel><text>Chapter 3</text></navLabel>
      <content src="chapter3.xhtml"/>
    </navPoint>
  </navMap>
</ncx>`)

	items, err := parseNCX(ncxData, "OEBPS/toc.ncx")
	if err != nil {
		t.Fatalf("parseNCX returned error: %v", err)
	}

	if len(items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(items))
	}

	tests := []struct {
		title string
		href  string
	}{
		{"Chapter 1", "OEBPS/chapter1.xhtml"},
		{"Chapter 2", "OEBPS/chapter2.xhtml"},
		{"Chapter 3", "OEBPS/chapter3.xhtml"},
	}

	for i, tt := range tests {
		if items[i].Title != tt.title {
			t.Errorf("item[%d].Title = %q, want %q", i, items[i].Title, tt.title)
		}
		if items[i].Href != tt.href {
			t.Errorf("item[%d].Href = %q, want %q", i, items[i].Href, tt.href)
		}
		if items[i].SpineIndex != -1 {
			t.Errorf("item[%d].SpineIndex = %d, want -1", i, items[i].SpineIndex)
		}
		if len(items[i].Children) != 0 {
			t.Errorf("item[%d].Children length = %d, want 0", i, len(items[i].Children))
		}
	}
}

func TestParseNCX_NestedTOC(t *testing.T) {
	ncxData := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<ncx xmlns="http://www.daisy.org/z3986/2005/ncx/" version="2005-1">
  <navMap>
    <navPoint id="np1" playOrder="1">
      <navLabel><text>Part I</text></navLabel>
      <content src="part1.xhtml"/>
      <navPoint id="np1.1" playOrder="2">
        <navLabel><text>Chapter 1</text></navLabel>
        <content src="chapter1.xhtml"/>
        <navPoint id="np1.1.1" playOrder="3">
          <navLabel><text>Section 1.1</text></navLabel>
          <content src="chapter1.xhtml#sec1"/>
        </navPoint>
      </navPoint>
      <navPoint id="np1.2" playOrder="4">
        <navLabel><text>Chapter 2</text></navLabel>
        <content src="chapter2.xhtml"/>
      </navPoint>
    </navPoint>
    <navPoint id="np2" playOrder="5">
      <navLabel><text>Part II</text></navLabel>
      <content src="part2.xhtml"/>
    </navPoint>
  </navMap>
</ncx>`)

	items, err := parseNCX(ncxData, "OEBPS/toc.ncx")
	if err != nil {
		t.Fatalf("parseNCX returned error: %v", err)
	}

	if len(items) != 2 {
		t.Fatalf("expected 2 top-level items, got %d", len(items))
	}

	// Part I
	part1 := items[0]
	if part1.Title != "Part I" {
		t.Errorf("part1.Title = %q, want %q", part1.Title, "Part I")
	}
	if part1.Href != "OEBPS/part1.xhtml" {
		t.Errorf("part1.Href = %q, want %q", part1.Href, "OEBPS/part1.xhtml")
	}
	if len(part1.Children) != 2 {
		t.Fatalf("Part I children count = %d, want 2", len(part1.Children))
	}

	// Chapter 1 (under Part I)
	ch1 := part1.Children[0]
	if ch1.Title != "Chapter 1" {
		t.Errorf("ch1.Title = %q, want %q", ch1.Title, "Chapter 1")
	}
	if ch1.Href != "OEBPS/chapter1.xhtml" {
		t.Errorf("ch1.Href = %q, want %q", ch1.Href, "OEBPS/chapter1.xhtml")
	}
	if len(ch1.Children) != 1 {
		t.Fatalf("Chapter 1 children count = %d, want 1", len(ch1.Children))
	}

	// Section 1.1 (under Chapter 1) — has fragment
	sec1 := ch1.Children[0]
	if sec1.Title != "Section 1.1" {
		t.Errorf("sec1.Title = %q, want %q", sec1.Title, "Section 1.1")
	}
	if sec1.Href != "OEBPS/chapter1.xhtml#sec1" {
		t.Errorf("sec1.Href = %q, want %q", sec1.Href, "OEBPS/chapter1.xhtml#sec1")
	}

	// Chapter 2 (under Part I)
	ch2 := part1.Children[1]
	if ch2.Title != "Chapter 2" {
		t.Errorf("ch2.Title = %q, want %q", ch2.Title, "Chapter 2")
	}

	// Part II
	part2 := items[1]
	if part2.Title != "Part II" {
		t.Errorf("part2.Title = %q, want %q", part2.Title, "Part II")
	}
	if part2.Href != "OEBPS/part2.xhtml" {
		t.Errorf("part2.Href = %q, want %q", part2.Href, "OEBPS/part2.xhtml")
	}
	if len(part2.Children) != 0 {
		t.Errorf("Part II children count = %d, want 0", len(part2.Children))
	}
}

func TestParseNCX_FragmentHref(t *testing.T) {
	ncxData := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<ncx xmlns="http://www.daisy.org/z3986/2005/ncx/" version="2005-1">
  <navMap>
    <navPoint id="np1" playOrder="1">
      <navLabel><text>Introduction</text></navLabel>
      <content src="main.xhtml#intro"/>
    </navPoint>
    <navPoint id="np2" playOrder="2">
      <navLabel><text>Conclusion</text></navLabel>
      <content src="main.xhtml#conclusion"/>
    </navPoint>
  </navMap>
</ncx>`)

	items, err := parseNCX(ncxData, "OEBPS/toc.ncx")
	if err != nil {
		t.Fatalf("parseNCX returned error: %v", err)
	}

	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}

	// Fragment should be preserved in Href.
	if items[0].Href != "OEBPS/main.xhtml#intro" {
		t.Errorf("item[0].Href = %q, want %q", items[0].Href, "OEBPS/main.xhtml#intro")
	}
	if items[1].Href != "OEBPS/main.xhtml#conclusion" {
		t.Errorf("item[1].Href = %q, want %q", items[1].Href, "OEBPS/main.xhtml#conclusion")
	}
}

func TestParseNCX_EmptyNavMap(t *testing.T) {
	ncxData := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<ncx xmlns="http://www.daisy.org/z3986/2005/ncx/" version="2005-1">
  <navMap/>
</ncx>`)

	items, err := parseNCX(ncxData, "OEBPS/toc.ncx")
	if err != nil {
		t.Fatalf("parseNCX returned error: %v", err)
	}

	if len(items) != 0 {
		t.Errorf("expected 0 items for empty navMap, got %d", len(items))
	}
}

func TestParseNCX_RootLevelNCX(t *testing.T) {
	// NCX file at the ZIP root (no subdirectory).
	ncxData := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<ncx xmlns="http://www.daisy.org/z3986/2005/ncx/" version="2005-1">
  <navMap>
    <navPoint id="np1" playOrder="1">
      <navLabel><text>Chapter 1</text></navLabel>
      <content src="chapter1.xhtml"/>
    </navPoint>
  </navMap>
</ncx>`)

	items, err := parseNCX(ncxData, "toc.ncx")
	if err != nil {
		t.Fatalf("parseNCX returned error: %v", err)
	}

	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}

	if items[0].Href != "chapter1.xhtml" {
		t.Errorf("item[0].Href = %q, want %q", items[0].Href, "chapter1.xhtml")
	}
}

func TestParseNCX_HTMLEntities(t *testing.T) {
	// NCX with HTML entities in the label text.
	ncxData := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<ncx xmlns="http://www.daisy.org/z3986/2005/ncx/" version="2005-1">
  <navMap>
    <navPoint id="np1" playOrder="1">
      <navLabel><text>Tom &amp; Jerry &mdash; Adventures</text></navLabel>
      <content src="chapter1.xhtml"/>
    </navPoint>
  </navMap>
</ncx>`)

	items, err := parseNCX(ncxData, "OEBPS/toc.ncx")
	if err != nil {
		t.Fatalf("parseNCX returned error: %v", err)
	}

	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}

	// &amp; is preserved as &, &mdash; is converted to —
	expected := "Tom & Jerry \u2014 Adventures"
	if items[0].Title != expected {
		t.Errorf("item[0].Title = %q, want %q", items[0].Title, expected)
	}
}

func TestParseNCX_SubdirectoryHref(t *testing.T) {
	// NCX in a subdirectory, content in a deeper subdirectory.
	ncxData := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<ncx xmlns="http://www.daisy.org/z3986/2005/ncx/" version="2005-1">
  <navMap>
    <navPoint id="np1" playOrder="1">
      <navLabel><text>Chapter 1</text></navLabel>
      <content src="text/chapter1.xhtml"/>
    </navPoint>
    <navPoint id="np2" playOrder="2">
      <navLabel><text>Chapter 2</text></navLabel>
      <content src="../other/chapter2.xhtml"/>
    </navPoint>
  </navMap>
</ncx>`)

	items, err := parseNCX(ncxData, "OEBPS/toc.ncx")
	if err != nil {
		t.Fatalf("parseNCX returned error: %v", err)
	}

	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}

	if items[0].Href != "OEBPS/text/chapter1.xhtml" {
		t.Errorf("item[0].Href = %q, want %q", items[0].Href, "OEBPS/text/chapter1.xhtml")
	}
	if items[1].Href != "other/chapter2.xhtml" {
		t.Errorf("item[1].Href = %q, want %q", items[1].Href, "other/chapter2.xhtml")
	}
}

func TestParseNCX_InvalidXML(t *testing.T) {
	ncxData := []byte(`<?xml version="1.0"?><ncx><broken`)

	_, err := parseNCX(ncxData, "OEBPS/toc.ncx")
	if err == nil {
		t.Fatal("expected error for invalid XML, got nil")
	}
}

func TestParseNCX_EmptySrc(t *testing.T) {
	// navPoint with empty content src.
	ncxData := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<ncx xmlns="http://www.daisy.org/z3986/2005/ncx/" version="2005-1">
  <navMap>
    <navPoint id="np1" playOrder="1">
      <navLabel><text>Empty Link</text></navLabel>
      <content src=""/>
    </navPoint>
  </navMap>
</ncx>`)

	items, err := parseNCX(ncxData, "OEBPS/toc.ncx")
	if err != nil {
		t.Fatalf("parseNCX returned error: %v", err)
	}

	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}

	if items[0].Href != "" {
		t.Errorf("item[0].Href = %q, want empty string", items[0].Href)
	}
	if items[0].Title != "Empty Link" {
		t.Errorf("item[0].Title = %q, want %q", items[0].Title, "Empty Link")
	}
}

// --- Nav Document (ePub 3) tests ---

func TestParseNavDocument_FlatTOC(t *testing.T) {
	navData := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<html xmlns="http://www.w3.org/1999/xhtml" xmlns:epub="http://www.idpf.org/2007/ops">
<body>
  <nav epub:type="toc">
    <h1>Table of Contents</h1>
    <ol>
      <li><a href="chapter1.xhtml">Chapter 1</a></li>
      <li><a href="chapter2.xhtml">Chapter 2</a></li>
      <li><a href="chapter3.xhtml">Chapter 3</a></li>
    </ol>
  </nav>
</body>
</html>`)

	toc, landmarks, err := parseNavDocument(navData, "OEBPS/nav.xhtml")
	if err != nil {
		t.Fatalf("parseNavDocument returned error: %v", err)
	}

	if len(landmarks) != 0 {
		t.Errorf("expected 0 landmarks, got %d", len(landmarks))
	}

	if len(toc) != 3 {
		t.Fatalf("expected 3 toc items, got %d", len(toc))
	}

	tests := []struct {
		title string
		href  string
	}{
		{"Chapter 1", "OEBPS/chapter1.xhtml"},
		{"Chapter 2", "OEBPS/chapter2.xhtml"},
		{"Chapter 3", "OEBPS/chapter3.xhtml"},
	}
	for i, tt := range tests {
		if toc[i].Title != tt.title {
			t.Errorf("toc[%d].Title = %q, want %q", i, toc[i].Title, tt.title)
		}
		if toc[i].Href != tt.href {
			t.Errorf("toc[%d].Href = %q, want %q", i, toc[i].Href, tt.href)
		}
		if toc[i].SpineIndex != -1 {
			t.Errorf("toc[%d].SpineIndex = %d, want -1", i, toc[i].SpineIndex)
		}
		if len(toc[i].Children) != 0 {
			t.Errorf("toc[%d].Children length = %d, want 0", i, len(toc[i].Children))
		}
	}
}

func TestParseNavDocument_NestedTOC(t *testing.T) {
	navData := []byte(`<html xmlns="http://www.w3.org/1999/xhtml" xmlns:epub="http://www.idpf.org/2007/ops">
<body>
  <nav epub:type="toc">
    <ol>
      <li>
        <a href="part1.xhtml">Part I</a>
        <ol>
          <li><a href="chapter1.xhtml">Chapter 1</a></li>
          <li>
            <a href="chapter2.xhtml">Chapter 2</a>
            <ol>
              <li><a href="chapter2.xhtml#sec1">Section 2.1</a></li>
            </ol>
          </li>
        </ol>
      </li>
      <li><a href="part2.xhtml">Part II</a></li>
    </ol>
  </nav>
</body>
</html>`)

	toc, _, err := parseNavDocument(navData, "OEBPS/nav.xhtml")
	if err != nil {
		t.Fatalf("parseNavDocument returned error: %v", err)
	}

	if len(toc) != 2 {
		t.Fatalf("expected 2 top-level items, got %d", len(toc))
	}

	// Part I
	part1 := toc[0]
	if part1.Title != "Part I" {
		t.Errorf("part1.Title = %q, want %q", part1.Title, "Part I")
	}
	if part1.Href != "OEBPS/part1.xhtml" {
		t.Errorf("part1.Href = %q, want %q", part1.Href, "OEBPS/part1.xhtml")
	}
	if len(part1.Children) != 2 {
		t.Fatalf("Part I children count = %d, want 2", len(part1.Children))
	}

	// Chapter 1
	ch1 := part1.Children[0]
	if ch1.Title != "Chapter 1" {
		t.Errorf("ch1.Title = %q, want %q", ch1.Title, "Chapter 1")
	}
	if ch1.Href != "OEBPS/chapter1.xhtml" {
		t.Errorf("ch1.Href = %q, want %q", ch1.Href, "OEBPS/chapter1.xhtml")
	}
	if len(ch1.Children) != 0 {
		t.Errorf("Chapter 1 children count = %d, want 0", len(ch1.Children))
	}

	// Chapter 2 with nested section
	ch2 := part1.Children[1]
	if ch2.Title != "Chapter 2" {
		t.Errorf("ch2.Title = %q, want %q", ch2.Title, "Chapter 2")
	}
	if ch2.Href != "OEBPS/chapter2.xhtml" {
		t.Errorf("ch2.Href = %q, want %q", ch2.Href, "OEBPS/chapter2.xhtml")
	}
	if len(ch2.Children) != 1 {
		t.Fatalf("Chapter 2 children count = %d, want 1", len(ch2.Children))
	}

	// Section 2.1 (fragment href)
	sec := ch2.Children[0]
	if sec.Title != "Section 2.1" {
		t.Errorf("sec.Title = %q, want %q", sec.Title, "Section 2.1")
	}
	if sec.Href != "OEBPS/chapter2.xhtml#sec1" {
		t.Errorf("sec.Href = %q, want %q", sec.Href, "OEBPS/chapter2.xhtml#sec1")
	}

	// Part II (no children)
	part2 := toc[1]
	if part2.Title != "Part II" {
		t.Errorf("part2.Title = %q, want %q", part2.Title, "Part II")
	}
	if len(part2.Children) != 0 {
		t.Errorf("Part II children count = %d, want 0", len(part2.Children))
	}
}

func TestParseNavDocument_WithLandmarks(t *testing.T) {
	navData := []byte(`<html xmlns="http://www.w3.org/1999/xhtml" xmlns:epub="http://www.idpf.org/2007/ops">
<body>
  <nav epub:type="toc">
    <ol>
      <li><a href="chapter1.xhtml">Chapter 1</a></li>
    </ol>
  </nav>
  <nav epub:type="landmarks">
    <ol>
      <li><a epub:type="toc" href="toc.xhtml">Table of Contents</a></li>
      <li><a epub:type="bodymatter" href="chapter1.xhtml">Begin Reading</a></li>
    </ol>
  </nav>
</body>
</html>`)

	toc, landmarks, err := parseNavDocument(navData, "OEBPS/nav.xhtml")
	if err != nil {
		t.Fatalf("parseNavDocument returned error: %v", err)
	}

	// TOC
	if len(toc) != 1 {
		t.Fatalf("expected 1 toc item, got %d", len(toc))
	}
	if toc[0].Title != "Chapter 1" {
		t.Errorf("toc[0].Title = %q, want %q", toc[0].Title, "Chapter 1")
	}
	if toc[0].Href != "OEBPS/chapter1.xhtml" {
		t.Errorf("toc[0].Href = %q, want %q", toc[0].Href, "OEBPS/chapter1.xhtml")
	}

	// Landmarks
	if len(landmarks) != 2 {
		t.Fatalf("expected 2 landmarks, got %d", len(landmarks))
	}
	if landmarks[0].Title != "Table of Contents" {
		t.Errorf("landmarks[0].Title = %q, want %q", landmarks[0].Title, "Table of Contents")
	}
	if landmarks[0].Href != "OEBPS/toc.xhtml" {
		t.Errorf("landmarks[0].Href = %q, want %q", landmarks[0].Href, "OEBPS/toc.xhtml")
	}
	if landmarks[1].Title != "Begin Reading" {
		t.Errorf("landmarks[1].Title = %q, want %q", landmarks[1].Title, "Begin Reading")
	}
	if landmarks[1].Href != "OEBPS/chapter1.xhtml" {
		t.Errorf("landmarks[1].Href = %q, want %q", landmarks[1].Href, "OEBPS/chapter1.xhtml")
	}
}

func TestParseNavDocument_SpanTitles(t *testing.T) {
	// Some ePubs use <span> instead of <a> for non-linked headings.
	navData := []byte(`<html xmlns="http://www.w3.org/1999/xhtml" xmlns:epub="http://www.idpf.org/2007/ops">
<body>
  <nav epub:type="toc">
    <ol>
      <li>
        <span>Part I: Introduction</span>
        <ol>
          <li><a href="chapter1.xhtml">Chapter 1</a></li>
        </ol>
      </li>
    </ol>
  </nav>
</body>
</html>`)

	toc, _, err := parseNavDocument(navData, "OEBPS/nav.xhtml")
	if err != nil {
		t.Fatalf("parseNavDocument returned error: %v", err)
	}

	if len(toc) != 1 {
		t.Fatalf("expected 1 toc item, got %d", len(toc))
	}

	// Span-based title, no href.
	if toc[0].Title != "Part I: Introduction" {
		t.Errorf("toc[0].Title = %q, want %q", toc[0].Title, "Part I: Introduction")
	}
	if toc[0].Href != "" {
		t.Errorf("toc[0].Href = %q, want empty", toc[0].Href)
	}

	// Children should still be parsed.
	if len(toc[0].Children) != 1 {
		t.Fatalf("toc[0].Children count = %d, want 1", len(toc[0].Children))
	}
	if toc[0].Children[0].Title != "Chapter 1" {
		t.Errorf("child.Title = %q, want %q", toc[0].Children[0].Title, "Chapter 1")
	}
	if toc[0].Children[0].Href != "OEBPS/chapter1.xhtml" {
		t.Errorf("child.Href = %q, want %q", toc[0].Children[0].Href, "OEBPS/chapter1.xhtml")
	}
}

func TestParseNavDocument_EmptyNav(t *testing.T) {
	navData := []byte(`<html xmlns="http://www.w3.org/1999/xhtml" xmlns:epub="http://www.idpf.org/2007/ops">
<body>
  <nav epub:type="toc">
    <h1>Table of Contents</h1>
  </nav>
</body>
</html>`)

	toc, landmarks, err := parseNavDocument(navData, "OEBPS/nav.xhtml")
	if err != nil {
		t.Fatalf("parseNavDocument returned error: %v", err)
	}

	if len(toc) != 0 {
		t.Errorf("expected 0 toc items, got %d", len(toc))
	}
	if len(landmarks) != 0 {
		t.Errorf("expected 0 landmarks, got %d", len(landmarks))
	}
}

func TestParseNavDocument_NoNavElement(t *testing.T) {
	// Document with no <nav> elements at all.
	navData := []byte(`<html xmlns="http://www.w3.org/1999/xhtml">
<body>
  <div>No nav here</div>
</body>
</html>`)

	toc, landmarks, err := parseNavDocument(navData, "OEBPS/nav.xhtml")
	if err != nil {
		t.Fatalf("parseNavDocument returned error: %v", err)
	}

	if len(toc) != 0 {
		t.Errorf("expected 0 toc items, got %d", len(toc))
	}
	if len(landmarks) != 0 {
		t.Errorf("expected 0 landmarks, got %d", len(landmarks))
	}
}

func TestParseNavDocument_RootLevel(t *testing.T) {
	// Nav document at the ZIP root.
	navData := []byte(`<html xmlns="http://www.w3.org/1999/xhtml" xmlns:epub="http://www.idpf.org/2007/ops">
<body>
  <nav epub:type="toc">
    <ol>
      <li><a href="chapter1.xhtml">Chapter 1</a></li>
    </ol>
  </nav>
</body>
</html>`)

	toc, _, err := parseNavDocument(navData, "nav.xhtml")
	if err != nil {
		t.Fatalf("parseNavDocument returned error: %v", err)
	}

	if len(toc) != 1 {
		t.Fatalf("expected 1 toc item, got %d", len(toc))
	}
	if toc[0].Href != "chapter1.xhtml" {
		t.Errorf("toc[0].Href = %q, want %q", toc[0].Href, "chapter1.xhtml")
	}
}

// --- Integration tests: Book.TOC() and spine association ---

// epub3OPFWithNav returns an ePub 3 OPF with both NCX and nav document references.
func epub3OPFWithNav(ncxID string) string {
	return `<?xml version="1.0" encoding="UTF-8"?>
<package xmlns="http://www.idpf.org/2007/opf" version="3.0">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
    <dc:title>Test Book</dc:title>
    <dc:language>en</dc:language>
  </metadata>
  <manifest>
    <item id="nav" href="nav.xhtml" media-type="application/xhtml+xml" properties="nav"/>
    <item id="ncx" href="toc.ncx" media-type="application/x-dtbncx+xml"/>
    <item id="ch1" href="chapter1.xhtml" media-type="application/xhtml+xml"/>
    <item id="ch2" href="chapter2.xhtml" media-type="application/xhtml+xml"/>
    <item id="ch3" href="chapter3.xhtml" media-type="application/xhtml+xml"/>
  </manifest>
  <spine toc="` + ncxID + `">
    <itemref idref="ch1"/>
    <itemref idref="ch2"/>
    <itemref idref="ch3"/>
  </spine>
</package>`
}

func epub2OPF() string {
	return `<?xml version="1.0" encoding="UTF-8"?>
<package xmlns="http://www.idpf.org/2007/opf" version="2.0">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
    <dc:title>Test Book</dc:title>
    <dc:language>en</dc:language>
  </metadata>
  <manifest>
    <item id="ncx" href="toc.ncx" media-type="application/x-dtbncx+xml"/>
    <item id="ch1" href="chapter1.xhtml" media-type="application/xhtml+xml"/>
    <item id="ch2" href="chapter2.xhtml" media-type="application/xhtml+xml"/>
    <item id="ch3" href="chapter3.xhtml" media-type="application/xhtml+xml"/>
  </manifest>
  <spine toc="ncx">
    <itemref idref="ch1"/>
    <itemref idref="ch2"/>
    <itemref idref="ch3"/>
  </spine>
</package>`
}

const testNavDoc = `<?xml version="1.0" encoding="UTF-8"?>
<html xmlns="http://www.w3.org/1999/xhtml" xmlns:epub="http://www.idpf.org/2007/ops">
<body>
  <nav epub:type="toc">
    <ol>
      <li><a href="chapter1.xhtml">Chapter 1 (Nav)</a></li>
      <li><a href="chapter2.xhtml">Chapter 2 (Nav)</a></li>
      <li><a href="chapter3.xhtml">Chapter 3 (Nav)</a></li>
    </ol>
  </nav>
  <nav epub:type="landmarks">
    <ol>
      <li><a epub:type="bodymatter" href="chapter1.xhtml">Begin Reading</a></li>
    </ol>
  </nav>
</body>
</html>`

const testNCX = `<?xml version="1.0" encoding="UTF-8"?>
<ncx xmlns="http://www.daisy.org/z3986/2005/ncx/" version="2005-1">
  <navMap>
    <navPoint id="np1" playOrder="1">
      <navLabel><text>Chapter 1 (NCX)</text></navLabel>
      <content src="chapter1.xhtml"/>
    </navPoint>
    <navPoint id="np2" playOrder="2">
      <navLabel><text>Chapter 2 (NCX)</text></navLabel>
      <content src="chapter2.xhtml"/>
    </navPoint>
    <navPoint id="np3" playOrder="3">
      <navLabel><text>Chapter 3 (NCX)</text></navLabel>
      <content src="chapter3.xhtml"/>
    </navPoint>
  </navMap>
</ncx>`

func TestBookTOC_EPUB3_PrefersNav(t *testing.T) {
	// ePub 3 with both nav and NCX: should prefer nav document.
	files := map[string]string{
		"mimetype":               "application/epub+zip",
		"META-INF/container.xml": validContainerXML,
		"OEBPS/content.opf":     epub3OPFWithNav("ncx"),
		"OEBPS/nav.xhtml":       testNavDoc,
		"OEBPS/toc.ncx":         testNCX,
		"OEBPS/chapter1.xhtml":  "<html><body>Ch1</body></html>",
		"OEBPS/chapter2.xhtml":  "<html><body>Ch2</body></html>",
		"OEBPS/chapter3.xhtml":  "<html><body>Ch3</body></html>",
	}

	fp := buildTestEPubFile(t, files)
	book, err := Open(fp)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer book.Close()

	toc := book.TOC()
	if len(toc) != 3 {
		t.Fatalf("expected 3 TOC items, got %d", len(toc))
	}

	// Should have "(Nav)" suffix since nav document is preferred.
	for i, item := range toc {
		if !strings.Contains(item.Title, "(Nav)") {
			t.Errorf("toc[%d].Title = %q, expected to contain '(Nav)'", i, item.Title)
		}
	}

	// Spine indices should be assigned.
	for i, item := range toc {
		if item.SpineIndex != i {
			t.Errorf("toc[%d].SpineIndex = %d, want %d", i, item.SpineIndex, i)
		}
	}

	// Landmarks should be parsed.
	landmarks := book.Landmarks()
	if len(landmarks) != 1 {
		t.Fatalf("expected 1 landmark, got %d", len(landmarks))
	}
	if landmarks[0].Title != "Begin Reading" {
		t.Errorf("landmarks[0].Title = %q, want %q", landmarks[0].Title, "Begin Reading")
	}
}

func TestBookTOC_EPUB3_FallbackToNCX(t *testing.T) {
	// ePub 3 with NCX but no nav document: should fall back to NCX.
	opf := `<?xml version="1.0" encoding="UTF-8"?>
<package xmlns="http://www.idpf.org/2007/opf" version="3.0">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
    <dc:title>Test Book</dc:title>
    <dc:language>en</dc:language>
  </metadata>
  <manifest>
    <item id="ncx" href="toc.ncx" media-type="application/x-dtbncx+xml"/>
    <item id="ch1" href="chapter1.xhtml" media-type="application/xhtml+xml"/>
    <item id="ch2" href="chapter2.xhtml" media-type="application/xhtml+xml"/>
  </manifest>
  <spine toc="ncx">
    <itemref idref="ch1"/>
    <itemref idref="ch2"/>
  </spine>
</package>`

	ncx := `<?xml version="1.0" encoding="UTF-8"?>
<ncx xmlns="http://www.daisy.org/z3986/2005/ncx/" version="2005-1">
  <navMap>
    <navPoint id="np1" playOrder="1">
      <navLabel><text>Chapter 1 (NCX)</text></navLabel>
      <content src="chapter1.xhtml"/>
    </navPoint>
    <navPoint id="np2" playOrder="2">
      <navLabel><text>Chapter 2 (NCX)</text></navLabel>
      <content src="chapter2.xhtml"/>
    </navPoint>
  </navMap>
</ncx>`

	files := map[string]string{
		"mimetype":               "application/epub+zip",
		"META-INF/container.xml": validContainerXML,
		"OEBPS/content.opf":     opf,
		"OEBPS/toc.ncx":         ncx,
		"OEBPS/chapter1.xhtml":  "<html><body>Ch1</body></html>",
		"OEBPS/chapter2.xhtml":  "<html><body>Ch2</body></html>",
	}

	fp := buildTestEPubFile(t, files)
	book, err := Open(fp)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer book.Close()

	toc := book.TOC()
	if len(toc) != 2 {
		t.Fatalf("expected 2 TOC items, got %d", len(toc))
	}

	for i, item := range toc {
		if !strings.Contains(item.Title, "(NCX)") {
			t.Errorf("toc[%d].Title = %q, expected to contain '(NCX)'", i, item.Title)
		}
		if item.SpineIndex != i {
			t.Errorf("toc[%d].SpineIndex = %d, want %d", i, item.SpineIndex, i)
		}
	}
}

func TestBookTOC_EPUB2_UsesNCX(t *testing.T) {
	files := map[string]string{
		"mimetype":               "application/epub+zip",
		"META-INF/container.xml": validContainerXML,
		"OEBPS/content.opf":     epub2OPF(),
		"OEBPS/toc.ncx":         testNCX,
		"OEBPS/chapter1.xhtml":  "<html><body>Ch1</body></html>",
		"OEBPS/chapter2.xhtml":  "<html><body>Ch2</body></html>",
		"OEBPS/chapter3.xhtml":  "<html><body>Ch3</body></html>",
	}

	fp := buildTestEPubFile(t, files)
	book, err := Open(fp)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer book.Close()

	toc := book.TOC()
	if len(toc) != 3 {
		t.Fatalf("expected 3 TOC items, got %d", len(toc))
	}

	for i, item := range toc {
		if !strings.Contains(item.Title, "(NCX)") {
			t.Errorf("toc[%d].Title = %q, expected to contain '(NCX)'", i, item.Title)
		}
		if item.SpineIndex != i {
			t.Errorf("toc[%d].SpineIndex = %d, want %d", i, item.SpineIndex, i)
		}
	}

	// ePub 2 should have no landmarks.
	if len(book.Landmarks()) != 0 {
		t.Errorf("expected 0 landmarks for ePub 2, got %d", len(book.Landmarks()))
	}
}

func TestBookTOC_SpineAssociation_Fragment(t *testing.T) {
	// TOC entries with fragments should still match spine items.
	opf := `<?xml version="1.0" encoding="UTF-8"?>
<package xmlns="http://www.idpf.org/2007/opf" version="2.0">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
    <dc:title>Test</dc:title>
    <dc:language>en</dc:language>
  </metadata>
  <manifest>
    <item id="ncx" href="toc.ncx" media-type="application/x-dtbncx+xml"/>
    <item id="ch1" href="chapter1.xhtml" media-type="application/xhtml+xml"/>
  </manifest>
  <spine toc="ncx">
    <itemref idref="ch1"/>
  </spine>
</package>`

	ncx := `<?xml version="1.0" encoding="UTF-8"?>
<ncx xmlns="http://www.daisy.org/z3986/2005/ncx/" version="2005-1">
  <navMap>
    <navPoint id="np1" playOrder="1">
      <navLabel><text>Introduction</text></navLabel>
      <content src="chapter1.xhtml#intro"/>
    </navPoint>
    <navPoint id="np2" playOrder="2">
      <navLabel><text>Section 1</text></navLabel>
      <content src="chapter1.xhtml#sec1"/>
    </navPoint>
  </navMap>
</ncx>`

	files := map[string]string{
		"mimetype":               "application/epub+zip",
		"META-INF/container.xml": validContainerXML,
		"OEBPS/content.opf":     opf,
		"OEBPS/toc.ncx":         ncx,
		"OEBPS/chapter1.xhtml":  "<html><body>Content</body></html>",
	}

	fp := buildTestEPubFile(t, files)
	book, err := Open(fp)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer book.Close()

	toc := book.TOC()
	if len(toc) != 2 {
		t.Fatalf("expected 2 TOC items, got %d", len(toc))
	}

	// Both entries point to chapter1.xhtml with different fragments; spine index should be 0.
	for i, item := range toc {
		if item.SpineIndex != 0 {
			t.Errorf("toc[%d].SpineIndex = %d, want 0", i, item.SpineIndex)
		}
	}

	// Hrefs should preserve fragments.
	if toc[0].Href != "OEBPS/chapter1.xhtml#intro" {
		t.Errorf("toc[0].Href = %q, want %q", toc[0].Href, "OEBPS/chapter1.xhtml#intro")
	}
	if toc[1].Href != "OEBPS/chapter1.xhtml#sec1" {
		t.Errorf("toc[1].Href = %q, want %q", toc[1].Href, "OEBPS/chapter1.xhtml#sec1")
	}
}

func TestBookTOC_SpineAssociation_Nested(t *testing.T) {
	// Nested TOC entries should also get spine indices.
	navDoc := `<html xmlns="http://www.w3.org/1999/xhtml" xmlns:epub="http://www.idpf.org/2007/ops">
<body>
  <nav epub:type="toc">
    <ol>
      <li>
        <a href="chapter1.xhtml">Part I</a>
        <ol>
          <li><a href="chapter1.xhtml#sec1">Section 1</a></li>
          <li><a href="chapter2.xhtml">Section 2</a></li>
        </ol>
      </li>
      <li><a href="chapter3.xhtml">Part II</a></li>
    </ol>
  </nav>
</body>
</html>`

	opf := `<?xml version="1.0" encoding="UTF-8"?>
<package xmlns="http://www.idpf.org/2007/opf" version="3.0">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
    <dc:title>Test</dc:title>
    <dc:language>en</dc:language>
  </metadata>
  <manifest>
    <item id="nav" href="nav.xhtml" media-type="application/xhtml+xml" properties="nav"/>
    <item id="ch1" href="chapter1.xhtml" media-type="application/xhtml+xml"/>
    <item id="ch2" href="chapter2.xhtml" media-type="application/xhtml+xml"/>
    <item id="ch3" href="chapter3.xhtml" media-type="application/xhtml+xml"/>
  </manifest>
  <spine>
    <itemref idref="ch1"/>
    <itemref idref="ch2"/>
    <itemref idref="ch3"/>
  </spine>
</package>`

	files := map[string]string{
		"mimetype":               "application/epub+zip",
		"META-INF/container.xml": validContainerXML,
		"OEBPS/content.opf":     opf,
		"OEBPS/nav.xhtml":       navDoc,
		"OEBPS/chapter1.xhtml":  "<html><body>Ch1</body></html>",
		"OEBPS/chapter2.xhtml":  "<html><body>Ch2</body></html>",
		"OEBPS/chapter3.xhtml":  "<html><body>Ch3</body></html>",
	}

	fp := buildTestEPubFile(t, files)
	book, err := Open(fp)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer book.Close()

	toc := book.TOC()
	if len(toc) != 2 {
		t.Fatalf("expected 2 top-level TOC items, got %d", len(toc))
	}

	// Part I → spine index 0
	if toc[0].SpineIndex != 0 {
		t.Errorf("toc[0].SpineIndex = %d, want 0", toc[0].SpineIndex)
	}
	if len(toc[0].Children) != 2 {
		t.Fatalf("toc[0].Children count = %d, want 2", len(toc[0].Children))
	}

	// Section 1 → chapter1.xhtml#sec1 → spine index 0
	if toc[0].Children[0].SpineIndex != 0 {
		t.Errorf("toc[0].Children[0].SpineIndex = %d, want 0", toc[0].Children[0].SpineIndex)
	}
	// Section 2 → chapter2.xhtml → spine index 1
	if toc[0].Children[1].SpineIndex != 1 {
		t.Errorf("toc[0].Children[1].SpineIndex = %d, want 1", toc[0].Children[1].SpineIndex)
	}

	// Part II → spine index 2
	if toc[1].SpineIndex != 2 {
		t.Errorf("toc[1].SpineIndex = %d, want 2", toc[1].SpineIndex)
	}
}

func TestBookTOC_NoTOC(t *testing.T) {
	// ePub with no NCX and no nav document: should get empty TOC, no error.
	opf := `<?xml version="1.0" encoding="UTF-8"?>
<package xmlns="http://www.idpf.org/2007/opf" version="2.0">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
    <dc:title>Test</dc:title>
    <dc:language>en</dc:language>
  </metadata>
  <manifest>
    <item id="ch1" href="chapter1.xhtml" media-type="application/xhtml+xml"/>
  </manifest>
  <spine>
    <itemref idref="ch1"/>
  </spine>
</package>`

	files := map[string]string{
		"mimetype":               "application/epub+zip",
		"META-INF/container.xml": validContainerXML,
		"OEBPS/content.opf":     opf,
		"OEBPS/chapter1.xhtml":  "<html><body>Ch1</body></html>",
	}

	fp := buildTestEPubFile(t, files)
	book, err := Open(fp)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer book.Close()

	toc := book.TOC()
	if len(toc) != 0 {
		t.Errorf("expected empty TOC, got %d items", len(toc))
	}
	if toc == nil {
		t.Error("TOC should be empty slice, not nil")
	}
}

func TestBookTOC_UnmatchedHref(t *testing.T) {
	// TOC entry referencing a file not in the spine should get SpineIndex = -1.
	ncx := `<?xml version="1.0" encoding="UTF-8"?>
<ncx xmlns="http://www.daisy.org/z3986/2005/ncx/" version="2005-1">
  <navMap>
    <navPoint id="np1" playOrder="1">
      <navLabel><text>Chapter 1</text></navLabel>
      <content src="chapter1.xhtml"/>
    </navPoint>
    <navPoint id="np2" playOrder="2">
      <navLabel><text>Appendix</text></navLabel>
      <content src="appendix.xhtml"/>
    </navPoint>
  </navMap>
</ncx>`

	opf := `<?xml version="1.0" encoding="UTF-8"?>
<package xmlns="http://www.idpf.org/2007/opf" version="2.0">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
    <dc:title>Test</dc:title>
    <dc:language>en</dc:language>
  </metadata>
  <manifest>
    <item id="ncx" href="toc.ncx" media-type="application/x-dtbncx+xml"/>
    <item id="ch1" href="chapter1.xhtml" media-type="application/xhtml+xml"/>
  </manifest>
  <spine toc="ncx">
    <itemref idref="ch1"/>
  </spine>
</package>`

	files := map[string]string{
		"mimetype":               "application/epub+zip",
		"META-INF/container.xml": validContainerXML,
		"OEBPS/content.opf":     opf,
		"OEBPS/toc.ncx":         ncx,
		"OEBPS/chapter1.xhtml":  "<html><body>Ch1</body></html>",
	}

	fp := buildTestEPubFile(t, files)
	book, err := Open(fp)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer book.Close()

	toc := book.TOC()
	if len(toc) != 2 {
		t.Fatalf("expected 2 TOC items, got %d", len(toc))
	}

	// chapter1.xhtml is in spine → index 0.
	if toc[0].SpineIndex != 0 {
		t.Errorf("toc[0].SpineIndex = %d, want 0", toc[0].SpineIndex)
	}

	// appendix.xhtml is NOT in spine → index -1.
	if toc[1].SpineIndex != -1 {
		t.Errorf("toc[1].SpineIndex = %d, want -1", toc[1].SpineIndex)
	}
}

func TestHrefWithoutFragment(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"chapter1.xhtml", "chapter1.xhtml"},
		{"chapter1.xhtml#sec1", "chapter1.xhtml"},
		{"OEBPS/chapter1.xhtml#intro", "OEBPS/chapter1.xhtml"},
		{"", ""},
		{"#fragment-only", ""},
	}
	for _, tt := range tests {
		got := hrefWithoutFragment(tt.input)
		if got != tt.want {
			t.Errorf("hrefWithoutFragment(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestBookTOC_SpineRanges(t *testing.T) {
	// FR-408: 5 spine items, 2 TOC entries (at spine 0 and spine 3).
	// TOC entry 0 should cover spine[0:3], TOC entry 1 should cover spine[3:5].
	ncx := `<?xml version="1.0" encoding="UTF-8"?>
<ncx xmlns="http://www.daisy.org/z3986/2005/ncx/" version="2005-1">
  <navMap>
    <navPoint id="np1" playOrder="1">
      <navLabel><text>Chapter 1</text></navLabel>
      <content src="ch1.xhtml"/>
    </navPoint>
    <navPoint id="np2" playOrder="2">
      <navLabel><text>Chapter 2</text></navLabel>
      <content src="ch4.xhtml"/>
    </navPoint>
  </navMap>
</ncx>`

	opf := `<?xml version="1.0" encoding="UTF-8"?>
<package xmlns="http://www.idpf.org/2007/opf" version="2.0">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
    <dc:title>Spine Range Test</dc:title>
    <dc:language>en</dc:language>
  </metadata>
  <manifest>
    <item id="ncx" href="toc.ncx" media-type="application/x-dtbncx+xml"/>
    <item id="c1" href="ch1.xhtml" media-type="application/xhtml+xml"/>
    <item id="c2" href="ch2.xhtml" media-type="application/xhtml+xml"/>
    <item id="c3" href="ch3.xhtml" media-type="application/xhtml+xml"/>
    <item id="c4" href="ch4.xhtml" media-type="application/xhtml+xml"/>
    <item id="c5" href="ch5.xhtml" media-type="application/xhtml+xml"/>
  </manifest>
  <spine toc="ncx">
    <itemref idref="c1"/>
    <itemref idref="c2"/>
    <itemref idref="c3"/>
    <itemref idref="c4"/>
    <itemref idref="c5"/>
  </spine>
</package>`

	files := map[string]string{
		"mimetype":               "application/epub+zip",
		"META-INF/container.xml": validContainerXML,
		"OEBPS/content.opf":     opf,
		"OEBPS/toc.ncx":         ncx,
		"OEBPS/ch1.xhtml":       "<html><body>Ch1</body></html>",
		"OEBPS/ch2.xhtml":       "<html><body>Ch2</body></html>",
		"OEBPS/ch3.xhtml":       "<html><body>Ch3</body></html>",
		"OEBPS/ch4.xhtml":       "<html><body>Ch4</body></html>",
		"OEBPS/ch5.xhtml":       "<html><body>Ch5</body></html>",
	}

	fp := buildTestEPubFile(t, files)
	book, err := Open(fp)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer book.Close()

	toc := book.TOC()
	if len(toc) != 2 {
		t.Fatalf("expected 2 TOC items, got %d", len(toc))
	}

	// Chapter 1: SpineIndex=0, covers spine[0:3]
	if toc[0].SpineIndex != 0 {
		t.Errorf("toc[0].SpineIndex = %d, want 0", toc[0].SpineIndex)
	}
	if toc[0].SpineEndIndex != 3 {
		t.Errorf("toc[0].SpineEndIndex = %d, want 3", toc[0].SpineEndIndex)
	}

	// Chapter 2: SpineIndex=3, covers spine[3:5]
	if toc[1].SpineIndex != 3 {
		t.Errorf("toc[1].SpineIndex = %d, want 3", toc[1].SpineIndex)
	}
	if toc[1].SpineEndIndex != 5 {
		t.Errorf("toc[1].SpineEndIndex = %d, want 5", toc[1].SpineEndIndex)
	}
}

func TestBookTOC_SpineRanges_UnmatchedEntry(t *testing.T) {
	// An unmatched TOC entry (SpineIndex=-1) should have SpineEndIndex=-1
	// and should not affect the ranges of other entries.
	ncx := `<?xml version="1.0" encoding="UTF-8"?>
<ncx xmlns="http://www.daisy.org/z3986/2005/ncx/" version="2005-1">
  <navMap>
    <navPoint id="np1" playOrder="1">
      <navLabel><text>Chapter 1</text></navLabel>
      <content src="ch1.xhtml"/>
    </navPoint>
    <navPoint id="np2" playOrder="2">
      <navLabel><text>Missing</text></navLabel>
      <content src="missing.xhtml"/>
    </navPoint>
  </navMap>
</ncx>`

	opf := `<?xml version="1.0" encoding="UTF-8"?>
<package xmlns="http://www.idpf.org/2007/opf" version="2.0">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
    <dc:title>Test</dc:title>
    <dc:language>en</dc:language>
  </metadata>
  <manifest>
    <item id="ncx" href="toc.ncx" media-type="application/x-dtbncx+xml"/>
    <item id="c1" href="ch1.xhtml" media-type="application/xhtml+xml"/>
    <item id="c2" href="ch2.xhtml" media-type="application/xhtml+xml"/>
  </manifest>
  <spine toc="ncx">
    <itemref idref="c1"/>
    <itemref idref="c2"/>
  </spine>
</package>`

	files := map[string]string{
		"mimetype":               "application/epub+zip",
		"META-INF/container.xml": validContainerXML,
		"OEBPS/content.opf":     opf,
		"OEBPS/toc.ncx":         ncx,
		"OEBPS/ch1.xhtml":       "<html><body>Ch1</body></html>",
		"OEBPS/ch2.xhtml":       "<html><body>Ch2</body></html>",
	}

	fp := buildTestEPubFile(t, files)
	book, err := Open(fp)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer book.Close()

	toc := book.TOC()
	if len(toc) != 2 {
		t.Fatalf("expected 2 TOC items, got %d", len(toc))
	}

	// Chapter 1: only matched entry, covers entire spine [0:2]
	if toc[0].SpineIndex != 0 {
		t.Errorf("toc[0].SpineIndex = %d, want 0", toc[0].SpineIndex)
	}
	if toc[0].SpineEndIndex != 2 {
		t.Errorf("toc[0].SpineEndIndex = %d, want 2", toc[0].SpineEndIndex)
	}

	// Missing: unmatched, SpineIndex=-1, SpineEndIndex=-1
	if toc[1].SpineIndex != -1 {
		t.Errorf("toc[1].SpineIndex = %d, want -1", toc[1].SpineIndex)
	}
	if toc[1].SpineEndIndex != -1 {
		t.Errorf("toc[1].SpineEndIndex = %d, want -1", toc[1].SpineEndIndex)
	}
}

func TestHasTOC_WithTOC(t *testing.T) {
	// ePub with a valid NCX TOC: HasTOC should return true.
	files := map[string]string{
		"mimetype":               "application/epub+zip",
		"META-INF/container.xml": validContainerXML,
		"OEBPS/content.opf":     epub2OPF(),
		"OEBPS/toc.ncx":         testNCX,
		"OEBPS/chapter1.xhtml":  "<html><body>Ch1</body></html>",
		"OEBPS/chapter2.xhtml":  "<html><body>Ch2</body></html>",
		"OEBPS/chapter3.xhtml":  "<html><body>Ch3</body></html>",
	}

	fp := buildTestEPubFile(t, files)
	book, err := Open(fp)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer book.Close()

	if !book.HasTOC() {
		t.Error("HasTOC() = false, want true for ePub with TOC")
	}
}

func TestHasTOC_WithoutTOC(t *testing.T) {
	// ePub with no NCX and no nav document: HasTOC should return false.
	opf := `<?xml version="1.0" encoding="UTF-8"?>
<package xmlns="http://www.idpf.org/2007/opf" version="2.0">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
    <dc:title>Test</dc:title>
    <dc:language>en</dc:language>
  </metadata>
  <manifest>
    <item id="ch1" href="chapter1.xhtml" media-type="application/xhtml+xml"/>
  </manifest>
  <spine>
    <itemref idref="ch1"/>
  </spine>
</package>`

	files := map[string]string{
		"mimetype":               "application/epub+zip",
		"META-INF/container.xml": validContainerXML,
		"OEBPS/content.opf":     opf,
		"OEBPS/chapter1.xhtml":  "<html><body>Ch1</body></html>",
	}

	fp := buildTestEPubFile(t, files)
	book, err := Open(fp)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer book.Close()

	if book.HasTOC() {
		t.Error("HasTOC() = true, want false for ePub without TOC")
	}
}
