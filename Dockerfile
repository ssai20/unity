FROM golang:1.23-bullseye AS builder
WORKDIR /app
COPY web ./web
COPY go.mod ./
RUN go mod download
COPY . .
RUN go build -o app ./cmd/api

EXPOSE 10000
CMD ["./app"]