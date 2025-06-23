FROM golang:1.23.4-alpine as buildbase

RUN apk add git build-base

WORKDIR /go/src/github.com/hyle-team/tss-svc

ENV GO111MODULE="on"
ENV CGO_ENABLED=1
ENV GOOS="linux"
ENV GOPRIVATE=github.com/*
ENV GONOSUMDB=github.com/*
ENV GONOPROXY=github.com/*

COPY ./go.mod ./go.sum ./
RUN go mod download

COPY . .

RUN go mod vendor
RUN go build  -o /usr/local/bin/tss-svc /go/src/github.com/hyle-team/tss-svc


FROM alpine:3.9

COPY --from=buildbase /usr/local/bin/tss-svc /usr/local/bin/tss-svc

RUN apk add --no-cache ca-certificates

ENTRYPOINT ["tss-svc"]