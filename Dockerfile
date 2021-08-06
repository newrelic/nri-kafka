FROM golang:1.10 as builder-kafka
COPY . /go/src/github.com/newrelic/nri-kafka/
RUN cd /go/src/github.com/newrelic/nri-kafka && \
    make && \
    strip ./bin/nri-kafka

FROM maven:3-jdk-11 as builder-jmx
RUN git clone https://github.com/newrelic/nrjmx && \
    cd nrjmx && \
    mvn package -DskipTests -P \!deb,\!rpm,\!test,\!tarball

FROM newrelic/infrastructure:latest
ENV NRIA_IS_FORWARD_ONLY true
ENV NRIA_K8S_INTEGRATION true
COPY --from=builder-kafka /go/src/github.com/newrelic/nri-kafka/bin/nri-kafka /nri-sidecar/newrelic-infra/newrelic-integrations/bin/nri-kafka
COPY --from=builder-jmx /nrjmx/bin /usr/bin/
RUN apk update && apk add openjdk8-jre
USER 1000
