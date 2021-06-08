#!/usr/bin/env bash
set -e
/bin/alertmanager --config.file=/etc/alertmanager/config.yml --storage.path=/alertmanager
