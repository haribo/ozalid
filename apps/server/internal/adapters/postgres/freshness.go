package postgres

import (
	"context"

	appintake "github.com/haribo/ozalid/apps/server/internal/app/intake"
	"github.com/haribo/ozalid/internal/contract"
)

// PixelThreshold is how many differing pixels this project calls noise.
func (r *Repository) PixelThreshold(ctx context.Context, projectSlug string) (int, error) {
	threshold, err := r.q.ProjectThreshold(ctx, projectSlug)
	if err != nil {
		return 0, translate("reading the project's threshold", err)
	}
	return int(threshold), nil
}

// AxisOrder returns the order the project declares its axes in.
//
// The caller uses it to build the same variant label the write will store. Two
// different orders would produce two different labels for one variant, and the
// lookup would miss every time.
func (r *Repository) AxisOrder(ctx context.Context, projectSlug string) ([]string, error) {
	project, err := r.q.GetProjectBySlug(ctx, projectSlug)
	if err != nil {
		return nil, translate("reading the project", err)
	}
	axes, err := r.q.ListAxes(ctx, project.ID)
	if err != nil {
		return nil, translate("reading the axes", err)
	}
	order := make([]string, 0, len(axes))
	for _, a := range axes {
		order = append(order, a.Name)
	}
	return order, nil
}

// ApprovedBytes returns, per square, the content address a reviewer last
// approved.
//
// Squares with no reference are absent rather than empty: "nobody approved
// this" and "what was approved is gone" are different answers, and only the
// first one is true here.
func (r *Repository) ApprovedBytes(
	ctx context.Context, projectSlug string, m contract.Manifest,
) (map[appintake.Square]string, error) {
	ids := make([]string, 0, len(m.Cases))
	for _, c := range m.Cases {
		ids = append(ids, c.ID)
	}
	rows, err := r.q.ReferencesForCases(ctx, ids)
	if err != nil {
		return nil, translate("reading the references", err)
	}

	out := make(map[appintake.Square]string, len(rows))
	for _, row := range rows {
		out[appintake.Square{
			CaseID:        row.CaseID,
			StepPosition:  int(row.StepPosition),
			VariantLabel:  row.VariantLabel,
			EnvironmentID: row.EnvironmentID,
		}] = row.BlobHash
	}
	return out, nil
}

// The compiler checks the adapter still answers everything intake asks for.
var _ appintake.Repository = (*Repository)(nil)
