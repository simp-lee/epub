package epub

import (
	"archive/zip"
	"fmt"
	"io"
	"net/url"
	"path"
	"strings"
)

// maxDecompressSize is the maximum allowed decompressed size for a single ZIP entry.
// This guards against zip bomb attacks. Defaults to 256 MB.
const maxDecompressSize int64 = 256 * 1024 * 1024

// findFileInsensitive looks up a ZIP entry by path, first trying an exact match,
// then falling back to a case-insensitive comparison.
// Returns nil if no match is found.
func findFileInsensitive(zr *zip.Reader, name string) *zip.File {
	// Exact match first.
	for _, f := range zr.File {
		if f.Name == name {
			return f
		}
	}
	// Case-insensitive fallback.
	lower := strings.ToLower(name)
	for _, f := range zr.File {
		if strings.ToLower(f.Name) == lower {
			return f
		}
	}
	return nil
}

// resolveRelativePath resolves href relative to the directory of basePath.
// Both basePath and href are ZIP-internal paths (forward-slash separated).
// The result is cleaned and validated to stay within the ZIP root.
// If the resolved path escapes root or is absolute, an empty string is returned.
func resolveRelativePath(basePath, href string) string {
	href = strings.TrimSpace(href)
	if strings.HasPrefix(href, "/") {
		return ""
	}
	if decoded, err := url.PathUnescape(href); err == nil {
		href = decoded
	}
	dir := path.Dir(basePath)
	joined := path.Join(dir, href)
	cleaned := path.Clean(joined)
	if !isSafePath(cleaned) {
		return ""
	}
	return cleaned
}

// isSafePath checks whether p is a safe ZIP-internal path that does not
// escape the archive root via path traversal (e.g., "../../../etc/passwd").
func isSafePath(p string) bool {
	cleaned := path.Clean(p)
	if strings.HasPrefix(cleaned, "/") {
		return false
	}
	if cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return false
	}
	return true
}

// stripBOM removes a leading UTF-8 BOM (0xEF 0xBB 0xBF) from data, if present.
func stripBOM(data []byte) []byte {
	if len(data) >= 3 && data[0] == 0xEF && data[1] == 0xBB && data[2] == 0xBF {
		return data[3:]
	}
	return data
}

// readZipFile reads the full contents of a ZIP entry.
// It enforces maxDecompressSize to guard against zip bombs and validates
// that the entry path is safe (no path traversal).
func readZipFile(f *zip.File) ([]byte, error) {
	return readZipFileWithLimit(f, maxDecompressSize)
}

// readZipFileWithLimit is the implementation of readZipFile with a configurable
// size limit. It is separated to allow tests to use a smaller limit.
func readZipFileWithLimit(f *zip.File, limit int64) ([]byte, error) {
	if !isSafePath(f.Name) {
		return nil, fmt.Errorf("epub: unsafe zip entry path: %s", f.Name)
	}

	if f.UncompressedSize64 > uint64(limit) {
		return nil, fmt.Errorf("epub: zip entry %s too large: %d bytes (max %d)", f.Name, f.UncompressedSize64, limit)
	}

	rc, err := f.Open()
	if err != nil {
		return nil, fmt.Errorf("epub: open zip entry %s: %w", f.Name, err)
	}
	defer rc.Close()

	// Read up to limit+1 to detect if the actual decompressed data
	// exceeds the limit (the declared size might be wrong/forged).
	lr := io.LimitReader(rc, limit+1)
	data, err := io.ReadAll(lr)
	if err != nil {
		return nil, fmt.Errorf("epub: read zip entry %s: %w", f.Name, err)
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("epub: zip entry %s decompressed size exceeds limit (%d bytes)", f.Name, limit)
	}

	return data, nil
}
