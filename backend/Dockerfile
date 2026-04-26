FROM golang:tip-trixie AS build

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go test ./...
RUN go build -o backend ./cmd/server

FROM debian:bookworm-slim

WORKDIR /app

COPY --from=build /app/backend ./backend

EXPOSE 8080

CMD ["./backend"]