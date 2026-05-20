package runtime

import (
	"context"
	"encoding/json"
	"fmt"

	"elydelva/one/internal/core"
	"elydelva/one/internal/ports"
)

// executePaginated walks a cursor-paginated action, concatenating each page's
// items into a single JSON array. Stops at the first page error or when
// max_pages is reached.
func (r *DeclarativeRuntime) executePaginated(ctx context.Context, svc *core.Service, req ports.ExecuteRequest) (ports.ExecuteResult, error) {
	cfg := req.Action.Pagination
	if cfg.MaxPages <= 0 {
		cfg.MaxPages = 50 // safety cap default
	}

	var allItems []any
	var allCalls []ports.HTTPCall
	token := ""

	for page := 0; page < cfg.MaxPages; page++ {
		result, err := r.executeOnce(ctx, svc, req, token)
		allCalls = append(allCalls, result.Calls...)
		if err != nil {
			return ports.ExecuteResult{Calls: allCalls}, err
		}

		items, nextToken, err := extractPage(result.Output, cfg)
		if err != nil {
			return ports.ExecuteResult{Calls: allCalls}, err
		}
		allItems = append(allItems, items...)

		if nextToken == "" {
			break
		}
		token = nextToken
	}

	out, err := json.Marshal(allItems)
	if err != nil {
		return ports.ExecuteResult{Calls: allCalls}, err
	}
	return ports.ExecuteResult{Output: out, Calls: allCalls}, nil
}

// extractPage pulls the items array and next-cursor token from a page body.
// items can be the top-level array (ResponseItems="") or a nested field.
func extractPage(body json.RawMessage, cfg *core.PaginationSpec) ([]any, string, error) {
	if cfg.ResponseItems == "" {
		var items []any
		if err := json.Unmarshal(body, &items); err != nil {
			return nil, "", fmt.Errorf("pagination expects top-level array: %w", err)
		}
		// Cursor is then nowhere in this body shape, but some APIs return the
		// cursor in a header — Phase 2 only supports body cursor with an envelope.
		return items, "", nil
	}

	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, "", fmt.Errorf("pagination expects object envelope: %w", err)
	}
	var items []any
	if raw, ok := envelope[cfg.ResponseItems]; ok {
		if err := json.Unmarshal(raw, &items); err != nil {
			return nil, "", fmt.Errorf("items field %q is not an array: %w", cfg.ResponseItems, err)
		}
	}
	var token string
	if raw, ok := envelope[cfg.ResponseToken]; ok {
		_ = json.Unmarshal(raw, &token) // best-effort; non-string → empty
	}
	return items, token, nil
}
