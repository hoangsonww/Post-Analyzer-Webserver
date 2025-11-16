# Migration Guide: v1.0 to v2.0 (Production Ready)

This guide will help you migrate from the simple version (v1.0) to the production-ready version (v2.0) of the Post Analyzer Webserver.

## What's Changed?

### Major Changes

1. **Application Architecture**
   - Restructured into modular packages
   - Introduced storage abstraction layer
   - Added comprehensive middleware stack

2. **Storage Layer**
   - File storage now has thread-safe operations
   - Added PostgreSQL support
   - Automatic schema management

3. **Configuration**
   - Environment-based configuration
   - Validation on startup
   - Support for multiple environments

4. **Observability**
   - Structured JSON logging
   - Prometheus metrics
   - Request tracing with IDs

5. **Security**
   - Input validation and sanitization
   - Rate limiting
   - Security headers
   - CORS configuration

## Migration Steps

### Step 1: Backup Your Data

If you're using file storage:

```bash
# Backup your existing posts.json
cp posts.json posts.json.backup
```

### Step 2: Update Dependencies

```bash
# Download new dependencies
go mod download
go mod tidy
```

### Step 3: Configuration

Create a `.env` file from the example:

```bash
cp .env.example .env
```

Edit `.env` to match your setup. For file storage (similar to v1.0):

```env
# Keep using file storage
DB_TYPE=file
DB_FILE_PATH=posts.json

# Other settings
PORT=8080
ENVIRONMENT=production
LOG_LEVEL=info
```

For PostgreSQL (recommended for production):

```env
# Use PostgreSQL
DB_TYPE=postgres
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=yourpassword
DB_NAME=postanalyzer

# Other settings
PORT=8080
ENVIRONMENT=production
LOG_LEVEL=info
```

### Step 4: Run the Application

#### Option A: Direct Run (File Storage)

```bash
# Run with default file storage
go run main.go
```

#### Option B: With PostgreSQL

```bash
# Start PostgreSQL with Docker
docker run -d \
  --name postgres \
  -e POSTGRES_DB=postanalyzer \
  -e POSTGRES_PASSWORD=postgres \
  -p 5432:5432 \
  postgres:16-alpine

# Run the application
export DB_TYPE=postgres
export DB_PASSWORD=postgres
go run main.go
```

#### Option C: Full Docker Stack

```bash
# Start everything with Docker Compose
docker-compose up -d
```

This includes:
- Application
- PostgreSQL
- Prometheus
- Grafana

### Step 5: Migrate Existing Data

If you have existing data in `posts.json` and want to move to PostgreSQL:

```bash
# The application will automatically handle this
# Just ensure posts.json exists in the same directory
# On first run with DB_TYPE=postgres, the data will be available
```

Or manually import:

```bash
# Start application with file storage first
DB_TYPE=file go run main.go

# In another terminal, fetch to ensure data is in posts.json
curl http://localhost:8080/fetch

# Stop the application (Ctrl+C)

# Start with PostgreSQL
DB_TYPE=postgres DB_PASSWORD=postgres go run main.go

# Fetch again to populate PostgreSQL
curl http://localhost:8080/fetch
```

## Compatibility

### API Endpoints

All original endpoints remain functional:

| v1.0 Endpoint | v2.0 Status | Notes |
|---------------|-------------|-------|
| `/` | ✅ Compatible | Enhanced with better error handling |
| `/fetch` | ✅ Compatible | Now supports batch operations |
| `/analyze` | ✅ Compatible | Improved performance |
| `/add` | ✅ Compatible | Added input validation |

### New Endpoints

| Endpoint | Purpose |
|----------|---------|
| `/health` | Health check for monitoring |
| `/readiness` | Kubernetes-style readiness probe |
| `/metrics` | Prometheus metrics |

### File Format

The `posts.json` file format remains compatible:

```json
[
  {
    "userId": 1,
    "id": 1,
    "title": "Post Title",
    "body": "Post body content"
  }
]
```

v2.0 adds optional fields:
- `createdAt`: Timestamp when post was created
- `updatedAt`: Timestamp when post was last updated

These are automatically managed and backward compatible.

## Feature Comparison

| Feature | v1.0 | v2.0 |
|---------|------|------|
| Post Management | ✅ | ✅ |
| Character Analysis | ✅ | ✅ (faster) |
| External API Fetch | ✅ | ✅ |
| File Storage | ✅ | ✅ (improved) |
| Database Support | ❌ | ✅ |
| Health Checks | ❌ | ✅ |
| Metrics | ❌ | ✅ |
| Structured Logging | ❌ | ✅ |
| Input Validation | ❌ | ✅ |
| Rate Limiting | ❌ | ✅ |
| Security Headers | ❌ | ✅ |
| CORS | ❌ | ✅ |
| Graceful Shutdown | ❌ | ✅ |
| Docker Support | ❌ | ✅ |
| CI/CD Pipeline | ❌ | ✅ |
| Test Suite | ❌ | ✅ |
| API Documentation | ❌ | ✅ |

## Troubleshooting

### Issue: Application Won't Start

**Error**: `invalid configuration: environment must be one of: development, staging, production`

**Solution**: Set the ENVIRONMENT variable:
```bash
export ENVIRONMENT=development
```

### Issue: Database Connection Failed

**Error**: `failed to ping database`

**Solution**: Ensure PostgreSQL is running:
```bash
# Check if PostgreSQL is running
docker ps | grep postgres

# Or check locally
pg_isready -h localhost
```

### Issue: Posts Not Showing

**Symptom**: Empty home page

**Solution**: Fetch posts first:
```bash
curl http://localhost:8080/fetch
```

### Issue: Rate Limited

**Error**: 429 Too Many Requests

**Solution**: Increase rate limit:
```bash
export RATE_LIMIT_REQUESTS=1000
```

Or wait for the rate limit window to reset (default: 1 minute).

## Performance Considerations

### File Storage vs PostgreSQL

**File Storage**:
- ✅ Simple setup
- ✅ No external dependencies
- ❌ Not suitable for high concurrency
- ❌ No advanced querying

**PostgreSQL**:
- ✅ High concurrency support
- ✅ ACID transactions
- ✅ Advanced querying
- ❌ Requires external service

**Recommendation**: Use file storage for development, PostgreSQL for production.

## Security Updates

v2.0 includes important security improvements:

1. **Input Sanitization**: All user input is sanitized
2. **Rate Limiting**: Prevents abuse
3. **Security Headers**: CSP, X-Frame-Options, etc.
4. **Request Timeouts**: Prevents resource exhaustion

**Action Required**: Review your CORS configuration in `.env`:
```env
# Development - allow all
ALLOWED_ORIGINS=*

# Production - specify allowed origins
ALLOWED_ORIGINS=https://yourdomain.com,https://www.yourdomain.com
```

## Monitoring Setup

### Basic Monitoring (Logs)

```bash
# View logs in production
tail -f /path/to/app/logs

# Or with Docker
docker logs -f post-analyzer-app
```

### Advanced Monitoring (Prometheus + Grafana)

```bash
# Start monitoring stack
docker-compose up -d prometheus grafana

# Access Grafana
open http://localhost:3000
# Login: admin/admin
```

Add Prometheus data source:
1. Go to Configuration → Data Sources
2. Add Prometheus
3. URL: `http://prometheus:9090`
4. Save & Test

Import dashboard from `grafana-dashboard.json` (if provided).

## Rollback Plan

If you need to rollback to v1.0:

```bash
# Stop v2.0
pkill post-analyzer

# Restore old version
mv main.go main_v2.go
mv main_old.go main.go

# Run v1.0
go run main.go
```

Your data in `posts.json` remains compatible.

## Getting Help

- 📖 [Full Documentation](README_PRODUCTION.md)
- 🐛 [Report Issues](https://github.com/hoangsonww/Post-Analyzer-Webserver/issues)
- 💬 [Discussions](https://github.com/hoangsonww/Post-Analyzer-Webserver/discussions)

## Next Steps

After successful migration:

1. ✅ Review configuration in `.env`
2. ✅ Set up monitoring (Prometheus/Grafana)
3. ✅ Configure backups (for PostgreSQL)
4. ✅ Set up CI/CD pipeline
5. ✅ Review security settings
6. ✅ Load test your application
7. ✅ Set up log aggregation

---

**Need assistance?** Open an issue on GitHub!
