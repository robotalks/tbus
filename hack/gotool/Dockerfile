FROM golang:1.6
RUN apt-get -y update && \
    apt-get -y install curl git tar zip && \
    apt-get -y clean && \
    curl -sSL https://github.com/google/protobuf/releases/download/v3.0.0/protoc-3.0.0-linux-x86_64.zip >/tmp/protoc.zip && \
    cd /usr/local && unzip /tmp/protoc.zip && rm /tmp/protoc.zip && chmod a+rx /usr/local/bin/protoc && \
    go get -v github.com/alecthomas/gometalinter && \
    go get -v golang.org/x/tools/cmd/... && \
    go get -v github.com/FiloSottile/gvt && \
    go get -v github.com/smartystreets/goconvey && \
    go get -v github.com/spf13/hugo && \
    gometalinter --install && \
    chmod -R a+rw /go
