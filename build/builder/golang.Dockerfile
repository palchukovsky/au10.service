ARG GOLANG_TAG

FROM golang:${GOLANG_TAG}

ARG GOLANGCI_VER

ENV GO111MODULE=on

WORKDIR /go/src/bitbucket.org/au10/service

COPY Makefile go.mod go.sum ./

RUN apk update && apk add build-base curl
RUN curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v${GOLANGCI_VER}
RUN make install-mock-deps && \
  go mod download