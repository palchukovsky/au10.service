FROM envoyproxy/envoy:v1.12.1

ENV ACCESSPOINT_HOST=accesspoint
ENV ACCESSPOINT_PORT=90107

COPY ./accesspoint/proxy/envoy.yaml /etc/envoy/envoy_template.yaml
COPY ./accesspoint/proxy/cert/* /etc/envoy/cert/

COPY ./accesspoint/proxy/envoy.sh /
RUN chmod 500 /docker-entrypoint.sh

RUN apt-get update && \
    apt-get install gettext -y

ENTRYPOINT ["/envoy.sh"]