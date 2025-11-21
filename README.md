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
  metrics:
    statistic: "avg"

export:
  port: 8081
```

### Configuration Reference

#### `discovery` section
Controls how the exporter discovers and monitors RDS/Aurora instances.

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `regions` | array | `["us-west-2"]` | List of AWS regions to scan for RDS/Aurora instances. **Note**: Only the first region is currently used (single-region support only) |
| `instances.max-instances` | integer | `25` | Maximum number of instances to monitor. When this limit is exceeded, only the oldest `max-instances` are selected |
| `metrics.statistic` | string | `"avg"` | Default statistic aggregation for Performance Insights metrics |

**Valid statistic values:**
- `"avg"` - Average values
- `"min"` - Minimum values
- `"max"` - Maximum values
- `"sum"` - Sum of values

#### `export` section
Controls how metrics are exposed via HTTP endpoint.

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `port` | integer | `8081` | HTTP port number for the Prometheus metrics endpoint |

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

### Full Configuration Example

```yaml
discovery:
  regions:
    - "us-west-1"
  instances:
    max-instances: 15
  metrics:
    statistic: "max"

export:
  port: 8081
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
Average discovery and processing times based on instance count:

| Instance Count | Discovery Time | Processing Time | Total Time |
|---------------|----------------|-----------------|------------|
| 10 instances | ~3 seconds | ~ 3 seconds | ~6 seconds |
| 25 instances  | ~10 seconds | ~5 seconds | ~15 seconds |

**Note**: Times may vary based on AWS API response times and network latency.

**Important**: When monitoring large numbers of instances, you **may need to increase** your Prometheus `scrape_interval` and `scrape_timeout` values to accommodate the longer collection times.

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
go build -o dbinsights-exporter ./cmd

# Development tools
make format          # Format code
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
