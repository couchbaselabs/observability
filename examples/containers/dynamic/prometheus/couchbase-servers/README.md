Update the `targets.json` file with any additional targets to scrape dynamically, it's re-read every 30 seconds currently.

See: https://prometheus.io/docs/guides/file-sd/

As an example to get logging metrics requires a separate path:
```
[
    {
      "targets": [
        "logging:2020"
      ],
      "labels": {
        "job": "couchbase",
        "container": "logging",
        "__metrics_path__": "/api/v1/metrics/prometheus"
      }
    },
    {
      "targets": [
        "exporter:9091"
      ],
      "labels": {
        "job": "couchbase",
        "container": "monitoring"
      }
    }
]
```
