FROM golang:1.6.2

EXPOSE 5678 6789

ENV TIME_ZONE=Asia/Shanghai
RUN ln -snf /usr/share/zoneinfo/$TIME_ZONE /etc/localtime && echo $TIME_ZONE > /etc/timezone

COPY . /go/src/github.com/zonesan/undefined

WORKDIR /go/src/github.com/zonesan/undefined

RUN go build

CMD ["sh", "-c", "./undefined"]
