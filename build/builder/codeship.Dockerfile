FROM docker
ENV CODECOV_TOKEN=19d4b2af-704c-4bd6-af1f-56ca071b7986
WORKDIR /usr/src/app
COPY . .
# install build-base - for Makefile, install bash, curl and git - for codecov.io 
RUN apk update && apk add bash build-base curl git
RUN docker login registry.gitlab.com -u builder -p FeAhJTRCQ57GdFYt8D8k