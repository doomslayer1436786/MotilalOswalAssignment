package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"kafka-pipeline/internal/store"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

var (
	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status"},
	)

	httpLatencySeconds = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_latency_seconds",
			Help:    "HTTP request latency in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint"},
	)
)

func init() {
	prometheus.MustRegister(httpRequestsTotal)
	prometheus.MustRegister(httpLatencySeconds)
}

type APIResponse struct {
	User         *store.User    `json:"user,omitempty"`
	RecentOrders []*store.Order `json:"recentOrders,omitempty"`
	Order        *store.Order   `json:"order,omitempty"`
	Payment      *store.Payment `json:"payment,omitempty"`
	Error        string         `json:"error,omitempty"`
}

func main() {
	// Initialize logger
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatal("Failed to initialize logger:", err)
	}
	defer logger.Sync()

	// Get configuration from environment
	mssqlConn := getEnv("MSSQL_CONN", "server=localhost;user id=sa;password=Your_strong_pwd1;database=events;encrypt=disable")
	servicePort := getEnv("SERVICE_PORT", "8082")

	// Initialize MS SQL store
	sqlStore, err := store.NewMSSQLStore(mssqlConn, logger)
	if err != nil {
		logger.Fatal("Failed to initialize SQL store", zap.Error(err))
	}
	defer sqlStore.Close()

	// Create HTTP server
	mux := http.NewServeMux()

	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Metrics endpoint
	mux.Handle("/metrics", promhttp.Handler())

	// API endpoints
	mux.HandleFunc("/users/", func(w http.ResponseWriter, r *http.Request) {
		handleGetUser(w, r, sqlStore, logger)
	})

	mux.HandleFunc("/orders/", func(w http.ResponseWriter, r *http.Request) {
		handleGetOrder(w, r, sqlStore, logger)
	})

	mux.HandleFunc("/reviews/", func(w http.ResponseWriter, r *http.Request) {
		handleGetProductReview(w, r, sqlStore, logger)
	})

	mux.HandleFunc("/products/", func(w http.ResponseWriter, r *http.Request) {
		handleGetProductReviewsByProduct(w, r, sqlStore, logger)
	})

	// Start server
	server := &http.Server{
		Addr:         ":" + servicePort,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	logger.Info("Starting API service", zap.String("port", servicePort))
	if err := server.ListenAndServe(); err != nil {
		logger.Fatal("Failed to start server", zap.Error(err))
	}
}

func handleGetUser(w http.ResponseWriter, r *http.Request, sqlStore *store.MSSQLStore, logger *zap.Logger) {
	start := time.Now()
	defer func() {
		httpLatencySeconds.WithLabelValues(r.Method, "/users/").Observe(time.Since(start).Seconds())
	}()

	if r.Method != http.MethodGet {
		httpRequestsTotal.WithLabelValues(r.Method, "/users/", "405").Inc()
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract user ID from URL path
	userID := extractIDFromPath(r.URL.Path, "/users/")
	if userID == "" {
		httpRequestsTotal.WithLabelValues(r.Method, "/users/", "400").Inc()
		http.Error(w, "User ID is required", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get user
	user, err := sqlStore.GetUser(ctx, userID)
	if err != nil {
		httpRequestsTotal.WithLabelValues(r.Method, "/users/", "500").Inc()
		logger.Error("Failed to get user", zap.String("userID", userID), zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if user == nil {
		httpRequestsTotal.WithLabelValues(r.Method, "/users/", "404").Inc()
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Get recent orders
	recentOrders, err := sqlStore.GetUserRecentOrders(ctx, userID)
	if err != nil {
		httpRequestsTotal.WithLabelValues(r.Method, "/users/", "500").Inc()
		logger.Error("Failed to get user orders", zap.String("userID", userID), zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Prepare response
	response := APIResponse{
		User:         user,
		RecentOrders: recentOrders,
	}

	// Set content type and write response
	w.Header().Set("Content-Type", "application/json")
	httpRequestsTotal.WithLabelValues(r.Method, "/users/", "200").Inc()

	if err := json.NewEncoder(w).Encode(response); err != nil {
		logger.Error("Failed to encode response", zap.Error(err))
	}
}

func handleGetOrder(w http.ResponseWriter, r *http.Request, sqlStore *store.MSSQLStore, logger *zap.Logger) {
	start := time.Now()
	defer func() {
		httpLatencySeconds.WithLabelValues(r.Method, "/orders/").Observe(time.Since(start).Seconds())
	}()

	if r.Method != http.MethodGet {
		httpRequestsTotal.WithLabelValues(r.Method, "/orders/", "405").Inc()
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract order ID from URL path
	orderID := extractIDFromPath(r.URL.Path, "/orders/")
	if orderID == "" {
		httpRequestsTotal.WithLabelValues(r.Method, "/orders/", "400").Inc()
		http.Error(w, "Order ID is required", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get order
	order, err := sqlStore.GetOrder(ctx, orderID)
	if err != nil {
		httpRequestsTotal.WithLabelValues(r.Method, "/orders/", "500").Inc()
		logger.Error("Failed to get order", zap.String("orderID", orderID), zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if order == nil {
		httpRequestsTotal.WithLabelValues(r.Method, "/orders/", "404").Inc()
		http.Error(w, "Order not found", http.StatusNotFound)
		return
	}

	// Get payment (optional)
	payment, err := sqlStore.GetPayment(ctx, orderID)
	if err != nil {
		httpRequestsTotal.WithLabelValues(r.Method, "/orders/", "500").Inc()
		logger.Error("Failed to get payment", zap.String("orderID", orderID), zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Prepare response
	response := APIResponse{
		Order:   order,
		Payment: payment,
	}

	// Set content type and write response
	w.Header().Set("Content-Type", "application/json")
	httpRequestsTotal.WithLabelValues(r.Method, "/orders/", "200").Inc()

	if err := json.NewEncoder(w).Encode(response); err != nil {
		logger.Error("Failed to encode response", zap.Error(err))
	}
}

func extractIDFromPath(path, prefix string) string {
	if len(path) <= len(prefix) {
		return ""
	}
	return path[len(prefix):]
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func handleGetProductReview(w http.ResponseWriter, r *http.Request, sqlStore *store.MSSQLStore, logger *zap.Logger) {
	start := time.Now()
	defer func() {
		httpLatencySeconds.WithLabelValues(r.Method, "/reviews/").Observe(time.Since(start).Seconds())
	}()

	if r.Method != http.MethodGet {
		httpRequestsTotal.WithLabelValues(r.Method, "/reviews/", "405").Inc()
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract review ID from URL path
	reviewID := extractIDFromPath(r.URL.Path, "/reviews/")
	if reviewID == "" {
		httpRequestsTotal.WithLabelValues(r.Method, "/reviews/", "400").Inc()
		http.Error(w, "Review ID is required", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get product review
	review, err := sqlStore.GetProductReview(ctx, reviewID)
	if err != nil {
		httpRequestsTotal.WithLabelValues(r.Method, "/reviews/", "500").Inc()
		logger.Error("Failed to get product review", zap.String("reviewID", reviewID), zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if review == nil {
		httpRequestsTotal.WithLabelValues(r.Method, "/reviews/", "404").Inc()
		http.Error(w, "Review not found", http.StatusNotFound)
		return
	}

	// Prepare response
	response := map[string]interface{}{
		"review": review,
	}

	// Set content type and write response
	w.Header().Set("Content-Type", "application/json")
	httpRequestsTotal.WithLabelValues(r.Method, "/reviews/", "200").Inc()

	if err := json.NewEncoder(w).Encode(response); err != nil {
		logger.Error("Failed to encode response", zap.Error(err))
	}
}

func handleGetProductReviewsByProduct(w http.ResponseWriter, r *http.Request, sqlStore *store.MSSQLStore, logger *zap.Logger) {
	start := time.Now()
	defer func() {
		httpLatencySeconds.WithLabelValues(r.Method, "/products/").Observe(time.Since(start).Seconds())
	}()

	if r.Method != http.MethodGet {
		httpRequestsTotal.WithLabelValues(r.Method, "/products/", "405").Inc()
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract product name from URL path
	productName := extractIDFromPath(r.URL.Path, "/products/")
	if productName == "" {
		httpRequestsTotal.WithLabelValues(r.Method, "/products/", "400").Inc()
		http.Error(w, "Product name is required", http.StatusBadRequest)
		return
	}

	// Remove "/reviews" suffix if present
	productName = strings.TrimSuffix(productName, "/reviews")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get product reviews
	reviews, err := sqlStore.GetProductReviewsByProduct(ctx, productName)
	if err != nil {
		httpRequestsTotal.WithLabelValues(r.Method, "/products/", "500").Inc()
		logger.Error("Failed to get product reviews", zap.String("productName", productName), zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Prepare response
	response := map[string]interface{}{
		"productName": productName,
		"reviews":     reviews,
		"count":       len(reviews),
	}

	// Set content type and write response
	w.Header().Set("Content-Type", "application/json")
	httpRequestsTotal.WithLabelValues(r.Method, "/products/", "200").Inc()

	if err := json.NewEncoder(w).Encode(response); err != nil {
		logger.Error("Failed to encode response", zap.Error(err))
	}
}
