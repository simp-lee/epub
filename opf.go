package epub

import (
	"encoding/xml"
	"fmt"
)

// opfPackage represents the root <package> element of an OPF file.
type opfPackage struct {
	XMLName          xml.Name    `xml:"package"`
	Version          string      `xml:"version,attr"`
	UniqueIdentifier string      `xml:"unique-identifier,attr"`
	Metadata         opfMetadata `xml:"metadata"`
	Manifest         opfManifest `xml:"manifest"`
	Spine            opfSpine    `xml:"spine"`
	Guide            opfGuide    `xml:"guide"`
}

// opfMetadata holds the raw metadata elements from the OPF file.
type opfMetadata struct {
	Titles       []opfDCElement `xml:"http://purl.org/dc/elements/1.1/ title"`
	Creators     []opfDCElement `xml:"http://purl.org/dc/elements/1.1/ creator"`
	Languages    []opfDCElement `xml:"http://purl.org/dc/elements/1.1/ language"`
	Identifiers  []opfDCElement `xml:"http://purl.org/dc/elements/1.1/ identifier"`
	Publishers   []opfDCElement `xml:"http://purl.org/dc/elements/1.1/ publisher"`
	Dates        []opfDCElement `xml:"http://purl.org/dc/elements/1.1/ date"`
	Descriptions []opfDCElement `xml:"http://purl.org/dc/elements/1.1/ description"`
	Subjects     []opfDCElement `xml:"http://purl.org/dc/elements/1.1/ subject"`
	Rights       []opfDCElement `xml:"http://purl.org/dc/elements/1.1/ rights"`
	Sources      []opfDCElement `xml:"http://purl.org/dc/elements/1.1/ source"`
	Metas        []opfMeta      `xml:"meta"`
}

// opfDCElement holds a Dublin Core element with optional OPF attributes.
// ePub 2 uses opf:file-as, opf:role, opf:scheme attributes directly.
// ePub 3 uses <meta refines="..."> elements to express the same information.
type opfDCElement struct {
	Value  string `xml:",chardata"`
	ID     string `xml:"id,attr"`
	FileAs string `xml:"file-as,attr"`
	Role   string `xml:"role,attr"`
	Scheme string `xml:"scheme,attr"`
}

// opfMeta represents a <meta> element in the OPF metadata.
// ePub 2: <meta name="..." content="..."/>
// ePub 3: <meta property="..." refines="..." scheme="...">value</meta>
type opfMeta struct {
	// ePub 2 attributes.
	Name    string `xml:"name,attr"`
	Content string `xml:"content,attr"`

	// ePub 3 attributes.
	Property string `xml:"property,attr"`
	Refines  string `xml:"refines,attr"`
	Scheme   string `xml:"scheme,attr"`

	// ePub 3 text content.
	Value string `xml:",chardata"`
}

// opfManifest wraps the <manifest> element.
type opfManifest struct {
	Items []opfManifestItem `xml:"item"`
}

// opfManifestItem represents a single <item> in the manifest.
type opfManifestItem struct {
	ID         string `xml:"id,attr"`
	Href       string `xml:"href,attr"`
	MediaType  string `xml:"media-type,attr"`
	Properties string `xml:"properties,attr"`
}

// opfSpine wraps the <spine> element.
type opfSpine struct {
	Toc      string            `xml:"toc,attr"`
	ItemRefs []opfSpineItemRef `xml:"itemref"`
}

// opfSpineItemRef represents a single <itemref> in the spine.
type opfSpineItemRef struct {
	IDRef  string `xml:"idref,attr"`
	Linear string `xml:"linear,attr"`
}

// opfGuide wraps the <guide> element.
type opfGuide struct {
	References []opfGuideReference `xml:"reference"`
}

// opfGuideReference represents a single <reference> in the guide.
type opfGuideReference struct {
	Type  string `xml:"type,attr"`
	Title string `xml:"title,attr"`
	Href  string `xml:"href,attr"`
}

// guideReference is the processed representation of a guide reference entry.
type guideReference struct {
	Type  string
	Title string
	Href  string
}

// parseOPF parses the OPF file content and returns the parsed package structure.
func parseOPF(data []byte) (*opfPackage, error) {
	data = preprocessHTMLEntities(data)
	data = stripBOM(data)

	var pkg opfPackage
	if err := xml.Unmarshal(data, &pkg); err != nil {
		return nil, fmt.Errorf("epub: parse OPF: %w", err)
	}

	if pkg.Version == "" {
		// Default to 2.0 if version attribute is missing.
		pkg.Version = "2.0"
	}

	return &pkg, nil
}

// buildManifestMaps creates lookup maps from the parsed OPF manifest.
// Returns maps keyed by ID and by Href for fast lookups.
func buildManifestMaps(manifest opfManifest) (byID, byHref map[string]*manifestItem) {
	byID = make(map[string]*manifestItem, len(manifest.Items))
	byHref = make(map[string]*manifestItem, len(manifest.Items))

	for _, item := range manifest.Items {
		mi := &manifestItem{
			ID:         item.ID,
			Href:       item.Href,
			MediaType:  item.MediaType,
			Properties: item.Properties,
		}
		byID[item.ID] = mi
		byHref[item.Href] = mi
	}

	return byID, byHref
}

// buildSpine creates a slice of spineItem from the parsed OPF spine,
// resolving manifest references for href and media-type.
func buildSpine(spine opfSpine, manifestByID map[string]*manifestItem) []spineItem {
	items := make([]spineItem, 0, len(spine.ItemRefs))

	for _, ref := range spine.ItemRefs {
		si := spineItem{
			IDRef:  ref.IDRef,
			Linear: ref.Linear != "no",
		}
		if mi, ok := manifestByID[ref.IDRef]; ok {
			si.ID = mi.ID
			si.Href = mi.Href
			si.MediaType = mi.MediaType
		}
		items = append(items, si)
	}

	return items
}

// buildGuide creates a slice of guideReference from the parsed OPF guide.
func buildGuide(guide opfGuide) []guideReference {
	refs := make([]guideReference, 0, len(guide.References))
	for _, r := range guide.References {
		refs = append(refs, guideReference{
			Type:  r.Type,
			Title: r.Title,
			Href:  r.Href,
		})
	}
	return refs
}
