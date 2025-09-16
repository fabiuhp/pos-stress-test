# syntax=docker/dockerfile:1
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .
RUN go build -o /stress-tester

FROM alpine:3.18
WORKDIR /app
COPY --from=builder /stress-tester /usr/local/bin/stress-tester
ENTRYPOINT ["stress-tester"]
