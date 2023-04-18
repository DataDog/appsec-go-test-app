# build-env allows to further define the build environment to use. Expected
# values are vendoring or musl:
#  - vendoring: compiles with vendoring enabled
#  - musl: compiles with musl-gcc
ARG buildenv="base"

FROM golang:1.20 AS base-build-env

RUN apt update && apt install -y jq

WORKDIR /app
COPY . .

ARG branch=""
RUN set -eux && \
    if [ "$branch" != "" ]; then \
      COMMIT=$(curl --fail -s "https://api.github.com/repos/DataDog/dd-trace-go/commits?sha=$branch" | jq -r .[0].sha); \
      go get -v -d gopkg.in/DataDog/dd-trace-go.v1@$COMMIT .; \
    fi

FROM base-build-env AS musl-build-env
RUN apt update && apt install -y musl-tools
ENV CC=musl-gcc

FROM base-build-env AS vendoring-build-env
RUN go mod vendor

FROM $buildenv-build-env AS build
RUN go build -v -tags appsec .

FROM debian:11-slim AS debian
COPY --from=build /app/go-dvwa /usr/local/bin
ENV DD_APPSEC_ENABLED=1
ENV DD_TRACE_DEBUG=true
CMD /usr/local/bin/go-dvwa

FROM musl-build-env AS build-alpine
RUN go build -v -tags appsec .

FROM alpine AS alpine
COPY --from=build-alpine /app/go-dvwa /usr/local/bin
RUN apk update && apk add libc6-compat ca-certificates
ENV DD_APPSEC_ENABLED=1
ENV DD_TRACE_DEBUG=true
CMD /usr/local/bin/go-dvwa
