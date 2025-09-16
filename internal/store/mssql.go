package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/denisenkom/go-mssqldb"
	"go.uber.org/zap"
)

type MSSQLStore struct {
	db     *sql.DB
	logger *zap.Logger
}

func NewMSSQLStore(connStr string, logger *zap.Logger) (*MSSQLStore, error) {
	db, err := sql.Open("mssql", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &MSSQLStore{
		db:     db,
		logger: logger,
	}, nil
}

func (s *MSSQLStore) Close() error {
	return s.db.Close()
}

// UpsertUser creates or updates a user record
func (s *MSSQLStore) UpsertUser(ctx context.Context, user *User) error {
	query := `
		IF EXISTS (SELECT 1 FROM users WHERE user_id = ?)
			UPDATE users SET name = ?, email = ?, updated_at = ? WHERE user_id = ?
		ELSE
			INSERT INTO users (user_id, name, email, created_at, updated_at) VALUES (?, ?, ?, ?, ?)
	`

	_, err := s.db.ExecContext(ctx, query,
		user.UserID, user.Name, user.Email, user.UpdatedAt, user.UserID,
		user.UserID, user.Name, user.Email, user.CreatedAt, user.UpdatedAt,
	)

	return err
}

// UpsertOrder creates or updates an order record
func (s *MSSQLStore) UpsertOrder(ctx context.Context, order *Order) error {
	query := `
		IF EXISTS (SELECT 1 FROM orders WHERE order_id = ?)
		BEGIN
			UPDATE orders
			SET user_id = ?,
				total = ?,
				status = ?,
				updated_at = ?
			WHERE order_id = ?
		END
		ELSE
		BEGIN
			INSERT INTO orders (order_id, user_id, total, status, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?)
		END
	`

	_, err := s.db.ExecContext(ctx, query,
		// For IF EXISTS
		order.OrderID,

		// For UPDATE
		order.UserID,
		order.Total,
		order.Status,
		order.UpdatedAt,
		order.OrderID, // WHERE order_id = ?

		// For INSERT
		order.OrderID,
		order.UserID,
		order.Total,
		order.Status,
		order.CreatedAt,
		order.UpdatedAt,
	)

	return err
}

// UpsertPayment creates or updates a payment record
func (s *MSSQLStore) UpsertPayment(ctx context.Context, payment *Payment) error {
	query := `
		IF EXISTS (SELECT 1 FROM payments WHERE order_id = ?)
		BEGIN
			UPDATE payments
			SET status = ?,
				amount = ?,
				settled_at = ?,
				updated_at = ?
			WHERE order_id = ?
		END
		ELSE
		BEGIN
			INSERT INTO payments (order_id, status, amount, settled_at, updated_at)
			VALUES (?, ?, ?, ?, ?)
		END
	`

	_, err := s.db.ExecContext(ctx, query,
		// For IF EXISTS
		payment.OrderID,

		// For UPDATE
		payment.Status,
		payment.Amount,
		payment.SettledAt,
		payment.UpdatedAt,
		payment.OrderID, // WHERE order_id = ?

		// For INSERT
		payment.OrderID,
		payment.Status,
		payment.Amount,
		payment.SettledAt,
		payment.UpdatedAt,
	)

	return err
}

// UpsertInventory creates or updates an inventory record
func (s *MSSQLStore) UpsertInventory(ctx context.Context, inventory *Inventory) error {
	query := `
		IF EXISTS (SELECT 1 FROM inventory WHERE sku = ?)
		BEGIN
			UPDATE inventory
			SET quantity = quantity + ?,
				last_adjusted_at = ?
			WHERE sku = ?
		END
		ELSE
		BEGIN
			INSERT INTO inventory (sku, quantity, last_adjusted_at)
			VALUES (?, ?, ?)
		END
	`

	_, err := s.db.ExecContext(ctx, query,
		// For IF EXISTS
		inventory.SKU,

		// For UPDATE
		inventory.Quantity,
		inventory.LastAdjustedAt,
		inventory.SKU, // WHERE sku = ?

		// For INSERT
		inventory.SKU,
		inventory.Quantity,
		inventory.LastAdjustedAt,
	)

	return err
}

// GetUser retrieves a user by ID
func (s *MSSQLStore) GetUser(ctx context.Context, userID string) (*User, error) {
	query := `SELECT user_id, name, email, created_at, updated_at FROM users WHERE user_id = ?`

	row := s.db.QueryRowContext(ctx, query, userID)

	user := &User{}
	err := row.Scan(&user.UserID, &user.Name, &user.Email, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return user, nil
}

// GetUserRecentOrders retrieves the last 5 orders for a user
func (s *MSSQLStore) GetUserRecentOrders(ctx context.Context, userID string) ([]*Order, error) {
	query := `
		SELECT TOP 5 order_id, user_id, total, status, created_at, updated_at 
		FROM orders 
		WHERE user_id = ? 
		ORDER BY created_at DESC
	`

	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []*Order
	for rows.Next() {
		order := &Order{}
		err := rows.Scan(&order.OrderID, &order.UserID, &order.Total, &order.Status, &order.CreatedAt, &order.UpdatedAt)
		if err != nil {
			return nil, err
		}
		orders = append(orders, order)
	}

	return orders, nil
}

// GetOrder retrieves an order by ID
func (s *MSSQLStore) GetOrder(ctx context.Context, orderID string) (*Order, error) {
	query := `SELECT order_id, user_id, total, status, created_at, updated_at FROM orders WHERE order_id = ?`

	row := s.db.QueryRowContext(ctx, query, orderID)

	order := &Order{}
	err := row.Scan(&order.OrderID, &order.UserID, &order.Total, &order.Status, &order.CreatedAt, &order.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return order, nil
}

// GetPayment retrieves a payment by order ID
func (s *MSSQLStore) GetPayment(ctx context.Context, orderID string) (*Payment, error) {
	query := `SELECT order_id, status, amount, settled_at, updated_at FROM payments WHERE order_id = ?`

	row := s.db.QueryRowContext(ctx, query, orderID)

	payment := &Payment{}
	err := row.Scan(&payment.OrderID, &payment.Status, &payment.Amount, &payment.SettledAt, &payment.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return payment, nil
}

func (s *MSSQLStore) UpsertProductReview(ctx context.Context, review *ProductReview) error {
	query := `
		IF EXISTS (SELECT 1 FROM product_reviews WHERE review_id = ?)
			UPDATE product_reviews 
			SET product_name = ?, username = ?, rating = ?, remarks = ?, updated_at = ? 
			WHERE review_id = ?
		ELSE
			INSERT INTO product_reviews (review_id, product_name, username, rating, remarks, created_at, updated_at) 
			VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	_, err := s.db.ExecContext(ctx, query,
		review.ReviewID, review.ProductName, review.Username, review.Rating, review.Remarks, review.UpdatedAt, review.ReviewID,
		review.ReviewID, review.ProductName, review.Username, review.Rating, review.Remarks, review.CreatedAt, review.UpdatedAt,
	)
	return err
}

// GetProductReview retrieves a product review by ID
func (s *MSSQLStore) GetProductReview(ctx context.Context, reviewID string) (*ProductReview, error) {
	query := `SELECT review_id, product_name, username, rating, remarks, created_at, updated_at FROM product_reviews WHERE review_id = ?`

	row := s.db.QueryRowContext(ctx, query, reviewID)

	review := &ProductReview{}
	err := row.Scan(&review.ReviewID, &review.ProductName, &review.Username, &review.Rating, &review.Remarks, &review.CreatedAt, &review.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return review, nil
}

// GetProductReviewsByProduct retrieves all reviews for a specific product
func (s *MSSQLStore) GetProductReviewsByProduct(ctx context.Context, productName string) ([]*ProductReview, error) {
	query := `SELECT review_id, product_name, username, rating, remarks, created_at, updated_at FROM product_reviews WHERE product_name = ? ORDER BY created_at DESC`

	rows, err := s.db.QueryContext(ctx, query, productName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reviews []*ProductReview
	for rows.Next() {
		review := &ProductReview{}
		err := rows.Scan(&review.ReviewID, &review.ProductName, &review.Username, &review.Rating, &review.Remarks, &review.CreatedAt, &review.UpdatedAt)
		if err != nil {
			return nil, err
		}
		reviews = append(reviews, review)
	}

	return reviews, nil
}
