FROM golang:1 AS build-env

RUN apt update && apt install -y jq

WORKDIR /app
COPY . .

ARG branch=""
RUN set -eux && \
    if [ "$branch" != "" ]; then \
      COMMIT=$(curl --fail -s "https://api.github.com/repos/DataDog/dd-trace-go/commits?sha=$branch" | jq -r .[0].sha); \
      go get -v -d gopkg.in/DataDog/dd-trace-go.v1@$COMMIT .; \
    fi


FROM build-env AS build
RUN go build -v -tags appsec .


FROM build-env AS build-vendoring
RUN go mod vendor
RUN go build -v -tags appsec .


FROM debian:11-slim AS debian
COPY --from=build /app/go-dvwa /usr/local/bin
ENV DD_APPSEC_ENABLED=1
ENV DD_TRACE_SAMPLE_RATE=0.5
ENV DD_TRACE_DEBUG=true
CMD /usr/local/bin/go-dvwa


FROM alpine AS alpine
COPY --from=build /app/go-dvwa /usr/local/bin
RUN apk update && apk add libc6-compat ca-certificates
ENV DD_APPSEC_ENABLED=1
ENV DD_TRACE_SAMPLE_RATE=0.5
ENV DD_TRACE_DEBUG=true
CMD /usr/local/bin/go-dvwa
