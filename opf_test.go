package epub

import (
	"bytes"
	"testing"
)

// --- OPF test data ---

const testOPFv2 = `<?xml version="1.0" encoding="UTF-8"?>
<package version="2.0" xmlns="http://www.idpf.org/2007/opf" unique-identifier="bookid">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
    <dc:title>Test Book v2</dc:title>
  </metadata>
  <manifest>
    <item id="ncx" href="toc.ncx" media-type="application/x-dtbncx+xml"/>
    <item id="chap1" href="chapter1.xhtml" media-type="application/xhtml+xml"/>
    <item id="chap2" href="chapter2.xhtml" media-type="application/xhtml+xml"/>
    <item id="css" href="style.css" media-type="text/css"/>
    <item id="cover-img" href="cover.jpg" media-type="image/jpeg"/>
  </manifest>
  <spine toc="ncx">
    <itemref idref="chap1"/>
    <itemref idref="chap2" linear="yes"/>
  </spine>
  <guide>
    <reference type="cover" title="Cover" href="cover.xhtml"/>
    <reference type="toc" title="Table of Contents" href="toc.xhtml"/>
  </guide>
</package>`

const testOPFv3 = `<?xml version="1.0" encoding="UTF-8"?>
<package version="3.0" xmlns="http://www.idpf.org/2007/opf" unique-identifier="uid">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
    <dc:title>Test Book v3</dc:title>
  </metadata>
  <manifest>
    <item id="chap1" href="chapter1.xhtml" media-type="application/xhtml+xml"/>
    <item id="chap2" href="chapter2.xhtml" media-type="application/xhtml+xml"/>
    <item id="nav" href="nav.xhtml" media-type="application/xhtml+xml" properties="nav"/>
    <item id="cover-img" href="cover.jpg" media-type="image/jpeg" properties="cover-image"/>
    <item id="css" href="style.css" media-type="text/css"/>
  </manifest>
  <spine>
    <itemref idref="chap1" linear="yes"/>
    <itemref idref="chap2" linear="no"/>
  </spine>
</package>`

const testOPFNoVersion = `<?xml version="1.0" encoding="UTF-8"?>
<package xmlns="http://www.idpf.org/2007/opf">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
    <dc:title>No Version</dc:title>
  </metadata>
  <manifest>
    <item id="chap1" href="chapter1.xhtml" media-type="application/xhtml+xml"/>
  </manifest>
  <spine>
    <itemref idref="chap1"/>
  </spine>
</package>`

const testOPFWithEntities = `<?xml version="1.0" encoding="UTF-8"?>
<package version="2.0" xmlns="http://www.idpf.org/2007/opf">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
    <dc:title>Caf&eacute; &amp; Cr&egrave;me</dc:title>
  </metadata>
  <manifest>
    <item id="chap1" href="chapter1.xhtml" media-type="application/xhtml+xml"/>
  </manifest>
  <spine>
    <itemref idref="chap1"/>
  </spine>
</package>`

// --- parseOPF tests ---

func TestParseOPF_V2(t *testing.T) {
	pkg, err := parseOPF([]byte(testOPFv2))
	if err != nil {
		t.Fatalf("parseOPF() error = %v", err)
	}

	if pkg.Version != "2.0" {
		t.Errorf("Version = %q, want %q", pkg.Version, "2.0")
	}
	if pkg.UniqueIdentifier != "bookid" {
		t.Errorf("UniqueIdentifier = %q, want %q", pkg.UniqueIdentifier, "bookid")
	}

	// Manifest.
	if got := len(pkg.Manifest.Items); got != 5 {
		t.Fatalf("Manifest items = %d, want 5", got)
	}

	// Spine.
	if pkg.Spine.Toc != "ncx" {
		t.Errorf("Spine.Toc = %q, want %q", pkg.Spine.Toc, "ncx")
	}
	if got := len(pkg.Spine.ItemRefs); got != 2 {
		t.Fatalf("Spine itemrefs = %d, want 2", got)
	}
	if pkg.Spine.ItemRefs[0].IDRef != "chap1" {
		t.Errorf("Spine[0].IDRef = %q, want %q", pkg.Spine.ItemRefs[0].IDRef, "chap1")
	}

	// Guide.
	if got := len(pkg.Guide.References); got != 2 {
		t.Fatalf("Guide references = %d, want 2", got)
	}
	if pkg.Guide.References[0].Type != "cover" {
		t.Errorf("Guide[0].Type = %q, want %q", pkg.Guide.References[0].Type, "cover")
	}
}

func TestParseOPF_V3(t *testing.T) {
	pkg, err := parseOPF([]byte(testOPFv3))
	if err != nil {
		t.Fatalf("parseOPF() error = %v", err)
	}

	if pkg.Version != "3.0" {
		t.Errorf("Version = %q, want %q", pkg.Version, "3.0")
	}

	// Check manifest item with properties.
	var navItem *opfManifestItem
	for i := range pkg.Manifest.Items {
		if pkg.Manifest.Items[i].ID == "nav" {
			navItem = &pkg.Manifest.Items[i]
			break
		}
	}
	if navItem == nil {
		t.Fatal("nav item not found in manifest")
	}
	if navItem.Properties != "nav" {
		t.Errorf("nav item Properties = %q, want %q", navItem.Properties, "nav")
	}

	// V3 has no guide.
	if got := len(pkg.Guide.References); got != 0 {
		t.Errorf("Guide references = %d, want 0 for ePub 3", got)
	}

	// Spine has no toc attribute in v3.
	if pkg.Spine.Toc != "" {
		t.Errorf("Spine.Toc = %q, want empty for ePub 3", pkg.Spine.Toc)
	}
}

func TestParseOPF_VersionDefault(t *testing.T) {
	pkg, err := parseOPF([]byte(testOPFNoVersion))
	if err != nil {
		t.Fatalf("parseOPF() error = %v", err)
	}

	if pkg.Version != "2.0" {
		t.Errorf("Version = %q, want %q (default)", pkg.Version, "2.0")
	}
}

func TestParseOPF_HTMLEntities(t *testing.T) {
	pkg, err := parseOPF([]byte(testOPFWithEntities))
	if err != nil {
		t.Fatalf("parseOPF() error = %v", err)
	}

	if len(pkg.Metadata.Titles) == 0 {
		t.Fatal("expected at least one title")
	}
	want := "Caf\u00e9 & Cr\u00e8me"
	if got := pkg.Metadata.Titles[0].Value; got != want {
		t.Errorf("Title = %q, want %q", got, want)
	}
}

func TestParseOPF_BOM(t *testing.T) {
	bomOPF := "\xEF\xBB\xBF" + testOPFv2
	pkg, err := parseOPF([]byte(bomOPF))
	if err != nil {
		t.Fatalf("parseOPF() with BOM error = %v", err)
	}
	if pkg.Version != "2.0" {
		t.Errorf("Version = %q, want %q", pkg.Version, "2.0")
	}
}

func TestParseOPF_InvalidXML(t *testing.T) {
	_, err := parseOPF([]byte("<package><broken"))
	if err == nil {
		t.Fatal("parseOPF() with invalid XML should return error")
	}
}

func TestParseOPF_MinimalPackage(t *testing.T) {
	pkg, err := parseOPF([]byte(`<?xml version="1.0"?><package/>`))
	if err != nil {
		t.Fatalf("parseOPF() error = %v", err)
	}
	if pkg.Version != "2.0" {
		t.Errorf("Version = %q, want %q (default)", pkg.Version, "2.0")
	}
	if len(pkg.Manifest.Items) != 0 {
		t.Errorf("Manifest items = %d, want 0", len(pkg.Manifest.Items))
	}
}

// --- buildManifestMaps tests ---

func TestBuildManifestMaps(t *testing.T) {
	pkg, err := parseOPF([]byte(testOPFv2))
	if err != nil {
		t.Fatalf("parseOPF() error = %v", err)
	}

	byID, byHref := buildManifestMaps(pkg.Manifest)

	// Check by ID.
	mi, ok := byID["chap1"]
	if !ok {
		t.Fatal("manifest item 'chap1' not found by ID")
	}
	if mi.Href != "chapter1.xhtml" {
		t.Errorf("byID[chap1].Href = %q, want %q", mi.Href, "chapter1.xhtml")
	}
	if mi.MediaType != "application/xhtml+xml" {
		t.Errorf("byID[chap1].MediaType = %q, want %q", mi.MediaType, "application/xhtml+xml")
	}

	// Check by Href.
	mi2, ok := byHref["style.css"]
	if !ok {
		t.Fatal("manifest item 'style.css' not found by Href")
	}
	if mi2.ID != "css" {
		t.Errorf("byHref[style.css].ID = %q, want %q", mi2.ID, "css")
	}
}

// --- buildSpine tests ---

func TestBuildSpine(t *testing.T) {
	pkg, err := parseOPF([]byte(testOPFv2))
	if err != nil {
		t.Fatalf("parseOPF() error = %v", err)
	}

	byID, _ := buildManifestMaps(pkg.Manifest)
	spine := buildSpine(pkg.Spine, byID)

	if len(spine) != 2 {
		t.Fatalf("spine length = %d, want 2", len(spine))
	}

	// First spine item.
	if spine[0].IDRef != "chap1" {
		t.Errorf("spine[0].IDRef = %q, want %q", spine[0].IDRef, "chap1")
	}
	if spine[0].Href != "chapter1.xhtml" {
		t.Errorf("spine[0].Href = %q, want %q", spine[0].Href, "chapter1.xhtml")
	}
	if spine[0].MediaType != "application/xhtml+xml" {
		t.Errorf("spine[0].MediaType = %q, want %q", spine[0].MediaType, "application/xhtml+xml")
	}
	if !spine[0].Linear {
		t.Error("spine[0].Linear = false, want true (default)")
	}

	// Second spine item (explicit linear="yes").
	if !spine[1].Linear {
		t.Error("spine[1].Linear = false, want true")
	}
}

func TestBuildSpine_NonLinear(t *testing.T) {
	pkg, err := parseOPF([]byte(testOPFv3))
	if err != nil {
		t.Fatalf("parseOPF() error = %v", err)
	}

	byID, _ := buildManifestMaps(pkg.Manifest)
	spine := buildSpine(pkg.Spine, byID)

	if len(spine) != 2 {
		t.Fatalf("spine length = %d, want 2", len(spine))
	}

	if !spine[0].Linear {
		t.Error("spine[0].Linear = false, want true")
	}
	if spine[1].Linear {
		t.Error("spine[1].Linear = true, want false")
	}
}

func TestBuildSpine_MissingManifestItem(t *testing.T) {
	opf := `<?xml version="1.0"?>
<package version="2.0" xmlns="http://www.idpf.org/2007/opf">
  <manifest>
    <item id="chap1" href="chapter1.xhtml" media-type="application/xhtml+xml"/>
  </manifest>
  <spine>
    <itemref idref="chap1"/>
    <itemref idref="missing"/>
  </spine>
</package>`
	pkg, err := parseOPF([]byte(opf))
	if err != nil {
		t.Fatalf("parseOPF() error = %v", err)
	}

	byID, _ := buildManifestMaps(pkg.Manifest)
	spine := buildSpine(pkg.Spine, byID)

	if len(spine) != 2 {
		t.Fatalf("spine length = %d, want 2", len(spine))
	}

	// The missing item should still appear with IDRef set but empty Href/MediaType.
	if spine[1].IDRef != "missing" {
		t.Errorf("spine[1].IDRef = %q, want %q", spine[1].IDRef, "missing")
	}
	if spine[1].Href != "" {
		t.Errorf("spine[1].Href = %q, want empty", spine[1].Href)
	}
}

// --- buildGuide tests ---

func TestBuildGuide(t *testing.T) {
	pkg, err := parseOPF([]byte(testOPFv2))
	if err != nil {
		t.Fatalf("parseOPF() error = %v", err)
	}

	guide := buildGuide(pkg.Guide)
	if len(guide) != 2 {
		t.Fatalf("guide length = %d, want 2", len(guide))
	}

	if guide[0].Type != "cover" {
		t.Errorf("guide[0].Type = %q, want %q", guide[0].Type, "cover")
	}
	if guide[0].Title != "Cover" {
		t.Errorf("guide[0].Title = %q, want %q", guide[0].Title, "Cover")
	}
	if guide[0].Href != "cover.xhtml" {
		t.Errorf("guide[0].Href = %q, want %q", guide[0].Href, "cover.xhtml")
	}

	if guide[1].Type != "toc" {
		t.Errorf("guide[1].Type = %q, want %q", guide[1].Type, "toc")
	}
}

// --- Integration tests ---

func TestOpen_ParsesOPF(t *testing.T) {
	files := map[string]string{
		"mimetype":               "application/epub+zip",
		"META-INF/container.xml": validContainerXML,
		"OEBPS/content.opf":     testOPFv2,
	}
	fp := buildTestEPubFile(t, files)

	book, err := Open(fp)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer book.Close()

	if book.opf == nil {
		t.Fatal("book.opf is nil after Open")
	}
	if book.opf.Version != "2.0" {
		t.Errorf("Version = %q, want %q", book.opf.Version, "2.0")
	}
	if len(book.manifestByID) != 5 {
		t.Errorf("manifestByID has %d entries, want 5", len(book.manifestByID))
	}
	if len(book.manifestByHref) != 5 {
		t.Errorf("manifestByHref has %d entries, want 5", len(book.manifestByHref))
	}
	if len(book.spine) != 2 {
		t.Errorf("spine has %d entries, want 2", len(book.spine))
	}
	if len(book.guide) != 2 {
		t.Errorf("guide has %d entries, want 2", len(book.guide))
	}
}

func TestNewReader_ParsesOPF_V3(t *testing.T) {
	files := map[string]string{
		"mimetype":               "application/epub+zip",
		"META-INF/container.xml": validContainerXML,
		"OEBPS/content.opf":     testOPFv3,
	}
	data := buildTestEPubBytes(t, files)

	book, err := NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("NewReader() error = %v", err)
	}
	defer book.Close()

	if book.opf == nil {
		t.Fatal("book.opf is nil after NewReader")
	}
	if book.opf.Version != "3.0" {
		t.Errorf("Version = %q, want %q", book.opf.Version, "3.0")
	}

	// Check that cover-image properties item is in manifest.
	mi, ok := book.manifestByID["cover-img"]
	if !ok {
		t.Fatal("cover-img not found in manifestByID")
	}
	if mi.Properties != "cover-image" {
		t.Errorf("cover-img Properties = %q, want %q", mi.Properties, "cover-image")
	}
}

func TestNewReader_OPFWithHTMLEntities(t *testing.T) {
	files := map[string]string{
		"mimetype":               "application/epub+zip",
		"META-INF/container.xml": validContainerXML,
		"OEBPS/content.opf":     testOPFWithEntities,
	}
	data := buildTestEPubBytes(t, files)

	book, err := NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("NewReader() error = %v", err)
	}
	defer book.Close()

	if book.opf == nil {
		t.Fatal("book.opf is nil")
	}
	if len(book.opf.Metadata.Titles) == 0 {
		t.Fatal("no titles parsed")
	}
}
