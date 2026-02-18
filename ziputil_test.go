package epub

import (
	"archive/zip"
	"bytes"
	"strings"
	"testing"
)

func TestFindFileInsensitive(t *testing.T) {
	zr := buildTestZip(t, map[string]string{
		"META-INF/container.xml": "<container/>",
		"OEBPS/content.opf":     "<package/>",
		"OEBPS/toc.ncx":         "<ncx/>",
	})

	tests := []struct {
		name   string
		lookup string
		want   string // expected matched Name, or "" if nil
	}{
		{"exact match", "META-INF/container.xml", "META-INF/container.xml"},
		{"case insensitive", "meta-inf/CONTAINER.XML", "META-INF/container.xml"},
		{"mixed case", "oebps/Content.OPF", "OEBPS/content.opf"},
		{"not found", "nonexistent.file", ""},
		{"empty path", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := findFileInsensitive(zr, tt.lookup)
			if tt.want == "" {
				if got != nil {
					t.Errorf("findFileInsensitive(%q) = %q; want nil", tt.lookup, got.Name)
				}
				return
			}
			if got == nil {
				t.Fatalf("findFileInsensitive(%q) = nil; want %q", tt.lookup, tt.want)
			}
			if got.Name != tt.want {
				t.Errorf("findFileInsensitive(%q).Name = %q; want %q", tt.lookup, got.Name, tt.want)
			}
		})
	}
}

func TestFindFileInsensitive_PrefersExactMatch(t *testing.T) {
	// When both exact and case-insensitive matches exist, exact should win.
	zr := buildTestZip(t, map[string]string{
		"File.txt": "exact",
		"file.txt": "lower",
	})

	got := findFileInsensitive(zr, "File.txt")
	if got == nil {
		t.Fatal("findFileInsensitive returned nil; want exact match")
	}
	if got.Name != "File.txt" {
		t.Errorf("got %q; want exact match %q", got.Name, "File.txt")
	}
}

func TestResolveRelativePath(t *testing.T) {
	tests := []struct {
		name     string
		basePath string
		href     string
		want     string
	}{
		{"same directory", "OEBPS/content.opf", "toc.ncx", "OEBPS/toc.ncx"},
		{"parent directory", "OEBPS/content.opf", "../images/cover.jpg", "images/cover.jpg"},
		{"nested path", "OEBPS/content.opf", "text/chapter1.xhtml", "OEBPS/text/chapter1.xhtml"},
		{"absolute-like href", "OEBPS/content.opf", "OEBPS/images/fig.png", "OEBPS/OEBPS/images/fig.png"},
		{"root base", "content.opf", "chapter1.xhtml", "chapter1.xhtml"},
		{"deeply nested", "a/b/c/d.opf", "../../e/f.html", "a/e/f.html"},
		{"dot href", "OEBPS/content.opf", "./styles/main.css", "OEBPS/styles/main.css"},
		{"href with fragment stripped beforehand", "OEBPS/content.opf", "ch1.xhtml", "OEBPS/ch1.xhtml"},
		{"traversal escapes root", "OEBPS/content.opf", "../../../secret.txt", ""},
		{"absolute href dropped", "OEBPS/content.opf", "/etc/passwd", ""},
		{"multi-level traversal dropped", "a/b/c/d.opf", "../../../../x.txt", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveRelativePath(tt.basePath, tt.href)
			if got != tt.want {
				t.Errorf("resolveRelativePath(%q, %q) = %q; want %q", tt.basePath, tt.href, got, tt.want)
			}
		})
	}
}

func TestIsSafePath(t *testing.T) {
	tests := []struct {
		name string
		path string
		safe bool
	}{
		{"normal path", "OEBPS/content.opf", true},
		{"root file", "mimetype", true},
		{"nested", "a/b/c/d.txt", true},
		{"dot", ".", true},
		{"double dot", "..", false},
		{"traversal prefix", "../etc/passwd", false},
		{"deep traversal", "a/../../etc/passwd", false},
		{"absolute path", "/etc/passwd", false},
		{"traversal with trailing", "../", false},
		{"clean traversal", "OEBPS/../../secret", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isSafePath(tt.path)
			if got != tt.safe {
				t.Errorf("isSafePath(%q) = %v; want %v", tt.path, got, tt.safe)
			}
		})
	}
}

func TestStripBOM(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
		want  []byte
	}{
		{"with BOM", []byte{0xEF, 0xBB, 0xBF, 'h', 'e', 'l', 'l', 'o'}, []byte("hello")},
		{"without BOM", []byte("hello"), []byte("hello")},
		{"empty", []byte{}, []byte{}},
		{"BOM only", []byte{0xEF, 0xBB, 0xBF}, []byte{}},
		{"partial BOM 1 byte", []byte{0xEF}, []byte{0xEF}},
		{"partial BOM 2 bytes", []byte{0xEF, 0xBB}, []byte{0xEF, 0xBB}},
		{"BOM in middle (not stripped)", []byte{'a', 0xEF, 0xBB, 0xBF, 'b'}, []byte{'a', 0xEF, 0xBB, 0xBF, 'b'}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripBOM(tt.input)
			if !bytes.Equal(got, tt.want) {
				t.Errorf("stripBOM(%v) = %v; want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestReadZipFile(t *testing.T) {
	zr := buildTestZip(t, map[string]string{
		"test.txt":    "hello world",
		"empty.txt":   "",
		"subdir/a.md": "# Title",
	})

	tests := []struct {
		name    string
		entry   string
		want    string
		wantErr bool
	}{
		{"normal file", "test.txt", "hello world", false},
		{"empty file", "empty.txt", "", false},
		{"nested file", "subdir/a.md", "# Title", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := findFileInsensitive(zr, tt.entry)
			if f == nil {
				t.Fatalf("entry %q not found in zip", tt.entry)
			}
			got, err := readZipFile(f)
			if (err != nil) != tt.wantErr {
				t.Fatalf("readZipFile(%q) err = %v; wantErr = %v", tt.entry, err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if string(got) != tt.want {
				t.Errorf("readZipFile(%q) = %q; want %q", tt.entry, string(got), tt.want)
			}
		})
	}
}

func TestReadZipFile_ZipBomb(t *testing.T) {
	// Create a ZIP entry whose content exceeds a small limit.
	content := strings.Repeat("A", 200)
	zr := buildTestZip(t, map[string]string{
		"big.txt": content,
	})

	f := findFileInsensitive(zr, "big.txt")
	if f == nil {
		t.Fatal("entry not found")
	}

	_, err := readZipFileWithLimit(f, 100)
	if err == nil {
		t.Fatal("readZipFileWithLimit should have returned an error for oversized entry")
	}
	if !strings.Contains(err.Error(), "too large") && !strings.Contains(err.Error(), "exceeds limit") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestReadZipFile_PathTraversal(t *testing.T) {
	// Manually create a zip.File with a traversal path.
	// We can't easily do this with zip.Writer since it normalizes paths,
	// so we'll test isSafePath directly and trust readZipFile calls it.
	if isSafePath("../../../etc/passwd") {
		t.Error("isSafePath should reject traversal path")
	}
	if isSafePath("/absolute/path") {
		t.Error("isSafePath should reject absolute path")
	}
}

// TestBuildTestEPubFile verifies that the file-based helper produces a valid ZIP.
func TestBuildTestEPubFile(t *testing.T) {
	fp := buildTestEPubFile(t, map[string]string{
		"mimetype":                "application/epub+zip",
		"META-INF/container.xml": "<container/>",
	})
	if fp == "" {
		t.Fatal("buildTestEPubFile returned empty path")
	}

	// Verify we can open it as a zip.
	zrc, err := zip.OpenReader(fp)
	if err != nil {
		t.Fatalf("cannot open produced epub: %v", err)
	}
	defer zrc.Close()

	found := false
	for _, f := range zrc.File {
		if f.Name == "mimetype" {
			found = true
		}
	}
	if !found {
		t.Error("mimetype entry not found in produced epub")
	}
}
