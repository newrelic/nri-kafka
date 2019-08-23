FROM golang:1.10 as builder-kafka
RUN go get -d github.com/newrelic/nri-kafka/... && \
    cd /go/src/github.com/newrelic/nri-kafka && \
    make && \
    strip ./bin/nr-kafka

FROM maven:3-jdk-11 as builder-jmx
RUN git clone https://github.com/newrelic/nrjmx && \
    cd nrjmx && \
    mvn package -DskipTests -P \!deb,\!rpm,\!test

FROM newrelic/infrastructure:latest
ENV NRIA_IS_FORWARD_ONLY true
ENV NRIA_K8S_INTEGRATION true
COPY --from=builder-kafka /go/src/github.com/newrelic/nri-kafka/bin/nr-kafka /nri-sidecar/newrelic-infra/newrelic-integrations/bin/nr-kafka
COPY --from=builder-kafka /go/src/github.com/newrelic/nri-kafka/kafka-definition.yml /nri-sidecar/newrelic-infra/newrelic-integrations/definition.yaml
COPY --from=builder-jmx nrjmx/bin/nrjmx /usr/bin/nrjmx
COPY --from=builder-jmx nrjmx/target/nrjmx-*-jar-with-dependencies.jar /usr/lib/nrjmx/nrjmx.jar
RUN apk update && apk add openjdk7-jre
USER 1000
