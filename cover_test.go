package epub

import (
	"errors"
	"testing"
)

// coverOPF returns an OPF template with the given metadata, manifest, spine,
// and guide XML fragments inserted.
func coverOPF(meta, manifest, spine, guide string) string {
	return `<?xml version="1.0" encoding="UTF-8"?>
<package xmlns="http://www.idpf.org/2007/opf" version="3.0">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">` + meta + `</metadata>
  <manifest>` + manifest + `</manifest>
  <spine>` + spine + `</spine>
  <guide>` + guide + `</guide>
</package>`
}

// coverEPubFiles returns the minimum ePub file set with the given OPF and any
// extra files merged in.
func coverEPubFiles(opf string, extra map[string]string) map[string]string {
	files := map[string]string{
		"mimetype":               "application/epub+zip",
		"META-INF/container.xml": validContainerXML,
		"OEBPS/content.opf":      opf,
	}
	for k, v := range extra {
		files[k] = v
	}
	return files
}

func TestCover_Strategy1_CoverImageProperty(t *testing.T) {
	opf := coverOPF("",
		`<item id="cover-img" href="images/cover.jpg" media-type="image/jpeg" properties="cover-image"/>
		 <item id="ch1" href="ch1.xhtml" media-type="application/xhtml+xml"/>`,
		`<itemref idref="ch1"/>`,
		"")
	files := coverEPubFiles(opf, map[string]string{
		"OEBPS/images/cover.jpg": "FAKE-JPEG-DATA",
	})
	fp := buildTestEPubFile(t, files)

	book, err := Open(fp)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer book.Close()

	cover, err := book.Cover()
	if err != nil {
		t.Fatalf("Cover() error = %v", err)
	}
	if cover.Path != "OEBPS/images/cover.jpg" {
		t.Errorf("Cover().Path = %q, want %q", cover.Path, "OEBPS/images/cover.jpg")
	}
	if cover.MediaType != "image/jpeg" {
		t.Errorf("Cover().MediaType = %q, want %q", cover.MediaType, "image/jpeg")
	}
	if string(cover.Data) != "FAKE-JPEG-DATA" {
		t.Errorf("Cover().Data = %q, want %q", string(cover.Data), "FAKE-JPEG-DATA")
	}
}

func TestCover_Strategy1_MultipleProperties(t *testing.T) {
	// Properties field may contain multiple space-separated values.
	opf := coverOPF("",
		`<item id="cover-img" href="cover.png" media-type="image/png" properties="cover-image svg"/>`,
		"", "")
	files := coverEPubFiles(opf, map[string]string{
		"OEBPS/cover.png": "PNG-DATA",
	})
	fp := buildTestEPubFile(t, files)

	book, err := Open(fp)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer book.Close()

	cover, err := book.Cover()
	if err != nil {
		t.Fatalf("Cover() error = %v", err)
	}
	if cover.Path != "OEBPS/cover.png" {
		t.Errorf("Cover().Path = %q, want %q", cover.Path, "OEBPS/cover.png")
	}
}

func TestCover_Strategy2_MetaCover(t *testing.T) {
	opf := coverOPF(
		`<meta name="cover" content="cover-id"/>`,
		`<item id="cover-id" href="cover.jpg" media-type="image/jpeg"/>`,
		"", "")
	files := coverEPubFiles(opf, map[string]string{
		"OEBPS/cover.jpg": "JPEG-DATA",
	})
	fp := buildTestEPubFile(t, files)

	book, err := Open(fp)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer book.Close()

	cover, err := book.Cover()
	if err != nil {
		t.Fatalf("Cover() error = %v", err)
	}
	if cover.Path != "OEBPS/cover.jpg" {
		t.Errorf("Cover().Path = %q, want %q", cover.Path, "OEBPS/cover.jpg")
	}
	if cover.MediaType != "image/jpeg" {
		t.Errorf("Cover().MediaType = %q, want %q", cover.MediaType, "image/jpeg")
	}
}

func TestCover_Strategy2_MetaCover_CaseInsensitive(t *testing.T) {
	opf := coverOPF(
		`<meta name="Cover" content="cover-id"/>`,
		`<item id="cover-id" href="cover.jpg" media-type="Image/JPEG"/>`,
		"", "")
	files := coverEPubFiles(opf, map[string]string{
		"OEBPS/cover.jpg": "JPEG-DATA",
	})
	fp := buildTestEPubFile(t, files)

	book, err := Open(fp)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer book.Close()

	cover, err := book.Cover()
	if err != nil {
		t.Fatalf("Cover() error = %v", err)
	}
	if cover.Path != "OEBPS/cover.jpg" {
		t.Errorf("Cover().Path = %q, want %q", cover.Path, "OEBPS/cover.jpg")
	}
	if cover.MediaType != "Image/JPEG" {
		t.Errorf("Cover().MediaType = %q, want %q", cover.MediaType, "Image/JPEG")
	}
}

func TestCover_Strategy2_MetaCover_XHTMLFallback(t *testing.T) {
	// When <meta name="cover"> points to an XHTML file, extract the <img>.
	opf := coverOPF(
		`<meta name="cover" content="cover-page"/>`,
		`<item id="cover-page" href="cover.xhtml" media-type="application/xhtml+xml"/>
		 <item id="cover-img" href="images/cover.jpg" media-type="image/jpeg"/>`,
		"", "")

	coverXHTML := `<?xml version="1.0" encoding="UTF-8"?>
<html xmlns="http://www.w3.org/1999/xhtml">
<body><img src="images/cover.jpg" alt="Cover"/></body>
</html>`

	files := coverEPubFiles(opf, map[string]string{
		"OEBPS/cover.xhtml":      coverXHTML,
		"OEBPS/images/cover.jpg": "JPEG-FROM-XHTML",
	})
	fp := buildTestEPubFile(t, files)

	book, err := Open(fp)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer book.Close()

	cover, err := book.Cover()
	if err != nil {
		t.Fatalf("Cover() error = %v", err)
	}
	if cover.Path != "OEBPS/images/cover.jpg" {
		t.Errorf("Cover().Path = %q, want %q", cover.Path, "OEBPS/images/cover.jpg")
	}
	if cover.MediaType != "image/jpeg" {
		t.Errorf("Cover().MediaType = %q, want %q", cover.MediaType, "image/jpeg")
	}
	if string(cover.Data) != "JPEG-FROM-XHTML" {
		t.Errorf("Cover().Data = %q, want %q", string(cover.Data), "JPEG-FROM-XHTML")
	}
}

func TestCover_Strategy3_GuideCover(t *testing.T) {
	opf := coverOPF("",
		`<item id="cover-page" href="cover.xhtml" media-type="application/xhtml+xml"/>
		 <item id="cover-img" href="images/cover.jpg" media-type="image/jpeg"/>`,
		`<itemref idref="cover-page"/>`,
		`<reference type="cover" title="Cover" href="cover.xhtml"/>`)

	coverXHTML := `<?xml version="1.0" encoding="UTF-8"?>
<html xmlns="http://www.w3.org/1999/xhtml">
<body><img src="images/cover.jpg" alt="Cover"/></body>
</html>`

	files := coverEPubFiles(opf, map[string]string{
		"OEBPS/cover.xhtml":      coverXHTML,
		"OEBPS/images/cover.jpg": "JPEG-COVER",
	})
	fp := buildTestEPubFile(t, files)

	book, err := Open(fp)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer book.Close()

	cover, err := book.Cover()
	if err != nil {
		t.Fatalf("Cover() error = %v", err)
	}
	if cover.Path != "OEBPS/images/cover.jpg" {
		t.Errorf("Cover().Path = %q, want %q", cover.Path, "OEBPS/images/cover.jpg")
	}
	if string(cover.Data) != "JPEG-COVER" {
		t.Errorf("Cover().Data = %q, want %q", string(cover.Data), "JPEG-COVER")
	}
}

func TestCover_Strategy4_ManifestHeuristic(t *testing.T) {
	opf := coverOPF("",
		`<item id="my-cover" href="images/front.png" media-type="image/png"/>
		 <item id="ch1" href="ch1.xhtml" media-type="application/xhtml+xml"/>`,
		`<itemref idref="ch1"/>`,
		"")
	files := coverEPubFiles(opf, map[string]string{
		"OEBPS/images/front.png": "PNG-FRONT",
	})
	fp := buildTestEPubFile(t, files)

	book, err := Open(fp)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer book.Close()

	cover, err := book.Cover()
	if err != nil {
		t.Fatalf("Cover() error = %v", err)
	}
	if cover.Path != "OEBPS/images/front.png" {
		t.Errorf("Cover().Path = %q, want %q", cover.Path, "OEBPS/images/front.png")
	}
}

func TestCover_Strategy4_HrefContainsCover(t *testing.T) {
	opf := coverOPF("",
		`<item id="img1" href="cover-image.jpg" media-type="image/jpeg"/>
		 <item id="ch1" href="ch1.xhtml" media-type="application/xhtml+xml"/>`,
		`<itemref idref="ch1"/>`,
		"")
	files := coverEPubFiles(opf, map[string]string{
		"OEBPS/cover-image.jpg": "JPG-DATA",
	})
	fp := buildTestEPubFile(t, files)

	book, err := Open(fp)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer book.Close()

	cover, err := book.Cover()
	if err != nil {
		t.Fatalf("Cover() error = %v", err)
	}
	if cover.Path != "OEBPS/cover-image.jpg" {
		t.Errorf("Cover().Path = %q, want %q", cover.Path, "OEBPS/cover-image.jpg")
	}
}

func TestCover_Strategy4_CaseInsensitiveMediaType(t *testing.T) {
	opf := coverOPF("",
		`<item id="my-cover" href="images/front.png" media-type="Image/PNG"/>
		 <item id="ch1" href="ch1.xhtml" media-type="application/xhtml+xml"/>`,
		`<itemref idref="ch1"/>`,
		"")
	files := coverEPubFiles(opf, map[string]string{
		"OEBPS/images/front.png": "PNG-FRONT",
	})
	fp := buildTestEPubFile(t, files)

	book, err := Open(fp)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer book.Close()

	cover, err := book.Cover()
	if err != nil {
		t.Fatalf("Cover() error = %v", err)
	}
	if cover.Path != "OEBPS/images/front.png" {
		t.Errorf("Cover().Path = %q, want %q", cover.Path, "OEBPS/images/front.png")
	}
}

func TestCover_Strategy5_FirstSpineImage(t *testing.T) {
	opf := coverOPF("",
		`<item id="page1" href="page1.xhtml" media-type="application/xhtml+xml"/>
		 <item id="img1" href="images/photo.jpg" media-type="image/jpeg"/>`,
		`<itemref idref="page1"/>`,
		"")

	page1 := `<html><body><h1>Title</h1><img src="images/photo.jpg"/></body></html>`

	files := coverEPubFiles(opf, map[string]string{
		"OEBPS/page1.xhtml":      page1,
		"OEBPS/images/photo.jpg": "PHOTO-DATA",
	})
	fp := buildTestEPubFile(t, files)

	book, err := Open(fp)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer book.Close()

	cover, err := book.Cover()
	if err != nil {
		t.Fatalf("Cover() error = %v", err)
	}
	if cover.Path != "OEBPS/images/photo.jpg" {
		t.Errorf("Cover().Path = %q, want %q", cover.Path, "OEBPS/images/photo.jpg")
	}
}

func TestCover_NoCover(t *testing.T) {
	opf := coverOPF("",
		`<item id="ch1" href="ch1.xhtml" media-type="application/xhtml+xml"/>`,
		`<itemref idref="ch1"/>`,
		"")

	ch1 := `<html><body><p>No images here.</p></body></html>`

	files := coverEPubFiles(opf, map[string]string{
		"OEBPS/ch1.xhtml": ch1,
	})
	fp := buildTestEPubFile(t, files)

	book, err := Open(fp)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer book.Close()

	_, err = book.Cover()
	if !errors.Is(err, ErrNoCover) {
		t.Errorf("Cover() error = %v, want ErrNoCover", err)
	}
}

func TestCover_Strategy1_TakesPriority(t *testing.T) {
	// Both strategy 1 and strategy 2 could match; strategy 1 should win.
	opf := coverOPF(
		`<meta name="cover" content="meta-cover"/>`,
		`<item id="prop-cover" href="prop-cover.png" media-type="image/png" properties="cover-image"/>
		 <item id="meta-cover" href="meta-cover.jpg" media-type="image/jpeg"/>`,
		"", "")
	files := coverEPubFiles(opf, map[string]string{
		"OEBPS/prop-cover.png": "PROP-COVER-DATA",
		"OEBPS/meta-cover.jpg": "META-COVER-DATA",
	})
	fp := buildTestEPubFile(t, files)

	book, err := Open(fp)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer book.Close()

	cover, err := book.Cover()
	if err != nil {
		t.Fatalf("Cover() error = %v", err)
	}
	if cover.Path != "OEBPS/prop-cover.png" {
		t.Errorf("Cover().Path = %q, want %q (strategy 1 should take priority)", cover.Path, "OEBPS/prop-cover.png")
	}
}

func TestCover_EmptySpine(t *testing.T) {
	opf := coverOPF("",
		`<item id="ch1" href="ch1.xhtml" media-type="application/xhtml+xml"/>`,
		"", "")
	files := coverEPubFiles(opf, nil)
	fp := buildTestEPubFile(t, files)

	book, err := Open(fp)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer book.Close()

	_, err = book.Cover()
	if !errors.Is(err, ErrNoCover) {
		t.Errorf("Cover() error = %v, want ErrNoCover", err)
	}
}

func TestFindFirstImageInHTML(t *testing.T) {
	tests := []struct {
		name     string
		html     string
		basePath string
		want     string
	}{
		{
			name:     "img tag",
			html:     `<html><body><img src="cover.jpg"/></body></html>`,
			basePath: "OEBPS/chapter.xhtml",
			want:     "OEBPS/cover.jpg",
		},
		{
			name:     "img in nested divs",
			html:     `<html><body><div><div><img src="../images/cover.png"/></div></div></body></html>`,
			basePath: "OEBPS/text/page.xhtml",
			want:     "OEBPS/images/cover.png",
		},
		{
			name:     "no img",
			html:     `<html><body><p>No images</p></body></html>`,
			basePath: "OEBPS/ch.xhtml",
			want:     "",
		},
		{
			name:     "empty src",
			html:     `<html><body><img src=""/></body></html>`,
			basePath: "OEBPS/ch.xhtml",
			want:     "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := findFirstImageInHTML([]byte(tt.html), tt.basePath)
			if got != tt.want {
				t.Errorf("findFirstImageInHTML() = %q, want %q", got, tt.want)
			}
		})
	}
}
