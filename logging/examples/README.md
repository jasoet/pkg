# Logging Examples

Runnable examples demonstrating the logging package features.

## Running Examples

All examples use the `example` build tag:

```bash
# Console only output
go run -tags=example ./logging/examples/console

# File only output
go run -tags=example ./logging/examples/file

# Both console and file output
go run -tags=example ./logging/examples/both

# Environment-based configuration
go run -tags=example ./logging/examples/environment
ENV=staging go run -tags=example ./logging/examples/environment
ENV=production go run -tags=example ./logging/examples/environment
```

## Examples Overview

### 1. Console Example (`console/`)

Demonstrates basic console-only logging (default behavior):
- Global logger usage
- Structured logging with fields
- Different log levels
- Component loggers

**Run:**
```bash
go run -tags=example ./logging/examples/console
```

### 2. File Example (`file/`)

Demonstrates file-only logging:
- Writing logs to a file
- JSON format output
- No console output
- Reading log file contents

**Run:**
```bash
go run -tags=example ./logging/examples/file
```

### 3. Both Example (`both/`)

Demonstrates dual output (console + file):
- Simultaneous console and file logging
- Component loggers
- Structured logging
- Human-readable console vs JSON file

**Run:**
```bash
go run -tags=example ./logging/examples/both
```

### 4. Environment Example (`environment/`)

Demonstrates environment-based configuration:
- Development: console only, debug level
- Staging: console + file, debug level
- Production: file only, info level

**Run:**
```bash
# Development (default)
go run -tags=example ./logging/examples/environment

# Staging
ENV=staging go run -tags=example ./logging/examples/environment

# Production
ENV=production go run -tags=example ./logging/examples/environment
```

## Output Formats

### Console Output

Human-readable with colors:
```
2025-11-24T12:30:45+07:00 INF Service started service=console-example pid=12345
2025-11-24T12:30:45+07:00 DBG Running in debug mode mode=development service=console-example pid=12345
```

### File Output

Structured JSON:
```json
{"level":"info","service":"file-example","pid":12345,"time":"2025-11-24T12:30:45+07:00","message":"Application started"}
{"level":"info","service":"file-example","pid":12345,"user_id":"67890","action":"login","time":"2025-11-24T12:30:45+07:00","message":"User action"}
```

## Learning Path

1. **Start with `console/`** - Understand basic logging
2. **Try `file/`** - See JSON output format
3. **Explore `both/`** - Learn dual output and component loggers
4. **Study `environment/`** - Apply environment-based patterns

## Integration Examples

For examples integrating logging with other packages (server, otel, etc.), see:
- [`examples/`](../../examples/) - Full-stack examples
- [`otel/README.md`](../../otel/README.md) - OpenTelemetry integration
- [`server/README.md`](../../server/README.md) - HTTP server logging
