FROM golang:alpine as builder

ENV TZ=Asia/Shanghai GOPATH='/gopath'

COPY . .

COPY ./.shell/docker-entrypoint.sh /run/docker-entrypoint.sh

RUN go build -o /run/xdd -ldflags="-w -s"

FROM node:alpine

ENV TZ=Asia/Shanghai

WORKDIR /run

VOLUME ["/data"]

COPY --from=builder /run/* /run/

RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.tuna.tsinghua.edu.cn/g' /etc/apk/repositories && apk add --no-cache tzdata

CMD ["/run/docker-entrypoint.sh"]