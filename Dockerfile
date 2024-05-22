#　编译
FROM golang:1.20.5-alpine as builder
# 设置docker中golang的mod代理
ENV GOPROXY https://goproxy.cn,direct
ENV GO111MODULE on

# 设置容器的工作目录
WORKDIR /go/src
COPY . /go/src

# build
RUN cd /go/src/beats/heartbeat/ && go build -o axhb

FROM alpine:latest
WORKDIR /axhb
RUN mkdir /axhb/monitors.d
COPY --from=builder /go/src/beats/heartbeat/axhb /axhb/
COPY --from=builder /go/src/beats/heartbeat/heartbeat.yml /axhb/
ENTRYPOINT ./axhb -e
