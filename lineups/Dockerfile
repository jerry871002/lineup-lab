FROM golang:1.22 AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /lineups .

FROM alpine:3.21
RUN adduser -D -u 10001 appuser
USER appuser
COPY --from=build /lineups /lineups
EXPOSE 80
ENTRYPOINT ["/lineups"]
