FROM golang:1.14-alpine


# Install aws-cli
RUN apk -Uuv add groff less python py-pip
RUN pip install awscli
RUN apk --purge -v del py-pip

# Install Goreleaser
RUN apk add --no-cache bash bzr curl docker-cli git rpm
RUN curl -sfL https://install.goreleaser.com/github.com/goreleaser/goreleaser.sh | sh

RUN rm /var/cache/apk/*

COPY .goreleaser.yml /
COPY . /go/src/github.com/newrelic/nri-nginx/

WORKDIR /go/src/github.com/newrelic/nri-nginx/
