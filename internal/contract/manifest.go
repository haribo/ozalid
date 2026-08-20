// Package contract holds the wire types shared by the server and the CLI.
//
// It is deliberately dependency-free: both binaries must agree on these shapes
// forever, so nothing here may pull in a framework or a driver.
package contract

// CaseRef names a case in a manifest. The id is the one ozalid generated when
// the case was created; the client never invents it (ADR 0014).
type CaseRef struct {
	ID string `json:"id"`
}
