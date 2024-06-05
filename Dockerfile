ARG GOLANG_VERSION=1.21.4
ARG ALPINE_VERSION=3.18
FROM golang:${GOLANG_VERSION}-alpine${ALPINE_VERSION} as builder

RUN apk --no-cache --virtual .build-deps add make gcc musl-dev binutils-gold

COPY . /app
WORKDIR /app

RUN apk add git
RUN go build -o tcp_server .


FROM alpine:${ALPINE_VERSION}

LABEL maintainer="ramaprakoso@dtc.team"

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /usr/bin

#COPY --from=builder /app/tcp_server /usr/bin/tcp_server

# Copy the config file from the builder stage
COPY --from=builder /app/config.yml /usr/bin/config.yml
# Copy the built binary from the builder stage
COPY --from=builder /app/tcp_server .

ENTRYPOINT [ "/usr/bin/tcp_server" ]
CMD [ "run" ]

EXPOSE 8082 3306
