FROM debian:stable-slim as builder
ENV GOPATH $HOME/go
ENV PATH $HOME/go/bin:$PATH
RUN apt-get update && apt-get install -y \
  make \
  golang-go \
  git
WORKDIR /result
COPY Makefile .
RUN make install-protoc

FROM debian:stable-slim
ENV GOPATH $HOME/go
RUN apt-get update && apt-get install -y \
  make
COPY --from=builder /go/bin/protoc-gen-go /usr/bin/
COPY --from=builder /result/build/bin/* /compiler/build/bin/