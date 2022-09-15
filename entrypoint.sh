#!/bin/sh
set -e

if [ ! -f "/etc/oms/config.yaml" ];then
    cp /opt/oms/config.yaml.example /etc/oms/config.yaml
fi

exec "$@"