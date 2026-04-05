FROM golang:1.22.5-bullseye as builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN go build -o stat-api-server ./cmd/stat-api-server

FROM debian:bullseye-slim

WORKDIR /app

COPY --from=builder /app/stat-api-server .

EXPOSE 80

CMD ["./stat-api-server"]
