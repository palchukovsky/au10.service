ARG TAG

FROM envoyproxy/envoy:${TAG}

RUN apt-get update && apt-get install -y gettext