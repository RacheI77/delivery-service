package repository_test

import (
	"database/sql"
	"fmt"
	"os"
	"sync"
	"testing"

	_ "github.com/go-sql-driver/mysql"

	"delivery-service/internal/repository"
)

func newTestDB(t *testing.T) *sql.DB {
	t.Helper()

	host := getEnv("TEST_DB_HOST", "localhost")
	port := getEnv("TEST_DB_PORT", "3306")
	user := getEnv("TEST_DB_USER", "root")
	password := getEnv("TEST_DB_PASSWORD", "root")
	name := getEnv("TEST_DB_NAME", "delivery_db")

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&charset=utf8mb4",
		user, password, host, port, name)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		t.Skipf("skipping integration test: database not reachable: %v", err)
	}

	if _, err := db.Exec("TRUNCATE TABLE orders"); err != nil {
		db.Close()
		t.Fatalf("failed to truncate orders table: %v", err)
	}

	return db
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func TestCreateOrder_Integration(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()

	repo := repository.NewOrderRepository(db)

	order, err := repo.Create(12345)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if order.ID <= 0 {
		t.Errorf("expected a positive id, got %d", order.ID)
	}
	if order.Distance != 12345 {
		t.Errorf("expected distance 12345, got %d", order.Distance)
	}
	if order.Status != "UNASSIGNED" {
		t.Errorf("expected status UNASSIGNED, got %s", order.Status)
	}
}

func TestTakeOrder_Integration(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()

	repo := repository.NewOrderRepository(db)

	order, err := repo.Create(100)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if err := repo.TakeOrder(order.ID); err != nil {
		t.Fatalf("first TakeOrder failed: %v", err)
	}

	if err := repo.TakeOrder(order.ID); err != repository.ErrOrderAlreadyTaken {
		t.Errorf("expected ErrOrderAlreadyTaken, got %v", err)
	}
}

func TestTakeOrder_NotFound_Integration(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()

	repo := repository.NewOrderRepository(db)

	if err := repo.TakeOrder(999999); err != repository.ErrOrderNotFound {
		t.Errorf("expected ErrOrderNotFound, got %v", err)
	}
}

func TestListOrders_Integration(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()

	repo := repository.NewOrderRepository(db)

	// Create 5 orders
	for i := 0; i < 5; i++ {
		if _, err := repo.Create(100 + i); err != nil {
			t.Fatalf("Create failed: %v", err)
		}
	}

	// Page 1, limit 2 -> 2 orders
	orders, err := repo.List(1, 2)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(orders) != 2 {
		t.Errorf("expected 2 orders on page 1, got %d", len(orders))
	}

	// Page 3, limit 2 -> 1 order (the 5th)
	orders, err = repo.List(3, 2)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(orders) != 1 {
		t.Errorf("expected 1 order on page 3, got %d", len(orders))
	}

	// Page 4, limit 2 -> empty
	orders, err = repo.List(4, 2)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(orders) != 0 {
		t.Errorf("expected 0 orders on page 4, got %d", len(orders))
	}
}

func TestTakeOrder_Concurrent_Integration(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()

	repo := repository.NewOrderRepository(db)

	order, err := repo.Create(500)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	const goroutines = 20
	var wg sync.WaitGroup
	successCount := 0
	var mu sync.Mutex

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := repo.TakeOrder(order.ID)
			if err == nil {
				mu.Lock()
				successCount++
				mu.Unlock()
			} else if err != repository.ErrOrderAlreadyTaken {
				t.Errorf("unexpected error: %v", err)
			}
		}()
	}

	wg.Wait()

	if successCount != 1 {
		t.Errorf("expected exactly 1 success, got %d", successCount)
	}
}

