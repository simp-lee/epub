// Package epub provides a pure-Go library for reading and parsing ePub 2 and ePub 3 files.
//
// It extracts metadata (Dublin Core), table of contents (NCX and Nav), spine-ordered
// chapters with lazy content loading, cover images, and landmarks. DRM-protected
// files are detected and rejected with [ErrDRMProtected].
//
// # Opening an ePub
//
// Use [Open] to open a file by path, or [NewReader] to read from an [io.ReaderAt]:
//
//	book, err := epub.Open("book.epub")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer book.Close()
//
// # Metadata
//
// The [Book.Metadata] method returns a [Metadata] struct containing titles, authors,
// language, identifiers (ISBN/UUID), publisher, date, description, subjects, and more:
//
//	md := book.Metadata()
//	fmt.Println(md.Titles[0])
//
// # Table of Contents
//
// The [Book.TOC] method returns a tree of [TOCItem] entries. Each item includes
// a spine index range indicating which spine items it covers:
//
//	for _, item := range book.TOC() {
//	    fmt.Println(item.Title, item.Href)
//	}
//
// # Chapters
//
// Chapters are returned in spine order via [Book.Chapters]. Content is loaded lazily;
// call [Chapter.RawContent] for raw XHTML, [Chapter.TextContent] for plain text, or
// [Chapter.BodyHTML] for sanitised inner HTML with rewritten image paths:
//
//	for _, ch := range book.Chapters() {
//	    text, _ := ch.TextContent()
//	    fmt.Println(ch.Title, len(text))
//	}
//
// Use [Book.ContentChapters] to exclude Project Gutenberg license pages.
//
// # Cover Image
//
// [Book.Cover] attempts multiple strategies (ePub 3 properties, ePub 2 meta,
// guide reference, manifest heuristic, first spine item) to locate the cover:
//
//	cover, err := book.Cover()
//	if err == nil {
//	    os.WriteFile("cover.jpg", cover.Data, 0644)
//	}
//
// # Error Handling
//
// The package defines sentinel errors for common failure cases:
//   - [ErrDRMProtected] – the file is DRM encrypted
//   - [ErrInvalidEPub] – structural validation failed
//   - [ErrInvalidChapter] – a Chapter handle is invalid
//   - [ErrFileNotFound] – a requested file is not in the archive
//   - [ErrNoCover] – no cover image could be detected
//
// If no table of contents is present, [Book.TOC] returns an empty slice
// and [Book.HasTOC] returns false.
package epub
