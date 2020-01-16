ARG NODE_OS_NAME
ARG NODE_OS_TAG
ARG BUILDER

FROM ${BUILDER} as builder

FROM ${NODE_OS_NAME}:${NODE_OS_TAG}
ARG SERVICE
COPY --from=builder /go/bin/${SERVICE} .
RUN ln -s /${SERVICE} /entrypoint
ENTRYPOINT ["/entrypoint"]