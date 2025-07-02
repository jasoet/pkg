# Worker Service Template

This template provides a complete background worker implementation using [github.com/jasoet/pkg](https://github.com/jasoet/pkg).

## Features

- **Job Queue Processing**: Database-backed job queue with retry logic
- **Concurrent Processing**: Configurable concurrent job processing using the concurrent package
- **Structured Configuration**: YAML configuration with environment variable overrides
- **Database Integration**: PostgreSQL with GORM for job persistence
- **External API Integration**: HTTP client for external service calls
- **Structured Logging**: Context-aware logging with job tracking
- **Graceful Shutdown**: Proper signal handling and resource cleanup
- **Retry Logic**: Configurable retry attempts with backoff
- **Health Monitoring**: Job processing metrics and error tracking

## Quick Start

1. **Copy Template**
   ```bash
   cp -r templates/worker my-worker-service
   cd my-worker-service
   ```

2. **Update Module Name**
   ```bash
   go mod edit -module github.com/yourusername/my-worker-service
   ```

3. **Install Dependencies**
   ```bash
   go mod tidy
   ```

4. **Start Database**
   ```bash
   docker run -d --name worker-postgres \
     -e POSTGRES_USER=postgres \
     -e POSTGRES_PASSWORD=password \
     -e POSTGRES_DB=worker_db \
     -p 5432:5432 \
     postgres:15-alpine
   ```

5. **Run Worker**
   ```bash
   go run main.go
   ```

## Configuration

The worker uses `config.yaml` for base configuration with environment variable overrides:

```yaml
environment: development
debug: true
database:
  dbType: POSTGRES
  host: localhost
  port: 5432
  username: postgres
  password: password
  dbName: worker_db
worker:
  pollInterval: 30s
  batchSize: 10
  maxConcurrency: 5
  processTimeout: 5m
```

### Environment Variable Overrides

Use the `WORKER_` prefix for environment variables:

```bash
export WORKER_DATABASE_HOST=production-db.example.com
export WORKER_DATABASE_PASSWORD=secure-password
export WORKER_WORKER_BATCHSIZE=20
export WORKER_WORKER_MAXCONCURRENCY=10
```

## Job Types

The worker supports multiple job types:

### Send Email Jobs
```sql
INSERT INTO jobs (id, type, payload, status) VALUES 
('email-1', 'send_email', '{"to": "user@example.com", "subject": "Welcome", "body": "Hello!"}', 'pending');
```

### Data Processing Jobs  
```sql
INSERT INTO jobs (id, type, payload, status) VALUES 
('data-1', 'process_data', '{"dataset": "user_analytics", "date": "2023-01-01"}', 'pending');
```

### Report Generation Jobs
```sql
INSERT INTO jobs (id, type, payload, status) VALUES 
('report-1', 'generate_report', '{"type": "monthly", "department": "sales"}', 'pending');
```

## Job Status Flow

1. **pending** → **processing** → **completed**
2. **pending** → **processing** → **failed** (with retry logic)
3. Jobs with `retry_count >= max_retries` are permanently failed

## Database Schema

The worker automatically creates a `jobs` table:

```sql
CREATE TABLE jobs (
    id VARCHAR PRIMARY KEY,
    type VARCHAR NOT NULL,
    payload TEXT,
    status VARCHAR DEFAULT 'pending',
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    processed_at TIMESTAMP,
    retry_count INTEGER DEFAULT 0,
    max_retries INTEGER DEFAULT 3,
    error_msg TEXT
);
```

## Adding Custom Job Types

1. **Add Job Handler**
   ```go
   func processCustomJob(ctx context.Context, services *Services, job *Job) error {
       logger := logging.ContextLogger(ctx, "custom-processor")
       
       // Parse job payload
       var payload CustomJobPayload
       if err := json.Unmarshal([]byte(job.Payload), &payload); err != nil {
           return fmt.Errorf("failed to parse payload: %w", err)
       }
       
       // Process the job
       logger.Info().Interface("payload", payload).Msg("Processing custom job")
       
       // Your custom logic here
       
       return nil
   }
   ```

2. **Register in processJob**
   ```go
   switch job.Type {
   case "send_email":
       err = processSendEmailJob(ctx, services, job)
   case "custom_job":
       err = processCustomJob(ctx, services, job)
   // ... other cases
   }
   ```

## Monitoring and Observability

### Logging
All job processing includes structured logging:
- Job start/completion
- Processing duration
- Error details
- Batch statistics

### Metrics
The worker logs key metrics:
- Batch processing counts
- Success/failure rates
- Processing durations

### Health Checks
Monitor worker health by checking:
- Database connectivity
- Job processing rates
- Error rates

## Production Deployment

### Docker Build
```dockerfile
FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o worker .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/worker .
COPY --from=builder /app/config.yaml .
CMD ["./worker"]
```

### Environment Configuration
```bash
# Production environment variables
export WORKER_ENVIRONMENT=production
export WORKER_DEBUG=false
export WORKER_DATABASE_HOST=prod-db.example.com
export WORKER_DATABASE_PASSWORD=secure-password
export WORKER_EXTERNAPI_APIKEY=production-api-key
```

### Scaling
- Run multiple worker instances for horizontal scaling
- Each worker processes jobs independently
- Database handles job locking automatically

## Error Handling and Retry Logic

### Automatic Retries
- Jobs are retried up to `max_retries` times
- Retry count is incremented on failures
- Failed jobs with max retries are marked as permanently failed

### Error Monitoring
```go
// Custom error handling
if err := processJob(ctx, services, job); err != nil {
    // Log detailed error information
    logger.Error().
        Err(err).
        Str("job_id", job.ID).
        Str("job_type", job.Type).
        Int("retry_count", job.RetryCount).
        Msg("Job processing failed")
    
    // Implement custom alerting if needed
    if job.RetryCount >= job.MaxRetries {
        sendFailureAlert(job, err)
    }
}
```

## Performance Tuning

### Database Optimization
- Adjust connection pool sizes based on load
- Add indexes on frequently queried columns:
  ```sql
  CREATE INDEX idx_jobs_status_created ON jobs(status, created_at);
  CREATE INDEX idx_jobs_retry ON jobs(retry_count, max_retries);
  ```

### Concurrency Tuning
- Adjust `maxConcurrency` based on job complexity and resources
- Monitor CPU and memory usage during peak loads
- Consider job-specific concurrency limits

### Batch Size Optimization
- Larger batch sizes reduce database queries
- Smaller batch sizes provide better responsiveness
- Monitor processing times to find optimal balance

## Troubleshooting

### Common Issues

**Jobs not being processed**
- Check database connectivity
- Verify job status is 'pending'
- Check retry count hasn't exceeded max_retries

**High error rates**
- Review external API connectivity
- Check payload format validation
- Monitor resource usage (CPU, memory)

**Slow processing**
- Optimize individual job handlers
- Adjust concurrency settings
- Check database query performance

### Debugging
Enable debug logging:
```bash
export WORKER_DEBUG=true
```

Check job status:
```sql
SELECT status, COUNT(*) FROM jobs GROUP BY status;
SELECT * FROM jobs WHERE status = 'failed' ORDER BY updated_at DESC LIMIT 10;
```

## Support

For issues with this template or the underlying library:
- Template issues: Create issue in your project repository  
- Library issues: https://github.com/jasoet/pkg/issues
- Documentation: Check the [integration guide](../../.claude/integration-guide.md)