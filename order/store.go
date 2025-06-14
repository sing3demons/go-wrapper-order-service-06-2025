package order

import (
	"database/sql"
	"encoding/json"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	commonlog "github.com/sing3demons/go-order-service/pkg/common-log"
	"github.com/sing3demons/go-order-service/pkg/common-log/logAction"
	"github.com/sing3demons/go-order-service/pkg/router"
)

type StoreOrder interface {
	Create(ctx *router.Context, order *Order) (*Order, error)
	GetAll(ctx *router.Context) ([]Order, error)
	GetByID(ctx *router.Context, id uuid.UUID) (*Order, error)
	Update(ctx *router.Context, order *Order) (*Order, error)
	Delete(ctx *router.Context, id string) error
}

type store struct{ db *sql.DB }

const createTable = `CREATE TABLE IF NOT EXISTS orders
(
    id          UUID        not null primary key,
    customer_id     UUID        not null,
    products    varchar[]   not null,
    status      varchar(10) not null,
    created_at  TIMESTAMP   not null,
    updated_at  TIMESTAMP   not null,
    deleted_at  TIMESTAMP
);`

// New is a factory function for store layer.
func New(db *sql.DB) StoreOrder {
	db.Exec(createTable) // Ensure the table exists
	return store{db: db}
}

func (s store) Create(ctx *router.Context, order *Order) (*Order, error) {
	start := time.Now()
	summary := commonlog.LogEventTag{
		Node:        "postgres",
		Command:     "create_order",
		Code:        "20000",
		Description: "success",
	}

	if order.ID == "" {
		order.ID = uuid.NewString()
	}

	createdAt := start.UTC()

	if order.CreatedAt.IsZero() {
		order.CreatedAt = createdAt
	}
	if order.UpdatedAt.IsZero() {
		order.UpdatedAt = createdAt
	}
	query := "INSERT INTO orders (id, customer_id, products, status, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6)"
	params := []any{
		order.ID,
		order.CustomerID,
		pq.Array(&order.Products),
		order.Status,
		order.CreatedAt,
		order.UpdatedAt,
	}

	rawData := strings.ReplaceAll(query, "$1", order.ID)
	rawData = strings.ReplaceAll(rawData, "$2", order.CustomerID)
	b, _ := json.Marshal(order.Products)
	rawData = strings.ReplaceAll(rawData, "$3", string(b))
	rawData = strings.ReplaceAll(rawData, "$4", order.Status)
	rawData = strings.ReplaceAll(rawData, "$5", order.CreatedAt.Format(time.RFC3339))
	rawData = strings.ReplaceAll(rawData, "$6", order.UpdatedAt.Format(time.RFC3339))

	ctx.Log.Info(logAction.DB_REQUEST(logAction.DB_CREATE, summary.Command), map[string]any{
		"table":    "orders",
		"query":    query,
		"params":   params,
		"raw_data": rawData,
	})

	r, err := s.db.Exec(query, params...)
	end := time.Since(start)
	summary.ResTime = end.Microseconds()

	if err != nil {
		summary.Code = "50000"
		summary.Description = err.Error()
		ctx.Log.SetSummary(summary).Error(logAction.DB_RESPONSE(logAction.DB_CREATE, summary.Command), err.Error())
		return nil, err
	}

	rowsAffected, _ := r.RowsAffected()

	ctx.Log.SetSummary(summary).Info(logAction.DB_RESPONSE(logAction.DB_CREATE, summary.Command), map[string]any{
		"rows_affected": rowsAffected,
		"order_id":      order.ID,
	})
	return order, nil
}

func (s store) GetAll(ctx *router.Context) ([]Order, error) {
	query := "SELECT id, customer_id, products, status FROM orders WHERE deleted_at IS NULL"
	summary := commonlog.LogEventTag{
		Node:        "postgres",
		Command:     "get_all_orders",
		Code:        "20000",
		Description: "success",
	}
	start := time.Now()
	ctx.Log.Info(logAction.DB_REQUEST(logAction.DB_READ, summary.Command), map[string]any{
		"table":    "orders",
		"query":    query,
		"raw_data": query,
	})
	rows, err := s.db.Query(query)
	if err != nil {
		summary.Code = "50000"
		summary.Description = err.Error()
		ctx.Log.SetSummary(summary).Error(logAction.DB_RESPONSE(logAction.DB_READ, summary.Command), err.Error())
		return nil, err
	}
	defer rows.Close()
	var orders []Order
	for rows.Next() {
		var order Order
		if err := rows.Scan(&order.ID, &order.CustomerID, pq.Array(&order.Products), &order.Status); err != nil {
			summary.Code = "50000"
			summary.Description = err.Error()
			ctx.Log.SetSummary(summary).Error(logAction.DB_RESPONSE(logAction.DB_READ, summary.Command), err.Error())
			return nil, err
		}

		order.Href = os.Getenv("HOST_NAME") + "/orders/" + order.ID
		orders = append(orders, order)
	}
	summary.ResTime = time.Since(start).Microseconds()
	if err := rows.Err(); err != nil {
		summary.Code = "50000"
		summary.Description = err.Error()
		ctx.Log.SetSummary(summary).Error(logAction.DB_RESPONSE(logAction.DB_READ, summary.Command), err.Error())
		return nil, err
	}

	ctx.Log.SetSummary(summary).Info(logAction.DB_RESPONSE(logAction.DB_READ, summary.Command), map[string]any{
		"rows_count": len(orders),
		"data":       orders,
	})

	return orders, nil
}

func (s store) parseToUUID(id string) uuid.UUID {
	parsedID, err := uuid.Parse(id)
	if err != nil {
		return uuid.Nil
	}
	return parsedID
}
func (s store) GetByID(ctx *router.Context, id uuid.UUID) (*Order, error) {
	query := "SELECT id, customer_id, products, status FROM orders WHERE id=$1 and deleted_at IS NULL"
	summary := commonlog.LogEventTag{
		Node:        "postgres",
		Command:     "get_order_by_id",
		Code:        "20000",
		Description: "success",
	}
	start := time.Now()
	ctx.Log.Info(logAction.DB_REQUEST(logAction.DB_READ, summary.Command), map[string]any{
		"table":  "orders",
		"query":  query,
		"params": id,
		// "raw_data": strings.Replace(query, "$1", id, -1),
	})

	row := s.db.QueryRow(query, id)
	var order Order
	if err := row.Scan(&order.ID, &order.CustomerID, pq.Array(&order.Products), &order.Status); err != nil {
		if err == sql.ErrNoRows {
			summary.Code = "40400"
			summary.Description = "data_not_found"
			summary.ResTime = time.Since(start).Microseconds()
			ctx.Log.SetSummary(summary).Error(logAction.DB_RESPONSE(logAction.DB_READ, summary.Command), err.Error())
			return &order, nil
		}
		summary.Code = "50000"
		summary.Description = err.Error()
		ctx.Log.SetSummary(summary).Error(logAction.DB_RESPONSE(logAction.DB_READ, summary.Command), err.Error())
		return nil, err // Return error for other issues
	}
	summary.ResTime = time.Since(start).Microseconds()
	ctx.Log.SetSummary(summary).Info(logAction.DB_RESPONSE(logAction.DB_READ, summary.Command), map[string]any{
		"order": order,
	})
	order.Href = os.Getenv("HOST_NAME") + "/orders/" + order.ID
	return &order, nil
}
func (s store) Update(ctx *router.Context, order *Order) (*Order, error) {
	start := time.Now()
	summary := commonlog.LogEventTag{
		Node:        "postgres",
		Command:     "update_order",
		Code:        "20000",
		Description: "success",
	}
	query := "UPDATE orders SET customer_id=$1, products=$2, status=$3, updated_at=$4 WHERE id=$5 and deleted_at IS NULL"
	params := []any{
		order.CustomerID,
		pq.Array(order.Products),
		order.Status,
		start.UTC(),
		order.ID,
	}

	rawData := strings.ReplaceAll(query, "$1", order.CustomerID)
	b, _ := json.Marshal(order.Products)
	rawData = strings.ReplaceAll(rawData, "$2", string(b))
	rawData = strings.ReplaceAll(rawData, "$3", order.Status)
	rawData = strings.ReplaceAll(rawData, "$4", start.UTC().Format(time.RFC3339))
	rawData = strings.ReplaceAll(rawData, "$5", order.ID)
	ctx.Log.Info(logAction.DB_REQUEST(logAction.DB_UPDATE, summary.Command), map[string]any{
		"table":    "orders",
		"query":    query,
		"params":   params,
		"raw_data": rawData,
	})
	r, err := s.db.Exec(query, params...)
	if err != nil {
		summary.Code = "50000"
		summary.Description = err.Error()
		summary.ResTime = time.Since(start).Microseconds()
		ctx.Log.SetSummary(summary).Error(logAction.DB_RESPONSE(logAction.DB_UPDATE, summary.Command), err.Error())
		return nil, err
	}
	rowsAffected, _ := r.RowsAffected()
	summary.ResTime = time.Since(start).Microseconds()
	ctx.Log.SetSummary(summary).Info(logAction.DB_RESPONSE(logAction.DB_UPDATE, summary.Command), map[string]any{
		"rows_affected": rowsAffected,
		"order_id":      order.ID,
	})
	return nil, nil
}
func (s store) Delete(ctx *router.Context, id string) error {
	query := "UPDATE orders SET deleted_at=$1 WHERE id=$2"
	summary := commonlog.LogEventTag{
		Node:        "postgres",
		Command:     "delete_order",
		Code:        "20000",
		Description: "success",
	}
	start := time.Now()
	ctx.Log.Info(logAction.DB_REQUEST(logAction.DB_DELETE, summary.Command), map[string]any{
		"table":    "orders",
		"query":    query,
		"params":   id,
		"raw_data": strings.Replace(query, "$2", id, -1),
	})
	r, err := s.db.Exec(query, start.UTC(), id)
	if err != nil {
		summary.Code = "50000"
		summary.Description = err.Error()
		summary.ResTime = time.Since(start).Microseconds()
		ctx.Log.SetSummary(summary).Error(logAction.DB_RESPONSE(logAction.DB_DELETE, summary.Command), err.Error())
		return err
	}

	rowsAffected, _ := r.RowsAffected()

	summary.ResTime = time.Since(start).Microseconds()
	ctx.Log.SetSummary(summary).Info(logAction.DB_RESPONSE(logAction.DB_DELETE, summary.Command), map[string]any{
		"order_id":      id,
		"rows_affected": rowsAffected,
	})
	return nil
}
func (s store) GetByCustomerID(ctx *router.Context, customerID string) ([]Order, error) {
	query := "SELECT id, customer_id, products, status FROM orders WHERE customer_id=$1 and deleted_at IS NULL"
	summary := commonlog.LogEventTag{
		Node:        "postgres",
		Command:     "get_orders_by_customer_id",
		Code:        "20000",
		Description: "success",
	}
	start := time.Now()
	ctx.Log.Info(logAction.DB_REQUEST(logAction.DB_READ, summary.Command), map[string]any{
		"table":    "orders",
		"query":    query,
		"params":   customerID,
		"raw_data": strings.Replace(query, "$1", customerID, -1),
	})
	rows, err := s.db.Query(query, customerID)
	if err != nil {
		summary.Code = "50000"
		summary.Description = err.Error()
		summary.ResTime = time.Since(start).Microseconds()
		ctx.Log.SetSummary(summary).Error(logAction.DB_RESPONSE(logAction.DB_READ, summary.Command), err.Error())
		return nil, err
	}
	defer rows.Close()
	var orders []Order
	for rows.Next() {
		var order Order
		if err := rows.Scan(&order.ID, &order.CustomerID, pq.Array(&order.Products), &order.Status); err != nil {
			summary.Code = "50000"
			summary.Description = err.Error()
			summary.ResTime = time.Since(start).Microseconds()
			ctx.Log.SetSummary(summary).Error(logAction.DB_RESPONSE(logAction.DB_READ, summary.Command), err.Error())
			return nil, err
		}
		order.Href = os.Getenv("HOST_NAME") + "/orders/" + order.ID
		orders = append(orders, order)
	}
	summary.ResTime = time.Since(start).Microseconds()
	if err := rows.Err(); err != nil {
		summary.Code = "50000"
		summary.Description = err.Error()
		summary.ResTime = time.Since(start).Microseconds()
		ctx.Log.SetSummary(summary).Error(logAction.DB_RESPONSE(logAction.DB_READ, summary.Command), err.Error())
		return nil, err
	}
	ctx.Log.SetSummary(summary).Info(logAction.DB_RESPONSE(logAction.DB_READ, summary.Command), map[string]any{
		"rows_count": len(orders),
		"data":       orders,
	})
	return orders, nil
}
