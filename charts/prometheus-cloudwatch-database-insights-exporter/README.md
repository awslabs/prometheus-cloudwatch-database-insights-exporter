# prometheus-cloudwatch-database-insights-exporter

Helm chart for the [Amazon CloudWatch Database Insights Exporter](https://github.com/awslabs/prometheus-cloudwatch-database-insights-exporter).

## Prerequisites

- Kubernetes 1.19+
- An image. The project does not publish one yet, so build the `Dockerfile` at the repository root
  and push it to a registry your cluster can pull from.
- AWS credentials reachable by the pod, with `rds:DescribeDBInstances`,
  `pi:ListAvailableResourceMetrics` and `pi:GetResourceMetrics`.

## Installing

```bash
helm install db-insights ./charts/prometheus-cloudwatch-database-insights-exporter \
  --set image.repository=<your-registry>/prometheus-cloudwatch-database-insights-exporter \
  --set image.tag=<your-tag> \
  --set config.discovery.regions[0]=us-west-2
```

`image.repository` has no default. Leaving it unset fails the render with a message saying why,
rather than installing something that cannot pull.

## AWS credentials

The exporter uses the default AWS credential chain, so on EKS the usual route is IRSA or Pod
Identity, both of which are annotations on the service account:

```yaml
serviceAccount:
  annotations:
    eks.amazonaws.com/role-arn: arn:aws:iam::111122223333:role/db-insights-exporter
```

Without an annotation the exporter falls back to whatever credentials the node provides, which is
usually the node role and usually not what you want.

## Configuration

Everything under `config` is rendered verbatim into a ConfigMap and mounted at
`/etc/dbinsights/config.yml`. The fields are documented in the project README. The deployment
carries a checksum of the rendered config, so changing it rolls the pods.

```yaml
config:
  discovery:
    regions:
      - us-west-2
    instances:
      max-instances: 25
      include:
        identifier:
          - "^prod-"
    metrics:
      statistic: "avg"
  export:
    port: 8081
```

The container port, the service port and the probes all follow `config.export.port`, so changing it
in one place is enough.

## Prometheus

With the Prometheus Operator:

```yaml
serviceMonitor:
  enabled: true
  interval: 30s
  scrapeTimeout: 30s
```

`scrapeTimeout` defaults to 30s rather than the usual 10s because a scrape calls the AWS API, and
how long that takes depends on how many instances are discovered.

Without the operator, scrape the service directly — the exporter also accepts an `identifiers`
query parameter to narrow a scrape to specific instances.

## Values

| Key | Default | Description |
|-----|---------|-------------|
| `image.repository` | `""` | **Required.** Image to run |
| `image.tag` | `""` | Defaults to the chart `appVersion` |
| `image.pullPolicy` | `IfNotPresent` | |
| `replicaCount` | `1` | Replicas each poll the API independently, so more than one multiplies API calls |
| `config` | see `values.yaml` | Rendered into the ConfigMap |
| `debug` | `false` | Passes `-debug`, which logs cache decisions and every API request |
| `extraArgs` | `[]` | Appended to the container args |
| `serviceAccount.create` | `true` | |
| `serviceAccount.annotations` | `{}` | Where IRSA / Pod Identity is wired |
| `service.type` | `ClusterIP` | |
| `service.port` | `null` | Defaults to `config.export.port` |
| `serviceMonitor.enabled` | `false` | Requires the Prometheus Operator CRDs |
| `serviceMonitor.interval` | `30s` | |
| `serviceMonitor.scrapeTimeout` | `30s` | |
| `resources` | `{}` | |
| `podSecurityContext` | `runAsNonRoot`, `RuntimeDefault` | |
| `securityContext` | read-only root, all capabilities dropped | |
| `livenessProbe` / `readinessProbe` | `/health` | `/health` answers without calling AWS |
| `nodeSelector`, `tolerations`, `affinity`, `topologySpreadConstraints` | `{}` / `[]` | |
| `extraVolumes`, `extraVolumeMounts` | `[]` | |
| `extraEnv`, `extraEnvFrom` | `[]` | |
