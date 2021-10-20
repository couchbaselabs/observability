#!/usr/bin/env bash

# Copyright 2021 Couchbase, Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file  except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the  License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Starts a Couchbase cluster in Vagrant, featuring couchbase-exporter and couchbase-fluent-bit configured.
# Uses the variables VAGRANT_HOST, VAGRANT_CPUS, and VAGRANT_RAM to determine the cluster configuration.

# Parameters:
# $1: The Couchbase Server version to test (default: 6.6.3)
# $2: The OS to use (default: centos7)
# $3: The location of the CB vagrants folder
function start_vagrant_cluster() {

    local SCRIPT_DIR
    SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"

    local CB_VERSION=$1
    local VAGRANT_OS=$2
    local VAGRANT_LOCATION=$3

    if pushd "$VAGRANT_LOCATION/vagrants"; then
        git pull
        popd || exit 1
    else
        git clone https://github.com/couchbaselabs/vagrants.git "$VAGRANT_LOCATION/vagrants"
    fi

    temporary_dir=$(mktemp -d)

    if pushd "$VAGRANT_LOCATION/vagrants/$CB_VERSION/$VAGRANT_OS"; then
        vagrant up --parallel
        vagrant ssh-config | tail -n +$(( VAGRANT_NODES + 1)) > "$temporary_dir/ssh.config"
        popd || exit 1
    fi

    cat <<EOF > "$SCRIPT_DIR/hosts.ini"
[couchbase:vars]
ansible_ssh_common_args = -F $temporary_dir/ssh.config
ansible_become_password = vagrant
[couchbase]
$(seq -f "node%g" 1 "$VAGRANT_NODES")
EOF

    ansible-playbook -i "$SCRIPT_DIR/hosts.ini" "$SCRIPT_DIR/playbook.yml"
    rm "$SCRIPT_DIR"/hosts.ini
}
