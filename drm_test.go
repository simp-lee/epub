package epub

import (
	"testing"
)

func TestCheckDRM(t *testing.T) {
	tests := []struct {
		name              string
		files             map[string]string
		wantFontObfuscate bool
		wantErr           error
	}{
		{
			name: "no encryption.xml",
			files: map[string]string{
				"mimetype":               "application/epub+zip",
				"META-INF/container.xml": `<?xml version="1.0"?><container/>`,
				"OEBPS/content.opf":      `<package/>`,
			},
			wantFontObfuscate: false,
			wantErr:           nil,
		},
		{
			name: "font obfuscation only IDPF",
			files: map[string]string{
				"mimetype": "application/epub+zip",
				"META-INF/encryption.xml": `<?xml version="1.0" encoding="UTF-8"?>
<encryption xmlns="urn:oasis:names:tc:opendocument:xmlns:container"
            xmlns:enc="http://www.w3.org/2001/04/xmlenc#">
  <enc:EncryptedData>
    <enc:EncryptionMethod Algorithm="http://www.idpf.org/2008/embedding"/>
    <enc:CipherData>
      <enc:CipherReference URI="OEBPS/fonts/myfont.otf"/>
    </enc:CipherData>
  </enc:EncryptedData>
</encryption>`,
			},
			wantFontObfuscate: true,
			wantErr:           nil,
		},
		{
			name: "font obfuscation only Adobe",
			files: map[string]string{
				"mimetype": "application/epub+zip",
				"META-INF/encryption.xml": `<?xml version="1.0" encoding="UTF-8"?>
<encryption xmlns="urn:oasis:names:tc:opendocument:xmlns:container"
            xmlns:enc="http://www.w3.org/2001/04/xmlenc#">
  <enc:EncryptedData>
    <enc:EncryptionMethod Algorithm="http://ns.adobe.com/pdf/enc#RC"/>
    <enc:CipherData>
      <enc:CipherReference URI="OEBPS/fonts/myfont.ttf"/>
    </enc:CipherData>
  </enc:EncryptedData>
</encryption>`,
			},
			wantFontObfuscate: true,
			wantErr:           nil,
		},
		{
			name: "mixed font obfuscation algorithms",
			files: map[string]string{
				"mimetype": "application/epub+zip",
				"META-INF/encryption.xml": `<?xml version="1.0" encoding="UTF-8"?>
<encryption xmlns="urn:oasis:names:tc:opendocument:xmlns:container"
            xmlns:enc="http://www.w3.org/2001/04/xmlenc#">
  <enc:EncryptedData>
    <enc:EncryptionMethod Algorithm="http://www.idpf.org/2008/embedding"/>
    <enc:CipherData>
      <enc:CipherReference URI="OEBPS/fonts/font1.otf"/>
    </enc:CipherData>
  </enc:EncryptedData>
  <enc:EncryptedData>
    <enc:EncryptionMethod Algorithm="http://ns.adobe.com/pdf/enc#RC"/>
    <enc:CipherData>
      <enc:CipherReference URI="OEBPS/fonts/font2.ttf"/>
    </enc:CipherData>
  </enc:EncryptedData>
</encryption>`,
			},
			wantFontObfuscate: true,
			wantErr:           nil,
		},
		{
			name: "Adobe ADEPT DRM",
			files: map[string]string{
				"mimetype": "application/epub+zip",
				"META-INF/encryption.xml": `<?xml version="1.0" encoding="UTF-8"?>
<encryption xmlns="urn:oasis:names:tc:opendocument:xmlns:container"
            xmlns:enc="http://www.w3.org/2001/04/xmlenc#">
  <enc:EncryptedData>
    <enc:EncryptionMethod Algorithm="http://www.w3.org/2001/04/xmlenc#aes128-cbc"/>
    <KeyInfo xmlns="http://www.w3.org/2000/09/xmldsig#">
      <resource xmlns="http://ns.adobe.com/adept"/>
    </KeyInfo>
    <enc:CipherData>
      <enc:CipherReference URI="OEBPS/chapter01.xhtml"/>
    </enc:CipherData>
  </enc:EncryptedData>
</encryption>`,
			},
			wantFontObfuscate: false,
			wantErr:           ErrDRMProtected,
		},
		{
			name: "Adobe ADEPT DRM via algorithm URI",
			files: map[string]string{
				"mimetype": "application/epub+zip",
				"META-INF/encryption.xml": `<?xml version="1.0" encoding="UTF-8"?>
<encryption xmlns="urn:oasis:names:tc:opendocument:xmlns:container"
            xmlns:enc="http://www.w3.org/2001/04/xmlenc#">
  <enc:EncryptedData>
    <enc:EncryptionMethod Algorithm="http://ns.adobe.com/adept/enc#aes256-cbc"/>
    <enc:CipherData>
      <enc:CipherReference URI="OEBPS/chapter01.xhtml"/>
    </enc:CipherData>
  </enc:EncryptedData>
</encryption>`,
			},
			wantFontObfuscate: false,
			wantErr:           ErrDRMProtected,
		},
		{
			name: "Readium LCP DRM",
			files: map[string]string{
				"mimetype": "application/epub+zip",
				"META-INF/encryption.xml": `<?xml version="1.0" encoding="UTF-8"?>
<encryption xmlns="urn:oasis:names:tc:opendocument:xmlns:container"
            xmlns:enc="http://www.w3.org/2001/04/xmlenc#">
  <enc:EncryptedData>
    <enc:EncryptionMethod Algorithm="http://www.w3.org/2001/04/xmlenc#aes256-cbc"/>
    <KeyInfo xmlns="http://www.w3.org/2000/09/xmldsig#">
      <resource xmlns="http://readium.org/2014/01/lcp"/>
    </KeyInfo>
    <enc:CipherData>
      <enc:CipherReference URI="OEBPS/chapter01.xhtml"/>
    </enc:CipherData>
  </enc:EncryptedData>
</encryption>`,
			},
			wantFontObfuscate: false,
			wantErr:           ErrDRMProtected,
		},
		{
			name: "DRM mixed with font obfuscation returns DRM error",
			files: map[string]string{
				"mimetype": "application/epub+zip",
				"META-INF/encryption.xml": `<?xml version="1.0" encoding="UTF-8"?>
<encryption xmlns="urn:oasis:names:tc:opendocument:xmlns:container"
            xmlns:enc="http://www.w3.org/2001/04/xmlenc#">
  <enc:EncryptedData>
    <enc:EncryptionMethod Algorithm="http://www.idpf.org/2008/embedding"/>
    <enc:CipherData>
      <enc:CipherReference URI="OEBPS/fonts/myfont.otf"/>
    </enc:CipherData>
  </enc:EncryptedData>
  <enc:EncryptedData>
    <enc:EncryptionMethod Algorithm="http://www.w3.org/2001/04/xmlenc#aes128-cbc"/>
    <KeyInfo xmlns="http://www.w3.org/2000/09/xmldsig#">
      <resource xmlns="http://ns.adobe.com/adept"/>
    </KeyInfo>
    <enc:CipherData>
      <enc:CipherReference URI="OEBPS/chapter01.xhtml"/>
    </enc:CipherData>
  </enc:EncryptedData>
</encryption>`,
			},
			wantFontObfuscate: false,
			wantErr:           ErrDRMProtected,
		},
		{
			name: "unknown encryption algorithm treated as DRM",
			files: map[string]string{
				"mimetype": "application/epub+zip",
				"META-INF/encryption.xml": `<?xml version="1.0" encoding="UTF-8"?>
<encryption xmlns="urn:oasis:names:tc:opendocument:xmlns:container"
            xmlns:enc="http://www.w3.org/2001/04/xmlenc#">
  <enc:EncryptedData>
    <enc:EncryptionMethod Algorithm="http://example.com/unknown-encryption"/>
    <enc:CipherData>
      <enc:CipherReference URI="OEBPS/chapter01.xhtml"/>
    </enc:CipherData>
  </enc:EncryptedData>
</encryption>`,
			},
			wantFontObfuscate: false,
			wantErr:           ErrDRMProtected,
		},
		{
			name: "empty encryption.xml with no EncryptedData",
			files: map[string]string{
				"mimetype": "application/epub+zip",
				"META-INF/encryption.xml": `<?xml version="1.0" encoding="UTF-8"?>
<encryption xmlns="urn:oasis:names:tc:opendocument:xmlns:container"
            xmlns:enc="http://www.w3.org/2001/04/xmlenc#">
</encryption>`,
			},
			wantFontObfuscate: false,
			wantErr:           nil,
		},
		{
			name: "Apple FairPlay DRM via sinf element",
			files: map[string]string{
				"mimetype": "application/epub+zip",
				"META-INF/encryption.xml": `<?xml version="1.0" encoding="UTF-8"?>
<encryption xmlns="urn:oasis:names:tc:opendocument:xmlns:container"
            xmlns:enc="http://www.w3.org/2001/04/xmlenc#">
  <enc:EncryptedData>
    <enc:EncryptionMethod Algorithm="http://www.w3.org/2001/04/xmlenc#aes128-cbc"/>
    <enc:CipherData>
      <enc:CipherReference URI="OEBPS/chapter01.xhtml"/>
    </enc:CipherData>
  </enc:EncryptedData>
</encryption>`,
				"META-INF/sinf.xml": `<sinf/>`,
			},
			wantFontObfuscate: false,
			wantErr:           ErrDRMProtected,
		},
		{
			name: "case insensitive encryption.xml path",
			files: map[string]string{
				"mimetype": "application/epub+zip",
				"meta-inf/Encryption.xml": `<?xml version="1.0" encoding="UTF-8"?>
<encryption xmlns="urn:oasis:names:tc:opendocument:xmlns:container"
            xmlns:enc="http://www.w3.org/2001/04/xmlenc#">
  <enc:EncryptedData>
    <enc:EncryptionMethod Algorithm="http://www.idpf.org/2008/embedding"/>
    <enc:CipherData>
      <enc:CipherReference URI="OEBPS/fonts/myfont.otf"/>
    </enc:CipherData>
  </enc:EncryptedData>
</encryption>`,
			},
			wantFontObfuscate: true,
			wantErr:           nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			zr := buildTestZip(t, tt.files)
			gotFont, gotErr := checkDRM(zr)

			if gotErr != tt.wantErr {
				t.Errorf("checkDRM() error = %v, want %v", gotErr, tt.wantErr)
			}
			if gotFont != tt.wantFontObfuscate {
				t.Errorf("checkDRM() fontObfuscation = %v, want %v", gotFont, tt.wantFontObfuscate)
			}
		})
	}
}
