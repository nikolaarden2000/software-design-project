FROM node:22-bookworm AS js-test
WORKDIR /software-design/static
COPY static/ ./
RUN npm ci || npm install
RUN npm test | tee js-test-report.txt

FROM golang:tip-trixie AS go-build
WORKDIR /software-design/backend
COPY ./backend/go.mod ./backend/go.sum* ./
RUN go mod download
COPY . .

RUN go test -cover ./... | tee go-coverage-report.txt
RUN go build -o app ./cmd

FROM debian:trixie-slim
WORKDIR /software-design

COPY --from=go-build /software-design/app .

COPY --from=go-build /software-design/coverage.out .
COPY --from=go-build /software-design/go-coverage-report.txt .
COPY --from=js-test /software-design/static/js-test-report.txt .

EXPOSE 8080
CMD ["./app"]