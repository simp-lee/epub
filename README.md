# epub

A pure-Go library for reading and parsing ePub 2 and ePub 3 files.

[![Go Reference](https://pkg.go.dev/badge/github.com/simp-lee/epub.svg)](https://pkg.go.dev/github.com/simp-lee/epub)

## Features

- ePub 2 and ePub 3 support
- Dublin Core metadata extraction (titles, authors, identifiers, language, etc.)
- Table of contents parsing (NCX for ePub 2, Nav document for ePub 3)
- Landmarks extraction (ePub 3)
- Spine-ordered chapter access with lazy content loading
- Plain text, raw XHTML, and sanitised body HTML output
- Cover image detection via multiple strategies
- Project Gutenberg license page detection
- DRM detection (Adobe ADEPT, Apple FairPlay, Readium LCP)
- Font obfuscation awareness
- ZIP bomb protection

## Installation

```
go get github.com/simp-lee/epub
```

## Quick Start

```go
package main

import (
    "fmt"
    "log"

    "github.com/simp-lee/epub"
)

func main() {
    book, err := epub.Open("book.epub")
    if err != nil {
        log.Fatal(err)
    }
    defer book.Close()

    // Metadata
    md := book.Metadata()
    if len(md.Titles) > 0 {
        fmt.Println("Title:", md.Titles[0])
    }
    for _, a := range md.Authors {
        fmt.Println("Author:", a.Name)
    }

    // Table of Contents
    for _, item := range book.TOC() {
        fmt.Printf("  %s → %s\n", item.Title, item.Href)
    }

    // Chapters
    for _, ch := range book.Chapters() {
        text, err := ch.TextContent()
        if err != nil {
            continue
        }
        fmt.Printf("  [%s] %d chars\n", ch.Title, len(text))
    }

    // Cover image
    cover, err := book.Cover()
    if err == nil {
        fmt.Printf("Cover: %s (%d bytes)\n", cover.MediaType, len(cover.Data))
    }
}
```

## API Overview

### Opening

| Function | Description |
|---|---|
| `Open(path)` | Open an ePub file by path |
| `NewReader(r, size)` | Open from an `io.ReaderAt` |

### Book Methods

| Method | Description |
|---|---|
| `Close()` | Release resources |
| `Metadata()` | Dublin Core metadata |
| `TOC()` | Table of contents tree |
| `Landmarks()` | ePub 3 landmarks |
| `Chapters()` | Spine-ordered chapters |
| `ContentChapters()` | Chapters excluding license pages |
| `Cover()` | Detect and return cover image |
| `ReadFile(name)` | Read any file from the archive |
| `HasTOC()` | Whether a TOC is present |
| `Warnings()` | Non-fatal parsing warnings |

### Chapter Methods

| Method | Description |
|---|---|
| `RawContent()` | Raw XHTML bytes |
| `TextContent()` | Extracted plain text |
| `BodyHTML()` | Sanitised `<body>` inner HTML |

### Error Handling

The package provides sentinel errors for common failure cases:

```go
errors.Is(err, epub.ErrDRMProtected)   // DRM-encrypted file
errors.Is(err, epub.ErrInvalidEPub)    // Invalid ePub structure
errors.Is(err, epub.ErrInvalidChapter) // Invalid chapter handle (zero-value)
errors.Is(err, epub.ErrNoCover)        // No cover image found
errors.Is(err, epub.ErrFileNotFound)   // File not in archive
```

When a book has no NCX/nav table of contents, `TOC()` returns an empty slice.

## License

MIT
