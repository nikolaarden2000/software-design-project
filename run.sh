#!/bin/bash

run_app() {
    docker-compose up --build app
}

case "$1" in
    ""|--app)
        run_app
        ;;
    --build)
        docker-compose build
        ;;
    --new)
        docker-compose down -v
        run_app
        ;;
    --unit-stat)
        docker-compose run --rm app cat go-coverage-report.txt
        docker-compose run --rm app cat js-test-report.txt
        ;;
    *)
        echo "Unknown option: $1"
        exit 1
        ;;
esac