package epub

import (
	"bytes"
	"slices"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// Cover detects and returns the cover image using multiple strategies.
// Strategies are tried in priority order:
//  1. ePub 3 manifest item with properties="cover-image"
//  2. ePub 2 <meta name="cover" content="ID"/> → manifest lookup
//  3. <guide> reference type="cover" → parse XHTML for first <img>
//  4. Manifest item whose ID or href contains "cover" with image/* media-type
//  5. First spine item's XHTML → first <img>
//
// Returns ErrNoCover if no strategy succeeds.
func (b *Book) Cover() (CoverImage, error) {
	// Strategy 1: ePub 3 cover-image property.
	if item := b.coverFromManifestProperties(); item != nil {
		return b.loadCoverImage(item)
	}

	// Strategy 2: ePub 2 meta name="cover".
	if item := b.coverFromMetaCover(); item != nil {
		return b.loadCoverImage(item)
	}

	// Strategy 3: guide reference type="cover" → parse XHTML.
	if item := b.coverFromGuide(); item != nil {
		return b.loadCoverImage(item)
	}

	// Strategy 4: manifest item with "cover" in ID/href and image media-type.
	if item := b.coverFromManifestHeuristic(); item != nil {
		return b.loadCoverImage(item)
	}

	// Strategy 5: first spine XHTML → first <img>.
	if item := b.coverFromFirstSpine(); item != nil {
		return b.loadCoverImage(item)
	}

	return CoverImage{}, ErrNoCover
}

// coverFromManifestProperties searches the manifest for an item whose
// Properties field contains "cover-image" (ePub 3).
// It iterates over the OPF manifest items slice to preserve document order.
func (b *Book) coverFromManifestProperties() *manifestItem {
	for _, raw := range b.opf.Manifest.Items {
		item, ok := b.manifestByID[raw.ID]
		if !ok {
			continue
		}
		if slices.Contains(strings.Fields(item.Properties), "cover-image") {
			return item
		}
	}
	return nil
}

// coverFromMetaCover looks for <meta name="cover" content="ID"/> in the OPF
// metadata and resolves the ID through the manifest (ePub 2).
// If the resolved item is an image, it is returned directly. Otherwise it is
// treated as an XHTML cover page and the first <img> is extracted.
func (b *Book) coverFromMetaCover() *manifestItem {
	for _, m := range b.opf.Metadata.Metas {
		if strings.EqualFold(m.Name, "cover") && m.Content != "" {
			item, ok := b.manifestByID[m.Content]
			if !ok {
				continue
			}
			if isImageMediaType(item.MediaType) {
				return item
			}
			// Non-image item — try parsing as XHTML cover page.
			xhtmlPath := b.resolveOPFPath(item.Href)
			data, err := b.ReadFile(xhtmlPath)
			if err != nil {
				continue
			}
			imgPath := findFirstImageInHTML(data, xhtmlPath)
			if imgPath != "" {
				if imgItem := b.resolveImageManifestItem(imgPath); imgItem != nil {
					return imgItem
				}
			}
		}
	}
	return nil
}

// coverFromGuide searches the <guide> for a reference with type="cover",
// reads that XHTML file, and extracts the first <img> src to resolve a
// manifest image item.
func (b *Book) coverFromGuide() *manifestItem {
	for _, ref := range b.guide {
		if !strings.EqualFold(ref.Type, "cover") {
			continue
		}
		// Strip fragment from href.
		href := ref.Href
		if idx := strings.IndexByte(href, '#'); idx >= 0 {
			href = href[:idx]
		}

		// Resolve the XHTML path relative to the OPF directory.
		xhtmlPath := b.resolveOPFPath(href)

		data, err := b.ReadFile(xhtmlPath)
		if err != nil {
			continue
		}

		imgPath := findFirstImageInHTML(data, xhtmlPath)
		if imgPath == "" {
			continue
		}

		// Look up the image in the manifest by href (relative to OPF dir).
		item := b.resolveImageManifestItem(imgPath)
		if item != nil {
			return item
		}
	}
	return nil
}

// coverFromManifestHeuristic searches all manifest items for one whose ID or
// href contains "cover" (case-insensitive) and has an image/* media-type.
// It iterates over the OPF manifest items slice to preserve document order.
func (b *Book) coverFromManifestHeuristic() *manifestItem {
	for _, raw := range b.opf.Manifest.Items {
		item, ok := b.manifestByID[raw.ID]
		if !ok || !isImageMediaType(item.MediaType) {
			continue
		}
		if containsFold(item.ID, "cover") || containsFold(item.Href, "cover") {
			return item
		}
	}
	return nil
}

// coverFromFirstSpine reads the first spine item's XHTML content and extracts
// the first <img> src to resolve a manifest image item.
func (b *Book) coverFromFirstSpine() *manifestItem {
	if len(b.spine) == 0 {
		return nil
	}
	first := b.spine[0]
	if first.Href == "" {
		return nil
	}

	xhtmlPath := b.resolveOPFPath(first.Href)
	data, err := b.ReadFile(xhtmlPath)
	if err != nil {
		return nil
	}

	imgPath := findFirstImageInHTML(data, xhtmlPath)
	if imgPath == "" {
		return nil
	}

	return b.resolveImageManifestItem(imgPath)
}

// loadCoverImage reads the image data from the ZIP archive and constructs a
// CoverImage. The path stored is the full ZIP-internal path.
func (b *Book) loadCoverImage(item *manifestItem) (CoverImage, error) {
	imgPath := b.resolveOPFPath(item.Href)
	data, err := b.ReadFile(imgPath)
	if err != nil {
		return CoverImage{}, err
	}
	return CoverImage{
		Path:      imgPath,
		MediaType: item.MediaType,
		Data:      data,
	}, nil
}

// resolveImageManifestItem resolves an absolute ZIP-internal image path to a
// manifestItem. It tries matching by making the path relative to opfDir,
// then falls back to iterating the manifest with case-insensitive comparison.
func (b *Book) resolveImageManifestItem(absPath string) *manifestItem {
	// Make relative to opfDir for manifest lookup.
	rel := absPath
	if b.opfDir != "." {
		prefix := b.opfDir + "/"
		if strings.HasPrefix(absPath, prefix) {
			rel = absPath[len(prefix):]
		}
	}

	if item, ok := b.manifestByHref[rel]; ok && isImageMediaType(item.MediaType) {
		return item
	}
	// Also try the absolute path itself as href.
	if item, ok := b.manifestByHref[absPath]; ok && isImageMediaType(item.MediaType) {
		return item
	}

	// Fallback: iterate manifest with path comparison and case-insensitive match.
	lowerAbs := strings.ToLower(absPath)
	lowerRel := strings.ToLower(rel)
	for _, item := range b.manifestByHref {
		if !isImageMediaType(item.MediaType) {
			continue
		}
		itemHrefLower := strings.ToLower(item.Href)
		if itemHrefLower == lowerRel || itemHrefLower == lowerAbs {
			return item
		}
		// Try matching the full ZIP path of the manifest item.
		if strings.EqualFold(b.resolveOPFPath(item.Href), absPath) {
			return item
		}
	}
	return nil
}

// findFirstImageInHTML parses HTML data and returns the resolved ZIP-internal
// path of the first <img> element's src attribute. If no image is found,
// returns an empty string. basePath is the ZIP-internal path of the HTML file,
// used to resolve relative image paths.
func findFirstImageInHTML(htmlData []byte, basePath string) string {
	tokenizer := html.NewTokenizer(bytes.NewReader(htmlData))
	for {
		tt := tokenizer.Next()
		switch tt {
		case html.ErrorToken:
			return ""
		case html.StartTagToken, html.SelfClosingTagToken:
			tn, hasAttr := tokenizer.TagName()
			a := atom.Lookup(tn)
			if a == atom.Img && hasAttr {
				for {
					key, val, more := tokenizer.TagAttr()
					if string(key) == "src" && string(val) != "" {
						return resolveRelativePath(basePath, string(val))
					}
					if !more {
						break
					}
				}
			}
			// Also check SVG <image> element with xlink:href or href.
			if a == atom.Image && hasAttr {
				for {
					key, val, more := tokenizer.TagAttr()
					k := string(key)
					if (k == "href" || k == "xlink:href") && string(val) != "" {
						return resolveRelativePath(basePath, string(val))
					}
					if !more {
						break
					}
				}
			}
		}
	}
}

// isImageMediaType returns true if the media type starts with "image/".
func isImageMediaType(mediaType string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(mediaType)), "image/")
}

// containsFold reports whether s contains substr, case-insensitively.
func containsFold(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}
