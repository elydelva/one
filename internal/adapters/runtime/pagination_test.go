package runtime

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"elydelva/one/internal/core"
	"elydelva/one/internal/ports"
	"elydelva/one/internal/testing/fakeapi"
)

// fixture: action that paginates an envelope with `items` + `next_page`.
func paginatedAction() core.Action {
	defs := `[{"name":"page","type":"integer","location":"query"}]`
	return core.Action{
		ID: "things.list", Service: "things", Permission: "things.read",
		Request:     &core.RequestSpec{Method: "GET", Path: "/things"},
		InputSchema: json.RawMessage(defs),
		Pagination: &core.PaginationSpec{
			Style: "cursor", RequestParam: "page", RequestLocation: "query",
			ResponseToken: "next_page", ResponseItems: "items", MaxPages: 5,
		},
	}
}

func TestPagination_FollowsCursor(t *testing.T) {
	page := 0
	srv := fakeapi.New(t, []fakeapi.Route{
		{Method: "GET", Path: "/things", Status: 200, OnMatch: func(r *http.Request) {
			page++
		}, Body: nil},
	})
	// override default body to be page-aware: rewrite handler manually via simpler structure.
	srv.Close()

	// Simpler: build 3 routes via a counter handler in test.
	hits := 0
	mux := http.NewServeMux()
	mux.HandleFunc("/things", func(w http.ResponseWriter, _ *http.Request) {
		hits++
		w.Header().Set("Content-Type", "application/json")
		switch hits {
		case 1:
			_, _ = w.Write([]byte(`{"items":[1,2],"next_page":"p2"}`))
		case 2:
			_, _ = w.Write([]byte(`{"items":[3,4],"next_page":"p3"}`))
		case 3:
			_, _ = w.Write([]byte(`{"items":[5],"next_page":""}`))
		}
	})
	rt, _, svc := newDeclWithMux(t, mux)
	svc.ID = "things"

	res, err := rt.Execute(context.Background(), ports.ExecuteRequest{
		Action: paginatedAction(), Inputs: core.Inputs{},
		Credential: core.Credential{Provider: core.ProviderPAT, AccessToken: core.NewSecret("t")},
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	var items []any
	_ = json.Unmarshal(res.Output, &items)
	if len(items) != 5 {
		t.Errorf("expected 5 items concatenated, got %d: %s", len(items), res.Output)
	}
	if hits != 3 {
		t.Errorf("expected 3 HTTP calls, got %d", hits)
	}
	if len(res.Calls) != 3 {
		t.Errorf("audit trail = %d calls", len(res.Calls))
	}
}

func TestPagination_StopsAtMaxPages(t *testing.T) {
	hits := 0
	mux := http.NewServeMux()
	mux.HandleFunc("/things", func(w http.ResponseWriter, _ *http.Request) {
		hits++
		_, _ = w.Write([]byte(`{"items":[1],"next_page":"keep-going"}`))
	})
	rt, _, _ := newDeclWithMux(t, mux)

	action := paginatedAction()
	action.Pagination.MaxPages = 2

	_, err := rt.Execute(context.Background(), ports.ExecuteRequest{
		Action: action, Inputs: core.Inputs{},
		Credential: core.Credential{Provider: core.ProviderPAT, AccessToken: core.NewSecret("t")},
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if hits != 2 {
		t.Errorf("expected cap at 2, got %d hits", hits)
	}
}
