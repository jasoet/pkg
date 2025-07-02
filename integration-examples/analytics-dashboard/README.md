# Analytics Dashboard Integration Example

A real-time analytics dashboard demonstrating advanced data processing patterns with [github.com/jasoet/pkg](https://github.com/jasoet/pkg).

## Features Demonstrated

### 📊 **Real-time Analytics**
- Event ingestion and processing
- Real-time metrics computation
- WebSocket-ready data updates
- Time-series data aggregation
- Concurrent data processing with the `concurrent` package

### 🎯 **Dashboard Functionality**
- Overview metrics (users, pageviews, sessions)
- Traffic analysis with time-series data
- Top pages analytics
- User behavior metrics
- Real-time visitor tracking

### ⚡ **Performance Optimization**
- Concurrent data fetching for dashboard
- Background metric aggregation
- Efficient database queries with indexes
- Batch event processing
- Data retention management

### 🏗️ **Architecture Patterns**
- Background workers for data processing
- Service-oriented architecture
- Event-driven data pipeline
- Separation of concerns (ingestion vs. analysis)
- Graceful shutdown handling

## API Endpoints

### Event Ingestion
- `POST /api/v1/events` - Ingest single event
- `POST /api/v1/events/batch` - Batch event ingestion

### Dashboard Data
- `GET /api/v1/dashboard` - Complete dashboard data
- `GET /api/v1/dashboard/overview` - Overview metrics
- `GET /api/v1/dashboard/traffic` - Traffic data over time
- `GET /api/v1/dashboard/pages` - Top pages analytics
- `GET /api/v1/dashboard/users` - User metrics
- `GET /api/v1/dashboard/realtime` - Real-time data

### Analytics Queries
- `GET /api/v1/analytics/events` - Query events with filters
- `GET /api/v1/analytics/sessions` - Query user sessions
- `GET /api/v1/analytics/metrics` - Query aggregated metrics

### Custom Reports
- `POST /api/v1/reports/custom` - Generate custom report
- `GET /api/v1/reports/:id` - Retrieve report

### System
- `GET /api/v1/health` - Health check
- `GET /api/v1/metrics` - System metrics

## Data Models

### Events
```go
type Event struct {
    ID         uint                   `json:"id"`
    EventType  string                 `json:"eventType"`  // page_view, click, conversion
    UserID     string                 `json:"userId"`
    SessionID  string                 `json:"sessionId"`
    Properties map[string]interface{} `json:"properties"` // Custom event data
    Timestamp  time.Time              `json:"timestamp"`
}
```

### User Sessions
```go
type UserSession struct {
    ID          uint      `json:"id"`
    SessionID   string    `json:"sessionId"`
    UserID      string    `json:"userId"`
    StartTime   time.Time `json:"startTime"`
    EndTime     *time.Time `json:"endTime"`
    PageViews   int       `json:"pageViews"`
    Duration    int       `json:"duration"`    // seconds
    UserAgent   string    `json:"userAgent"`
    IPAddress   string    `json:"ipAddress"`
    Referrer    string    `json:"referrer"`
}
```

### Daily Metrics
```go
type DailyMetric struct {
    ID          uint      `json:"id"`
    Date        time.Time `json:"date"`
    MetricName  string    `json:"metricName"`  // page_views, unique_users, etc.
    Value       float64   `json:"value"`
    Dimension   string    `json:"dimension"`   // page, source, device, etc.
}
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
   cd pkg/integration-examples/analytics-dashboard
   ```

2. **Install Dependencies**
   ```bash
   go mod init analytics-dashboard
   go mod tidy
   ```

3. **Start PostgreSQL**
   ```bash
   # Using Docker
   docker run -d --name analytics-postgres \
     -e POSTGRES_USER=postgres \
     -e POSTGRES_PASSWORD=password \
     -e POSTGRES_DB=analytics_db \
     -p 5432:5432 \
     postgres:15-alpine
   
   # Or use Docker Compose
   docker-compose up -d postgres
   ```

4. **Run Application**
   ```bash
   go run main.go
   ```

5. **Test Event Ingestion**
   ```bash
   # Send a test event
   curl -X POST http://localhost:8080/api/v1/events \
     -H "Content-Type: application/json" \
     -d '{
       "eventType": "page_view",
       "userId": "user123",
       "sessionId": "session456",
       "properties": {
         "page": "/products",
         "title": "Products Page",
         "referrer": "https://google.com"
       }
     }'
   ```

6. **View Dashboard Data**
   ```bash
   curl http://localhost:8080/api/v1/dashboard
   ```

## Configuration

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
  dbName: analytics_db
analytics:
  retentionDays: 90           # How long to keep raw event data
  processInterval: 1m         # How often to process events
  batchSize: 1000            # Events to process in each batch
  enableRealTime: true       # Enable real-time processing
dataSources:
  - name: web_analytics
    type: webhook
    url: /api/v1/events
    interval: 0s
  - name: app_analytics  
    type: api
    url: https://api.example.com/analytics
    interval: 5m
```

### Environment Variables

Use the `ANALYTICS_` prefix:

```bash
export ANALYTICS_DATABASE_HOST=prod-db.example.com
export ANALYTICS_DATABASE_PASSWORD=secure-password
export ANALYTICS_ANALYTICS_RETENTIONDAYS=180
export ANALYTICS_ANALYTICS_ENABLEREALTIME=true
```

## Integration Patterns Demonstrated

### 1. **Concurrent Dashboard Data Fetching**

```go
func getDashboardDataHandler(services *Services) echo.HandlerFunc {
    return func(c echo.Context) error {
        // Fetch all dashboard components concurrently
        dataFunctions := map[string]concurrent.Func[interface{}]{
            "overview": func(ctx context.Context) (interface{}, error) {
                return getOverviewMetrics(services.DB, startDate, endDate)
            },
            "traffic": func(ctx context.Context) (interface{}, error) {
                return getTrafficData(services.DB, startDate, endDate)
            },
            "pages": func(ctx context.Context) (interface{}, error) {
                return getTopPages(services.DB, startDate, endDate)
            },
            "users": func(ctx context.Context) (interface{}, error) {
                return getUserMetrics(services.DB, startDate, endDate)
            },
        }
        
        // Execute all queries concurrently
        results, err := concurrent.ExecuteConcurrently(c.Request().Context(), dataFunctions)
        if err != nil {
            return c.JSON(500, ErrorResponse{Error: "Failed to fetch data"})
        }
        
        // Combine results into dashboard response
        response := DashboardResponse{
            Overview:    results["overview"].(OverviewMetrics),
            TrafficData: results["traffic"].([]TrafficPoint),
            TopPages:    results["pages"].([]PageMetric),
            UserMetrics: results["users"].(UserMetrics),
            Timestamp:   time.Now(),
        }
        
        return c.JSON(200, response)
    }
}
```

### 2. **Background Data Processing**

```go
func startEventProcessor(ctx context.Context, services *Services) {
    logger := logging.ContextLogger(ctx, "event-processor")
    ticker := time.NewTicker(services.Config.Analytics.ProcessInterval)
    defer ticker.Stop()
    
    for {
        select {
        case <-ctx.Done():
            logger.Info().Msg("Event processor shutting down")
            return
        case <-ticker.C:
            if err := processEvents(ctx, services); err != nil {
                logger.Error().Err(err).Msg("Event processing failed")
            }
        }
    }
}
```

### 3. **Batch Event Processing**

```go
func ingestEventBatchHandler(services *Services) echo.HandlerFunc {
    return func(c echo.Context) error {
        var events []Event
        if err := c.Bind(&events); err != nil {
            return c.JSON(400, ErrorResponse{Error: "Invalid batch data"})
        }
        
        // Process events concurrently for better performance
        batchFunctions := make(map[string]concurrent.Func[bool])
        for i, event := range events {
            eventCopy := event
            key := fmt.Sprintf("event_%d", i)
            batchFunctions[key] = func(ctx context.Context) (bool, error) {
                if eventCopy.Timestamp.IsZero() {
                    eventCopy.Timestamp = time.Now()
                }
                return true, services.DB.Create(&eventCopy).Error
            }
        }
        
        results, err := concurrent.ExecuteConcurrently(c.Request().Context(), batchFunctions)
        if err != nil {
            return c.JSON(500, ErrorResponse{Error: "Batch processing failed"})
        }
        
        successCount := 0
        for _, result := range results {
            if result.Success {
                successCount++
            }
        }
        
        return c.JSON(201, BatchResponse{
            Processed: successCount,
            Total:     len(events),
        })
    }
}
```

### 4. **Time-Series Data Queries**

```go
func getTrafficData(db *gorm.DB, startDate, endDate time.Time) ([]TrafficPoint, error) {
    var traffic []TrafficPoint
    
    query := `
        SELECT 
            DATE(timestamp) as date,
            COUNT(*) as page_views,
            COUNT(DISTINCT user_id) as users,
            COUNT(DISTINCT session_id) as sessions
        FROM events 
        WHERE event_type = 'page_view' 
        AND timestamp >= ? AND timestamp < ?
        GROUP BY DATE(timestamp)
        ORDER BY date
    `
    
    rows, err := db.Raw(query, startDate, endDate).Rows()
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    for rows.Next() {
        var point TrafficPoint
        if err := db.ScanRows(rows, &point); err != nil {
            return nil, err
        }
        traffic = append(traffic, point)
    }
    
    return traffic, nil
}
```

### 5. **Real-time Metrics Processing**

```go
func startRealtimeProcessor(ctx context.Context, services *Services) {
    logger := logging.ContextLogger(ctx, "realtime-processor")
    ticker := time.NewTicker(10 * time.Second)
    defer ticker.Stop()
    
    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            if err := updateRealtimeMetrics(ctx, services); err != nil {
                logger.Error().Err(err).Msg("Realtime update failed")
            }
        }
    }
}

func updateRealtimeMetrics(ctx context.Context, services *Services) error {
    logger := logging.ContextLogger(ctx, "realtime-update")
    
    // Get current active users (last 5 minutes)
    fiveMinutesAgo := time.Now().Add(-5 * time.Minute)
    
    var currentUsers int64
    err := services.DB.Model(&Event{}).
        Where("timestamp >= ?", fiveMinutesAgo).
        Distinct("user_id").
        Count(&currentUsers).Error
    
    if err != nil {
        return fmt.Errorf("failed to count current users: %w", err)
    }
    
    // Store or broadcast realtime metric
    metric := DailyMetric{
        Date:       time.Now(),
        MetricName: "current_users_5min",
        Value:      float64(currentUsers),
        Dimension:  "realtime",
    }
    
    if err := services.DB.Create(&metric).Error; err != nil {
        logger.Error().Err(err).Msg("Failed to store realtime metric")
    }
    
    logger.Info().Int64("current_users", currentUsers).Msg("Updated realtime metrics")
    return nil
}
```

## Event Types and Schema

### Page View Event
```json
{
  "eventType": "page_view",
  "userId": "user123",
  "sessionId": "session456",
  "properties": {
    "page": "/products",
    "title": "Products Page",
    "referrer": "https://google.com",
    "device": "desktop",
    "browser": "Chrome",
    "country": "US"
  }
}
```

### Click Event
```json
{
  "eventType": "click",
  "userId": "user123",
  "sessionId": "session456",
  "properties": {
    "element": "buy_button",
    "page": "/products/laptop",
    "position": {"x": 450, "y": 320},
    "product_id": "prod_789"
  }
}
```

### Conversion Event
```json
{
  "eventType": "conversion",
  "userId": "user123",
  "sessionId": "session456",
  "properties": {
    "type": "purchase",
    "value": 299.99,
    "currency": "USD",
    "product_ids": ["prod_789", "prod_012"],
    "campaign": "summer_sale"
  }
}
```

## Dashboard Widgets

### Overview Widget
- Total Users (unique visitors)
- Total Page Views
- Total Sessions
- Average Session Duration
- Bounce Rate

### Traffic Chart
- Page views over time
- Users over time
- Sessions over time
- Configurable time ranges (7d, 30d, 90d)

### Top Pages Table
- Page URL
- Total views
- Unique users
- Average time on page

### User Metrics
- New vs Returning Users
- Average sessions per user
- User engagement metrics

### Real-time Feed
- Current active users
- Page views in last 5 minutes
- Top pages right now
- Live activity stream

## Performance Optimizations

### Database Indexes
```sql
-- Event queries
CREATE INDEX idx_events_type_timestamp ON events(event_type, timestamp);
CREATE INDEX idx_events_user_timestamp ON events(user_id, timestamp);
CREATE INDEX idx_events_session_timestamp ON events(session_id, timestamp);

-- Metrics queries
CREATE INDEX idx_daily_metrics_date_name ON daily_metrics(date, metric_name);
CREATE INDEX idx_daily_metrics_dimension ON daily_metrics(dimension);

-- Session queries
CREATE INDEX idx_user_sessions_start_time ON user_sessions(start_time);
CREATE INDEX idx_user_sessions_user_id ON user_sessions(user_id);
```

### Query Optimization
- Use prepared statements for repeated queries
- Implement query result caching for dashboard data
- Use database views for complex aggregations
- Batch database operations where possible

### Memory Management
- Stream large result sets instead of loading all into memory
- Use connection pooling for database connections
- Implement proper cleanup for background workers
- Monitor memory usage and optimize data structures

## Testing Examples

### Unit Test for Event Processing
```go
func TestEventProcessor(t *testing.T) {
    db := setupTestDB(t)
    defer db.Close()
    
    // Create test events
    events := []Event{
        {EventType: "page_view", UserID: "user1", SessionID: "session1"},
        {EventType: "click", UserID: "user1", SessionID: "session1"},
    }
    
    for _, event := range events {
        db.Create(&event)
    }
    
    // Test processing
    services := &Services{DB: db, Config: testConfig}
    err := processEvents(context.Background(), services)
    
    assert.NoError(t, err)
    
    // Verify metrics were created
    var metrics []DailyMetric
    db.Find(&metrics)
    assert.Greater(t, len(metrics), 0)
}
```

### Integration Test for Dashboard API
```go
func TestDashboardAPI_Integration(t *testing.T) {
    app := setupTestApp(t)
    defer app.Cleanup()
    
    // Create test data
    createTestEvents(t, app.DB)
    
    // Test dashboard endpoint
    resp, err := http.Get(app.URL + "/api/v1/dashboard?days=7")
    assert.NoError(t, err)
    assert.Equal(t, 200, resp.StatusCode)
    
    var dashboard DashboardResponse
    json.NewDecoder(resp.Body).Decode(&dashboard)
    
    assert.Greater(t, dashboard.Overview.TotalUsers, int64(0))
    assert.Greater(t, len(dashboard.TrafficData), 0)
}
```

## Production Deployment

### Environment Configuration
```yaml
# Production config
environment: production
debug: false
server:
  port: ${PORT:8080}
database:
  host: ${DB_HOST}
  password: ${DB_PASSWORD}
  maxOpenConns: 100
analytics:
  retentionDays: 365
  processInterval: 30s
  batchSize: 5000
  enableRealTime: true
```

### Docker Deployment
```dockerfile
FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o analytics-dashboard .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/analytics-dashboard .
EXPOSE 8080
CMD ["./analytics-dashboard"]
```

### Monitoring
- Set up database performance monitoring
- Monitor event ingestion rates
- Track processing latency
- Set up alerts for failed background jobs
- Monitor memory and CPU usage

## Next Steps

To extend this analytics dashboard:

1. **Add Real-time WebSocket Updates** for live dashboard updates
2. **Implement Custom Event Schemas** with validation
3. **Add Data Export Functionality** (CSV, JSON, API)
4. **Create User Segmentation** and cohort analysis
5. **Add A/B Testing Support** with experiment tracking
6. **Implement Funnel Analysis** for conversion tracking
7. **Add Geographic Analytics** with IP-based location
8. **Create Alerting System** for threshold-based notifications
9. **Add Data Visualization** with charts and graphs
10. **Implement Data Privacy Features** with GDPR compliance

This analytics dashboard demonstrates how to build a scalable, real-time analytics system using the github.com/jasoet/pkg utility library with proper concurrent processing, background workers, and performance optimization.