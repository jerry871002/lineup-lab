FROM golang:1.22.5-bullseye as builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN go build -o stats ./cmd/stats

FROM debian:bullseye-slim

WORKDIR /app

COPY --from=builder /app/stats .

EXPOSE 80

CMD ["./stats"]
