// Package account carries what the interface is told about a person.
//
// A projection, not the account itself: the name and address a signed-in
// reviewer sees at the top of the page, and whether they administer the
// instance. Everything else about an account is nobody's business.
package account

// Person is a signed-in human, as the client needs them.
type Person struct {
	ID    string
	Name  string
	Email string
	// IsAdmin manages accounts and creates projects. It reaches no content
	// (product.md §8.2), so the client uses it to show an administration
	// screen and never to decide what a review may do.
	IsAdmin bool
}
