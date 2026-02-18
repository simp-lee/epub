package epub

import (
	"errors"
	"testing"
)

// validContainerXML is a well-formed META-INF/container.xml pointing to an OPF.
const validContainerXML = `<?xml version="1.0" encoding="UTF-8"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
  <rootfiles>
    <rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/>
  </rootfiles>
</container>`

func TestParseContainer_Normal(t *testing.T) {
	zr := buildTestZip(t, map[string]string{
		"META-INF/container.xml": validContainerXML,
		"OEBPS/content.opf":      `<package/>`,
	})

	opfPath, err := parseContainer(zr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opfPath != "OEBPS/content.opf" {
		t.Errorf("opfPath = %q, want %q", opfPath, "OEBPS/content.opf")
	}
}

func TestParseContainer_CaseInsensitive(t *testing.T) {
	zr := buildTestZip(t, map[string]string{
		"meta-inf/container.xml": validContainerXML,
		"OEBPS/content.opf":      `<package/>`,
	})

	opfPath, err := parseContainer(zr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opfPath != "OEBPS/content.opf" {
		t.Errorf("opfPath = %q, want %q", opfPath, "OEBPS/content.opf")
	}
}

func TestParseContainer_WithBOM(t *testing.T) {
	bomContainer := "\xEF\xBB\xBF" + validContainerXML
	zr := buildTestZip(t, map[string]string{
		"META-INF/container.xml": bomContainer,
		"OEBPS/content.opf":      `<package/>`,
	})

	opfPath, err := parseContainer(zr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opfPath != "OEBPS/content.opf" {
		t.Errorf("opfPath = %q, want %q", opfPath, "OEBPS/content.opf")
	}
}

func TestParseContainer_FallbackOPF(t *testing.T) {
	// No container.xml; should find the .opf file by scanning.
	zr := buildTestZip(t, map[string]string{
		"content.opf": `<package/>`,
	})

	opfPath, err := parseContainer(zr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opfPath != "content.opf" {
		t.Errorf("opfPath = %q, want %q", opfPath, "content.opf")
	}
}

func TestParseContainer_FallbackOPF_CaseInsensitive(t *testing.T) {
	zr := buildTestZip(t, map[string]string{
		"OEBPS/Book.OPF": `<package/>`,
	})

	opfPath, err := parseContainer(zr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opfPath != "OEBPS/Book.OPF" {
		t.Errorf("opfPath = %q, want %q", opfPath, "OEBPS/Book.OPF")
	}
}

func TestParseContainer_NoOPF(t *testing.T) {
	// No container.xml and no .opf file → error.
	zr := buildTestZip(t, map[string]string{
		"readme.txt": "hello",
	})

	_, err := parseContainer(zr)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrInvalidEPub) {
		t.Errorf("error = %v, want wrapped ErrInvalidEPub", err)
	}
}

func TestParseContainer_EmptyRootfiles(t *testing.T) {
	emptyContainer := `<?xml version="1.0" encoding="UTF-8"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
  <rootfiles/>
</container>`

	zr := buildTestZip(t, map[string]string{
		"META-INF/container.xml": emptyContainer,
	})

	_, err := parseContainer(zr)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrInvalidEPub) {
		t.Errorf("error = %v, want wrapped ErrInvalidEPub", err)
	}
}

func TestParseContainer_EmptyFullPath(t *testing.T) {
	badContainer := `<?xml version="1.0" encoding="UTF-8"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
  <rootfiles>
    <rootfile full-path="" media-type="application/oebps-package+xml"/>
  </rootfiles>
</container>`

	zr := buildTestZip(t, map[string]string{
		"META-INF/container.xml": badContainer,
	})

	_, err := parseContainer(zr)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrInvalidEPub) {
		t.Errorf("error = %v, want wrapped ErrInvalidEPub", err)
	}
}

func TestParseContainer_PrefersRootfileByMediaType(t *testing.T) {
	multiRootContainer := `<?xml version="1.0" encoding="UTF-8"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
  <rootfiles>
    <rootfile full-path="" media-type="application/oebps-package+xml"/>
    <rootfile full-path="OPS/preview.opf" media-type="application/x-preview+xml"/>
    <rootfile full-path="OPS/book.opf" media-type="application/oebps-package+xml"/>
  </rootfiles>
</container>`

	zr := buildTestZip(t, map[string]string{
		"META-INF/container.xml": multiRootContainer,
	})

	opfPath, err := parseContainer(zr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opfPath != "OPS/book.opf" {
		t.Errorf("opfPath = %q, want %q", opfPath, "OPS/book.opf")
	}
}

func TestParseContainer_FallbackToFirstNonEmptyRootfile(t *testing.T) {
	multiRootContainer := `<?xml version="1.0" encoding="UTF-8"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
  <rootfiles>
    <rootfile full-path="" media-type="application/x-other+xml"/>
    <rootfile full-path="OPS/first-non-empty.opf" media-type="application/x-other+xml"/>
    <rootfile full-path="OPS/second-non-empty.opf" media-type="application/x-another+xml"/>
  </rootfiles>
</container>`

	zr := buildTestZip(t, map[string]string{
		"META-INF/container.xml": multiRootContainer,
	})

	opfPath, err := parseContainer(zr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opfPath != "OPS/first-non-empty.opf" {
		t.Errorf("opfPath = %q, want %q", opfPath, "OPS/first-non-empty.opf")
	}
}
