FROM golang:1.26.9-alpine@sha256:3082400e369fa24d5fc60bca20edab3f6d604e0c5a690ec66b295eff4dd87ade AS builder

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

RUN apk upgrade --no-cache libcrypto3 libssl3 \
    && apk add --no-cache ca-certificates wget 'pcre2>=10.49-r0' \
    && addgroup -S shortli \
    && adduser -S -G shortli -h /var/lib/shortli shortli \
    && mkdir -p /var/lib/shortli/click-spool \
    && chown -R shortli:shortli /var/lib/shortli

COPY --from=builder /out/shortli /usr/local/bin/shortli

USER shortli
WORKDIR /var/lib/shortli
EXPOSE 43178
ENTRYPOINT ["/usr/local/bin/shortli"]
