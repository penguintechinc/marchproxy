# DBLB Usage Guide

Comprehensive guide for using MarchProxy Database Load Balancer (DBLB) for database protocol proxying and load balancing.

## Quick Start

### 1. Start DBLB

Using Docker:
```bash
docker run -d \
  --name dblb \
  -p 3306:3306 \
  -p 5432:5432 \
  -p 7002:7002 \
  -v /path/to/config.yaml:/app/config.yaml:ro \
  -e CLUSTER_API_KEY=your-api-key \
  marchproxy/dblb:latest
```

Or standalone:
```bash
./proxy-dblb --config config.yaml
```

### 2. Verify Health

```bash
curl http://localhost:7002/healthz
```

Response:
```json
{
  "status": "healthy",
  "module": "dblb"
}
```

### 3. Connect to Database

Connect through DBLB instead of directly to backend:

```bash
# Before (direct connection)
mysql -h backend-server -P 3306 -u user -p

# After (through DBLB)
mysql -h localhost -P 3306 -u user -p
```

## Basic Usage Examples

### MySQL Routing

**Configuration:**
```yaml
routes:
  - name: "mysql-primary"
    protocol: "mysql"
    listen_port: 3306
    backend_host: "mysql.internal"
    backend_port: 3306
    max_connections: 100
    enable_auth: false
```

**Connection:**
```bash
mysql -h localhost -P 3306 -u root
```

**Query:**
```sql
SELECT VERSION();
```

### PostgreSQL Routing

**Configuration:**
```yaml
routes:
  - name: "postgres-main"
    protocol: "postgresql"
    listen_port: 5432
    backend_host: "postgres.internal"
    backend_port: 5432
    enable_auth: true
    username: "dbuser"
    password: "${POSTGRESQL_PASSWORD}"
```

**Connection:**
```bash
psql -h localhost -p 5432 -U dbuser -d mydb
```

**Query:**
```sql
SELECT version();
```

### MongoDB Routing

**Configuration:**
```yaml
routes:
  - name: "mongodb-cluster"
    protocol: "mongodb"
    listen_port: 27017
    backend_host: "mongodb.internal"
    backend_port: 27017
    enable_auth: true
    username: "dbuser"
    password: "${MONGODB_PASSWORD}"
```

**Connection:**
```bash
mongosh "mongodb://dbuser:password@localhost:27017"
```

**Query:**
```javascript
db.collection.find({})
```

### Redis Routing

**Configuration:**
```yaml
routes:
  - name: "redis-cache"
    protocol: "redis"
    listen_port: 6379
    backend_host: "redis.internal"
    backend_port: 6379
    enable_auth: true
    password: "${REDIS_PASSWORD}"
```

**Connection:**
```bash
redis-cli -h localhost -p 6379 -a password
```

**Query:**
```
GET mykey
SET mykey myvalue
```

### MSSQL Routing

**Configuration:**
```yaml
routes:
  - name: "mssql-enterprise"
    protocol: "mssql"
    listen_port: 1433
    backend_host: "mssql.internal"
    backend_port: 1433
    enable_auth: true
    username: "sa"
    password: "${MSSQL_PASSWORD}"
    enable_ssl: true
```

**Connection:**
```bash
sqlcmd -S localhost,1433 -U sa -P password
```

**Query:**
```sql
SELECT @@VERSION
```

## Monitoring and Observability

### Health Monitoring

**Basic Health Check:**
```bash
curl http://localhost:7002/healthz
```

**Detailed Status:**
```bash
curl http://localhost:7002/status | jq .
```

Response shows per-route statistics:
```json
{
  "module": "dblb",
  "status": "healthy",
  "handlers": {
    "mysql-primary": {
      "protocol": "mysql",
      "connections": {
        "active": 15,
        "total": 250,
        "idle": 5
      }
    }
  }
}
```

### Prometheus Metrics

**View Metrics:**
```bash
curl http://localhost:7002/metrics
```

**Key Metrics:**
```
# Active connections per route
marchproxy_dblb_connections_active{route="mysql-primary"} 15

# Total connections (lifetime)
marchproxy_dblb_connections_total{route="mysql-primary"} 250

# Blocked queries due to security
marchproxy_dblb_queries_blocked{route="mysql-primary"} 2

# Connection pool size
marchproxy_dblb_pool_size{route="mysql-primary"} 20
```

### Prometheus Scraping

Add to Prometheus config:
```yaml
scrape_configs:
  - job_name: 'dblb'
    static_configs:
      - targets: ['localhost:7002']
    metrics_path: '/metrics'
    scrape_interval: 15s
```

### Grafana Dashboards

Create dashboards using these metrics:

**Active Connections:**
```promql
marchproxy_dblb_connections_active{route="$route"}
```

**Query Rate:**
```promql
rate(marchproxy_dblb_queries_total{route="$route"}[5m])
```

**Blocked Queries:**
```promql
rate(marchproxy_dblb_queries_blocked{route="$route"}[5m])
```

## Performance Tuning

### Connection Pool Optimization

**For High Throughput (> 1000 connections/sec):**
```yaml
max_connections_per_route: 500
connection_idle_timeout: 10m
connection_max_lifetime: 60m
default_connection_rate: 500.0
default_query_rate: 5000.0
```

**For Low Latency (< 100ms p99):**
```yaml
max_connections_per_route: 50
connection_idle_timeout: 1m
connection_max_lifetime: 10m
default_connection_rate: 100.0
default_query_rate: 1000.0
```

**For Balance (typical production):**
```yaml
max_connections_per_route: 100
connection_idle_timeout: 5m
connection_max_lifetime: 30m
default_connection_rate: 100.0
default_query_rate: 1000.0
```

### Rate Limiting Usage

**Check Rate Limits:**
```bash
curl http://localhost:7002/status | jq '.handlers.mysql-primary.rates'
```

Response:
```json
{
  "connection_rate": 50.0,
  "query_rate": 500.0
}
```

**Monitor Rate Limit Events:**
```bash
curl http://localhost:7002/metrics | grep rate_limited
```

## Security Usage

### SQL Injection Detection

DBLB automatically detects and blocks malicious queries:

**Configuration:**
```yaml
enable_sql_injection_detection: true
block_suspicious_queries: true
```

**Behavior:**
- Suspicious queries are blocked
- Log entries created for blocked queries
- Metric incremented: `marchproxy_dblb_queries_blocked`

**Example Blocked Patterns:**
```sql
-- Union-based injection
SELECT * FROM users WHERE id = 1 UNION SELECT password FROM admin

-- Comment injection
SELECT * FROM users WHERE id = 1 -- SELECT * FROM secrets

-- Time-based blind injection
SELECT * FROM users WHERE id = 1 AND SLEEP(5)

-- Stacked queries
SELECT * FROM users; DROP TABLE users;
```

### TLS/SSL Configuration

**For Secure Backend Connections:**
```yaml
routes:
  - name: "postgres-secure"
    protocol: "postgresql"
    enable_ssl: true
    ssl_verify: true
    ssl_ca_cert: "/path/to/ca.crt"
```

**Client Connection (still plain text to DBLB):**
```bash
psql -h localhost -p 5432 -U user
```

**Verify SSL to Backend:**
```bash
# View connection details
curl http://localhost:7002/status | jq '.handlers.postgres-secure'
```

## Troubleshooting

### Connection Issues

**Symptom: "Connection refused"**

1. Verify DBLB is running:
```bash
curl http://localhost:7002/healthz
```

2. Check route configuration:
```bash
curl http://localhost:7002/status | jq '.handlers'
```

3. Verify backend is accessible:
```bash
mysql -h backend-host -P 3306 -u user -p
```

**Symptom: "Too many connections"**

1. Check active connections:
```bash
curl http://localhost:7002/status | jq '.handlers.mysql-primary.connections'
```

2. Increase pool size:
```yaml
max_connections: 200
```

3. Check application connection leaks

### Performance Issues

**Symptom: High latency**

1. Check connection pool exhaustion:
```bash
curl http://localhost:7002/metrics | grep pool_size
```

2. Increase pool size if needed
3. Check backend database performance
4. Review query logs for slow queries

**Symptom: Low throughput**

1. Check rate limiting:
```bash
curl http://localhost:7002/status | jq '.handlers.mysql-primary.rates'
```

2. Adjust rate limits if necessary
3. Check backend capacity
4. Review metrics for bottlenecks

### Query Blocked Issues

**Symptom: Query returns "403 Forbidden"**

1. Check if SQL injection detection is enabled:
```bash
curl http://localhost:7002/status | jq '.handlers.mysql-primary'
```

2. Review logs for blocked query patterns:
```bash
docker logs dblb | grep "blocked\|injection"
```

3. Disable injection detection if false positive:
```yaml
enable_sql_injection_detection: false
```

## Integration Patterns

### Application Integration

**Direct Connection Replacement:**
```python
# Before
import mysql.connector
conn = mysql.connector.connect(
    host="backend-server",
    port=3306,
    user="user",
    password="pass"
)

# After (through DBLB)
import mysql.connector
conn = mysql.connector.connect(
    host="localhost",
    port=3306,
    user="user",
    password="pass"
)
```

### Multi-Database Routing

**Configuration for Multiple Databases:**
```yaml
routes:
  - name: "mysql-primary"
    protocol: "mysql"
    listen_port: 3306
    backend_host: "mysql.internal"
    backend_port: 3306

  - name: "postgres-main"
    protocol: "postgresql"
    listen_port: 5432
    backend_host: "postgres.internal"
    backend_port: 5432

  - name: "mongodb-cluster"
    protocol: "mongodb"
    listen_port: 27017
    backend_host: "mongodb.internal"
    backend_port: 27017
```

**Application Connects to All:**
```python
import mysql.connector
import psycopg2
import pymongo

mysql_conn = mysql.connector.connect(host="localhost", port=3306)
postgres_conn = psycopg2.connect("host=localhost port=5432")
mongo_conn = pymongo.MongoClient("mongodb://localhost:27017")
```

### Kubernetes Deployment

**Example Pod with DBLB:**
```yaml
apiVersion: v1
kind: Pod
metadata:
  name: app-with-dblb
spec:
  containers:
  - name: dblb
    image: marchproxy/dblb:latest
    ports:
    - containerPort: 3306  # MySQL
    - containerPort: 5432  # PostgreSQL
    - containerPort: 7002  # Metrics
    volumeMounts:
    - name: config
      mountPath: /app/config.yaml
      subPath: config.yaml
    env:
    - name: CLUSTER_API_KEY
      valueFrom:
        secretKeyRef:
          name: dblb-secret
          key: api-key

  - name: app
    image: myapp:latest
    env:
    - name: DB_HOST
      value: "127.0.0.1"
    - name: DB_PORT
      value: "3306"
    dependsOn:
    - dblb

  volumes:
  - name: config
    configMap:
      name: dblb-config
```

## Best Practices

1. **Monitor Health**: Regularly check `/healthz` and `/metrics`
2. **Set Appropriate Timeouts**: Balance between connection reuse and staleness
3. **Enable Security Features**: Use SQL injection detection in production
4. **Right-Size Pools**: Match pool size to workload
5. **Use TLS for Sensitive Data**: Enable SSL for backend connections
6. **Log All Activity**: Set `log_level: debug` during troubleshooting
7. **Test Configuration**: Validate config before deployment
8. **Plan for Failover**: Use health checks for automatic recovery
9. **Monitor Metrics**: Track connection and query rates
10. **Document Routes**: Comment configuration for clarity

## Support and Resources

- **Configuration**: See `CONFIGURATION.md`
- **API Reference**: See `API.md`
- **Testing**: See `TESTING.md`
- **Release Notes**: See `RELEASE_NOTES.md`
- **GitHub Issues**: https://github.com/penguintech/marchproxy/issues
- **Documentation**: https://docs.marchproxy.io
- **Website**: https://www.penguintech.io
