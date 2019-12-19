ARG ENVOY

FROM ${ENVOY}

ENV HOST_LIST="[{socket_address: {address: accesspoint, port_value: 2917}}]"

COPY ./accesspoint/proxy/envoy_template.yaml ./accesspoint/proxy/cert ./accesspoint/proxy/envoy.sh \
  /etc/envoy/
RUN chmod +x /etc/envoy/envoy.sh

ENTRYPOINT ["/etc/envoy/envoy.sh"]