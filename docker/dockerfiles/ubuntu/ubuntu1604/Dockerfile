FROM ubuntu:16.04

ENV GO_VERSION 1.14.2

RUN apt-get update -y && apt-get install -y make curl git

## Go install
RUN curl https://storage.googleapis.com/golang/go$GO_VERSION.linux-amd64.tar.gz | tar xz \
    && chown -R root:root ./go \
    && mv go /root

# Let's people find our Go binaries
ENV PATH $PATH:/root/go/bin
ENV GOPATH /root/go