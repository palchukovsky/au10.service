#!/bin/sh
set -e

echo "Generating envoy.yaml config file..."
echo HOST_LIST: "${HOST_LIST}"
cat /etc/envoy/envoy_template.yaml | envsubst \$HOST_LIST > /etc/envoy/envoy.yaml

echo "Starting Envoy..."
/usr/local/bin/envoy -c /etc/envoy/envoy.yaml