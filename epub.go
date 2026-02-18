package epub

import (
	"archive/zip"
	"fmt"
	"io"
	"path"
	"strings"
)

// expectedMimetype is the required content of the "mimetype" file in a valid ePub.
const expectedMimetype = "application/epub+zip"

// Book is the main public API type for reading ePub files.
// Use Open or NewReader to create a Book instance.
//
// A Book is not safe for concurrent use by multiple goroutines.
type Book struct {
	zip             *zip.Reader
	zipExact        map[string]*zip.File  // exact-match ZIP file index
	zipLower        map[string]*zip.File  // lowercase ZIP file index
	closer          io.Closer // non-nil only when created via Open()
	opfPath         string
	opfDir          string
	opf             *opfPackage
	manifestByID    map[string]*manifestItem
	manifestByHref  map[string]*manifestItem
	spine           []spineItem
	guide           []guideReference
	metadata        Metadata
	toc             []TOCItem
	landmarks       []TOCItem
	chapters        []Chapter
	warnings        []string
	licenseDetected bool
}

// Open opens an ePub file at the given path.
// The caller must call Close when done reading from the book.
func Open(path string) (*Book, error) {
	zrc, err := zip.OpenReader(path)
	if err != nil {
		return nil, fmt.Errorf("epub: open %s: %w", path, err)
	}

	b, err := initBook(&zrc.Reader, zrc)
	if err != nil {
		zrc.Close()
		return nil, err
	}
	return b, nil
}

// NewReader creates a Book from an io.ReaderAt with the given size.
// The caller is responsible for the lifetime of r; Close only cleans
// up internal state.
func NewReader(r io.ReaderAt, size int64) (*Book, error) {
	zr, err := zip.NewReader(r, size)
	if err != nil {
		return nil, fmt.Errorf("epub: open zip: %w", err)
	}

	return initBook(zr, nil)
}

// initBook performs common initialisation: mimetype validation, container
// parsing, and DRM detection.
func initBook(zr *zip.Reader, closer io.Closer) (*Book, error) {
	b := &Book{
		zip:    zr,
		closer: closer,
	}

	// Build ZIP file index for O(1) lookups.
	b.buildZipIndex()

	// Validate mimetype.
	b.validateMimetype()

	// Parse container to find OPF path.
	opfPath, err := parseContainer(zr)
	if err != nil {
		return nil, err
	}
	b.opfPath = opfPath
	b.opfDir = path.Dir(opfPath)

	// Check for DRM.
	fontObfuscation, err := checkDRM(zr)
	if err != nil {
		return nil, err
	}
	if fontObfuscation {
		b.warnings = append(b.warnings, "font obfuscation detected; obfuscated fonts may not render correctly")
	}

	// Read and parse OPF.
	opfFile := b.findFile(opfPath)
	if opfFile == nil {
		return nil, fmt.Errorf("epub: OPF file not found in archive: %s: %w", opfPath, ErrInvalidEPub)
	}
	opfData, err := readZipFile(opfFile)
	if err != nil {
		return nil, fmt.Errorf("epub: read OPF file: %w", err)
	}

	pkg, err := parseOPF(opfData)
	if err != nil {
		return nil, err
	}
	b.opf = pkg
	b.manifestByID, b.manifestByHref = buildManifestMaps(pkg.Manifest)
	b.spine = buildSpine(pkg.Spine, b.manifestByID)
	b.guide = buildGuide(pkg.Guide)
	b.metadata = extractMetadata(pkg)

	// Parse TOC (nav document or NCX). Errors are non-fatal;
	// a missing TOC results in an empty slice.
	b.parseTOC()

	return b, nil
}

// validateMimetype checks that the first ZIP entry is named "mimetype" and
// contains "application/epub+zip". Deviations are recorded as warnings.
func (b *Book) validateMimetype() {
	if len(b.zip.File) == 0 {
		b.warnings = append(b.warnings, "empty ZIP archive; mimetype entry missing")
		return
	}

	first := b.zip.File[0]
	if first.Name != "mimetype" {
		b.warnings = append(b.warnings, "first ZIP entry is not \"mimetype\"")
		return
	}

	data, err := readZipFile(first)
	if err != nil {
		b.warnings = append(b.warnings, fmt.Sprintf("cannot read mimetype entry: %v", err))
		return
	}

	if string(data) != expectedMimetype {
		b.warnings = append(b.warnings, fmt.Sprintf("unexpected mimetype: %q", string(data)))
	}
}

// Close releases resources held by the Book. When the Book was created via
// Open, Close closes the underlying file. Close is idempotent.
func (b *Book) Close() error {
	if b.closer != nil {
		err := b.closer.Close()
		b.closer = nil
		return err
	}
	return nil
}

// ReadFile reads a file from the ePub archive by its ZIP-internal path.
// The lookup is case-insensitive as a fallback.
func (b *Book) ReadFile(name string) ([]byte, error) {
	f := b.findFile(name)
	if f == nil {
		return nil, ErrFileNotFound
	}
	return readZipFile(f)
}

// readFile implements the bookReader interface for lazy content loading.
func (b *Book) readFile(name string) ([]byte, error) {
	return b.ReadFile(name)
}

// buildZipIndex builds exact-match and lowercase ZIP file indexes for O(1) lookups.
func (b *Book) buildZipIndex() {
	b.zipExact = make(map[string]*zip.File, len(b.zip.File))
	b.zipLower = make(map[string]*zip.File, len(b.zip.File))
	for _, f := range b.zip.File {
		if _, exists := b.zipExact[f.Name]; !exists {
			b.zipExact[f.Name] = f // first match wins for exact
		}
		lower := strings.ToLower(f.Name)
		if _, exists := b.zipLower[lower]; !exists {
			b.zipLower[lower] = f // first match wins for case-insensitive
		}
	}
}

// findFile looks up a ZIP entry by path using the pre-built index.
// It tries an exact match first, then falls back to a case-insensitive match.
func (b *Book) findFile(name string) *zip.File {
	if f, ok := b.zipExact[name]; ok {
		return f
	}
	if f, ok := b.zipLower[strings.ToLower(name)]; ok {
		return f
	}
	return nil
}

// resolveOPFPath resolves a path relative to the OPF directory.
// If href is empty, returns empty. If opfDir is ".", returns href as-is.
func (b *Book) resolveOPFPath(href string) string {
	if href == "" {
		return ""
	}
	if b.opfDir == "." {
		return href
	}
	return path.Join(b.opfDir, href)
}

// HasTOC reports whether the ePub contains a table of contents.
func (b *Book) HasTOC() bool {
	return len(b.toc) > 0
}

// Metadata returns the extracted metadata from the ePub.
func (b *Book) Metadata() Metadata {
	return copyMetadata(b.metadata)
}

// Warnings returns the list of non-fatal warnings accumulated during parsing.
func (b *Book) Warnings() []string {
	return append([]string(nil), b.warnings...)
}

// TOC returns the table of contents as a tree of TOCItem.
// Each item's SpineIndex is set to the index of the corresponding spine item,
// or -1 if no match was found.
func (b *Book) TOC() []TOCItem {
	return copyTOCItems(b.toc)
}

// Landmarks returns the landmarks from an ePub 3 nav document.
// Returns nil for ePub 2 files or when no landmarks are present.
func (b *Book) Landmarks() []TOCItem {
	return copyTOCItems(b.landmarks)
}

// Chapters returns the chapters in spine order. Each Chapter is a lightweight
// handle; content is loaded lazily when RawContent, TextContent, or BodyHTML
// is called. Title is derived from the TOC by matching Href (ignoring fragment).
// The result is cached after the first call.
//
// Note: IsLicense is not populated by Chapters(). Call ContentChapters() to
// trigger Gutenberg license detection; after that call, the cached chapters
// returned by Chapters() will also have IsLicense set.
func (b *Book) Chapters() []Chapter {
	if b.chapters != nil {
		return copyChapters(b.chapters)
	}

	// Build a map from file path (without fragment) → TOC title.
	tocTitleMap := buildTOCTitleMap(b.toc)

	chapters := make([]Chapter, 0, len(b.spine))
	for _, si := range b.spine {
		href := b.resolveOPFPath(si.Href)

		ch := Chapter{
			ID:     si.ID,
			Href:   href,
			Title:  tocTitleMap[href],
			Linear: si.Linear,
			book:   b,
		}

		chapters = append(chapters, ch)
	}

	b.chapters = chapters
	return copyChapters(b.chapters)
}

// ContentChapters returns the chapters in spine order, excluding any
// detected Project Gutenberg license pages (IsLicense == true).
// On the first call, it reads every chapter file to perform license
// detection; subsequent calls use the cached result. After this call,
// Chapters() also returns chapters with IsLicense correctly set.
func (b *Book) ContentChapters() []Chapter {
	b.detectLicenses()
	out := make([]Chapter, 0, len(b.chapters))
	for _, ch := range b.chapters {
		if !ch.IsLicense {
			out = append(out, ch)
		}
	}
	return out
}

// detectLicenses reads each chapter file and marks Gutenberg license pages.
// It runs at most once per Book instance.
func (b *Book) detectLicenses() {
	if b.licenseDetected {
		return
	}
	_ = b.Chapters() // ensure chapters are built
	for i := range b.chapters {
		if raw, err := b.readFile(b.chapters[i].Href); err == nil {
			b.chapters[i].IsLicense = isGutenbergLicense(raw)
		}
	}
	b.licenseDetected = true
}

// buildTOCTitleMap flattens the TOC tree and builds a map from
// file path (without fragment) → title. The first matching entry wins.
func buildTOCTitleMap(items []TOCItem) map[string]string {
	m := make(map[string]string)
	var flat []*TOCItem
	flattenTOCItems(&flat, items)
	for _, item := range flat {
		if item.Href == "" {
			continue
		}
		filePath := hrefWithoutFragment(item.Href)
		if _, exists := m[filePath]; !exists {
			m[filePath] = item.Title
		}
	}
	return m
}

func copyMetadata(in Metadata) Metadata {
	out := in
	out.Titles = append([]string(nil), in.Titles...)
	out.Authors = append([]Author(nil), in.Authors...)
	out.Language = append([]string(nil), in.Language...)
	out.Identifiers = append([]Identifier(nil), in.Identifiers...)
	out.Subjects = append([]string(nil), in.Subjects...)
	return out
}

func copyTOCItems(in []TOCItem) []TOCItem {
	if in == nil {
		return nil
	}
	out := make([]TOCItem, len(in))
	for i := range in {
		out[i] = in[i]
		out[i].Children = copyTOCItems(in[i].Children)
	}
	return out
}

func copyChapters(in []Chapter) []Chapter {
	if in == nil {
		return nil
	}
	return append([]Chapter(nil), in...)
}
