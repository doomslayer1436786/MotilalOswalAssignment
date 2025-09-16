-- Create database if it doesn't exist
IF NOT EXISTS (SELECT name FROM sys.databases WHERE name = 'events')
BEGIN
    CREATE DATABASE events;
END
GO

USE events;
GO

-- Create users table
IF NOT EXISTS (SELECT * FROM sysobjects WHERE name='users' AND xtype='U')
BEGIN
    CREATE TABLE users (
        user_id VARCHAR(100) PRIMARY KEY,
        name VARCHAR(255),
        email VARCHAR(255),
        created_at DATETIME2,
        updated_at DATETIME2
    );
END
GO

-- Create orders table
IF NOT EXISTS (SELECT * FROM sysobjects WHERE name='orders' AND xtype='U')
BEGIN
    CREATE TABLE orders (
        order_id VARCHAR(100) PRIMARY KEY,
        user_id VARCHAR(100),
        total DECIMAL(18,2),
        status VARCHAR(50),
        created_at DATETIME2,
        updated_at DATETIME2,
        CONSTRAINT FK_Orders_Users FOREIGN KEY (user_id) REFERENCES users(user_id)
    );
END
GO

-- Create payments table
IF NOT EXISTS (SELECT * FROM sysobjects WHERE name='payments' AND xtype='U')
BEGIN
    CREATE TABLE payments (
        order_id VARCHAR(100) PRIMARY KEY,
        status VARCHAR(50),
        amount DECIMAL(18,2),
        settled_at DATETIME2,
        updated_at DATETIME2
    );
END
GO

-- Create inventory table
IF NOT EXISTS (SELECT * FROM sysobjects WHERE name='inventory' AND xtype='U')
BEGIN
    CREATE TABLE inventory (
        sku VARCHAR(100) PRIMARY KEY,
        quantity INT,
        last_adjusted_at DATETIME2
    );
END
GO


IF NOT EXISTS (SELECT * FROM sysobjects WHERE name='product_reviews' AND xtype='U')
BEGIN
    CREATE TABLE product_reviews (
        review_id VARCHAR(100) PRIMARY KEY,
        product_name VARCHAR(255) NOT NULL,
        username VARCHAR(255) NOT NULL,
        rating INT NOT NULL CHECK (rating >= 1 AND rating <= 5),
        remarks VARCHAR(MAX),
        created_at DATETIME2,
        updated_at DATETIME2
    );
END
GO


IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'IX_orders_user_id')
BEGIN
    CREATE INDEX IX_orders_user_id ON orders(user_id);
END
GO

IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'IX_orders_created_at')
BEGIN
    CREATE INDEX IX_orders_created_at ON orders(created_at DESC);
END
GO

IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'IX_product_reviews_product_name')
BEGIN
    CREATE INDEX IX_product_reviews_product_name ON product_reviews(product_name);
END
GO

IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'IX_product_reviews_username')
BEGIN
    CREATE INDEX IX_product_reviews_username ON product_reviews(username);
END
GO

IF NOT EXISTS (SELECT * FROM sys.indexes WHERE name = 'IX_product_reviews_rating')
BEGIN
    CREATE INDEX IX_product_reviews_rating ON product_reviews(rating);
END
GO

PRINT 'Database schema created successfully';
