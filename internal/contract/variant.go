package contract

import (
	"sort"
	"strings"
)

// VariantLabel renders a combination of axis values as a stable, readable
// name: "desktop·dark".
//
// The order comes from the project: an axis it placed earlier reads earlier.
// Anything the project has not ordered falls back to alphabetical, so a label
// exists from the very first intake, before anyone has declared anything.
//
// A label is a rendering, never an identity — two variants are the same when
// their values are, whatever they happen to be called.
func VariantLabel(values map[string]string, order []string) string {
	if len(values) == 0 {
		return "default"
	}

	rank := make(map[string]int, len(order))
	for i, axis := range order {
		rank[axis] = i
	}

	axes := make([]string, 0, len(values))
	for axis := range values {
		axes = append(axes, axis)
	}
	sort.Slice(axes, func(i, j int) bool {
		ri, oki := rank[axes[i]]
		rj, okj := rank[axes[j]]
		switch {
		case oki && okj:
			return ri < rj
		case oki:
			// A declared axis comes before an undeclared one: the project said
			// something about it, and that outranks alphabetical order.
			return true
		case okj:
			return false
		default:
			return axes[i] < axes[j]
		}
	})

	parts := make([]string, 0, len(axes))
	for _, axis := range axes {
		parts = append(parts, values[axis])
	}
	return strings.Join(parts, "·")
}
