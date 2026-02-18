package epub

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// benchEPubFiles builds a realistic ePub 2 file map with the given number of chapters.
// Each chapter has a title, heading, and a few paragraphs of text.
func benchEPubFiles(numChapters int) map[string]string {
	// Build manifest items, spine refs, and NCX navPoints.
	var manifestItems, spineRefs, navPoints strings.Builder
	manifestItems.WriteString(`<item id="ncx" href="toc.ncx" media-type="application/x-dtbncx+xml"/>`)
	manifestItems.WriteByte('\n')

	for i := 1; i <= numChapters; i++ {
		id := fmt.Sprintf("ch%d", i)
		href := fmt.Sprintf("chapter%03d.xhtml", i)
		fmt.Fprintf(&manifestItems, `    <item id="%s" href="%s" media-type="application/xhtml+xml"/>`, id, href)
		manifestItems.WriteByte('\n')
		fmt.Fprintf(&spineRefs, `    <itemref idref="%s"/>`, id)
		spineRefs.WriteByte('\n')
		fmt.Fprintf(&navPoints, `    <navPoint id="np%d" playOrder="%d"><navLabel><text>Chapter %d</text></navLabel><content src="%s"/></navPoint>`, i, i, i, href)
		navPoints.WriteByte('\n')
	}

	opf := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<package version="2.0" xmlns="http://www.idpf.org/2007/opf" unique-identifier="bookid">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/" xmlns:opf="http://www.idpf.org/2007/opf">
    <dc:title>Benchmark Book</dc:title>
    <dc:creator opf:file-as="Doe, John" opf:role="aut">John Doe</dc:creator>
    <dc:language>en</dc:language>
    <dc:identifier id="bookid" opf:scheme="ISBN">978-0-00-000000-0</dc:identifier>
    <dc:publisher>Bench Press</dc:publisher>
    <dc:date>2025-06-01</dc:date>
    <dc:description>A benchmark test book with %d chapters.</dc:description>
    <dc:subject>Benchmark</dc:subject>
    <dc:subject>Testing</dc:subject>
  </metadata>
  <manifest>
    %s
  </manifest>
  <spine toc="ncx">
    %s
  </spine>
</package>`, numChapters, manifestItems.String(), spineRefs.String())

	ncx := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<ncx xmlns="http://www.daisy.org/z3986/2005/ncx/" version="2005-1">
  <navMap>
    %s
  </navMap>
</ncx>`, navPoints.String())

	containerXML := `<?xml version="1.0" encoding="UTF-8"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
  <rootfiles>
    <rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/>
  </rootfiles>
</container>`

	files := map[string]string{
		"mimetype":               "application/epub+zip",
		"META-INF/container.xml": containerXML,
		"OEBPS/content.opf":     opf,
		"OEBPS/toc.ncx":         ncx,
	}

	// Generate chapter XHTML files with realistic content.
	for i := 1; i <= numChapters; i++ {
		href := fmt.Sprintf("OEBPS/chapter%03d.xhtml", i)
		files[href] = fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<html xmlns="http://www.w3.org/1999/xhtml">
<head><title>Chapter %d</title></head>
<body>
<h1>Chapter %d</h1>
<p>This is the opening paragraph of chapter %d. It contains enough text to simulate a realistic reading experience for benchmark purposes.</p>
<p>The second paragraph continues the narrative with additional details and descriptions that help establish the setting and characters.</p>
<p>A third paragraph adds more substance to ensure the text extraction benchmarks have meaningful content to process.</p>
<p>Finally, the chapter concludes with a closing paragraph that wraps up the events described in this section of the book.</p>
</body>
</html>`, i, i, i)
	}

	return files
}

// buildBenchEPubFile writes an ePub file for benchmarks and returns the path.
// Uses testing.B for fatal errors. The file is written to dir.
func buildBenchEPubFile(b *testing.B, files map[string]string, dir string) string {
	b.Helper()
	buf := new(bytes.Buffer)
	zw := zip.NewWriter(buf)

	// Write mimetype first.
	if mt, ok := files["mimetype"]; ok {
		fw, err := zw.Create("mimetype")
		if err != nil {
			b.Fatalf("buildBenchEPubFile: create mimetype: %v", err)
		}
		if _, err := io.WriteString(fw, mt); err != nil {
			b.Fatalf("buildBenchEPubFile: write mimetype: %v", err)
		}
	}
	for name, content := range files {
		if name == "mimetype" {
			continue
		}
		fw, err := zw.Create(name)
		if err != nil {
			b.Fatalf("buildBenchEPubFile: create %s: %v", name, err)
		}
		if _, err := io.WriteString(fw, content); err != nil {
			b.Fatalf("buildBenchEPubFile: write %s: %v", name, err)
		}
	}
	if err := zw.Close(); err != nil {
		b.Fatalf("buildBenchEPubFile: close writer: %v", err)
	}

	fp := filepath.Join(dir, "bench.epub")
	if err := os.WriteFile(fp, buf.Bytes(), 0644); err != nil {
		b.Fatalf("buildBenchEPubFile: write file: %v", err)
	}
	return fp
}

// BenchmarkOpen measures the time to Open an ePub file, extract metadata, and close it.
// Uses a realistic 10-chapter ePub with NCX and full metadata.
func BenchmarkOpen(b *testing.B) {
	files := benchEPubFiles(10)
	dir := b.TempDir()
	fp := buildBenchEPubFile(b, files, dir)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		book, err := Open(fp)
		if err != nil {
			b.Fatalf("Open: %v", err)
		}
		_ = book.Metadata()
		book.Close()
	}
}

// BenchmarkTextContent measures the time to extract plain text from a single chapter.
// The book is opened once; only TextContent is benchmarked.
func BenchmarkTextContent(b *testing.B) {
	files := benchEPubFiles(10)
	dir := b.TempDir()
	fp := buildBenchEPubFile(b, files, dir)

	book, err := Open(fp)
	if err != nil {
		b.Fatalf("Open: %v", err)
	}
	defer book.Close()

	chapters := book.Chapters()
	if len(chapters) == 0 {
		b.Fatal("no chapters")
	}
	ch := chapters[0]

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := ch.TextContent()
		if err != nil {
			b.Fatalf("TextContent: %v", err)
		}
	}
}

// BenchmarkTOC measures the time to access the TOC.
// TOC is parsed during Open, so this benchmarks the cached access path.
func BenchmarkTOC(b *testing.B) {
	files := benchEPubFiles(10)
	dir := b.TempDir()
	fp := buildBenchEPubFile(b, files, dir)

	book, err := Open(fp)
	if err != nil {
		b.Fatalf("Open: %v", err)
	}
	defer book.Close()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		toc := book.TOC()
		if len(toc) != 10 {
			b.Fatalf("TOC() returned %d items, want 10", len(toc))
		}
	}
}

// BenchmarkChaptersScaling verifies that Chapters() does not read chapter content
// (lazy loading) by benchmarking it across different chapter counts.
// If content were read eagerly, time would scale linearly with chapter count.
func BenchmarkChaptersScaling(b *testing.B) {
	for _, n := range []int{10, 50, 100} {
		b.Run(fmt.Sprintf("chapters_%d", n), func(b *testing.B) {
			files := benchEPubFiles(n)
			dir := b.TempDir()
			fp := buildBenchEPubFile(b, files, dir)

			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				book, err := Open(fp)
				if err != nil {
					b.Fatalf("Open: %v", err)
				}
				chapters := book.Chapters()
				if len(chapters) != n {
					b.Fatalf("Chapters() = %d, want %d", len(chapters), n)
				}
				book.Close()
			}
		})
	}
}
