package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// testEntity is a minimal entity type used in named_entity unit tests.
type testEntity struct {
	ID   string
	Name string
}

// testEntityDTO is the DTO counterpart.
type testEntityDTO struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// testEntityRequest is the request body type.
type testEntityRequest struct {
	Name string `json:"name"`
}

// Test-local sentinels so named_entity tests are decoupled from any real entity.
var (
	errInvalidWidgetName = errors.New("invalid widget name")
	errWidgetNameExists  = errors.New("widget name already exists")
)

const (
	testWidgetIDKey       = "widget_id"
	testAuditWidgetCreate = "widget.created"
	testAuditWidgetUpdate = "widget.updated"
)

func toTestEntityDTO(e *testEntity) testEntityDTO {
	return testEntityDTO{ID: e.ID, Name: e.Name}
}

// makeTestNamedEntityOps returns a namedEntityOps backed by simple in-memory
// closures. The caller can override individual fields after receiving the ops.
func makeTestNamedEntityOps(t *testing.T) namedEntityOps[testEntity, testEntityDTO, testEntityRequest] {
	t.Helper()
	d := newTestDB(t)

	return namedEntityOps[testEntity, testEntityDTO, testEntityRequest]{
		db:             d,
		entityLabel:    "widget",
		entityArticle:  "a widget",
		idKey:          testWidgetIDKey,
		errInvalidName: errInvalidWidgetName,
		errNameExists:  errWidgetNameExists,
		auditCreate:    testAuditWidgetCreate,
		auditUpdate:    testAuditWidgetUpdate,
		get: func(_ context.Context, id string) (*testEntity, error) {
			return nil, sql.ErrNoRows // default to not-found; override in tests
		},
		create: func(_ context.Context, req testEntityRequest) (*testEntity, error) {
			return &testEntity{ID: "new-id", Name: req.Name}, nil
		},
		update: func(_ context.Context, id string, req testEntityRequest) (*testEntity, error) {
			return &testEntity{ID: id, Name: req.Name}, nil
		},
		reqName:    func(req testEntityRequest) string { return req.Name },
		entityName: func(e *testEntity) string { return e.Name },
		entityID:   func(e *testEntity) string { return e.ID },
		toDTO:      toTestEntityDTO,
	}
}

// ---- createNamedEntity ----

func TestCreateNamedEntity_Success(t *testing.T) {
	ops := makeTestNamedEntityOps(t)

	body := mustMarshal(t, testEntityRequest{Name: "My Widget"})
	r := httptest.NewRequest(http.MethodPost, "/api/widgets", bytes.NewReader(body))
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	createNamedEntity(ops, w, r)

	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusCreated, w.Body.String())
	}

	var dto testEntityDTO
	if err := json.Unmarshal(w.Body.Bytes(), &dto); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if dto.Name != "My Widget" {
		t.Errorf("name = %q, want %q", dto.Name, "My Widget")
	}
}

func TestCreateNamedEntity_InvalidJSON(t *testing.T) {
	ops := makeTestNamedEntityOps(t)

	r := httptest.NewRequest(http.MethodPost, "/api/widgets", strings.NewReader("not-json"))
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	createNamedEntity(ops, w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestCreateNamedEntity_EmptyName(t *testing.T) {
	ops := makeTestNamedEntityOps(t)

	body := mustMarshal(t, testEntityRequest{Name: ""})
	r := httptest.NewRequest(http.MethodPost, "/api/widgets", bytes.NewReader(body))
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	createNamedEntity(ops, w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestCreateNamedEntity_WhitespaceOnlyName(t *testing.T) {
	ops := makeTestNamedEntityOps(t)

	body := mustMarshal(t, testEntityRequest{Name: "   "})
	r := httptest.NewRequest(http.MethodPost, "/api/widgets", bytes.NewReader(body))
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	createNamedEntity(ops, w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestCreateNamedEntity_ErrInvalidName(t *testing.T) {
	ops := makeTestNamedEntityOps(t)
	ops.create = func(_ context.Context, _ testEntityRequest) (*testEntity, error) {
		return nil, errInvalidWidgetName
	}

	body := mustMarshal(t, testEntityRequest{Name: "Valid Name"})
	r := httptest.NewRequest(http.MethodPost, "/api/widgets", bytes.NewReader(body))
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	createNamedEntity(ops, w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestCreateNamedEntity_ErrNameExists(t *testing.T) {
	ops := makeTestNamedEntityOps(t)
	ops.create = func(_ context.Context, _ testEntityRequest) (*testEntity, error) {
		return nil, errWidgetNameExists
	}

	body := mustMarshal(t, testEntityRequest{Name: "Duplicate"})
	r := httptest.NewRequest(http.MethodPost, "/api/widgets", bytes.NewReader(body))
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	createNamedEntity(ops, w, r)

	if w.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusConflict, w.Body.String())
	}
}

func TestCreateNamedEntity_GenericCreateError(t *testing.T) {
	ops := makeTestNamedEntityOps(t)
	ops.create = func(_ context.Context, _ testEntityRequest) (*testEntity, error) {
		return nil, errors.New("unexpected db error")
	}

	body := mustMarshal(t, testEntityRequest{Name: "Widgets Inc"})
	r := httptest.NewRequest(http.MethodPost, "/api/widgets", bytes.NewReader(body))
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	createNamedEntity(ops, w, r)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestCreateNamedEntity_NilEntityWithoutError(t *testing.T) {
	ops := makeTestNamedEntityOps(t)
	ops.create = func(_ context.Context, _ testEntityRequest) (*testEntity, error) {
		return nil, nil
	}

	body := mustMarshal(t, testEntityRequest{Name: "Widget"})
	r := httptest.NewRequest(http.MethodPost, "/api/widgets", bytes.NewReader(body))
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	createNamedEntity(ops, w, r)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

// ---- getNamedEntity ----

func TestGetNamedEntity_Success(t *testing.T) {
	ops := makeTestNamedEntityOps(t)
	ops.get = func(_ context.Context, id string) (*testEntity, error) {
		return &testEntity{ID: id, Name: "Existing Widget"}, nil
	}

	r := httptest.NewRequest(http.MethodGet, "/api/widgets/entity-1", nil)
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	getNamedEntity(ops, w, r, "entity-1")

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var dto testEntityDTO
	if err := json.Unmarshal(w.Body.Bytes(), &dto); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if dto.Name != "Existing Widget" {
		t.Errorf("name = %q, want %q", dto.Name, "Existing Widget")
	}
	if dto.ID != "entity-1" {
		t.Errorf("id = %q, want %q", dto.ID, "entity-1")
	}
}

func TestGetNamedEntity_NotFound(t *testing.T) {
	ops := makeTestNamedEntityOps(t)
	// ops.get already returns sql.ErrNoRows by default

	r := httptest.NewRequest(http.MethodGet, "/api/widgets/missing", nil)
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	getNamedEntity(ops, w, r, "missing")

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestGetNamedEntity_WrappedNotFound(t *testing.T) {
	ops := makeTestNamedEntityOps(t)
	ops.get = func(_ context.Context, _ string) (*testEntity, error) {
		return nil, fmt.Errorf("query failed: %w", sql.ErrNoRows)
	}

	r := httptest.NewRequest(http.MethodGet, "/api/widgets/missing", nil)
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	getNamedEntity(ops, w, r, "missing")

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestGetNamedEntity_DBError(t *testing.T) {
	ops := makeTestNamedEntityOps(t)
	ops.get = func(_ context.Context, _ string) (*testEntity, error) {
		return nil, errors.New("connection refused")
	}

	r := httptest.NewRequest(http.MethodGet, "/api/widgets/any", nil)
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	getNamedEntity(ops, w, r, "any")

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestGetNamedEntity_NilEntityWithoutError(t *testing.T) {
	ops := makeTestNamedEntityOps(t)
	ops.get = func(_ context.Context, _ string) (*testEntity, error) {
		return nil, nil
	}

	r := httptest.NewRequest(http.MethodGet, "/api/widgets/any", nil)
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	getNamedEntity(ops, w, r, "any")

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

// ---- updateNamedEntity ----

func TestUpdateNamedEntity_Success(t *testing.T) {
	ops := makeTestNamedEntityOps(t)

	body := mustMarshal(t, testEntityRequest{Name: "Updated Widget"})
	r := httptest.NewRequest(http.MethodPut, "/api/widgets/entity-1", bytes.NewReader(body))
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	updateNamedEntity(ops, w, r, "entity-1")

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusOK, w.Body.String())
	}

	var dto testEntityDTO
	if err := json.Unmarshal(w.Body.Bytes(), &dto); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if dto.Name != "Updated Widget" {
		t.Errorf("name = %q, want %q", dto.Name, "Updated Widget")
	}
}

func TestUpdateNamedEntity_InvalidJSON(t *testing.T) {
	ops := makeTestNamedEntityOps(t)

	r := httptest.NewRequest(http.MethodPut, "/api/widgets/entity-1", strings.NewReader("not-json"))
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	updateNamedEntity(ops, w, r, "entity-1")

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestUpdateNamedEntity_EmptyName(t *testing.T) {
	ops := makeTestNamedEntityOps(t)

	body := mustMarshal(t, testEntityRequest{Name: ""})
	r := httptest.NewRequest(http.MethodPut, "/api/widgets/entity-1", bytes.NewReader(body))
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	updateNamedEntity(ops, w, r, "entity-1")

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestUpdateNamedEntity_NotFound(t *testing.T) {
	ops := makeTestNamedEntityOps(t)
	ops.update = func(_ context.Context, _ string, _ testEntityRequest) (*testEntity, error) {
		return nil, sql.ErrNoRows
	}

	body := mustMarshal(t, testEntityRequest{Name: "New Name"})
	r := httptest.NewRequest(http.MethodPut, "/api/widgets/missing", bytes.NewReader(body))
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	updateNamedEntity(ops, w, r, "missing")

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestUpdateNamedEntity_WrappedNotFound(t *testing.T) {
	ops := makeTestNamedEntityOps(t)
	ops.update = func(_ context.Context, _ string, _ testEntityRequest) (*testEntity, error) {
		return nil, fmt.Errorf("update failed: %w", sql.ErrNoRows)
	}

	body := mustMarshal(t, testEntityRequest{Name: "New Name"})
	r := httptest.NewRequest(http.MethodPut, "/api/widgets/missing", bytes.NewReader(body))
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	updateNamedEntity(ops, w, r, "missing")

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestUpdateNamedEntity_ErrInvalidName(t *testing.T) {
	ops := makeTestNamedEntityOps(t)
	ops.update = func(_ context.Context, _ string, _ testEntityRequest) (*testEntity, error) {
		return nil, errInvalidWidgetName
	}

	body := mustMarshal(t, testEntityRequest{Name: "Anything"})
	r := httptest.NewRequest(http.MethodPut, "/api/widgets/entity-1", bytes.NewReader(body))
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	updateNamedEntity(ops, w, r, "entity-1")

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusBadRequest, w.Body.String())
	}
}

func TestUpdateNamedEntity_ErrNameExists(t *testing.T) {
	ops := makeTestNamedEntityOps(t)
	ops.update = func(_ context.Context, _ string, _ testEntityRequest) (*testEntity, error) {
		return nil, errWidgetNameExists
	}

	body := mustMarshal(t, testEntityRequest{Name: "Duplicate"})
	r := httptest.NewRequest(http.MethodPut, "/api/widgets/entity-1", bytes.NewReader(body))
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	updateNamedEntity(ops, w, r, "entity-1")

	if w.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d; body: %s", w.Code, http.StatusConflict, w.Body.String())
	}
}

func TestUpdateNamedEntity_GenericUpdateError(t *testing.T) {
	ops := makeTestNamedEntityOps(t)
	ops.update = func(_ context.Context, _ string, _ testEntityRequest) (*testEntity, error) {
		return nil, errors.New("unexpected db failure")
	}

	body := mustMarshal(t, testEntityRequest{Name: "Widget Name"})
	r := httptest.NewRequest(http.MethodPut, "/api/widgets/entity-1", bytes.NewReader(body))
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	updateNamedEntity(ops, w, r, "entity-1")

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestUpdateNamedEntity_NilEntityWithoutError(t *testing.T) {
	ops := makeTestNamedEntityOps(t)
	ops.update = func(_ context.Context, _ string, _ testEntityRequest) (*testEntity, error) {
		return nil, nil
	}

	body := mustMarshal(t, testEntityRequest{Name: "Widget"})
	r := httptest.NewRequest(http.MethodPut, "/api/widgets/entity-1", bytes.NewReader(body))
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	updateNamedEntity(ops, w, r, "entity-1")

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}
