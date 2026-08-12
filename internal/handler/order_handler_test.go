package handler_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"delivery-service/internal/handler"
	"delivery-service/internal/model"
	"delivery-service/internal/repository"
	"delivery-service/internal/service"
)

// ---------- Mocks ----------

type mockRepo struct {
	createFn    func(distance int) (*model.Order, error)
	takeOrderFn func(id int64) error
	listFn      func(page, limit int) ([]model.Order, error)
}

func (m *mockRepo) Create(distance int) (*model.Order, error) {
	return m.createFn(distance)
}

func (m *mockRepo) TakeOrder(id int64) error {
	return m.takeOrderFn(id)
}

func (m *mockRepo) List(page, limit int) ([]model.Order, error) {
	return m.listFn(page, limit)
}

type mockDistance struct {
	getDistanceFn func(origin, destination []string) (int, error)
}

func (m *mockDistance) GetDistance(origin, destination []string) (int, error) {
	return m.getDistanceFn(origin, destination)
}

// ---------- Helpers ----------

func newTestHandler(repo repository.OrderRepositoryInterface, dist service.DistanceCalculator) *handler.OrderHandler {
	return &handler.OrderHandler{
		Repo:            repo,
		DistanceService: dist,
	}
}

func doRequest(h *handler.OrderHandler, method, path, body string) *httptest.ResponseRecorder {
	var req *http.Request
	if body != "" {
		req = httptest.NewRequest(method, path, bytes.NewBufferString(body))
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	req.SetPathValue("id", "1")
	w := httptest.NewRecorder()

	switch method {
	case http.MethodPost:
		h.PlaceOrder(w, req)
	case http.MethodPatch:
		h.TakeOrder(w, req)
	case http.MethodGet:
		h.ListOrders(w, req)
	}
	return w
}

// ---------- Method Not Allowed Tests ----------

func TestPlaceOrder_MethodNotAllowed(t *testing.T) {
	repo := &mockRepo{}
	dist := &mockDistance{}
	h := newTestHandler(repo, dist)

	req := httptest.NewRequest(http.MethodGet, "/orders", nil)
	w := httptest.NewRecorder()
	h.PlaceOrder(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}

	var resp model.ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Error != "METHOD_NOT_ALLOWED" {
		t.Errorf("expected METHOD_NOT_ALLOWED, got %s", resp.Error)
	}
}

func TestTakeOrder_MethodNotAllowed(t *testing.T) {
	repo := &mockRepo{}
	dist := &mockDistance{}
	h := newTestHandler(repo, dist)

	req := httptest.NewRequest(http.MethodGet, "/orders/1", nil)
	req.SetPathValue("id", "1")
	w := httptest.NewRecorder()
	h.TakeOrder(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestListOrders_MethodNotAllowed(t *testing.T) {
	repo := &mockRepo{}
	dist := &mockDistance{}
	h := newTestHandler(repo, dist)

	req := httptest.NewRequest(http.MethodPost, "/orders", nil)
	w := httptest.NewRecorder()
	h.ListOrders(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

// ---------- PlaceOrder Tests ----------

func TestPlaceOrder_Success(t *testing.T) {
	repo := &mockRepo{
		createFn: func(distance int) (*model.Order, error) {
			return &model.Order{ID: 1, Distance: distance, Status: "UNASSIGNED"}, nil
		},
	}
	dist := &mockDistance{
		getDistanceFn: func(origin, destination []string) (int, error) {
			return 12345, nil
		},
	}
	h := newTestHandler(repo, dist)

	body := `{"origin":["22.3193","114.1694"],"destination":["22.3964","114.1095"]}`
	w := doRequest(h, http.MethodPost, "/orders", body)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp model.Order
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.ID != 1 || resp.Distance != 12345 || resp.Status != "UNASSIGNED" {
		t.Errorf("unexpected response: %+v", resp)
	}
}

func TestPlaceOrder_InvalidCoordinates(t *testing.T) {
	repo := &mockRepo{}
	dist := &mockDistance{}
	h := newTestHandler(repo, dist)

	// Coordinates must be exactly two strings
	body := `{"origin":["22.3193"],"destination":["22.3964","114.1095"]}`
	w := doRequest(h, http.MethodPost, "/orders", body)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}

	var resp model.ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Error != "INVALID_COORDINATES" {
		t.Errorf("expected INVALID_COORDINATES, got %s", resp.Error)
	}
}

func TestPlaceOrder_CoordinatesMustBeStrings(t *testing.T) {
	repo := &mockRepo{}
	dist := &mockDistance{}
	h := newTestHandler(repo, dist)

	body := `{"origin":[22.3193,114.1694],"destination":["22.3964","114.1095"]}`
	w := doRequest(h, http.MethodPost, "/orders", body)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestPlaceOrder_OutOfRangeLatitude(t *testing.T) {
	repo := &mockRepo{}
	dist := &mockDistance{}
	h := newTestHandler(repo, dist)

	body := `{"origin":["91.0","114.1694"],"destination":["22.3964","114.1095"]}`
	w := doRequest(h, http.MethodPost, "/orders", body)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestPlaceOrder_OutOfRangeLongitude(t *testing.T) {
	repo := &mockRepo{}
	dist := &mockDistance{}
	h := newTestHandler(repo, dist)

	body := `{"origin":["22.3193","181.0"],"destination":["22.3964","114.1095"]}`
	w := doRequest(h, http.MethodPost, "/orders", body)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestPlaceOrder_InvalidRequestBody(t *testing.T) {
	repo := &mockRepo{}
	dist := &mockDistance{}
	h := newTestHandler(repo, dist)

	body := `{invalid json`
	w := doRequest(h, http.MethodPost, "/orders", body)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestPlaceOrder_DistanceServiceError(t *testing.T) {
	repo := &mockRepo{}
	dist := &mockDistance{
		getDistanceFn: func(origin, destination []string) (int, error) {
			return 0, errors.New("GOOGLE_MAPS_API_KEY environment variable is missing")
		},
	}
	h := newTestHandler(repo, dist)

	body := `{"origin":["22.3193","114.1694"],"destination":["22.3964","114.1095"]}`
	w := doRequest(h, http.MethodPost, "/orders", body)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestPlaceOrder_DBError(t *testing.T) {
	repo := &mockRepo{
		createFn: func(distance int) (*model.Order, error) {
			return nil, errors.New("db down")
		},
	}
	dist := &mockDistance{
		getDistanceFn: func(origin, destination []string) (int, error) {
			return 100, nil
		},
	}
	h := newTestHandler(repo, dist)

	body := `{"origin":["22.3193","114.1694"],"destination":["22.3964","114.1095"]}`
	w := doRequest(h, http.MethodPost, "/orders", body)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

// ---------- TakeOrder Tests ----------

func TestTakeOrder_Success(t *testing.T) {
	repo := &mockRepo{
		takeOrderFn: func(id int64) error {
			return nil
		},
	}
	dist := &mockDistance{}
	h := newTestHandler(repo, dist)

	body := `{"status":"TAKEN"}`
	w := doRequest(h, http.MethodPatch, "/orders/1", body)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp model.TakeOrderResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Status != "SUCCESS" {
		t.Errorf("expected SUCCESS, got %s", resp.Status)
	}
}

func TestTakeOrder_AlreadyTaken(t *testing.T) {
	repo := &mockRepo{
		takeOrderFn: func(id int64) error {
			return repository.ErrOrderAlreadyTaken
		},
	}
	dist := &mockDistance{}
	h := newTestHandler(repo, dist)

	body := `{"status":"TAKEN"}`
	w := doRequest(h, http.MethodPatch, "/orders/1", body)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", w.Code)
	}

	var resp model.ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Error != "ORDER_ALREADY_TAKEN" {
		t.Errorf("expected ORDER_ALREADY_TAKEN, got %s", resp.Error)
	}
}

func TestTakeOrder_NotFound(t *testing.T) {
	repo := &mockRepo{
		takeOrderFn: func(id int64) error {
			return repository.ErrOrderNotFound
		},
	}
	dist := &mockDistance{}
	h := newTestHandler(repo, dist)

	body := `{"status":"TAKEN"}`
	w := doRequest(h, http.MethodPatch, "/orders/999", body)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestTakeOrder_InvalidStatus(t *testing.T) {
	repo := &mockRepo{}
	dist := &mockDistance{}
	h := newTestHandler(repo, dist)

	body := `{"status":"UNASSIGNED"}`
	w := doRequest(h, http.MethodPatch, "/orders/1", body)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestTakeOrder_InvalidOrderID(t *testing.T) {
	repo := &mockRepo{}
	dist := &mockDistance{}
	h := newTestHandler(repo, dist)

	body := `{"status":"TAKEN"}`
	req := httptest.NewRequest(http.MethodPatch, "/orders/abc", bytes.NewBufferString(body))
	req.SetPathValue("id", "abc")
	w := httptest.NewRecorder()
	h.TakeOrder(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

// ---------- ListOrders Tests ----------

func TestListOrders_Success(t *testing.T) {
	repo := &mockRepo{
		listFn: func(page, limit int) ([]model.Order, error) {
			return []model.Order{
				{ID: 1, Distance: 100, Status: "UNASSIGNED"},
				{ID: 2, Distance: 200, Status: "TAKEN"},
			}, nil
		},
	}
	dist := &mockDistance{}
	h := newTestHandler(repo, dist)

	w := doRequest(h, http.MethodGet, "/orders?page=1&limit=10", "")

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp []model.Order
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(resp) != 2 {
		t.Errorf("expected 2 orders, got %d", len(resp))
	}
}

func TestListOrders_EmptyResult(t *testing.T) {
	repo := &mockRepo{
		listFn: func(page, limit int) ([]model.Order, error) {
			return []model.Order{}, nil
		},
	}
	dist := &mockDistance{}
	h := newTestHandler(repo, dist)

	w := doRequest(h, http.MethodGet, "/orders?page=1&limit=10", "")

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	// Must be an empty array, not null
	body := w.Body.String()
	if body != "[]\n" && body != "[]" {
		t.Errorf("expected empty array, got %s", body)
	}
}

func TestListOrders_InvalidPage(t *testing.T) {
	repo := &mockRepo{}
	dist := &mockDistance{}
	h := newTestHandler(repo, dist)

	// page=0 is invalid (must start at 1)
	w := doRequest(h, http.MethodGet, "/orders?page=0&limit=10", "")

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestListOrders_InvalidLimit(t *testing.T) {
	repo := &mockRepo{}
	dist := &mockDistance{}
	h := newTestHandler(repo, dist)

	w := doRequest(h, http.MethodGet, "/orders?page=1&limit=abc", "")

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestListOrders_MissingParams(t *testing.T) {
	repo := &mockRepo{}
	dist := &mockDistance{}
	h := newTestHandler(repo, dist)

	w := doRequest(h, http.MethodGet, "/orders", "")

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}
