package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"delivery-service/internal/handler"
	"delivery-service/internal/model"
	"delivery-service/internal/repository"
	"delivery-service/internal/service"
)

func main() {
	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "3306")
	dbUser := getEnv("DB_USER", "root")
	dbPassword := getEnv("DB_PASSWORD", "root")
	dbName := getEnv("DB_NAME", "delivery_db")

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&charset=utf8mb4",
		dbUser, dbPassword, dbHost, dbPort, dbName)

	db, err := sql.Open("mysql", dsn)
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
	distanceService := &service.GoogleDistanceService{}
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

	port := getEnv("PORT", "8080")
	addr := ":" + port

	log.Printf("Delivery service listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
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
