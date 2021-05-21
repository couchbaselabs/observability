#!/bin/bash
set -eu
TEXTFILE_COLLECTOR_DIR=${TEXTFILE_COLLECTOR_DIR:-/custom-logs}
REPORTING_PERIOD_SECS=${REPORTING_PERIOD_SECS:-30}
declare -a PORTS_TO_CHECK=(8091 8093 9100 12345)

checkPorts() {
    HOST=$1
    for PORT in "${PORTS_TO_CHECK[@]}"; do
        echo "Checking ${HOST}:${PORT}"
        IS_UP=0
        if nmap "${HOST}" -p "${PORT}" --open 2>/dev/null | grep -q "Host is up"; then
            IS_UP=1
        fi
        echo "cbip_${HOST}_port_${PORT}_open ${IS_UP}" | tr ./ _ >> "${TEXTFILE_COLLECTOR_DIR}/couchbase_portcheck.prom.$$"
    done
}

apt-get update
apt install -y nmap

while true; do
    rm -f "${TEXTFILE_COLLECTOR_DIR}/couchbase_portcheck.prom"

    # LOCAL_IP=$(hostname -I)
    # for IP in $(seq 1 15); do
    #     HOST=${LOCAL_IP%.*}.${IP}
    #     checkPorts "$HOST"
    # done
    for HOST in db1 db2 db3; do
        checkPorts "$HOST"
    done
    # Prevent partial metrics appearing just as we scrape
    mv -f "${TEXTFILE_COLLECTOR_DIR}/couchbase_portcheck.prom.$$" "${TEXTFILE_COLLECTOR_DIR}/couchbase_portcheck.prom"
    cat "${TEXTFILE_COLLECTOR_DIR}/couchbase_portcheck.prom"
    sleep "${REPORTING_PERIOD_SECS}"
done