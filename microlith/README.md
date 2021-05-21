A prototype for deploying a single image to rule them all for observability.

To build and run:
```
docker build -t couchbase-observability .
docker run --name=couchbase-grafana --rm -d -P -v /proc:/host/proc:ro -v /sys:/host/sys:ro -v /:/host/rootfs:ro couchbase-observability
```
Use `docker ps` or `docker inspect X` to see the local ports exposed, the mapping to `3000` is the Grafana one so to get this:
```
docker container port couchbase-grafana 3000
0.0.0.0:55124
:::55124
```
Browse to `localhost:55124` and log in with the default creds of `admin:password` for Grafana.

You can disable each of the tools using a `-e DISABLE_XXX=` to set an environment variable named `DISABLE_<tool>` for each.
