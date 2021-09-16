FROM golang:alpine AS builder

WORKDIR /go/app/src

COPY . ./

# We don't need any CGo.
ENV CGO_ENABLED=0

RUN go mod download
RUN go test ./...
RUN go install ./cmd/...


FROM alpine:latest

COPY --from=builder /go/bin/chargen2pd /usr/local/sbin/
COPY --from=builder /go/bin/testthroughput /usr/local/bin/

# Exposes the command line options as environment variables for ease of use.
ENV LISTEN_ADDR=tcp:0.0.0.0:19
ENV MAX_CONNS=10
ENV CONN_TIMEOUT=10
ENV LOG_PER_CONNECTION=true

EXPOSE 19
CMD /usr/local/sbin/chargen2pd \
    -standalone-log=false \
    -listen-addr=$LISTEN_ADDR \
    -max-conns=$MAX_CONNS \
    -conn-timeout=$CONN_TIMEOUT \
    -log-per-connection=$LOG_PER_CONNECTION
