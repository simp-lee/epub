package epub

import "errors"

// Sentinel errors returned by the epub package.
var (
	// ErrDRMProtected indicates the ePub file is protected by DRM
	// (e.g., Adobe ADEPT, Apple FairPlay, Readium LCP) and cannot be read.
	ErrDRMProtected = errors.New("epub: file is DRM protected")

	// ErrInvalidEPub indicates the file is not a valid ePub
	// (e.g., missing container.xml and no .opf file found).
	ErrInvalidEPub = errors.New("epub: invalid ePub file")

	// ErrInvalidChapter indicates a Chapter handle is invalid
	// (for example, a zero-value Chapter without an associated Book).
	ErrInvalidChapter = errors.New("epub: invalid chapter handle")

	// ErrFileNotFound indicates the requested file does not exist
	// in the ePub archive.
	ErrFileNotFound = errors.New("epub: file not found in archive")

	// ErrNoCover indicates no cover image could be detected
	// using any of the supported strategies.
	ErrNoCover = errors.New("epub: no cover image found")
)
