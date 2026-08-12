FROM golang:1.24-alpine AS builder

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

FROM alpine:3.21

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
