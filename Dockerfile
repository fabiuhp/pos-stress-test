# syntax=docker/dockerfile:1
FROM golang:1.25-alpine AS builder
ENV GOTOOLCHAIN=auto
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .
RUN go build -o /stress-tester

FROM alpine:3.20
WORKDIR /app
COPY --from=builder /stress-tester /usr/local/bin/stress-tester
ENTRYPOINT ["stress-tester"]
