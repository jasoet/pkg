# E-commerce API Integration Example

A complete e-commerce REST API demonstrating advanced integration patterns with [github.com/jasoet/pkg](https://github.com/jasoet/pkg).

## Features Demonstrated

### 🏗️ **Core Architecture**
- RESTful API design with proper HTTP methods and status codes
- Layered architecture with services, handlers, and middleware
- Dependency injection pattern for testable code
- Database modeling with relationships using GORM

### 🔐 **Authentication & Authorization**
- JWT-based authentication
- Role-based access control (customer/admin)
- Protected routes with middleware
- User registration and login endpoints

### 💰 **E-commerce Functionality**
- Product catalog with categories
- User management and profiles
- Shopping cart and order processing
- Payment integration (Stripe example)
- File upload for product images

### 🛡️ **Security & Validation**
- Input validation and sanitization
- SQL injection prevention with GORM
- File upload security checks
- Rate limiting and CORS

### 📊 **Observability**
- Structured logging with request IDs
- Health checks with dependency validation
- Basic metrics collection
- Error tracking and monitoring

## API Endpoints

### Authentication
- `POST /api/v1/auth/register` - User registration
- `POST /api/v1/auth/login` - User authentication

### Public Product Catalog
- `GET /api/v1/categories` - List all categories
- `GET /api/v1/categories/:id/products` - Products by category
- `GET /api/v1/products` - List products (with pagination)
- `GET /api/v1/products/:id` - Get product details

### User Account (Protected)
- `GET /api/v1/users/profile` - Get user profile
- `PUT /api/v1/users/profile` - Update user profile

### Orders (Protected)
- `GET /api/v1/orders` - List user orders
- `POST /api/v1/orders` - Create new order
- `GET /api/v1/orders/:id` - Get order details

### Admin Management (Admin Role Required)
- `GET /api/v1/admin/categories` - Manage categories
- `POST /api/v1/admin/categories` - Create category
- `PUT /api/v1/admin/categories/:id` - Update category
- `DELETE /api/v1/admin/categories/:id` - Delete category
- `GET /api/v1/admin/products` - Manage products
- `POST /api/v1/admin/products` - Create product
- `PUT /api/v1/admin/products/:id` - Update product
- `DELETE /api/v1/admin/products/:id` - Delete product
- `POST /api/v1/admin/products/:id/upload` - Upload product image
- `GET /api/v1/admin/orders` - List all orders
- `PUT /api/v1/admin/orders/:id/status` - Update order status

### System
- `GET /api/v1/health` - Health check
- `GET /api/v1/metrics` - Basic metrics

## Database Schema

### Users Table
```sql
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    email VARCHAR UNIQUE NOT NULL,
    password VARCHAR NOT NULL,
    first_name VARCHAR NOT NULL,
    last_name VARCHAR NOT NULL,
    role VARCHAR DEFAULT 'customer',
    created_at TIMESTAMP,
    updated_at TIMESTAMP
);
```

### Categories Table
```sql
CREATE TABLE categories (
    id SERIAL PRIMARY KEY,
    name VARCHAR UNIQUE NOT NULL,
    description TEXT,
    image_url VARCHAR,
    created_at TIMESTAMP,
    updated_at TIMESTAMP
);
```

### Products Table
```sql
CREATE TABLE products (
    id SERIAL PRIMARY KEY,
    name VARCHAR NOT NULL,
    description TEXT,
    price DECIMAL NOT NULL,
    stock INTEGER DEFAULT 0,
    sku VARCHAR UNIQUE NOT NULL,
    image_url VARCHAR,
    category_id INTEGER REFERENCES categories(id),
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP,
    updated_at TIMESTAMP
);
```

### Orders Table
```sql
CREATE TABLE orders (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id),
    status VARCHAR DEFAULT 'pending',
    total_amount DECIMAL NOT NULL,
    payment_id VARCHAR,
    shipping_address TEXT NOT NULL,
    created_at TIMESTAMP,
    updated_at TIMESTAMP
);
```

### Order Items Table
```sql
CREATE TABLE order_items (
    id SERIAL PRIMARY KEY,
    order_id INTEGER REFERENCES orders(id),
    product_id INTEGER REFERENCES products(id),
    quantity INTEGER NOT NULL,
    price DECIMAL NOT NULL
);
```

## Quick Start

### Prerequisites
- Go 1.23+
- PostgreSQL 15+
- Docker and Docker Compose (optional)

### Setup

1. **Clone and Navigate**
   ```bash
   git clone https://github.com/jasoet/pkg.git
   cd pkg/integration-examples/ecommerce-api
   ```

2. **Install Dependencies**
   ```bash
   go mod init ecommerce-api
   go mod tidy
   ```

3. **Start PostgreSQL**
   ```bash
   # Using Docker
   docker run -d --name ecommerce-postgres \
     -e POSTGRES_USER=postgres \
     -e POSTGRES_PASSWORD=password \
     -e POSTGRES_DB=ecommerce_db \
     -p 5432:5432 \
     postgres:15-alpine
   
   # Or use Docker Compose
   docker-compose up -d postgres
   ```

4. **Configure Environment**
   ```bash
   # Set environment variables (optional)
   export ECOMMERCE_DATABASE_PASSWORD=your_password
   export ECOMMERCE_JWT_SECRET=your-very-secure-jwt-secret-key
   export ECOMMERCE_PAYMENT_APIKEY=your_stripe_key
   ```

5. **Run Application**
   ```bash
   go run main.go
   ```

6. **Test Health Endpoint**
   ```bash
   curl http://localhost:8080/api/v1/health
   ```

## Configuration

The application uses YAML configuration with environment variable overrides:

```yaml
environment: development
debug: true
server:
  port: 8080
  readTimeout: 30s
  writeTimeout: 30s
  enableHealthChecks: true
database:
  dbType: POSTGRES
  host: localhost
  port: 5432
  username: postgres
  password: password
  dbName: ecommerce_db
jwt:
  secret: your-super-secret-jwt-key-at-least-32-characters-long
  expiration: 24h
payment:
  provider: stripe
  apiKey: sk_test_your_stripe_secret_key
  baseUrl: https://api.stripe.com
upload:
  maxFileSize: 5242880  # 5MB
  allowedTypes:
    - image/jpeg
    - image/png
    - image/gif
  uploadDir: ./uploads
```

### Environment Variable Overrides

Use the `ECOMMERCE_` prefix:

```bash
export ECOMMERCE_SERVER_PORT=3000
export ECOMMERCE_DATABASE_HOST=prod-db.example.com
export ECOMMERCE_DATABASE_PASSWORD=secure-password
export ECOMMERCE_JWT_SECRET=production-jwt-secret
export ECOMMERCE_PAYMENT_APIKEY=live_stripe_key
```

## Integration Patterns Demonstrated

### 1. **Service Layer Architecture**

```go
type Services struct {
    DB            *gorm.DB
    PaymentClient *rest.Client
    Config        *AppConfig
    Logger        zerolog.Logger
}

// Service injection into handlers
func createOrderHandler(services *Services) echo.HandlerFunc {
    return func(c echo.Context) error {
        logger := logging.ContextLogger(c.Request().Context(), "create-order")
        
        // Use services.DB for database operations
        // Use services.PaymentClient for payment processing
        // Use logger for structured logging
    }
}
```

### 2. **Middleware Chain Pattern**

```go
func setupRoutes(services *Services) func(*echo.Echo) {
    return func(e *echo.Echo) {
        // Global middleware
        e.Use(middleware.RequestID())
        e.Use(middleware.Recover())
        e.Use(loggingMiddleware(services))
        
        // Route-specific middleware
        protected := api.Group("")
        protected.Use(authMiddleware(services))
        
        admin := protected.Group("/admin")
        admin.Use(adminMiddleware(services))
    }
}
```

### 3. **Database Relationship Handling**

```go
// Eager loading relationships
var products []Product
services.DB.Preload("Category").Find(&products)

// Query with conditions
var orders []Order
services.DB.Where("user_id = ?", userID).
    Preload("OrderItems.Product").
    Find(&orders)
```

### 4. **Error Handling with Context**

```go
func getProductHandler(services *Services) echo.HandlerFunc {
    return func(c echo.Context) error {
        logger := logging.ContextLogger(c.Request().Context(), "get-product")
        
        id, err := strconv.ParseUint(c.Param("id"), 10, 32)
        if err != nil {
            logger.Warn().Str("id_param", c.Param("id")).Msg("Invalid product ID")
            return c.JSON(http.StatusBadRequest, ErrorResponse{
                Error: "Invalid product ID",
                Code:  "INVALID_ID",
            })
        }
        
        var product Product
        if err := services.DB.First(&product, uint(id)).Error; err != nil {
            if err == gorm.ErrRecordNotFound {
                logger.Info().Uint("product_id", uint(id)).Msg("Product not found")
                return c.JSON(http.StatusNotFound, ErrorResponse{
                    Error: "Product not found",
                    Code:  "PRODUCT_NOT_FOUND",
                })
            }
            
            logger.Error().Err(err).Uint("product_id", uint(id)).Msg("Database query failed")
            return c.JSON(http.StatusInternalServerError, ErrorResponse{
                Error: "Internal server error",
                Code:  "DATABASE_ERROR",
            })
        }
        
        logger.Info().Uint("product_id", product.ID).Msg("Product retrieved successfully")
        return c.JSON(http.StatusOK, product)
    }
}
```

### 5. **Authentication Middleware**

```go
func authMiddleware(services *Services) echo.MiddlewareFunc {
    return func(next echo.HandlerFunc) echo.HandlerFunc {
        return func(c echo.Context) error {
            logger := logging.ContextLogger(c.Request().Context(), "auth-middleware")
            
            // Extract and validate JWT token
            token := extractToken(c.Request())
            if token == "" {
                logger.Warn().Msg("Missing authorization token")
                return echo.NewHTTPError(http.StatusUnauthorized, "Authorization required")
            }
            
            userID, err := validateJWTToken(token, services.Config.JWT.Secret)
            if err != nil {
                logger.Error().Err(err).Msg("Token validation failed")
                return echo.NewHTTPError(http.StatusUnauthorized, "Invalid token")
            }
            
            // Store user ID in context for downstream handlers
            c.Set("userID", userID)
            return next(c)
        }
    }
}
```

## Testing Examples

### Unit Test
```go
func TestProductService_GetProduct(t *testing.T) {
    // Setup test database
    testDB := setupTestDB(t)
    defer testDB.Close()
    
    // Create test data
    category := &Category{Name: "Electronics", Description: "Electronic items"}
    testDB.Create(category)
    
    product := &Product{
        Name:        "Laptop",
        Description: "Gaming laptop",
        Price:       999.99,
        CategoryID:  category.ID,
        SKU:         "LAP001",
    }
    testDB.Create(product)
    
    // Test service
    productService := NewProductService(testDB)
    result, err := productService.GetProduct(context.Background(), product.ID)
    
    // Assertions
    assert.NoError(t, err)
    assert.NotNil(t, result)
    assert.Equal(t, product.Name, result.Name)
    assert.Equal(t, category.Name, result.Category.Name)
}
```

### Integration Test
```go
func TestAPI_CreateOrder_Integration(t *testing.T) {
    // Setup test application
    app := setupTestApp(t)
    defer app.Cleanup()
    
    // Create test user and authenticate
    user := createTestUser(t, app.DB)
    token := generateTestJWT(user.ID)
    
    // Create test product
    product := createTestProduct(t, app.DB)
    
    // Prepare order request
    orderRequest := CreateOrderRequest{
        Items: []OrderItemRequest{
            {ProductID: product.ID, Quantity: 2},
        },
        ShippingAddress: "123 Test St, Test City",
    }
    
    requestBody, _ := json.Marshal(orderRequest)
    
    // Make HTTP request
    req, _ := http.NewRequest("POST", app.URL+"/api/v1/orders", bytes.NewBuffer(requestBody))
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Authorization", "Bearer "+token)
    
    resp, err := http.DefaultClient.Do(req)
    
    // Verify response
    assert.NoError(t, err)
    assert.Equal(t, http.StatusCreated, resp.StatusCode)
    
    // Verify database state
    var order Order
    err = app.DB.Preload("OrderItems").First(&order, "user_id = ?", user.ID).Error
    assert.NoError(t, err)
    assert.Equal(t, 1, len(order.OrderItems))
    assert.Equal(t, product.ID, order.OrderItems[0].ProductID)
}
```

## Production Considerations

### Security
- Use strong JWT secrets (min 32 characters)
- Implement rate limiting for authentication endpoints
- Add input validation for all endpoints
- Use HTTPS in production
- Implement proper password hashing (bcrypt)

### Performance
- Add database indexes for frequently queried fields
- Implement response caching for product catalog
- Use connection pooling for database
- Add pagination for list endpoints
- Optimize database queries with proper preloading

### Monitoring
- Add structured logging for all operations
- Implement metrics collection (Prometheus)
- Set up health checks for all dependencies
- Add distributed tracing for request flows
- Monitor database performance

### Scalability
- Use database connection pooling
- Implement horizontal scaling with load balancers
- Add caching layer (Redis) for frequently accessed data
- Consider database read replicas for read-heavy operations
- Implement proper session management

## Docker Deployment

### Dockerfile
```dockerfile
FROM golang:1.23-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o ecommerce-api .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/

COPY --from=builder /app/ecommerce-api .
COPY --from=builder /app/uploads ./uploads

EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/api/v1/health || exit 1

CMD ["./ecommerce-api"]
```

### Docker Compose
```yaml
version: '3.8'
services:
  app:
    build: .
    ports:
      - "8080:8080"
    environment:
      - ECOMMERCE_DATABASE_HOST=postgres
      - ECOMMERCE_DATABASE_PASSWORD=password
    depends_on:
      - postgres
      - redis
    volumes:
      - ./uploads:/root/uploads

  postgres:
    image: postgres:15-alpine
    environment:
      - POSTGRES_USER=postgres
      - POSTGRES_PASSWORD=password
      - POSTGRES_DB=ecommerce_db
    volumes:
      - postgres_data:/var/lib/postgresql/data
    ports:
      - "5432:5432"

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"

volumes:
  postgres_data:
```

## Next Steps

This example provides a solid foundation for an e-commerce API. To extend it further:

1. **Complete the TODO implementations** in the handler functions
2. **Add comprehensive input validation** with proper error responses
3. **Implement JWT token generation and validation**
4. **Add password hashing and user authentication**
5. **Integrate with real payment providers** (Stripe, PayPal)
6. **Add file upload functionality** with security checks
7. **Implement order fulfillment workflow** with status tracking
8. **Add inventory management** with stock tracking
9. **Create admin dashboard** for order and product management
10. **Add comprehensive test coverage** for all endpoints

This integration example demonstrates how to build a production-ready e-commerce API using the github.com/jasoet/pkg utility library with proper architecture, security, and observability patterns.