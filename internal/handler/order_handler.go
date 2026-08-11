package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"delivery-service/internal/model"
	"delivery-service/internal/repository"
	"delivery-service/internal/service"
)

type OrderHandler struct {
	Repo            repository.OrderRepositoryInterface
	DistanceService service.DistanceCalculator
}

func WriteJSON(w http.ResponseWriter, code int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, code int, msg string) {
	WriteJSON(w, code, model.ErrorResponse{Error: msg})
}

func validateCoordinate(coord []string) bool {
	if len(coord) != 2 {
		return false
	}
	lat, err1 := strconv.ParseFloat(coord[0], 64)
	lng, err2 := strconv.ParseFloat(coord[1], 64)
	if err1 != nil || err2 != nil {
		return false
	}
	return lat >= -90 && lat <= 90 && lng >= -180 && lng <= 180
}

func (h *OrderHandler) PlaceOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED")
		return
	}

	var req model.PlaceOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST_BODY")
		return
	}

	if !validateCoordinate(req.Origin) || !validateCoordinate(req.Destination) {
		writeError(w, http.StatusBadRequest, "INVALID_COORDINATES")
		return
	}

	dist, err := h.DistanceService.GetDistance(req.Origin, req.Destination)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	order, err := h.Repo.Create(dist)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "DB_ERROR")
		return
	}

	WriteJSON(w, http.StatusOK, order)
}

func (h *OrderHandler) TakeOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED")
		return
	}

	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "INVALID_ORDER_ID")
		return
	}

	var req model.TakeOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Status != "TAKEN" {
		writeError(w, http.StatusBadRequest, "INVALID_STATUS")
		return
	}

	err = h.Repo.TakeOrder(id)
	if err == repository.ErrOrderAlreadyTaken {
		writeError(w, http.StatusConflict, "ORDER_ALREADY_TAKEN")
		return
	}
	if err == repository.ErrOrderNotFound {
		writeError(w, http.StatusNotFound, "ORDER_NOT_FOUND")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "DB_ERROR")
		return
	}

	WriteJSON(w, http.StatusOK, model.TakeOrderResponse{Status: "SUCCESS"})
}

func (h *OrderHandler) ListOrders(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED")
		return
	}

	query := r.URL.Query()
	pageStr := query.Get("page")
	limitStr := query.Get("limit")

	page, err1 := strconv.Atoi(pageStr)
	limit, err2 := strconv.Atoi(limitStr)

	if err1 != nil || err2 != nil || page < 1 || limit < 1 {
		writeError(w, http.StatusBadRequest, "INVALID_PAGINATION_PARAMETERS")
		return
	}

	orders, err := h.Repo.List(page, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "DB_ERROR")
		return
	}

	WriteJSON(w, http.StatusOK, orders)
}
