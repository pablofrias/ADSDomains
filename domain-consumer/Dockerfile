FROM golang:1.16-alpine as build

WORKDIR /usr/src/app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY *.go ./

RUN go build -o /domain-consumer

ENTRYPOINT [ "/domain-consumer" ]