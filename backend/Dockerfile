FROM golang:1.26.2-trixie AS build

WORKDIR /app

COPY go.mod go.sum* ./
RUN go mod download

COPY . .

RUN go test -count=1 -coverprofile=coverage.out ./... 2>&1 | tee backend-unit.log
RUN go tool cover -func=coverage.out 2>&1 | tee -a backend-unit.log

RUN go build -o backend ./cmd/server

FROM debian:bookworm-slim

WORKDIR /app

COPY --from=build /app/backend ./backend
COPY --from=build /app/backend-unit.log ./backend-unit.log

EXPOSE 8080

CMD ["./backend"]