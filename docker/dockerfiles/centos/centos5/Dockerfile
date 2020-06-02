# Inspired on https://github.com/jeromerobert/centos5-gcc5/blob/master/Dockerfile
FROM centos:5
ENV GO_VERSION 1.9
ENV GIT_VERSION 1.8.2.3

RUN rm -f /etc/yum.repos.d/*
ADD yum.repos.d/ /etc/yum.repos.d/
RUN echo 'exclude = *.i?86' >> /etc/yum.conf
RUN echo /usr/local/lib > /etc/ld.so.conf.d/local.conf && echo /usr/local/lib64 >> /etc/ld.so.conf.d/local.conf
ENV PKG_CONFIG_PATH=/usr/local/lib64/pkgconfig:/usr/local/lib/pkgconfig
ARG NUM_CPU=24
RUN yum install -y curl bzip2 gcc-c++ make m4 file pkgconfig perl expat-devel zlib-devel gettext \
    which openssh-clients rsync bzip2-devel readline-devel mesa-libGLU-devel man xz patch \
    libXt-devel libXtst libXrender libXi unzip \
    && yum clean all && rm -rf /usr/share/locale
ADD clean /root/
RUN chmod u+x /root/clean
WORKDIR /tmp

# OpenSSL
RUN curl https://www.openssl.org/source/openssl-1.0.2q.tar.gz | tar xz \
    && cd * \
    && ./config --prefix=/usr/local --openssldir=/usr/local shared \
    && make && make install \
    && mkdir /usr/local/ssl && cd /usr/local/ssl \
    && ln -s ../include/openssl include && ln -s ../lib64 lib \
    && rm /usr/local/lib64/libcrypto.a /usr/local/lib64/libssl.a \
    && /root/clean

# Curl
RUN curl http://cdn-fastly.deb.debian.org/debian/pool/main/c/curl/curl_7.64.0.orig.tar.gz | tar xz \
    && cd * && ./configure --disable-static && make -j $NUM_CPU && make install \
    && yum remove -y curl expat-devel && /root/clean \
    && curl https://curl.haxx.se/ca/cacert.pem -o /etc/pki/tls/certs/ca-bundle.crt

# Git
RUN curl -L https://www.kernel.org/pub/software/scm/git/git-2.20.1.tar.gz | tar xz \
    && cd * && ./configure --prefix=/usr/local && make -j $NUM_CPU  \
    && make NO_INSTALL_HARDLINKS=YesPlease install && /root/clean

## Go install
RUN curl https://storage.googleapis.com/golang/go$GO_VERSION.linux-amd64.tar.gz | tar xz \
    && chown -R root:root ./go \
    && mv go /root

ENV GOPATH /root


