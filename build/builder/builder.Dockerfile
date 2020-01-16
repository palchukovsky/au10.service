ARG PROTOC
ARG GOLANG

FROM ${PROTOC} as protoc
WORKDIR /compiler
COPY ./accesspoint/*.proto ./accesspoint/
COPY ./Makefile ./
RUN make stub

FROM ${GOLANG}
WORKDIR /go/src/bitbucket.org/au10/service
COPY . .
COPY --from=protoc /compiler/accesspoint/proto/ ./accesspoint/proto/
RUN export CGO_ENABLED=0 && \
  make mock && go test -timeout 15s -v -coverprofile=coverage.txt -covermode=atomic ./... && \
  cd ./cli && go build -o /go/bin/ && cd - && \
  cd ./accesspoint && go build -o /go/bin/ && cd -