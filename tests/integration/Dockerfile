FROM golang:1.22.5 as builder
ARG CGO_ENABLED=0
ARG NRJMX_VERSION
WORKDIR /go/src/github.com/newrelic/nri-kafka
COPY . .
RUN make clean compile
WORKDIR /
ADD http://download.newrelic.com/infrastructure_agent/binaries/linux/noarch/nrjmx_linux_${NRJMX_VERSION}_noarch.tar.gz ./
RUN tar -xzf nrjmx_linux_${NRJMX_VERSION}_noarch.tar.gz

FROM alpine:latest
RUN apk --update add openjdk8-jre
COPY --from=builder /usr/bin/nrjmx /usr/bin/nrjmx
COPY --from=builder /usr/lib/nrjmx/nrjmx.jar /usr/lib/nrjmx/nrjmx.jar
COPY --from=builder /go/src/github.com/newrelic/nri-kafka/bin /
CMD ["sleep", "1h"]
