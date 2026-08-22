package comment

import "errors"

// ErrIssueRequired means tracking was asked for without an issue to point at.
// A comment "tracked" by nothing is a comment nobody is working on.
var ErrIssueRequired = errors.New("comment: tracking needs an issue reference")
