package epub

import (
	"bytes"
	"reflect"
	"testing"
)

// --- ePub 2 metadata OPF ---

const testMetadataOPFv2 = `<?xml version="1.0" encoding="UTF-8"?>
<package version="2.0" xmlns="http://www.idpf.org/2007/opf" unique-identifier="bookid">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/" xmlns:opf="http://www.idpf.org/2007/opf">
    <dc:title>Main Title</dc:title>
    <dc:creator opf:file-as="Doe, John" opf:role="aut">John Doe</dc:creator>
    <dc:creator opf:file-as="Smith, Jane" opf:role="edt">Jane Smith</dc:creator>
    <dc:language>en</dc:language>
    <dc:language>fr</dc:language>
    <dc:identifier id="bookid" opf:scheme="ISBN">978-3-16-148410-0</dc:identifier>
    <dc:identifier opf:scheme="UUID">urn:uuid:12345</dc:identifier>
    <dc:publisher>Test Publisher</dc:publisher>
    <dc:date>2024-01-15</dc:date>
    <dc:description>A test book description.</dc:description>
    <dc:subject>Fiction</dc:subject>
    <dc:subject>Science</dc:subject>
    <dc:rights>Copyright 2024</dc:rights>
    <dc:source>http://example.com/source</dc:source>
    <meta name="cover" content="cover-img"/>
  </metadata>
  <manifest>
    <item id="chap1" href="chapter1.xhtml" media-type="application/xhtml+xml"/>
  </manifest>
  <spine>
    <itemref idref="chap1"/>
  </spine>
</package>`

// --- ePub 3 metadata OPF ---

const testMetadataOPFv3 = `<?xml version="1.0" encoding="UTF-8"?>
<package version="3.0" xmlns="http://www.idpf.org/2007/opf" unique-identifier="uid">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
    <dc:title id="title1">Subtitle</dc:title>
    <dc:title id="title2">Main Title</dc:title>
    <dc:creator id="creator1">John Doe</dc:creator>
    <dc:creator id="creator2">Jane Smith</dc:creator>
    <dc:language>en</dc:language>
    <dc:identifier id="uid">urn:uuid:12345-67890</dc:identifier>
    <dc:publisher>EPUB 3 Publisher</dc:publisher>
    <dc:date>2024-06-01</dc:date>
    <dc:description>An EPUB 3 test book.</dc:description>
    <dc:subject>Technology</dc:subject>
    <dc:rights>CC BY 4.0</dc:rights>
    <dc:source>http://example.com</dc:source>
    <meta property="dcterms:modified">2024-06-15T00:00:00Z</meta>
    <meta refines="#title1" property="display-seq">2</meta>
    <meta refines="#title2" property="display-seq">1</meta>
    <meta refines="#creator1" property="file-as">Doe, John</meta>
    <meta refines="#creator1" property="role" scheme="marc:relators">aut</meta>
    <meta refines="#creator2" property="file-as">Smith, Jane</meta>
    <meta refines="#creator2" property="role" scheme="marc:relators">edt</meta>
    <meta refines="#uid" property="identifier-type">UUID</meta>
  </metadata>
  <manifest>
    <item id="chap1" href="chapter1.xhtml" media-type="application/xhtml+xml"/>
  </manifest>
  <spine>
    <itemref idref="chap1"/>
  </spine>
</package>`

// --- Minimal metadata OPF ---

const testMetadataOPFMinimal = `<?xml version="1.0" encoding="UTF-8"?>
<package version="2.0" xmlns="http://www.idpf.org/2007/opf">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
    <dc:title>Only Title</dc:title>
  </metadata>
  <manifest>
    <item id="chap1" href="chapter1.xhtml" media-type="application/xhtml+xml"/>
  </manifest>
  <spine>
    <itemref idref="chap1"/>
  </spine>
</package>`

// --- extractMetadata unit tests ---

func TestExtractMetadata_V2(t *testing.T) {
	pkg, err := parseOPF([]byte(testMetadataOPFv2))
	if err != nil {
		t.Fatalf("parseOPF() error = %v", err)
	}

	md := extractMetadata(pkg)

	if md.Version != "2.0" {
		t.Errorf("Version = %q, want %q", md.Version, "2.0")
	}

	// Titles.
	wantTitles := []string{"Main Title"}
	if !reflect.DeepEqual(md.Titles, wantTitles) {
		t.Errorf("Titles = %v, want %v", md.Titles, wantTitles)
	}

	// Authors.
	if len(md.Authors) != 2 {
		t.Fatalf("Authors count = %d, want 2", len(md.Authors))
	}
	if md.Authors[0].Name != "John Doe" {
		t.Errorf("Authors[0].Name = %q, want %q", md.Authors[0].Name, "John Doe")
	}
	if md.Authors[0].FileAs != "Doe, John" {
		t.Errorf("Authors[0].FileAs = %q, want %q", md.Authors[0].FileAs, "Doe, John")
	}
	if md.Authors[0].Role != "aut" {
		t.Errorf("Authors[0].Role = %q, want %q", md.Authors[0].Role, "aut")
	}
	if md.Authors[1].Name != "Jane Smith" {
		t.Errorf("Authors[1].Name = %q, want %q", md.Authors[1].Name, "Jane Smith")
	}
	if md.Authors[1].Role != "edt" {
		t.Errorf("Authors[1].Role = %q, want %q", md.Authors[1].Role, "edt")
	}

	// Languages.
	wantLangs := []string{"en", "fr"}
	if !reflect.DeepEqual(md.Language, wantLangs) {
		t.Errorf("Language = %v, want %v", md.Language, wantLangs)
	}

	// Identifiers.
	if len(md.Identifiers) != 2 {
		t.Fatalf("Identifiers count = %d, want 2", len(md.Identifiers))
	}
	if md.Identifiers[0].Value != "978-3-16-148410-0" {
		t.Errorf("Identifiers[0].Value = %q, want %q", md.Identifiers[0].Value, "978-3-16-148410-0")
	}
	if md.Identifiers[0].Scheme != "ISBN" {
		t.Errorf("Identifiers[0].Scheme = %q, want %q", md.Identifiers[0].Scheme, "ISBN")
	}
	if md.Identifiers[0].ID != "bookid" {
		t.Errorf("Identifiers[0].ID = %q, want %q", md.Identifiers[0].ID, "bookid")
	}
	if md.Identifiers[1].Value != "urn:uuid:12345" {
		t.Errorf("Identifiers[1].Value = %q, want %q", md.Identifiers[1].Value, "urn:uuid:12345")
	}

	// Single-value fields.
	if md.Publisher != "Test Publisher" {
		t.Errorf("Publisher = %q, want %q", md.Publisher, "Test Publisher")
	}
	if md.Date != "2024-01-15" {
		t.Errorf("Date = %q, want %q", md.Date, "2024-01-15")
	}
	if md.Description != "A test book description." {
		t.Errorf("Description = %q, want %q", md.Description, "A test book description.")
	}
	if md.Rights != "Copyright 2024" {
		t.Errorf("Rights = %q, want %q", md.Rights, "Copyright 2024")
	}
	if md.Source != "http://example.com/source" {
		t.Errorf("Source = %q, want %q", md.Source, "http://example.com/source")
	}

	// Subjects.
	wantSubjects := []string{"Fiction", "Science"}
	if !reflect.DeepEqual(md.Subjects, wantSubjects) {
		t.Errorf("Subjects = %v, want %v", md.Subjects, wantSubjects)
	}
}

func TestExtractMetadata_V3(t *testing.T) {
	pkg, err := parseOPF([]byte(testMetadataOPFv3))
	if err != nil {
		t.Fatalf("parseOPF() error = %v", err)
	}

	md := extractMetadata(pkg)

	if md.Version != "3.0" {
		t.Errorf("Version = %q, want %q", md.Version, "3.0")
	}

	// Titles: display-seq reorders — "Main Title" (seq=1) before "Subtitle" (seq=2).
	wantTitles := []string{"Main Title", "Subtitle"}
	if !reflect.DeepEqual(md.Titles, wantTitles) {
		t.Errorf("Titles = %v, want %v", md.Titles, wantTitles)
	}

	// Authors from refines.
	if len(md.Authors) != 2 {
		t.Fatalf("Authors count = %d, want 2", len(md.Authors))
	}
	if md.Authors[0].Name != "John Doe" {
		t.Errorf("Authors[0].Name = %q, want %q", md.Authors[0].Name, "John Doe")
	}
	if md.Authors[0].FileAs != "Doe, John" {
		t.Errorf("Authors[0].FileAs = %q, want %q", md.Authors[0].FileAs, "Doe, John")
	}
	if md.Authors[0].Role != "aut" {
		t.Errorf("Authors[0].Role = %q, want %q", md.Authors[0].Role, "aut")
	}
	if md.Authors[1].FileAs != "Smith, Jane" {
		t.Errorf("Authors[1].FileAs = %q, want %q", md.Authors[1].FileAs, "Smith, Jane")
	}
	if md.Authors[1].Role != "edt" {
		t.Errorf("Authors[1].Role = %q, want %q", md.Authors[1].Role, "edt")
	}

	// Identifier with scheme from refines.
	if len(md.Identifiers) != 1 {
		t.Fatalf("Identifiers count = %d, want 1", len(md.Identifiers))
	}
	if md.Identifiers[0].Scheme != "UUID" {
		t.Errorf("Identifiers[0].Scheme = %q, want %q", md.Identifiers[0].Scheme, "UUID")
	}

	// Single-value fields.
	if md.Publisher != "EPUB 3 Publisher" {
		t.Errorf("Publisher = %q, want %q", md.Publisher, "EPUB 3 Publisher")
	}
	if md.Date != "2024-06-01" {
		t.Errorf("Date = %q, want %q", md.Date, "2024-06-01")
	}
	if md.Description != "An EPUB 3 test book." {
		t.Errorf("Description = %q, want %q", md.Description, "An EPUB 3 test book.")
	}
	if md.Rights != "CC BY 4.0" {
		t.Errorf("Rights = %q, want %q", md.Rights, "CC BY 4.0")
	}
	if md.Source != "http://example.com" {
		t.Errorf("Source = %q, want %q", md.Source, "http://example.com")
	}
}

func TestExtractMetadata_Minimal(t *testing.T) {
	pkg, err := parseOPF([]byte(testMetadataOPFMinimal))
	if err != nil {
		t.Fatalf("parseOPF() error = %v", err)
	}

	md := extractMetadata(pkg)

	if md.Version != "2.0" {
		t.Errorf("Version = %q, want %q", md.Version, "2.0")
	}
	wantTitles := []string{"Only Title"}
	if !reflect.DeepEqual(md.Titles, wantTitles) {
		t.Errorf("Titles = %v, want %v", md.Titles, wantTitles)
	}

	// All optional fields should be zero-valued.
	if len(md.Authors) != 0 {
		t.Errorf("Authors = %v, want nil", md.Authors)
	}
	if len(md.Language) != 0 {
		t.Errorf("Language = %v, want nil", md.Language)
	}
	if len(md.Identifiers) != 0 {
		t.Errorf("Identifiers = %v, want nil", md.Identifiers)
	}
	if md.Publisher != "" {
		t.Errorf("Publisher = %q, want empty", md.Publisher)
	}
	if md.Date != "" {
		t.Errorf("Date = %q, want empty", md.Date)
	}
	if md.Description != "" {
		t.Errorf("Description = %q, want empty", md.Description)
	}
	if len(md.Subjects) != 0 {
		t.Errorf("Subjects = %v, want nil", md.Subjects)
	}
	if md.Rights != "" {
		t.Errorf("Rights = %q, want empty", md.Rights)
	}
	if md.Source != "" {
		t.Errorf("Source = %q, want empty", md.Source)
	}
}

func TestExtractMetadata_EmptyMetadata(t *testing.T) {
	pkg, err := parseOPF([]byte(`<?xml version="1.0"?><package version="3.0"/>`))
	if err != nil {
		t.Fatalf("parseOPF() error = %v", err)
	}

	md := extractMetadata(pkg)

	if md.Version != "3.0" {
		t.Errorf("Version = %q, want %q", md.Version, "3.0")
	}
	if md.Titles != nil {
		t.Errorf("Titles = %v, want nil", md.Titles)
	}
	if md.Authors != nil {
		t.Errorf("Authors = %v, want nil", md.Authors)
	}
}

func TestExtractMetadata_V3TitleOrdering(t *testing.T) {
	opf := `<?xml version="1.0" encoding="UTF-8"?>
<package version="3.0" xmlns="http://www.idpf.org/2007/opf">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
    <dc:title id="t1">Third</dc:title>
    <dc:title id="t2">First</dc:title>
    <dc:title id="t3">Second</dc:title>
    <dc:title>No Seq</dc:title>
    <meta refines="#t1" property="display-seq">3</meta>
    <meta refines="#t2" property="display-seq">1</meta>
    <meta refines="#t3" property="display-seq">2</meta>
  </metadata>
  <manifest/>
  <spine/>
</package>`
	pkg, err := parseOPF([]byte(opf))
	if err != nil {
		t.Fatalf("parseOPF() error = %v", err)
	}

	md := extractMetadata(pkg)

	// "First" (seq=1), "Second" (seq=2), "Third" (seq=3), "No Seq" (no seq → goes last).
	want := []string{"First", "Second", "Third", "No Seq"}
	if !reflect.DeepEqual(md.Titles, want) {
		t.Errorf("Titles = %v, want %v", md.Titles, want)
	}
}

func TestExtractMetadata_CreatorNoRole(t *testing.T) {
	opf := `<?xml version="1.0" encoding="UTF-8"?>
<package version="2.0" xmlns="http://www.idpf.org/2007/opf">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
    <dc:creator>Plain Author</dc:creator>
  </metadata>
  <manifest/>
  <spine/>
</package>`
	pkg, err := parseOPF([]byte(opf))
	if err != nil {
		t.Fatalf("parseOPF() error = %v", err)
	}

	md := extractMetadata(pkg)

	if len(md.Authors) != 1 {
		t.Fatalf("Authors count = %d, want 1", len(md.Authors))
	}
	if md.Authors[0].Name != "Plain Author" {
		t.Errorf("Authors[0].Name = %q, want %q", md.Authors[0].Name, "Plain Author")
	}
	if md.Authors[0].FileAs != "" {
		t.Errorf("Authors[0].FileAs = %q, want empty", md.Authors[0].FileAs)
	}
	if md.Authors[0].Role != "" {
		t.Errorf("Authors[0].Role = %q, want empty", md.Authors[0].Role)
	}
}

// --- Integration test: Book.Metadata() ---

func TestBookMetadata_V2(t *testing.T) {
	files := map[string]string{
		"mimetype":               "application/epub+zip",
		"META-INF/container.xml": validContainerXML,
		"OEBPS/content.opf":     testMetadataOPFv2,
	}
	data := buildTestEPubBytes(t, files)

	book, err := NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("NewReader() error = %v", err)
	}
	defer book.Close()

	md := book.Metadata()

	if md.Version != "2.0" {
		t.Errorf("Version = %q, want %q", md.Version, "2.0")
	}
	if len(md.Titles) != 1 || md.Titles[0] != "Main Title" {
		t.Errorf("Titles = %v, want [Main Title]", md.Titles)
	}
	if len(md.Authors) != 2 {
		t.Fatalf("Authors count = %d, want 2", len(md.Authors))
	}
	if md.Authors[0].Name != "John Doe" {
		t.Errorf("Authors[0].Name = %q, want %q", md.Authors[0].Name, "John Doe")
	}
	if md.Publisher != "Test Publisher" {
		t.Errorf("Publisher = %q, want %q", md.Publisher, "Test Publisher")
	}
}

func TestBookMetadata_V3(t *testing.T) {
	files := map[string]string{
		"mimetype":               "application/epub+zip",
		"META-INF/container.xml": validContainerXML,
		"OEBPS/content.opf":     testMetadataOPFv3,
	}
	data := buildTestEPubBytes(t, files)

	book, err := NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("NewReader() error = %v", err)
	}
	defer book.Close()

	md := book.Metadata()

	if md.Version != "3.0" {
		t.Errorf("Version = %q, want %q", md.Version, "3.0")
	}

	// Titles ordered by display-seq.
	wantTitles := []string{"Main Title", "Subtitle"}
	if !reflect.DeepEqual(md.Titles, wantTitles) {
		t.Errorf("Titles = %v, want %v", md.Titles, wantTitles)
	}

	if md.Authors[0].FileAs != "Doe, John" {
		t.Errorf("Authors[0].FileAs = %q, want %q", md.Authors[0].FileAs, "Doe, John")
	}
}
