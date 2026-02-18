package epub

import (
	"archive/zip"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"
)

// buildTestZip creates an in-memory ZIP archive from the provided files map
// (path → content) and returns a *zip.Reader over the resulting bytes.
// It calls t.Fatal on any error.
func buildTestZip(t *testing.T, files map[string]string) *zip.Reader {
	t.Helper()
	buf := new(bytes.Buffer)
	zw := zip.NewWriter(buf)
	for name, content := range files {
		fw, err := zw.Create(name)
		if err != nil {
			t.Fatalf("buildTestZip: create %s: %v", name, err)
		}
		if _, err := io.WriteString(fw, content); err != nil {
			t.Fatalf("buildTestZip: write %s: %v", name, err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("buildTestZip: close writer: %v", err)
	}

	data := buf.Bytes()
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("buildTestZip: open reader: %v", err)
	}
	return r
}

// buildTestEPub creates an in-memory ZIP archive intended to simulate an ePub.
// The files map uses ZIP-internal paths as keys and file content as values.
// It returns a *zip.Reader for use in unit tests.
func buildTestEPub(t *testing.T, files map[string]string) *zip.Reader {
	t.Helper()
	return buildTestZip(t, files)
}

// buildTestEPubFile writes an ePub (ZIP) archive to a temporary file and returns
// the file path. This variant is useful for testing Open() which requires a file path.
func buildTestEPubFile(t *testing.T, files map[string]string) string {
	t.Helper()
	buf := new(bytes.Buffer)
	zw := zip.NewWriter(buf)
	// Write mimetype first if present (ePub spec requires it as first entry).
	if mt, ok := files["mimetype"]; ok {
		fw, err := zw.Create("mimetype")
		if err != nil {
			t.Fatalf("buildTestEPubFile: create mimetype: %v", err)
		}
		if _, err := io.WriteString(fw, mt); err != nil {
			t.Fatalf("buildTestEPubFile: write mimetype: %v", err)
		}
	}
	for name, content := range files {
		if name == "mimetype" {
			continue
		}
		fw, err := zw.Create(name)
		if err != nil {
			t.Fatalf("buildTestEPubFile: create %s: %v", name, err)
		}
		if _, err := io.WriteString(fw, content); err != nil {
			t.Fatalf("buildTestEPubFile: write %s: %v", name, err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("buildTestEPubFile: close writer: %v", err)
	}

	dir := t.TempDir()
	fp := filepath.Join(dir, "test.epub")
	if err := os.WriteFile(fp, buf.Bytes(), 0644); err != nil {
		t.Fatalf("buildTestEPubFile: write file: %v", err)
	}
	return fp
}
