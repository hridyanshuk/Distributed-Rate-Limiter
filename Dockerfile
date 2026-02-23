FROM golang:1.26.0 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o rate-limiter ./cmd/limiter

FROM alpine:latest
RUN apk --no-cache add ca-certificates iptables
WORKDIR /root/
COPY --from=builder /app/rate-limiter .
EXPOSE 7946 7946/udp 50051 6000/udp
ENTRYPOINT ["./rate-limiter"]
