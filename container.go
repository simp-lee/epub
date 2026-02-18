package epub

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"strings"
)

// containerXML models the META-INF/container.xml file used to locate the OPF.
type containerXML struct {
	XMLName   xml.Name   `xml:"container"`
	RootFiles []rootFile `xml:"rootfiles>rootfile"`
}

// rootFile represents a single <rootfile> element inside container.xml.
type rootFile struct {
	FullPath  string `xml:"full-path,attr"`
	MediaType string `xml:"media-type,attr"`
}

// containerPath is the well-known location of container.xml in an ePub archive.
const containerPath = "META-INF/container.xml"

// parseContainer locates and parses the OPF path from the ePub ZIP archive.
//
// It first tries META-INF/container.xml (case-insensitive lookup). If the file
// is missing, it falls back to scanning all ZIP entries for a ".opf" file.
// Returns a wrapped ErrInvalidEPub if no OPF path can be determined.
func parseContainer(zr *zip.Reader) (string, error) {
	// Try container.xml first.
	if f := findFileInsensitive(zr, containerPath); f != nil {
		return parseContainerXML(f)
	}

	// Fallback: scan for .opf files.
	return fallbackFindOPF(zr)
}

// parseContainerXML reads and decodes a container.xml ZIP entry, returning
// the full-path of the first rootfile.
func parseContainerXML(f *zip.File) (string, error) {
	data, err := readZipFile(f)
	if err != nil {
		return "", fmt.Errorf("epub: read container.xml: %w", err)
	}

	data = stripBOM(data)

	var c containerXML
	if err := xml.Unmarshal(data, &c); err != nil {
		return "", fmt.Errorf("epub: parse container.xml: %w", err)
	}

	if len(c.RootFiles) == 0 {
		return "", fmt.Errorf("epub: container.xml has no rootfile entries: %w", ErrInvalidEPub)
	}

	var fallbackPath string
	for _, rf := range c.RootFiles {
		fullPath := strings.TrimSpace(rf.FullPath)
		if fullPath == "" {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(rf.MediaType), "application/oebps-package+xml") {
			return fullPath, nil
		}
		if fallbackPath == "" {
			fallbackPath = fullPath
		}
	}

	if fallbackPath == "" {
		return "", fmt.Errorf("epub: container.xml rootfile has empty full-path: %w", ErrInvalidEPub)
	}

	return fallbackPath, nil
}

// fallbackFindOPF scans the ZIP entries for the first file ending in ".opf"
// (case-insensitive). Returns ErrInvalidEPub if none is found.
func fallbackFindOPF(zr *zip.Reader) (string, error) {
	for _, f := range zr.File {
		if strings.HasSuffix(strings.ToLower(f.Name), ".opf") {
			return f.Name, nil
		}
	}
	return "", fmt.Errorf("epub: no OPF file found in archive: %w", ErrInvalidEPub)
}
