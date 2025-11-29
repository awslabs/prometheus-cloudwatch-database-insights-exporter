# Amazon CloudWatch Database Insights Exporter for Prometheus

A Prometheus exporter that provides Aurora/RDS Performance Insights metrics with auto-discovery capabilities.

As of Q4 2025, the exporter remains under active development. Current limitations include support for a single region and a maximum of 25 instances to maintain a 15-second scrape interval. These constraints are scheduled to be addressed in forthcoming releases.

## Features

- **Auto-discovery**: Automatically discovers Aurora/RDS instances in specified AWS regions
- **Instance filtering**: Query metrics for specific instances using URL parameters
- **Prometheus-compatible**: Standard `/metrics` endpoint with Prometheus format
- **Low-latency collection**: Efficient metric collection from Amazon RDS Performance Insights API
- **Simple configuration**: YAML-based configuration with sensible defaults

## Prerequisites

- **Go 1.23 or later** (for building from source)
- **AWS credentials configured**
- **Required AWS permissions**:
  - `rds:DescribeDBInstances`
  - `pi:ListAvailableResourceMetrics`
  - `pi:GetResourceMetrics`

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

## Configuration

The DB Insights Exporter has a simple configuration mechanism using a YAML configuration file. Currently, no command-line flags are supported - all configuration is done through the `config.yml` file.

The configuration file must be named `config.yml` and placed in the same directory as the executable.

## YAML Configuration File

The file is written in YAML format:

```yaml
# Example complete configuration
discovery:
  regions:
    - "us-west-2"
  instances:
    max-instances: 25
    ttl: "5m"
    include:
      identifier: ["^(prod|staging)-"]
      engine: ["^postgres", "aurora-postgresql"]
    exclude:
      identifier: ["-temp-", "-test$"]
  metrics:
    statistic: "avg"
    metadata-ttl: "1h"
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
| `instances.max-instances` | integer | Optional | `25` | Maximum number of instances to monitor. When this limit is exceeded, only the oldest `max-instances` are selected |
| `instances.ttl` | string | Optional | `"5m"` | Time-to-live for cached instance discovery results |
| `instances.include` | map | Optional | `{}` | Map of field names to regex pattern arrays for instance filtering (allowlist mode). Supported fields: `identifier`, `engine` |
| `instances.exclude` | map | Optional | `{}` | Map of field names to regex pattern arrays for instance filtering (denylist mode). Supported fields: `identifier`, `engine` |
| `metrics.statistic` | string | Required | `"avg"` | Default statistic aggregation for Performance Insights metrics |
| `metrics.metadata-ttl` | string | Optional | `"60m"` | Time-to-live for cached metric definitions |
| `metrics.include` | map | Optional | `{}` | Map of field names to regex pattern arrays for metric filtering (allowlist mode). Supported fields: `name`, `category`, `unit` |
| `metrics.exclude` | map | Optional | `{}` | Map of field names to regex pattern arrays for metric filtering (denylist mode). Supported fields: `name`, `category`, `unit` |
| `processing.concurrency` | integer | Optional | `4` | Number of concurrent goroutines for metric collection |

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

### Minimal Configuration Example

```yaml
discovery:
  regions:
    - "us-east-1"
```

This minimal configuration will:
- Monitor RDS/Aurora instances in `us-east-1`
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
The exporter has a **default limit of 25 instances** to ensure optimal performance. This limit can be configured using the `discovery.instances.max-instances` setting. The instances are sorted by their creation time and only the oldest `max-instances` are monitored.

Note: Removal of this constraint is currently under development and will be included in a subsequent release.

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
# Direct execution
./dbinsights-exporter

# Check metrics endpoint
curl http://localhost:8081/metrics

# Health check
curl -I http://localhost:8081/metrics
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
