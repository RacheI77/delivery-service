package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"delivery-service/internal/logger"
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
	logger.Info("PlaceOrder request received")

	if r.Method != http.MethodPost {
		logger.Error("PlaceOrder method not allowed: %s", r.Method)
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED")
		return
	}

	var req model.PlaceOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Error("PlaceOrder invalid request body: %v", err)
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST_BODY")
		return
	}

	logger.Info("PlaceOrder request body: origin=%v destination=%v", req.Origin, req.Destination)

	if !validateCoordinate(req.Origin) || !validateCoordinate(req.Destination) {
		logger.Error("PlaceOrder invalid coordinates: origin=%v destination=%v", req.Origin, req.Destination)
		writeError(w, http.StatusBadRequest, "INVALID_COORDINATES")
		return
	}

	dist, err := h.DistanceService.GetDistance(req.Origin, req.Destination)
	if err != nil {
		logger.Error("PlaceOrder distance service error: %v", err)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	order, err := h.Repo.Create(dist)
	if err != nil {
		logger.Error("PlaceOrder failed to create order: %v", err)
		writeError(w, http.StatusInternalServerError, "DB_ERROR")
		return
	}

	logger.Info("PlaceOrder success: order id=%d distance=%d", order.ID, order.Distance)
	WriteJSON(w, http.StatusOK, order)
}

func (h *OrderHandler) TakeOrder(w http.ResponseWriter, r *http.Request) {
	logger.Info("TakeOrder request received")

	if r.Method != http.MethodPatch {
		logger.Error("TakeOrder method not allowed: %s", r.Method)
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED")
		return
	}

	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		logger.Error("TakeOrder invalid order id: %s", idStr)
		writeError(w, http.StatusBadRequest, "INVALID_ORDER_ID")
		return
	}

	var req model.TakeOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Status != "TAKEN" {
		logger.Error("TakeOrder invalid status for order id=%d", id)
		writeError(w, http.StatusBadRequest, "INVALID_STATUS")
		return
	}

	err = h.Repo.TakeOrder(id)
	if err == repository.ErrOrderAlreadyTaken {
		logger.Error("TakeOrder order already taken: id=%d", id)
		writeError(w, http.StatusConflict, "ORDER_ALREADY_TAKEN")
		return
	}
	if err == repository.ErrOrderNotFound {
		logger.Error("TakeOrder order not found: id=%d", id)
		writeError(w, http.StatusNotFound, "ORDER_NOT_FOUND")
		return
	}
	if err != nil {
		logger.Error("TakeOrder db error for order id=%d: %v", id, err)
		writeError(w, http.StatusInternalServerError, "DB_ERROR")
		return
	}

	logger.Info("TakeOrder success: order id=%d", id)
	WriteJSON(w, http.StatusOK, model.TakeOrderResponse{Status: "SUCCESS"})
}

func (h *OrderHandler) ListOrders(w http.ResponseWriter, r *http.Request) {
	logger.Info("ListOrders request received")

	if r.Method != http.MethodGet {
		logger.Error("ListOrders method not allowed: %s", r.Method)
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED")
		return
	}

	query := r.URL.Query()
	pageStr := query.Get("page")
	limitStr := query.Get("limit")

	logger.Info("ListOrders request params: page=%s limit=%s", pageStr, limitStr)

	page, err1 := strconv.Atoi(pageStr)
	limit, err2 := strconv.Atoi(limitStr)

	if err1 != nil || err2 != nil || page < 1 || limit < 1 {
		logger.Error("ListOrders invalid pagination parameters: page=%s limit=%s", pageStr, limitStr)
		writeError(w, http.StatusBadRequest, "INVALID_PAGINATION_PARAMETERS")
		return
	}

	orders, err := h.Repo.List(page, limit)
	if err != nil {
		logger.Error("ListOrders db error: %v", err)
		writeError(w, http.StatusInternalServerError, "DB_ERROR")
		return
	}

	logger.Info("ListOrders success: returned %d orders (page=%d limit=%d)", len(orders), page, limit)
	WriteJSON(w, http.StatusOK, orders)
}
