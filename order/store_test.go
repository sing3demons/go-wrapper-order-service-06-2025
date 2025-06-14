package order_test

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/sing3demons/go-order-service/order"
)

func TestCreateOrder(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual), sqlmock.MonitorPingsOption(true))
	if err != nil {
		t.Fatalf("failed to open sqlmock database: %s", err)
	}
	defer db.Close()
	mockDB := order.New(db)

	t.Run("create order success", func(t *testing.T) {
		orderID := "123e4567-e89b-12d3-a456-426614174000"
		customerID := "123e4567-e89b-12d3-a456-426614174001"
		products := []string{"product1", "product2"}
		status := "pending"

		mock.ExpectExec("INSERT INTO orders").
			WithArgs(orderID, customerID, products, status).
			WillReturnResult(sqlmock.NewResult(1, 1))

		mockDB.Create(nil, &order.Order{})

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("there were unfulfilled expectations: %s", err)
		}
	})

}
