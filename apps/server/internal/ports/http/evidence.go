package http

import (
	"context"
	"errors"

	app "github.com/haribo/ozalid/apps/server/internal/app/catalogue"
	"github.com/haribo/ozalid/apps/server/internal/app/evidence"
	"net/http"

	"github.com/haribo/ozalid/apps/server/internal/domain/access"
	"github.com/haribo/ozalid/apps/server/internal/ports/http/openapi"
	"github.com/haribo/ozalid/internal/contract"
)

// GetCaseCaptures returns the grid a case is judged from.
func (s *Server) GetCaseCaptures(ctx context.Context, request openapi.GetCaseCapturesRequestObject) (openapi.GetCaseCapturesResponseObject, error) {
	if why, no := s.mayNot(ctx, request.Slug, access.ReadProject); no {
		if why.Status == http.StatusUnauthorized {
			return openapi.GetCaseCaptures401ApplicationProblemPlusJSONResponse{
				UnauthenticatedApplicationProblemPlusJSONResponse: openapi.UnauthenticatedApplicationProblemPlusJSONResponse(why),
			}, nil
		}
		return openapi.GetCaseCaptures403ApplicationProblemPlusJSONResponse{
			ForbiddenApplicationProblemPlusJSONResponse: openapi.ForbiddenApplicationProblemPlusJSONResponse(why),
		}, nil
	}
	grid, err := s.evidence.Grid(ctx, request.Slug, request.CaseId, request.Params.EditionId)
	if errors.Is(err, app.ErrNotFound) {
		return openapi.GetCaseCaptures404ApplicationProblemPlusJSONResponse{
			NotFoundApplicationProblemPlusJSONResponse: notFound("case"),
		}, nil
	}
	if err != nil {
		return nil, err
	}
	return openapi.GetCaseCaptures200JSONResponse(toAPIGrid(grid)), nil
}

func toAPIGrid(g evidence.Grid) openapi.Grid {
	out := openapi.Grid{
		CaseId:     g.CaseID,
		Variants:   make([]openapi.GridVariant, 0, len(g.Variants)),
		Steps:      make([]openapi.GridStep, 0, len(g.Steps)),
		Recordings: make([]openapi.GridRecording, 0, len(g.Recordings)),
	}
	if g.EditionID != "" {
		out.EditionId = &g.EditionID
		takenAt := g.TakenAt
		out.TakenAt = &takenAt
	}
	if g.Revision != "" {
		out.Revision = &g.Revision
	}

	for _, v := range g.Variants {
		out.Variants = append(out.Variants, openapi.GridVariant{
			Id: v.ID, Label: v.Label, Values: v.Values,
		})
	}
	for _, st := range g.Steps {
		step := openapi.GridStep{
			Id: st.ID, Name: st.Name, Position: st.Position,
			Cells: make([]openapi.GridCell, 0, len(st.Cells)),
		}
		for _, c := range st.Cells {
			cell := openapi.GridCell{
				Id:         c.ID,
				VariantId:  c.VariantID,
				Hash:       c.Hash,
				Status:     openapi.GridCellStatus(c.Status),
				Provenance: toAPIProvenance(c.Provenance),
			}
			// Absent rather than empty: "nothing to compare against" is a
			// different answer from "unchanged" (ADR 0017).
			if c.Freshness != "" {
				fresh := openapi.GridCellFreshness(c.Freshness)
				cell.Freshness = &fresh
			}
			cell.MovedPixels = c.MovedPixels
			step.Cells = append(step.Cells, cell)
		}
		out.Steps = append(out.Steps, step)
	}
	for _, r := range g.Recordings {
		out.Recordings = append(out.Recordings, openapi.GridRecording{
			Id: r.ID, VariantId: r.VariantID, Hash: r.Hash,
		})
	}
	return out
}

// toAPIProvenance omits the whole object when nothing was recorded, rather than
// serialising five empty strings.
func toAPIProvenance(p contract.Provenance) *openapi.Provenance {
	if p == (contract.Provenance{}) {
		return nil
	}
	return &openapi.Provenance{
		Os:             optionalString(p.OS),
		Browser:        optionalString(p.Browser),
		BrowserVersion: optionalString(p.BrowserVersion),
		Resolution:     optionalString(p.Resolution),
		EnvironmentId:  optionalString(p.EnvironmentID),
	}
}

func optionalString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
