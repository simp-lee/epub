package epub

// Metadata holds the Dublin Core and other metadata extracted from the OPF file.
type Metadata struct {
	// Version is the ePub specification version (e.g., "2.0", "3.0").
	Version string

	// Titles contains all dc:title values. The first entry is the primary title.
	Titles []string

	// Authors contains all dc:creator entries with their roles and file-as values.
	Authors []Author

	// Language contains all dc:language values (BCP 47 tags, e.g., "en", "zh-CN").
	Language []string

	// Identifiers contains all dc:identifier entries (ISBN, UUID, URI, etc.).
	Identifiers []Identifier

	// Publisher is the dc:publisher value.
	Publisher string

	// Date is the dc:date value (publication date as raw string).
	Date string

	// Description is the dc:description value.
	Description string

	// Subjects contains all dc:subject values.
	Subjects []string

	// Rights is the dc:rights value.
	Rights string

	// Source is the dc:source value.
	Source string
}

// Author represents a dc:creator entry with optional file-as and role attributes.
type Author struct {
	// Name is the display name of the author (dc:creator text content).
	Name string

	// FileAs is the opf:file-as attribute value (e.g., "Dickens, Charles").
	FileAs string

	// Role is the opf:role attribute value (e.g., "aut", "edt", "trl").
	Role string
}

// Identifier represents a dc:identifier entry.
type Identifier struct {
	// Value is the identifier text content (e.g., ISBN, UUID, URI).
	Value string

	// Scheme is the opf:scheme attribute value (e.g., "ISBN", "UUID").
	Scheme string

	// ID is the xml id attribute of this identifier element.
	ID string
}

// TOCItem represents a single entry in the table of contents.
// TOC is a tree structure; each item may have nested children.
type TOCItem struct {
	// Title is the display text of the TOC entry.
	Title string

	// Href is the content file reference (may include a fragment, e.g., "chapter01.xhtml#section2").
	Href string

	// Children contains nested TOC entries under this item.
	Children []TOCItem

	// SpineIndex is the index into the spine that this TOC entry points to.
	// A value of -1 indicates no spine association was found.
	SpineIndex int

	// SpineEndIndex is the exclusive end index into the spine for this TOC entry.
	// The entry covers spine[SpineIndex:SpineEndIndex]. For example, if SpineIndex=0
	// and SpineEndIndex=3, the entry covers spine items 0, 1, and 2.
	// A value of -1 indicates no spine association was found.
	SpineEndIndex int
}

// Chapter represents a spine item with methods for content access.
// Content is loaded lazily from the underlying ePub archive.
type Chapter struct {
	// Title is the chapter title derived from the TOC (empty if not in TOC).
	Title string

	// Href is the content file path within the ePub archive.
	Href string

	// ID is the manifest item ID for this chapter.
	ID string

	// Linear indicates whether this chapter is part of the linear reading order.
	Linear bool

	// IsLicense indicates whether this chapter is a Project Gutenberg license page.
	// Detection is based on known Gutenberg license patterns in the text content.
	IsLicense bool

	// book is a reference to the parent Book for lazy content loading.
	// This will be set when chapters are constructed during parsing.
	book bookReader
}

// bookReader is a private interface for lazy content loading from the ePub archive.
// It is implemented by the Book type defined in epub.go.
type bookReader interface {
	readFile(path string) ([]byte, error)
}

// CoverImage holds the detected cover image data.
type CoverImage struct {
	// Path is the ZIP-internal path to the cover image file.
	Path string

	// MediaType is the MIME type of the cover image (e.g., "image/jpeg").
	MediaType string

	// Data is the raw image bytes.
	Data []byte
}

// spineItem represents an entry in the OPF <spine> element.
type spineItem struct {
	// ID is the manifest item ID referenced by this spine entry.
	ID string

	// Href is the content file path within the ePub archive.
	Href string

	// MediaType is the MIME type of the referenced content file.
	MediaType string

	// Linear indicates whether this item is part of the linear reading order.
	// Items with linear="no" in the OPF are non-linear.
	Linear bool

	// IDRef is the idref attribute value from the <itemref> element.
	IDRef string
}

// manifestItem represents an entry in the OPF <manifest> element.
type manifestItem struct {
	// ID is the unique identifier of this manifest item.
	ID string

	// Href is the file path relative to the OPF file location.
	Href string

	// MediaType is the MIME type of the resource.
	MediaType string

	// Properties contains space-separated property values (ePub 3, e.g., "nav", "cover-image").
	Properties string
}
