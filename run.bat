@echo off

IF "%1"=="" GOTO app
IF "%1"=="--build" GOTO build
IF "%1"=="--app" GOTO app
IF "%1"=="--new" GOTO new
IF "%1"=="--unit-stat" GOTO unit

ECHO Unknown option %1
GOTO end

:build
docker compose build --no-cache
GOTO end

:app
docker-compose up --build
GOTO end

:new
docker-compose down -v
GOTO app

:unit
docker compose run --rm backend cat backend-unit.log
docker compose run --rm frontend cat frontend-unit.log
GOTO end

:end
