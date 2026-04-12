package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	lctx "github.com/gtrig/laightdb/internal/context"
)

func TestPostEdge_Basic(t *testing.T) {
	t.Parallel()
	s, as := newTestServer(t)
	h := testHandler(s, as)

	body := `{"from_id":"node-a","to_id":"node-b","label":"child","weight":1.0}`
	req := httptest.NewRequest(http.MethodPost, "/v1/edges", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp["id"] == "" {
		t.Error("expected non-empty id in response")
	}
}

func TestPostEdge_BadRequest(t *testing.T) {
	t.Parallel()
	s, as := newTestServer(t)
	h := testHandler(s, as)

	// Missing from_id and to_id
	body := `{"label":"child"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/edges", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestGetEdge_Handler(t *testing.T) {
	t.Parallel()
	s, as := newTestServer(t)
	h := testHandler(s, as)

	// Create edge first
	body := `{"from_id":"x","to_id":"y","label":"related_to"}`
	createReq := httptest.NewRequest(http.MethodPost, "/v1/edges", strings.NewReader(body))
	createReq.Header.Set("Content-Type", "application/json")
	createRec := httptest.NewRecorder()
	h.ServeHTTP(createRec, createReq)
	var createResp map[string]string
	_ = json.NewDecoder(createRec.Body).Decode(&createResp)
	edgeID := createResp["id"]

	// Fetch it
	getReq := httptest.NewRequest(http.MethodGet, "/v1/edges/"+edgeID, nil)
	getRec := httptest.NewRecorder()
	h.ServeHTTP(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", getRec.Code)
	}
	var e map[string]any
	if err := json.NewDecoder(getRec.Body).Decode(&e); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if e["from_id"] != "x" || e["to_id"] != "y" {
		t.Errorf("unexpected edge data: %v", e)
	}
}

func TestGetEdge_NotFound_Handler(t *testing.T) {
	t.Parallel()
	s, as := newTestServer(t)
	h := testHandler(s, as)
	req := httptest.NewRequest(http.MethodGet, "/v1/edges/does-not-exist", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestListEdges_From(t *testing.T) {
	t.Parallel()
	s, as := newTestServer(t)
	h := testHandler(s, as)

	for _, to := range []string{"B", "C"} {
		body := fmt.Sprintf(`{"from_id":"A","to_id":%q,"label":"child"}`, to)
		req := httptest.NewRequest(http.MethodPost, "/v1/edges", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("create edge to %s: %d", to, rec.Code)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/edges?from=A", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]any
	_ = json.NewDecoder(rec.Body).Decode(&resp)
	edges, _ := resp["edges"].([]any)
	if len(edges) != 2 {
		t.Errorf("expected 2 edges, got %d", len(edges))
	}
}

func TestListEdges_MissingParam(t *testing.T) {
	t.Parallel()
	s, as := newTestServer(t)
	h := testHandler(s, as)
	req := httptest.NewRequest(http.MethodGet, "/v1/edges", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestDeleteEdge_Handler(t *testing.T) {
	t.Parallel()
	s, as := newTestServer(t)
	h := testHandler(s, as)

	body := `{"from_id":"p","to_id":"q","label":"child"}`
	createReq := httptest.NewRequest(http.MethodPost, "/v1/edges", strings.NewReader(body))
	createReq.Header.Set("Content-Type", "application/json")
	createRec := httptest.NewRecorder()
	h.ServeHTTP(createRec, createReq)
	var resp map[string]string
	_ = json.NewDecoder(createRec.Body).Decode(&resp)
	edgeID := resp["id"]

	delReq := httptest.NewRequest(http.MethodDelete, "/v1/edges/"+edgeID, nil)
	delRec := httptest.NewRecorder()
	h.ServeHTTP(delRec, delReq)
	if delRec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", delRec.Code)
	}
}

func TestGraphNeighbors_Handler(t *testing.T) {
	t.Parallel()
	s, as := newTestServer(t)
	h := testHandler(s, as)

	s.Store.PutEdge(t.Context(), lctx.PutEdgeRequest{FromID: "root", ToID: "child1", Label: "child"}) //nolint:errcheck
	s.Store.PutEdge(t.Context(), lctx.PutEdgeRequest{FromID: "root", ToID: "child2", Label: "child"}) //nolint:errcheck

	req := httptest.NewRequest(http.MethodGet, "/v1/graph/root/neighbors?depth=1", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]any
	_ = json.NewDecoder(rec.Body).Decode(&resp)
	neighbors, _ := resp["neighbors"].([]any)
	if len(neighbors) != 2 {
		t.Errorf("expected 2 neighbors, got %d", len(neighbors))
	}
}

func TestGraphSubtree_Handler(t *testing.T) {
	t.Parallel()
	s, as := newTestServer(t)
	h := testHandler(s, as)

	s.Store.PutEdge(t.Context(), lctx.PutEdgeRequest{FromID: "root", ToID: "A", Label: "child"})  //nolint:errcheck
	s.Store.PutEdge(t.Context(), lctx.PutEdgeRequest{FromID: "A", ToID: "A1", Label: "child"})   //nolint:errcheck

	req := httptest.NewRequest(http.MethodGet, "/v1/graph/root/subtree?depth=3", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]any
	_ = json.NewDecoder(rec.Body).Decode(&resp)
	edges, _ := resp["edges"].([]any)
	if len(edges) != 2 {
		t.Errorf("expected 2 subtree edges, got %d", len(edges))
	}
}

func TestGraphSearch_Handler(t *testing.T) {
	t.Parallel()
	s, as := newTestServer(t)
	h := testHandler(s, as)

	body := `{"query":"hello","top_k":5}`
	req := httptest.NewRequest(http.MethodPost, "/v1/graph/search", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestGraphSuggestLinks_Handler_NotFound(t *testing.T) {
	t.Parallel()
	s, as := newTestServer(t)
	h := testHandler(s, as)

	req := httptest.NewRequest(http.MethodGet, "/v1/graph/does-not-exist/suggest-links", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

