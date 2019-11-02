#!/bin/sh
set -e

echo "Generating envoy.yaml config file..."
echo ACCESSPOINT_HOST: "${ACCESSPOINT_HOST}"
echo ACCESSPOINT_PORT: "${ACCESSPOINT_PORT}"
cat /etc/envoy/envoy_template.yaml | envsubst \$ACCESSPOINT_HOST,\$ACCESSPOINT_PORT > /etc/envoy/envoy.yaml

echo "Starting Envoy..."
/usr/local/bin/envoy -c /etc/envoy/envoy.yaml