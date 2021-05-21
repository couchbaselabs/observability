
Spin up with docker-compose:
`docker-compose up`

You can log into Grafana on `https://localhost:4000` as `admin:password` and data sources for Loki and Prometheus should be set up as `http://loki:3100` and `http://prometheus:9090`.
This should then allow us to view logs with labels and metrics.

We need to set up a cluster then.
Add the two servers for db2 and db3, you can get the IP address for each like so: 

```
docker inspect -f '{{range.NetworkSettings.Networks}}{{.IPAddress}}{{end}}' db2
docker inspect -f '{{range.NetworkSettings.Networks}}{{.IPAddress}}{{end}}' db3
```

Textfile collector scripts: https://github.com/prometheus-community/node-exporter-textfile-collector-scripts

You may need to configure `/` as a shareable directory for the docker runtime.
We can also scrape metrics from the container engine for docker desktop on port 9323:
```
{
  "experimental": true,
  "metrics-addr": "127.0.0.1:9323"
}
```
The stack should run without this though, it just may not get all the metrics for all the dashboards.