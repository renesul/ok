FROM golang:1.25-alpine AS builder
RUN apk add --no-cache git make
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN make build

FROM alpine:3.23
RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -g 1000 ok && adduser -D -u 1000 -G ok ok
COPY --from=builder /src/build/ok /usr/local/bin/ok
USER ok
RUN ok onboard
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget -q --spider http://localhost:18790/health || exit 1
ENTRYPOINT ["ok"]
CMD ["gateway"]
