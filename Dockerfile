ARG buildenv="base"
ARG golang="1.21"

FROM golang:$golang AS base-build-env

RUN apt update && apt install -y jq

WORKDIR /app
COPY . .

ARG tracer=""

# If the first go get fails, we wait for github to register the commit (or stop rate limiting us)
RUN set -eux && \
    if [ "$tracer" != "" ]; then \
      if !go get -v -u gopkg.in/DataDog/dd-trace-go.v1@$tracer; then \
        COMMIT=""; \
        while [ -z "$COMMIT" ]; do \
          COMMIT=$(curl --fail-with-body -s "https://api.github.com/repos/DataDog/dd-trace-go/commits?sha=$tracer" | jq -r .[0].sha); \
          sleep 1; \
        done; \
        go get -v -u gopkg.in/DataDog/dd-trace-go.v1@$tracer; \
      fi; \
      go mod tidy; \
    fi

# We must enforce CGO disabled in order to:
# 1. Make sure dd-trace-go doesn't rely on it indeed.
# 2. We avoid Go's CGO implementations which dynamically link the compiled
#    program to the system libraries of the build environment, which are not
#    necessarily compatible nor the same than the target environment (debian or
#    alpine in this Dockerfile).
# Said otherwise, not disabling CGO prevents copying the compiled program
# into a different distribution. So we make sure to test the onboarding
# experience we want to provide.
ENV CGO_ENABLED=0

# vendoring alternative
FROM base-build-env AS vendoring-build-env
RUN go mod vendor

# $buildenv defaults to base and allows to be changed into vendoring to test
# this alternative
FROM $buildenv-build-env AS build
RUN go build -v -tags appsec .

# debian target
FROM debian:11-slim AS debian
COPY --from=build /app/go-dvwa /usr/local/bin
ENV DD_APPSEC_ENABLED=1 DD_TRACE_DEBUG=true DD_APPSEC_WAF_TIMEOUT=1h
CMD ["/usr/local/bin/go-dvwa", ":7777"]

# alpine target
FROM alpine AS alpine
COPY --from=build /app/go-dvwa /usr/local/bin
RUN apk update && apk add libc6-compat
ENV DD_APPSEC_ENABLED=1 DD_TRACE_DEBUG=true DD_APPSEC_WAF_TIMEOUT=1h
CMD ["/usr/local/bin/go-dvwa", ":7777"]
