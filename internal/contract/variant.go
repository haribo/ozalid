package contract

import (
	"sort"
	"strings"
)

// VariantLabel renders a combination of axis values as a stable, readable
// name: "desktop·light·fr".
//
// The values are sorted by axis name so the same combination always renders
// the same way, whatever order the client sent it in.
func VariantLabel(values map[string]string) string {
	if len(values) == 0 {
		return "default"
	}
	axes := make([]string, 0, len(values))
	for axis := range values {
		axes = append(axes, axis)
	}
	sort.Strings(axes)

	parts := make([]string, 0, len(axes))
	for _, axis := range axes {
		parts = append(parts, values[axis])
	}
	return strings.Join(parts, "·")
}
