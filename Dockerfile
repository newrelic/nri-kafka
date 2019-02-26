FROM golang:1.10 as builder-kafka
RUN go get -d github.com/newrelic/nri-kafka/... && \
    cd /go/src/github.com/newrelic/nri-kafka && \
    make && \
    strip ./bin/nr-kafka

FROM maven:3-jdk-11 as builder-jmx
RUN git clone https://github.com/newrelic/nrjmx && \
    cd nrjmx && \
    mvn package -P \!deb,\!rpm 

FROM newrelic/infrastructure:latest
COPY --from=builder-kafka /go/src/github.com/newrelic/nri-kafka/bin/nr-kafka /var/db/newrelic-infra/newrelic-integrations/bin/nr-kafka
COPY --from=builder-kafka /go/src/github.com/newrelic/nri-kafka/kafka-definition.yml /var/db/newrelic-infra/newrelic-integrations/definition.yaml
COPY --from=builder-jmx nrjmx/bin/nrjmx /usr/bin/nrjmx
COPY --from=builder-jmx nrjmx/target/nrjmx-*-jar-with-dependencies.jar /usr/lib/nrjmx/nrjmx.jar
RUN apk update && apk add openjdk7-jre