package epub

import (
	"archive/zip"
	"encoding/xml"
	"strings"
)

// encryptionFilePath is the standard path for the encryption descriptor.
const encryptionFilePath = "META-INF/encryption.xml"

// sinfFilePath is the path that indicates Apple FairPlay DRM.
const sinfFilePath = "META-INF/sinf.xml"

// Font obfuscation algorithm URIs – these do NOT constitute DRM.
var fontObfuscationAlgorithms = map[string]bool{
	"http://www.idpf.org/2008/embedding": true, // IDPF font obfuscation
	"http://ns.adobe.com/pdf/enc#RC":     true, // Adobe font obfuscation
}

// Known DRM namespace prefixes found in KeyInfo child elements or algorithm URIs.
var drmSignatures = []string{
	"http://ns.adobe.com/adept",      // Adobe ADEPT
	"http://readium.org/2014/01/lcp", // Readium LCP
}

// XML structures for parsing encryption.xml.

type xmlEncryption struct {
	XMLName       xml.Name           `xml:"encryption"`
	EncryptedData []xmlEncryptedData `xml:"EncryptedData"`
}

type xmlEncryptedData struct {
	EncryptionMethod xmlEncryptionMethod `xml:"EncryptionMethod"`
	KeyInfo          xmlKeyInfo          `xml:"KeyInfo"`
}

type xmlEncryptionMethod struct {
	Algorithm string `xml:"Algorithm,attr"`
}

type xmlKeyInfo struct {
	InnerXML string `xml:",innerxml"`
}

// checkDRM parses META-INF/encryption.xml (if present) and determines whether
// the ePub is DRM-protected or merely uses font obfuscation.
//
// Returns:
//   - (false, nil)            – no encryption.xml found or it's empty
//   - (true,  nil)            – only font obfuscation entries detected
//   - (false, ErrDRMProtected) – real DRM encryption detected
func checkDRM(zr *zip.Reader) (fontObfuscation bool, err error) {
	// Check for Apple FairPlay indicator first.
	if findFileInsensitive(zr, sinfFilePath) != nil {
		return false, ErrDRMProtected
	}

	f := findFileInsensitive(zr, encryptionFilePath)
	if f == nil {
		return false, nil
	}

	data, err := readZipFile(f)
	if err != nil {
		return false, err
	}
	data = stripBOM(data)

	var enc xmlEncryption
	if err := xml.Unmarshal(data, &enc); err != nil {
		// If we can't parse it, treat conservatively as potential DRM.
		return false, ErrDRMProtected
	}

	if len(enc.EncryptedData) == 0 {
		return false, nil
	}

	for _, ed := range enc.EncryptedData {
		algo := ed.EncryptionMethod.Algorithm

		// Check if this entry is font obfuscation.
		if fontObfuscationAlgorithms[algo] {
			fontObfuscation = true
			continue
		}

		// Check algorithm URI for known DRM signatures.
		if isDRMSignature(algo) {
			return false, ErrDRMProtected
		}

		// Check KeyInfo content for known DRM signatures.
		if isDRMSignature(ed.KeyInfo.InnerXML) {
			return false, ErrDRMProtected
		}

		// Any EncryptedData that is NOT font obfuscation is treated as DRM.
		return false, ErrDRMProtected
	}

	return fontObfuscation, nil
}

// isDRMSignature checks whether s contains any known DRM namespace or identifier.
func isDRMSignature(s string) bool {
	for _, sig := range drmSignatures {
		if strings.Contains(s, sig) {
			return true
		}
	}
	return false
}
