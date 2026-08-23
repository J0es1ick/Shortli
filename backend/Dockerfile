FROM golang:1.25.13-alpine@sha256:844b27705f54e73773e0f9bc3c780633b9d7f4b4831bf35cdad02a81a4c80bd0 AS builder

WORKDIR /src
RUN apk add --no-cache ca-certificates
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /out/shortli \
    ./cmd/shortliService

FROM alpine:3.24@sha256:28bd5fe8b56d1bd048e5babf5b10710ebe0bae67db86916198a6eec434943f8b

RUN apk add --no-cache ca-certificates wget \
    && addgroup -S shortli \
    && adduser -S -G shortli -h /var/lib/shortli shortli \
    && mkdir -p /var/lib/shortli/click-spool \
    && chown -R shortli:shortli /var/lib/shortli

COPY --from=builder /out/shortli /usr/local/bin/shortli

USER shortli
WORKDIR /var/lib/shortli
EXPOSE 43178
ENTRYPOINT ["/usr/local/bin/shortli"]
