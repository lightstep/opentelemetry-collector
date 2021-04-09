# Prometheus Discovery Receiver
Receives meta labels based on the [Prometheus](https://prometheus.io/) Service Discovery.

Supported pipeline types: metrics (and labels?)

## ⚠️ Warning

Note: This component is currently work in progress. It has several limitations
and please don't use it if the following limitations is a concern:

* Collector cannot auto-scale the scraping yet when multiple replicas of the
  collector is run. 
* When running multiple replicas of the collector with the same config, it will
  scrape the targets multiple times.
* Users need to configure each replica with different scraping configuration
  if they want to manually shard the scraping.

## Getting Started

This receiver uses the Prometheus' Service Discovery, to find out targets and fetch
their metadata. Using this, a `present` metric is generated for every target discovered
attached with their metadata as labels (e.g., a k8s service discovery can attach the pod name
or deployment name labels). It supports the full set of Prometheus configuration for discovering
services. Just like you would write in a YAML configuration file
before starting Prometheus, such as with:

**Note**: Since the collector configuration supports env variable substitution
`$` characters in your prometheus configuration are interpreted as environment
variables.  If you want to use $ characters in your prometheus configuration,
you must escape them using `$$`.

```shell
prometheus --config.file=prom.yaml
```

You can copy and paste that same configuration under:

```yaml
receivers:
  prometheus_discovery:
    config:
```

For example:

```yaml
receivers:
  prometheus_discovery:
      config:
        scrape_configs:
          - job_name: 'otel-collector'
            scrape_interval: 5s
            static_configs:
              - targets: ['0.0.0.0:8888']
          - job_name: k8s
            kubernetes_sd_configs:
            - role: pod
            relabel_configs:
            - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_scrape]
              regex: "true"
              action: keep
            metric_relabel_configs:
            - source_labels: [__name__]
              regex: "(request_duration_seconds.*|response_duration_seconds.*)"
              action: keep
```
