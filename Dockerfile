FROM alpine as tape-base

FROM golang:alpine as golang

WORKDIR /root

ENV GOPROXY=https://goproxy.cn,direct
ENV export GOSUMDB=off

COPY . .

RUN go build -o tape ./benchmark
RUN go build -o tape_v1 ./cmd/tape

FROM tape-base
RUN mkdir -p /config
COPY --from=golang /root/tape /usr/local/bin
COPY --from=golang /root/tape_v1 /usr/local/bin

CMD ["tape"]
