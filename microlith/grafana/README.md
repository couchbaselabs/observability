# Grafana Dashboard Development Guide #

Creating your own dashboards (or modifying existing ones) is very easy in Grafana. This guide will outline the steps we recommend for developing for CMOS.

## Introduction ##

Dashboards are currently stored under /microlith/grafana/provisioning/dashboards. They are in `.JSON` format, which specifies all of the information used to create them: datasources, variables, panels, and some default view settings.

The `dashboard.yml` in this folder is used to specify various defaults. You may find it useful to change the value of `updateIntervalSeconds` from `3600` to `10` or similar - this will allow you to change the JSON file and have these changes immediately reflected in the Grafana browser (upon refreshing the webpage).

The `datasources.yml` file, under /microlith/grafana/provisioning/datasources, specifies the types of datasources that should be available to dashboards. For example, Prometheus or Loki. Most dashboards use a combination of Prometheus and "JSON API", a community plugin that allows querying of REST APIs from within Grafana.

Note: for Couchbase Server 6 and below only, the exporter must be installed on each node to import statistics into the Prometheus instance.

## Example Workflow ##

When developing dashboards, especially for those with SSH access to the private cbmultimanager repository, it may be helpful to call `make example-multi`. This will build and run the CMOS container as well as a number of Couchbase Server nodes, divided into the specified amount of clusters, with sample buckets and indexes loaded, and a gentle cbpillowfight activated. More information on how to use this and the various variables that can be changed can be found in the README under `examples/containers/multi`.

Once you have a local environment configured, you can visit http://localhost:8080/grafana, and begin editing the dashboards. It is important to note that they will reset every time the page is reloaded, so make sure to hit "Save" every time you make a change (and before exporting the dashboard).

## Exporting dashboards from the web UI for provisioning ##

1. Click the "share" icon at the top-left of the webpage.
2. Click export.
3. Make sure "Export for sharing externally" is NOT ticked, and then click show JSON -> Copy JSON to clipboard.
4. Paste this JSON into the appropriate .JSON file (under .../dashboards), and save it.
5. Ensure the dashboard's `time` field is:
    ```"time": {
        "from": "now-3h",
        "to": "now"
    ```

    If this is set to some specific date and time and not `now-3h`/`now`, the dashboard will break on re-provisioning as the data does not exist.
