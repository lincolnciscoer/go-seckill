package repository

import (
	"context"
	"database/sql"
	"errors"

	"go-seckill/internal/model"

	mysqlDriver "github.com/go-sql-driver/mysql"
)

var (
	ErrDuplicateOrder = errors.New("duplicate order")
	ErrStockNotEnough = errors.New("stock not enough")
)

type CreateSeckillOrderInput struct {
	OrderNo    string
	UserID     uint64
	ActivityID uint64
	ProductID  uint64
	Quantity   int
	Amount     int64
	Status     int8
}

type OrderRepository interface {
	CreateSeckillOrder(ctx context.Context, input CreateSeckillOrderInput) (*model.Order, error)
	GetByOrderNo(ctx context.Context, orderNo string) (*model.Order, error)
	GetByUserActivity(ctx context.Context, userID uint64, activityID uint64) (*model.Order, error)
	ListByUserID(ctx context.Context, userID uint64) ([]model.Order, error)
}

type SQLOrderRepository struct {
	db *sql.DB
}

func NewSQLOrderRepository(db *sql.DB) *SQLOrderRepository {
	return &SQLOrderRepository{db: db}
}

// CreateSeckillOrder 使用原生 SQL 执行库存扣减和订单创建。
// 这条链路是后续热点写路径的雏形，所以这里刻意不走 GORM，方便你更清楚地看到事务边界和 SQL 行为。
func (r *SQLOrderRepository) CreateSeckillOrder(ctx context.Context, input CreateSeckillOrderInput) (*model.Order, error) {
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	result, err := tx.ExecContext(ctx, `
		UPDATE seckill_stocks
		SET available_stock = available_stock - 1,
		    sold_stock = sold_stock + 1,
		    updated_at = CURRENT_TIMESTAMP(3)
		WHERE activity_id = ? AND available_stock >= 1
	`, input.ActivityID)
	if err != nil {
		return nil, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}
	if rowsAffected == 0 {
		return nil, ErrStockNotEnough
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO orders (
			order_no, user_id, activity_id, product_id, quantity, amount, status, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP(3), CURRENT_TIMESTAMP(3))
	`,
		input.OrderNo,
		input.UserID,
		input.ActivityID,
		input.ProductID,
		input.Quantity,
		input.Amount,
		input.Status,
	)
	if err != nil {
		var mysqlErr *mysqlDriver.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
			return nil, ErrDuplicateOrder
		}
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}

	return r.GetByOrderNo(ctx, input.OrderNo)
}

func (r *SQLOrderRepository) GetByOrderNo(ctx context.Context, orderNo string) (*model.Order, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, order_no, user_id, activity_id, product_id, quantity, amount, status, created_at, updated_at
		FROM orders
		WHERE order_no = ?
	`, orderNo)

	var order model.Order
	err := row.Scan(
		&order.ID,
		&order.OrderNo,
		&order.UserID,
		&order.ActivityID,
		&order.ProductID,
		&order.Quantity,
		&order.Amount,
		&order.Status,
		&order.CreatedAt,
		&order.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &order, nil
}

func (r *SQLOrderRepository) ListByUserID(ctx context.Context, userID uint64) ([]model.Order, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, order_no, user_id, activity_id, product_id, quantity, amount, status, created_at, updated_at
		FROM orders
		WHERE user_id = ?
		ORDER BY id DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	orders := make([]model.Order, 0)
	for rows.Next() {
		var order model.Order
		if err := rows.Scan(
			&order.ID,
			&order.OrderNo,
			&order.UserID,
			&order.ActivityID,
			&order.ProductID,
			&order.Quantity,
			&order.Amount,
			&order.Status,
			&order.CreatedAt,
			&order.UpdatedAt,
		); err != nil {
			return nil, err
		}
		orders = append(orders, order)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return orders, nil
}

func (r *SQLOrderRepository) GetByUserActivity(ctx context.Context, userID uint64, activityID uint64) (*model.Order, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, order_no, user_id, activity_id, product_id, quantity, amount, status, created_at, updated_at
		FROM orders
		WHERE user_id = ? AND activity_id = ?
		LIMIT 1
	`, userID, activityID)

	var order model.Order
	err := row.Scan(
		&order.ID,
		&order.OrderNo,
		&order.UserID,
		&order.ActivityID,
		&order.ProductID,
		&order.Quantity,
		&order.Amount,
		&order.Status,
		&order.CreatedAt,
		&order.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &order, nil
}
