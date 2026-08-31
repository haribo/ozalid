package http_test

import (
	"context"
	"testing"
	"time"

	ozhttp "github.com/haribo/ozalid/apps/server/internal/ports/http"
	"github.com/haribo/ozalid/apps/server/internal/ports/http/openapi"
)

func TestGetHealthReportsTheBuildItRuns(t *testing.T) {
	srv := ozhttp.New(ozhttp.Deps{Version: "1.2.3", Now: time.Now})

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
