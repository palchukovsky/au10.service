ARG NODE_OS_NAME
ARG NODE_OS_TAG
ARG BUILDER

FROM ${BUILDER} as builder

FROM ${NODE_OS_NAME}:${NODE_OS_TAG}
WORKDIR /opt/au10/service
COPY --from=builder /go/bin/accesspoint .
ENTRYPOINT ["./accesspoint"]