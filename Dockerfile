FROM debian:stable-slim as debian
RUN mkdir -p /app
COPY . /app
ENV DD_APPSEC_RULES=${DD_APPSEC_RULES}
ENV DD_APPSEC_ENABLED=1
ENV DD_SAMPLE_RATE=0.5
ENV DD_TRACE_DEBUG=true
CMD /app/dvwa
