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

	"github.com/amalgamated-tools/biblioteka/internal/auth"
	"github.com/amalgamated-tools/biblioteka/internal/otelkeys"
	"github.com/stretchr/testify/require"
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

	require.Equal(t, http.StatusCreated, w.Code)

	var dto testEntityDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &dto), "unmarshal")
	require.Equal(t, "My Widget", dto.Name)
}

func TestCreateNamedEntity_AuditEntityTypeDefaultsToLabel(t *testing.T) {
	ops := makeTestNamedEntityOps(t)
	ops.auditEntityType = ""

	body := mustMarshal(t, testEntityRequest{Name: "My Widget"})
	r := httptest.NewRequest(http.MethodPost, "/api/widgets", bytes.NewReader(body))
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	createNamedEntity(ops, w, r)

	require.Equal(t, http.StatusCreated, w.Code)

	ctx := t.Context()
	logs, _, err := ops.db.ListAuditLogs(ctx, 10, 0)
	require.NoError(t, err, "list audit logs")
	require.Len(t, logs, 1)
	require.Equal(t, "widget", logs[0].EntityType)
}

func TestCreateNamedEntity_InvalidJSON(t *testing.T) {
	ops := makeTestNamedEntityOps(t)

	r := httptest.NewRequest(http.MethodPost, "/api/widgets", strings.NewReader("not-json"))
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	createNamedEntity(ops, w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateNamedEntity_EmptyName(t *testing.T) {
	ops := makeTestNamedEntityOps(t)

	body := mustMarshal(t, testEntityRequest{Name: ""})
	r := httptest.NewRequest(http.MethodPost, "/api/widgets", bytes.NewReader(body))
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	createNamedEntity(ops, w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateNamedEntity_WhitespaceOnlyName(t *testing.T) {
	ops := makeTestNamedEntityOps(t)

	body := mustMarshal(t, testEntityRequest{Name: "   "})
	r := httptest.NewRequest(http.MethodPost, "/api/widgets", bytes.NewReader(body))
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	createNamedEntity(ops, w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)
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

	require.Equal(t, http.StatusBadRequest, w.Code)
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

	require.Equal(t, http.StatusConflict, w.Code)
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

	require.Equal(t, http.StatusInternalServerError, w.Code)
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

	require.Equal(t, http.StatusInternalServerError, w.Code)
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

	require.Equal(t, http.StatusOK, w.Code)

	var dto testEntityDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &dto), "unmarshal")
	require.Equal(t, "Existing Widget", dto.Name)
	require.Equal(t, "entity-1", dto.ID)
}

func TestGetNamedEntity_NotFound(t *testing.T) {
	ops := makeTestNamedEntityOps(t)
	// ops.get already returns sql.ErrNoRows by default

	r := httptest.NewRequest(http.MethodGet, "/api/widgets/missing", nil)
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	getNamedEntity(ops, w, r, "missing")

	require.Equal(t, http.StatusNotFound, w.Code)
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

	require.Equal(t, http.StatusNotFound, w.Code)
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

	require.Equal(t, http.StatusInternalServerError, w.Code)
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

	require.Equal(t, http.StatusInternalServerError, w.Code)
}

// ---- updateNamedEntity ----

func TestUpdateNamedEntity_Success(t *testing.T) {
	ops := makeTestNamedEntityOps(t)

	body := mustMarshal(t, testEntityRequest{Name: "Updated Widget"})
	r := httptest.NewRequest(http.MethodPut, "/api/widgets/entity-1", bytes.NewReader(body))
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	updateNamedEntity(ops, w, r, "entity-1")

	require.Equal(t, http.StatusOK, w.Code)

	var dto testEntityDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &dto), "unmarshal")
	require.Equal(t, "Updated Widget", dto.Name)
}

func TestUpdateNamedEntity_InvalidJSON(t *testing.T) {
	ops := makeTestNamedEntityOps(t)

	r := httptest.NewRequest(http.MethodPut, "/api/widgets/entity-1", strings.NewReader("not-json"))
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	updateNamedEntity(ops, w, r, "entity-1")

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateNamedEntity_EmptyName(t *testing.T) {
	ops := makeTestNamedEntityOps(t)

	body := mustMarshal(t, testEntityRequest{Name: ""})
	r := httptest.NewRequest(http.MethodPut, "/api/widgets/entity-1", bytes.NewReader(body))
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	updateNamedEntity(ops, w, r, "entity-1")

	require.Equal(t, http.StatusBadRequest, w.Code)
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

	require.Equal(t, http.StatusNotFound, w.Code)
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

	require.Equal(t, http.StatusNotFound, w.Code)
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

	require.Equal(t, http.StatusBadRequest, w.Code)
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

	require.Equal(t, http.StatusConflict, w.Code)
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

	require.Equal(t, http.StatusInternalServerError, w.Code)
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

	require.Equal(t, http.StatusInternalServerError, w.Code)
}

// ---- user-scoped namedEntityOps helpers ----

// makeTestUserOwnedNamedEntityOps returns a namedEntityOps backed by
// simple in-memory closures. The caller can override individual fields after
// receiving the ops.
func makeTestUserOwnedNamedEntityOps(t *testing.T) namedEntityOps[testEntity, testEntityDTO, testEntityRequest] {
	t.Helper()
	d := newTestDB(t)

	return namedEntityOps[testEntity, testEntityDTO, testEntityRequest]{
		db:              d,
		entityLabel:     "widget",
		auditEntityType: "widget",
		entityArticle:   "a widget",
		idKey:           testWidgetIDKey,
		errInvalidName:  errInvalidWidgetName,
		errNameExists:   errWidgetNameExists,
		auditCreate:     testAuditWidgetCreate,
		auditUpdate:     testAuditWidgetUpdate,
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

// ---- createNamedEntity with user-scoped closures ----

func TestCreateNamedEntity_UserScoped_Success(t *testing.T) {
	ops := makeTestUserOwnedNamedEntityOps(t)

	var capturedUserID string
	ops.create = func(ctx context.Context, req testEntityRequest) (*testEntity, error) {
		capturedUserID = auth.UserIDFromContext(ctx)
		return &testEntity{ID: "new-id", Name: req.Name}, nil
	}

	body := mustMarshal(t, testEntityRequest{Name: "My Widget"})
	r := httptest.NewRequest(http.MethodPost, "/api/widgets", bytes.NewReader(body))
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	createNamedEntity(ops, w, r)

	require.Equal(t, http.StatusCreated, w.Code)
	require.Equal(t, "user-1", capturedUserID)

	var dto testEntityDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &dto), "unmarshal")
	require.Equal(t, "My Widget", dto.Name)
}

func TestCreateNamedEntity_UserScoped_UsesConfiguredAuditEntityType(t *testing.T) {
	ops := makeTestUserOwnedNamedEntityOps(t)

	body := mustMarshal(t, testEntityRequest{Name: "My Widget"})
	r := httptest.NewRequest(http.MethodPost, "/api/widgets", bytes.NewReader(body))
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	createNamedEntity(ops, w, r)

	require.Equal(t, http.StatusCreated, w.Code)

	ctx := t.Context()
	logs, _, err := ops.db.ListAuditLogs(ctx, 10, 0)
	require.NoError(t, err, "list audit logs")
	require.Len(t, logs, 1)
	require.Equal(t, "widget", logs[0].EntityType)
	require.NotNil(t, logs[0].Metadata)

	var metadata map[string]string
	require.NoError(t, json.Unmarshal([]byte(*logs[0].Metadata), &metadata), "unmarshal metadata")
	require.Equal(t, "My Widget", metadata[otelkeys.Name])
}

func TestCreateUserOwnedNamedEntity_InvalidJSON(t *testing.T) {
	ops := makeTestUserOwnedNamedEntityOps(t)

	r := httptest.NewRequest(http.MethodPost, "/api/widgets", strings.NewReader("not-json"))
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	createNamedEntity(ops, w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateUserOwnedNamedEntity_EmptyName(t *testing.T) {
	ops := makeTestUserOwnedNamedEntityOps(t)

	body := mustMarshal(t, testEntityRequest{Name: ""})
	r := httptest.NewRequest(http.MethodPost, "/api/widgets", bytes.NewReader(body))
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	createNamedEntity(ops, w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateUserOwnedNamedEntity_ErrInvalidName(t *testing.T) {
	ops := makeTestUserOwnedNamedEntityOps(t)
	ops.create = func(_ context.Context, _ testEntityRequest) (*testEntity, error) {
		return nil, errInvalidWidgetName
	}

	body := mustMarshal(t, testEntityRequest{Name: "Valid Name"})
	r := httptest.NewRequest(http.MethodPost, "/api/widgets", bytes.NewReader(body))
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	createNamedEntity(ops, w, r)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateUserOwnedNamedEntity_ErrNameExists(t *testing.T) {
	ops := makeTestUserOwnedNamedEntityOps(t)
	ops.create = func(_ context.Context, _ testEntityRequest) (*testEntity, error) {
		return nil, errWidgetNameExists
	}

	body := mustMarshal(t, testEntityRequest{Name: "Duplicate"})
	r := httptest.NewRequest(http.MethodPost, "/api/widgets", bytes.NewReader(body))
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	createNamedEntity(ops, w, r)

	require.Equal(t, http.StatusConflict, w.Code)
}

func TestCreateUserOwnedNamedEntity_GenericCreateError(t *testing.T) {
	ops := makeTestUserOwnedNamedEntityOps(t)
	ops.create = func(_ context.Context, _ testEntityRequest) (*testEntity, error) {
		return nil, errors.New("unexpected db error")
	}

	body := mustMarshal(t, testEntityRequest{Name: "Widgets Inc"})
	r := httptest.NewRequest(http.MethodPost, "/api/widgets", bytes.NewReader(body))
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	createNamedEntity(ops, w, r)

	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestCreateUserOwnedNamedEntity_NilEntityWithoutError(t *testing.T) {
	ops := makeTestUserOwnedNamedEntityOps(t)
	ops.create = func(_ context.Context, _ testEntityRequest) (*testEntity, error) {
		return nil, nil
	}

	body := mustMarshal(t, testEntityRequest{Name: "Widget"})
	r := httptest.NewRequest(http.MethodPost, "/api/widgets", bytes.NewReader(body))
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	createNamedEntity(ops, w, r)

	require.Equal(t, http.StatusInternalServerError, w.Code)
}

// ---- getNamedEntity with user-scoped closures ----

func TestGetUserOwnedNamedEntity_Success(t *testing.T) {
	ops := makeTestUserOwnedNamedEntityOps(t)

	var capturedUserID string
	ops.get = func(ctx context.Context, id string) (*testEntity, error) {
		capturedUserID = auth.UserIDFromContext(ctx)
		return &testEntity{ID: id, Name: "Existing Widget"}, nil
	}

	r := httptest.NewRequest(http.MethodGet, "/api/widgets/entity-1", nil)
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	getNamedEntity(ops, w, r, "entity-1")

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "user-1", capturedUserID)

	var dto testEntityDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &dto), "unmarshal")
	require.Equal(t, "Existing Widget", dto.Name)
	require.Equal(t, "entity-1", dto.ID)
}

func TestGetUserOwnedNamedEntity_NotFound(t *testing.T) {
	ops := makeTestUserOwnedNamedEntityOps(t)
	// ops.get already returns sql.ErrNoRows by default

	r := httptest.NewRequest(http.MethodGet, "/api/widgets/missing", nil)
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	getNamedEntity(ops, w, r, "missing")

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetUserOwnedNamedEntity_DBError(t *testing.T) {
	ops := makeTestUserOwnedNamedEntityOps(t)
	ops.get = func(_ context.Context, _ string) (*testEntity, error) {
		return nil, errors.New("connection refused")
	}

	r := httptest.NewRequest(http.MethodGet, "/api/widgets/any", nil)
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	getNamedEntity(ops, w, r, "any")

	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestGetUserOwnedNamedEntity_NilEntityWithoutError(t *testing.T) {
	ops := makeTestUserOwnedNamedEntityOps(t)
	ops.get = func(_ context.Context, _ string) (*testEntity, error) {
		return nil, nil
	}

	r := httptest.NewRequest(http.MethodGet, "/api/widgets/any", nil)
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	getNamedEntity(ops, w, r, "any")

	require.Equal(t, http.StatusInternalServerError, w.Code)
}

// ---- updateNamedEntity with user-scoped closures ----

func TestUpdateUserOwnedNamedEntity_Success(t *testing.T) {
	ops := makeTestUserOwnedNamedEntityOps(t)

	var capturedUserID string
	ops.update = func(ctx context.Context, id string, req testEntityRequest) (*testEntity, error) {
		capturedUserID = auth.UserIDFromContext(ctx)
		return &testEntity{ID: id, Name: req.Name}, nil
	}

	body := mustMarshal(t, testEntityRequest{Name: "Updated Widget"})
	r := httptest.NewRequest(http.MethodPut, "/api/widgets/entity-1", bytes.NewReader(body))
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	updateNamedEntity(ops, w, r, "entity-1")

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "user-1", capturedUserID)

	var dto testEntityDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &dto), "unmarshal")
	require.Equal(t, "Updated Widget", dto.Name)
}

func TestUpdateUserOwnedNamedEntity_InvalidJSON(t *testing.T) {
	ops := makeTestUserOwnedNamedEntityOps(t)

	r := httptest.NewRequest(http.MethodPut, "/api/widgets/entity-1", strings.NewReader("not-json"))
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	updateNamedEntity(ops, w, r, "entity-1")

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateUserOwnedNamedEntity_EmptyName(t *testing.T) {
	ops := makeTestUserOwnedNamedEntityOps(t)

	body := mustMarshal(t, testEntityRequest{Name: ""})
	r := httptest.NewRequest(http.MethodPut, "/api/widgets/entity-1", bytes.NewReader(body))
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	updateNamedEntity(ops, w, r, "entity-1")

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateUserOwnedNamedEntity_NotFound(t *testing.T) {
	ops := makeTestUserOwnedNamedEntityOps(t)
	ops.update = func(_ context.Context, _ string, _ testEntityRequest) (*testEntity, error) {
		return nil, sql.ErrNoRows
	}

	body := mustMarshal(t, testEntityRequest{Name: "New Name"})
	r := httptest.NewRequest(http.MethodPut, "/api/widgets/missing", bytes.NewReader(body))
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	updateNamedEntity(ops, w, r, "missing")

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestUpdateUserOwnedNamedEntity_ErrInvalidName(t *testing.T) {
	ops := makeTestUserOwnedNamedEntityOps(t)
	ops.update = func(_ context.Context, _ string, _ testEntityRequest) (*testEntity, error) {
		return nil, errInvalidWidgetName
	}

	body := mustMarshal(t, testEntityRequest{Name: "Anything"})
	r := httptest.NewRequest(http.MethodPut, "/api/widgets/entity-1", bytes.NewReader(body))
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	updateNamedEntity(ops, w, r, "entity-1")

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateUserOwnedNamedEntity_ErrNameExists(t *testing.T) {
	ops := makeTestUserOwnedNamedEntityOps(t)
	ops.update = func(_ context.Context, _ string, _ testEntityRequest) (*testEntity, error) {
		return nil, errWidgetNameExists
	}

	body := mustMarshal(t, testEntityRequest{Name: "Duplicate"})
	r := httptest.NewRequest(http.MethodPut, "/api/widgets/entity-1", bytes.NewReader(body))
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	updateNamedEntity(ops, w, r, "entity-1")

	require.Equal(t, http.StatusConflict, w.Code)
}

func TestUpdateUserOwnedNamedEntity_GenericUpdateError(t *testing.T) {
	ops := makeTestUserOwnedNamedEntityOps(t)
	ops.update = func(_ context.Context, _ string, _ testEntityRequest) (*testEntity, error) {
		return nil, errors.New("unexpected db failure")
	}

	body := mustMarshal(t, testEntityRequest{Name: "Widget Name"})
	r := httptest.NewRequest(http.MethodPut, "/api/widgets/entity-1", bytes.NewReader(body))
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	updateNamedEntity(ops, w, r, "entity-1")

	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestUpdateUserOwnedNamedEntity_NilEntityWithoutError(t *testing.T) {
	ops := makeTestUserOwnedNamedEntityOps(t)
	ops.update = func(_ context.Context, _ string, _ testEntityRequest) (*testEntity, error) {
		return nil, nil
	}

	body := mustMarshal(t, testEntityRequest{Name: "Widget"})
	r := httptest.NewRequest(http.MethodPut, "/api/widgets/entity-1", bytes.NewReader(body))
	r = withUserID(r, "user-1")
	w := httptest.NewRecorder()

	updateNamedEntity(ops, w, r, "entity-1")

	require.Equal(t, http.StatusInternalServerError, w.Code)
}
