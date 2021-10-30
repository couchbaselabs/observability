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
# Uses the variables VAGRANT_HOST, VAGRANT_CPUS, and VAGRANT_RAM to determine
# the cluster configuration, and CMOS_HOST as the Fluent Bit Loki destination.
# Parameters:
# $1: The Couchbase Server version to test (default: 6.6.3)
# $2: The OS to use (default: centos7)
function start_vagrant_cluster() {
    local cb_version=${1:-6.6.3}
    local os=${2:-centos7}
    # Check if requirements are installed
    if ! command -v vagrant &> /dev/null
    then
        echo "Vagrant not installed!"
        exit 1
    fi
    if ! command -v ansible &> /dev/null
    then
        echo "Ansible not installed!"
        exit 1
    fi

    if pushd "$RESOURCES_ROOT/native/vagrants"; then
        git pull
        popd || exit 1
    else
        git clone --depth=1 https://github.com/couchbaselabs/vagrants.git "$RESOURCES_ROOT/native/vagrants"
    fi

    ansible-galaxy install -r "$RESOURCES_ROOT/native/requirements.yml"

    temporary_dir=$(mktemp -d)

    export VAGRANT_NODES=${VAGRANT_NODES:-3}
    export VAGRANT_CPUS=${VAGRANT_CPUS:-2}
    export VAGRANT_RAM=${VAGRANT_RAM:-1024}
    pushd "$RESOURCES_ROOT/native/vagrants/$cb_version/$os" || exit 1
        vagrant up --parallel

        # On every Vagrant command, the Vagrantfile prints a header like `Private network (host only) : http://node1-cb663-centos7.vagrants:8091/` for each node.
        COUCHBASE_SERVER_HOSTS=$(vagrant status | head -n "$VAGRANT_NODES" | sed -e 's/^.*http:\/\///; s/\/$//')
        export COUCHBASE_SERVER_HOSTS
        vagrant ssh-config | tail -n +$(( VAGRANT_NODES + 1)) > "$temporary_dir/ssh.config"
    popd || exit 1

    cat <<EOF > "$RESOURCES_ROOT/native/hosts.ini"
[couchbase:vars]
ansible_ssh_common_args = -F $temporary_dir/ssh.config
ansible_become_password = vagrant

[couchbase]
$(seq -f "node%g" 1 "$VAGRANT_NODES")
EOF

    ansible-playbook -i "$RESOURCES_ROOT/native/hosts.ini" "$RESOURCES_ROOT/native/playbook.yml"
}

# Parameters:
# $1: The Couchbase Server version to test (default: 6.6.3)
# $2: The OS to use (default: centos7)
function teardown_vagrant_cluster() {
    local cb_version=${1:-6.6.3}
    local os=${2:-centos7}
    pushd "$RESOURCES_ROOT/native/vagrants/$cb_version/$os" || return
        vagrant destroy --no-tty -f
    popd || return
}
