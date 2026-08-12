package repository

import (
	"database/sql"
	"errors"

	"delivery-service/internal/logger"
	"delivery-service/internal/model"
)

var ErrOrderAlreadyTaken = errors.New("ORDER_ALREADY_TAKEN")
var ErrOrderNotFound = errors.New("ORDER_NOT_FOUND")

type OrderRepositoryInterface interface {
	Create(distance int) (*model.Order, error)
	TakeOrder(id int64) error
	List(page, limit int) ([]model.Order, error)
}

type OrderRepository struct {
	DB *sql.DB
}

var _ OrderRepositoryInterface = (*OrderRepository)(nil)

func NewOrderRepository(db *sql.DB) *OrderRepository {
	logger.Info("order repository created")
	return &OrderRepository{DB: db}
}

func (r *OrderRepository) Create(distance int) (*model.Order, error) {
	logger.Info("Create called with distance=%d", distance)

	res, err := r.DB.Exec("INSERT INTO orders (distance, status) VALUES (?, 'UNASSIGNED')", distance)
	if err != nil {
		logger.Error("failed to insert order with distance=%d: %v", distance, err)
		return nil, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		logger.Error("failed to get last insert id for distance=%d: %v", distance, err)
		return nil, err
	}

	order := &model.Order{ID: id, Distance: distance, Status: "UNASSIGNED"}
	logger.Info("order created: id=%d distance=%d status=UNASSIGNED", order.ID, order.Distance)
	return order, nil
}

func (r *OrderRepository) TakeOrder(id int64) error {
	logger.Info("TakeOrder called with id=%d", id)

	res, err := r.DB.Exec("UPDATE orders SET status = 'TAKEN' WHERE id = ? AND status = 'UNASSIGNED'", id)
	if err != nil {
		logger.Error("failed to update order id=%d: %v", id, err)
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		logger.Error("failed to get rows affected for order id=%d: %v", id, err)
		return err
	}
	if affected == 0 {
		var count int
		if err := r.DB.QueryRow("SELECT COUNT(*) FROM orders WHERE id = ?", id).Scan(&count); err != nil {
			logger.Error("failed to check order existence for id=%d: %v", id, err)
		}
		if count == 0 {
			logger.Error("order not found: id=%d", id)
			return ErrOrderNotFound
		}
		logger.Error("order already taken: id=%d", id)
		return ErrOrderAlreadyTaken
	}

	logger.Info("order taken successfully: id=%d", id)
	return nil
}

func (r *OrderRepository) List(page, limit int) ([]model.Order, error) {
	logger.Info("List called with page=%d limit=%d", page, limit)

	offset := (page - 1) * limit
	rows, err := r.DB.Query("SELECT id, distance, status FROM orders ORDER BY id ASC LIMIT ? OFFSET ?", limit, offset)
	if err != nil {
		logger.Error("failed to query orders (page=%d limit=%d): %v", page, limit, err)
		return nil, err
	}
	defer rows.Close()

	orders := []model.Order{}
	for rows.Next() {
		var o model.Order
		if err := rows.Scan(&o.ID, &o.Distance, &o.Status); err != nil {
			logger.Error("failed to scan order row: %v", err)
			return nil, err
		}
		orders = append(orders, o)
	}
	if err := rows.Err(); err != nil {
		logger.Error("failed to iterate order rows: %v", err)
		return nil, err
	}

	logger.Info("List returned %d orders (page=%d limit=%d)", len(orders), page, limit)
	return orders, nil
}
