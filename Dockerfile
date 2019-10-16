FROM golang:1.12.5-stretch

WORKDIR apps/iotex-core

RUN apt-get install -y --no-install-recommends make

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

RUN mkdir -p $GOPATH/pkg/linux_amd64/github.com/iotexproject/ && \
    make clean build && \
    cp ./bin/server /usr/local/bin/iotex-server  && \
    cp ./bin/ioctl /usr/local/bin/ioctl && \
    mkdir -p /etc/iotex/ && \
    rm -rf apps/iotex-core/

CMD [ "iotex-server"]
