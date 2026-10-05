# Amazon CloudWatch Database Insights Exporter for Prometheus

A Prometheus exporter that provides Aurora/RDS Performance Insights metrics with auto-discovery capabilities.

As of Q4 2025, the exporter remains under active development. Current filtering capabilities include instance-level filtering (by identifier and engine), metric-level filtering, and tag-based instance filtering. 

## Features

- **Auto-discovery**: Automatically discovers Aurora/RDS instances in specified AWS regions
- **Instance filtering**: Query metrics for specific instances using URL parameters
- **Prometheus-compatible**: Standard `/metrics` endpoint with Prometheus format
- **Health endpoint**: Lightweight `/health` endpoint for Kubernetes liveness and readiness probes
- **Low-latency collection**: Efficient metric collection from Amazon RDS Performance Insights API
- **Simple configuration**: YAML-based configuration with sensible defaults

## Prerequisites

- **Go 1.23 or later** (for building from source)
- **AWS credentials configured**
- **Required AWS permissions**:
  - `rds:DescribeDBInstances`
  - `pi:ListAvailableResourceMetrics`
  - `pi:GetResourceMetrics`
  - `sts:AssumeRole` (only when using cross-account `role_arn`)

## Quick Start

1. **Create configuration file if not exists** (`config.yml`):
   ```yaml
   discovery:
     regions:
       - "us-west-2"
   ```

2. **Build and run**:
   ```bash
   go build -o dbinsights-exporter ./cmd
   ./dbinsights-exporter
   ```

3. **Access metrics**:
   ```bash
   curl http://localhost:8081/metrics
   ```

## Command-Line Flags

The exporter supports the following command-line flags:

| Flag | Default | Description |
|------|---------|-------------|
| `-config` | `config.yml` | Path to configuration file |
| `-debug` | `false` | Enable debug logging for cache operations |

**Examples**:
```bash
# Use default config.yml in current directory
./dbinsights-exporter

# Specify custom config file path
./dbinsights-exporter -config /etc/dbinsights/my-config.yml

# Enable debug mode
./dbinsights-exporter -debug

# Combine flags
./dbinsights-exporter -config /path/to/config.yml -debug
```

## Debug Mode

Enable debug logging to see detailed cache behavior including cache hits, misses, and expiration events:

```bash
# Enable debug mode with -debug flag
./dbinsights-exporter -debug
```

**Debug logs include**:
- Instance-Discovery Cache: Expiration and refresh events
- Metric-Metadata Cache: Expiration and refresh events for each instance
- Metric-Data Cache: Cache hits, misses, and updates for metric values

**Example debug output**:
```
[MAIN] Debug mode enabled
[MAIN] Loaded configuration from: config.yml
[DEBUG] Instance-Discovery Cache Expired, fetching new instance list from AWS RDS
[DEBUG] Instance-Discovery Cache Updated, cached 25 instances
[DEBUG] Metric-Metadata Cache Expired for instance: prod-db-1, fetching new metric definitions from AWS
[DEBUG] Metric-Metadata Cache Updated for instance: prod-db-1, cached 47 metrics
[DEBUG] Metric-Data Cache Expired: instance=prod-db-2, expired_metrics=15/15, fetching from AWS
[DEBUG] Metric-Data Cache Updated: instance=prod-db-2, new_entries=15
```

## Configuration

The DB Insights Exporter uses a YAML configuration file for all settings. By default, it looks for `config.yml` in the current directory, but you can specify a custom path using the `-config` flag.

```bash
# Use default config.yml
./dbinsights-exporter

# Use custom config file
./dbinsights-exporter -config /path/to/my-config.yml
```

## YAML Configuration File

The file is written in YAML format:

```yaml
# Example complete configuration
discovery:
  regions:
    - "us-west-2"
  role_arn: "arn:aws:iam::123456789012:role/pi-exporter-role"  # optional: assume role for cross-account access
  role_arn_external_id: "my-external-id"                       # optional: ExternalID for confused-deputy mitigation
  instances:
    max-instances: 25
    cache:
      ttl: "5m"
    include:
      identifier: ["^(prod|staging)-"]
      engine: ["^postgres", "aurora-postgresql"]
    exclude:
      identifier: ["-temp-", "-test$"]
  metrics:
    statistic: "avg"
    cache:
      metric-metadata-ttl: "60m"
      metric-data:
        max-size: 100000
        pattern-ttls:
          - pattern: ".*/db\\.load\\..*"
            ttl: "1s"
          - pattern: ".*/os\\.memory\\..*"
            ttl: "10m"
    include:
      name: ["^(db|os)\\.", ".*\\.max$"]
      category: ["os", "db"]
    exclude:
      name: ["\\.idle$", "\\.wait$"]
  processing:
    concurrency: 4

export:
  port: 8081
  prometheus:
    metric-prefix: "aws_rds_pi_"
```

### Configuration Reference

#### `discovery` section
Controls how the exporter discovers and monitors RDS/Aurora instances.

| Field | Type | Required/Optional | Default | Description |
|-------|------|------------------|---------|-------------|
| `regions` | array | Required | `["us-west-2"]` | List of AWS regions to scan for RDS/Aurora instances. **Note**: Only the first region is currently used (single-region support only) |
| `role_arn` | string | Optional | `""` | IAM role ARN to assume before calling AWS APIs. Use for cross-account access. The exporter's own IAM identity (e.g., IRSA service account) must have `sts:AssumeRole` permission on this role. If empty, the default credential chain is used. Format: `arn:(aws\|aws-cn\|aws-us-gov):iam::<12-digit-account>:role/<name>`. Role name segment must not exceed 64 characters. |
| `role_arn_external_id` | string | Optional | `""` | ExternalID condition for the AssumeRole call (confused-deputy mitigation). Only valid when `role_arn` is also set. Must match the `sts:ExternalId` condition in the target account's trust policy. Must be 2–1224 characters. Allowed characters: `[a-zA-Z0-9_+=,.@:/-]`. |
| `instances.max-instances` | integer | Optional | `25` | Maximum number of instances to monitor (accepted range: 1–10,000; values above 10,000 are capped at 10,000). When this limit is exceeded, only the oldest `max-instances` are selected |
| `instances.cache.ttl` | string | Optional | `"5m"` | Time-to-live for cached instance discovery results. How long to cache the list of RDS/Aurora instances before re-discovering |
| `instances.include` | map | Optional | `{}` | Map of field names to regex pattern arrays for instance filtering (allowlist mode). Supported fields: `identifier`, `engine`, `tag.<TagKey>` (e.g., `tag.Environment`, `tag.Team`) |
| `instances.exclude` | map | Optional | `{}` | Map of field names to regex pattern arrays for instance filtering (denylist mode). Supported fields: `identifier`, `engine`, `tag.<TagKey>` (e.g., `tag.Status`, `tag.Maintenance`) |
| `metrics.statistic` | string | Required | `"avg"` | Default statistic aggregation for Performance Insights metrics |
| `metrics.cache.metric-metadata-ttl` | string | Optional | `"60m"` | Time-to-live for cached metric definitions. How long to cache the list of available metrics for each instance |
| `metrics.cache.metric-data.max-size` | integer | Optional | `100000` | Maximum number of metric values to cache. When exceeded, oldest entries are evicted |
| `metrics.cache.metric-data.pattern-ttls` | array | Optional | `[]` | Pattern-based TTL overrides for specific metrics. Patterns match against metric names (e.g., `db.load.avg`). If pattern TTL is smaller than the metric's data interval, the dynamic TTL will be used instead. If no pattern matches, TTL is calculated dynamically based on metric granularity |
| `metrics.include` | map | Optional | `{}` | Map of field names to regex pattern arrays for metric filtering (allowlist mode). Supported fields: `name`, `category`, `unit` |
| `metrics.exclude` | map | Optional | `{}` | Map of field names to regex pattern arrays for metric filtering (denylist mode). Supported fields: `name`, `category`, `unit` |
| `processing.concurrency` | integer | Optional | `4` | Number of concurrent goroutines for metric collection (accepted range: 1–60; values above 60 are capped at 60) |

**Valid statistic values:**
- `"avg"` - Average values
- `"min"` - Minimum values
- `"max"` - Maximum values
- `"sum"` - Sum of values

**TTL Duration Format:**
- `"30s"` - 30 seconds
- `"5m"` - 5 minutes
- `"1h"` - 1 hour
- `"24h"` - 24 hours

#### `export` section
Controls how metrics are exposed via HTTP endpoint.

| Field | Type | Required/Optional | Default | Description |
|-------|------|------------------|---------|-------------|
| `port` | integer | Required | `8081` | HTTP port number for the Prometheus metrics endpoint |
| `prometheus.metric-prefix` | string | Optional | `"dbi_"` | Prefix added to all exported Prometheus metric names |

**Pattern Format for TTL Overrides**:

Patterns are regex expressions that match against metric names with statistics in the format:
```
<metric-name>.<statistic>
```

**Example metric names**:
- `db.load.avg` - Database load average
- `os.cpuUtilization.user.avg` - CPU user time average
- `db.SQL.queries.sum` - SQL queries sum
- `os.memory.total.max` - Memory total maximum

**Example patterns**:
- `^db\\.load\\..*` - Matches all `db.load` metrics (any statistic)
- `^os\\.memory\\..*` - Matches all `os.memory` metrics
- `^db\\.SQL\\..*\\.sum$` - Matches all `db.SQL` metrics with sum statistic
- `.*\\.avg$` - Matches any metric with avg statistic
- `^os\\.cpuUtilization\\.(user|system)\\..*` - Matches CPU user or system metrics

**Pattern tips**:
- Use `^` to match from the beginning of the metric name
- Use `$` to match to the end (useful for matching specific statistics)
- Use `\\.` to match literal dots in metric names
- Use `.*` to match any characters
- Patterns are evaluated in order - first match wins
- If pattern TTL is smaller than the metric's actual data interval (dynamic TTL), the dynamic TTL will be used instead to respect metric granularity

**Dynamic TTL Calculation**: When no pattern matches a metric, the system automatically calculates an appropriate TTL by:
1. Examining the last two valid datapoints from the metric's 3-minute history
2. Calculating the time difference between these datapoints
3. Using this difference as the TTL (adapts to metric granularity automatically)

This means per-second metrics (like `db.load`) get ~1s TTL, while per-minute metrics get ~60s TTL, without manual configuration.

## Metric Data Caching Recommendations

**IMPORTANT**: Understanding metric data caching behavior is critical for balancing API efficiency with data freshness.

### When NOT to Configure Metric Data Cache

**It is NOT recommended to configure metric data caching** if you need real-time, always-fresh data on every scrape. Without caching configured:
- Every Prometheus scrape fetches the latest available datapoint from AWS Performance Insights
- You always get the most recent metric value
- Higher AWS API usage and costs
- Potential for API throttling with many instances

### When to Configure Metric Data Cache

Configure caching when you want to:
- Reduce AWS API calls and costs
- Avoid API throttling when monitoring many instances
- Accept some data staleness in exchange for efficiency

### Critical Caching Behavior

**Pattern TTL vs Dynamic TTL**:
- If you configure a pattern TTL that is **smaller** than the metric's actual data interval (dynamic TTL), the system will automatically use the dynamic TTL instead
- This prevents caching metrics for less time than their natural update interval
- Example: If `os.cpuUtilization.total` updates every 10 second (dynamic TTL = 10s) but you set pattern TTL to 30s, the cache will use 30s
- Example: If `os.cpuUtilization.total` updates every 10 second (dynamic TTL = 10s) but you set pattern TTL to 1s for a 1-second metric, the system will use 10s (dynamic TTL) instead

**Data Staleness with Caching**:
- When caching is enabled, cached values are served until the TTL expires
- Maximum staleness equals the configured TTL duration
- Example with 30s TTL for `db.load`:
  - Scrape at T=0s: Fetch from AWS, cache with timestamp T=0s, expires at T=30s
  - Scrape at T=15s: Serve cached value (timestamp T=0s) - data is 15s old
  - Scrape at T=31s: Cache expired, fetch from AWS, get value with timestamp T=31s

**Scrape Interval vs Cache TTL**:
- If scrape interval > cache TTL: Most scrapes will be cache misses (fetching fresh data)
- If scrape interval < cache TTL: Most scrapes will be cache hits (serving cached data)
- If scrape interval = cache TTL: Balanced between fresh data and API efficiency

### Configuration Recommendations

**For Real-Time Monitoring** (scrape interval 15-30s):
```yaml
cache:
  metric-data:
    max-size: 100000
    # Option 1: No pattern-ttls = always use dynamic TTL (adapts to metric granularity)
    # Option 2: Short TTLs matching or slightly longer than scrape interval
```

### Best Practices

1. **Start without caching** to understand your metric update patterns
2. **Monitor AWS API usage** to determine if caching is needed
3. **Set TTLs based on**:
   - Your scrape interval
   - Metric update frequency (per-second vs per-minute)
   - Acceptable staleness for your use case
4. **Use dynamic TTL** (no pattern-ttls) to automatically adapt to metric granularity
5. **Set pattern TTLs** only when you need specific caching behavior different from the metric's natural interval

**Metric Data Freshness & Staleness**:

### Understanding Cache Expiration

The metric data cache is designed to balance API efficiency with data freshness:

- **Cache Expiration Based on Metric Timestamp**: Cache entries expire based on when the metric was measured (not when it was cached). This ensures consistent freshness regardless of fetch delays.

- **Per-Second Metrics** (e.g., `db.load`):
  - These metrics update every second in Performance Insights
  - With dynamic TTL (~1s) or short pattern TTL, you always get recent data
  - The cache stores the most recent measurement and serves it until the TTL expires
  - After expiration, the next scrape fetches fresh data with the latest timestamp

- **Per-Minute Metrics** (e.g., some database counters):
  - These metrics update less frequently
  - Dynamic TTL automatically adapts to ~60s based on datapoint intervals
  - Reduces unnecessary API calls for metrics that don't change every second

### Minimal Configuration Example

```yaml
discovery:
  regions:
    - "us-west-2"
```

This minimal configuration will:
- Monitor RDS/Aurora instances in `us-west-2`
- Use `avg` statistic for all metrics
- Serve metrics on port `8081`


## Enhanced Map-Based Filtering Configuration

The exporter provides flexible **map-based filtering** for both **instances** and **metrics** using field-specific regex patterns. This allows you to filter on multiple fields simultaneously and precisely control which database instances are monitored and which metrics are collected.

### Filtering Logic

The filtering system uses **field-based filtering** with the following logic:

#### **AND Logic Across Fields**
All specified fields must match their patterns for an item to be included.

#### **OR Logic Within Field Patterns**
Any pattern within a field's array can match.

#### **Exclude Precedence**
Exclude patterns take precedence over include patterns when both are specified.

### Supported Filter Fields

#### **Instance Fields**
- `identifier` - RDS instance identifier (e.g., "prod-db-1")
- `engine` - Database engine (e.g., "postgres", "aurora-mysql")
- `tag.<TagKey>` - AWS resource tags (e.g., "tag.Environment", "tag.Team", "tag.CostCenter")

#### **Metric Fields**
- `name` - Performance Insights metric name (e.g., "db.SQL.queries", "os.cpuUtilization.user")
- `category` - Metric category (e.g., "os", "db")
- `unit` - Metric unit (e.g., "Percent", "Count", "Bytes")

### Configuration Examples

#### **1. No Patterns = Include Everything**
```yaml
discovery:
  instances: {}  # No include/exclude patterns
  metrics: {}    # No include/exclude patterns
```
- **Result**: All discovered instances and all available metrics are included

#### **2. Include Patterns Only = Allowlist Mode**
```yaml
discovery:
  instances:
    include:
      identifier: ["^(prod|staging)-"]    # Production or staging instances
      engine: ["postgres", "aurora-postgresql"]  # PostgreSQL engines only
  metrics:
    include:
      name: ["^(db|os)\\."]               # Database and OS metrics only
      category: ["os", "db"]              # OS and database categories
```
- **Result**: **ONLY** instances/metrics matching **ALL** include field patterns are processed
- **Example Match**: Instance "prod-db-1" with engine "postgres"
- **Example Reject**: Instance "prod-db-1" with engine "mysql" (engine doesn't match)

#### **3. Exclude Patterns Only = Denylist Mode**
```yaml
discovery:
  instances:
    exclude:
      identifier: [".*-test$", ".*-temp.*"]  # Exclude test and temp instances
  metrics:
    exclude:
      name: ["\\.idle$", "\\.wait$"]         # Exclude idle and wait metrics
```
- **Result**: Include everything **EXCEPT** instances/metrics matching exclude patterns

#### **4. Advanced Multi-Field Filtering**
```yaml
discovery:
  instances:
    include:
      identifier: ["^prod-", "^staging-"]
      engine: ["postgres", "aurora-postgresql"]
  metrics:
    include:
      name: ["^db\\.", "^os\\.cpu"]
      category: ["db", "os"]
    exclude:
      name: ["\\.idle$", "\\.wait$"]      # Exclude takes precedence - idle/wait metrics will be excluded
```

### Detailed Examples

#### **Instance Filtering Examples**
```yaml
instances:
  include:
    identifier:
      - "^prod-.*"                     # Starts with "prod-"
      - ".*-primary$"                  # Ends with "-primary"
      - "^(staging|prod|preprod)-.*"   # Multiple environment prefixes
    engine:
      - "postgres"                     # PostgreSQL instances
      - "aurora-postgresql"            # Aurora PostgreSQL instances
  exclude:
    identifier:
      - ".*-temp.*"                    # Contains "-temp"
      - ".*-backup$"                   # Ends with "-backup"
```

**Filter Logic**: An instance must match:
- `identifier` matches ANY of the include patterns **AND**
- `engine` matches ANY of the include patterns **AND**

#### **Tag-Based Instance Filtering**

The exporter supports filtering instances based on AWS resource tags. Tags are retrieved automatically during instance discovery and can be used in both include and exclude filters.

**Tag Filter Syntax**: Use `tag.<TagKey>` as the field name, where `<TagKey>` is the AWS tag key.

**Example Configuration**:
```yaml
instances:
  include:
    identifier: ["^prod-", "^staging-"]
    tag.Environment:
      - "^production$"
      - "^staging$"
    tag.Team:
      - "backend"
      - "data"
  exclude:
    tag.Status:
      - "deprecated"
      - "decommissioned"
    tag.Maintenance:
      - "true"
```

**Tag Filter Behavior**:
- **Include Logic**: ALL tag filters must match (AND logic across tag keys)
- **Exclude Logic**: ANY tag filter match will exclude (OR logic)
- **Exclude Precedence**: Exclude patterns take precedence over include patterns
- **Independent Evaluation**: Tag filters are evaluated independently from field filters (identifier, engine)
- **Combined Application**: An instance must pass BOTH field filters AND tag filters to be included
- **Untagged Instances**: Instances with no tags will fail all tag-based include filters

**Example Scenarios**:

1. **Include only production instances from specific teams**:
```yaml
instances:
  include:
    tag.Environment: ["^production$"]
    tag.Team: ["backend", "data", "platform"]
```

2. **Exclude instances under maintenance or deprecated**:
```yaml
instances:
  exclude:
    tag.Status: ["deprecated", "decommissioned"]
    tag.Maintenance: ["true", "scheduled"]
```

3. **Combined field and tag filtering**:
```yaml
instances:
  include:
    identifier: ["^prod-"]           # Field filter
    engine: ["postgres"]             # Field filter
    tag.Environment: ["production"]  # Tag filter
    tag.CostCenter: ["engineering"]  # Tag filter
  exclude:
    tag.Status: ["deprecated"]       # Exclude deprecated instances
```

In this example, an instance must:
- Have identifier starting with "prod-" AND
- Have engine containing "postgres" AND
- Have Environment tag matching "production" AND
- Have CostCenter tag matching "engineering" AND
- NOT have Status tag matching "deprecated"

#### **Metric Filtering Examples**
```yaml
metrics:
  statistic: "avg"                          # Default statistic
  include:
    name:
      - "^db\\."                            # All database metrics
      - "^os\\.cpu.*"                       # All CPU metrics
      - ".*\\.active_transactions$"         # Active transaction metrics
    category:
      - "os"                                # Operating system metrics
      - "db"                                # Database metrics
    unit:
      - "Count"                             # Count-based metrics
      - "Percent"                           # Percentage metrics
  exclude:
    name:
      - ".*\\.idle$"                        # Exclude idle metrics (exclude takes precedence)
```

### Custom Statistic Filtering
You can also filter metrics with specific statistics by including the statistic in the metric name:

```yaml
metrics:
  include:
    name:
      - "db\\.SQL\\.queries\\.max$"         # Only max statistic for SQL queries
      - "os\\.cpuUtilization\\.user\\.avg$" # Only avg statistic for CPU user time
      - ".*\\.sum$"                         # All sum statistics
```

## Metrics / Limits / Performance

### Supported Metrics
This exporter provides support for retrieving Operating System metrics as well as database-wide performance counters. For comprehensive metrics definitions, please refer to
[AWS public documentation](https://docs.aws.amazon.com/AmazonRDS/latest/AuroraUserGuide/USER_PerfInsights_Counters.html)

Naming pattern examples:
* `os.cpuUtilization.user` with `.avg` ==> `dbi_os_cpuutilization_user_avg`
* `db.Cache.Innodb_buffer_pool_read_requests` for Aurora-MySQL engine with `.avg` ==> `dbi_ams_db_cache_innodb_buffer_pool_read_requests_avg`

### Instance Limit & Sorting
The exporter has a **default limit of 25 instances** to ensure optimal performance. This limit can be configured using the `discovery.instances.max-instances` setting (accepted range: 1–10,000). Values above 10,000 are capped at 10,000 with a log warning. The instances are sorted by their creation time and only the oldest `max-instances` are monitored.

### Performance & Timing

When monitoring a large number of instances, proper configuration of concurrency, instance limits, and Prometheus scrape settings is critical for optimal performance.

#### Baseline Configuration (200 instances)
For monitoring up to 200 instances with a 15-second scrape interval:

```
discovery:
 instances:
   max-instances: 200
 processing:
   concurrency: 30
```

**Prometheus configuration:**

```
global:
  scrape_interval: 15s
  scrape_timeout: 15s
```

#### Handling API Throttling

If you encounter AWS API throttling errors: `ThrottlingException: Rate exceeded`, try redcuing the processing.concurrency and/or increasing the scrape interval.


#### Configuration Guidelines

| Instance Count | Recommended Concurrency | Scrape Interval | Scrape Timeout |
|---------------|------------------------|-----------------|----------------|
| 1-50          | 10-15                  | 15s             | 15s            |
| 51-200        | 20-30                  | 15s             | 15s            |
| 201-500       | 15-25                  | 30s             | 30s            |
| 500+          | 10-20                  | 60s             | 60s            |

## Usage Examples

### Collect All Instance Metrics
```bash
curl http://localhost:8081/metrics
```

### Filter Specific Instances
```bash
# Single RDS instance
curl http://localhost:8081/metrics?identifiers=my-db

# Multiple instances
curl http://localhost:8081/metrics?identifiers=my-db1,mydb-2,my-db3,mydb-4,my-db5
```

**Note**: Limit of 5 instance identifiers when using the instance specific metrics endpoint.

### Health Check

The `/health` endpoint returns `200 OK` immediately without making any AWS API calls, making it safe to use as a Kubernetes liveness or readiness probe:

```bash
curl http://localhost:8081/health
# {"status":"ok"}
```

### Kubernetes Deployment

Use `/health` for liveness and readiness probes to avoid triggering expensive PI API scrapes on every probe hit:

```yaml
livenessProbe:
  httpGet:
    path: /health
    port: 8081
  initialDelaySeconds: 5
  periodSeconds: 10
readinessProbe:
  httpGet:
    path: /health
    port: 8081
  initialDelaySeconds: 5
  periodSeconds: 10
```

### Integration with Prometheus

Add to your `prometheus.yml`:
```yaml
global:
  scrape_interval: 30s
  scrape_timeout: 30s

scrape_configs:
  - job_name: "db-insights-all"
    static_configs:
      - targets: ['localhost:8081']

  - job_name: "db-insights-production"
    static_configs:
      - targets: ['localhost:8081']
    params:
      identifiers: ['prod-db-1,prod-db-2']
```

## Cross-Account Access

By default the exporter uses the credential chain of the process that runs it (environment variables, EC2 instance profile, EKS IRSA, etc.) to call RDS and Performance Insights APIs. When your RDS instances live in a **different AWS account** from where the exporter runs, configure `discovery.role_arn` to have the exporter assume a role in the target account before making any API calls.

### How it works

```
Exporter process (source account)
  │
  ├─ AssumeRoleWithWebIdentity (IRSA / instance profile)
  │    → obtains source-account credentials
  │
  └─ sts:AssumeRole → target-account role
       → rds:DescribeDBInstances  (instance discovery)
       → pi:GetResourceMetrics    (metric collection)
       → pi:ListAvailableResourceMetrics
```

Credentials are cached in-process and refreshed automatically before expiry — no pod restarts needed.

### IAM setup

**Source account** — the exporter's identity (IRSA role / instance profile) needs:
```json
{
  "Effect": "Allow",
  "Action": "sts:AssumeRole",
  "Resource": "arn:aws:iam::<target-account-id>:role/<role-name>"
}
```

**Target account** — create a role with:

*Trust policy* (allows the source identity to assume it):
```json
{
  "Version": "2012-10-17",
  "Statement": [{
    "Effect": "Allow",
    "Principal": { "AWS": "arn:aws:iam::<source-account-id>:role/<exporter-role>" },
    "Action": "sts:AssumeRole"
  }]
}
```

*Permissions policy*:
```json
{
  "Version": "2012-10-17",
  "Statement": [{
    "Effect": "Allow",
    "Action": [
      "rds:DescribeDBInstances",
      "pi:ListAvailableResourceMetrics",
      "pi:GetResourceMetrics"
    ],
    "Resource": "*"
  }]
}
```

### Configuration

```yaml
discovery:
  regions:
    - "eu-central-1"
  role_arn: "arn:aws:iam::123456789012:role/pi-exporter-cross-account"
```

### Confused-deputy protection (ExternalID)

If multiple tenants can assume your target role, add an `ExternalID` condition to the trust policy and mirror it in the exporter config:

```json
"Condition": { "StringEquals": { "sts:ExternalId": "my-unique-id" } }
```

```yaml
discovery:
  role_arn: "arn:aws:iam::123456789012:role/pi-exporter-cross-account"
  role_arn_external_id: "my-unique-id"
```

### Error behaviour

Credentials are resolved lazily — the exporter starts successfully even if the role cannot be assumed. The error surfaces on the first `/metrics` scrape and is logged per-scrape until the IAM configuration is fixed. No `dbi_*` metrics are emitted while the role assumption fails.

## Building & Development

### Build Commands
```bash
# Build executable
make build

# Build and run executable
make run

# Development tools
make format         # Format code
make lint           # Run linter
make test           # Run tests
make coverage-html  # Generate test coverage report
```

### Running Locally
```bash
# Direct execution (uses config.yml in current directory)
./dbinsights-exporter

# With custom config file
./dbinsights-exporter -config /path/to/my-config.yml

# With debug logging
./dbinsights-exporter -debug

# Combine flags
./dbinsights-exporter -config /etc/dbinsights/config.yml -debug

# Check metrics endpoint
curl http://localhost:8081/metrics

# Health check
curl http://localhost:8081/health
```

## Prometheus Server Setup

### Installation
For installation instructions, see the [official Prometheus installation guide](https://prometheus.io/docs/introduction/first_steps/#downloading-prometheus).


### Configuration
Edit your Prometheus configuration file:
- **macOS (Homebrew)**: `/opt/homebrew/etc/prometheus.yml`
- **Linux**: `/etc/prometheus/prometheus.yml` or `./prometheus.yml`

```yaml
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: "database-insights-exporter"
    static_configs:
      - targets: ['localhost:8081']
```
For detailed configuration, see the [official Prometheus guide](https://prometheus.io/docs/introduction/first_steps/#configuring-prometheus).

### Start Prometheus
```bash
# macOS (Homebrew service)
brew services start prometheus

# macOS (direct execution)
/opt/homebrew/opt/prometheus/bin/prometheus \
  --config.file=/opt/homebrew/etc/prometheus.yml

# Linux (systemd service)
sudo systemctl start prometheus

# Linux (direct execution)
prometheus --config.file=/etc/prometheus/prometheus.yml
```

Access Prometheus UI at `http://localhost:9090`

## Security

See [CONTRIBUTING](CONTRIBUTING.md#security-issue-notifications) for more information.

## License

This project is licensed under the Apache-2.0 License.
