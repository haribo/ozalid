// Package contract holds the wire types shared by the server and the CLI.
//
// It is deliberately dependency-free: both binaries must agree on these shapes
// forever, so nothing here may pull in a framework or a driver.
package contract

// Manifest is what a client pushes to fill the book with one run's evidence.
//
// It names cases by the id ozalid generated for them; a client never invents
// one (ADR 0014).
type Manifest struct {
	// Revision is opaque to the server: displayed, never computed on
	// (ADR 0013). A commit hash, a build number, whatever the client means by
	// "which version produced this".
	Revision string         `json:"revision,omitempty"`
	Cases    []ManifestCase `json:"cases"`
}

// ManifestCase is one case's evidence in a run.
type ManifestCase struct {
	ID         string              `json:"id"`
	Steps      []ManifestStep      `json:"steps"`
	Recordings []ManifestRecording `json:"recordings,omitempty"`
}

// ManifestStep is a named business moment and the captures taken at it.
type ManifestStep struct {
	Name     string            `json:"name"`
	Captures []ManifestCapture `json:"captures"`
}

// ManifestCapture is one image: a variant, and the address of its bytes.
type ManifestCapture struct {
	// Variant is a combination of axis values. An axis the client does not
	// supply is absent from the map, not null: the site has no theme, so the
	// variant has no theme.
	Variant    map[string]string `json:"variant"`
	Hash       string            `json:"hash"`
	Provenance Provenance        `json:"provenance,omitempty"`
}

// ManifestRecording is the flow video for one variant. Optional, and never
// compared byte-wise (ADR 0013).
type ManifestRecording struct {
	Variant map[string]string `json:"variant"`
	Hash    string            `json:"hash"`
}

// Provenance records where a capture was produced. Byte comparison only means
// something within one environment (ADR 0004).
type Provenance struct {
	OS             string `json:"os,omitempty"`
	Browser        string `json:"browser,omitempty"`
	BrowserVersion string `json:"browserVersion,omitempty"`
	Resolution     string `json:"resolution,omitempty"`
	EnvironmentID  string `json:"environmentId,omitempty"`
}

// CaseRef names a case in a manifest.
type CaseRef struct {
	ID string `json:"id"`
}
