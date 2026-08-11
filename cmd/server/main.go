package main

import (
	"database/sql"
	"log"
	"net/http"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"delivery-service/internal/config"
	"delivery-service/internal/handler"
	"delivery-service/internal/model"
	"delivery-service/internal/repository"
	"delivery-service/internal/service"
)

func main() {
	cfg := config.Load()

	db, err := sql.Open("mysql", cfg.DSN())
	if err != nil {
		log.Fatalf("failed to open database connection: %v", err)
	}
	defer db.Close()

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := waitForDB(db); err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	orderRepo := repository.NewOrderRepository(db)
	distanceService := &service.GoogleDistanceService{
		APIKey: cfg.GoogleMapsAPIKey,
	}
	orderHandler := &handler.OrderHandler{
		Repo:            orderRepo,
		DistanceService: distanceService,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/orders", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			orderHandler.PlaceOrder(w, r)
		case http.MethodGet:
			orderHandler.ListOrders(w, r)
		default:
			handler.WriteJSON(w, http.StatusMethodNotAllowed, model.ErrorResponse{Error: "METHOD_NOT_ALLOWED"})
		}
	})
	mux.HandleFunc("/orders/{id}", orderHandler.TakeOrder)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		handler.WriteJSON(w, http.StatusNotFound, model.ErrorResponse{Error: "NOT_FOUND"})
	})

	addr := ":" + cfg.Port

	log.Printf("Delivery service listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}

func waitForDB(db *sql.DB) error {
	var err error
	for i := 0; i < 30; i++ {
		if err = db.Ping(); err == nil {
			return nil
		}
		log.Printf("waiting for database to be ready (%d/30): %v", i+1, err)
		time.Sleep(2 * time.Second)
	}
	return err
}
