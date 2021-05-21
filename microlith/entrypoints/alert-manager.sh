#!/usr/bin/env bash
set -e
/bin/alertmanager --config.file=/etc/alertmanager/alertmanager.yml --storage.path=/alertmanager
