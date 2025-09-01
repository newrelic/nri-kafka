FROM golang:1.24.6 as builder
ARG CGO_ENABLED=0
ARG NRJMX_VERSION
ARG DOWNLOAD_BASE_URL=http://nr-downloads-ohai-staging.s3-website-us-east-1.amazonaws.com
WORKDIR /go/src/github.com/newrelic/nri-kafka
COPY . .
RUN make clean compile
WORKDIR /
ADD ${DOWNLOAD_BASE_URL}/infrastructure_agent/binaries/linux/noarch/nrjmx_linux_${NRJMX_VERSION}_noarch.tar.gz ./
RUN tar -xzf nrjmx_linux_${NRJMX_VERSION}_noarch.tar.gz

FROM alpine:latest
RUN apk --update add openjdk8-jre
COPY --from=builder /usr/bin/nrjmx /usr/bin/nrjmx
COPY --from=builder /usr/lib/nrjmx/nrjmx.jar /usr/lib/nrjmx/nrjmx.jar
COPY --from=builder /go/src/github.com/newrelic/nri-kafka/bin /
CMD ["sleep", "1h"]