package repository

import (
	"database/sql"
	"delivery-service/internal/model"
	"errors"
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
	return &OrderRepository{DB: db}
}

func (r *OrderRepository) Create(distance int) (*model.Order, error) {
	res, err := r.DB.Exec("INSERT INTO orders (distance, status) VALUES (?, 'UNASSIGNED')", distance)
	if err != nil {
		return nil, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	return &model.Order{ID: id, Distance: distance, Status: "UNASSIGNED"}, nil
}

func (r *OrderRepository) TakeOrder(id int64) error {
	res, err := r.DB.Exec("UPDATE orders SET status = 'TAKEN' WHERE id = ? AND status = 'UNASSIGNED'", id)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		var count int
		_ = r.DB.QueryRow("SELECT COUNT(*) FROM orders WHERE id = ?", id).Scan(&count)
		if count == 0 {
			return ErrOrderNotFound
		}
		return ErrOrderAlreadyTaken
	}
	return nil
}

func (r *OrderRepository) List(page, limit int) ([]model.Order, error) {
	offset := (page - 1) * limit
	rows, err := r.DB.Query("SELECT id, distance, status FROM orders ORDER BY id ASC LIMIT ? OFFSET ?", limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	orders := []model.Order{}
	for rows.Next() {
		var o model.Order
		if err := rows.Scan(&o.ID, &o.Distance, &o.Status); err != nil {
			return nil, err
		}
		orders = append(orders, o)
	}
	return orders, nil
}
