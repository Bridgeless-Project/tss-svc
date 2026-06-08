FROM golang:1.26.0-alpine as buildbase

RUN apk add git build-base

WORKDIR /go/src/github.com/Bridgeless-Project/tss-svc

ENV GO111MODULE="on"
ENV CGO_ENABLED=1
ENV GOOS="linux"
ENV GOPRIVATE=github.com/*
ENV GONOSUMDB=github.com/*
ENV GONOPROXY=github.com/*

COPY ./go.mod ./go.sum ./vendor ./

COPY . .

RUN go build -mod=vendor -o /usr/local/bin/tss-svc /go/src/github.com/Bridgeless-Project/tss-svc


FROM alpine:3.21

COPY --from=buildbase /usr/local/bin/tss-svc /usr/local/bin/tss-svc

RUN apk add --no-cache ca-certificates

ENTRYPOINT ["tss-svc"]