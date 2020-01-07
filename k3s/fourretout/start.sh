#!/bin/bash
set -Eeuo pipefail
cd vagrant
# check if it is running
VAG_STATUS=$(vagrant status 2>&1)
NOT_RUNNING=$(vagrant status | awk 'NR>2 && ($2 != "running") && $NF == "(virtualbox)" {print $1}')

if [ -z "${NOT_RUNNING}" ] && [ $(echo ${VAG_STATUS} | grep "target machine is required to run" | wc -l) -eq 0 ]; then
    echo "VM are running relaunching provision"
    vagrant provision
else
    MACHINE=$(echo ${NOT_RUNNING} |
        awk 'BEGIN{ORS=","} {print $0}')
    MACHINE="${MACHINE:0:-1} are not running. relaunching vagrant"
    echo ${MACHINE}
    vagrant up
fi
cd ..
