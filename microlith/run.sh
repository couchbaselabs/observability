#!/usr/bin/env bash
set -e
for i in /entrypoints/*; do
    if [[ -x "$i" ]]; then
        echo "Running: $i"
        if [[ "$i" == "/entrypoints/loki" ]]; then
            tini -- "$i" "-config.file=/etc/loki/local-config.yaml" &
        elif [[ "$i" == "/entrypoints/prometheus" ]]; then
            tini -- "$i" "--config.file=/etc/prometheus/prometheus.yml" \
             "--storage.tsdb.path=/prometheus" \
             "--web.console.libraries=/usr/share/prometheus/console_libraries" \
             "--web.console.templates=/usr/share/prometheus/consoles" &
        elif [[ "$i" == "/entrypoints/alertmanager" ]]; then
            tini -- "$i" "--config.file=/etc/alertmanager/alertmanager.yml" \
             "--storage.path=/alertmanager" &
        else
            tini -- "$i" "$@" &
        fi
    else
        echo "Skipping non-executable: $i"
    fi
done

wait -n