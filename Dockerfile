FROM golang:1.16-alpine3.13 as BUILD
RUN apk add make --no-cache
RUN mkdir /app
WORKDIR /app

ARG MODULE=mustset
ARG VERSION=0.0.0
COPY $MODULE/go.mod /app/$MODULE/
COPY $MODULE/go.sum /app/$MODULE/
# For go.mod `replace`
COPY ./shared /app/shared/


RUN ls /app
WORKDIR /app/$MODULE/
RUN go mod download

ADD ./ /app
RUN version=$VERSION make build_linux

RUN ls /app/$MODULE/bin/*

FROM alpine
RUN mkdir /app
WORKDIR /app

ARG MODULE=mustset
ARG VERSION=0.0.0

COPY --from=BUILD /app/$MODULE/bin/ /app/
RUN ls
RUN chmod a+x *

RUN mv $MODULE-amd64-linux-$VERSION entrypoint
ENTRYPOINT ["/app/entrypoint"]
