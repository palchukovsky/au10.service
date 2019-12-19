ARG GOLANG_TAG

FROM golang:${GOLANG_TAG}

ENV GO111MODULE=on

WORKDIR /go/src/bitbucket.org/au10/service

COPY Makefile go.mod go.sum ./

RUN apk update && apk add build-base && \
  make install-mock-deps && \
  go mod download