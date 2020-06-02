FROM centos:7

ENV AWS_ACCESS_KEY_ID none
ENV AWS_SECRET_ACCESS_KEY none
ENV AWS_STORAGE_BUCKET_NAME none
ENV AWS_S3_MOUNTPOINT none
ENV REPO_YUM_UPDATE_METADATA_PATH none

# Add S3Fuse https://github.com/s3fs-fuse/s3fs-fuse/wiki/Installation-Notes
RUN yum update -y
RUN yum install -y gcc libstdc++-devel gcc-c++ fuse fuse-devel curl-devel libxml2-devel mailcap git automake make openssl-devel \
    && git clone https://github.com/s3fs-fuse/s3fs-fuse \
    && cd s3fs-fuse/ \
    && ./autogen.sh \
    && export PKG_CONFIG_PATH=/usr/local/lib/pkgconfig \
    && ./configure --prefix=/usr --with-openssl \
    && make \
    && make install

# Add Createrepo
RUN yum install -y curl createrepo

RUN yum clean all

ADD entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

CMD ["/entrypoint.sh"]