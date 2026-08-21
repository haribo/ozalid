package http_test

import (
	"context"
	"testing"

	ozhttp "github.com/haribo/ozalid/apps/server/internal/ports/http"
	"github.com/haribo/ozalid/apps/server/internal/ports/http/openapi"
)

func TestGetHealthReportsTheBuildItRuns(t *testing.T) {
	srv := ozhttp.New("1.2.3", nil, nil)

	got, err := srv.GetHealth(context.Background(), openapi.GetHealthRequestObject{})
	if err != nil {
		t.Fatalf("GetHealth returned an error: %v", err)
	}

	resp, ok := got.(openapi.GetHealth200JSONResponse)
	if !ok {
		t.Fatalf("GetHealth returned %T, want GetHealth200JSONResponse", got)
	}
	if resp.Status != openapi.Ok {
		t.Errorf("status = %q, want %q", resp.Status, openapi.Ok)
	}
	if resp.Version != "1.2.3" {
		t.Errorf("version = %q, want the build the server was given", resp.Version)
	}
}
