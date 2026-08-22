package contract_test

import (
	"testing"

	"github.com/haribo/ozalid/internal/contract"
)

func TestALabelFollowsTheOrderTheProjectDeclared(t *testing.T) {
	values := map[string]string{"theme": "dark", "viewport": "desktop"}

	// Alphabetically theme wins, which reads backwards. The project's own
	// order is what fixes it.
	if got := contract.VariantLabel(values, nil); got != "dark·desktop" {
		t.Errorf("without an order, label = %q, want alphabetical", got)
	}
	if got := contract.VariantLabel(values, []string{"viewport", "theme"}); got != "desktop·dark" {
		t.Errorf("with an order, label = %q, want %q", got, "desktop·dark")
	}
}

func TestAnUndeclaredAxisStillGetsALabel(t *testing.T) {
	values := map[string]string{"viewport": "mobile", "locale": "fr", "theme": "dark"}

	// Only two of the three are ordered; the third must not vanish, and the
	// two that are ordered must still come first.
	got := contract.VariantLabel(values, []string{"viewport", "theme"})
	if got != "mobile·dark·fr" {
		t.Errorf("label = %q, want the declared axes first then the rest", got)
	}
}

func TestALabelDoesNotDependOnMapOrder(t *testing.T) {
	order := []string{"viewport", "theme"}
	first := contract.VariantLabel(map[string]string{"theme": "dark", "viewport": "mobile"}, order)
	second := contract.VariantLabel(map[string]string{"viewport": "mobile", "theme": "dark"}, order)
	if first != second {
		t.Errorf("the same combination rendered %q then %q", first, second)
	}
}

func TestAVariantWithNoAxesIsNamedRatherThanBlank(t *testing.T) {
	// A project capturing one theme and one language has no axes at all.
	if got := contract.VariantLabel(nil, nil); got != "default" {
		t.Errorf("label = %q, want %q", got, "default")
	}
}
