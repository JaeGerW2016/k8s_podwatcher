FROM golang:1.11.6-stretch AS build

ENV PKG_PATH=$GOPATH/src/k8s_podwatcher

WORKDIR $PKG_PATH

ADD . .

RUN CGO_ENABLED=0 go build -mod=vendor -a -installsuffix cgo -o /tmp/k8s_podwatcher

FROM alpine:latest

LABEL maintainer="JaeGerW2016"

RUN apk add --no-cache -U tzdata ca-certificates

COPY --from=build /tmp/k8s_podwatcher /usr/bin/k8s_podwatcher

ENTRYPOINT ["k8s_podwatcher"]

